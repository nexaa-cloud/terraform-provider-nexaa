// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"math/rand/v2"
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
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Skip("Environment variables NEXAA_USERNAME and NEXAA_PASSWORD must be set for acceptance tests - skipping")
	}
}

// generateRandomString generates a random lowercase string of given length.
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}

// generateResourceName generates a random resource name with prefix.
func generateResourceName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, generateRandomString(8))
}

// generateTestNamespace generates a random namespace name for tests.
func generateTestNamespace() string {
	return generateResourceName("tf-test-ns")
}

func TestAcc_Namespace_basic(t *testing.T) {
	user, pass := os.Getenv("NEXAA_USERNAME"), os.Getenv("NEXAA_PASSWORD")
	namespaceName := generateTestNamespace()

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
						name = "%s"
					}`, user, pass, namespaceName),

				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_namespace.foo", "name", namespaceName),
					resource.TestCheckResourceAttrSet("nexaa_namespace.foo", "id"),
				),
			},
		},
	})
}
