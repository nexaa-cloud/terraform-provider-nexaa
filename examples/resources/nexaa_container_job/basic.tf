provider "nexaa" {
  username = "your-nexaa-username"
  password = "your-nexaa-password"
}

resource "nexaa_namespace" "project" {
  name        = "project-name"
  description = "This is an optional description"
}

resource "nexaa_container_job" "containerjob" {
  name      = "container-job-name"
  namespace = nexaa_namespace.project.name
  image     = "busybox:1.28"

  resources = {
    cpu = 0.25
    ram = 0.5
  }

  schedule_cron_expression = "0 * * * *" # Every hour
}
