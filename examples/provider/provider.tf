terraform {
  required_providers {
    middmonitor = {
      source = "middle-monitor/middmonitor"
    }
  }
}

variable "middmonitor_access_token" {
  type        = string
  sensitive   = true
  description = "JWT from POST /api/v1/auth/login, or an organization API key"
}

provider "middmonitor" {
  base_url     = "https://api.middlemonitor.io"
  org_slug     = "default"
  access_token = var.middmonitor_access_token
}
