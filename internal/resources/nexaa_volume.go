// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
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
	ID            types.String `tfsdk:"id"`
	NamespaceName types.String `tfsdk:"namespace_name"`
	Name          types.String `tfsdk:"name"`
	Size          types.Int64  `tfsdk:"size"`
	Usage         types.Int64  `tfsdk:"usage"`
	Locked		  types.Bool   `tfsdk:"locked"`
	LastUpdated   types.String `tfsdk:"last_updated"`
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
				Description: "Numeric identifier of the volume.",
				Computed: true,
			},
			"namespace_name": schema.StringAttribute{
				Description: "Name of the namespace where the volume is located",
				Required: true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the volume",
				Required: true,
			},
			"size": schema.Int64Attribute{
				Description: "Size of the volume in GB, min 1GB/ max 100GB.",
				Required: true,
			},
			"usage": schema.Int64Attribute{
				Description: "Amount of GB that is being used",
				Computed: true,
			},
			"locked": schema.BoolAttribute{
				Description: "If the volume is locked it can't be edited",
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the volume",
				Computed: true,
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

	input := api.VolumeInput{
		Namespace: plan.NamespaceName.ValueString(),
		Name:      plan.Name.ValueString(),
		Size:      int(plan.Size.ValueInt64()),
	}

	time.Sleep(6 * time.Second)
	_, err := api.CreateVolume(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating volume",
			"Could not create volume, error: "+err.Error(),
		)
		return
	}

	volume, err := api.ListVolumeByName(plan.NamespaceName.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading volume",
			"Could not read volume, error: "+err.Error(),
		)
	}

	plan.ID = types.StringValue(volume.Id)
	plan.NamespaceName = types.StringValue(volume.Namespace)
	plan.Name = types.StringValue(volume.Name)
	plan.Size = types.Int64Value(int64(volume.Size))
	plan.Usage = types.Int64Value(int64(volume.Usage))
	plan.Locked = types.BoolValue(volume.Locked)
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

	volume, err := api.ListVolumeByName(state.NamespaceName.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Volume",
			"Could not read volume "+state.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	state.NamespaceName = types.StringValue(volume.Namespace)
	state.Name = types.StringValue(volume.Name)
	state.Size = types.Int64Value(int64(volume.Size))
	state.Usage = types.Int64Value(int64(volume.Usage))
	state.Locked = types.BoolValue(volume.Locked)

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


	input := api.VolumeInput{
		Namespace: plan.NamespaceName.ValueString(),
		Name:      plan.Name.ValueString(),
		Size:      int(plan.Size.ValueInt64()),
	}

	_, err := api.IncreaseVolume(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Volume",
			"Could not update volume, error: "+err.Error(),
		)
		return
	}

	volume, err := api.ListVolumeByName(plan.NamespaceName.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Volume",
			"Could not read volume "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(volume.Id)
	plan.NamespaceName = types.StringValue(volume.Namespace)
	plan.Name = types.StringValue(volume.Name)
	plan.Size = types.Int64Value(int64(volume.Size))
	plan.Usage = types.Int64Value(int64(volume.Usage))
	plan.Locked = types.BoolValue(volume.Locked)
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
        maxRetries = 4
        initialDelay = 5 * time.Second
    )
    delay := initialDelay

    // 1) Poll until unlocked or we exhaust retries
    var volume *api.Volume
    var err error
    for i := 0; i <= maxRetries; i++ {
        volume, err = api.ListVolumeByName(
            state.NamespaceName.ValueString(),
            state.Name.ValueString(),
        )
        if err != nil {
            if strings.Contains(err.Error(), "Not found") {
                // Already gone: warn & exit
                resp.Diagnostics.AddWarning(
					"Volume not found",
					"Volume is already deleted. Or the name and/or the namespace name is incorrect.",
				)
                return
            }
            resp.Diagnostics.AddError(
                "Error reading volume",
                "Could not read volume "+state.Name.ValueString()+": "+err.Error(),
            )
            return
        }
        if !volume.Locked {
            // Unlocked! Break out and delete
            break
        }
        // Still locked → wait & backoff
        time.Sleep(delay)
        delay *= 2
    }

    // 2) Now try the actual delete
    err = api.DeleteVolume(
        state.Name.ValueString(),
        state.NamespaceName.ValueString(),
    )
    if err != nil {
        if strings.Contains(err.Error(), "Not found") {
            // Already gone as of when we called DeleteVolume
            resp.Diagnostics.AddWarning(
                "Volume not found",
				"Volume is already deleted. Or the name and/or the namespace name is incorrect.",
            )
            return
        }
        resp.Diagnostics.AddError(
            "Error deleting volume",
            "Could not delete volume "+state.Name.ValueString()+": "+err.Error(),
        )
        return
    }
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

    // Fetch the volume using the namespace and volume name
    volume, err := api.ListVolumeByName(ns, volName)
    if err != nil {
        resp.Diagnostics.AddError(
            "Error Reading Volume",
            "Could not read volume "+volName+": "+err.Error(),
        )
        return
    }

    // Set the volume attributes in the state
    resp.State.SetAttribute(ctx, path.Root("id"), volume.Id)
    resp.State.SetAttribute(ctx, path.Root("namespace_name"), volume.Namespace)
    resp.State.SetAttribute(ctx, path.Root("name"), volume.Name)
    resp.State.SetAttribute(ctx, path.Root("size"), int64(volume.Size))
    resp.State.SetAttribute(ctx, path.Root("usage"), int64(volume.Usage))
    resp.State.SetAttribute(ctx, path.Root("last_updated"), time.Now().Format(time.RFC850))
}