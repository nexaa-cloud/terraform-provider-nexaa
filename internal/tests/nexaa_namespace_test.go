// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/nexaa-cloud/terraform-provider-nexaa/internal/provider"
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"nexaa": providerserver.NewProtocol6WithError(provider.New("test")()),
	}

	providerConfig = fmt.Sprintf(
		`provider "nexaa" {
				username = %q
				password = %q
			}
			`,
		os.Getenv("USERNAME"),
		os.Getenv("PASSWORD"),
	)
)

func TestAcc_NamespaceResource_basic(t *testing.T) {
	if os.Getenv("USERNAME") == "" || os.Getenv("PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set for acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				resource "nexaa_namespace" "test" {
				name        = "tf-test-ns-%s"
				description = "A BDD-style test namespace"
				}
				`, time.Now()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_namespace.test", "id"),
					resource.TestCheckResourceAttr("nexaa_namespace.test", "name", "tf-test-ns"),
					resource.TestCheckResourceAttr("nexaa_namespace.test", "description", "A BDD-style test namespace"),
					resource.TestCheckResourceAttrSet("nexaa_namespace.test", "last_updated"),
				),
			},
			{
				ResourceName:            "nexaa_namespace.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}
