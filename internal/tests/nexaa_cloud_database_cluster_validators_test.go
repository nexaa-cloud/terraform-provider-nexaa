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

// --- Helpers ---

func clusterWithForbiddenName(name string) string {
	return fmt.Sprintf(`
data "nexaa_cloud_database_cluster_plans" "plan" {
  cpu      = "1"
  memory   = "2.0"
  storage  = "10"
  replicas = "1"
}

resource "nexaa_cloud_database_cluster" "cluster-database" {
  depends_on = [nexaa_namespace.ns]
  cluster = {
    name      = %q
    namespace = nexaa_namespace.ns.name
  }
  spec = {
    type    = "PostgreSQL"
    version = "18.1"
  }
  plan = data.nexaa_cloud_database_cluster_plans.plan.id
  external_connection = {
    ports = {
      allowlist = ["192.168.1.1"]
    }
  }
}
`, name)
}

func userValidationConfig(userName string, password string) string {
	return fmt.Sprintf(`
resource "nexaa_cloud_database_cluster_user" "test_user" {
  depends_on = [
    nexaa_cloud_database_cluster.cluster-database,
    nexaa_cloud_database_cluster_database.db1,
  ]
  cluster  = nexaa_cloud_database_cluster.cluster-database.cluster
  name     = %q
  password = %q
  permissions = [
    {
      database_name = nexaa_cloud_database_cluster_database.db1.name
      permission    = "read_write"
      state         = "present"
    }
  ]
}
`, userName, password)
}

func databaseWithInvalidName() string {
	return `
resource "nexaa_cloud_database_cluster_database" "db_invalid" {
  depends_on  = [nexaa_cloud_database_cluster.cluster-database]
  name        = "1invalid"
  cluster     = nexaa_cloud_database_cluster.cluster-database.cluster
}
`
}

// --- Cluster forbidden name tests ---

func TestAcc_CloudDatabaseClusterResource_ForbiddenNamePg(t *testing.T) {
	testAccPreCheck(t)
	namespaceName := generateTestNamespace()
	t.Logf("=== CLOUD DATABASE CLUSTER FORBIDDEN NAME 'pg' TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: givenProvider() + givenNamespace(namespaceName, ""),
			},
			{
				Config:      givenProvider() + givenNamespace(namespaceName, "") + clusterWithForbiddenName("pg"),
				ExpectError: regexp.MustCompile(`(?i)forbidden`),
			},
			{
				Config:  givenProvider() + givenNamespace(namespaceName, ""),
				Destroy: true,
			},
		},
	})
}

func TestAcc_CloudDatabaseClusterResource_ForbiddenNamePgPrefix(t *testing.T) {
	testAccPreCheck(t)
	namespaceName := generateTestNamespace()
	t.Logf("=== CLOUD DATABASE CLUSTER FORBIDDEN NAME 'pg-' TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: givenProvider() + givenNamespace(namespaceName, ""),
			},
			{
				Config:      givenProvider() + givenNamespace(namespaceName, "") + clusterWithForbiddenName("pg-mydb"),
				ExpectError: regexp.MustCompile(`(?i)forbidden|cannot start|pg`),
			},
			{
				Config:  givenProvider() + givenNamespace(namespaceName, ""),
				Destroy: true,
			},
		},
	})
}

// --- User and database field validation tests ---
// These are expensive: they require a running cluster. All three scenarios share
// a single cluster creation to minimize test time.

func TestAcc_CloudDatabase_UserAndDatabaseValidation(t *testing.T) {
	testAccPreCheck(t)
	namespaceName := generateTestNamespace()
	clusterName := generateTestClusterName()
	dbName := generateTestCloudDatabaseClusterDatabaseName()

	t.Logf("=== CLOUD DATABASE USER/DB VALIDATION TEST USING NAMESPACE: %s ===", namespaceName)

	base := givenProvider() +
		givenNamespace(namespaceName, "") +
		givenCloudDatabaseCluster(clusterName, "PostgreSQL", "18.1", "1", "2.0", "10", "1", []string{"192.168.1.1"})
	baseWithDb := base + givenCloudDatabaseClusterDatabase(dbName, "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create namespace, cluster, and a valid database.
			{
				Config: baseWithDb,
			},
			// Reserved username — backend must reject "admin".
			{
				Config:      baseWithDb + userValidationConfig("admin", "TestPassword123"),
				ExpectError: regexp.MustCompile(`(?i)forbidden`),
			},
			// Forbidden password character — backend must reject passwords containing "@".
			{
				Config:      baseWithDb + userValidationConfig("validtestuser", "TestPass@word123"),
				ExpectError: regexp.MustCompile(`(?i)forbidden|character`),
			},
			// Database name starting with digit — backend must reject it.
			{
				Config:      baseWithDb + databaseWithInvalidName(),
				ExpectError: regexp.MustCompile(`(?i)cannot start|number|invalid|alphabetic`),
			},
			// Destroy all resources.
			{
				Config:  baseWithDb,
				Destroy: true,
				PreConfig: func() {
					t.Log("Waiting 10 seconds before destroy...")
					time.Sleep(10 * time.Second)
				},
			},
		},
	})
}
