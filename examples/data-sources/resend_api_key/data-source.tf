data "resend_api_key" "example" {
  id = "dacf4072-4119-4d88-932f-6202748ac7c8"
}

output "api_key_name" {
  value = data.resend_api_key.example.name
}

output "api_key_created_at" {
  value = data.resend_api_key.example.created_at
}
