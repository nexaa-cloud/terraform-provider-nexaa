terraform {
  required_providers {
    nexaa = {
      source = "nexaa-cloud/nexaa/nexaa"
    }
  }
}

# ===================================================================
# Variables
# ===================================================================

variable "nexaa_username" {
  type        = string
  description = "Username for the nexaa provider"
}

variable "nexaa_password" {
  type        = string
  sensitive   = true
  description = "Password for the nexaa provider"
}

# ===================================================================
# Provider
# ===================================================================

provider "nexaa" {
  username = "user@example.com"
  password = "password"
}