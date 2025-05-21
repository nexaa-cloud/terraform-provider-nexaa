// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/nexaa-cloud/terraform-provider-nexaa/internal/provider"
)

var (
    testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
        "nexaa": providerserver.NewProtocol6WithError(provider.New("test")()),
    }

    providerConfig = fmt.Sprintf(
        `terraform {
			required_providers {
				nexaa = { source = "nexaa", version = "0.1.0" }
				}
			}

			provider "nexaa" {
				username = %q
				password = %q
			}
			`,
        os.Getenv("NEXAA_USERNAME"),
        os.Getenv("NEXAA_PASSWORD"),
    )
)

func TestAcc_NamespaceResource_basic(t *testing.T) {
    if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
        t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set for acceptance tests")
    }

    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: providerConfig + `
				resource "nexaa_namespace" "test" {
				name        = "tf-test-ns-123"
				description = "A BDD-style test namespace"
				}
				`,
                ConfigStateChecks: []statecheck.StateCheck{
                    statecheck.ExpectKnownValue(
                        "nexaa_namespace.test",
                        tfjsonpath.New("name"),
                        knownvalue.StringExact("tf-test-ns-123"),
                    ),
                    statecheck.ExpectKnownValue(
                        "nexaa_namespace.test",
                        tfjsonpath.New("id"),
                        knownvalue.StringExact("tf-test-ns-123"),
                    ),
                    statecheck.ExpectKnownValue(
                        "nexaa_namespace.test",
                        tfjsonpath.New("description"),
                        knownvalue.StringExact("A BDD-style test namespace"),
                    ),
                    statecheck.ExpectKnownValue(
                        "nexaa_namespace.test",
                        tfjsonpath.New("last_updated"),
                        knownvalue.StringRegexp(regexp.MustCompile(`^.+$`)),
                    ),
                },
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
