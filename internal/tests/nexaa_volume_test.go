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

func volumeConfig(size int) string {
	return providerConfig + fmt.Sprintf(`
        resource "nexaa_namespace" "test" {
        name        = "tf-test-vol1"
        }

        resource "nexaa_volume" "volume1" {
        namespace      = "tf-test-vol1"
        name           = "tf-vol1"
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
					resource.TestCheckResourceAttr("nexaa_volume.volume1", "namespace", "tf-test-vol1"),
					resource.TestCheckResourceAttr("nexaa_volume.volume1", "name", "tf-vol1"),
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
				ImportStateId:           "tf-test-vol1/tf-vol1",
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
