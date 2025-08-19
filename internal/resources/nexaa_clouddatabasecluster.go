// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nexaa-cloud/nexaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &cloudDatabaseClusterResource{}
	_ resource.ResourceWithImportState = &cloudDatabaseClusterResource{}
)

func NewCloudDatabaseClusterResource() resource.Resource {
	return &cloudDatabaseClusterResource{}
}

type cloudDatabaseClusterResource struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Namespace   types.String `tfsdk:"namespace"`
	Spec        types.Object `tfsdk:"spec"`
	Plan        types.Object `tfsdk:"plan"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

type specResource struct {
	Type    types.String `tfsdk:"type"`
	Version types.String `tfsdk:"version"`
}

type planResource struct {
	// User-specified inputs for plan selection
	Cpu      types.Int64   `tfsdk:"cpu"`
	Memory   types.Float64 `tfsdk:"memory"`
	Storage  types.Int64   `tfsdk:"storage"`
	Replicas types.Int64   `tfsdk:"replicas"`

	// Computed fields
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Group types.String `tfsdk:"group"`
	Price types.Object `tfsdk:"price"`
}

// replicasToGroup maps replica count to API group names
func replicasToGroup(replicas int64) string {
	switch replicas {
	case 1:
		return "Single (1 node)"
	case 2:
		return "Redundant (2 nodes)"
	case 3:
		return "Highly available (3 nodes)"
	default:
		return "Single (1 node)" // fallback
	}
}

// findMatchingPlan finds a plan that matches the user's specifications
func findMatchingPlan(client *api.Client, specs planResource) (*api.CloudDatabaseClusterPlan, error) {
	plans, err := client.CloudDatabaseClusterListPlans()
	if err != nil {
		return nil, fmt.Errorf("failed to list available plans: %w", err)
	}

	requiredCpu := int(specs.Cpu.ValueInt64())
	requiredMemory := specs.Memory.ValueFloat64()
	requiredStorage := int(specs.Storage.ValueInt64())
	requiredReplicas := specs.Replicas.ValueInt64()
	requiredGroup := replicasToGroup(requiredReplicas)

	// Find exact matches first
	for _, plan := range plans {
		if plan.Cpu == requiredCpu &&
			plan.Memory == requiredMemory &&
			plan.Storage == requiredStorage &&
			plan.Group == requiredGroup {
			// TODO: Add replica matching once available in API
			return &plan, nil
		}
	}

	// No exact match found - create a helpful error message with available plans
	var availablePlans []string
	for _, plan := range plans {
		availablePlans = append(availablePlans, fmt.Sprintf("cpu=%d, memory=%.1f, storage=%d, group=%s",
			plan.Cpu, plan.Memory, plan.Storage, plan.Group))
	}
	return nil, fmt.Errorf("no plan found matching requirements: cpu=%d, memory=%.1f, storage=%d, replicas=%d (group=%s). Available plans: %v",
		requiredCpu, requiredMemory, requiredStorage, requiredReplicas, requiredGroup, availablePlans)
}

func (r *cloudDatabaseClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clouddatabasecluster"
}

func (r *cloudDatabaseClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Cloud Database Cluster resource representing a managed database cluster on Nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the cloud database cluster",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the cloud database cluster",
			},
			"namespace": schema.StringAttribute{
				Required:    true,
				Description: "Name of the namespace that the cluster will belong to",
			},
			"spec": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:    true,
						Description: "Database type (e.g., postgresql, mysql)",
					},
					"version": schema.StringAttribute{
						Required:    true,
						Description: "Database version",
					},
				},
				Required:    true,
				Description: "Database specification including type and version",
			},
			"plan": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					// User inputs for plan selection
					"cpu": schema.Int64Attribute{
						Required:    true,
						Description: "Number of CPU cores required",
					},
					"memory": schema.Float64Attribute{
						Required:    true,
						Description: "Memory required in GB",
					},
					"storage": schema.Int64Attribute{
						Required:    true,
						Description: "Storage required in GB",
					},
					"replicas": schema.Int64Attribute{
						Required:    true,
						Description: "Number of replicas/nodes (1 = single node, 2 = redundant, 3 = highly available)",
						Validators: []validator.Int64{
							int64validator.Between(1, 3),
						},
					},
					// Computed fields
					"id": schema.StringAttribute{
						Computed:    true,
						Description: "Matched plan ID",
					},
					"name": schema.StringAttribute{
						Computed:    true,
						Description: "Matched plan name",
					},
					"group": schema.StringAttribute{
						Computed:    true,
						Description: "Matched plan group (e.g., 'Single (1 node)', 'Redundant (2 nodes)')",
					},
					"price": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"amount": schema.Int64Attribute{
								Computed:    true,
								Description: "Price amount in cents",
							},
							"currency": schema.StringAttribute{
								Computed:    true,
								Description: "Price currency",
							},
						},
						Computed:    true,
						Description: "Matched plan pricing information",
					},
				},
				Required:    true,
				Description: "Database cluster plan specification - provider will find matching plan",
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the cloud database cluster",
				Computed:    true,
			},
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

	var spec specResource
	diags = plan.Spec.As(ctx, &spec, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var planSpecs planResource
	diags = plan.Plan.As(ctx, &planSpecs, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Find matching plan based on user specifications
	client := api.NewClient()
	matchingPlan, err := findMatchingPlan(client, planSpecs)
	if err != nil {
		resp.Diagnostics.AddError("Plan Selection Error", "Could not find a plan matching your specifications: "+err.Error())
		return
	}

	input := api.CloudDatabaseClusterCreateInput{
		Name:      plan.Name.ValueString(),
		Namespace: plan.Namespace.ValueString(),
		Spec: api.CloudDatabaseClusterSpecInput{
			Type:    spec.Type.ValueString(),
			Version: spec.Version.ValueString(),
		},
		Plan: matchingPlan.Id,
	}

	cluster, err := client.CloudDatabaseClusterCreate(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cloud database cluster", "Could not create cluster: "+err.Error())
		return
	}

	plan.ID = types.StringValue(cluster.Id)
	plan.Name = types.StringValue(cluster.Name)
	plan.Namespace = types.StringValue(cluster.Namespace.Name)

	// Set plan object with both user specs and matched plan info
	var amount *int64
	if cluster.Plan.Price.Amount != nil {
		val := int64(*cluster.Plan.Price.Amount)
		amount = &val
	}

	planPriceObj := types.ObjectValueMust(
		map[string]attr.Type{
			"amount":   types.Int64Type,
			"currency": types.StringType,
		},
		map[string]attr.Value{
			"amount":   types.Int64PointerValue(amount),
			"currency": types.StringPointerValue(cluster.Plan.Price.Currency),
		},
	)

	// Use the user-specified replicas value
	replicas := planSpecs.Replicas

	planObj := types.ObjectValueMust(
		map[string]attr.Type{
			"cpu":      types.Int64Type,
			"memory":   types.Float64Type,
			"storage":  types.Int64Type,
			"replicas": types.Int64Type,
			"id":       types.StringType,
			"name":     types.StringType,
			"group":    types.StringType,
			"price": types.ObjectType{AttrTypes: map[string]attr.Type{
				"amount":   types.Int64Type,
				"currency": types.StringType,
			}},
		},
		map[string]attr.Value{
			"cpu":      types.Int64Value(int64(cluster.Plan.Cpu)),
			"memory":   types.Float64Value(cluster.Plan.Memory),
			"storage":  types.Int64Value(int64(cluster.Plan.Storage)),
			"replicas": replicas,
			"id":       types.StringValue(cluster.Plan.Id),
			"name":     types.StringValue(matchingPlan.Name), // Use the matched plan's name
			"group":    types.StringValue(cluster.Plan.Group),
			"price":    planPriceObj,
		},
	)
	plan.Plan = planObj

	specObj := types.ObjectValueMust(
		map[string]attr.Type{
			"type":    types.StringType,
			"version": types.StringType,
		},
		map[string]attr.Value{
			"type":    types.StringValue(cluster.Spec.Type),
			"version": types.StringValue(cluster.Spec.Version),
		},
	)
	plan.Spec = specObj

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudDatabaseClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudDatabaseClusterResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	clusters, err := client.CloudDatabaseClusterList()
	if err != nil {
		resp.Diagnostics.AddError("Error reading cloud database clusters", "Could not list clusters: "+err.Error())
		return
	}

	var cluster *api.CloudDatabaseClusterResult
	for _, c := range clusters {
		if c.Name == state.Name.ValueString() && c.Namespace.Name == state.Namespace.ValueString() {
			cluster = &c
			break
		}
	}

	if cluster == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(cluster.Id)
	state.Name = types.StringValue(cluster.Name)
	state.Namespace = types.StringValue(cluster.Namespace.Name)

	// Set plan object for Read - preserve from current state or use API data
	var currentPlan planResource
	if !state.Plan.IsNull() && !state.Plan.IsUnknown() {
		diags = state.Plan.As(ctx, &currentPlan, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	var amount *int64
	if cluster.Plan.Price.Amount != nil {
		val := int64(*cluster.Plan.Price.Amount)
		amount = &val
	}

	planPriceObj := types.ObjectValueMust(
		map[string]attr.Type{
			"amount":   types.Int64Type,
			"currency": types.StringType,
		},
		map[string]attr.Value{
			"amount":   types.Int64PointerValue(amount),
			"currency": types.StringPointerValue(cluster.Plan.Price.Currency),
		},
	)

	// Preserve user input values or derive from API group name
	replicas := types.Int64Value(1) // default
	if !currentPlan.Replicas.IsNull() && !currentPlan.Replicas.IsUnknown() {
		replicas = currentPlan.Replicas
	} else {
		// Try to derive replicas from group name if not preserved
		switch cluster.Plan.Group {
		case "Single (1 node)":
			replicas = types.Int64Value(1)
		case "Redundant (2 nodes)":
			replicas = types.Int64Value(2)
		case "Highly available (3 nodes)":
			replicas = types.Int64Value(3)
		}
	}

	planObj := types.ObjectValueMust(
		map[string]attr.Type{
			"cpu":      types.Int64Type,
			"memory":   types.Float64Type,
			"storage":  types.Int64Type,
			"replicas": types.Int64Type,
			"id":       types.StringType,
			"name":     types.StringType,
			"group":    types.StringType,
			"price": types.ObjectType{AttrTypes: map[string]attr.Type{
				"amount":   types.Int64Type,
				"currency": types.StringType,
			}},
		},
		map[string]attr.Value{
			"cpu":      types.Int64Value(int64(cluster.Plan.Cpu)),
			"memory":   types.Float64Value(cluster.Plan.Memory),
			"storage":  types.Int64Value(int64(cluster.Plan.Storage)),
			"replicas": replicas,
			"id":       types.StringValue(cluster.Plan.Id),
			"name":     types.StringNull(), // Plan result doesn't include name
			"group":    types.StringValue(cluster.Plan.Group),
			"price":    planPriceObj,
		},
	)
	state.Plan = planObj

	specObj := types.ObjectValueMust(
		map[string]attr.Type{
			"type":    types.StringType,
			"version": types.StringType,
		},
		map[string]attr.Value{
			"type":    types.StringValue(cluster.Spec.Type),
			"version": types.StringValue(cluster.Spec.Version),
		},
	)
	state.Spec = specObj

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudDatabaseClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cloudDatabaseClusterResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := api.CloudDatabaseClusterModifyInput{
		Name:      plan.Name.ValueString(),
		Namespace: plan.Namespace.ValueString(),
	}

	client := api.NewClient()
	cluster, err := client.CloudDatabaseClusterModify(input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating cloud database cluster", "Could not update cluster: "+err.Error())
		return
	}

	plan.ID = types.StringValue(cluster.Id)
	plan.Name = types.StringValue(cluster.Name)
	plan.Namespace = types.StringValue(cluster.Namespace.Name)

	// Get current plan specs from the plan
	var currentPlan planResource
	if !plan.Plan.IsNull() && !plan.Plan.IsUnknown() {
		diags = plan.Plan.As(ctx, &currentPlan, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	// Set plan object with both user specs and API data
	var amount *int64
	if cluster.Plan.Price.Amount != nil {
		val := int64(*cluster.Plan.Price.Amount)
		amount = &val
	}

	planPriceObj := types.ObjectValueMust(
		map[string]attr.Type{
			"amount":   types.Int64Type,
			"currency": types.StringType,
		},
		map[string]attr.Value{
			"amount":   types.Int64PointerValue(amount),
			"currency": types.StringPointerValue(cluster.Plan.Price.Currency),
		},
	)

	planObj := types.ObjectValueMust(
		map[string]attr.Type{
			"cpu":      types.Int64Type,
			"memory":   types.Float64Type,
			"storage":  types.Int64Type,
			"replicas": types.Int64Type,
			"id":       types.StringType,
			"name":     types.StringType,
			"group":    types.StringType,
			"price": types.ObjectType{AttrTypes: map[string]attr.Type{
				"amount":   types.Int64Type,
				"currency": types.StringType,
			}},
		},
		map[string]attr.Value{
			"cpu":      currentPlan.Cpu,      // Preserve user input
			"memory":   currentPlan.Memory,   // Preserve user input
			"storage":  currentPlan.Storage,  // Preserve user input
			"replicas": currentPlan.Replicas, // Preserve user input
			"id":       types.StringValue(cluster.Plan.Id),
			"name":     types.StringValue(cluster.Plan.Name),
			"group":    types.StringValue(cluster.Plan.Group),
			"price":    planPriceObj,
		},
	)
	plan.Plan = planObj

	specObj := types.ObjectValueMust(
		map[string]attr.Type{
			"type":    types.StringType,
			"version": types.StringType,
		},
		map[string]attr.Value{
			"type":    types.StringValue(cluster.Spec.Type),
			"version": types.StringValue(cluster.Spec.Version),
		},
	)
	plan.Spec = specObj

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudDatabaseClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudDatabaseClusterResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	input := api.CloudDatabaseClusterResourceInput{
		Name:      state.Name.ValueString(),
		Namespace: state.Namespace.ValueString(),
	}

	_, err := client.CloudDatabaseClusterDelete(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting cloud database cluster",
			fmt.Sprintf("Failed to delete cluster %q: %s", state.Name.ValueString(), err.Error()),
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
	name := parts[1]

	client := api.NewClient()
	clusters, err := client.CloudDatabaseClusterList()
	if err != nil {
		resp.Diagnostics.AddError("Error importing cloud database cluster", "Could not list clusters: "+err.Error())
		return
	}

	var cluster *api.CloudDatabaseClusterResult
	for _, c := range clusters {
		if c.Name == name && c.Namespace.Name == namespace {
			cluster = &c
			break
		}
	}

	if cluster == nil {
		resp.Diagnostics.AddError(
			"Error importing cloud database cluster",
			fmt.Sprintf("Unable to find cluster %q in namespace %q", name, namespace),
		)
		return
	}

	// Set plan object for ImportState
	var amount *int64
	if cluster.Plan.Price.Amount != nil {
		val := int64(*cluster.Plan.Price.Amount)
		amount = &val
	}

	planPriceObj := types.ObjectValueMust(
		map[string]attr.Type{
			"amount":   types.Int64Type,
			"currency": types.StringType,
		},
		map[string]attr.Value{
			"amount":   types.Int64PointerValue(amount),
			"currency": types.StringPointerValue(cluster.Plan.Price.Currency),
		},
	)

	// Derive replicas from group name for imported resources
	var replicas int64 = 1 // default
	switch cluster.Plan.Group {
	case "Single (1 node)":
		replicas = 1
	case "Redundant (2 nodes)":
		replicas = 2
	case "Highly available (3 nodes)":
		replicas = 3
	}

	planObj := types.ObjectValueMust(
		map[string]attr.Type{
			"cpu":      types.Int64Type,
			"memory":   types.Float64Type,
			"storage":  types.Int64Type,
			"replicas": types.Int64Type,
			"id":       types.StringType,
			"name":     types.StringType,
			"group":    types.StringType,
			"price": types.ObjectType{AttrTypes: map[string]attr.Type{
				"amount":   types.Int64Type,
				"currency": types.StringType,
			}},
		},
		map[string]attr.Value{
			"cpu":      types.Int64Value(int64(cluster.Plan.Cpu)),
			"memory":   types.Float64Value(cluster.Plan.Memory),
			"storage":  types.Int64Value(int64(cluster.Plan.Storage)),
			"replicas": types.Int64Value(replicas),
			"id":       types.StringValue(cluster.Plan.Id),
			"name":     types.StringNull(), // Plan result doesn't include name
			"group":    types.StringValue(cluster.Plan.Group),
			"price":    planPriceObj,
		},
	)

	state := cloudDatabaseClusterResource{
		ID:        types.StringValue(cluster.Id),
		Name:      types.StringValue(cluster.Name),
		Namespace: types.StringValue(cluster.Namespace.Name),
		Plan:      planObj,
	}

	specObj := types.ObjectValueMust(
		map[string]attr.Type{
			"type":    types.StringType,
			"version": types.StringType,
		},
		map[string]attr.Value{
			"type":    types.StringValue(cluster.Spec.Type),
			"version": types.StringValue(cluster.Spec.Version),
		},
	)
	state.Spec = specObj
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
