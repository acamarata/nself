variable "server_type" {
  description = "Hetzner server type (cx21, cx31, cx41, etc.)"
  type        = string
  default     = "cx21"
}

variable "location" {
  description = "Hetzner datacenter location (fsn1, nbg1, hel1, ash, hil)"
  type        = string
  default     = "fsn1"
}

variable "domain" {
  description = "Domain name for the nSelf deployment (e.g. myapp.com)"
  type        = string
}

variable "ssh_keys" {
  description = "Hetzner SSH key names to allow on the server"
  type        = list(string)
  default     = []
}

variable "ssh_allowed_ips" {
  description = "IP ranges allowed to SSH to the server"
  type        = list(string)
  default     = ["0.0.0.0/0", "::/0"]
}

variable "volume_size_gb" {
  description = "Size of the persistent data volume in GB"
  type        = number
  default     = 20
}

variable "nself_version" {
  description = "nSelf CLI version to install at boot"
  type        = string
  default     = "latest"
}

variable "backup_destination" {
  description = "S3-compatible URL for automated backups (e.g. s3://bucket/nself)"
  type        = string
  default     = ""
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for DNS records (leave empty to skip DNS)"
  type        = string
  default     = ""
}

variable "deploy_ssh_pubkey" {
  description = "Public key for the nself deploy user's authorized_keys (~/.ssh/authorized_keys). Set from NSELF_DEPLOY_KEY_PATH.pub."
  type        = string
  default     = ""
}
