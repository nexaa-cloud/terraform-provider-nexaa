package resources

import (
	"context"
	"time"

	//"strconv"
	//"time"

	//"fmt"

	//"gitlab.com/Tilaa/tilaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/Tilaa/tilaa-cli/api"
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
	ID			types.String 	`tfsdk:"id"`
	Namespace 	types.String 	`tfsdk:"namespace"`
	Name 		types.String	`tfsdk:"name"`
	Source		types.String	`tfsdk:"source"`
	Username	types.String	`tfsdk:"username"`
	Password	types.String	`tfsdk:"password"`
	Verify		types.Bool		`tfsdk:"verify"` 
    LastUpdated types.String    `tfsdk:"last_updated"`
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
                Computed: true,
            },
			"namespace": schema.StringAttribute{
				Required: true,
			},
            "name": schema.StringAttribute{
                Required: true,
            },
            "source": schema.StringAttribute{
				Required: true,
			},
			"username": schema.StringAttribute{
				Required: true,
			},
			"password": schema.StringAttribute{
				Required: true,
			},
			"verify": schema.BoolAttribute{
				Optional: true,
			},
            "last_updated": schema.StringAttribute{
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
        //Namespace: plan.Namespace.ValueString(),
		Name: plan.Name.ValueString(),
		Source: plan.Source.ValueString(),
		Username: plan.Username.ValueString(),
		Password: plan.Password.ValueString(),
		Verify: plan.Verify.ValueBool(),
    } 

	registry, err := api.CreateRegistry(input)
    if err != nil {
        resp.Diagnostics.AddError(
            "Error creating registry",
            "Could not create registry, error: "+err.Error(),
        )
        return
    }

    plan.ID = types.StringValue(registry.Id)
    plan.Namespace = types.StringValue(registry.Namespace)
    plan.Name = types.StringValue(registry.Name)
    plan.Source = types.StringValue(registry.Source)
    plan.Username = types.StringValue(registry.Username)
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

    state.ID = types.StringValue(registry.Id)
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
    if resp.Diagnostics.HasError() {
        return
    }

    resp.Diagnostics.AddError(
        "You can't update a registry",
        "",
    )
    return
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *registryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state registryResource
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    resp.Diagnostics.AddError(
        "Delete registry not implemented yet",
        "",
    )
    return
}
