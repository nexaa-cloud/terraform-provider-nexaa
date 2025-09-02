// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
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
	ID          types.String   `tfsdk:"id"`
	Namespace   types.String   `tfsdk:"namespace"`
	Name        types.String   `tfsdk:"name"`
	Size        types.Int64    `tfsdk:"size"`
	Status      types.String   `tfsdk:"status"`
	LastUpdated types.String   `tfsdk:"last_updated"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

// Metadata returns the resource type name.
func (r *volumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

// Schema defines the schema for the resource.
func (r *volumeResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"status": schema.StringAttribute{
				Description: "The status of the volume",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the volume",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Delete: true,
			}),
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

	client := api.NewClient()
	volume, err := client.VolumeCreate(input)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating volume",
			"Encountered error while creating a volume: "+err.Error(),
		)
		return
	}

	plan = translateApiToVolumeResource(plan, volume)

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

	state = translateApiToVolumeResource(state, *volume)

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

	plan = translateApiToVolumeResource(plan, volume)

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

	deleteTimeout, diags := state.Timeouts.Delete(ctx, 2*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	client := api.NewClient()

	namespaceName := state.Namespace.ValueString()
	volumeName := state.Name.ValueString()

	err := waitForUnlocked(ctx, volumeLocked(), *client, namespaceName, volumeName)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting volume", "Could not reach a unlocked state: "+err.Error())
		return
	}

	err = waitForAllContainersToBeUnmounted(ctx, *client, namespaceName, volumeName)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting volume", "Could not reach a unmounted state: "+err.Error())
		return
	}

	_, err = client.VolumeDelete(namespaceName, volumeName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting volume",
			"Could not delete volume "+volumeName+": "+err.Error(),
		)
		return
	}
}

// ImportState implements resource.ResourceWithImportState.
func (r *volumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expect import ID as "namespace/volumeName"
	id, err := unpackNamespaceChildId(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			err.Error(),
		)
		return
	}

	client := api.NewClient()
	// Fetch the volume using the namespace and volume name
	volume, err := client.ListVolumeByName(id.Namespace, id.Name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Volume",
			"Could not read volume "+id.Name+": "+err.Error(),
		)
		return
	}

	if volume == nil {
		resp.Diagnostics.AddError(
			"Error Reading Volume",
			"Could not read volume "+id.Name+", volume not found",
		)
		return
	}

	var state volumeResource
	state = translateApiToVolumeResource(state, *volume)
	state.Timeouts = timeouts.Value{
		Object: types.ObjectValueMust(
			map[string]attr.Type{
				"delete": types.StringType,
			},
			map[string]attr.Value{
				"delete": types.StringValue("2m"),
			},
		),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func waitForAllContainersToBeUnmounted(ctx context.Context, client api.Client, namespace string, volumeName string) error {
	const (
		initialDelay = 2 * time.Second
		maxDelay     = 15 * time.Second
	)
	delay := initialDelay

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		volume, err := client.ListVolumeByName(namespace, volumeName)
		if err != nil {
			return err
		}

		if volume == nil {
			return fmt.Errorf("volume %s not found", volumeName)
		}

		if !hasContainersAttached(*volume) && !hasContainerJobsAttached(*volume) {
			break
		}

		// Backoff between polls
		time.Sleep(delay)
		if delay < maxDelay {
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}

	return nil
}

func hasContainersAttached(volume api.VolumeResult) bool {
	return len(volume.Containers) != 0
}

func hasContainerJobsAttached(volume api.VolumeResult) bool {
	return len(volume.ContainerJobs) != 0
}
