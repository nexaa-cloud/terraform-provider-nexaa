// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// internal/tests/volume_test.go

package tests

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_RegistryResource_basic(t *testing.T) {

	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create & Read
			{
				Config: providerConfig + `
				resource "nexaa_namespace" "test" {
				name        = "tf-test-reg1"
				}

				resource "nexaa_registry" "registry" {
				namespace		= "tf-test-reg1"
				name           	= "gitlab"
				source		 	= "registry.gitlab.com"
				username		= "mvangastel"
				password		= "pass"
				verify		 	= false
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_registry.registry", "id"),
					resource.TestCheckResourceAttr("nexaa_registry.registry", "namespace", "tf-test-reg1"),
					resource.TestCheckResourceAttr("nexaa_registry.registry", "name", "gitlab"),
					resource.TestCheckResourceAttr("nexaa_registry.registry", "source", "registry.gitlab.com"),
					resource.TestCheckResourceAttr("nexaa_registry.registry", "username", "mvangastel"),
					resource.TestCheckResourceAttr("nexaa_registry.registry", "verify", "false"),
					resource.TestCheckResourceAttrSet("nexaa_registry.registry", "locked"),
					resource.TestCheckResourceAttrSet("nexaa_registry.registry", "last_updated"),
				),
			},

			// 2) ImportState
			{
				ResourceName:            "nexaa_registry.registry",
				ImportState:             true,
				ImportStateId:           "tf-test-reg1/gitlab",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated", "verify", "password"},
			},
		},
	})
}
