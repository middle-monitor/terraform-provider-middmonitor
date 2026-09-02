# Terraform Provider `middmonitor`

Terraform provider to drive [Middle Monitor](https://github.com/middle-monitor/middle-monitor): organization, hosts, monitoring services, agent install tokens, and **URLs / shell snippets** to install the agent (without running commands on your machines).

**Provider address:** `registry.terraform.io/middle-monitor/middmonitor`

**Documentation:** [middlemonitor.io/docs#terraform-setup](https://middlemonitor.io/docs#terraform-setup)

---

## Table of contents

1. [Prerequisites](#prerequisites)
2. [Build and install locally](#build-and-install-locally)
3. [Provider configuration](#provider-configuration)
4. [Resources and data sources](#resources-and-data-sources)
5. [Full example](#full-example)
6. [Agent: token, URLs and cloud-init](#agent-token-urls-and-cloud-init)
7. [State import](#state-import)
8. [Security and secrets](#security-and-secrets)
9. [Troubleshooting](#troubleshooting)

---

## Prerequisites

- **Terraform** >= 1.0
- A **JWT** to access the Middle Monitor dashboard (e.g. via `POST /api/v1/auth/login` with email / password)
- The dashboard **base URL** (JSON API), without a trailing slash, e.g. `https://api.middlemonitor.io`
- The **organization slug** used in API URLs (`/api/v1/organizations/{slug}/...`), often `default`

The two URLs can differ if the **receiver** (agent download, install script) is exposed separately from the dashboard: in that case use `receiver_base_url`.

---

## Build and install locally

```bash
cd terraform-provider-middmonitor
go build -o terraform-provider-middmonitor .
```

For Terraform to use the local binary (development), create or extend `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/middle-monitor/middmonitor" = "/absolute/path/to/terraform-provider-middmonitor"
  }
  direct {}
}
```

The given directory must **contain** the `terraform-provider-middmonitor` executable (not just the parent folder).

Then, in a test directory:

```bash
terraform init
terraform plan
```

Without `dev_overrides`, you would need to publish the provider to the Terraform registry or use another install method (`.terraformrc` with a mirror, etc.).

---

## Provider configuration

### `provider` block

```hcl
terraform {
  required_providers {
    middmonitor = {
      source  = "registry.terraform.io/middle-monitor/middmonitor"
      version = "~> 0.1"
    }
  }
}

provider "middmonitor" {
  base_url      = "https://api.middlemonitor.io"
  org_slug      = "default"
  access_token  = var.middmonitor_access_token # or via an environment variable
}
```

### Attributes

| Attribute | Required | Description |
|----------|-------------|-------------|
| `base_url` | yes | Base URL of the **dashboard** (JWT API), without a trailing `/`. |
| `org_slug` | yes | Organization slug (URL segment). |
| `access_token` | yes | JWT `Authorization: Bearer ...` (marked **sensitive**). |
| `receiver_base_url` | no | Base URL of the **receiver** (agent scripts and binaries). If empty, reuses `base_url` (monolithic deployment). |

### Tokens and CI

In CI, use **`TF_VAR_middmonitor_access_token`** (or equivalent) with a `variable "middmonitor_access_token" { sensitive = true }` to avoid committing the secret.

The provider can also read **`MIDDLE_MONITOR_BASE_URL`**, **`MIDDLE_MONITOR_ORG_SLUG`** and **`MIDDLE_MONITOR_ACCESS_TOKEN`** when the matching value in the `provider` block is an empty string (advanced case; in practice, prefer Terraform variables).

---

## Resources and data sources

### `middmonitor_organization` (data)

Reads the current organization (the one from the JWT + `org_slug`).

**Computed attributes:** `id`, `name`, `slug`, `plan`.

### `middmonitor_host` (resource)

Monitored host (inventory + agent target).

| Attribute | Notes |
|----------|--------|
| `id` | Computed after creation. |
| `name` | Technical name, unique per org. **Forced replacement** if changed. |
| `hostname` | Address or IP used for checks (API field `host`). **Forced replacement** if changed. |
| `service` | Logical application label (optional; may default to `name` on the API side). |
| `display_name` | Label shown in the UI. |
| `created_at` | Computed. |

**Update:** only the fields supported by the API (e.g. `display_name`) are sent; `name` / `hostname` imply a **new host** (replace).

### `middmonitor_service` (resource)

Service / health check attached to a host.

| Attribute | Notes |
|----------|--------|
| `id` | Computed. |
| `host_id` | ID of the `middmonitor_host`. **Forced replacement** if changed. |
| `name` | **Forced replacement** if changed. |
| `type` | E.g. `http`, `ping`, `sql`, `certificate`, `snmp`. |
| `hostname` | Check target (often aligned with the parent host). |
| `service` | Logical label (consistent with the host). |
| `path` | HTTP path for `http` type. |
| `credentials` | Optional JSON (auth, SQL, SNMP, etc.) - **sensitive**. |
| `service_interval`, `max_attempts` | Optional; defaults on the API side / state after read. |
| `failure_threshold` | Legacy single threshold. Prefer the two below. |
| `warning_threshold`, `critical_threshold` | Two-level thresholds: warning then critical (e.g. latency in ms). |
| `created_at` | Computed. |

### `middmonitor_host_group` (resource)

A group of hosts. Alert correlation is scoped to a host and its group, so the grouping is structural rather than cosmetic.

| Attribute | Notes |
|----------|--------|
| `id` | Computed. |
| `name` | Technical name, unique per organization. |
| `display_name` | Optional label shown in the UI. |
| `is_default` | Computed: whether this is the organization default group. |

### `middmonitor_alert_rule` (resource)

A threshold rule evaluated every minute.

| Attribute | Notes |
|----------|--------|
| `id` | Computed. |
| `name` | Becomes the incident title. |
| `metric` | Built-in signal: `cpu`, `ram`, `disk`, `latency`, `error_count`, `failure_rate`. |
| `custom_metric` | A scraped or SDK-reported series, e.g. `node_load1`. This is how you alert on anything the agent pulls. |
| `custom_labels` | Map of label equality constraints narrowing the custom series. |
| `target_type`, `target_id` | `any` (default), `service` or `host`, plus the id it is scoped to. |
| `aggregation` | `avg` (default), `min`, `max`, `sum`, `p50`...`p99`. |
| `operator` | `gt`, `gte`, `lt`, `lte`, `eq`. |
| `threshold` | Single threshold; ignored when the two-level ones are set. |
| `warning_threshold`, `critical_threshold` | Two-level thresholds. Critical wins over warning. |
| `recovery_threshold` | Hysteresis: the incident resolves only once the value crosses back past this. |
| `duration` | Evaluation window in seconds (default 300). |
| `enabled`, `notify_warning`, `notify_critical` | Default `true`. |
| `channels` | List of notification channel IDs. Empty means every enabled channel. |

### `middmonitor_notification_channel` (resource)

Where alerts are delivered.

| Attribute | Notes |
|----------|--------|
| `id` | Computed. |
| `name` | Channel name. |
| `type` | `email`, `slack`, `webhook`, `jsm`, `whatsapp`. |
| `config` | JSON object (`jsonencode({...})`) - **sensitive**. The accepted keys depend on `type`. |
| `enabled` | Defaults to `true`. |

For a webhook, `config` accepts `webhook_url`, `secret` (HMAC key), `format`
(`slack` by default, `structured` for the typed payload), `headers`, and the
anti-burst settings `group_by`, `group_wait` and `repeat_interval`. It is a JSON
string rather than a typed block because the keys differ per channel type; the
provider validates that it parses as an object.

**Read:** the API does not return secrets, so the configured `config` string
stays in state as written. A drift on the platform side is not detected.

### `middmonitor_maintenance_window` (resource)

Suppresses alerts on one target for a period.

| Attribute | Notes |
|----------|--------|
| `id` | Computed. |
| `name` | Why the window exists. **Forced replacement** if changed. |
| `target_type` | `service` or `host`. **Forced replacement**. |
| `target_id` | ID of the service or host. **Forced replacement**. |
| `starts_at`, `ends_at` | RFC3339. **Forced replacement**. |

The API has no update for maintenance windows, so every attribute forces a
replacement rather than pretending an in-place edit happened.

### `middmonitor_install_token` (resource)

Creates an **install token** for the agent (receiver API).

| Attribute | Notes |
|----------|--------|
| `id` | Token ID in the database. |
| `name` | Display label. **Forced replacement** if changed. |
| `expires_at` | Optional, RFC3339. **Forced replacement** if changed. |
| `token` | Secret - **returned only on creation**; sensitive. |
| `token_prefix`, `created_at` | Display / metadata if the API returns them. |

**Read:** the API generally does **not** return the full secret; Terraform state keeps the token as it was after the initial `apply`. A `terraform refresh` does not "recover" the secret from the API.

**Import:** `terraform import middmonitor_install_token.example <numeric_id>` - the secret will not be in state; recreate a token if needed.

### `middmonitor_agent_install` (data)

From a **token** (often `middmonitor_install_token.token`), computes:

| Attribute | Description |
|----------|-------------|
| `install_script_url` | GET URL of the install shell script (`?token=` included, properly encoded). |
| `agent_binary_url` | Direct binary URL (`os` / `arch`: default `linux` / `amd64`). |
| `curl_install_command` | Example: `curl -fsSL "..." \| sudo bash` |
| `export_env_snippet` | `MIDDLE_MONITOR_API_URL` and `X_INSTALL_TOKEN` variables for manual install. |

**Inputs:** `install_token` (required, sensitive), `os`, `arch` (optional).

This data source **runs nothing** on your servers: it is up to you to inject the command into **cloud-init**, **Ansible**, **Packer**, etc.

---

## Full example

- [`examples/basic/`](examples/basic/) - core resources + agent install URLs (no remote execution).
- [`examples/alerting/`](examples/alerting/) - host group, structured webhook channel, a rule on a scraped series, two-level check thresholds and a maintenance window.
- [`examples/agent-install/`](examples/agent-install/) - full flow **with agent installation over SSH** (`remote-exec`): host, token, `agent_install` data source, checks, and agent bootstrap on the target machine.

---

## Agent: token, URLs and cloud-init

1. Create a `middmonitor_install_token`.
2. Reference `middmonitor_agent_install` with that token.
3. Use `curl_install_command` or `install_script_url` in your bootstrap pipeline.

Conceptual example (cloud-init):

```yaml
#cloud-config
runcmd:
  - curl -fsSL "${install_script_url}" | bash
```

In Terraform, use `templatefile()` or a value rendered by the `middmonitor_agent_install` data source.

### Install over SSH (`remote-exec`)

To install the agent directly from Terraform on an existing machine, add a `provisioner "remote-exec"` on the `middmonitor_host`. Pass **`MIDDLE_MONITOR_HOST_NAME`** to the script to link the agent to the host (no interactive prompt):

```hcl
resource "middmonitor_host" "app_server" {
  name        = "app-server-01"
  hostname    = "10.0.1.50"

  provisioner "remote-exec" {
    inline = [
      "curl -fsSL '${data.middmonitor_agent_install.agent.install_script_url}' -o /tmp/mm-install.sh",
      "echo '${var.ssh_password}' | sudo -SE MIDDLE_MONITOR_HOST_NAME=${self.name} bash /tmp/mm-install.sh",
      "rm /tmp/mm-install.sh",
    ]
    connection {
      type        = "ssh"
      host        = self.hostname
      user        = var.ssh_user
      password    = var.ssh_password != "" ? var.ssh_password : null
      private_key = var.ssh_password == "" ? file(var.ssh_private_key_path) : null
    }
  }
}
```

Ready-to-use example: [`examples/agent-install/`](examples/agent-install/).

- `receiver_base_url` must be **reachable from the target machine** (not `localhost`).
- With NOPASSWD sudo or a root connection, drop the `echo ... | sudo -S` pipe.
- The `provisioner` only runs on **creation**; to reinstall: `terraform apply -replace=middmonitor_host.app_server`.

**Separate receiver:** if the `/api/v1/agents/download/...` URLs are not on the same host as the dashboard, set `receiver_base_url` in the provider so the generated URLs point to the right service.

---

## State import

| Resource | Command |
|-----------|----------|
| Host | `terraform import middmonitor_host.name <id>` |
| Host group | `terraform import middmonitor_host_group.name <id>` |
| Service | `terraform import middmonitor_service.name <id>` |
| Token | `terraform import middmonitor_install_token.name <id>` |
| Alert rule | `terraform import middmonitor_alert_rule.name <id>` |
| Notification channel | `terraform import middmonitor_notification_channel.name <id>` |
| Maintenance window | `terraform import middmonitor_maintenance_window.name <id>` |

IDs are the numeric identifiers returned by the API.

---

## Security and secrets

- The **JWT** and **install tokens** are secrets: Terraform state, CI and backups must be protected (encrypted `terraform state`, restricted access).
- The services' `credentials` field is **sensitive** in the provider; the API may still return sensitive data on read - restrict access to state and the dashboard.
- Do not commit `.tfvars` containing tokens; use environment variables or a vault (Vault, etc.).

---

## Troubleshooting

| Issue | Hint |
|----------|--------|
| `401` / `403` | Expired JWT or wrong `org_slug`. |
| `404` on the API | Check `base_url` (scheme, no extra `/`) and the `/api/v1/organizations/...` path. |
| Wrong agent URLs | Set `receiver_base_url` if the receiver is not behind `base_url`. |
| Provider not found locally | Check `dev_overrides` in `~/.terraformrc` and the path to the binary. |
| Empty install token after import | Expected: the API does not return the secret; recreate a token or do not import sensitive tokens. |

---

## Development

```bash
go test ./...
go build -o terraform-provider-middmonitor .
```

`docs/` is what the Terraform Registry publishes; it is generated from the schemas
and from `examples/`, so regenerate it whenever either changes, before tagging a release.

```bash
make docs
```

License: aligned with the parent Middle Monitor repository.
