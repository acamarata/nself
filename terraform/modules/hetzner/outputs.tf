output "nself_url" {
  description = "URL of the nSelf deployment"
  value       = "https://${var.domain}"
}

output "ssh_host" {
  description = "IP address for SSH access"
  value       = hcloud_server.nself.ipv4_address
}

output "server_id" {
  description = "Hetzner server ID"
  value       = hcloud_server.nself.id
}

output "volume_id" {
  description = "Hetzner volume ID"
  value       = hcloud_volume.data.id
}
