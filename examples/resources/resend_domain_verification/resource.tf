resource "resend_domain" "example" {
  name = "updates.example.com"
}

# Publish resend_domain.example.records with your DNS provider first and add
# those records to depends_on, so verification starts after they exist.
resource "resend_domain_verification" "example" {
  domain_id = resend_domain.example.id
}

output "domain_status" {
  value = resend_domain_verification.example.status
}
