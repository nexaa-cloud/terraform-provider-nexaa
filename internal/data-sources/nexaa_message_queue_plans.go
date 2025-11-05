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
	_ datasource.DataSource              = &messageQueuePlansDataSource{}
	_ datasource.DataSourceWithConfigure = &messageQueuePlansDataSource{}
)

// NewMessageQueuePlans is a helper function to simplify the provider implementation.
func NewMessageQueuePlans() datasource.DataSource {
	return &messageQueuePlansDataSource{}
}

type messageQueuePlansDataSource struct {
}

// messageQueuePlansDataSourceModel is the data source implementation.
type messageQueuePlansDataSourceModel struct {
	Id       types.String `tfsdk:"id"`
	Replicas types.Int64  `tfsdk:"replicas"`
	Cpu      types.Int64  `tfsdk:"cpu"`
	Memory   types.Int64  `tfsdk:"memory"`
	Storage  types.Int64  `tfsdk:"storage"`
}

// Metadata returns the data source type name.
func (d *messageQueuePlansDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_message_queue_plans"
}

// Schema defines the schema for the data source.
func (d *messageQueuePlansDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the plan ID for a message queue based on resource requirements.",
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
					),
				},
			},
			"memory": schema.Int64Attribute{
				MarkdownDescription: "Memory in GB required",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(
						2,
						4,
						8,
						16,
						32,
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
						80,
					),
				},
			},
			"replicas": schema.Int64Attribute{
				MarkdownDescription: "Number of replicas required",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(
						1,
						3,
					),
				},
			},
		},
	}
}

// Configure initializes the data source.
func (d *messageQueuePlansDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *messageQueuePlansDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data messageQueuePlansDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	if client == nil {
		resp.Diagnostics.AddError("Error creating API client", "Failed to create API client")
		return
	}

	plans, err := client.MessageQueuePlans()
	if err != nil {
		resp.Diagnostics.AddError("Error reading message queue plans", err.Error())
		return
	}

	plan, err := getMessageQueuePlan(
		plans,
		data.Replicas.ValueInt64(),
		data.Cpu.ValueInt64(),
		data.Memory.ValueInt64(),
		data.Storage.ValueInt64(),
	)

	if err != nil {
		resp.Diagnostics.AddError("Error could not find message queue plan", err.Error())
		return
	}

	data.Id = types.StringValue(plan.Id.ValueString())
	diags := resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diags...)
}

func getMessageQueuePlan(plans []api.MessageQueuePlanResult, Replicas int64, Cpu int64, Memory int64, Storage int64) (messageQueuePlansDataSourceModel, error) {
	var planId string

	for _, plan := range plans {
		if plan.Replicas != int(Replicas) {
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
		break
	}

	if planId == "" {
		var sb strings.Builder
		sb.WriteString("No plan found for the given parameters. These are the available plans:\n")
		sb.WriteString("ID\tNAME\tCPU\tRAM (GB)\tSTORAGE (GB)\tREPLICAS\tGROUP\n")
		for _, plan := range plans {
			sb.WriteString(fmt.Sprintf("%q\t%q\t%d\t%.1f\t%d\t%d\t%q\n",
				plan.Id, plan.Name, plan.Cpu, plan.Memory, plan.Storage, plan.Replicas, plan.Group))
		}

		return messageQueuePlansDataSourceModel{}, errors.New(sb.String())
	}

	return messageQueuePlansDataSourceModel{
		Id:       types.StringValue(planId),
		Cpu:      types.Int64Value(Cpu),
		Memory:   types.Int64Value(Memory),
		Storage:  types.Int64Value(Storage),
		Replicas: types.Int64Value(Replicas),
	}, nil
}
