// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/tilaa/tilaa-cli/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &volumeResource{}
	_ resource.ResourceWithImportState = &volumeResource{}
)

// NewVolumeResource is a helper function to simplify the provider implementation.
func NewVolumeResource() resource.Resource {
	return &volumeResource{}
}

// volumeResource is the resource implementation.
type volumeResource struct {
	ID          types.String `tfsdk:"id"`
	Namespace   types.String `tfsdk:"namespace"`
	Name        types.String `tfsdk:"name"`
	Size        types.Int64  `tfsdk:"size"`
	Usage       types.Int64  `tfsdk:"usage"`
	Locked      types.Bool   `tfsdk:"locked"`
	Status      types.String `tfsdk:"status"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *volumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

// Schema defines the schema for the resource.
func (r *volumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the volume, equal to the name of the volume",
				Computed:    true,
			},
			"namespace": schema.StringAttribute{
				Description: "Name of the namespace where the volume is located",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the volume",
				Required:    true,
			},
			"size": schema.Int64Attribute{
				Description: "Size of the volume in GB, min 1GB/ max 100GB.",
				Required:    true,
			},
			"usage": schema.Int64Attribute{
				Description: "Amount of GB that is being used",
				Computed:    true,
			},
			"locked": schema.BoolAttribute{
				Description: "If the volume is locked it can't be edited",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of the volume",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the volume",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *volumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan volumeResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := api.VolumeCreateInput{
		Namespace: plan.Namespace.ValueString(),
		Name:      plan.Name.ValueString(),
		Size:      int(plan.Size.ValueInt64()),
	}

	const (
		maxRetries   = 4
		initialDelay = 3 * time.Second
	)
	delay := initialDelay
	var err error
	var volume api.VolumeResult

	client := api.NewClient()

	for i := 0; i <= maxRetries; i++ {
		volume, err = client.VolumeCreate(input)
		if err == nil {
			break
		}

		time.Sleep(delay)
		delay *= 2
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating volume",
			"Encountered error while creating a volume: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(volume.Name)
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(volume.Name)
	plan.Size = types.Int64Value(int64(volume.Size))
	plan.Usage = types.Int64Value(int64(volume.Usage))
	plan.Locked = types.BoolValue(volume.Locked)
	plan.Status = types.StringValue(volume.State)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *volumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state volumeResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()

	volume, err := client.ListVolumeByName(state.Namespace.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Volume",
			"Could not read volume "+state.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	state.ID = types.StringValue(volume.Name)
	state.Namespace = types.StringValue(state.Namespace.ValueString())
	state.Name = types.StringValue(volume.Name)
	state.Size = types.Int64Value(int64(volume.Size))
	state.Usage = types.Int64Value(int64(volume.Usage))
	state.Locked = types.BoolValue(volume.Locked)
	state.Status = types.StringValue(volume.State)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *volumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan volumeResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := api.VolumeModifyInput{
		Namespace: plan.Namespace.ValueString(),
		Name:      plan.Name.ValueString(),
		Size:      int(plan.Size.ValueInt64()),
	}

	client := api.NewClient()

	volume, err := client.VolumeIncrease(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Volume",
			"Could not update volume, error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(volume.Name)
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(volume.Name)
	plan.Size = types.Int64Value(int64(volume.Size))
	plan.Usage = types.Int64Value(int64(volume.Usage))
	plan.Locked = types.BoolValue(volume.Locked)
	plan.Status = types.StringValue(volume.State)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *volumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state volumeResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	const (
		maxRetries   = 10
		initialDelay = 2 * time.Second
	)
	delay := initialDelay

	client := api.NewClient()

	for i := 0; i <= maxRetries; i++ {
		volume, err := client.ListVolumeByName(state.Namespace.ValueString(), state.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting volume",
				fmt.Sprintf("Could not find volume with name %q: %s", state.Name.ValueString(), err.Error()),
			)
			return
		}
		if volume.State == "created" || volume.State == "failed" {
			_, err := client.VolumeDelete(state.Namespace.ValueString(), state.Name.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error deleting volume",
					fmt.Sprintf("Failed to delete volume %q: %s", state.Name.ValueString(), err.Error()),
				)
				return
			}
			return
		}
		if volume.State == "failed" && volume.Locked {
			resp.Diagnostics.AddError(
				"Error deleting volume",
				fmt.Sprintf("Failed to delete volume %q, the volume is locked and could not be deleted", state.Name.ValueString()),
			)
			return
		}
		time.Sleep(delay)
		delay *= 2
	}

	resp.Diagnostics.AddError(
		"Timeout deleting volume",
		fmt.Sprintf("Volume %q did not reach a deletable state after a couple retries", state.Name.ValueString()),
	)
}

// ImportState implements resource.ResourceWithImportState.
func (r *volumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expect import ID as "namespace/volumeName"
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format \"<namespace>/<volume_name>\", got: "+req.ID,
		)
		return
	}
	ns := parts[0]
	volName := parts[1]

	client := api.NewClient()
	// Fetch the volume using the namespace and volume name
	volume, err := client.ListVolumeByName(ns, volName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Volume",
			"Could not read volume "+volName+": "+err.Error(),
		)
		return
	}

	// Set the volume attributes in the state
	resp.State.SetAttribute(ctx, path.Root("id"), volume.Name)
	resp.State.SetAttribute(ctx, path.Root("namespace"), ns)
	resp.State.SetAttribute(ctx, path.Root("name"), volume.Name)
	resp.State.SetAttribute(ctx, path.Root("size"), int64(volume.Size))
	resp.State.SetAttribute(ctx, path.Root("usage"), int64(volume.Usage))
	resp.State.SetAttribute(ctx, path.Root("status"), volume.State)
	resp.State.SetAttribute(ctx, path.Root("last_updated"), time.Now().Format(time.RFC850))
}
