resource "nexaa_cloud_database_cluster_database" "database" {
  depends_on = [
    nexaa_cloud_database_cluster.cluster,
  ]

  cluster = nexaa_cloud_database_cluster.cluster.cluster
  name    = "myDatabase"

}
