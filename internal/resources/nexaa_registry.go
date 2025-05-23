// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/tilaa/tilaa-cli/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
    _ resource.Resource = &registryResource{}
)

// NewRegistryResource is a helper function to simplify the provider implementation.
func NewRegistryResource() resource.Resource {
    return &registryResource{}
}

// registryResource is the resource implementation.
type registryResource struct{
	ID			    types.String 	`tfsdk:"id"`
	Namespace       types.String 	`tfsdk:"namespace"`
	Name 		    types.String	`tfsdk:"name"`
	Source		    types.String	`tfsdk:"source"`
	Username	    types.String	`tfsdk:"username"`
	Password	    types.String	`tfsdk:"password"`
	Verify		    types.Bool		`tfsdk:"verify"` 
    Locked          types.Bool      `tfsdk:"locked"`
    LastUpdated     types.String    `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *registryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_registry"
}

// Schema defines the schema for the resource.
func (r *registryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id" : schema.StringAttribute{
                Description: "Identifier of the private registry, equal to the name of the registry",
                Computed: true,
            },
			"namespace": schema.StringAttribute{
                Description: "Name of the namespace the private registry belongs to",
				Required: true,
			},
            "name": schema.StringAttribute{
                Description: "The name given to the private registry",
                Required: true,
            },
            "source": schema.StringAttribute{
                Description: "The URL of the site where the credentials are used",
				Required: true,
			},
			"username": schema.StringAttribute{
                Description: "The username used to connect to the source",
				Required: true,
			},
			"password": schema.StringAttribute{
                Description: "The password used to connect to the source",
                Sensitive: true,
				Required: true,
			},
			"verify": schema.BoolAttribute{
                Description: "If true(default) the connection will be tested immediately to check if the credentials are true",
				Optional: true,
                Computed: true,
			},
            "locked": schema.BoolAttribute{
                Description: "If the registry is locked it can't be deleted",
                Computed: true,
            },
            "last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update of the private registry",
                Computed: true,
            },
        },
    }
}

// Create creates the resource and sets the initial Terraform state.
func (r *registryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan registryResource
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError(){
        return
    }
    
	input := api.RegistryInput {
        Namespace: plan.Namespace.ValueString(),
		Name: plan.Name.ValueString(),
		Source: plan.Source.ValueString(),
		Username: plan.Username.ValueString(),
		Password: plan.Password.ValueString(),
		Verify: plan.Verify.ValueBool(),
    } 

    const (
        maxRetries = 4
        initialDelay = 3 * time.Second
    )
    delay := initialDelay
    var err error

    for i:=0; i<=maxRetries; i++ {
        _, err = api.CreateRegistry(input)
        if err == nil {
            break
        }

        time.Sleep(delay)
        delay *= 2
    }

    if err != nil {
        resp.Diagnostics.AddError(
            "Error creating registry",
            "Could not create registry, error: "+err.Error(),
        )
        return
    }

    registry, err := api.ListRegistryByName(plan.Namespace.ValueString(), plan.Name.ValueString())
    if err != nil {
        resp.Diagnostics.AddError(
            "Error reading registry",
            "Could not read registry, error: "+err.Error(),
        )
        return
    }

    plan.ID = types.StringValue(registry.Name)
    plan.Namespace = types.StringValue(registry.Namespace)
    plan.Name = types.StringValue(registry.Name)
    plan.Source = types.StringValue(registry.Source)
    plan.Username = types.StringValue(registry.Username)
    plan.Password = types.StringValue(input.Password)
    plan.Verify = types.BoolValue(input.Verify)
    plan.Locked = types.BoolValue(registry.Locked)
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError(){
        return
    }
}

// Read refreshes the Terraform state with the latest data.
func (r *registryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state registryResource
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    registry, err := api.ListRegistryByName(state.Namespace.ValueString(), state.Name.ValueString())
    if err != nil {
        resp.Diagnostics.AddError(
            "Error Reading Registry",
            "Could not read registry with name "+state.Name.ValueString()+", error: "+err.Error(),
        )
        return
    }

    state.ID = types.StringValue(registry.Name)
    state.Namespace = types.StringValue(registry.Namespace)
    state.Name = types.StringValue(registry.Name)
    state.Source = types.StringValue(registry.Source)
    state.Username = types.StringValue(registry.Username)

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError(){
        return
    }
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *registryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan registryResource
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    
    resp.Diagnostics.AddError(
        "You can't update a registry",
        "You can't change a registry. You can only create and delete a registry",
    )

    if resp.Diagnostics.HasError() {
        return
    }    
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *registryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state registryResource
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

        const (
        maxRetries   = 4
        initialDelay = 5 * time.Second
    )
    delay := initialDelay

    // Retry DeleteVolume while “locked” errors persist
    for i := 0; i <= maxRetries; i++ {
        err := api.DeleteRegistry(
            state.Namespace.ValueString(),
            state.Name.ValueString(),
        )
        if err == nil {
            // Successfully deleted
            return
        }
        msg := err.Error()
        switch {
        case strings.Contains(msg, "locked"):
            // Service still cleaning up—wait & back off
            time.Sleep(delay)
            delay *= 2
            continue
        case strings.Contains(msg, "Not found"):
            // Not found error
            resp.Diagnostics.AddWarning(
                "Registry not found",
                "The given registry name is incorrect. Or the registry is already deleted.",
            )
            return
		case strings.Contains(msg, "Namespace"):
			//Namespace doesn't exist
			resp.Diagnostics.AddWarning(
				"Namespace not found",
				"The namespace of the registry is already deleted or the given name is incorrect.",
			)
        default:
            // Any other error is fatal
            resp.Diagnostics.AddError(
                "Error deleting registry",
                "Could not delete registry "+state.Name.ValueString()+": "+msg,
            )
            return
        }
    }

    // If we reach here, we exhausted retries with only “locked” errors
    resp.Diagnostics.AddError(
        "Timeout waiting for registry to become deletable",
        "Could not delete registry after a couple of retries, try again later.",
    )
}
