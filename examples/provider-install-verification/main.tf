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

resource "nexaa_namespace" "namespace" {
  name = "terraform7"
}

resource "nexaa_container" "container" {
  name      = "tf-container"
  namespace = "terraform7"
  image     = "nginx:latest"
  registry  = null

  resources = {
    cpu = 0.25
    ram = 0.5
  }

  ports = ["8000:8000", "80:80", "8008:8008"]

  environment_variables = [
    {
      name   = "ENV"
      value  = "production"
      secret = false
    },
    {
      name   = "API_KEY"
      value  = "supersecret"
      secret = true
    }
  ]

  ingresses = [
    {
      domain_name = null
      port        = 80
      tls         = true
      allow_list  = ["0.0.0.0/0"]
    }
  ]

  mounts = [
    {
      path   = "/storage/mount1"
      volume = "storage"
    }
  ]

  health_check = {
    port = 80
    path = "/storage/health"
  }

  scaling = {
    type         = "auto"
    #manual_input = 1

    auto_input = {
      minimal_replicas = 1
      maximal_replicas = 3

      triggers = [
        {
          type     = "CPU"
          threshold = 70
        },
        {
          type     = "MEMORY"
          threshold = 80
        }
      ]
    }
  }
}

resource "nexaa_volume" "volume" {
  name = "storage"
  namespace = "terraform7"
  size = 3
}

resource "nexaa_registry" "registry" {
  namespace     = "terraform7"
  name          = "gitlab"
  source        = "registry.gitlab.com"
  username      = "mvangastel"
  password      = "pass"
  verify        = false
}

