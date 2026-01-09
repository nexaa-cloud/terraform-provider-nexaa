terraform {
  required_providers {
    nexaa = {
      source = "nexaa-cloud/nexaa/nexaa"
    }
  }
}

provider "nexaa" {
  username = "user@example.com"
  password = "password"
}