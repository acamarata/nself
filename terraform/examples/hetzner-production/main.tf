# Example: nSelf on Hetzner (production single-node)
#
# Prerequisites:
#   export HCLOUD_TOKEN=<your-hetzner-token>
#   export CLOUDFLARE_API_TOKEN=<your-cloudflare-token>  # optional, for DNS
#
# Usage:
#   terraform init
#   terraform plan
#   terraform apply

module "nself" {
  source = "../../modules/hetzner"

  server_type        = "cx21"
  location           = "fsn1"
  domain             = "myapp.com"
  ssh_keys           = ["my-key"]
  volume_size_gb     = 40
  backup_destination = "s3://my-bucket/nself-backup"
  cloudflare_zone_id = ""  # Set to enable automatic DNS records
}

output "nself_url" {
  value = module.nself.nself_url
}

output "ssh_host" {
  value = module.nself.ssh_host
}
