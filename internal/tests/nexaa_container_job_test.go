// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func containerJobConfig(namespaceName string, registryName string, registryUsername string, registryPassword string, containerJobName string, image string, entrypoint string, command string, schedule string) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		givenRegistry(registryName, registryUsername, registryPassword) +
		givenContainerJob(containerJobName, image, command, entrypoint, schedule)
}

func containerJobUpdateConfig(namespaceName string, registryName string, registryUsername string, registryPassword string, containerJobName string, image string, entrypoint string, command string, schedule string) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		givenRegistry(registryName, registryUsername, registryPassword) +
		givenContainerJob(containerJobName, image, command, entrypoint, schedule)
}

func TestAcc_ContainerJobResource_basic(t *testing.T) {
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set")
	}

	namespaceName := generateTestNamespace()
	containerJobName := generateTestContainerJobName()
	image := generateTestImage()
	registryName := generateTestRegistryName()
	registryUsername := generateTestUsername()
	registryPassword := generateTestPassword()
	entrypoint := generateTestEntrypoint()
	command := generateTestCommands()
	schedule := generateTestSchedule()

	t.Logf("=== CONTAINER JOB TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: containerJobConfig(namespaceName, registryName, registryUsername, registryPassword, containerJobName, image, entrypoint, command, schedule),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_container_job.job", "id"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "image", "nginx:latest"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "registry", registryName),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "resources.cpu", "0.25"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "resources.ram", "0.5"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "mounts.#", "0"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "environment_variables.#", "0"),
				),
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
				},
			},

			{
				Config: containerJobUpdateConfig(namespaceName, registryName, registryUsername, registryPassword, containerJobName, "nginx:alpine", `["/bin/sh", "-c"]`, `["ping", "google.com"]`, "* * 1 * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_container_job.job", "image", "nginx:alpine"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "registry", registryName),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "resources.cpu", "0.25"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "resources.ram", "0.5"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "mounts.#", "0"),
					resource.TestCheckResourceAttr("nexaa_container_job.job", "environment_variables.#", "0"),
				),
			},
		},
	})
}
