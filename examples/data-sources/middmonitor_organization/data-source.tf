data "middmonitor_organization" "current" {}

output "org_plan" {
  value = data.middmonitor_organization.current.plan
}
