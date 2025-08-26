terraform {
  required_version = ">= 1.0"
  required_providers {
    nexaa = {
      source = "nexaa-cloud/nexaa"
    }
  }
}

provider "nexaa" {
  username = var.nexaa_username
  password = var.nexaa_password
}

variable "nexaa_username" {
  description = "Username for Nexaa authentication"
  type        = string
  sensitive   = true
}

variable "nexaa_password" {
  description = "Password for Nexaa authentication"
  type        = string
  sensitive   = true
}

variable "namespace" {
  description = "Namespace name"
  type        = string
  sensitive   = true
}

variable "namespace_description" {
  description = "Namespace description"
  type        = string
  sensitive   = true
}

variable "container_job_name" {
  description = "Container Job name"
  type        = string
  sensitive   = true
}

resource "nexaa_namespace" "project" {
  name        = var.namespace
  description = var.namespace_description
}

resource "nexaa_registry" "registry" {
  namespace = nexaa_namespace.project.name
  name      = "example"
  source    = "registry.example.com"
  username  = "user"
  password  = "pass"
  verify    = true
}

resource "nexaa_volume" "volume" {
  namespace = nexaa_namespace.project.name
  name      = "volume-name"
  size      = 1
}

resource "nexaa_container_job" "containerjob" {
  name      = var.container_job_name
  namespace = nexaa_namespace.project.name
  image     = "registry.example.com/busybox:1.28"
  registry  = nexaa_registry.registry.name

  resources = {
    cpu = 0.25
    ram = 0.5
  }

  environment_variables = [
    {
      name   = "ENV"
      value  = "production"
      secret = false
    },
    {
      name   = "Variable"
      value  = "terraform"
      secret = false
    },
    {
      name   = "API_KEY"
      value  = "supersecret"
      secret = true
    }
  ]

  mounts = [
    {
      path   = "/storage/mount"
      volume = nexaa_volume.volume.name
    }
  ]

  schedule_cron_expression = "0 * * * *" # Every hour
  command                  = ["/bin/sh", "-c", "date; echo Hello World"]
}
