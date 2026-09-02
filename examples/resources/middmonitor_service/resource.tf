# An HTTP check with the two-level thresholds rather than the legacy single one.
resource "middmonitor_service" "api_health" {
  host_id            = middmonitor_host.web1.id
  name               = "api-health"
  type               = "http"
  hostname           = middmonitor_host.web1.hostname
  service            = middmonitor_host.web1.service
  path               = "/healthz"
  service_interval   = 60
  warning_threshold  = 800
  critical_threshold = 1500
}
