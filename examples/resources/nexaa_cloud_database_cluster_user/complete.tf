terraform {
  required_version = ">= 1.0"
  required_providers {
    nexaa = {
      source = "nexaa-cloud/nexaa"
    }
  }
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

variable "cluster_name" {
  description = "Cluster name"
  type        = string
  sensitive   = true
}

variable "database_name" {
  description = "Database name getting created on the cluster"
  type        = string
  sensitive   = true
}

variable "user_name" {
  description = "Name of user created on the cluster"
  type        = string
  sensitive   = true
}

variable "password" {
  sensitive = true
  type      = string
}

provider "nexaa" {
  username = var.nexaa_username
  password = var.nexaa_password
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
    name      = var.cluster_name
    namespace = nexaa_namespace.project.name
  }

  spec = {
    type    = "PostgreSQL"
    version = "16.4"
  }

  plan = data.nexaa_cloud_database_cluster_plans.plan.id
}

resource "nexaa_cloud_database_cluster_database" "db1" {
  depends_on  = [nexaa_cloud_database_cluster.cluster]
  name        = var.database_name
  description = ""
  cluster     = nexaa_cloud_database_cluster.cluster.cluster
}

resource "nexaa_cloud_database_cluster_user" "user" {
  depends_on = [
    nexaa_cloud_database_cluster.cluster,
  ]

  cluster  = nexaa_cloud_database_cluster.cluster.cluster
  name     = var.user_name
  password = var.password
  permissions = [
    {
      database_name = nexaa_cloud_database_cluster_database.db1.name,
      permission    = "read_write",
      state         = "present",
    }
  ]
}
