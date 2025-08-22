resource "nexaa_container_job" "containerjob" {
  name      = "tf-containerjob"
  namespace = "terraform"
  image     = "nginx:latest"
  registry  = "gitlab"

  resources = {
    cpu = 0.25
    ram = 0.5
  }

  environment_variables = [
    {
      name   = "ENV"
      value  = "production"
      secret = false
    },
    {
      name   = "Variable"
      value  = "terraform"
      secret = false
    },
    {
      name   = "API_KEY"
      value  = "supersecret"
      secret = true
    }
  ]

  mounts = [
    {
      path   = "/storage/mount"
      volume = "storage"
    }
  ]

  schedule_cron_expression = "0 * * * *" # Every hour
  entrypoint               = ["/bin/bash"]
  command                  = ["echo", "Hello World"]
}
