terraform {
  required_providers {
    nexaa = {
      source = "nexaa-cloud/nexaa/nexaa"
    }
  }
}

provider "nexaa" {
  username = "user@example.com"
  password = "pass"
}

resource "nexaa_namespace" "namespace" {
  name = "terraform"
}

resource "nexaa_volume" "volume" {
  name      = "storage"
  namespace = nexaa_namespace.namespace.name
  size      = 4
}

resource "nexaa_registry" "registry" {
  namespace = nexaa_namespace.namespace.name
  name      = "gitlab"
  source    = "registry.gitlab.com"
  username  = "user"
  password  = "pass"
  verify    = false
}

data "nexaa_container_resources" "container_resource" {
  cpu    = 0.25
  memory = 0.5
}

resource "nexaa_container" "container" {
  name      = "tf-container2"
  namespace = nexaa_namespace.namespace.name
  image     = "nginx:latest"
  registry  = null

  resources = data.nexaa_container_resources.container_resource.id

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
      port        = 8008
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

    auto_input = {
      minimal_replicas = 1
      maximal_replicas = 3

      triggers = [
        {
          type      = "CPU"
          threshold = 70
        },
        {
          type      = "MEMORY"
          threshold = 80
        }
      ]
    }
  }
}

resource "nexaa_starter_container" "starter-container" {
  name      = "tf-starter-container"
  namespace = nexaa_namespace.namespace.name
  image     = "nginx:latest"
  registry  = null

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
      port        = 8008
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
}

# Cloud Database Cluster example
resource "nexaa_clouddatabasecluster" "database" {
  name      = "test-db-cluster3"
  namespace = nexaa_namespace.namespace.name

  spec = {
    type    = "PostgreSQL"
    version = "16.4"
  }

  plan = {
    cpu      = 1
    memory   = 2.0
    storage  = 60
    replicas = 1
  }
}

