resource "resend_domain" "example" {
  name   = "updates.example.com"
  region = "eu-west-1"

  open_tracking  = true
  click_tracking = true
  tls            = "enforced"
}

# The records to publish with your DNS provider.
output "dns_records" {
  value = resend_domain.example.records
}
