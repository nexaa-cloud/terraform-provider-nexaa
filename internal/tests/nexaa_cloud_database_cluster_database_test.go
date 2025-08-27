// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func cloudDatabaseClusterDatabaseConfig(namespaceName string, clusterName string, dbType string, version string, cpu string, memory string, storage string, replicas string, databaseName string, databaseDescription string) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		givenCloudDatabaseCluster(clusterName, dbType, version, cpu, memory, storage, replicas) +
		givenCloudDatabaseClusterDatabase(databaseName, databaseDescription, databaseName, namespaceName)
}

func TestAccCloudDatabaseClusterDatabaseResource(t *testing.T) {
	testAccPreCheck(t)

	// Generate random test data
	namespaceName := generateTestNamespace()
	clusterName := generateTestClusterName()
	databaseName := generateTestCloudDatabaseClusterDatabaseName()

	t.Logf("=== CLOUD DATABASE CLUSTER DATABASES TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: cloudDatabaseClusterDatabaseConfig(namespaceName, clusterName, "PostgreSQL", "16.4", "1", "2048", "60", "1", databaseName, ""),
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
			// Delete testing automatically occurs in TestCase
		},
	})
}
