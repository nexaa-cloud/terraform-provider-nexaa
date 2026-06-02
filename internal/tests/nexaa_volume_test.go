// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"regexp"
	"testing"
	"time"

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
				),
			},

			// 2) ImportState
			{
				ResourceName:            "nexaa_volume.volume",
				ImportState:             true,
				ImportStateId:           fmt.Sprintf("%s/volume/%s", namespaceName, volumeName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated", "status"},
			},

			// 3) Update & Read (increase)
			{
				Config: volumeConfig(namespaceName, volumeName, updatedSize),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_volume.volume", "size", fmt.Sprintf("%d", updatedSize)),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume", "usage"),
					resource.TestCheckResourceAttrSet("nexaa_volume.volume", "locked"),
				),
				PreConfig: func() {
					t.Log("Waiting 10 seconds before update...")
					time.Sleep(10 * time.Second)
				},
			},

			// 4) Attempt to decrease size — API must reject this
			{
				Config:      volumeConfig(namespaceName, volumeName, initialSize),
				ExpectError: regexp.MustCompile(`Error Updating Volume`),
				PreConfig: func() {
					t.Log("Waiting 10 seconds before decrease attempt...")
					time.Sleep(10 * time.Second)
				},
			},

			{
				Config:  volumeConfig(namespaceName, volumeName, updatedSize),
				Destroy: true,
				PreConfig: func() {
					t.Log("Waiting 10 seconds before destroy...")
					time.Sleep(10 * time.Second)
				},
			},
		},
	})
}

func TestAcc_VolumeResource_NameTooLong(t *testing.T) {
	testAccPreCheck(t)
	namespaceName := generateTestNamespace()
	t.Logf("=== VOLUME NAME VALIDATION TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: givenProvider() + givenNamespace(namespaceName, ""),
			},
			{
				Config:      givenProvider() + givenNamespace(namespaceName, "") + givenVolume("toolongvolname", 1),
				ExpectError: regexp.MustCompile(`(?i)name|length|long`),
			},
			{
				Config:  givenProvider() + givenNamespace(namespaceName, ""),
				Destroy: true,
			},
		},
	})
}
