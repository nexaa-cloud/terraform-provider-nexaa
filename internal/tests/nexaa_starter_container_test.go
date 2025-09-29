// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func starterContainerConfig(namespaceName, containerName, registryName, registryUsername, registryPassword, envVar, envValue, healthPath string) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		givenRegistry(registryName, registryUsername, registryPassword) +
		fmt.Sprintf(`
resource "nexaa_starter_container" "starter_container" {
  depends_on = [nexaa_registry.registry]
  name      = %q
  namespace = nexaa_namespace.ns.name
  image     = "nginx:latest"
  registry  = null

  command = ["nginx", "-g", "daemon off;"]
  entrypoint = ["/docker-entrypoint.sh"]

  ports = ["80:80"]

  environment_variables = [
    {
      name   = %q
      value  = %q
      secret = false
    }
  ]

  ingresses = [
    {
      domain_name = null
      port        = 80
      tls         = true
      allow_list  = ["0.0.0.0/0"]
    }
  ]

  health_check = {
    port = 80
    path = %q
  }
}
`, containerName, envVar, envValue, healthPath)
}

func starterContainerUpdateConfig(namespaceName, containerName, registryName, registryUsername, registryPassword, envVar1, envValue1, envVar2, envValue2, healthPath string, port int) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		givenRegistry(registryName, registryUsername, registryPassword) +
		fmt.Sprintf(`
resource "nexaa_starter_container" "starter_container" {
  depends_on = [nexaa_registry.registry]
  name      = %q
  namespace = nexaa_namespace.ns.name
  image     = "nginx:alpine"
  registry  = %q

  command = ["nginx", "-g", "daemon off;", "-c", "/etc/nginx/nginx.conf"]
  entrypoint = ["/docker-entrypoint.sh"]

  ports = ["80:80", "%d:%d"]

  environment_variables = [
    {
      name   = %q
      value  = %q
      secret = false
    },
    {
      name   = %q
      value  = %q
      secret = false
    }
  ]

  ingresses = [
    {
      domain_name = null
      port        = 80
      tls         = true
      allow_list  = ["0.0.0.0/0"]
    }
  ]

  health_check = {
    port = 80
    path = %q
  }
}
`, containerName, registryName, port, port, envVar1, envValue1, envVar2, envValue2, healthPath)
}

func TestAcc_StarterContainerResource_basic(t *testing.T) {
	testAccPreCheck(t)

	// Generate random test data
	namespaceName := generateTestNamespace()
	containerName := generateTestContainerName()
	registryName := generateTestRegistryName()
	registryUsername := generateTestUsername()
	registryPassword := generateTestPassword()
	envVar1 := generateTestEnvVar()
	envValue1 := generateTestEnvValue()
	healthPath1 := generateTestPath()

	t.Logf("=== STARTER CONTAINER TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create
			{
				Config: starterContainerConfig(namespaceName, containerName, registryName, registryUsername, registryPassword, envVar1, envValue1, healthPath1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_starter_container.starter_container", "id"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "name", containerName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "image", "nginx:latest"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "command.#", "3"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "command.0", "nginx"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "command.1", "-g"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "command.2", "daemon off;"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "entrypoint.#", "1"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "entrypoint.0", "/docker-entrypoint.sh"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ports.#", "1"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "environment_variables.#", "1"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.#", "1"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "health_check.port", "80"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "health_check.path", "/"),
					// Verify that scaling and resources fields don't exist
					resource.TestCheckNoResourceAttr("nexaa_starter_container.starter_container", "scaling"),
					resource.TestCheckNoResourceAttr("nexaa_starter_container.starter_container", "resources"),
				),
			},

			// 2) ImportState
			{
				ResourceName:      "nexaa_starter_container.starter_container",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s", namespaceName, containerName),
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"registry",
					"mounts",
					"ingresses.0.domain_name",
					"last_updated",
					"status",
				},
			},
		},
	})
}

func minimalStarterContainerConfig(namespaceName, containerName string) string {
	return givenProvider() + givenNamespace(namespaceName, "") + fmt.Sprintf(`
resource "nexaa_starter_container" "starter_container" {
  depends_on = [nexaa_namespace.ns]
  name      = %q
  namespace = nexaa_namespace.ns.name
  image     = "nginx:latest"
  registry  = null
}
`, containerName)
}

