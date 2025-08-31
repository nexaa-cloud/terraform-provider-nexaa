// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/nexaa-cloud/nexaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &namespaceResource{}
	_ resource.ResourceWithImportState = &namespaceResource{}
)

// NewNamespaceResource is a helper function to simplify the provider implementation.
func NewNamespaceResource() resource.Resource {
	return &namespaceResource{}
}

// namespaceResource is the resource implementation.
type namespaceResource struct {
	ID          types.String   `tfsdk:"id"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	LastUpdated types.String   `tfsdk:"last_updated"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

// Metadata returns the resource type name.
func (r *namespaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_namespace"
}

// Schema defines the schema for the resource.
func (r *namespaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the namespace, equal to the name",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the namespace",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the namespace",
				Optional:    true,
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the namespace",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(context.Background(), timeouts.Opts{
				Delete: true,
			}),
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *namespaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan namespaceResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := api.NamespaceCreateInput{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueStringPointer(),
	}

	client := api.NewClient()
	namespace, err := client.NamespaceCreate(input)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating namespace",
			"Could not create namespace, error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(namespace.Name)
	plan.Name = types.StringValue(namespace.Name)
	plan.Description = types.StringValue(namespace.Description)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *namespaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state namespaceResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	namespace, err := client.NamespaceListByName(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Namespace",
			"Could not find namespace "+state.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	state.ID = types.StringValue(namespace.Name)
	state.Name = types.StringValue(namespace.Name)
	state.Description = types.StringValue(namespace.Description)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *namespaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan namespaceResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.AddError(
		"Error updating namespace",
		"You can't change the name of your namespace, you can only create and delete a namespace.",
	)

	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *namespaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state namespaceResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, 5*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	client := api.NewClient()
	namespaceName := state.Name.ValueString()
	err := waitForAllChildrenToBeRemoved(ctx, *client, namespaceName)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting namespace", err.Error())
		return
	}

	_, err = client.NamespaceDelete(state.ID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting namespace",
			"Could not delete namespace "+state.Name.ValueString()+": "+err.Error(),
		)
	}
}

func (r *namespaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	client := api.NewClient()

	id := req.ID
	item, err := client.NamespaceListByName(id)
	if err != nil {
		resp.Diagnostics.AddError("Error listing namespaces", err.Error())
		return
	}

	if item.Name == id {
		resp.State.SetAttribute(ctx, path.Root("name"), item.Name)
		resp.State.SetAttribute(ctx, path.Root("description"), item.Description)
		resp.State.SetAttribute(ctx, path.Root("last_updated"), time.Now().Format(time.RFC850))
		return
	}
	resp.Diagnostics.AddError(
		"Error importing namespace",
		"Could not find namespace with name: "+id,
	)
}

func waitForAllChildrenToBeRemoved(ctx context.Context, client api.Client, namespaceName string) error {
	const (
		initialDelay = 2 * time.Second
		maxDelay     = 15 * time.Second
	)
	delay := initialDelay

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		namespace, err := client.NamespaceListByName(namespaceName)
		if err != nil {
			return err
		}

		if !namespaceHasContainers(namespace) && !namespaceHasContainerJobs(namespace) && !namespaceHasCloudDatabaseClusters(namespace) && !namespaceHasVolumes(namespace) {
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

func namespaceHasContainers(namespace api.NamespaceResult) bool {
	return len(namespace.Containers) != 0
}

func namespaceHasContainerJobs(namespace api.NamespaceResult) bool {
	return len(namespace.ContainerJobs) != 0
}

func namespaceHasVolumes(namespace api.NamespaceResult) bool {
	return len(namespace.Volumes) != 0
}

func namespaceHasCloudDatabaseClusters(namespace api.NamespaceResult) bool {
	return len(namespace.CloudDatabaseClusters) != 0
}
