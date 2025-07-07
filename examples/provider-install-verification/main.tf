terraform {
  required_providers {
    nexaa = {
      source  = "registry.terraform.io/tilaa/nexaa"
      version = "0.1.0"
    }
  }
}

provider "nexaa" {
  username = "expample@tilaa.com"
  password = "pass"
}

resource "nexaa_namespace" "namespace" {
  name = "terraform8"
}

resource "nexaa_volume" "volume" {
  name      = "storage"
  namespace = "terraform8"
  size      = 3
}

resource "nexaa_registry" "registry" {
  namespace = "terraform8"
  name      = "gitlab"
  source    = "registry.gitlab.com"
  username  = "user"
  password  = "pass"
  verify    = false
}

resource "nexaa_container" "container" {
  name      = "tf-container"
  namespace = "terraform8"
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
      name   = "Variable"
      value  = "finish"
      secret = false
    },
    {
      name   = "API_KEY"
      value  = "supersecret"
      secret = true
    },
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
    type         = "manual"
    manual_input = 1

    # auto_input = {
    #   minimal_replicas = 1
    #   maximal_replicas = 3

    #   triggers = [
    #     {
    #       type      = "CPU"
    #       threshold = 70
    #     },
    #     {
    #       type      = "MEMORY"
    #       threshold = 80
    #     }
    #   ]
    # }
  }
}

