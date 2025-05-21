terraform {
  required_providers {
    nexaa = {
      source  = "registry.terraform.io/tilaa/nexaa"
      version = "0.1.0"
    }
  }
}

provider "nexaa" {
  username = "mail@tilaa.com"
  password = "pass"
}

resource "nexaa_namespace" "test" {
  name = "terraform5"
}

resource "nexaa_registry" "registry" {
  namespace_name = "terraform5"
  name           = "gitlab"
  source         = "registry.gitlab.com"
  username       = "mvangastel"
  password       = "pass"
}