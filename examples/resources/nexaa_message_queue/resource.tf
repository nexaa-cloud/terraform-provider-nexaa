# Find a plan matching the desired resources
data "nexaa_message_queue_plans" "plan" {
  cpu      = 1
  memory   = 2
  storage  = 10
  replicas = 1
}

# Create a RabbitMQ message queue
resource "nexaa_message_queue" "queue" {
  ## We need a namespace before we can create a container. Therefor create a dependancy on the namespace
  depends_on = [
    nexaa_namespace.test,
  ]
  namespace = "terraform-test"
  name      = "tf-queue"
  plan      = data.nexaa_message_queue_plans.plan.id
  type      = "RabbitMQ"
  version   = "3.13"
  allowlist = ["192.168.1.1"]
  external_connection = {
    ports = {
      internal_port = 5672
      protocol      = "TCP"
      allowlist     = ["192.168.1.1/32"]
    }
  }
}
