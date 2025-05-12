// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"strconv"
	"time"

	"gitlab.com/tilaa/tilaa-cli/api"

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
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	LastUpdated types.String `tfsdk:"last_updated"`
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
                Description: "Numeric identifier of the namespace",
				Computed: true,
			},
			"name": schema.StringAttribute{
                Description: "Name of the namespace",
				Required: true,
			},
			"description": schema.StringAttribute{
                Description: "Description of the namespace",
				Optional: true,
                Computed: true,
			},
			"last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update of the namespace",
				Computed: true,
			},
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

	err := api.CreateNamespace(plan.Name.ValueString(), plan.Description.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating namespace",
			"Could not create namespace, error: "+err.Error(),
		)
        return
	}

    namespace, err := api.ListNamespaceByName(plan.Name.ValueString())

	// namespaces, err := api.ListNamespaces()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Namespace",
			"Could not find namespace "+plan.Name.ValueString()+": "+err.Error(),
		)
        return
	}

    plan.ID = types.StringValue(namespace.Id)
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

	namespace, err := api.ListNamespaceByName(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Namespace",
			"Could not find namespace "+state.Name.ValueString()+": "+err.Error(),
		)
        return
	}

    state.ID = types.StringValue(namespace.Id)
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
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddError(
		"Update method for namespaces doesn't exist",
		"You can't change the name of your namespace",
	)

	if resp.Diagnostics.HasError(){
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

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		return
	}

	err1 := api.DeleteNamespace(id)
	if err1 != nil {
		resp.Diagnostics.AddError(
			"Error deleting namespace",
			"Could not delete namespace, error: "+err1.Error(),
		)
		return
	}
}

func (r *namespaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // 1) Passthrough the ID field
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

    // 2) Use that ID to fetch name & description
    id := req.ID
    list, err := api.ListNamespaces()
    if err != nil {
        resp.Diagnostics.AddError("Error listing namespaces", err.Error())
        return
    }
    for _, item := range list {
        if item.Id == id {
            // Populate name & description in the state
            resp.State.SetAttribute(ctx, path.Root("name"), item.Name)
            resp.State.SetAttribute(ctx, path.Root("description"), item.Description)
            // Optionally set last_updated
            resp.State.SetAttribute(ctx, path.Root("last_updated"), time.Now().Format(time.RFC850))
            return
        }
    }
    resp.Diagnostics.AddError(
        "Error importing namespace",
        "Could not find namespace with ID: "+id,
    )
}
