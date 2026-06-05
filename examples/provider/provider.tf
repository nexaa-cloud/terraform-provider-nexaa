terraform {
  required_providers {
    nexaa = {
      source = "nexaa-cloud/nexaa"
    }
  }
}

provider "nexaa" {
  username = "user@example.com"
  password = "password"
}