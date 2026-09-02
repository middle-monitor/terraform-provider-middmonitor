resource "middmonitor_host" "web1" {
  name         = "web-prod-01"
  hostname     = "10.0.1.5"
  service      = "api"
  display_name = "API production"
}
