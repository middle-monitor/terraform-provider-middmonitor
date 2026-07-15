# `agent-install` example

Full flow: creates a monitored host, generates an agent token, **installs the agent
on the target machine over SSH** (`remote-exec`), then creates health checks.

## What this example does

1. `middmonitor_install_token` - agent install token.
2. `middmonitor_agent_install` (data) - computes the install script URL (token injected).
3. `middmonitor_host` - creates the host, then the `remote-exec` provisioner:
   - downloads the install script from the receiver,
   - runs it with `MIDDLE_MONITOR_HOST_NAME` (links the agent to the host, no prompt),
   - cleans up the temporary file.
4. `middmonitor_service` - HTTP, ping and certificate checks attached to the host.

## Usage

1. Copy `terraform.tfvars.example` to `terraform.tfvars` and adapt it.
2. Configure the provider for local dev (see the provider README, section
   *Build and install locally*).
3. Run:

   ```bash
   terraform init
   terraform apply
   ```

## Notes

- **`receiver_base_url`** must be reachable **from the target machine**, not
  `localhost`: that machine downloads the agent script and binary.
- **sudo**: `echo '...' | sudo -SE ... bash` passes the password via stdin.
  With NOPASSWD sudo or a **root** connection, drop the pipe and call
  `sudo bash ...` directly.
- **Suppressed output**: since `ssh_password` is `sensitive`, Terraform hides the
  provisioner output. This is intentional (the secret never appears in the logs).
- The `provisioner` only runs on host **creation**. To reinstall the agent on an
  existing host: `terraform apply -replace=middmonitor_host.app_server`.
