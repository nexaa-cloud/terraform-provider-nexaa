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

func containerWithHttpsDomainIngress(containerName string) string {
	return `
data "nexaa_container_resources" "small" {
  cpu    = 0.25
  memory = 0.5
}
` + fmt.Sprintf(`
resource "nexaa_container" "container" {
  depends_on   = [nexaa_namespace.ns]
  namespace    = nexaa_namespace.ns.name
  name         = %q
  image        = "nginx:latest"
  resources    = data.nexaa_container_resources.small.id
  scaling = {
    type         = "manual"
    manual_input = 1
  }
  ports = ["80:80"]
  ingresses = [
    {
      port        = 80
      domain_name = "https://example.com"
	  tls = true
    }
  ]
}
`, containerName)
}

// --- Backend validation test (passes plan, rejected by API during apply) ---

func TestAcc_ContainerResource_InvalidIngressDomain_HttpsPrefix(t *testing.T) {
	testAccPreCheck(t)
	namespaceName := generateTestNamespace()
	containerName := generateTestContainerName()

	t.Logf("=== CONTAINER HTTPS DOMAIN BACKEND VALIDATION TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create namespace first so it exists in state for cleanup after the expected failure.
			{
				Config: givenProvider() + givenNamespace(namespaceName, ""),
			},
			// Container create passes plan validation but the backend rejects the https:// domain during apply.
			{
				Config:      givenProvider() + givenNamespace(namespaceName, "") + containerWithHttpsDomainIngress(containerName),
				ExpectError: regexp.MustCompile(`(?i)domain|invalid|ingress|https`),
			},
			// Destroy namespace.
			{
				Config:  givenProvider() + givenNamespace(namespaceName, ""),
				Destroy: true,
				PreConfig: func() {
					t.Log("Waiting 10 seconds before destroy...")
					time.Sleep(10 * time.Second)
				},
			},
		},
	})
}
