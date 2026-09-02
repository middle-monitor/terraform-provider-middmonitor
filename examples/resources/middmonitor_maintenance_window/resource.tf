# Suppress alerts on a check during a planned upgrade.
resource "middmonitor_maintenance_window" "upgrade" {
  name        = "Kernel upgrade"
  target_type = "service"
  target_id   = middmonitor_service.api_health.id
  starts_at   = "2026-09-15T22:00:00Z"
  ends_at     = "2026-09-16T02:00:00Z"
}
