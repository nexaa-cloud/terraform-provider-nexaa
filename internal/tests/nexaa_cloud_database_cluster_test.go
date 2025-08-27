// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func cloudDatabaseClusterConfig(namespaceName string, clusterName string, dbType string, version string, cpu string, memory string, storage string, replicas string) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		givenCloudDatabaseCluster(clusterName, dbType, version, cpu, memory, storage, replicas)

}

func TestAccCloudDatabaseClusterResource(t *testing.T) {
	testAccPreCheck(t)

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
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster-database", "cluster.name", clusterName),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster-database", "cluster.namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster-database", "spec.type", "PostgreSQL"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster-database", "spec.version", "16.4"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster-database", "plan.cpu", "1"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster-database", "plan.memory", "2"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster-database", "plan.storage", "10"),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster-database", "plan.replicas", "1"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster.cluster-database", "id"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster.cluster-database", "last_updated"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "nexaa_cloud_database_cluster.cluster-database",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           fmt.Sprintf("%s/%s", namespaceName, clusterName),
				ImportStateVerifyIgnore: []string{"last_updated", "state"},
			},
			// Update and Read testing
			{
				Config: cloudDatabaseClusterConfig(namespaceName, clusterName, "PostgreSQL", "16.4", "1", "2.0", "10", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster-database", "cluster.name", clusterName),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster.cluster-database", "cluster.namespace", namespaceName),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
