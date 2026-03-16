resource "nexaa_starter_container" "starter-container" {
  depends_on = [
    nexaa_namespace.test,
  ]

  ## Define your name, namespace, image and (if required) add registry credentials
  name      = "tf-starter-container"
  namespace = "terraform-test"
  image     = "nginx:latest"
  registry  = null

  ## With command and entrypoint you can override the startup behaviour of your container
  #command    = ["nginx", "-g", "daemon off;"]
  #entrypoint = ["/docker-entrypoint.sh"]

  ports = ["80:80", "8080"]

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

  ## When you want to expose your container to the internet you can add an ingress
  ingresses = [
    {
      domain_name = null
      port        = 80
      tls         = true
      allowlist   = ["0.0.0.0/0", "::/0"]
    }
  ]

  ## When you want to expose your container to the internet you can add an external connection
  external_connection = {
    ports = [{
      internal_port = 8080
      protocol      = "TCP"
      allowlist     = ["192.168.1.1/32"]
    }]
  }


  ## When using volumes you can mount the volume on a specific path
  #mounts = [
  #  {
  #    path   = "/storage/mount"
  #    volume = "storage"
  #  }
  #]

  # The health check will check your container if the application is still responding as it should
  health_check = {
    port = 80
    path = "/"
  }
}