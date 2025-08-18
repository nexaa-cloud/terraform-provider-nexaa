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
	return providerConfig + fmt.Sprintf(`
resource "nexaa_namespace" "ns" {
  name = %q
}

resource "nexaa_clouddatabasecluster" "cluster" {
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
  
  databases = [
    {
      name        = "app_db"
      description = "Application database"
      state       = "present"
    }
  ]
  
  users = [
    {
      name     = "app_user"
      password = "secure_password123"
      state    = "present"
      permissions = [
        {
          database   = "app_db"
          permission = "read"
        },
        {
          database   = "app_db"
          permission = "write"
        }
      ]
    }
  ]
}
`, namespaceName, clusterName, dbType, version, cpu, memory, storage, replicas)
}

func TestAccCloudDatabaseClusterResource(t *testing.T) {
	// Skip test if credentials are not provided
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set for acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: cloudDatabaseClusterConfig("test-ns-cdb", "test-cluster", "postgresql", "15", "1", "2.0", "10", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "name", "test-cluster"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "namespace", "test-ns-cdb"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "spec.type", "postgresql"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "spec.version", "15"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "plan.cpu", "1"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "plan.memory", "2"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "plan.storage", "10"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "plan.replicas", "1"),
					resource.TestCheckResourceAttrSet("nexaa_clouddatabasecluster.cluster", "plan.group"),
					resource.TestCheckResourceAttrSet("nexaa_clouddatabasecluster.cluster", "plan.id"),
					resource.TestCheckResourceAttrSet("nexaa_clouddatabasecluster.cluster", "plan.name"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "databases.#", "1"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "databases.0.name", "app_db"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "users.#", "1"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "users.0.name", "app_user"),
					resource.TestCheckResourceAttrSet("nexaa_clouddatabasecluster.cluster", "id"),
					resource.TestCheckResourceAttrSet("nexaa_clouddatabasecluster.cluster", "last_updated"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "nexaa_clouddatabasecluster.cluster",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "test-ns-cdb/test-cluster",
				ImportStateVerifyIgnore: []string{"users.0.password"}, // Password is not returned by API
			},
			// Update and Read testing
			{
				Config: cloudDatabaseClusterConfig("test-ns-cdb", "test-cluster", "postgresql", "15", "1", "2.0", "10", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "name", "test-cluster"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "databases.#", "1"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster", "users.#", "1"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func cloudDatabaseClusterConfigMinimal(namespaceName, clusterName string) string {
	return providerConfig + fmt.Sprintf(`
resource "nexaa_namespace" "ns" {
  name = %q
}

resource "nexaa_clouddatabasecluster" "cluster_minimal" {
  depends_on = [nexaa_namespace.ns]
  name      = %q
  namespace = nexaa_namespace.ns.name
  
  spec = {
    type    = "postgresql"
    version = "15"
  }
  
  plan = {
    cpu     = 1
    memory  = 2.0
    storage = 10
    replicas = 1
  }
}
`, namespaceName, clusterName)
}

func TestAccCloudDatabaseClusterResource_Minimal(t *testing.T) {
	// Skip test if credentials are not provided
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set for acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with minimal configuration
			{
				Config: cloudDatabaseClusterConfigMinimal("test-ns-cdb-min", "test-cluster-minimal"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster_minimal", "name", "test-cluster-minimal"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster_minimal", "namespace", "test-ns-cdb-min"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster_minimal", "spec.type", "postgresql"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster_minimal", "spec.version", "15"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster_minimal", "plan.cpu", "1"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster_minimal", "plan.memory", "2"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster_minimal", "plan.storage", "10"),
					resource.TestCheckResourceAttr("nexaa_clouddatabasecluster.cluster_minimal", "plan.replicas", "1"),
					resource.TestCheckResourceAttrSet("nexaa_clouddatabasecluster.cluster_minimal", "plan.group"),
					resource.TestCheckResourceAttrSet("nexaa_clouddatabasecluster.cluster_minimal", "plan.id"),
					resource.TestCheckResourceAttrSet("nexaa_clouddatabasecluster.cluster_minimal", "id"),
					resource.TestCheckResourceAttrSet("nexaa_clouddatabasecluster.cluster_minimal", "last_updated"),
				),
			},
		},
	})
}
