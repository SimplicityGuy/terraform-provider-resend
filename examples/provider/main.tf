terraform {
  required_providers {
    resend = {
      source = "SimplicityGuy/resend"
    }
  }
}

# Configure the Resend provider.
# The API key can also be set via the RESEND_API_KEY environment variable.
provider "resend" {
  api_key = var.resend_api_key
}

variable "resend_api_key" {
  type      = string
  sensitive = true
}
