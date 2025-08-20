terraform {
  required_providers {
    nexaa = {
      source = "registry.terraform.io/tilaa/nexaa"
    }
  }
}

provider "nexaa" {
  username = var.nexaa_username
  password = var.nexaa_password
}

variable "nexaa_username" {
  description = "Nexaa API username"
  type        = string
  sensitive   = true
}

variable "nexaa_password" {
  description = "Nexaa API password"
  type        = string
  sensitive   = true
}

# Create a namespace first
resource "nexaa_namespace" "example" {
  name = "db-example"
}

# Basic cloud database cluster
resource "nexaa_clouddatabasecluster" "basic" {
  name      = "my-database-cluster"
  namespace = nexaa_namespace.example.name

  spec = {
    type    = "postgresql"
    version = "15"
  }

  plan = {
    cpu      = 1
    memory   = 2.0
    storage  = 10
    replicas = 1
  }
}

# Cloud database cluster with databases and users
resource "nexaa_clouddatabasecluster" "complete" {
  name      = "complete-database-cluster"
  namespace = nexaa_namespace.example.name

  spec = {
    type    = "postgresql"
    version = "15"
  }

  plan = {
    cpu      = 2
    memory   = 4.0
    storage  = 20
    replicas = 1
  }

  databases = [
    {
      name        = "app_database"
      description = "Main application database"
      state       = "present"
    },
    {
      name        = "analytics_db"
      description = "Analytics and reporting database"
      state       = "present"
    }
  ]

  users = [
    {
      name     = "app_user"
      password = "secure_app_password_123"
      state    = "present"
      permissions = [
        {
          database   = "app_database"
          permission = "read"
        },
        {
          database   = "app_database"
          permission = "write"
        }
      ]
    },
    {
      name     = "analytics_user"
      password = "analytics_password_456"
      state    = "present"
      permissions = [
        {
          database   = "analytics_db"
          permission = "read"
        },
        {
          database   = "analytics_db"
          permission = "write"
        },
        {
          database   = "app_database"
          permission = "read"
        }
      ]
    },
    {
      name     = "readonly_user"
      password = "readonly_password_789"
      state    = "present"
      permissions = [
        {
          database   = "app_database"
          permission = "read"
        },
        {
          database   = "analytics_db"
          permission = "read"
        }
      ]
    }
  ]
}

# MySQL cluster example
resource "nexaa_clouddatabasecluster" "mysql" {
  name      = "mysql-cluster"
  namespace = nexaa_namespace.example.name

  spec = {
    type    = "mysql"
    version = "8.0"
  }

  plan = {
    cpu      = 4
    memory   = 8.0
    storage  = 40
    replicas = 2 # Use redundant setup for larger plan
  }

  databases = [
    {
      name        = "wordpress"
      description = "WordPress database"
      state       = "present"
    }
  ]

  users = [
    {
      name     = "wp_user"
      password = "wordpress_password_321"
      state    = "present"
      permissions = [
        {
          database   = "wordpress"
          permission = "admin"
        }
      ]
    }
  ]
}

# Output cluster information
output "basic_cluster_id" {
  value = nexaa_clouddatabasecluster.basic.id
}

output "complete_cluster_databases" {
  value = [for db in nexaa_clouddatabasecluster.complete.databases : db.name]
}

output "complete_cluster_users" {
  value = [for user in nexaa_clouddatabasecluster.complete.users : user.name]
}