# Alerting described in code: a host group, a webhook channel that speaks the
# structured payload, a rule on a scraped series, and a maintenance window.
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
  description = "JWT from POST /api/v1/auth/login, or an organization API key"
}

variable "webhook_secret" {
  type        = string
  sensitive   = true
  description = "HMAC key the receiver verifies the signature with"
}

provider "middmonitor" {
  base_url     = "https://api.middlemonitor.io"
  org_slug     = "default"
  access_token = var.middmonitor_access_token
}

resource "middmonitor_host_group" "frontends" {
  name         = "frontends"
  display_name = "Front-end servers"
}

resource "middmonitor_host" "web1" {
  name     = "web-prod-01"
  hostname = "10.0.1.5"
  service  = "api"
}

# The receiver gets the typed payload rather than the Slack shape, and a burst
# on one host arrives as a single delivery.
resource "middmonitor_notification_channel" "automation" {
  name = "automation"
  type = "webhook"

  config = jsonencode({
    webhook_url = "https://ops.example.com/hooks/middlemonitor"
    secret      = var.webhook_secret
    format      = "structured"

    headers = {
      Authorization = "Bearer gateway-token"
    }

    group_by        = ["host_id"]
    group_wait      = 30
    repeat_interval = 14400
  })
}

# A rule on a series the agent scrapes: node_load1 on the ext4 root filesystem
# of this host, warning then critical, with hysteresis on the way back down.
resource "middmonitor_alert_rule" "load" {
  name          = "Load average high"
  custom_metric = "node_load1"

  custom_labels = {
    instance = "10.0.1.5:9100"
  }

  target_type = "host"
  target_id   = middmonitor_host.web1.id

  aggregation        = "avg"
  operator           = "gt"
  warning_threshold  = 4
  critical_threshold = 8
  recovery_threshold = 3
  duration           = 300

  channels = [middmonitor_notification_channel.automation.id]
}

# An HTTP check with the two-level thresholds rather than the legacy single one.
resource "middmonitor_service" "api_health" {
  host_id            = middmonitor_host.web1.id
  name               = "api-health"
  type               = "http"
  hostname           = "10.0.1.5"
  service            = "api"
  path               = "/healthz"
  service_interval   = 60
  warning_threshold  = 800
  critical_threshold = 1500
}

# Suppress alerts on that check during a planned upgrade.
resource "middmonitor_maintenance_window" "upgrade" {
  name        = "Kernel upgrade"
  target_type = "service"
  target_id   = middmonitor_service.api_health.id
  starts_at   = "2026-09-15T22:00:00Z"
  ends_at     = "2026-09-16T02:00:00Z"
}