func TestAcc_StarterContainerResource_Minimal(t *testing.T) {
	testAccPreCheck(t)

	// Generate random test data
	namespaceName := generateTestNamespace()
	containerName := generateTestContainerName()

	t.Logf("=== STARTER CONTAINER MINIMAL TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create with minimal config
			{
				Config: minimalStarterContainerConfig(namespaceName, containerName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_starter_container.starter_container", "id"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "name", containerName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "image", "nginx:latest"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.#", "0"),
					// Verify that scaling and resources fields don't exist
					resource.TestCheckNoResourceAttr("nexaa_starter_container.starter_container", "scaling"),
					resource.TestCheckNoResourceAttr("nexaa_starter_container.starter_container", "resources"),
				),
			},
			// 2) Apply the same config again - should result in no changes
			{
				Config:   minimalStarterContainerConfig(namespaceName, containerName),
				PlanOnly: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_starter_container.starter_container", "id"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "name", containerName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "image", "nginx:latest"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.#", "0"),
				),
			},
		},
	})
}

func minimalStarterContainerWithIngressConfig(namespaceName, containerName string) string {
	return givenProvider() + givenNamespace(namespaceName, "") + fmt.Sprintf(`
resource "nexaa_starter_container" "starter_container" {
  depends_on = [nexaa_namespace.ns]
  name      = %q
  namespace = nexaa_namespace.ns.name
  image     = "nginx:latest"
  registry  = null

  ports = ["80:80"]

  ingresses = [
    {
      port        = 80
      tls         = true
      allow_list  = ["0.0.0.0/0"]
    }
  ]
}
`, containerName)
}

func TestAcc_StarterContainerResource_IngressDomainNamePlanStability(t *testing.T) {
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set")
	}

	// Generate random test data
	namespaceName := generateTestNamespace()
	containerName := generateTestContainerName()

	t.Logf("=== STARTER CONTAINER INGRESS DOMAIN NAME PLAN STABILITY TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create with minimal config (domain_name omitted)
			{
				Config: minimalStarterContainerWithIngressConfig(namespaceName, containerName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_starter_container.starter_container", "id"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "name", containerName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "image", "nginx:latest"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.#", "1"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.0.port", "80"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.0.tls", "true"),
					// domain_name should be computed and set by the provider
					resource.TestCheckResourceAttrSet("nexaa_starter_container.starter_container", "ingresses.0.domain_name"),
					// Verify that scaling and resources fields don't exist
					resource.TestCheckNoResourceAttr("nexaa_starter_container.starter_container", "scaling"),
					resource.TestCheckNoResourceAttr("nexaa_starter_container.starter_container", "resources"),
				),
			},
			// 2) Apply the same config again - should result in no changes
			{
				Config:   minimalStarterContainerWithIngressConfig(namespaceName, containerName),
				PlanOnly: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_starter_container.starter_container", "id"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "name", containerName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "image", "nginx:latest"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.#", "1"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.0.port", "80"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.0.tls", "true"),
					// domain_name should remain the same computed value
					resource.TestCheckResourceAttrSet("nexaa_starter_container.starter_container", "ingresses.0.domain_name"),
				),
			},
		},
	})
}

func minimalStarterContainerWithDomainNameConfig(namespaceName, containerName string, domainName string) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		fmt.Sprintf(`
resource "nexaa_starter_container" "starter_container" {
  depends_on = [nexaa_namespace.ns]
  name      = %q
  namespace = nexaa_namespace.ns.name
  image     = "nginx:latest"
  registry  = null

  ports = ["80:80"]

  ingresses = [
    {
      domain_name = %q
      port        = 80
      tls         = true
      allow_list  = ["0.0.0.0/0"]
    }
  ]
}
`, containerName, domainName)
}

func TestAcc_StarterContainerResource_IngressDomainNameChangeReplaceExisting(t *testing.T) {
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set")
	}

	// Generate random test data
	namespaceName := generateTestNamespace()
	containerName := generateTestContainerName()

	t.Logf("=== STARTER CONTAINER INGRESS DOMAIN NAME CHANGE TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: minimalStarterContainerWithDomainNameConfig(namespaceName, containerName, "example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_starter_container.starter_container", "id"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "name", containerName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "image", "nginx:latest"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.#", "1"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.0.port", "80"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.0.tls", "true"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.0.domain_name", "example.com"),
					// Verify that scaling and resources fields don't exist
					resource.TestCheckNoResourceAttr("nexaa_starter_container.starter_container", "scaling"),
					resource.TestCheckNoResourceAttr("nexaa_starter_container.starter_container", "resources"),
				),
			},
			{
				Config: minimalStarterContainerWithDomainNameConfig(namespaceName, containerName, "example.org"),

				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_starter_container.starter_container", "id"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "name", containerName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "image", "nginx:latest"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.#", "1"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.0.port", "80"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.0.tls", "true"),
					resource.TestCheckResourceAttr("nexaa_starter_container.starter_container", "ingresses.0.domain_name", "example.org"),
				),
			},
		},
	})
}
