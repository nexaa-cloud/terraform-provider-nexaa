data "nexaa_cloud_database_cluster_plans" "plan" {
  cpu      = 1
  memory   = 2.0
  storage  = 10
  replicas = 1
}

# Basic cloud database cluster
resource "nexaa_cloud_database_cluster" "cluster" {
  depends_on = [
    nexaa_namespace.test
  ]
  cluster = {
    name      = "tf-db-cluster"
    namespace = "terraform-test"
  }

  spec = {
    type    = "PostgreSQL"
    version = "17.5"
  }

  plan = data.nexaa_cloud_database_cluster_plans.plan.id

  external_connection = {
    ports = {
      allowlist = ["192.168.1.1"]
    }
  }
}