# A rule on a series the agent scrapes, warning then critical, with hysteresis
# on the way back down.
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
