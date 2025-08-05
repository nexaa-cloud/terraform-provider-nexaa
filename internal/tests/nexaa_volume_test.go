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

func volumeConfig(namespaceName, volumeName string, size int) string {
	return providerConfig + fmt.Sprintf(`
        resource "nexaa_namespace" "test" {
        name        = %q
        }

        resource "nexaa_volume" "volume1" {
        namespace      = %q
        name           = %q
        size           = %d
        }
        `, namespaceName, namespaceName, volumeName, size)
}

func TestAcc_VolumeResource_basic(t *testing.T) {
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set")
	}

	// Generate random test data
	namespaceName := generateTestNamespace()
	volumeName := generateTestVolumeName()
	initialSize := generateRandomSize()
	updatedSize := initialSize + generateRandomSize() // Ensure updated size is different

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create & Read
			{
				Config: volumeConfig(namespaceName, volumeName, initialSize),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "id"),
					resource.TestCheckResourceAttr("nexaa_volume.volume1", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("nexaa_volume.volume1", "size", fmt.Sprintf("%d", initialSize)),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "usage"),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "locked"),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "last_updated"),
				),
			},

			// 2) ImportState
			{
				ResourceName:            "nexaa_volume.volume1",
				ImportState:             true,
				ImportStateId:           fmt.Sprintf("%s/%s", namespaceName, volumeName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},

			// 3) Update & Read
			{
				Config: volumeConfig(namespaceName, volumeName, updatedSize),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_volume.volume1", "size", fmt.Sprintf("%d", updatedSize)),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "usage"),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "locked"),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume1", "last_updated"),
				),
			},
		},
	})
}
