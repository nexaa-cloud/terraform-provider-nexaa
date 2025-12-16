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
}

variable "queue_name" {
  description = "Message queue name"
  type        = string
}

provider "nexaa" {
  username = var.nexaa_username
  password = var.nexaa_password
}

resource "nexaa_namespace" "example" {
  name        = var.namespace
  description = "Example namespace for message queue"
}

# Find a plan matching the desired resources
data "nexaa_message_queue_plans" "plan" {
  cpu      = 1
  memory   = 2
  storage  = 10
  replicas = 1
}

# Create a RabbitMQ message queue
resource "nexaa_message_queue" "queue" {
  namespace = nexaa_namespace.example.name
  name      = var.queue_name
  plan      = data.nexaa_message_queue_plans.plan.id
  type      = "RabbitMQ"
  version   = "3.13"
}

output "queue_id" {
  description = "The ID of the message queue"
  value       = nexaa_message_queue.queue.id
}

output "queue_state" {
  description = "The current state of the message queue"
  value       = nexaa_message_queue.queue.state
}
