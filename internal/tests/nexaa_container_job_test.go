// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func containerJobConfig(namespaceName string, registryName string, registryUsername string, registryPassword string, containerJobName string, image string, entrypoint string, command string, schedule string) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		givenRegistry(registryName, registryUsername, registryPassword) +
		fmt.Sprintf(`
data "nexaa_container_resources" "job" {
  cpu    = 0.25
  memory = 0.5
}

resource "nexaa_container_job" "job" {
  depends_on = [nexaa_registry.registry, nexaa_namespace.ns]
  namespace  = nexaa_namespace.ns.name
  name       = %q
  image      = %q
  registry   = null
  enabled    = false
  command    = %s
  entrypoint = %s
  resources  = data.nexaa_container_resources.job.id
  schedule   = %q
}
`, containerJobName, image, command, entrypoint, schedule)
}

func containerJobUpdateConfig(namespaceName string, registryName string, registryUsername string, registryPassword string, containerJobName string, image string, entrypoint string, command string, schedule string) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		givenRegistry(registryName, registryUsername, registryPassword) +
		fmt.Sprintf(`
data "nexaa_container_resources" "job" {
  cpu    = 0.25
  memory = 0.5
}

resource "nexaa_container_job" "job" {
  depends_on = [nexaa_registry.registry, nexaa_namespace.ns]
  namespace  = nexaa_namespace.ns.name
  name       = %q
  image      = %q
  registry   = %q
  enabled    = false
  command    = %s
  entrypoint = %s
  resources  = data.nexaa_container_resources.job.id
  schedule   = %q
}
`, containerJobName, image, registryName, command, entrypoint, schedule)
}

func TestAcc_ContainerJobResource_public_registry(t *testing.T) {
	testAccPreCheck(t)

	namespaceName := generateTestNamespace()
	containerJobName := generateTestContainerJobName()
	entrypoint := generateTestEntrypoint()
	command := generateTestCommands()
	schedule := generateTestSchedule()

	t.Logf("=== CONTAINER JOB PUBLIC REGISTRY TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: givenProvider() + givenNamespace(namespaceName, "") +
					givenContainerJobPublic(containerJobName, "nginx:latest", command, entrypoint, schedule),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_container_job.job", "id"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "image", "nginx:latest"),
					resource.TestCheckNoResourceAttr("nexaa_container_job.job", "registry"),
				),
			},
			// ImportState with no private registry — exercises the nil pointer fix in ImportState
			{
				ResourceName:      "nexaa_container_job.job",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s", namespaceName, containerJobName),
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"mounts",
					"last_updated",
					"status",
					"state",
					"timeouts",
				},
			},
			{
				Config: givenProvider() + givenNamespace(namespaceName, "") +
					givenContainerJobPublic(containerJobName, "nginx:latest", command, entrypoint, schedule),
				Destroy: true,
				PreConfig: func() {
					t.Log("Waiting 10 seconds before destroy...")
					time.Sleep(10 * time.Second)
				},
			},
		},
	})
}

func TestAcc_ContainerJobResource_enabled_toggle(t *testing.T) {
	testAccPreCheck(t)

	namespaceName := generateTestNamespace()
	containerJobName := generateTestContainerJobName()
	entrypoint := generateTestEntrypoint()
	command := generateTestCommands()
	schedule := generateTestSchedule()

	t.Logf("=== CONTAINER JOB ENABLED TOGGLE TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create with enabled=true
			{
				Config: givenProvider() + givenNamespace(namespaceName, "") +
					givenContainerJobPublicWithEnabled(containerJobName, "nginx:latest", command, entrypoint, schedule, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_container_job.job", "id"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "enabled", "true"),
				),
			},
			// 2) Disable the job
			{
				Config: givenProvider() + givenNamespace(namespaceName, "") +
					givenContainerJobPublicWithEnabled(containerJobName, "nginx:latest", command, entrypoint, schedule, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_container_job.job", "enabled", "false"),
				),
				PreConfig: func() {
					t.Log("Waiting 5 seconds before disable...")
					time.Sleep(5 * time.Second)
				},
			},
			// 3) Re-enable the job
			{
				Config: givenProvider() + givenNamespace(namespaceName, "") +
					givenContainerJobPublicWithEnabled(containerJobName, "nginx:latest", command, entrypoint, schedule, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_container_job.job", "enabled", "true"),
				),
				PreConfig: func() {
					t.Log("Waiting 5 seconds before re-enable...")
					time.Sleep(5 * time.Second)
				},
			},
			{
				Config:  givenProvider() + givenNamespace(namespaceName, "") + givenContainerJobPublicWithEnabled(containerJobName, "nginx:latest", command, entrypoint, schedule, true),
				Destroy: true,
				PreConfig: func() {
					t.Log("Waiting 10 seconds before destroy...")
					time.Sleep(10 * time.Second)
				},
			},
		},
	})
}

func TestAcc_ContainerJobResource_basic(t *testing.T) {
	testAccPreCheck(t)

	namespaceName := generateTestNamespace()
	containerJobName := generateTestContainerJobName()
	image := generateTestImage()
	registryName := generateTestRegistryName()
	registryUsername := generateTestUsername()
	registryPassword := generateTestPassword()
	entrypoint := `["/bin/sh"]`
	command := `["-c", "echo hello"]`
	schedule := generateTestSchedule()

	t.Logf("=== CONTAINER JOB TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					t.Log("Waiting 5 seconds before update...")
					time.Sleep(5 * time.Second)
				},
				Config: containerJobConfig(namespaceName, registryName, registryUsername, registryPassword, containerJobName, image, entrypoint, command, schedule),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_container_job.job", "id"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "image", "nginx:latest"),
					resource.TestCheckNoResourceAttr("nexaa_container_job.job", "registry"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "enabled", "false"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "resources", "CPU_250_RAM_500"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "mounts.#", "0"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "environment_variables.#", "0"),
				),
			},
			{
				RefreshState: true,
			},

			// 2) ImportState
			{
				ResourceName:      "nexaa_container_job.job",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s", namespaceName, containerJobName),
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"registry",
					"mounts",
					"last_updated",
					"status",
					"state",
					"timeouts",
				},
			},

			// Refresh state so status reflects the actual running state before the update.
			{
				RefreshState: true,
			},

			// 3) Update — also exercises setting a private registry
			{
				PreConfig: func() {
					t.Log("Waiting 5 seconds before update...")
					time.Sleep(5 * time.Second)
				},
				Config: containerJobUpdateConfig(namespaceName, registryName, registryUsername, registryPassword, containerJobName, "nginx:alpine", `["/bin/sh"]`, `["-c", "echo update"]`, "* * 1 * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_container_job.job", "image", "nginx:alpine"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "registry", registryName),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "resources", "CPU_250_RAM_500"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "mounts.#", "0"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "environment_variables.#", "0"),
				),
			},

			{

				Config:  containerJobUpdateConfig(namespaceName, registryName, registryUsername, registryPassword, containerJobName, "nginx:alpine", `["/bin/sh"]`, `["-c", "echo update"]`, "* * 1 * *"),
				Destroy: true,
				PreConfig: func() {
					t.Log("Waiting 10 seconds before destroy...")
					time.Sleep(10 * time.Second)
				},
			},
		},
	})
}
