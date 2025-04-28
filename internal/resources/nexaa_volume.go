package resources

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/Tilaa/tilaa-cli/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
    _ resource.Resource = &volumeResource{}
)

// NewVolumeResource is a helper function to simplify the provider implementation.
func NewVolumeResource() resource.Resource {
    return &volumeResource{}
}

// volumeResource is the resource implementation.
type volumeResource struct{
    ID              types.String        `tfsdk:"id"`
    NamespaceName   types.String        `tfsdk:"namespace_name"`
    Name            types.String        `tfsdk:"name"`
    Size            types.Int64         `tfsdk:"size"`
    Usage           types.Int64         `tfsdk:"usage"`
    LastUpdated     types.String        `tfsdk:"last_updated"`
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
                Computed: true,
            },
            "namespace_name": schema.StringAttribute{
                Required: true,
            },
            "name": schema.StringAttribute{
                Required: true,
            },
            "size": schema.Int64Attribute{
                Required: true,
            },
            "usage": schema.Int64Attribute{
                Computed: true,
            },
            "last_updated": schema.StringAttribute{
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
    if resp.Diagnostics.HasError(){
        return
    }

    input := api.VolumeInput {
        Namespace: plan.NamespaceName.ValueString(),
        Name: plan.Name.ValueString(),
        Size : int(plan.Size.ValueInt64()),
    }

    volume, err := api.CreateVolume(input)

    if err != nil {
        resp.Diagnostics.AddError(
            "Error creating volume",
            "Could not create volume, error: "+err.Error(),
        )
        return
    }

    plan.ID = types.StringValue(volume.Id)
    plan.NamespaceName = types.StringValue(volume.Namespace)
    plan.Name = types.StringValue(volume.Name)
    plan.Size = types.Int64Value(int64(volume.Size))
    plan.Usage = types.Int64Value(int64(volume.Usage))
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
    if resp.Diagnostics.HasError(){
        return
    }

    volume, err := api.ListVolumeById(state.NamespaceName.ValueString(), state.ID.ValueString())
    if err != nil {
        resp.Diagnostics.AddError(
            "Error Reading Volume",
            "Could not read volume "+state.ID.ValueString()+": "+err.Error(),
        )
        return
    }

    state.NamespaceName = types.StringValue(volume.Namespace)
    state.Name = types.StringValue(volume.Name)
    state.Size = types.Int64Value(int64(volume.Size))
    state.Usage = types.Int64Value(int64(volume.Usage))

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
    if resp.Diagnostics.HasError(){
        return
    }

    input := api.VolumeInput {
        Namespace: plan.NamespaceName.ValueString(),
        Name: plan.Name.ValueString(),
        Size : int(plan.Size.ValueInt64()),
    }    

    _, err := api.IncreaseVolume(input)
    if err != nil {
        resp.Diagnostics.AddError(
            "Error Updating Volume",
            "Could not update volume, error: "+err.Error(),
        )
        return
    }

    volume, err := api.ListVolumeById(plan.NamespaceName.ValueString(), plan.ID.ValueString())
    if err != nil {
        resp.Diagnostics.AddError(
            "Error Reading Volume",
            "Could not read volume "+plan.ID.ValueString()+": "+err.Error(),
        )
        return
    }
    
    plan.NamespaceName = types.StringValue(volume.Namespace)
    plan.Name = types.StringValue(volume.Name)
    plan.Size = types.Int64Value(int64(volume.Size))
    plan.Usage = types.Int64Value(int64(volume.Usage))
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
    if resp.Diagnostics.HasError(){
        return
    }

    err := api.DeleteVolume(state.Name.ValueString(), state.NamespaceName.ValueString())
    if err != nil {
        resp.Diagnostics.AddError(
            "Error Deleting Volume",
            "Could not delete volume, error: "+err.Error(),
        )
        return
    }
}
