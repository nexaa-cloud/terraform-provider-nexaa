// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// internal/tests/volume_test.go

package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// A small reusable config: just the provider + namespace block,
// we append the volume resource snippet in each step.
var baseConfig = `
              terraform {
                required_providers {
                  nexaa = { source = "nexaa", version = "0.1.0" }
                }
              }

              provider "nexaa" {
                username = "` + os.Getenv("NEXAA_USERNAME") + `"
                password = "` + os.Getenv("NEXAA_PASSWORD") + `"
              }

              resource "nexaa_namespace" "test" {
                name        = "tf-test-ns-vol"
              }
              `

// volumeConfig returns baseConfig + a volume block with the given size.
func volumeConfig(size int) string {
    return baseConfig + fmt.Sprintf(`
resource "nexaa_volume" "volume1" {
  namespace_name = nexaa_namespace.test.name
  name           = "tf-test-vol"
  size           = %d
}
`, size)
}

func TestAcc_VolumeResource_basic(t *testing.T) {
    if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
        t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set")
    }

    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // 1) Create & Read
            {
                Config: volumeConfig(3),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "id"),
                    resource.TestCheckResourceAttr("nexaa_volume.volume1", "namespace_name", "tf-test-ns-vol"),
                    resource.TestCheckResourceAttr("nexaa_volume.volume1", "name", "tf-test-vol"),
                    resource.TestCheckResourceAttr("nexaa_volume.volume1", "size", "3"),
                    resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "usage"),
                    resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "locked"),
                    resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "last_updated"),
                ),
            },

            // 2) ImportState
            {
                ResourceName:            "nexaa_volume.volume1",
                ImportState:             true,
                ImportStateId:           "tf-test-ns-vol/tf-test-vol",
                ImportStateVerify:       true,
                ImportStateVerifyIgnore: []string{"last_updated"},
            },

            // 3) Update & Read
            {
                Config: volumeConfig(5),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("nexaa_volume.volume1", "size", "5"),
                    resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "usage"),
                    resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "locked"),
                    resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "last_updated"),
                ),
            },
        },
    })
}

