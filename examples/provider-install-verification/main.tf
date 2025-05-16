terraform {
  required_providers {
    nexaa = {
      source  = "registry.terraform.io/tilaa/nexaa"
      version = "0.1.0"
    }
  }
}

provider "nexaa" {
  username = "experimental-qa@tilaa.com"
  password = "EAG7pnp!jcq@ech6nbn"
}

resource "nexaa_namespace" "test" {
  name = "terraform3"
}

resource "nexaa_volume" "volume1" {
  namespace_name = "terraform3"
  name           = "terraform"
  size           = 3
}