// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nexaa-cloud/nexaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &databaseUserResource{}
	_ resource.ResourceWithImportState = &databaseUserResource{}
)

func NewDatabaseUserResource() resource.Resource {
	return &databaseUserResource{}
}

type databaseUserResource struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Password    types.String `tfsdk:"password"`
	Cluster     types.String `tfsdk:"cluster"`
	Namespace   types.String `tfsdk:"namespace"`
	Permissions types.List   `tfsdk:"permissions"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

type databaseUserPermissionResource struct {
	Database   types.String `tfsdk:"database"`
	Permission types.String `tfsdk:"permission"`
}

func (r *databaseUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_user"
}

func (r *databaseUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Database User resource representing a database user within a cloud database cluster on Nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the database user",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the database user",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Password for the database user",
			},
			"cluster": schema.StringAttribute{
				Required:    true,
				Description: "Name of the cloud database cluster this user belongs to",
			},
			"namespace": schema.StringAttribute{
				Required:    true,
				Description: "Name of the namespace that the cluster belongs to",
			},
			"permissions": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"database": schema.StringAttribute{
							Required:    true,
							Description: "Database name for the permission",
						},
						"permission": schema.StringAttribute{
							Required:    true,
							Description: "Permission type (e.g., read, write, admin)",
						},
					},
				},
				Optional:    true,
				Computed:    true,
				Description: "List of database permissions for the user",
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the database user",
				Computed:    true,
			},
		},
	}
}

func (r *databaseUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan databaseUserResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	input := api.CloudDatabaseClusterModifyInput{
		Name:      plan.Cluster.ValueString(),
		Namespace: plan.Namespace.ValueString(),
	}

	var password *string
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		pwd := plan.Password.ValueString()
		password = &pwd
	}

	userInput := api.DatabaseUserInput{
		Name:     plan.Name.ValueString(),
		Password: password,
		State:    api.StatePresent,
	}

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var permissions []databaseUserPermissionResource
		diags = plan.Permissions.ElementsAs(ctx, &permissions, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, perm := range permissions {
			userInput.Permissions = append(userInput.Permissions, api.DatabaseUserPermissionInput{
				DatabaseName: perm.Database.ValueString(),
				Permission:   api.DatabasePermission(perm.Permission.ValueString()),
				State:        api.StatePresent,
			})
		}
	}

	input.Users = []api.DatabaseUserInput{userInput}

	cluster, err := client.CloudDatabaseClusterModify(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating database user", "Could not create database user: "+err.Error())
		return
	}

	// Find the created user in the response
	var createdUser *api.CloudDatabaseClusterResultUsersDatabaseUser
	for _, user := range cluster.Users {
		if user.Name == plan.Name.ValueString() {
			createdUser = &user
			break
		}
	}

	if createdUser == nil {
		resp.Diagnostics.AddError("Error creating database user", "Database user was not found in cluster after creation")
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", plan.Namespace.ValueString(), plan.Cluster.ValueString(), createdUser.Name))
	plan.Name = types.StringValue(createdUser.Name)

	// Build permissions list from API response
	if len(createdUser.Permissions) > 0 {
		permissions := make([]attr.Value, len(createdUser.Permissions))
		for i, perm := range createdUser.Permissions {
			permObj := types.ObjectValueMust(
				map[string]attr.Type{
					"database":   types.StringType,
					"permission": types.StringType,
				},
				map[string]attr.Value{
					"database":   types.StringValue(perm.DatabaseName),
					"permission": types.StringValue(string(perm.Permission)),
				})
			permissions[i] = permObj
		}
		permList, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"database":   types.StringType,
				"permission": types.StringType,
			},
		}, permissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Permissions = permList
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *databaseUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseUserResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	clusters, err := client.CloudDatabaseClusterList()
	if err != nil {
		resp.Diagnostics.AddError("Error reading database user", "Could not list clusters: "+err.Error())
		return
	}

	var cluster *api.CloudDatabaseClusterResult
	for _, c := range clusters {
		if c.Name == state.Cluster.ValueString() && c.Namespace.Name == state.Namespace.ValueString() {
			cluster = &c
			break
		}
	}

	if cluster == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Find the user in the cluster
	var user *api.CloudDatabaseClusterResultUsersDatabaseUser
	for _, u := range cluster.Users {
		if u.Name == state.Name.ValueString() {
			user = &u
			break
		}
	}

	if user == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", state.Namespace.ValueString(), state.Cluster.ValueString(), user.Name))
	state.Name = types.StringValue(user.Name)

	// Build permissions list from API response
	if len(user.Permissions) > 0 {
		permissions := make([]attr.Value, len(user.Permissions))
		for i, perm := range user.Permissions {
			permObj := types.ObjectValueMust(
				map[string]attr.Type{
					"database":   types.StringType,
					"permission": types.StringType,
				},
				map[string]attr.Value{
					"database":   types.StringValue(perm.DatabaseName),
					"permission": types.StringValue(string(perm.Permission)),
				})
			permissions[i] = permObj
		}
		permList, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"database":   types.StringType,
				"permission": types.StringType,
			},
		}, permissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Permissions = permList
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *databaseUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan databaseUserResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	input := api.CloudDatabaseClusterModifyInput{
		Name:      plan.Cluster.ValueString(),
		Namespace: plan.Namespace.ValueString(),
	}

	var password *string
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		pwd := plan.Password.ValueString()
		password = &pwd
	}

	userInput := api.DatabaseUserInput{
		Name:     plan.Name.ValueString(),
		Password: password,
		State:    api.StatePresent,
	}

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var permissions []databaseUserPermissionResource
		diags = plan.Permissions.ElementsAs(ctx, &permissions, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, perm := range permissions {
			userInput.Permissions = append(userInput.Permissions, api.DatabaseUserPermissionInput{
				DatabaseName: perm.Database.ValueString(),
				Permission:   api.DatabasePermission(perm.Permission.ValueString()),
				State:        api.StatePresent,
			})
		}
	}

	input.Users = []api.DatabaseUserInput{userInput}

	cluster, err := client.CloudDatabaseClusterModify(input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating database user", "Could not update database user: "+err.Error())
		return
	}

	// Find the updated user in the response
	var updatedUser *api.CloudDatabaseClusterResultUsersDatabaseUser
	for _, user := range cluster.Users {
		if user.Name == plan.Name.ValueString() {
			updatedUser = &user
			break
		}
	}

	if updatedUser == nil {
		resp.Diagnostics.AddError("Error updating database user", "Database user was not found in cluster after update")
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", plan.Namespace.ValueString(), plan.Cluster.ValueString(), updatedUser.Name))
	plan.Name = types.StringValue(updatedUser.Name)

	// Build permissions list from API response
	if len(updatedUser.Permissions) > 0 {
		permissions := make([]attr.Value, len(updatedUser.Permissions))
		for i, perm := range updatedUser.Permissions {
			permObj := types.ObjectValueMust(
				map[string]attr.Type{
					"database":   types.StringType,
					"permission": types.StringType,
				},
				map[string]attr.Value{
					"database":   types.StringValue(perm.DatabaseName),
					"permission": types.StringValue(string(perm.Permission)),
				})
			permissions[i] = permObj
		}
		permList, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"database":   types.StringType,
				"permission": types.StringType,
			},
		}, permissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Permissions = permList
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *databaseUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state databaseUserResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	input := api.CloudDatabaseClusterModifyInput{
		Name:      state.Cluster.ValueString(),
		Namespace: state.Namespace.ValueString(),
	}

	input.Users = []api.DatabaseUserInput{
		{
			Name:  state.Name.ValueString(),
			State: api.StateAbsent,
		},
	}

	_, err := client.CloudDatabaseClusterModify(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting database user",
			fmt.Sprintf("Failed to delete database user %q: %s", state.Name.ValueString(), err.Error()),
		)
		return
	}
}

func (r *databaseUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format \"<namespace>/<cluster_name>/<user_name>\", got: "+req.ID,
		)
		return
	}
	namespace := parts[0]
	clusterName := parts[1]
	userName := parts[2]

	client := api.NewClient()
	clusters, err := client.CloudDatabaseClusterList()
	if err != nil {
		resp.Diagnostics.AddError("Error importing database user", "Could not list clusters: "+err.Error())
		return
	}

	var cluster *api.CloudDatabaseClusterResult
	for _, c := range clusters {
		if c.Name == clusterName && c.Namespace.Name == namespace {
			cluster = &c
			break
		}
	}

	if cluster == nil {
		resp.Diagnostics.AddError(
			"Error importing database user",
			fmt.Sprintf("Unable to find cluster %q in namespace %q", clusterName, namespace),
		)
		return
	}

	// Find the user in the cluster
	var user *api.CloudDatabaseClusterResultUsersDatabaseUser
	for _, u := range cluster.Users {
		if u.Name == userName {
			user = &u
			break
		}
	}

	if user == nil {
		resp.Diagnostics.AddError(
			"Error importing database user",
			fmt.Sprintf("Unable to find database user %q in cluster %q", userName, clusterName),
		)
		return
	}

	// Build permissions list from API response
	var permList types.List
	if len(user.Permissions) > 0 {
		permissions := make([]attr.Value, len(user.Permissions))
		for i, perm := range user.Permissions {
			permObj := types.ObjectValueMust(
				map[string]attr.Type{
					"database":   types.StringType,
					"permission": types.StringType,
				},
				map[string]attr.Value{
					"database":   types.StringValue(perm.DatabaseName),
					"permission": types.StringValue(string(perm.Permission)),
				})
			permissions[i] = permObj
		}
		list, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"database":   types.StringType,
				"permission": types.StringType,
			},
		}, permissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		permList = list
	} else {
		permList = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"database":   types.StringType,
				"permission": types.StringType,
			},
		})
	}

	state := databaseUserResource{
		ID:          types.StringValue(fmt.Sprintf("%s/%s/%s", namespace, clusterName, user.Name)),
		Name:        types.StringValue(user.Name),
		Password:    types.StringNull(), // Password is not returned by the API
		Cluster:     types.StringValue(clusterName),
		Namespace:   types.StringValue(namespace),
		Permissions: permList,
		LastUpdated: types.StringValue(time.Now().Format(time.RFC3339)),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
