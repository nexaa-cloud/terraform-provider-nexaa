# Copyright Tilaa B.V. 2026
# SPDX-License-Identifier: MPL-2.0

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
  depends_on = [nexaa_volume.volume]
  name       = "tf-container"
  namespace  = nexaa_namespace.namespace.name
  image      = "nginx:latest"
  registry   = null

  resources = data.nexaa_container_resources.container_resource.id

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
    },
  ]

  ingresses = [
    {
      domain_name = null
      port        = 8008
      tls         = true
      allowlist   = ["0.0.0.0/0"]
    }
  ]

  mounts = [
    {
      path   = "/storage/mount1"
      volume = nexaa_volume.volume.name
    }
  ]

  health_check = {
    port = 80
    path = "/"
  }

  scaling = {
    type         = "manual"
    manual_input = 1
  }
}

resource "nexaa_starter_container" "starter-container" {
  depends_on = [nexaa_volume.volume]
  name       = "tf-starter-container"
  namespace  = nexaa_namespace.namespace.name
  image      = "nginx:latest"
  registry   = null

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
    },
  ]

  ingresses = [
    {
      domain_name = null
      port        = 8008
      tls         = true
      allowlist   = ["0.0.0.0/0"]
    }
  ]

  mounts = [
    {
      path   = "/storage/mount1"
      volume = nexaa_volume.volume.name
    }
  ]

  health_check = {
    port = 80
    path = "/"
  }
}

resource "nexaa_container_job" "container-job" {
  depends_on = [nexaa_volume.volume]
  namespace  = nexaa_namespace.namespace.name
  name       = "my-container-job"
  image      = "ubuntu:latest"
  resources  = data.nexaa_container_resources.container_resource.id
  schedule   = "0 4 * * *"

  command    = ["echo", "hello"]
  entrypoint = ["/bin/bash"]

  environment_variables = [
    {
      name   = "ENV"
      value  = "production"
      secret = false
    },
  ]

  mounts = [
    {
      path   = "/storage/mount1"
      volume = nexaa_volume.volume.name
    }
  ]

  timeouts {
    create = "30s"
    update = "30s"
    delete = "30s"
  }
}

data "nexaa_cloud_database_cluster_plans" "plan" {
  cpu      = 1
  memory   = 2.0
  storage  = 10
  replicas = 1
}

resource "nexaa_cloud_database_cluster" "cluster" {
  depends_on = [
    nexaa_namespace.namespace
  ]
  cluster = {
    name      = "tf-db-cluster"
    namespace = nexaa_namespace.namespace.name
  }

  spec = {
    type    = "PostgreSQL"
    version = "18.1"
  }

  plan = data.nexaa_cloud_database_cluster_plans.plan.id

  timeouts {
    create = "10m"
  }

  external_connection = {
    ports = {
      allowlist = ["192.168.1.1"]
    }
  }
}

resource "nexaa_cloud_database_cluster_database" "database" {
  depends_on = [
    nexaa_cloud_database_cluster.cluster,
  ]

  cluster = nexaa_cloud_database_cluster.cluster.cluster
  name    = "myDatabase"
}

resource "nexaa_cloud_database_cluster_user" "user" {
  depends_on = [
    nexaa_cloud_database_cluster_database.database,
  ]

  cluster  = nexaa_cloud_database_cluster.cluster.cluster
  name     = "myUser"
  password = "IThinkYouCanDoBetter"
  permissions = [
    {
      database_name = nexaa_cloud_database_cluster_database.database.name
      permission    = "read_write"
    }
  ]
}

data "nexaa_message_queue_plans" "queue_plan" {
  cpu      = 0.25
  memory   = 0.5
  storage  = 5.0
  replicas = 1
}

resource "nexaa_message_queue" "queue" {
  namespace = nexaa_namespace.namespace.name
  name      = "my-rabbitmq-queue"
  plan      = data.nexaa_message_queue_plans.queue_plan.id
  type      = "RabbitMQ"
  version   = "3.13"

  allowlist = [
    "0.0.0.0/0",
    "::/0"
  ]

  external_connection = {
    ports = {
      allowlist = ["0.0.0.0/0", "::/0"]
    }
  }

  timeouts {
    create = "2m"
    update = "2m"
    delete = "2m"
  }
}
