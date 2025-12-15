// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"

	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/nexaa-cloud/nexaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &starterContainerResource{}
	_ resource.ResourceWithImportState = &starterContainerResource{}
	_ resource.ResourceWithIdentity    = &starterContainerResource{}
)

// NewStarterContainerResource is a helper function to simplify the provider implementation.
func NewStarterContainerResource() resource.Resource {
	return &starterContainerResource{}
}

// starterContainerResource is the resource implementation for starter containers.
type starterContainerResource struct {
	ID                   types.String   `tfsdk:"id"`
	Name                 types.String   `tfsdk:"name"`
	Namespace            types.String   `tfsdk:"namespace"`
	Image                types.String   `tfsdk:"image"`
	Registry             types.String   `tfsdk:"registry"`
	Command              types.List     `tfsdk:"command"`
	Entrypoint           types.List     `tfsdk:"entrypoint"`
	EnvironmentVariables types.Set      `tfsdk:"environment_variables"`
	Ports                types.List     `tfsdk:"ports"`
	Ingresses            types.List     `tfsdk:"ingresses"`
	Mounts               types.List     `tfsdk:"mounts"`
	HealthCheck          types.Object   `tfsdk:"health_check"`
	LastUpdated          types.String   `tfsdk:"last_updated"`
	Status               types.String   `tfsdk:"status"`
	Timeouts             timeouts.Value `tfsdk:"timeouts"`
}

// Metadata returns the resource type name.
func (r *starterContainerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_starter_container"
}

func (r *starterContainerResource) IdentitySchema(ctx context.Context, request resource.IdentitySchemaRequest, response *resource.IdentitySchemaResponse) {
	response.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"name": identityschema.StringAttribute{
				Description:       "The name of the container.",
				RequiredForImport: true,
			},
			"namespace": identityschema.StringAttribute{
				Description:       "The namespace where the container belongs to.",
				RequiredForImport: true,
			},
		},
	}
}

