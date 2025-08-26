// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nexaa-cloud/nexaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &databaseResource{}
	_ resource.ResourceWithImportState = &databaseResource{}
)

func NewDatabaseResource() resource.Resource {
	return &databaseResource{}
}

type databaseResource struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Cluster     types.String `tfsdk:"cluster"`
	Namespace   types.String `tfsdk:"namespace"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (r *databaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Database resource representing a database within a cloud database cluster on Nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the database",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the database",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Optional description of the database",
			},
			"cluster": schema.StringAttribute{
				Required:    true,
				Description: "Name of the cloud database cluster this database belongs to",
			},
			"namespace": schema.StringAttribute{
				Required:    true,
				Description: "Name of the namespace that the cluster belongs to",
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the database",
				Computed:    true,
			},
		},
	}
}

func (r *databaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan databaseResource
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

	var description *string
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc := plan.Description.ValueString()
		description = &desc
	}

	input.Databases = []api.DatabaseInput{
		{
			Name:        plan.Name.ValueString(),
			Description: description,
			State:       api.StatePresent,
		},
	}

	cluster, err := client.CloudDatabaseClusterModify(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating database", "Could not create database: "+err.Error())
		return
	}

	// Find the created database in the response
	var createdDB *api.CloudDatabaseClusterResultDatabasesDatabase
	for _, db := range cluster.Databases {
		if db.Name == plan.Name.ValueString() {
			createdDB = &db
			break
		}
	}

	if createdDB == nil {
		resp.Diagnostics.AddError("Error creating database", "Database was not found in cluster after creation")
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", plan.Namespace.ValueString(), plan.Cluster.ValueString(), createdDB.Name))
	plan.Name = types.StringValue(createdDB.Name)
	plan.Description = types.StringPointerValue(createdDB.Description)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	clusters, err := client.CloudDatabaseClusterList()
	if err != nil {
		resp.Diagnostics.AddError("Error reading database", "Could not list clusters: "+err.Error())
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

	// Find the database in the cluster
	var database *api.CloudDatabaseClusterResultDatabasesDatabase
	for _, db := range cluster.Databases {
		if db.Name == state.Name.ValueString() {
			database = &db
			break
		}
	}

	if database == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", state.Namespace.ValueString(), state.Cluster.ValueString(), database.Name))
	state.Name = types.StringValue(database.Name)
	state.Description = types.StringPointerValue(database.Description)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *databaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan databaseResource
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

	var description *string
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc := plan.Description.ValueString()
		description = &desc
	}

	input.Databases = []api.DatabaseInput{
		{
			Name:        plan.Name.ValueString(),
			Description: description,
			State:       api.StatePresent,
		},
	}

	cluster, err := client.CloudDatabaseClusterModify(input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating database", "Could not update database: "+err.Error())
		return
	}

	// Find the updated database in the response
	var updatedDB *api.CloudDatabaseClusterResultDatabasesDatabase
	for _, db := range cluster.Databases {
		if db.Name == plan.Name.ValueString() {
			updatedDB = &db
			break
		}
	}

	if updatedDB == nil {
		resp.Diagnostics.AddError("Error updating database", "Database was not found in cluster after update")
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", plan.Namespace.ValueString(), plan.Cluster.ValueString(), updatedDB.Name))
	plan.Name = types.StringValue(updatedDB.Name)
	plan.Description = types.StringPointerValue(updatedDB.Description)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *databaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state databaseResource
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

	input.Databases = []api.DatabaseInput{
		{
			Name:  state.Name.ValueString(),
			State: api.StateAbsent,
		},
	}

	_, err := client.CloudDatabaseClusterModify(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting database",
			fmt.Sprintf("Failed to delete database %q: %s", state.Name.ValueString(), err.Error()),
		)
		return
	}
}

func (r *databaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format \"<namespace>/<cluster_name>/<database_name>\", got: "+req.ID,
		)
		return
	}
	namespace := parts[0]
	clusterName := parts[1]
	databaseName := parts[2]

	client := api.NewClient()
	clusters, err := client.CloudDatabaseClusterList()
	if err != nil {
		resp.Diagnostics.AddError("Error importing database", "Could not list clusters: "+err.Error())
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
			"Error importing database",
			fmt.Sprintf("Unable to find cluster %q in namespace %q", clusterName, namespace),
		)
		return
	}

	// Find the database in the cluster
	var database *api.CloudDatabaseClusterResultDatabasesDatabase
	for _, db := range cluster.Databases {
		if db.Name == databaseName {
			database = &db
			break
		}
	}

	if database == nil {
		resp.Diagnostics.AddError(
			"Error importing database",
			fmt.Sprintf("Unable to find database %q in cluster %q", databaseName, clusterName),
		)
		return
	}

	state := databaseResource{
		ID:          types.StringValue(fmt.Sprintf("%s/%s/%s", namespace, clusterName, database.Name)),
		Name:        types.StringValue(database.Name),
		Description: types.StringPointerValue(database.Description),
		Cluster:     types.StringValue(clusterName),
		Namespace:   types.StringValue(namespace),
		LastUpdated: types.StringValue(time.Now().Format(time.RFC3339)),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
