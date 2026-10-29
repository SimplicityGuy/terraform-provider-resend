# A key that can manage every resource in the Resend team.
resource "resend_api_key" "admin" {
  name = "terraform-admin"
}

# A key that can only send email, and only from one domain.
resource "resend_domain" "example" {
  name = "updates.example.com"
}

resource "resend_api_key" "sending" {
  name       = "app-sending"
  permission = "sending_access"
  domain_id  = resend_domain.example.id
}

# The token is only available here, right after the key is created. Hand it to
# whatever needs it (a secret store, an application's configuration).
output "sending_api_key" {
  value     = resend_api_key.sending.token
  sensitive = true
}
