// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"

	"github.com/nexaa-cloud/nexaa-cli/api"
)

func translatePlanToUserCreateInput(ctx context.Context, plan cloudDatabaseClusterUserResource) api.CloudDatabaseClusterUserCreateInput {
	permissions := []api.DatabaseUserPermissionInput{}

	type databasePermission struct {
		DatabaseName string `tfsdk:"database_name"`
		Permission   string `tfsdk:"permission"`
		State        string `tfsdk:"state"`
	}

	var databasePermissions []databasePermission
	plan.Permissions.ElementsAs(ctx, &databasePermissions, false)
	for _, permission := range databasePermissions {
		var role = api.DatabasePermissionReadWrite
		if permission.Permission == "read_only" {
			role = api.DatabasePermissionReadOnly
		}

		var state = api.StatePresent
		if permission.State == "absent" {
			state = api.StateAbsent
		}

		permissions = append(permissions, api.DatabaseUserPermissionInput{
			DatabaseName: permission.DatabaseName,
			Permission:   role,
			State:        state,
		})
	}

	userInput := api.DatabaseUserInput{
		Name:        plan.Name.ValueString(),
		Password:    plan.Password.ValueStringPointer(),
		Permissions: permissions,
		State:       api.StatePresent,
	}
	return api.CloudDatabaseClusterUserCreateInput{
		Cluster: api.CloudDatabaseClusterResourceInput{
			Name:      plan.Cluster.Name.ValueString(),
			Namespace: plan.Cluster.Namespace.ValueString(),
		},
		User: userInput,
	}
}

func translatePlanToUserModifyInput(ctx context.Context, plan cloudDatabaseClusterUserResource, state cloudDatabaseClusterUserResource) api.CloudDatabaseClusterUserModifyInput {
	permissions := []api.DatabaseUserPermissionInput{}

	type databasePermission struct {
		DatabaseName string `tfsdk:"database_name"`
		Permission   string `tfsdk:"permission"`
		State        string `tfsdk:"state"`
	}

	var planPermissions []databasePermission
	plan.Permissions.ElementsAs(ctx, &planPermissions, false)

	planPermissionKeys := make(map[string]bool)
	for _, permission := range planPermissions {
		var role = api.DatabasePermissionReadWrite
		if permission.Permission == "read_only" {
			role = api.DatabasePermissionReadOnly
		}

		var permState = api.StatePresent
		if permission.State == "absent" {
			permState = api.StateAbsent
		}

		planPermissionKeys[permission.DatabaseName] = true
		permissions = append(permissions, api.DatabaseUserPermissionInput{
			DatabaseName: permission.DatabaseName,
			Permission:   role,
			State:        permState,
		})
	}

	// Permissions that existed in state but are gone from the plan must be marked absent
	var statePermissions []databasePermission
	state.Permissions.ElementsAs(ctx, &statePermissions, false)
	for _, permission := range statePermissions {
		if !planPermissionKeys[permission.DatabaseName] {
			var role = api.DatabasePermissionReadWrite
			if permission.Permission == "read_only" {
				role = api.DatabasePermissionReadOnly
			}
			permissions = append(permissions, api.DatabaseUserPermissionInput{
				DatabaseName: permission.DatabaseName,
				Permission:   role,
				State:        api.StateAbsent,
			})
		}
	}

	userInput := api.DatabaseUserInput{
		Name:        plan.Name.ValueString(),
		Password:    plan.Password.ValueStringPointer(),
		Permissions: permissions,
		State:       api.StatePresent,
	}
	return api.CloudDatabaseClusterUserModifyInput{
		Cluster: &api.CloudDatabaseClusterResourceInput{
			Name:      plan.Cluster.Name.ValueString(),
			Namespace: plan.Cluster.Namespace.ValueString(),
		},
		User: &userInput,
	}
}
