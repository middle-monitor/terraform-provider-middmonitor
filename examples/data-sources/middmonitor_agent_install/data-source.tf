data "middmonitor_agent_install" "agent" {
  install_token = middmonitor_install_token.agent.token
  os            = "linux"
  arch          = "amd64"
}

output "curl_install" {
  value = data.middmonitor_agent_install.agent.curl_install_command
}
