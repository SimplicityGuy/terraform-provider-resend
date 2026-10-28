# Add a domain to Resend, publish the DNS records Resend generates in a
# Cloudflare zone, then ask Resend to verify them.
#
# The provider block reads RESEND_API_KEY and CLOUDFLARE_API_TOKEN from the
# environment. The Cloudflare token needs Zone Read and DNS Edit permission on
# the zone.

terraform {
  required_providers {
    resend = {
      source  = "SimplicityGuy/resend"
      version = "~> 0.1"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5.0"
    }
  }
}

provider "resend" {}

provider "cloudflare" {}

variable "cloudflare_zone_id" {
  description = "The Cloudflare zone for the root domain, such as example.com."
  type        = string
}

variable "domain" {
  description = "The domain to send from: the zone itself or a subdomain of it."
  type        = string
  default     = "updates.example.com"
}

data "cloudflare_zone" "this" {
  zone_id = var.cloudflare_zone_id
}

resource "resend_domain" "this" {
  name = var.domain
}

locals {
  zone_name = data.cloudflare_zone.this.name

  # Key each record by its purpose, type and name, which are unique within a
  # domain. Resend returns names relative to the root domain (send.updates for
  # updates.example.com), and some, such as the Tracking record, fully
  # qualified; normalize them all to fully qualified names.
  resend_records = {
    for r in resend_domain.this.records : "${r.record}/${r.type}/${r.name}" => merge(r, {
      fqdn = r.name == local.zone_name || endswith(r.name, ".${local.zone_name}") ? r.name : "${r.name}.${local.zone_name}"
    })
  }
}

# Resend generates the records when the domain is created, so on the first
# apply the set of records isn't known yet and for_each can't be planned.
# Create the domain first, then apply the rest:
#
#   terraform apply -target=resend_domain.this
#   terraform apply
resource "cloudflare_dns_record" "resend" {
  for_each = local.resend_records

  zone_id  = var.cloudflare_zone_id
  name     = each.value.fqdn
  type     = each.value.type
  content  = each.value.value
  priority = each.value.priority == "" ? null : tonumber(each.value.priority)
  ttl      = 1 # automatic, which is what Resend's "Auto" asks for
  proxied  = false
  comment  = "Resend ${each.value.record} record for ${resend_domain.this.name}"
}

# Ask Resend to check the records once they are published.
resource "resend_domain_verification" "this" {
  domain_id = resend_domain.this.id

  depends_on = [cloudflare_dns_record.resend]
}

output "verification_status" {
  value = resend_domain_verification.this.status
}
