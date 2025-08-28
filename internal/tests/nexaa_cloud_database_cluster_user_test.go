// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func cloudDatabaseClusterUserConfig(namespaceName string, clusterName string, dbType string, version string, cpu string, memory string, storage string, replicas string, databaseName string, databaseDescription string, user string) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		givenCloudDatabaseCluster(clusterName, dbType, version, cpu, memory, storage, replicas) +
		givenCloudDatabaseClusterDatabase(databaseName, databaseDescription) +
		givenCloudDatabaseClusterUser(user)
}

func TestAccCloudDatabaseClusterUserResource(t *testing.T) {
	testAccPreCheck(t)

	// Generate random test data
	namespaceName := generateTestNamespace()
	clusterName := generateTestClusterName()
	databaseName := generateTestCloudDatabaseClusterDatabaseName()
	user := generateTestUsername()

	t.Logf("=== CLOUD DATABASE CLUSTER DATABASES TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: cloudDatabaseClusterUserConfig(namespaceName, clusterName, "PostgreSQL", "16.4", "1", "2", "10", "1", databaseName, "", user),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster_user.user", "id"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster_user.user", "last_updated"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "nexaa_cloud_database_cluster_user.user",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           fmt.Sprintf("%s/%s/%s", namespaceName, clusterName, user),
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Update and Read testing
			//{
			//	Config: cloudDatabaseClusterDatabaseConfig(namespaceName, clusterName, "PostgreSQL", "16.4", "1", "2", "10", "1", databaseName, ""),
			//	Check: resource.ComposeAggregateTestCheckFunc(
			//		resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster_database.db1", "id"),
			//		resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster_database.db1", "last_updated"),
			//	),
			//},
			// Delete testing automatically occurs in TestCase
		},
	})
}
