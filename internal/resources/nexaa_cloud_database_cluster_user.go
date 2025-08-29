// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

var (
	_ resource.Resource                = &cloudDatabaseClusterUserResource{}
	_ resource.ResourceWithImportState = &cloudDatabaseClusterUserResource{}
)

func NewDatabaseUserResource() resource.Resource {
	return &cloudDatabaseClusterUserResource{}
}

type cloudDatabaseClusterUserResource struct {
	ID          types.String   `tfsdk:"id"`
	Cluster     ClusterRef     `tfsdk:"cluster"`
	Name        types.String   `tfsdk:"name"`
	Password    types.String   `tfsdk:"password"`
	Permissions types.Set      `tfsdk:"permissions"`
	LastUpdated types.String   `tfsdk:"last_updated"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

func (r *cloudDatabaseClusterUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_database_cluster_user"
}

func (r *cloudDatabaseClusterUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Database User resource representing a database user within a cloud database cluster on Nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the database user",
			},
			"cluster": schema.ObjectAttribute{
				Required:       true,
				Description:    "Cloud database cluster this database belongs to.",
				CustomType:     NewClusterRefType(),
				AttributeTypes: ClusterRefAttributes(),
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the database user",
			},
			"password": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Sensitive:   true,
				Description: "Password for the database user",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"permissions": schema.SetNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"database_name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the database",
						},
						"permission": schema.StringAttribute{
							Required:    true,
							Description: "The permission to be granted to the database user",
							Validators: []validator.String{
								stringvalidator.OneOf("read_only", "read_write"),
							},
						},
						"state": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The state of the permission",
							Default:     stringdefault.StaticString("present"),
							Validators: []validator.String{
								stringvalidator.OneOf("present", "absent"),
							},
						},
					},
				},
				Optional:    true,
				Computed:    true,
				Description: "",
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the database user",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(context.Background(), timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func translatePlanToUserInput(ctx context.Context, plan cloudDatabaseClusterUserResource) api.CloudDatabaseClusterModifyInput {
	var permissions []api.DatabaseUserPermissionInput

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

	return api.CloudDatabaseClusterModifyInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
		Users: []api.DatabaseUserInput{
			userInput,
		},
	}

}

func (r *cloudDatabaseClusterUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudDatabaseClusterUserResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 2*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	client := api.NewClient()
	err := waitForUnlocked(ctx, cloudDatabaseClusterLocked(), *client, plan.Cluster.Namespace.ValueString(), plan.Cluster.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster", "Could not reach a unlocked state: "+err.Error())
	}

	input := translatePlanToUserInput(ctx, plan)
	result, err := client.CloudDatabaseClusterModify(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", "Could not create user: "+err.Error())
		return
	}

	plan, err = translateApiToCloudDatabaseClusterUserResource(plan, result, plan.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Error creating user", "Could not create user: "+err.Error())
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudDatabaseClusterUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var plan cloudDatabaseClusterUserResource
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace := plan.Cluster.Namespace.ValueString()
	name := plan.Cluster.Name.ValueString()

	if namespace == "" || name == "" {
		id, err := unpackCloudDatabaseClusterChildId(plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Could not unpack ID", err.Error(),
			)
			return
		}

		namespace = id.Namespace
		name = id.Cluster
	}

	client := api.NewClient()
	clusterInput := api.CloudDatabaseClusterResourceInput{
		Name:      name,
		Namespace: namespace,
	}

	cluster, err := client.CloudDatabaseClusterGet(clusterInput)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading cluster \""+clusterInput.Name+"\" not found in namespace \""+clusterInput.Namespace+"\"",
			err.Error(),
		)
		return
	}

	plan, err = translateApiToCloudDatabaseClusterUserResource(plan, cluster, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", "Could not read user: "+err.Error())
		return
	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudDatabaseClusterUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No in-place updates supported; preserve current plan.
	var plan cloudDatabaseClusterUserResource

	// Read current plan and write it back unchanged
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, 2*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	client := api.NewClient()
	err := waitForUnlocked(ctx, cloudDatabaseClusterLocked(), *client, plan.Cluster.Namespace.ValueString(), plan.Cluster.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster", "Could not reach a unlocked state: "+err.Error())
	}

	input := translatePlanToUserInput(ctx, plan)
	result, err := client.CloudDatabaseClusterModify(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating database", "Could not create database: "+err.Error())
		return
	}

	plan, err = translateApiToCloudDatabaseClusterUserResource(plan, result, plan.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Error creating user", "Could not create user: "+err.Error())
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudDatabaseClusterUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan cloudDatabaseClusterUserResource
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Delete(ctx, 2*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	client := api.NewClient()
	err := waitForUnlocked(ctx, cloudDatabaseClusterLocked(), *client, plan.Cluster.Namespace.ValueString(), plan.Cluster.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster", "Could not reach a unlocked state: "+err.Error())
	}

	userInput := api.DatabaseUserInput{
		Name:        plan.Name.ValueString(),
		Password:    plan.Password.ValueStringPointer(),
		Permissions: []api.DatabaseUserPermissionInput{},
		State:       api.StateAbsent,
	}

	input := api.CloudDatabaseClusterModifyInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
		Users: []api.DatabaseUserInput{
			userInput,
		},
	}

	_, err = client.CloudDatabaseClusterModify(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting database",
			fmt.Sprintf("Failed to delete user %q: %s", plan.Name.ValueString(), err.Error()),
		)
		return
	}
	fmt.Printf("Deleted user %q\n", plan.Name.ValueString())
}

func (r *cloudDatabaseClusterUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := unpackCloudDatabaseClusterChildId(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			err.Error(),
		)
		return
	}

	client := api.NewClient()
	clusterResourceInput := api.CloudDatabaseClusterResourceInput{
		Namespace: id.Namespace,
		Name:      id.Cluster,
	}
	cluster, err := client.CloudDatabaseClusterGet(clusterResourceInput)
	if err != nil {
		resp.Diagnostics.AddError("Error importing database", "Could not list clusters: "+err.Error())
		return
	}

	var plan cloudDatabaseClusterUserResource
	plan, err = translateApiToCloudDatabaseClusterUserResource(plan, cluster, id.Name)
	if err != nil {
		resp.Diagnostics.AddError("Error importing user", err.Error())
		return
	}
	plan.Timeouts = timeouts.Value{
		Object: types.ObjectValueMust(
			map[string]attr.Type{
				"create": types.StringType,
				"update": types.StringType,
				"delete": types.StringType,
			},
			map[string]attr.Value{
				"create": types.StringValue("2m"),
				"update": types.StringValue("2m"),
				"delete": types.StringValue("2m"),
			},
		),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}
