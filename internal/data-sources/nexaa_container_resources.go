// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data_sources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
	"github.com/nexaa-cloud/terraform-provider-nexaa/internal/enums"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &containerResourceDataSource{}
	_ datasource.DataSourceWithConfigure = &containerResourceDataSource{}
)

// NewContainerResources is a helper function to simplify the provider implementation.
func NewContainerResources() datasource.DataSource {
	return &containerResourceDataSource{}
}

type containerResourceDataSource struct {
}

// cloudDatabasePlansDataSource is the data source implementation.
type containerResourceDataSourceModel struct {
	Id     types.String  `tfsdk:"id"`
	Cpu    types.Float64 `tfsdk:"cpu"`
	Memory types.Float64 `tfsdk:"memory"`
}

// Metadata returns the data source type name.
func (d *containerResourceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_resources"
}

// Schema defines the schema for the data source.
func (d *containerResourceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a specific user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier",
				Computed:            true,
			},
			"cpu": schema.Float64Attribute{
				Required:    true,
				Description: "The amount of cpu used for the container, can be the following values: 0.25, 0.5, 0.75, 1, 2, 3, 4",
				Validators: []validator.Float64{
					float64validator.OneOf(enums.CPU...),
				},
			},
			"memory": schema.Float64Attribute{
				Required:    true,
				Description: "The amount of memory used for the container (in GB), can be the following values: 0.5, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16",
				Validators: []validator.Float64{
					float64validator.OneOf(enums.RAM...),
				},
			},
		},
	}
}

// Configure initializes the data source and retrieves the list of available database cluster plans.
func (d *containerResourceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *containerResourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data containerResourceDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	cpu := int(data.Cpu.ValueFloat64() * 1000)
	memory := int(data.Memory.ValueFloat64() * 1000)
	combined := fmt.Sprintf("CPU_%d_RAM_%d", cpu, memory)

	isValid := false
	for _, resource := range api.AllContainerResources {
		if string(resource) == combined {
			isValid = true
			break
		}
	}

	if !isValid {
		resp.Diagnostics.AddError(
			"Invalid container resource combination",
			fmt.Sprintf("CPU %g and RAM %g GB is not a valid combination", data.Cpu.ValueFloat64(), data.Memory.ValueFloat64()),
		)
		return
	}

	data.Id = types.StringValue(combined)
	diags := resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diags...)
}
