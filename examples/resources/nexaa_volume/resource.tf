resource "nexaa_volume" "volume-test" {
  namespace_name = "terraform-test"
  name           = "terraform-volume"
  size           = 1
}