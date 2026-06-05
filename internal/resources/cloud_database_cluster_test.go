// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
	"github.com/stretchr/testify/assert"
)

// --- unpackCloudDatabaseClusterChildId ---

func Test_UnpackCloudDatabaseClusterChildId_valid(t *testing.T) {
	id, err := unpackCloudDatabaseClusterChildId("my-ns/my-cluster/database/my-db")
	assert.NoError(t, err)
	assert.Equal(t, "my-ns", id.Namespace)
	assert.Equal(t, "my-cluster", id.Cluster)
	assert.Equal(t, "my-db", id.Name)
}

func Test_UnpackCloudDatabaseClusterChildId_three_parts_errors(t *testing.T) {
	_, err := unpackCloudDatabaseClusterChildId("ns/cluster/name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "namespace")
}

func Test_UnpackCloudDatabaseClusterChildId_empty_part_errors(t *testing.T) {
	_, err := unpackCloudDatabaseClusterChildId("ns//database/child")
	assert.Error(t, err)
}

func Test_UnpackCloudDatabaseClusterChildId_empty_string_errors(t *testing.T) {
	_, err := unpackCloudDatabaseClusterChildId("")
	assert.Error(t, err)
}

// --- translatePlanToUserCreateInput ---

func makePermissionSet(permissions []map[string]string) types.Set {
	elemType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"database_name": types.StringType,
			"permission":    types.StringType,
			"state":         types.StringType,
		},
	}
	elems := make([]attr.Value, len(permissions))
	for i, p := range permissions {
		elems[i] = types.ObjectValueMust(elemType.AttrTypes, map[string]attr.Value{
			"database_name": types.StringValue(p["database_name"]),
			"permission":    types.StringValue(p["permission"]),
			"state":         types.StringValue(p["state"]),
		})
	}
	return types.SetValueMust(elemType, elems)
}

func makeUserPlan(name string, permissions []map[string]string) cloudDatabaseClusterUserResource {
	return cloudDatabaseClusterUserResource{
		Name:     types.StringValue(name),
		Password: types.StringValue("secret"),
		Cluster: ClusterRef{
			Namespace: types.StringValue("test-ns"),
			Name:      types.StringValue("test-cluster"),
		},
		Permissions: makePermissionSet(permissions),
	}
}

func Test_TranslatePlanToUserCreateInput_read_write_present(t *testing.T) {
	plan := makeUserPlan("alice", []map[string]string{
		{"database_name": "mydb", "permission": "read_write", "state": "present"},
	})
	input := translatePlanToUserCreateInput(context.Background(), plan)
	assert.Equal(t, "alice", input.User.Name)
	assert.Len(t, input.User.Permissions, 1)
	assert.Equal(t, api.DatabasePermissionReadWrite, input.User.Permissions[0].Permission)
	assert.Equal(t, api.StatePresent, input.User.Permissions[0].State)
}

func Test_TranslatePlanToUserCreateInput_read_only(t *testing.T) {
	plan := makeUserPlan("bob", []map[string]string{
		{"database_name": "mydb", "permission": "read_only", "state": "present"},
	})
	input := translatePlanToUserCreateInput(context.Background(), plan)
	assert.Equal(t, api.DatabasePermissionReadOnly, input.User.Permissions[0].Permission)
}

func Test_TranslatePlanToUserCreateInput_absent_state(t *testing.T) {
	plan := makeUserPlan("carol", []map[string]string{
		{"database_name": "mydb", "permission": "read_write", "state": "absent"},
	})
	input := translatePlanToUserCreateInput(context.Background(), plan)
	assert.Equal(t, api.StateAbsent, input.User.Permissions[0].State)
}

// --- translatePlanToUserModifyInput ---

func Test_TranslatePlanToUserModifyInput_keeps_existing_permission(t *testing.T) {
	plan := makeUserPlan("alice", []map[string]string{
		{"database_name": "mydb", "permission": "read_write", "state": "present"},
	})
	state := makeUserPlan("alice", []map[string]string{
		{"database_name": "mydb", "permission": "read_write", "state": "present"},
	})
	input := translatePlanToUserModifyInput(context.Background(), plan, state)
	perms := input.User.Permissions
	assert.Len(t, perms, 1)
	assert.Equal(t, "mydb", perms[0].DatabaseName)
	assert.Equal(t, api.StatePresent, perms[0].State)
}

func Test_TranslatePlanToUserModifyInput_removed_permission_marked_absent(t *testing.T) {
	plan := makeUserPlan("alice", []map[string]string{})
	state := makeUserPlan("alice", []map[string]string{
		{"database_name": "old-db", "permission": "read_write", "state": "present"},
	})
	input := translatePlanToUserModifyInput(context.Background(), plan, state)
	perms := input.User.Permissions
	assert.Len(t, perms, 1)
	assert.Equal(t, "old-db", perms[0].DatabaseName)
	assert.Equal(t, api.StateAbsent, perms[0].State)
}

func Test_TranslatePlanToUserModifyInput_new_permission_added(t *testing.T) {
	plan := makeUserPlan("alice", []map[string]string{
		{"database_name": "new-db", "permission": "read_only", "state": "present"},
	})
	state := makeUserPlan("alice", []map[string]string{})
	input := translatePlanToUserModifyInput(context.Background(), plan, state)
	perms := input.User.Permissions
	assert.Len(t, perms, 1)
	assert.Equal(t, "new-db", perms[0].DatabaseName)
	assert.Equal(t, api.StatePresent, perms[0].State)
	assert.Equal(t, api.DatabasePermissionReadOnly, perms[0].Permission)
}
