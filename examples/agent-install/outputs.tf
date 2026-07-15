output "org_name" {
  description = "Organization name"
  value       = data.middmonitor_organization.current.name
}

output "host_id" {
  description = "ID of the created host"
  value       = middmonitor_host.app_server.id
}

output "install_token" {
  description = "Agent install token (secret)"
  value       = middmonitor_install_token.agent.token
  sensitive   = true
}

output "agent_install_command" {
  description = "curl install command (reference / manual bootstrap)"
  value       = data.middmonitor_agent_install.agent.curl_install_command
}

output "service_ids" {
  description = "IDs of the created health checks"
  value = {
    http = middmonitor_service.http_check.id
    ping = middmonitor_service.ping_check.id
    cert = middmonitor_service.cert_check.id
  }
}
