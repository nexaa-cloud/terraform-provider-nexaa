// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/options"
	"github.com/joho/godotenv"
)

// testAccPreCheck loads .env file and checks required environment variables
func testAccPreCheck(t *testing.T) {
	// Try to load .env file from project root (go up two levels from internal/tests)
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Logf("Could not load .env file: %v", err)
	} else {
		t.Logf("Loaded .env file successfully")
	}
	
	// Debug: print what we got
	t.Logf("NEXAA_USERNAME = %s", os.Getenv("NEXAA_USERNAME"))
	t.Logf("NEXAA_PASSWORD = %s", os.Getenv("NEXAA_PASSWORD"))
	
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set for acceptance tests")
	}
}

// generateRandomString generates a random lowercase string of given length.
func generateRandomString(length int) string {
	word := faker.Word(options.WithRandomStringLength(uint(length)))
	return word
}

// generateResourceName generates a random resource name with prefix.
func generateResourceName(prefix string) string {
	resourceName := faker.Word()

	return fmt.Sprintf("%s-%s", prefix, resourceName)
}

func generateTestImage() string {
	return "nginx:latest"
}

// generateTestNamespace generates a random namespace name for tests.
func generateTestNamespace() string {
	return generateResourceName("tf-test-ns")
}

// generateTestVolumeName generates a random volume name for tests.
func generateTestVolumeName() string {
	return fmt.Sprintf("vol-%s", generateRandomString(6))
}

// generateTestContainerName generates a random container name for tests.
func generateTestContainerName() string {
	return generateResourceName("tf-container")
}

func generateTestContainerJobName() string {
	return generateResourceName("tf-container-job")
}

func generateTestEntrypoint() string {
	return `[
		"/bin/bash",
	]`
}

func generateTestCommands() string {
	return `[
		"echo",
		"hello",	
	]`
}

func generateTestSchedule() string {
	return "30 * * * 3"
}

// generateTestRegistryName generates a random registry name for tests.
func generateTestRegistryName() string {
	return generateResourceName("tf-reg")
}

// generateTestUsername generates a random username for registry tests.
func generateTestUsername() string {
	return fmt.Sprintf("testuser%s", generateRandomString(6))
}

// generateTestPassword generates a random password for registry tests.
func generateTestPassword() string {
	return fmt.Sprintf("testpass%s", generateRandomString(8))
}

// generateTestDescription generates a random description.
func generateTestDescription() string {
	descriptions := []string{
		"Test namespace for Terraform provider",
		"Automated test environment",
		"CI/CD test namespace",
		"Development test space",
		"Integration test environment",
	}
	return descriptions[rand.IntN(len(descriptions))]
}

// generateTestEnvVar generates a random environment variable name.
func generateTestEnvVar() string {
	vars := []string{"TEST_VAR", "APP_ENV", "CONFIG_VAL", "RUNTIME_MODE", "SERVICE_NAME"}
	return vars[rand.IntN(len(vars))]
}

// generateTestEnvValue generates a random environment variable value.
func generateTestEnvValue() string {
	values := []string{"production", "staging", "development", "test", "terraform"}
	return values[rand.IntN(len(values))]
}

// generateTestPath generates a random path for health checks.
func generateTestPath() string {
	paths := []string{"/"}
	return paths[rand.IntN(len(paths))]
}

// generateRandomSize generates a random size for volumes (between 1-10 GB).
func generateRandomSize() int {
	return rand.IntN(10) + 1
}

// generateRandomPort generates a random port number (between 8000-9000).
func generateRandomPort() int {
	return rand.IntN(1000) + 8000
}

// generateTestClusterName generates a random database cluster name for tests.
func generateTestClusterName() string {
	return generateResourceName("tf-cluster")
}

func givenProvider() string {
	return fmt.Sprintf(
		`provider "nexaa" {
				username = %q
				password = %q
			}
			`,
		os.Getenv("NEXAA_USERNAME"),
		os.Getenv("NEXAA_PASSWORD"),
	)
}

func givenNamespace(name string, description string) string {
	if name == "" {
		name = generateTestNamespace()
	}

	return fmt.Sprintf(
		`
	resource "nexaa_namespace" "ns" {
  		name = %q
		description = %q
	}
	`,
		name,
		description,
	)
}

func givenRegistry(name string, username string, password string) string {
	if name == "" {
		name = generateTestRegistryName()
	}

	return fmt.Sprintf(
		`
resource "nexaa_registry" "registry" {
  depends_on = [nexaa_namespace.ns]
  namespace = nexaa_namespace.ns.name
  name      = %q
  source    = "registry.gitlab.com"
  username  = %q
  password  = %q
  verify    = false
}
`,
		name,
		username,
		password,
	)
}

func givenVolume(name string, size int) string {
	if name == "" {
		name = generateTestVolumeName()
	}

	return fmt.Sprintf(`
        resource "nexaa_volume" "volume" {
        namespace      = nexaa_namespace.ns.name
        name           = %q
        size           = %d
        }`, name, size)
}

func givenContainerJob(name string, image string, command string, entrypoint string, schedule string) string {
	if name == "" {
		name = generateTestContainerJobName()
	}

	return fmt.Sprintf(
		`
resource "nexaa_container_job" "job" {
  namespace = nexaa_namespace.ns.name
  name = %q
  image = %q
  registry = nexaa_registry.registry.name
  command = %s
  entrypoint = %s
  resources = {
    cpu = 0.25
    ram = 0.5
  }
  schedule = %q
}
`, name, image, command, entrypoint, schedule)
}
