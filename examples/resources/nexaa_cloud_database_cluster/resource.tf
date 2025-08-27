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

data "nexaa_cloud_database_cluster_plans" "plan" {
  cpu      = 1
  memory   = 2.0
  storage  = 10
  replicas = 1
}

resource "nexaa_namespace" "project" {
  name        = var.namespace
  description = var.namespace_description
}

# Basic cloud database cluster
resource "nexaa_cloud_database_cluster" "cluster" {
  depends_on = [nexaa_namespace.project]
  cluster = {
    name      = "database-cluster"
    namespace = nexaa_namespace.project.name
  }

  spec = {
    type    = "PostgreSQL"
    version = "16.4"
  }

  plan = data.nexaa_cloud_database_cluster_plans.plan.id
}
