resource "nexaa_registry" "registry" {
  ## We need a namespace before we can create a container. Therefor create a dependancy on the namespace
  depends_on = [
    nexaa_namespace.test,
  ]
  namespace = "terraform-test"
  name      = "example"
  source    = "registry.gitlab.com"
  username  = "user"
  password  = "pass"
  verify    = true
}