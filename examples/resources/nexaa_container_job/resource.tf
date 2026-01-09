resource "nexaa_container_job" "containerjob" {
  ## We need a namespace before we can create a container. Therefor create a dependancy on the namespace
  depends_on = [
    nexaa_namespace.test,
  ]

  ## Define your name, namespace, image and (if required) add registry credentials
  name      = "tf-containerjob"
  namespace = "terraform-test"
  image     = "busybox:1.28"
  registry  = null

  ## Set your cron schedule in crontab format
  schedule_cron_expression = "0 * * * *" # Every hour

  ## With command and entrypoint you can override the startup behaviour of your container
  command = ["/bin/sh", "-c", "date; echo Hello World"]
  # entrypoint = ["/docker-entrypoint.sh"]

  resources = {
    cpu = 0.25
    ram = 0.5
  }

  ## Adding environment variables to your container
  ## When setting it as secret, it will be encrypted
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

  ## When using volumes you can mount the volume on a specific path
  # mounts = [
  #   {
  #     path   = "/storage/mount"
  #     volume = nexaa_volume.volume.name
  #   }
  # ]

}
