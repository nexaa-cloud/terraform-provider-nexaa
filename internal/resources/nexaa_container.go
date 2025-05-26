// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"strings"
	"time"

	"gitlab.com/tilaa/tilaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &containerResource{}
	_ resource.ResourceWithImportState = &containerResource{}
)

// NewContainerResource is a helper function to simplify the provider implementation.
func NewContainerResource() resource.Resource {
	return &containerResource{}
}

// containerResource is the resource implementation.
type containerResource struct {
	ID          			types.String 				`tfsdk:"id"`
	Name        			types.String 				`tfsdk:"name"`
	Image 					types.String 				`tfsdk:"image"`
	Registry 				types.String	 			`tfsdk:"registry"`
	EnvironmentVariables 	[]environvariableResource	`tfsdk:"environment_variables"`
	Ports					[]types.String				`tfsdk:"ports"`
	Ingresses				[]ingresResource			`tfsdk:"ingresses"`
	Mounts	 				[]mountResource				`tfsdk:"mounts"`
	HealthCheck				healthcheckResource			`tfsdk:"health_check"`
}

type mountResource struct {
	Path 	types.String	`tfsdk:"path"`
	Volume 	types.String	`tfsdk:"volume"`
}

type environvariableResource struct {
	ID			types.String	`tfsdk:"id"`
	Name 		types.String	`tfsdk:"name"`
	Value 		types.String	`tfsdk:"value"`
	Secret 		types.Bool		`tfsdk:"secret"`
}

type ingresResource struct {
	DomainName	types.String	`tfsdk:"domain_name"`
	Port		types.String	`tfsdk:"port"`
	TLS 		types.Bool		`tfsdk:"tls"`
	AllowList	[]types.String	`tfsdk:"allow_list"`

}

type healthcheckResource struct {
	Port		types.Int32		`tfsdk:"port"`
	Path 		types.String	`tfsdk:"path"`
}

// Metadata returns the resource type name.
func (r *containerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container"
}

// Schema defines the schema for the resource.
func (r *containerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Container resource representing a deployable service.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"image": schema.StringAttribute{
				Required: true,
			},
			"registry": schema.StringAttribute{
				Required: true,
			},
			"ports": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"environment_variables": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Optional: true,
						},
						"name": schema.StringAttribute{
							Required: true,
						},
						"value": schema.StringAttribute{
							Optional: true,
						},
						"secret": schema.BoolAttribute{
							Optional: true,
						},
					},
				},
				Optional: true,
			},
			"ingresses": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"domain_name": schema.StringAttribute{Required: true},
						"port":        schema.StringAttribute{Required: true},
						"tls":         schema.BoolAttribute{Optional: true},
						"allow_list": schema.ListAttribute{
							ElementType: types.StringType,
							Optional:    true,
						},
					},
				},
				Optional: true,
			},
			"mounts": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path":  schema.StringAttribute{Required: true},
						"volume": schema.StringAttribute{Required: true},
					},
				},
				Optional: true,
			},
			"health_check": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"port": schema.Int64Attribute{Required: true},
					"path": schema.StringAttribute{Required: true},
				},
				Optional: true,
			},
		},
	}
}


// Create creates the resource and sets the initial Terraform state.
func (r *containerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan containerResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := api.ContainerInput {
		
	}

	_, err := api.CreateContainer(input)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating container",
			"Could not create container, error: "+err.Error(),
		)
        return
	}

    container, err := api.ListContainerByName(plan.Name.ValueString())

	// containers, err := api.ListContainers()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Container",
			"Could not find container "+plan.Name.ValueString()+": "+err.Error(),
		)
        return
	}

    plan.ID = types.StringValue(container.Name)
    plan.Name = types.StringValue(container.Name)
    plan.Description = types.StringValue(container.Description)
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *containerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state containerResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	container, err := api.ListContainerByName(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Container",
			"Could not find container "+state.Name.ValueString()+": "+err.Error(),
		)
        return
	}

    state.ID = types.StringValue(container.Name)
    state.Name = types.StringValue(container.Name)
    state.Description = types.StringValue(container.Description)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *containerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan containerResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.AddError(
		"Error updating container",
		"You can't change the name of your container, you can only create and delete a container.",
	)

	if resp.Diagnostics.HasError(){
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *containerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state containerResource
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    const (
        maxRetries   = 4
        initialDelay = 10 * time.Second
    )
    delay := initialDelay

	var err error

    // Retry DeleteContainer until it no longer complains about "locked"
    for i := 0; i <= maxRetries; i++ {
        err = api.DeleteContainer(state.ID.ValueString())
        if err == nil {
            // Success
            return
        }
        msg := err.Error()
        if strings.Contains(msg, "locked") {
            // Still locked—wait & back off
            time.Sleep(delay)
            delay *= 2
            continue
        }
        if strings.Contains(msg, "Not found") {
            // Gone already—treat as success
            resp.Diagnostics.AddWarning(
                "Container already deleted",
                "DeleteContainer returned Not Found; assuming success.",
            )
            return
        }
        // Any other error is fatal
        resp.Diagnostics.AddError(
            "Error deleting container",
            "Could not delete container "+state.Name.ValueString()+": "+msg,
        )
        return
    }

    // If we exit the loop still with locked error, report it
    resp.Diagnostics.AddError(
        "Timeout waiting for container to unlock",
        "Container is locked and can't be deleted, try again after a bit. Error: "+err.Error(),
    )
}




func (r *containerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

    id := req.ID
    list, err := api.ListContainers()
    if err != nil {
        resp.Diagnostics.AddError("Error listing containers", err.Error())
        return
    }
    for _, item := range list {
        if item.Name == id {
            resp.State.SetAttribute(ctx, path.Root("name"), item.Name)
            resp.State.SetAttribute(ctx, path.Root("description"), item.Description)
            resp.State.SetAttribute(ctx, path.Root("last_updated"), time.Now().Format(time.RFC850))
            return
        }
    }
    resp.Diagnostics.AddError(
        "Error importing container",
        "Could not find container with name: "+id,
    )
}
