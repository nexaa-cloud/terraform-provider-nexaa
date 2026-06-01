// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_NamespaceResource_basic(t *testing.T) {
	testAccPreCheck(t)

	// Generate random test data
	namespaceName := generateTestNamespace()
	description := generateTestDescription()

	t.Logf("=== NAMESPACE TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: givenProvider() + givenNamespace(namespaceName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_namespace.ns", "id"),
					resource.TestCheckResourceAttr("nexaa_namespace.ns", "name", namespaceName),
					resource.TestCheckResourceAttr("nexaa_namespace.ns", "description", description),
				),
			},
			{
				ResourceName:            "nexaa_namespace.ns",
				ImportState:             true,
				ImportStateId:           namespaceName,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Attempt to update the description — provider must reject this
			{
				Config:      givenProvider() + givenNamespace(namespaceName, "a different description"),
				ExpectError: regexp.MustCompile(`You can't change your namespace`),
			},
			{
				Config:  givenProvider() + givenNamespace(namespaceName, description),
				Destroy: true,
				PreConfig: func() {
					t.Log("Waiting 30 seconds before destroy...")
					time.Sleep(30 * time.Second)
				},
			},
		},
	})
}
