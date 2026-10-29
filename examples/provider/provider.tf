terraform {
  required_providers {
    resend = {
      source  = "SimplicityGuy/resend"
      version = "~> 0.1"
    }
  }
}

# The API key can come from the RESEND_API_KEY environment variable instead;
# omit api_key from the block to use it.
provider "resend" {
  api_key = var.resend_api_key
}

variable "resend_api_key" {
  description = "A Resend API key with full_access permission."
  type        = string
  sensitive   = true
}
