// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

var (
	_ resource.Resource                = &cloudDatabaseClusterResource{}
	_ resource.ResourceWithImportState = &cloudDatabaseClusterResource{}
)

func NewCloudDatabaseClusterResource() resource.Resource {
	return &cloudDatabaseClusterResource{}
}

type cloudDatabaseClusterResource struct {
	ID          types.String   `tfsdk:"id"`
	Cluster     ClusterRef     `tfsdk:"cluster"`
	Spec        Spec           `tfsdk:"spec"`
	Plan        Plan           `tfsdk:"plan"`
	State       types.String   `tfsdk:"state"`
	LastUpdated types.String   `tfsdk:"last_updated"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

func (r *cloudDatabaseClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_database_cluster"
}

func (r *cloudDatabaseClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Cloud Database Cluster resource representing a managed database cluster on Nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the cloud database cluster",
			},
			"cluster": schema.ObjectAttribute{
				Required:       true,
				Description:    "Cloud database cluster",
				CustomType:     NewClusterRefType(),
				AttributeTypes: ClusterRefAttributes(),
			},
			"spec": schema.ObjectAttribute{
				Required:       true,
				Description:    "Database specification including type and version",
				CustomType:     NewSpecType(),
				AttributeTypes: SpecAttributes(),
			},
			"plan": schema.ObjectAttribute{
				Required:       true,
				Description:    "Plan for the cloud database cluster.",
				CustomType:     NewPlanType(),
				AttributeTypes: PlanAttributes(),
			},
			"state": schema.StringAttribute{
				Description: "Current state of the cloud database cluster",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the cloud database cluster",
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

func (r *cloudDatabaseClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudDatabaseClusterResource
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

	planId, err := getPlanId(client, plan.Plan.Replicas.ValueInt64(), plan.Plan.Cpu.ValueInt64(), plan.Plan.Memory.ValueInt64(), plan.Plan.Storage.ValueInt64())

	if err != nil {
		resp.Diagnostics.AddError(
			"Error finding plan",
			"Could not get plan: "+err.Error(),
		)
		return
	}

	input := api.CloudDatabaseClusterCreateInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
		Spec: api.CloudDatabaseClusterSpecInput{
			Type:    plan.Spec.Type.ValueString(),
			Version: plan.Spec.Version.ValueString(),
		},
		Plan:      planId,
		Databases: []api.DatabaseInput{},
		Users:     []api.DatabaseUserInput{},
	}

	_, err = client.CloudDatabaseClusterCreate(input)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating cluster",
			"Could not create cluster: "+err.Error(),
		)
		return
	}

	err = waitForUnlocked(ctx, cloudDatabaseClusterLocked(), *client, plan.Cluster.Namespace.ValueString(), plan.Cluster.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster", "Could not reach a unlocked state: "+err.Error())
	}

	cluster, err := client.CloudDatabaseClusterGet(api.CloudDatabaseClusterResourceInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading cluster, cluster not found",
			err.Error(),
		)
		return
	}

	plan = translateApiToCloudDatabaseClusterResource(plan, cluster)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudDatabaseClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var plan cloudDatabaseClusterResource
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	input := api.CloudDatabaseClusterResourceInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
	}
	cluster, err := client.CloudDatabaseClusterGet(input)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading cluster, cluster not found",
			err.Error(),
		)
		return
	}

	plan = translateApiToCloudDatabaseClusterResource(plan, cluster)
	plan.Timeouts = timeouts.Value{
		Object: types.ObjectValueMust(
			map[string]attr.Type{
				"create": types.StringType,
				"delete": types.StringType,
			},
			map[string]attr.Value{
				"create": types.StringValue("2m"),
				"delete": types.StringValue("2m"),
			},
		),
	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)

}

// Omitting is not supported for this resource. So we write the current state back unchanged.
func (r *cloudDatabaseClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No in-place updates supported; preserve current plan.
	var plan cloudDatabaseClusterResource

	// Read current plan and write it back unchanged
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Timeouts = timeouts.Value{
		Object: types.ObjectValueMust(
			map[string]attr.Type{
				"create": types.StringType,
				"delete": types.StringType,
			},
			map[string]attr.Value{
				"create": types.StringValue("2m"),
				"delete": types.StringValue("2m"),
			},
		),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cloudDatabaseClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan cloudDatabaseClusterResource
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
		resp.Diagnostics.AddError("Error deleting cluster", "Could not reach a unlocked state: "+err.Error())
		return
	}

	input := api.CloudDatabaseClusterResourceInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
	}
	_, err = client.CloudDatabaseClusterDelete(input)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting cluster",
			"Could not delete cluster: "+err.Error(),
		)
		return
	}

}

func (r *cloudDatabaseClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format \"<namespace>/<cluster_name>\", got: "+req.ID,
		)
		return
	}
	namespace := parts[0]
	clusterName := parts[1]

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

	var plan cloudDatabaseClusterResource
	plan = translateApiToCloudDatabaseClusterResource(plan, cluster)
	plan.Timeouts = timeouts.Value{
		Object: types.ObjectValueMust(
			map[string]attr.Type{
				"create": types.StringType,
				"delete": types.StringType,
			},
			map[string]attr.Value{
				"create": types.StringValue("2m"),
				"delete": types.StringValue("2m"),
			},
		),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
