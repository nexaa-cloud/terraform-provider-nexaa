// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/nexaa-cloud/nexaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nexaa-cloud/terraform-provider-nexaa/internal/enums"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &containerJobResource{}
	_ resource.ResourceWithImportState = &containerJobResource{}
)

// NewContainerJobResource is a helper function to simplify the provider implementation.
func NewContainerJobResource() resource.Resource {
	return &containerJobResource{}
}

// containerJobResource is the resource implementation.
type containerJobResource struct {
	ID                   types.String   `tfsdk:"id"`
	Name                 types.String   `tfsdk:"name"`
	Namespace            types.String   `tfsdk:"namespace"`
	Image                types.String   `tfsdk:"image"`
	Registry             types.String   `tfsdk:"registry"`
	Resources            types.Object   `tfsdk:"resources"`
	EnvironmentVariables types.Set      `tfsdk:"environment_variables"`
	Command              types.List     `tfsdk:"command"`
	Entrypoint           types.List     `tfsdk:"entrypoint"`
	Mounts               types.List     `tfsdk:"mounts"`
	Schedule             types.String   `tfsdk:"schedule"`
	Enabled              types.Bool     `tfsdk:"enabled"`
	LastUpdated          types.String   `tfsdk:"last_updated"`
	State                types.String   `tfsdk:"state"`
	Timeouts             timeouts.Value `tfsdk:"timeouts"`
}

// Metadata returns the resource type name.
func (r *containerJobResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_job"
}

