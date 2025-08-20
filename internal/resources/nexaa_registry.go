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
	"github.com/nexaa-cloud/nexaa-cli/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &registryResource{}
	_ resource.ResourceWithImportState = &registryResource{}
)

// NewRegistryResource is a helper function to simplify the provider implementation.
func NewRegistryResource() resource.Resource {
	return &registryResource{}
}

// registryResource is the resource implementation.
type registryResource struct {
	ID          types.String `tfsdk:"id"`
	Namespace   types.String `tfsdk:"namespace"`
	Name        types.String `tfsdk:"name"`
	Source      types.String `tfsdk:"source"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	Verify      types.Bool   `tfsdk:"verify"`
	Locked      types.Bool   `tfsdk:"locked"`
	Status      types.String `tfsdk:"status"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *registryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registry"
}

// Schema defines the schema for the resource.
func (r *registryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the private registry, equal to the name of the registry",
				Computed:    true,
			},
			"namespace": schema.StringAttribute{
				Description: "Name of the namespace the private registry belongs to",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name given to the private registry",
				Required:    true,
			},
			"source": schema.StringAttribute{
				Description: "The URL of the site where the credentials are used",
				Required:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username used to connect to the source",
				Required:    true,
			},
			"password": schema.StringAttribute{
				Description: "The password used to connect to the source. Is required.",
				Sensitive:   true,
				Optional:    true,
			},
			"verify": schema.BoolAttribute{
				Description: "If true(default) the connection will be tested immediately to check if the credentials are true",
				Optional:    true,
				Computed:    true,
			},
			"locked": schema.BoolAttribute{
				Description: "If the registry is locked it can't be deleted",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of the registry",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the private registry",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *registryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan registryResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Password.IsNull() || plan.Password.IsUnknown() {
		resp.Diagnostics.AddError(
			"Missing password",
			"Password is required to connect to a private registry.",
		)
		return
	}

	input := api.RegistryCreateInput{
		Namespace: plan.Namespace.ValueString(),
		Name:      plan.Name.ValueString(),
		Source:    plan.Source.ValueString(),
		Username:  plan.Username.ValueString(),
		Password:  plan.Password.ValueString(),
		Verify:    plan.Verify.ValueBool(),
	}

	client := api.NewClient()

	const (
		maxRetries   = 4
		initialDelay = 3 * time.Second
	)
	delay := initialDelay
	var err error
	var registry api.RegistryResult

	for i := 0; i <= maxRetries; i++ {
		registry, err = client.RegistryCreate(input)
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

	plan.ID = types.StringValue(registry.Name)
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(registry.Name)
	plan.Source = types.StringValue(registry.Source)
	plan.Username = types.StringValue(registry.Username)
	plan.Password = types.StringValue(input.Password)
	plan.Verify = types.BoolValue(input.Verify)
	plan.Locked = types.BoolValue(registry.Locked)
	plan.Status = types.StringValue(registry.State)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
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

	client := api.NewClient()

	registry, err := client.ListRegistryByName(state.Namespace.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Registry",
			"Could not read registry with name "+state.Name.ValueString()+", error: "+err.Error(),
		)
		return
	}

	state.ID = types.StringValue(registry.Name)
	state.Namespace = types.StringValue(state.Namespace.ValueString())
	state.Name = types.StringValue(registry.Name)
	state.Source = types.StringValue(registry.Source)
	state.Username = types.StringValue(registry.Username)
	state.Status = types.StringValue(registry.State)
	state.Locked = types.BoolValue(registry.Locked)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
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
		maxRetries   = 10
		initialDelay = 2 * time.Second
	)
	delay := initialDelay

	client := api.NewClient()

	var lastErr error

	// Retry DeleteRegistry with context timeout
	for i := 0; i <= maxRetries; i++ {
		// Check context timeout
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError(
				"Timeout deleting registry",
				fmt.Sprintf("Context timeout while waiting to delete registry %q", state.Name.ValueString()),
			)
			return
		default:
		}

		registry, err := client.ListRegistryByName(state.Namespace.ValueString(), state.Name.ValueString())
		if err != nil {
			lastErr = err
			resp.Diagnostics.AddError(
				"Error fetching registry",
				fmt.Sprintf("Could not find registry with name %q: %s", state.Name.ValueString(), err.Error()),
			)
			return
		}

		if registry.State == "created" {
			_, err := client.RegistryDelete(state.Namespace.ValueString(), state.Name.ValueString())

			if err != nil {
				lastErr = err
				resp.Diagnostics.AddError(
					"Error deleting registry",
					fmt.Sprintf("Failed to delete registry %q: %s", state.Name.ValueString(), err.Error()),
				)
				return
			}
			return
		}
		if registry.State == "failed" && registry.Locked {
			resp.Diagnostics.AddError(
				"Error deleting registry",
				fmt.Sprintf("Failed to delete registry %q, the registry is locked and could not be deleted", state.Name.ValueString()),
			)
			return
		}
		
		// Sleep with context timeout
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError(
				"Timeout deleting registry",
				fmt.Sprintf("Context timeout while waiting to delete registry %q", state.Name.ValueString()),
			)
			return
		case <-time.After(delay):
		}
		delay *= 2
	}

	if lastErr != nil {
		resp.Diagnostics.AddError(
			"Timeout waiting for registry to unlock",
			fmt.Sprintf("Registry could not be deleted after retries. Last error: %s", lastErr.Error()),
		)
	} else {
		resp.Diagnostics.AddError(
			"Timeout waiting for registry to unlock",
			"Registry could not be deleted after retries, and no specific error was returned.",
		)
	}
}

// ImportState implements resource.ResourceWithImportState.
func (r *registryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expect import ID as "namespace/registryName"
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format \"<namespace>/<registry_name>\", got: "+req.ID,
		)
		return
	}
	ns := parts[0]
	registryName := parts[1]

	client := api.NewClient()

	// Fetch the registry using the namespace and registry name
	registry, err := client.ListRegistryByName(ns, registryName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Registry",
			"Could not read registry "+registryName+": "+err.Error(),
		)
		return
	}

	// Set the registry attributes in the state
	resp.State.SetAttribute(ctx, path.Root("id"), registry.Name)
	resp.State.SetAttribute(ctx, path.Root("namespace"), ns)
	resp.State.SetAttribute(ctx, path.Root("name"), registry.Name)
	resp.State.SetAttribute(ctx, path.Root("source"), registry.Source)
	resp.State.SetAttribute(ctx, path.Root("username"), registry.Username)
	resp.State.SetAttribute(ctx, path.Root("locked"), registry.Locked)
	resp.State.SetAttribute(ctx, path.Root("status"), registry.State)
	resp.State.SetAttribute(ctx, path.Root("last_updated"), time.Now().Format(time.RFC850))
}
