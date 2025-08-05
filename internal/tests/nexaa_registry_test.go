// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_RegistryResource_basic(t *testing.T) {
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set")
	}

	// Generate random test data
	namespaceName := generateTestNamespace()
	registryName := generateTestRegistryName()
	username := generateTestUsername()
	password := generateTestPassword()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create & Read
			{
				Config: providerConfig + fmt.Sprintf(`
				resource "nexaa_namespace" "test" {
				name        = %q
				}

				resource "nexaa_registry" "registry" {
				namespace		= %q
				name           	= %q
				source		 	= "registry.gitlab.com"
				username		= %q
				password		= %q
				verify		 	= false
				}
				`, namespaceName, namespaceName, registryName, username, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_registry.registry", "id"),
					resource.TestCheckResourceAttr("nexaa_registry.registry", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_registry.registry", "name", registryName),
					resource.TestCheckResourceAttr("nexaa_registry.registry", "source", "registry.gitlab.com"),
					resource.TestCheckResourceAttr("nexaa_registry.registry", "username", username),
					resource.TestCheckResourceAttr("nexaa_registry.registry", "verify", "false"),
					resource.TestCheckResourceAttrSet("nexaa_registry.registry", "locked"),
					resource.TestCheckResourceAttrSet("nexaa_registry.registry", "last_updated"),
				),
			},

			// 2) ImportState
			{
				ResourceName:            "nexaa_registry.registry",
				ImportState:             true,
				ImportStateId:           fmt.Sprintf("%s/%s", namespaceName, registryName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated", "verify", "password"},
			},
		},
	})
}
