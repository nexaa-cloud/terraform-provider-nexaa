resource "nexaa_starter_container" "starter-container" {
  name      = "tf-starter-container"
  namespace = "terraform"
  image     = "nginx:latest"
  registry  = "gitlab"

  command    = ["nginx", "-g", "daemon off;"]
  entrypoint = ["/docker-entrypoint.sh"]

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
}