# Full example: monitored host + agent token + agent installation over SSH
# (remote-exec) + health checks. Copy and adapt (URLs, IDs, SSH).
# See the provider README.md for the ~/.terraformrc dev configuration.

terraform {
  required_providers {
    middmonitor = {
      source = "registry.terraform.io/middle-monitor/middmonitor"
    }
  }
}

provider "middmonitor" {
  base_url     = var.base_url
  org_slug     = var.org_slug
  access_token = var.access_token

  # receiver_base_url must be reachable FROM the target machine (not localhost)
  # if the receiver is exposed separately from the dashboard.
  receiver_base_url = var.receiver_base_url != "" ? var.receiver_base_url : null
}

# --- Current organization (data) ---

data "middmonitor_organization" "current" {}

# --- Agent install token ---

resource "middmonitor_install_token" "agent" {
  name = "terraform-${var.host_name}"
}

# --- Agent install URLs / commands ---

data "middmonitor_agent_install" "agent" {
  install_token = middmonitor_install_token.agent.token
  os            = "linux"
  arch          = "amd64"
}

# --- Monitored host + agent installation over SSH ---

resource "middmonitor_host" "app_server" {
  name         = var.host_name
  hostname     = var.host_address
  display_name = var.host_name
  service      = "api-backend"

  # Install the agent on the target machine.
  # MIDDLE_MONITOR_HOST_NAME links the agent to this host (avoids the interactive prompt).
  # sudo -S reads the password from stdin; sudo -E preserves the environment.
  # Tip: with NOPASSWD sudo or a root connection, drop the "echo ... | sudo -S"
  #      pipe and call "sudo bash ..." directly.
  provisioner "remote-exec" {
    inline = [
      "curl -fsSL '${data.middmonitor_agent_install.agent.install_script_url}' -o /tmp/mm-install.sh",
      "echo '${var.ssh_password}' | sudo -SE MIDDLE_MONITOR_HOST_NAME=${self.name} bash /tmp/mm-install.sh",
      "rm /tmp/mm-install.sh"
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

# --- Health checks attached to the host ---

resource "middmonitor_service" "http_check" {
  host_id     = middmonitor_host.app_server.id
  name        = "api-health"
  type        = "http"
  hostname    = middmonitor_host.app_server.hostname
  service     = middmonitor_host.app_server.service
  path        = "/healthz"
}

resource "middmonitor_service" "ping_check" {
  host_id     = middmonitor_host.app_server.id
  name        = "api-ping"
  type        = "ping"
  hostname    = middmonitor_host.app_server.hostname
  service     = middmonitor_host.app_server.service
}

resource "middmonitor_service" "cert_check" {
  host_id     = middmonitor_host.app_server.id
  name        = "api-cert"
  type        = "certificate"
  hostname    = "example.com"
  service     = middmonitor_host.app_server.service
}
