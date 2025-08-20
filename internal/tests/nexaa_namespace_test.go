// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/nexaa-cloud/terraform-provider-nexaa/internal/provider"
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"nexaa": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
)

func TestAcc_NamespaceResource_basic(t *testing.T) {
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set for acceptance tests")
	}

	// Generate random test data
	namespaceName := generateTestNamespace()
	description := generateTestDescription()

	t.Logf("=== NAMESPACE TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: givenProvider() + giveNamespace(namespaceName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_namespace.ns", "id"),
					resource.TestCheckResourceAttr("nexaa_namespace.ns", "name", namespaceName),
					resource.TestCheckResourceAttr("nexaa_namespace.ns", "description", description),
					resource.TestCheckResourceAttrSet("nexaa_namespace.ns", "last_updated"),
				),
			},
			{
				ResourceName:            "nexaa_namespace.ns",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}
