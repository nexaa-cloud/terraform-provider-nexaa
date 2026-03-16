data "nexaa_container_resources" "container_resource" {
  cpu    = 0.25
  memory = 0.5
}

resource "nexaa_container" "container" {
  ## We need a namespace before we can create a container. Therefor create a dependancy on the namespace
  depends_on = [
    nexaa_namespace.test,
  ]

  ## Define your name, namespace, image and (if required) add registry credentials
  name      = "tf-container"
  namespace = "terraform-test"
  image     = "nginx:latest"
  registry  = null

  ## With command and entrypoint you can override the startup behaviour of your container
  # command    = ["nginx", "-g", "daemon off;"]
  # entrypoint = ["/docker-entrypoint.sh"]

  resources = data.nexaa_container_resources.container_resource.id

  ## Exposing ports from the container.
  ## This is required when you want to communicate from outside the container to this container
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
  # mounts = [
  #   {
  #     path   = "/storage/mount"
  #     volume = "storage"
  #   }
  # ]

  # The health check will check your container if the application is still responding as it should
  health_check = {
    port = 80
    path = "/"
  }

  ## With scaling you can scale horizontal automatically or manual
  ## When automatically it will scale based on the triggers
  ## When manual you have to change the number of replicas yourself
  scaling = {
    type = "auto"

    # manual_input = 1
    auto_input = {
      minimal_replicas = 1
      maximal_replicas = 3

      triggers = [
        {
          type      = "CPU"
          threshold = 70
        },
        {
          type      = "MEMORY"
          threshold = 80
        }
      ]
    }
  }
}