func (r *containerJobResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Container job resource representing a scheduled container job that will be deployed on nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the container, equal to the name",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the container job",
			},
			"namespace": schema.StringAttribute{
				Required:    true,
				Description: "Name of the namespace that the container job will belong to",
			},
			"image": schema.StringAttribute{
				Required:    true,
				Description: "The image used to run the container job",
			},
			"registry": schema.StringAttribute{
				Optional:    true,
				Description: "The registry used to access images that are saved in a private environment, leave empty to use a public registry",
			},
			"resources": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"cpu": schema.Float64Attribute{
						Required:    true,
						Description: "The amount of cpu used for the container job, can be the following values: 0.25, 0.5, 0.75, 1, 2, 3, 4",
						Validators: []validator.Float64{
							float64validator.OneOf(enums.CPU...),
						},
					},
					"ram": schema.Float64Attribute{
						Required:    true,
						Description: "The amount of ram used for the container job (in GB), can be the following values: 0.5, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16",
						Validators: []validator.Float64{
							float64validator.OneOf(enums.RAM...),
						},
					},
				},
				Required:    true,
				Description: "The resources used for running the container job",
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
							Description: "The value used for the environment variable",
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
				Description: "Environment variables used in the container job; order is not significant and matched by name",
			},
			"command": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "Command to run. This is the command executed at the given schedule. When omitted, the default command of the image will be used.",
			},
			"entrypoint": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "Entrypoint of the container. This field will overwrite the default entrypoint of the image. When omitted, the default entrypoint of the image will be used.",
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
				Description: "Used to add persistent storage to your container job",
			},
			"schedule": schema.StringAttribute{
				Required:    true,
				Description: "Cron notation to schedule jobs. Format is equal to regular cron notation. For example, to run a job every day at 4am, use `0 4 * * *`. You can use https://crontab.guru/ to help you build your cron expressions.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Enable or disable the job. By disabling a job, it will not be executed, but the configuration is kept.",
			},
			"state": schema.StringAttribute{
				Description: "The state of the container job",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the container job",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Update: true,
				Delete: true,
			}),
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *containerJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan containerJobResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Construct CPU/RAM string
	var resources resourcesResource
	diags = plan.Resources.As(ctx, &resources, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return
	}

	cpu := int(resources.CPU.ValueFloat64() * 1000)
	ram := int(resources.RAM.ValueFloat64() * 1000)
	cpuRam := fmt.Sprintf("CPU_%d_RAM_%d", cpu, ram)

	if plan.Registry.IsNull() || plan.Registry.IsUnknown() {
		plan.Registry = types.StringNull()
	}

	// Build input struct
	input := api.ContainerJobCreateInput{
		Namespace: plan.Namespace.ValueString(),
		Name:      plan.Name.ValueString(),
		Image:     plan.Image.ValueString(),
		Registry:  plan.Registry.ValueStringPointer(),
		Resources: api.ContainerResources(cpuRam),
		Schedule:  plan.Schedule.ValueString(),
		Enabled:   plan.Enabled.ValueBool(),
	}

	// Command
	if !plan.Command.IsNull() && !plan.Command.IsUnknown() {
		var command []string
		diags = plan.Command.ElementsAs(ctx, &command, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Command = command
	} else {
		input.Command = make([]string, 0)
	}

	// Entrypoint
	if !plan.Entrypoint.IsNull() && !plan.Entrypoint.IsUnknown() {
		var entrypoint []string
		diags = plan.Entrypoint.ElementsAs(ctx, &entrypoint, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Entrypoint = entrypoint
	} else {
		input.Entrypoint = make([]string, 0)
	}

	// Mounts
	if !plan.Mounts.IsNull() && !plan.Mounts.IsUnknown() {
		var mounts []mountResource
		diags = plan.Mounts.ElementsAs(ctx, &mounts, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, m := range mounts {
			input.Mounts = append(input.Mounts, api.MountInput{
				Path: m.Path.ValueString(),
				Volume: api.MountVolumeInput{
					Name:       m.Volume.ValueString(),
					AutoCreate: false,
					Increase:   false,
					Size:       nil,
				},
				State: api.StatePresent,
			})
		}
	} else {
		input.Mounts = []api.MountInput{}
	}

	// Environment variables (build API input from plan)
	inputs, dEnv := extractEnvInputsFromSet(ctx, plan.EnvironmentVariables)
	resp.Diagnostics.Append(dEnv...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(inputs) > 0 {
		input.EnvironmentVariables = inputs
	}

	// Create container job
	client := api.NewClient()
	containerJobResult, err := client.ContainerJobCreate(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating container job", "Could not create container job: "+err.Error())
		return
	}

	// Set all fields in plan from returned container job result
	plan.ID = types.StringValue(containerJobResult.Name)
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(containerJobResult.Name)
	plan.Image = types.StringValue(containerJobResult.Image)
	plan.Schedule = types.StringValue(containerJobResult.Schedule)
	plan.Enabled = types.BoolValue(containerJobResult.Enabled)
	plan.State = types.StringValue(containerJobResult.State)

	if containerJobResult.PrivateRegistry == nil || containerJobResult.PrivateRegistry.Name == "public" {
		plan.Registry = types.StringNull()
	} else {
		plan.Registry = types.StringValue(*input.Registry)
	}

	// Parse CPU and RAM from the string
	resourcesObj, err := buildResourcesFromAPI(containerJobResult.Resources)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			err.Error(),
		)
		return
	}
	plan.Resources = resourcesObj

	// Environment variables (state population)
	if containerJobResult.EnvironmentVariables != nil {
		setVal, d := buildEnvSetFromAPI(ctx, containerJobResult.EnvironmentVariables, input.EnvironmentVariables, types.SetNull(envVarObjectType()), secretUseProvided)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.EnvironmentVariables = setVal
	}

	// Command
	if containerJobResult.Command != nil {
		commands := make([]attr.Value, len(containerJobResult.Command))
		for i, c := range containerJobResult.Command {
			commands[i] = types.StringValue(c)
		}

		commandList, diags := types.ListValue(types.StringType, commands)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Command = commandList
	}

	if containerJobResult.Entrypoint != nil {
		entrypoints := make([]attr.Value, len(containerJobResult.Entrypoint))
		for i, c := range containerJobResult.Entrypoint {
			entrypoints[i] = types.StringValue(c)
		}

		entrypointList, diags := types.ListValue(types.StringType, entrypoints)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Entrypoint = entrypointList
	}

	// Mounts
	if containerJobResult.Mounts != nil {
		mountList, d := buildMountsFromApi(containerJobResult.Mounts)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Mounts = mountList
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *containerJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state containerJobResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch the created container job
	client := api.NewClient()
	containerJob, err := client.ContainerJobByName(state.Namespace.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading container job", "Could not find container job: "+err.Error())
		return
	}

	// Set all fields in state from returned container job
	state.ID = types.StringValue(containerJob.Name)
	state.Namespace = types.StringValue(state.Namespace.ValueString())
	state.Name = types.StringValue(containerJob.Name)
	state.Image = types.StringValue(containerJob.Image)
	state.Schedule = types.StringValue(containerJob.Schedule)
	state.Enabled = types.BoolValue(containerJob.Enabled)
	state.State = types.StringValue(containerJob.State)

	if containerJob.PrivateRegistry == nil || containerJob.PrivateRegistry.Name == "public" {
		state.Registry = types.StringNull()
	} else {
		state.Registry = types.StringValue(containerJob.PrivateRegistry.Name)
	}

	// Parse CPU and RAM from the string
	resourcesObj, err := buildResourcesFromAPI(containerJob.Resources)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			err.Error(),
		)
		return
	}
	state.Resources = resourcesObj

	// Environment variables (refresh state)
	if containerJob.EnvironmentVariables != nil {
		setVal, d := buildEnvSetFromAPI(ctx, containerJob.EnvironmentVariables, nil, state.EnvironmentVariables, secretPreservePrev)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.EnvironmentVariables = setVal
	}

	// Command
	if containerJob.Command != nil {
		commands := make([]attr.Value, len(containerJob.Command))
		for i, c := range containerJob.Command {
			commands[i] = types.StringValue(c)
		}

		commandList, diags := types.ListValue(types.StringType, commands)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Command = commandList
	}

	if containerJob.Entrypoint != nil {
		entrypoints := make([]attr.Value, len(containerJob.Entrypoint))
		for i, c := range containerJob.Entrypoint {
			entrypoints[i] = types.StringValue(c)
		}

		entrypointList, diags := types.ListValue(types.StringType, entrypoints)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Entrypoint = entrypointList
	}

	// Mounts
	if containerJob.Mounts != nil {
		mountList, d := buildMountsFromApi(containerJob.Mounts)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Mounts = mountList
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *containerJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan containerJobResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Construct CPU/RAM string
	var resources resourcesResource
	diags = plan.Resources.As(ctx, &resources, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return
	}

	cpu := int(resources.CPU.ValueFloat64() * 1000)
	ram := int(resources.RAM.ValueFloat64() * 1000)
	cpuRam := fmt.Sprintf("CPU_%d_RAM_%d", cpu, ram)

	if plan.Registry.IsNull() || plan.Registry.IsUnknown() {
		plan.Registry = types.StringNull()
	}

	// Build input struct
	input := api.ContainerJobModifyInput{
		Namespace: plan.Namespace.ValueString(),
		Name:      plan.Name.ValueString(),
		Image:     plan.Image.ValueStringPointer(),
		Registry:  plan.Registry.ValueStringPointer(),
		Resources: (*api.ContainerResources)(&cpuRam),
		Schedule:  plan.Schedule.ValueStringPointer(),
		Enabled:   plan.Enabled.ValueBoolPointer(),
	}

	// Command
	if !plan.Command.IsNull() && !plan.Command.IsUnknown() {
		var command []string
		diags = plan.Command.ElementsAs(ctx, &command, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Command = command
	}

	// Entrypoint
	if !plan.Entrypoint.IsNull() && !plan.Entrypoint.IsUnknown() {
		var entrypoint []string
		diags = plan.Entrypoint.ElementsAs(ctx, &entrypoint, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Entrypoint = entrypoint
	}

	// Mounts
	var previousMounts []mountResource
	if !req.State.Raw.IsNull() && req.State.Raw.IsKnown() {
		var prev containerJobResource
		diags := req.State.Get(ctx, &prev)
		resp.Diagnostics.Append(diags...)
		if !prev.Mounts.IsNull() && !prev.Mounts.IsUnknown() {
			_ = prev.Mounts.ElementsAs(ctx, &previousMounts, false)
		}
	}

	input.Mounts = []api.MountInput{}
	plannedMounts := map[string]struct{}{}
	if !plan.Mounts.IsNull() && !plan.Mounts.IsUnknown() {
		var mounts []mountResource
		diags = plan.Mounts.ElementsAs(ctx, &mounts, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, m := range mounts {
			key := fmt.Sprintf("%s|%s", m.Path.ValueString(), m.Volume.ValueString())
			plannedMounts[key] = struct{}{}
			input.Mounts = append(input.Mounts, api.MountInput{
				Path: m.Path.ValueString(),
				Volume: api.MountVolumeInput{
					Name:       m.Volume.ValueString(),
					AutoCreate: false,
					Increase:   false,
					Size:       nil,
				},
				State: api.StatePresent,
			})
		}
	}
	for _, m := range previousMounts {
		key := fmt.Sprintf("%s|%s", m.Path.ValueString(), m.Volume.ValueString())
		if _, exists := plannedMounts[key]; !exists {
			input.Mounts = append(input.Mounts, api.MountInput{
				Path: m.Path.ValueString(),
				Volume: api.MountVolumeInput{
					Name:       m.Volume.ValueString(),
					AutoCreate: false,
					Increase:   false,
					Size:       nil,
				},
				State: api.StateAbsent,
			})
		}
	}

	// Environment variables (build API input from plan)
	inputsUpd, dEnvU := extractEnvInputsFromSet(ctx, plan.EnvironmentVariables)
	resp.Diagnostics.Append(dEnvU...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(inputsUpd) > 0 {
		input.EnvironmentVariables = inputsUpd
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, 30*time.Second)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	client := api.NewClient()
	_, err := waitForUnlocked(ctx, containerJobLocked(), *client, plan.Namespace.ValueString(), plan.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Error creating containerResult", "Could not reach a running state: "+err.Error())
	}

	containerJobResult, err := client.ContainerJobModify(input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating container job", "Could not update container job: "+err.Error())
		return
	}

	// Set all fields in plan from returned container job result
	plan.ID = types.StringValue(containerJobResult.Name)
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(containerJobResult.Name)
	plan.Image = types.StringValue(containerJobResult.Image)
	plan.Schedule = types.StringValue(containerJobResult.Schedule)
	plan.Enabled = types.BoolValue(containerJobResult.Enabled)
	plan.State = types.StringValue(containerJobResult.State)

	if containerJobResult.PrivateRegistry == nil {
		plan.Registry = types.StringNull()
	} else {
		plan.Registry = types.StringValue(containerJobResult.PrivateRegistry.Name)
	}

	// Parse CPU and RAM from the string
	resourcesObj, err := buildResourcesFromAPI(containerJobResult.Resources)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			err.Error(),
		)
		return
	}
	plan.Resources = resourcesObj

	// Environment variables (update state)
	if containerJobResult.EnvironmentVariables != nil {
		setVal, d := buildEnvSetFromAPI(ctx, containerJobResult.EnvironmentVariables, input.EnvironmentVariables, plan.EnvironmentVariables, secretUseProvided)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.EnvironmentVariables = setVal
	}

	// Command
	if containerJobResult.Command != nil {
		commands := make([]attr.Value, len(containerJobResult.Command))
		for i, c := range containerJobResult.Command {
			commands[i] = types.StringValue(c)
		}

		commandList, diags := types.ListValue(types.StringType, commands)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Command = commandList
	}

	// Command
	if containerJobResult.Entrypoint != nil {
		entrypoints := make([]attr.Value, len(containerJobResult.Entrypoint))
		for i, c := range containerJobResult.Entrypoint {
			entrypoints[i] = types.StringValue(c)
		}

		entrypointList, diags := types.ListValue(types.StringType, entrypoints)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Entrypoint = entrypointList
	}

	// Mounts
	if containerJobResult.Mounts != nil {
		mountList, d := buildMountsFromApi(containerJobResult.Mounts)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Mounts = mountList
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *containerJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan containerJobResource
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := plan.Timeouts.Delete(ctx, 30*time.Second)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	client := api.NewClient()
	_, err := waitForUnlocked(ctx, containerJobLocked(), *client, plan.Namespace.ValueString(), plan.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Error creating containerResult", "Could not reach a running state: "+err.Error())
	}

	_, err = client.ContainerJobDelete(plan.Namespace.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting container job",
			fmt.Sprintf("Failed to delete container job %q: %s", plan.Name.ValueString(), err.Error()),
		)
		return
	}
}

func (r *containerJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format \"<namespace>/<container_job_name>\", got: "+req.ID,
		)
		return
	}
	namespace := parts[0]
	name := parts[1]

	// Fetch the container job from your API
	client := api.NewClient()
	containerJob, err := client.ContainerJobByName(namespace, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing container job",
			fmt.Sprintf("Unable to fetch container job %q in namespace %q: %s", name, namespace, err.Error()),
		)
		return
	}

	// Parse resources
	resourcesObj, err := buildResourcesFromAPI(containerJob.Resources)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing container job",
			err.Error(),
		)
		return
	}

	// Environment Variables (import)
	envTF := types.SetNull(envVarObjectType())
	if containerJob.EnvironmentVariables != nil {
		setVal, _ := buildEnvSetFromAPI(ctx, containerJob.EnvironmentVariables, nil, types.SetNull(envVarObjectType()), secretMaskOnly)
		envTF = setVal
	}

	// Command
	commands := make([]attr.Value, len(containerJob.Command))
	for i, c := range containerJob.Command {
		commands[i] = types.StringValue(c)
	}

	commandList, diags := types.ListValue(types.StringType, commands)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	entrypoints := make([]attr.Value, len(containerJob.Entrypoint))
	for i, c := range containerJob.Entrypoint {
		entrypoints[i] = types.StringValue(c)
	}

	entrypointList, diags := types.ListValue(types.StringType, entrypoints)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Mounts
	mountTF := types.ListNull(MountsObjectType())
	if containerJob.Mounts != nil {
		mountList, d := buildMountsFromApi(containerJob.Mounts)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		mountTF = mountList
	}

	state := containerJobResource{
		ID:                   types.StringValue(containerJob.Name),
		Name:                 types.StringValue(containerJob.Name),
		Namespace:            types.StringValue(namespace),
		Image:                types.StringValue(containerJob.Image),
		Registry:             types.StringValue(containerJob.PrivateRegistry.Name),
		Resources:            resourcesObj,
		EnvironmentVariables: envTF,
		Command:              commandList,
		Entrypoint:           entrypointList,
		Mounts:               mountTF,
		Schedule:             types.StringValue(containerJob.Schedule),
		Enabled:              types.BoolValue(containerJob.Enabled),
		State:                types.StringValue(containerJob.State),
		LastUpdated:          types.StringValue(time.Now().Format(time.RFC3339)),
		Timeouts: timeouts.Value{
			Object: types.ObjectValueMust(
				map[string]attr.Type{
					"update": types.StringType,
					"delete": types.StringType,
				},
				map[string]attr.Value{
					"update": types.StringValue("30s"),
					"delete": types.StringValue("30s"),
				},
			),
		},
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
