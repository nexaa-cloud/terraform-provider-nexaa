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
  password = "pass"
}

resource "nexaa_namespace" "test" {
  name = "terraform5"
}

resource "nexaa_registry" "registry" {
  namespace     = "terraform5"
  name          = "gitlab"
  source        = "registry.gitlab.com"
  username      = "mvangastel"
  password      = "pass"
  verify        = false
}