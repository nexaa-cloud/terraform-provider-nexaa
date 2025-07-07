resource "nexaa_container" "container" {
  name      = "tf-container"
  namespace = "terraform"
  image     = "nginx:latest"
  registry  = "gitlab"

  resources = {
    cpu = 0.25
    ram = 0.5
  }

  ports = ["8000:8000", "80:80"]

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

  ingresses = [
    {
      domain_name = null
      port        = 80
      tls         = true
      allow_list  = ["0.0.0.0/0"]
    }
  ]

  mounts = [
    {
      path   = "/storage/mount"
      volume = "storage"
    }
  ]

  health_check = {
    port = 80
    path = "/storage/health"
  }

  scaling = {
    type = "auto"

    #manual_input = 1
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