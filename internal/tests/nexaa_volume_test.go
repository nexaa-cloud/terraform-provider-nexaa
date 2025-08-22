// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// internal/tests/volume_test.go

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func volumeConfig(namespaceName, volumeName string, size int) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		givenVolume(volumeName, size)
}

func TestAcc_VolumeResource_basic(t *testing.T) {
	testAccPreCheck(t)

	// Generate random test data
	namespaceName := generateTestNamespace()
	volumeName := generateTestVolumeName()
	initialSize := generateRandomSize()
	updatedSize := initialSize + generateRandomSize() // Ensure updated size is different

	t.Logf("=== VOLUME TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create & Read
			{
				Config: volumeConfig(namespaceName, volumeName, initialSize),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_volume.volume", "id"),
					resource.TestCheckResourceAttr("nexaa_volume.volume", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_volume.volume", "name", volumeName),
					resource.TestCheckResourceAttr("nexaa_volume.volume", "size", fmt.Sprintf("%d", initialSize)),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume", "usage"),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume", "locked"),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume", "last_updated"),
				),
			},

			// 2) ImportState
			{
				ResourceName:            "nexaa_volume.volume",
				ImportState:             true,
				ImportStateId:           fmt.Sprintf("%s/%s", namespaceName, volumeName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},

			// 3) Update & Read
			{
				Config: volumeConfig(namespaceName, volumeName, updatedSize),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_volume.volume", "size", fmt.Sprintf("%d", updatedSize)),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume", "usage"),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume", "locked"),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume", "last_updated"),
				),
			},
		},
	})
}
