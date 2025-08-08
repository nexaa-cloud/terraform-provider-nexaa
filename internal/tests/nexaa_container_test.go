// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func containerConfig(namespaceName, containerName, registryName, registryUsername, registryPassword, envVar, envValue, healthPath string) string {
	return providerConfig + fmt.Sprintf(`
resource "nexaa_namespace" "ns" {
  name = %q
}

resource "nexaa_registry" "registry" {
  depends_on = [nexaa_namespace.ns]
  namespace = nexaa_namespace.ns.name
  name      = %q
  source    = "registry.gitlab.com"
  username  = %q
  password  = %q
  verify    = false
}

resource "nexaa_container" "container" {
  depends_on = [nexaa_registry.registry]
  name      = %q
  namespace = nexaa_namespace.ns.name
  image     = "nginx:latest"
  registry  = null

  resources = {
    cpu = 0.25
    ram = 0.5
  }

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

  scaling = {
    type = "auto"
    auto_input = {
      minimal_replicas = 1
      maximal_replicas = 3

      triggers = [
        {
          type      = "CPU"
          threshold = 70
        },
        {
          type      = "MEMORY"
          threshold = 80
        }
      ]
    }
  }
}
`, namespaceName, registryName, registryUsername, registryPassword, containerName, envVar, envValue, healthPath)
}

func containerUpdateConfig(namespaceName, containerName, registryName, registryUsername, registryPassword, envVar1, envValue1, envVar2, envValue2, healthPath string, port int) string {
	return providerConfig + fmt.Sprintf(`
resource "nexaa_namespace" "ns" {
  name = %q
}

resource "nexaa_registry" "registry" {
  depends_on = [nexaa_namespace.ns]
  namespace = nexaa_namespace.ns.name
  name      = %q
  source    = "registry.gitlab.com"
  username  = %q
  password  = %q
  verify    = false
}

resource "nexaa_container" "container" {
  depends_on = [nexaa_registry.registry]
  name      = %q
  namespace = nexaa_namespace.ns.name
  image     = "nginx:alpine"
  registry  = %q

  resources = {
    cpu = 0.5
    ram = 1.0
  }

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

  scaling = {
    type = "manual"
    manual_input = 3
  }
}
`, namespaceName, registryName, registryUsername, registryPassword, containerName, registryName, port, port, envVar1, envValue1, envVar2, envValue2, healthPath)
}

func TestAcc_ContainerResource_basic(t *testing.T) {
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set")
	}

	// Generate random test data
	namespaceName := generateTestNamespace()
	containerName := generateTestContainerName()
	registryName := generateTestRegistryName()
	registryUsername := generateTestUsername()
	registryPassword := generateTestPassword()
	envVar1 := generateTestEnvVar()
	envValue1 := generateTestEnvValue()
	envVar2 := generateTestEnvVar() + "2"
	envValue2 := generateTestEnvValue()
	healthPath1 := generateTestPath()
	healthPath2 := generateTestPath()
	randomPort := generateRandomPort()

	t.Logf("=== CONTAINER TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create
			{
				Config: containerConfig(namespaceName, containerName, registryName, registryUsername, registryPassword, envVar1, envValue1, healthPath1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_container.container", "id"),
					resource.TestCheckResourceAttr("nexaa_container.container", "name", containerName),
					resource.TestCheckResourceAttr("nexaa_container.container", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_container.container", "image", "nginx:latest"),
					resource.TestCheckResourceAttr("nexaa_container.container", "resources.cpu", "0.25"),
					resource.TestCheckResourceAttr("nexaa_container.container", "resources.ram", "0.5"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ports.#", "1"),
					resource.TestCheckResourceAttr("nexaa_container.container", "environment_variables.#", "1"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ingresses.#", "1"),
					resource.TestCheckResourceAttr("nexaa_container.container", "health_check.port", "80"),
					resource.TestCheckResourceAttr("nexaa_container.container", "health_check.path", "/"),
					resource.TestCheckResourceAttr("nexaa_container.container", "scaling.type", "auto"),
					resource.TestCheckResourceAttr("nexaa_container.container", "scaling.auto_input.minimal_replicas", "1"),
					resource.TestCheckResourceAttr("nexaa_container.container", "scaling.auto_input.maximal_replicas", "3"),
					resource.TestCheckResourceAttr("nexaa_container.container", "scaling.auto_input.triggers.#", "2"),
				),
			},

			// 2) ImportState
			{
				ResourceName:      "nexaa_container.container",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s", namespaceName, containerName),
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"registry",
					"mounts",
					"environment_variables.0.value",
					"environment_variables.1.value",
					"ingresses.0.domain_name",
					"last_updated",
					"status",
				},
			},

			// 3) Update
			{
				Config: containerUpdateConfig(namespaceName, containerName, registryName, registryUsername, registryPassword, envVar1, envValue1, envVar2, envValue2, healthPath2, randomPort),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_container.container", "image", "nginx:alpine"),
					resource.TestCheckResourceAttr("nexaa_container.container", "registry", registryName),
					resource.TestCheckResourceAttr("nexaa_container.container", "resources.cpu", "0.5"),
					resource.TestCheckResourceAttr("nexaa_container.container", "resources.ram", "1"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ports.#", "2"),
					resource.TestCheckResourceAttr("nexaa_container.container", "environment_variables.#", "2"),
					resource.TestCheckResourceAttr("nexaa_container.container", "environment_variables.0.name", envVar1),
					resource.TestCheckResourceAttr("nexaa_container.container", "environment_variables.0.value", envValue1),
					resource.TestCheckResourceAttr("nexaa_container.container", "environment_variables.1.name", envVar2),
					resource.TestCheckResourceAttr("nexaa_container.container", "environment_variables.1.value", envValue2),
					resource.TestCheckResourceAttr("nexaa_container.container", "ingresses.#", "1"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ingresses.0.port", "80"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ingresses.0.tls", "true"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ingresses.0.allow_list.0", "0.0.0.0/0"),
					resource.TestCheckResourceAttr("nexaa_container.container", "mounts.#", "0"),
					resource.TestCheckResourceAttr("nexaa_container.container", "health_check.port", "80"),
					resource.TestCheckResourceAttr("nexaa_container.container", "health_check.path", healthPath2),
					resource.TestCheckResourceAttr("nexaa_container.container", "scaling.type", "manual"),
					resource.TestCheckResourceAttr("nexaa_container.container", "scaling.manual_input", "3"),
				),
			},
		},
	})
}
