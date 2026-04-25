# nSelf Terraform module — Hetzner provider
# Provisions a Hetzner CX server + volume + Cloudflare DNS for nSelf.
#
# Usage:
#   module "nself" {
#     source      = "nself-org/nself/hetzner"
#     version     = "1.0.0"
#     server_type = "cx21"
#     location    = "fsn1"
#     domain      = "myapp.com"
#     ssh_keys    = ["my-key"]
#   }

terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.45"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
  required_version = ">= 1.5"
}

resource "hcloud_server" "nself" {
  name        = "nself-${var.location}"
  server_type = var.server_type
  image       = "ubuntu-22.04"
  location    = var.location
  ssh_keys    = var.ssh_keys

  user_data = templatefile("${path.module}/cloud-init.yaml.tpl", {
    domain          = var.domain
    nself_version   = var.nself_version
    backup_destination = var.backup_destination
  })

  labels = {
    managed-by = "nself-terraform"
    domain     = var.domain
  }
}

resource "hcloud_volume" "data" {
  name      = "nself-data-${var.location}"
  size      = var.volume_size_gb
  server_id = hcloud_server.nself.id
  automount = true
  format    = "ext4"
}

resource "hcloud_firewall" "nself" {
  name = "nself-firewall-${var.location}"

  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "22"
    source_ips = var.ssh_allowed_ips
  }
  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "80"
    source_ips = ["0.0.0.0/0", "::/0"]
  }
  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "443"
    source_ips = ["0.0.0.0/0", "::/0"]
  }
}

resource "hcloud_firewall_attachment" "nself" {
  firewall_id = hcloud_firewall.nself.id
  server_ids  = [hcloud_server.nself.id]
}

# DNS — creates an A record pointing the domain to the server IP.
# Requires CLOUDFLARE_API_TOKEN env var and var.cloudflare_zone_id.
resource "cloudflare_record" "api" {
  count   = var.cloudflare_zone_id != "" ? 1 : 0
  zone_id = var.cloudflare_zone_id
  name    = "api.${var.domain}"
  value   = hcloud_server.nself.ipv4_address
  type    = "A"
  ttl     = 60
  proxied = false
}

resource "cloudflare_record" "root" {
  count   = var.cloudflare_zone_id != "" ? 1 : 0
  zone_id = var.cloudflare_zone_id
  name    = var.domain
  value   = hcloud_server.nself.ipv4_address
  type    = "A"
  ttl     = 60
  proxied = true
}
