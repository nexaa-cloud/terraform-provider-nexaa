// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"nexaa": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("USERNAME") == "" || os.Getenv("PASSWORD") == "" {
		t.Fatal("Environment variables NEXAA_USERNAME and NEXAA_PASSWORD must be set")
	}
}

func TestAcc_Namespace_basic(t *testing.T) {
	user, pass := os.Getenv("USERNAME"), os.Getenv("PASSWORD")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					provider "nexaa" {
						username = "%s"
						password = "%s"
					}
					resource "nexaa_namespace" "foo" {
						name = "tf-test-provider"
					}`, user, pass),

				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_namespace.foo", "name", "tf-test-provider"),
					resource.TestCheckResourceAttrSet("nexaa_namespace.foo", "id"),
				),
			},
		},
	})
}