func (r *starterContainerResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Starter container resource representing a starter container that will be deployed on nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the container, equal to the name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the container",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"namespace": schema.StringAttribute{
				Required:    true,
				Description: "Name of the namespace that the container will belong to",
			},
			"image": schema.StringAttribute{
				Required:    true,
				Description: "The image use to run the container",
			},
			"registry": schema.StringAttribute{
				Optional:    true,
				Description: "The registry used to be able to acces images that are saved in a private environment, fill in null to use a public registry",
			},
			"command": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "Command to run. When the field is omitted, the default command of the image will be used. The command will be passed to the entrypoint as arguments. Environment variables can be used in the command by using the syntax $(ENVIRONMENT_VARIABLE).",
			},
			"entrypoint": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "Entrypoint of the container. This field will overwrite the default entrypoint of the image. When the field is omitted, the default entrypoint of the image will be used. Entry point is the first command executed when the container starts. It will receive the command as arguments.",
			},
			"ports": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "The ports used to expose for traffic, format as from:to",
			},
			"environment_variables": schema.SetNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The name used for the environment variable",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The value used for the environment variable, is required",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"secret": schema.BoolAttribute{
							Optional:    true,
							Description: "A boolean to represent if the environment variable is a secret or not",
						},
					},
				},
				Optional:    true,
				Computed:    true,
				Description: "Environment variables used in the container; order is not significant and matched by name",
			},
			"ingresses": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"domain_name": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The domain used for the ingress, defaults to https://{tenant}-{namespaceName}-{containerName}.container.tilaa.cloud",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"port": schema.Int64Attribute{
							Required:    true,
							Description: "The port used for the ingress, must be one of the exposed ports",
						},
						"tls": schema.BoolAttribute{
							Required:    true,
							Description: "Boolean representing if you want TLS enabled or not",
						},
						"allow_list": schema.ListAttribute{
							ElementType: types.StringType,
							Optional:    true,
							Computed:    true,
							Description: "A list with the IP's that can access the ingress url, 0.0.0.0/0 to make it accessible for everyone",
						},
					},
				},
				Optional:    true,
				Computed:    true,
				Description: "Used to access the container from the internet",
			},
			"mounts": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Required:    true,
							Description: "The path to the location where the data will be saved",
						},
						"volume": schema.StringAttribute{
							Required:    true,
							Description: "The name of the volume that is used for the mount",
						},
					},
				},
				Computed:    true,
				Optional:    true,
				Description: "Used to add persistent storage to your container",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"health_check": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"port": schema.Int64Attribute{
						Required: true,
					},
					"path": schema.StringAttribute{
						Required: true,
					},
				},
				Optional: true,
			},
			"status": schema.StringAttribute{
				Description: "The status of the container",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the starter container",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *starterContainerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan starterContainerResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 30*time.Second)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	if plan.Registry.IsNull() || plan.Registry.IsUnknown() {
		plan.Registry = types.StringNull()
	}

	// Build input struct for starter container (default resources and no scaling)
	input := api.ContainerCreateInput{
		Namespace: plan.Namespace.ValueString(),
		Name:      plan.Name.ValueString(),
		Image:     plan.Image.ValueString(),
		Registry:  plan.Registry.ValueStringPointer(),
		Type:      api.ContainerTypeStarter,
		Resources: api.ContainerResourcesCpu250Ram500,
	}

	// Command - only set if provided
	command, shouldUpdateCmd, diags := buildCommandUpdateInput(ctx, plan.Command)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if shouldUpdateCmd {
		input.Command = command
	}

	// Entrypoint - only set if provided
	entrypoint, shouldUpdateEp, diags := buildEntrypointUpdateInput(ctx, plan.Entrypoint)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if shouldUpdateEp {
		input.Entrypoint = entrypoint
	}

	// Use common functions to build input
	ports, diags := buildPortsInput(ctx, plan.Ports)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input.Ports = ports

	mounts, diags := buildMountsInput(ctx, plan.Mounts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input.Mounts = mounts

	ingresses, diags := buildIngressesInput(ctx, plan.Ingresses)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input.Ingresses = ingresses

	// Environment variables (build API input from plan)
	inputs, dEnv := extractEnvInputsFromSet(ctx, plan.EnvironmentVariables)
	resp.Diagnostics.Append(dEnv...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(inputs) > 0 {
		input.EnvironmentVariables = inputs
	}

	// Health check
	healthCheck, diags := buildHealthCheckInput(ctx, plan.HealthCheck)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input.HealthCheck = healthCheck

	// Create containerResult
	client := api.NewClient()

	// Log the request for debugging
	tflog.Debug(ctx, "Creating starter container", map[string]interface{}{
		"namespace":    input.Namespace,
		"name":         input.Name,
		"image":        input.Image,
		"registry":     input.Registry,
		"type":         input.Type,
		"resources":    input.Resources,
		"ports":        input.Ports,
		"mounts":       input.Mounts,
		"ingresses":    input.Ingresses,
		"env_vars":     input.EnvironmentVariables,
		"health_check": input.HealthCheck,
	})

	containerResult, err := client.ContainerCreate(input)
	if err != nil {
		tflog.Error(ctx, "Failed to create starter container", map[string]interface{}{
			"error": err.Error(),
			"input": input,
		})
		resp.Diagnostics.AddError("Error creating starter container", "Could not create starter container: "+err.Error())
		return
	}

	tflog.Debug(ctx, "Successfully created starter container", map[string]interface{}{
		"container_name":  containerResult.Name,
		"container_state": containerResult.State,
	})

	// Set all fields in plan from returned containerResult
	plan.ID = types.StringValue(containerResult.Name)
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(containerResult.Name)
	plan.Image = types.StringValue(containerResult.Image)
	plan.Status = types.StringValue(containerResult.State)

	plan.Registry = processRegistryName(containerResult)

	// Command
	plan.Command, diags = buildCommandState(containerResult.Command)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Entrypoint
	plan.Entrypoint, diags = buildEntrypointState(containerResult.Entrypoint)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Environment variables (state population)
	if containerResult.EnvironmentVariables != nil {
		setVal, d := buildEnvSetFromAPI(ctx, containerResult.EnvironmentVariables, input.EnvironmentVariables, types.SetNull(envVarObjectType()), secretUseProvided)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.EnvironmentVariables = setVal
	}

	// Use common functions for response processing
	portList, diags := buildPortsState(containerResult)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Ports = portList

	// Mounts
	if containerResult.Mounts != nil {
		mountList, d := buildMountsFromApi(containerResult.Mounts)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Mounts = mountList
	}

	// Ingresses
	ingressesList, d := buildIngressesFromApi(containerResult)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Ingresses = ingressesList

	// Health check
	plan.HealthCheck = buildHealthCheckState(containerResult)

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set identity data (required for ResourceWithIdentity)
	identity := struct {
		Name      types.String `tfsdk:"name"`
		Namespace types.String `tfsdk:"namespace"`
	}{
		Name:      plan.Name,
		Namespace: plan.Namespace,
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *starterContainerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state starterContainerResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch the created container
	client := api.NewClient()
	container, err := client.ListContainerByName(state.Namespace.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading starter container", "Could not find starter container: "+err.Error())
		return
	}

	// Set all fields in state from returned container
	state.ID = types.StringValue(container.Name)
	state.Namespace = types.StringValue(state.Namespace.ValueString())
	state.Name = types.StringValue(container.Name)
	state.Image = types.StringValue(container.Image)

	state.Registry = processRegistryName(container)

	// Command
	state.Command, diags = buildCommandState(container.Command)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Entrypoint
	state.Entrypoint, diags = buildEntrypointState(container.Entrypoint)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Environment variables (refresh state)
	if container.EnvironmentVariables != nil {
		setVal, d := buildEnvSetFromAPI(ctx, container.EnvironmentVariables, nil, state.EnvironmentVariables, secretPreservePrev)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.EnvironmentVariables = setVal
	}

	// Use common functions for state processing
	portList, diags := buildPortsState(container)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Ports = portList

	// Mounts
	if container.Mounts != nil {
		mountList, d := buildMountsFromApi(container.Mounts)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Mounts = mountList
	}

	// Ingresses
	ingressesList, diags := buildIngressesFromApi(container)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Ingresses = ingressesList

	// Health check
	state.HealthCheck = buildHealthCheckState(container)

	state.Status = types.StringValue(container.State)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set identity data (required for ResourceWithIdentity)
	identity := struct {
		Name      types.String `tfsdk:"name"`
		Namespace types.String `tfsdk:"namespace"`
	}{
		Name:      state.Name,
		Namespace: state.Namespace,
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *starterContainerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan starterContainerResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Registry.IsNull() || plan.Registry.IsUnknown() {
		plan.Registry = types.StringNull()
	}

	// Build input struct for starter container (no resources or scaling)
	input := api.ContainerModifyInput{
		Namespace: plan.Namespace.ValueString(),
		Name:      plan.Name.ValueString(),
		Image:     plan.Image.ValueStringPointer(),
		Registry:  plan.Registry.ValueStringPointer(),
	}

	// Command - only set if provided (following CLI modify behavior)
	command, shouldUpdateCmd, diags := buildCommandUpdateInput(ctx, plan.Command)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if shouldUpdateCmd {
		input.Command = command
	}

	// Entrypoint - only set if provided (following CLI modify behavior)
	entrypoint, shouldUpdateEp, diags := buildEntrypointUpdateInput(ctx, plan.Entrypoint)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if shouldUpdateEp {
		input.Entrypoint = entrypoint
	}

	// Use common functions for input building
	ports, diags := buildPortsInput(ctx, plan.Ports)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input.Ports = ports

	// Get previous state for comparison
	var prev starterContainerResource
	if !req.State.Raw.IsNull() && req.State.Raw.IsKnown() {
		diags := req.State.Get(ctx, &prev)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Mounts
	mounts, diags := buildMountsUpdateInput(ctx, plan.Mounts, prev.Mounts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input.Mounts = mounts

	// Ingresses
	ingresses, diags := buildIngressesUpdateInput(ctx, plan.Ingresses, prev.Ingresses)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input.Ingresses = ingresses

	// Environment variables (build API input from plan)
	inputsUpd, dEnvU := extractEnvInputsFromSet(ctx, plan.EnvironmentVariables)
	resp.Diagnostics.Append(dEnvU...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(inputsUpd) > 0 {
		input.EnvironmentVariables = inputsUpd
	}

	// Health check
	healthCheck, diags := buildHealthCheckInput(ctx, plan.HealthCheck)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input.HealthCheck = healthCheck

	createTimeout, diags := plan.Timeouts.Update(ctx, 30*time.Second)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	client := api.NewClient()
	err := waitForUnlocked(ctx, containerLocked(), *client, plan.Namespace.ValueString(), plan.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Error updating starter container", "Could not reach a running state: "+err.Error())
		return
	}

	// modify containerResult
	containerResult, err := client.ContainerModify(input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating starter container", "Could not update starter container: "+err.Error())
		return
	}

	err = waitForUnlocked(ctx, containerLocked(), *client, plan.Namespace.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating starter container", "Could not reach a running state: "+err.Error())
		return
	}

	containerResult, err = client.ListContainerByName(plan.Namespace.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating starter container", "Could not update starter container: "+err.Error())
		return
	}

	// Set all fields in plan from returned containerResult
	plan.ID = types.StringValue(containerResult.Name)
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(containerResult.Name)
	plan.Image = types.StringValue(containerResult.Image)

	plan.Registry = processRegistryName(containerResult)

	// Command
	plan.Command, diags = buildCommandState(containerResult.Command)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Entrypoint
	plan.Entrypoint, diags = buildEntrypointState(containerResult.Entrypoint)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Environment variables (update state)
	if containerResult.EnvironmentVariables != nil {
		setVal, d := buildEnvSetFromAPI(ctx, containerResult.EnvironmentVariables, input.EnvironmentVariables, plan.EnvironmentVariables, secretUseProvided)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.EnvironmentVariables = setVal
	}

	// Use common functions for response processing
	portList, diags := buildPortsState(containerResult)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Ports = portList

	// Mounts
	if containerResult.Mounts != nil {
		mountList, d := buildMountsFromApi(containerResult.Mounts)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Mounts = mountList
	}

	ingressesList, d := buildIngressesFromApi(containerResult)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Ingresses = ingressesList

	// Health check
	plan.HealthCheck = buildHealthCheckState(containerResult)

	plan.Status = types.StringValue(containerResult.State)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set identity data (required for ResourceWithIdentity)
	identity := struct {
		Name      types.String `tfsdk:"name"`
		Namespace types.String `tfsdk:"namespace"`
	}{
		Name:      plan.Name,
		Namespace: plan.Namespace,
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *starterContainerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan starterContainerResource
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	deleteTimeout, diags := plan.Timeouts.Delete(ctx, 2*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	err := waitForUnlocked(ctx, containerLocked(), *client, plan.Namespace.ValueString(), plan.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Error deleting starter container", "Could not reach a running state: "+err.Error())
		return
	}

	_, err = client.ContainerDelete(plan.Namespace.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting starter container",
			fmt.Sprintf("Failed to delete starter container %q: %s", plan.Name.ValueString(), err.Error()),
		)
		return
	}
}

func (r *starterContainerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	namespace, name, err := parseContainerImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	// Fetch the container from your API
	client := api.NewClient()
	container, err := client.ListContainerByName(namespace, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing starter container",
			fmt.Sprintf("Unable to fetch starter container %q in namespace %q: %s", name, namespace, err.Error()),
		)
		return
	}

	// Use common function to build import state
	stateAttrs, diags := buildContainerImportState(ctx, container, namespace, name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Add timeout values
	stateAttrs["timeouts"] = timeouts.Value{
		Object: types.ObjectValueMust(
			map[string]attr.Type{
				"create": types.StringType,
				"update": types.StringType,
				"delete": types.StringType,
			},
			map[string]attr.Value{
				"create": types.StringValue("30s"),
				"update": types.StringValue("30s"),
				"delete": types.StringValue("30s"),
			},
		),
	}

	// Build the final state object
	state := starterContainerResource{
		ID:                   stateAttrs["id"].(types.String),
		Name:                 stateAttrs["name"].(types.String),
		Namespace:            stateAttrs["namespace"].(types.String),
		Image:                stateAttrs["image"].(types.String),
		Registry:             stateAttrs["registry"].(types.String),
		Command:              stateAttrs["command"].(types.List),
		Entrypoint:           stateAttrs["entrypoint"].(types.List),
		EnvironmentVariables: stateAttrs["environment_variables"].(types.Set),
		Ports:                stateAttrs["ports"].(types.List),
		Ingresses:            stateAttrs["ingresses"].(types.List),
		Mounts:               stateAttrs["mounts"].(types.List),
		HealthCheck:          stateAttrs["health_check"].(types.Object),
		Status:               stateAttrs["status"].(types.String),
		LastUpdated:          stateAttrs["last_updated"].(types.String),
		Timeouts:             stateAttrs["timeouts"].(timeouts.Value),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
