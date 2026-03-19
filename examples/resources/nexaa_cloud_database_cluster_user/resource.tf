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
