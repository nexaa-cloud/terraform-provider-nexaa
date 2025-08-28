// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data_sources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &cloudDatabasePlansDataSource{}
	_ datasource.DataSourceWithConfigure = &cloudDatabasePlansDataSource{}
)

// NewCloudDatabaseClusterPlans is a helper function to simplify the provider implementation.
func NewCloudDatabaseClusterPlans() datasource.DataSource {
	return &cloudDatabasePlansDataSource{}
}

type cloudDatabasePlansDataSource struct {
}

// cloudDatabasePlansDataSource is the data source implementation.
type cloudDatabasePlansDataSourceModel struct {
	Id       types.String `tfsdk:"id"`
	Replicas types.Int64  `tfsdk:"replicas"`
	Cpu      types.Int64  `tfsdk:"cpu"`
	Memory   types.Int64  `tfsdk:"memory"`
	Storage  types.Int64  `tfsdk:"storage"`
}

// Metadata returns the data source type name.
func (d *cloudDatabasePlansDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_database_cluster_plans"
}

// Schema defines the schema for the data source.
func (d *cloudDatabasePlansDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a specific user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Plan identifier",
				Computed:            true,
			},
			"cpu": schema.Int64Attribute{
				MarkdownDescription: "Number of CPU cores required",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(
						1,
						2,
						4,
						8,
						16,
					),
				},
			},
			"memory": schema.Int64Attribute{
				MarkdownDescription: "Memory in MB required",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(
						2,
						4,
						8,
						12,
						16,
						24,
						32,
						48,
						64,
					),
				},
			},
			"storage": schema.Int64Attribute{
				MarkdownDescription: "Storage in GB required",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(
						10,
						20,
						40,
						60,
						80,
						100,
						120,
						160,
						240,
						320,
						480,
					),
				},
			},
			"replicas": schema.Int64Attribute{
				MarkdownDescription: "Number of replicas required",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(
						1,
						2,
						3,
					),
				},
			},
		},
	}
}

// Configure initializes the data source and retrieves the list of available database cluster plans.
func (d *cloudDatabasePlansDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *cloudDatabasePlansDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data cloudDatabasePlansDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	if client == nil {
		resp.Diagnostics.AddError("Error creating API client", "Failed to create API client")
		return
	}

	plans, err := client.CloudDatabaseClusterListPlans()
	if err != nil {
		resp.Diagnostics.AddError("Error reading plans", err.Error())
		return
	}

	plan, err := getPlan(
		plans,
		data.Replicas.ValueInt64(),
		data.Cpu.ValueInt64(),
		data.Memory.ValueInt64(),
		data.Storage.ValueInt64(),
	)

	resp.Diagnostics.AddWarning("plan found", fmt.Sprintf("Plan found: %s (CPU: %d, Memory: %d MB, Storage: %d GB, Replicas: %d)",
		plan.Id.ValueString(),
		plan.Cpu.ValueInt64(),
		plan.Memory.ValueInt64(),
		plan.Storage.ValueInt64(),
		plan.Replicas.ValueInt64()))

	if err != nil {
		resp.Diagnostics.AddError("Error could not find plan", err.Error())
	}

	data.Id = types.StringValue(plan.Id.ValueString())
	diags := resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diags...)
}

func getPlan(plans []api.CloudDatabaseClusterPlan, Replicas int64, Cpu int64, Memory int64, Storage int64) (cloudDatabasePlansDataSourceModel, error) {
	Group := translateReplicasToGroup(Replicas)
	var planId string
	for _, plan := range plans {
		if plan.Group != Group {
			continue
		}

		if plan.Cpu != int(Cpu) {
			continue
		}

		if int(plan.Memory) != int(Memory) {
			continue
		}

		if plan.Storage != int(Storage) {
			continue
		}

		planId = plan.Id
	}

	if planId == "" {
		var sb strings.Builder
		sb.WriteString("No plan found for the given parameters, These are the available plans: \n")
		sb.WriteString("ID\tNAME\tCPU\tSTORAGE\tRAM\tGROUP\t")
		for _, plan := range plans {
			sb.WriteString(fmt.Sprintf("%q \t%q \t%d \t%d \t%g \t%q \n", plan.Id, plan.Name, plan.Cpu, plan.Storage, plan.Memory, plan.Group))
		}

		return cloudDatabasePlansDataSourceModel{}, errors.New(sb.String())
	}

	return cloudDatabasePlansDataSourceModel{
		Id:       types.StringValue(planId),
		Cpu:      types.Int64Value(Cpu),
		Memory:   types.Int64Value(Memory),
		Storage:  types.Int64Value(Storage),
		Replicas: types.Int64Value(Replicas),
	}, nil
}

func translateReplicasToGroup(Replicas int64) string {
	switch Replicas {
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
