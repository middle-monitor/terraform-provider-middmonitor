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
