resource "nexaa_registry" "registry" {
  namespace = "namespace"
  name      = "example"
  source    = "registry.gitlab.com"
  username  = "user"
  password  = "pass"
  verify    = true
}