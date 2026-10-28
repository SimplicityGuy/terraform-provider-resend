data "resend_domain" "example" {
  id = "d91cd9bd-1176-453e-8fc1-35364d380206"
}

output "domain_status" {
  value = data.resend_domain.example.status
}

# The DKIM record, for example to publish it with a DNS provider.
output "dkim_record" {
  value = one([for r in data.resend_domain.example.records : r if r.record == "DKIM"])
}
