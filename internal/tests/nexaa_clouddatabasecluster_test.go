// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func cloudDatabaseClusterConfig(namespaceName, clusterName, dbType, version, cpu, memory, storage, replicas string) string {
	return givenProvider() + givenNamespace(namespaceName, "") + fmt.Sprintf(`
resource "nexaa_cloud_database_cluster" "cluster" {
  depends_on = [nexaa_namespace.ns]
  name      = %q
  namespace = nexaa_namespace.ns.name
  
  spec = {
    type    = %q
    version = %q
  }
  
  plan = {
    cpu     = %s
    memory  = %s
    storage = %s
    replicas = %s
  }
}
`, clusterName, dbType, version, cpu, memory, storage, replicas)
}

func TestAccCloudDatabaseClusterResource(t *testing.T) {
	// Skip test if credentials are not provided
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set for acceptance tests")
	}

	// Generate random test data
	namespaceName := generateTestNamespace()
	clusterName := generateTestClusterName()

	t.Logf("=== CLOUD DATABASE CLUSTER TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: cloudDatabaseClusterConfig(namespaceName, clusterName, "PostgreSQL", "16.4", "1", "2.0", "10", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster", "name", clusterName),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster", "spec.type", "PostgreSQL"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster", "spec.version", "16.4"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster", "plan.cpu", "1"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster", "plan.memory", "2"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster", "plan.storage", "10"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster", "plan.replicas", "1"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster.cluster", "plan.group"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster.cluster", "plan.id"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster.cluster", "plan.name"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster.cluster", "id"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster.cluster", "last_updated"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "nexaa_cloud_database_cluster.cluster",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           fmt.Sprintf("%s/%s", namespaceName, clusterName),
				ImportStateVerifyIgnore: []string{"plan.name", "last_updated", "state"},
			},
			// Update and Read testing
			{
				Config: cloudDatabaseClusterConfig(namespaceName, clusterName, "PostgreSQL", "16.4", "1", "2.0", "10", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster", "name", clusterName),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster", "namespace", namespaceName),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func cloudDatabaseClusterConfigMinimal(namespaceName, clusterName string) string {
	return givenProvider() + givenNamespace(namespaceName, "") + fmt.Sprintf(`
resource "nexaa_cloud_database_cluster" "cluster_minimal" {
  depends_on = [nexaa_namespace.ns]
  name      = %q
  namespace = nexaa_namespace.ns.name
  
  spec = {
    type    = "PostgreSQL"
    version = "16.4"
  }
  
  plan = {
    cpu     = 1
    memory  = 2.0
    storage = 10
    replicas = 1
  }
}
`, clusterName)
}

func TestAccCloudDatabaseClusterResource_Minimal(t *testing.T) {
	// Skip test if credentials are not provided
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set for acceptance tests")
	}

	// Generate random test data
	namespaceName := generateTestNamespace()
	clusterName := generateTestClusterName()

	t.Logf("=== CLOUD DATABASE CLUSTER MINIMAL TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with minimal configuration
			{
				Config: cloudDatabaseClusterConfigMinimal(namespaceName, clusterName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster_minimal", "name", clusterName),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster_minimal", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster_minimal", "spec.type", "PostgreSQL"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster_minimal", "spec.version", "16.4"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster_minimal", "plan.cpu", "1"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster_minimal", "plan.memory", "2"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster_minimal", "plan.storage", "10"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster_minimal", "plan.replicas", "1"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster.cluster_minimal", "plan.group"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster.cluster_minimal", "plan.id"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster.cluster_minimal", "id"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster.cluster_minimal", "last_updated"),
				),
			},
		},
	})
}
