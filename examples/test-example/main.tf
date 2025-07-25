terraform {
  required_providers {
    nexaa = {
      source  = "registry.terraform.io/tilaa/nexaa"
      version = "0.1.0"
    }
  }
}

provider "nexaa" {
  username = "user@example.com"
  password = "pass"
}

resource "nexaa_namespace" "namespace" {
  name = "terraform8"
}

resource "nexaa_volume" "volume" {
  name      = "storage"
  namespace = nexaa_namespace.namespace.name
  size      = 3
}