# Container Job

## Creating a container job

You can use the plan below to create a simple container job which runs every hour.

```tf
resource "nexaa_namespace" "ns" {
   name        = "namespace-name"
   description = "This is a description"
}

resource "nexaa_container_job" "containerjob" {
  name      = "container-job"
  namespace = nexaa_namespace.ns.name
  image     = "busybox:1.28"

  resources = {
    cpu = 0.25
    ram = 0.5
  }

  command = [
     "/bin/sh",
     "-c",
     "date; echo Hello from Nexaa"
  ]
  schedule_cron_expression = "0 * * * *" # Every hour
}
```