// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func cloudDatabaseClusterUserConfig(namespaceName string, clusterName string, dbType string, version string, cpu string, memory string, storage string, replicas string, databaseName string, databaseDescription string, user string, allowlist []string) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		givenCloudDatabaseCluster(clusterName, dbType, version, cpu, memory, storage, replicas, allowlist) +
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

	password := generateTestPassword()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: cloudDatabaseClusterUserConfig(namespaceName, clusterName, "PostgreSQL", "18.1", "1", "2", "10", "1", databaseName, "", user, []string{"192.168.1.1"}),
				ConfigVariables: config.Variables{
					"password": config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster_user.user", "cluster.name", clusterName),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster_user.user", "cluster.namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster_user.user", "name", user),
					resource.TestCheckResourceAttr("nexaa_cloud_database_cluster_user.user", "permissions.#", "1"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster_user.user", "id"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster_user.user", "password"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster_user.user", "last_updated"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "nexaa_cloud_database_cluster_user.user",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           fmt.Sprintf("%s/%s/user/%s", namespaceName, clusterName, user),
				ImportStateVerifyIgnore: []string{"last_updated", "password"},
				ConfigVariables: config.Variables{
					"password": config.StringVariable(password),
				},
			},
			// Update and Read testing — re-apply the same config to verify stability
			{
				Config: cloudDatabaseClusterDatabaseConfig(namespaceName, clusterName, "PostgreSQL", "18.1", "1", "2", "10", "1", databaseName, "", []string{"192.168.1.1"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster_user.user", "id"),
					resource.TestCheckResourceAttrSet("nexaa_cloud_database_cluster_user.user", "last_updated"),
				),
				ConfigVariables: config.Variables{
					"password": config.StringVariable(password),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
