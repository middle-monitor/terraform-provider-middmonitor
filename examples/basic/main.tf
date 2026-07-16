# Minimal example - copy and adapt (JWT, URLs, IDs).
# See the provider README.md for the ~/.terraformrc dev configuration.

terraform {
  required_providers {
    middmonitor = {
      source = "registry.terraform.io/middle-monitor/middmonitor"
    }
  }
}

variable "middmonitor_access_token" {
  type        = string
  sensitive   = true
  description = "JWT from POST /api/v1/auth/login"
}

provider "middmonitor" {
  base_url     = "https://api.middlemonitor.io"
  org_slug     = "default"
  access_token = var.middmonitor_access_token
}

data "middmonitor_organization" "current" {}

output "org_name" {
  value = data.middmonitor_organization.current.name
}

resource "middmonitor_host" "app1" {
  name         = "prod-app-01"
  hostname     = "10.0.1.50"
  service      = "api"
  display_name = "API production"
}

resource "middmonitor_service" "http_health" {
  host_id          = middmonitor_host.app1.id
  name             = "api-https"
  type             = "http"
  hostname         = middmonitor_host.app1.hostname
  service          = middmonitor_host.app1.service
  path             = "/health"
  service_interval = 60
  # expected_status_code = 200 # exact code to treat as success; default is any 2xx
}

resource "middmonitor_install_token" "agent" {
  name = "terraform-prod-${middmonitor_host.app1.name}"
  # expires_at = "2026-12-31T23:59:59Z"
}

# URLs / commands for agent bootstrap (cloud-init, Ansible, etc.)
data "middmonitor_agent_install" "agent" {
  install_token = middmonitor_install_token.agent.token
  os            = "linux"
  arch          = "amd64"
}

output "curl_install" {
  value     = data.middmonitor_agent_install.agent.curl_install_command
  sensitive = false
}

output "install_script_url" {
  value     = data.middmonitor_agent_install.agent.install_script_url
  sensitive = false
}
