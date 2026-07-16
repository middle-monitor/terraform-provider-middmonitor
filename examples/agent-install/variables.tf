variable "base_url" {
  description = "Base URL of the Middle Monitor dashboard (e.g. https://api.middlemonitor.io)"
  type        = string
}

variable "org_slug" {
  description = "Organization slug (API URL segment)"
  type        = string
  default     = "default"
}

variable "access_token" {
  description = "JWT (POST /api/v1/auth/login) or \"mm_...\" API key. Use TF_VAR_access_token in CI."
  type        = string
  sensitive   = true
}

variable "receiver_base_url" {
  description = "Receiver URL if separate from the dashboard. Must be reachable from the target machine."
  type        = string
  default     = ""
}

variable "host_name" {
  description = "Technical host name (immutable after creation)"
  type        = string
  default     = "app-server-01"
}

variable "host_address" {
  description = "Address or IP of the target machine (checks + SSH connection)"
  type        = string
}

variable "ssh_user" {
  description = "SSH user on the target machine"
  type        = string
  default     = "ubuntu"
}

variable "ssh_password" {
  description = "SSH password (leave empty to use a private key)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "ssh_private_key_path" {
  description = "Path to the SSH private key (used when ssh_password is empty)"
  type        = string
  default     = "~/.ssh/id_rsa"
}
