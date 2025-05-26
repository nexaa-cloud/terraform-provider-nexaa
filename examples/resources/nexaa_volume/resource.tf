resource "nexaa_volume" "volume-test" {
  namespace = "terraform-test"
  name      = "terraform-volume"
  size      = 1
}