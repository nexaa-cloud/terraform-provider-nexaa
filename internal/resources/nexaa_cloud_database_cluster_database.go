// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/nexaa-cloud/nexaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &cloudDatabaseClusterDatabaseResource{}
	_ resource.ResourceWithImportState = &cloudDatabaseClusterDatabaseResource{}
)

func NewCloudDatabaseClusterDatabaseResource() resource.Resource {
	return &cloudDatabaseClusterDatabaseResource{}
}

type cloudDatabaseClusterDatabaseResource struct {
	ID          types.String   `tfsdk:"id"`
	Cluster     ClusterRef     `tfsdk:"cluster"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	LastUpdated types.String   `tfsdk:"last_updated"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

func (r *cloudDatabaseClusterDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (r *cloudDatabaseClusterDatabaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Database resource representing a database within a cloud database cluster on Nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the database",
			},
			"cluster": schema.ObjectAttribute{
				Required:       true,
				Description:    "Cloud database cluster this database belongs to.",
				CustomType:     NewClusterRefType(),
				AttributeTypes: ClusterRefAttributes(),
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
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the database",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(context.Background(), timeouts.Opts{
				Create: true,
				Delete: true,
			}),
		},
	}
}

func (r *cloudDatabaseClusterDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudDatabaseClusterDatabaseResource
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
		resp.Diagnostics.AddError("Error creating database", "Cloud database cluster is not ready yet: "+err.Error())
	}

	clusterInput := api.CloudDatabaseClusterResourceInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
	}

	var description *string
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc := plan.Description.ValueString()
		description = &desc
	}
	databaseInput := api.DatabaseInput{
		Name:        plan.Name.ValueString(),
		Description: description,
		State:       api.StatePresent,
	}
	input := api.CloudDatabaseClusterDatabaseCreateInput{
		Cluster:  clusterInput,
		Database: databaseInput,
	}

	database, err := client.CloudDatabaseClusterDatabaseCreate(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating database", "Could not create database: "+err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", plan.Cluster.Namespace.ValueString(), plan.Cluster.Name.ValueString(), plan.Name.ValueString()))
	plan.Name = types.StringValue(database.Name)
	plan.Description = types.StringPointerValue(database.Description)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudDatabaseClusterDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var plan cloudDatabaseClusterDatabaseResource
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()

	clusterInput := api.CloudDatabaseClusterResourceInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
	}
	cluster, err := client.CloudDatabaseClusterDatabaseList(clusterInput)
	if err != nil {
		resp.Diagnostics.AddError("Error reading database", "Could not list clusters: "+err.Error())
		return
	}

	if len(cluster.GetDatabases()) == 0 {
		resp.Diagnostics.AddError("Error reading database: no databases found", "")
	}

	// Find the database in the cluster
	var database *api.CloudDatabaseClusterResultDatabasesDatabase
	for _, db := range cluster.GetDatabases() {
		if db.Name == plan.Name.ValueString() {
			database = &api.CloudDatabaseClusterResultDatabasesDatabase{
				Name:        db.Name,
				Description: db.Description,
			}
			break
		}
	}

	if database == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	plan.ID = types.StringValue(generateCloudDatabaseClusterChildId(plan.Cluster.Namespace.ValueString(), plan.Cluster.Name.ValueString(), plan.Name.ValueString()))
	plan.Name = types.StringValue(database.Name)
	plan.Description = types.StringPointerValue(database.Description)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Omitting is not supported for this resource. So we write the current state back unchanged.
func (r *cloudDatabaseClusterDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No in-place updates supported; preserve current state.
	var state cloudDatabaseClusterDatabaseResource

	// Read current state and write it back unchanged
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *cloudDatabaseClusterDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan cloudDatabaseClusterDatabaseResource
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := plan.Timeouts.Delete(ctx, 2*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	client := api.NewClient()
	err := waitForUnlocked(ctx, cloudDatabaseClusterLocked(), *client, plan.Cluster.Namespace.ValueString(), plan.Cluster.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Error creating database", "Cloud database cluster is not ready yet: "+err.Error())
	}

	clusterInput := api.CloudDatabaseClusterResourceInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
	}

	input := api.CloudDatabaseClusterDatabaseResourceInput{
		Cluster: clusterInput,
		Name:    plan.Name.ValueString(),
	}

	_, err = client.CloudDatabaseClusterDatabaseDelete(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting database",
			fmt.Sprintf("Failed to delete database %q: %s", plan.Name.ValueString(), err.Error()),
		)
		return
	}
	fmt.Printf("Deleted database %q\n", plan.Name.ValueString())
}

func (r *cloudDatabaseClusterDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
	clusterResourceInput := api.CloudDatabaseClusterResourceInput{
		Namespace: namespace,
		Name:      clusterName,
	}
	cluster, err := client.CloudDatabaseClusterGet(clusterResourceInput)
	if err != nil {
		resp.Diagnostics.AddError("Error importing database", "Could not list clusters: "+err.Error())
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

	state := cloudDatabaseClusterDatabaseResource{
		ID:          types.StringValue(fmt.Sprintf("%s/%s/%s", namespace, clusterName, database.Name)),
		Name:        types.StringValue(database.Name),
		Description: types.StringPointerValue(database.Description),
		Cluster: ClusterRef{
			Name:      types.StringValue(cluster.Name),
			Namespace: types.StringValue(namespace),
		},
		LastUpdated: types.StringValue(time.Now().Format(time.RFC3339)),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
