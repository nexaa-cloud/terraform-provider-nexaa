// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"

	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/nexaa-cloud/nexaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &containerResource{}
	_ resource.ResourceWithImportState = &containerResource{}
	_ resource.ResourceWithIdentity    = &containerResource{}
)

// NewContainerResource is a helper function to simplify the provider implementation.
func NewContainerResource() resource.Resource {
	return &containerResource{}
}

// containerResource is the resource implementation.
type containerResource struct {
	ID                   types.String   `tfsdk:"id"`
	Name                 types.String   `tfsdk:"name"`
	Namespace            types.String   `tfsdk:"namespace"`
	Image                types.String   `tfsdk:"image"`
	Registry             types.String   `tfsdk:"registry"`
	Resources            types.String   `tfsdk:"resources"`
	EnvironmentVariables types.Set      `tfsdk:"environment_variables"`
	Ports                types.List     `tfsdk:"ports"`
	Ingresses            types.List     `tfsdk:"ingresses"`
	Mounts               types.List     `tfsdk:"mounts"`
	HealthCheck          types.Object   `tfsdk:"health_check"`
	Scaling              types.Object   `tfsdk:"scaling"`
	LastUpdated          types.String   `tfsdk:"last_updated"`
	Status               types.String   `tfsdk:"status"`
	Timeouts             timeouts.Value `tfsdk:"timeouts"`
}

type mountResource struct {
	Path   types.String `tfsdk:"path"`
	Volume types.String `tfsdk:"volume"`
}

type environmentVariableResource struct {
	Name   types.String `tfsdk:"name"`
	Value  types.String `tfsdk:"value"`
	Secret types.Bool   `tfsdk:"secret"`
}

type ingresResource struct {
	DomainName types.String `tfsdk:"domain_name"`
	Port       types.Int64  `tfsdk:"port"`
	TLS        types.Bool   `tfsdk:"tls"`
	AllowList  types.List   `tfsdk:"allow_list"`
}

type healthcheckResource struct {
	Port types.Int64  `tfsdk:"port"`
	Path types.String `tfsdk:"path"`
}

type scalingResource struct {
	Type        types.String `tfsdk:"type"`
	Manualinput types.Int64  `tfsdk:"manual_input"`
	AutoInput   types.Object `tfsdk:"auto_input"`
}

type autoscaleResource struct {
	MinimalReplicas types.Int64 `tfsdk:"minimal_replicas"`
	MaximalReplicas types.Int64 `tfsdk:"maximal_replicas"`
	Triggers        types.List  `tfsdk:"triggers"`
}

type triggerResource struct {
	Type      types.String `tfsdk:"type"`
	Threshold types.Int64  `tfsdk:"threshold"`
}

// Metadata returns the resource type name.
func (r *containerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container"
}

func (r *containerResource) IdentitySchema(ctx context.Context, request resource.IdentitySchemaRequest, response *resource.IdentitySchemaResponse) {
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

func (r *containerResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Container resource representing a container that will be deployed on nexaa.",
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
			"resources": schema.StringAttribute{
				Required:    true,
				Description: "The resources used for running the container, this can be gotten via the nexaa_container_resources data source, with specifying the amount of cpu and memory",
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
			"scaling": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Used to specify or automaticaly scale the amount of replicas running",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:    true,
						Description: "The type of scaling you want, auto or manual",
						Validators: []validator.String{
							stringvalidator.OneOf("auto", "manual"),
						},
					},
					"manual_input": schema.Int64Attribute{
						Optional:    true,
						Description: "The input for manual scaling, equal to the amount of running replicas you want",
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"auto_input": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "The input for the autoscaling",
						Attributes: map[string]schema.Attribute{
							"minimal_replicas": schema.Int64Attribute{
								Required:    true,
								Description: "The minimal amount of replicas you want",
							},
							"maximal_replicas": schema.Int64Attribute{
								Required:    true,
								Description: "The maximum amount of replicas you want to scale to",
							},
							"triggers": schema.ListNestedAttribute{
								Optional:    true,
								Description: "Used as condition as to when the container needs to add a replica, you can have 2 triggers, one for each type",
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Required:    true,
											Description: "The type of metric used for specifying what the triggers monitors, is either MEMORY or CPU",
											Validators: []validator.String{
												stringvalidator.OneOf("MEMORY", "CPU"),
											},
										},
										"threshold": schema.Int64Attribute{
											Required:    true,
											Description: "The amount percentage wise needed to add another replica",
										},
									},
								},
							},
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the container",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the private registry",
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
func (r *containerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan containerResource
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

	// Build input struct
	input := api.ContainerCreateInput{
		Namespace: plan.Namespace.ValueString(),
		Name:      plan.Name.ValueString(),
		Image:     plan.Image.ValueString(),
		Registry:  plan.Registry.ValueStringPointer(),
		Resources: api.ContainerResources(plan.Resources.ValueString()),
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

	// Scaling
	var scaling scalingResource
	diags = plan.Scaling.As(ctx, &scaling, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch scaling.Type.ValueString() {
	case "auto":
		if !scaling.AutoInput.IsNull() && !scaling.AutoInput.IsUnknown() {
			var autoInput autoscaleResource
			diags := scaling.AutoInput.As(ctx, &autoInput, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			var triggers []triggerResource
			diags = autoInput.Triggers.ElementsAs(ctx, &triggers, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			auto := api.AutoScalingInput{
				Replicas: api.ReplicasInput{
					Minimum: int(autoInput.MinimalReplicas.ValueInt64()),
					Maximum: int(autoInput.MaximalReplicas.ValueInt64()),
				},
			}

			for _, t := range triggers {
				auto.Triggers = append(auto.Triggers, api.AutoScalingTriggerInput{
					Type:      api.AutoScalingType(t.Type.ValueString()),
					Threshold: int(t.Threshold.ValueInt64()),
				})
			}

			input.Scaling = &api.ScalingInput{Auto: &auto}
		}

	case "manual":
		if !scaling.Manualinput.IsNull() && !scaling.Manualinput.IsUnknown() {
			replicas := int(scaling.Manualinput.ValueInt64())
			input.Scaling = &api.ScalingInput{
				Manual: &api.ManualScalingInput{
					Replicas: replicas,
				},
			}
		}
	}

	// Create containerResult
	client := api.NewClient()
	containerResult, err := client.ContainerCreate(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating containerResult", "Could not create containerResult: "+err.Error())
		return
	}

	// Set all fields in plan from returned containerResult
	plan.ID = types.StringValue(containerResult.Name)
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(containerResult.Name)
	plan.Image = types.StringValue(containerResult.Image)
	plan.Status = types.StringValue(containerResult.State)

	plan.Registry = processRegistryName(containerResult)

	plan.Resources = types.StringValue(string(containerResult.Resources))

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

	// Scaling
	autoInputType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"minimal_replicas": types.Int64Type,
			"maximal_replicas": types.Int64Type,
			"triggers": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":      types.StringType,
						"threshold": types.Int64Type,
					},
				},
			},
		},
	}

	scalingType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type":         types.StringType,
			"manual_input": types.Int64Type,
			"auto_input":   autoInputType,
		},
	}

	var scalingObj attr.Value = types.ObjectNull(scalingType.AttrTypes)

	if containerResult.AutoScaling != nil {
		var triggerVals []attr.Value
		for _, t := range containerResult.AutoScaling.Triggers {
			triggerObj := types.ObjectValueMust(
				map[string]attr.Type{
					"type":      types.StringType,
					"threshold": types.Int64Type,
				},
				map[string]attr.Value{
					"type":      types.StringValue(strings.ToUpper(t.Type)),
					"threshold": types.Int64Value(int64(t.Threshold)),
				},
			)
			triggerVals = append(triggerVals, triggerObj)
		}

		triggersList, diags := types.ListValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"type":      types.StringType,
					"threshold": types.Int64Type,
				},
			}, triggerVals)
		if diags.HasError() {
			resp.Diagnostics.AddError("Scaling triggers error", "Failed to build scaling triggers list")
			return
		}

		autoInput := types.ObjectValueMust(
			autoInputType.AttrTypes,
			map[string]attr.Value{
				"minimal_replicas": types.Int64Value(int64(containerResult.AutoScaling.Replicas.Minimum)),
				"maximal_replicas": types.Int64Value(int64(containerResult.AutoScaling.Replicas.Maximum)),
				"triggers":         triggersList,
			},
		)

		scalingObj = types.ObjectValueMust(
			scalingType.AttrTypes,
			map[string]attr.Value{
				"type":         types.StringValue("auto"),
				"manual_input": types.Int64Null(),
				"auto_input":   autoInput,
			},
		)
	} else if containerResult.NumberOfReplicas > 0 {
		scalingObj = types.ObjectValueMust(
			scalingType.AttrTypes,
			map[string]attr.Value{
				"type":         types.StringValue("manual"),
				"manual_input": types.Int64Value(int64(containerResult.NumberOfReplicas)),
				"auto_input":   types.ObjectNull(autoInputType.AttrTypes),
			},
		)
	}

	if obj, ok := scalingObj.(types.Object); ok {
		plan.Scaling = obj
	} else {
		resp.Diagnostics.AddError("Error creating containerResult", "Could not transform scaling object")
		return
	}

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

	// Fetch the created container
	client := api.NewClient()
	container, err := client.ListContainerByName(state.Namespace.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading container", "Could not find container: "+err.Error())
		return
	}

	// Set all fields in state from returned container
	state.ID = types.StringValue(container.Name)
	state.Namespace = types.StringValue(state.Namespace.ValueString())
	state.Name = types.StringValue(container.Name)
	state.Image = types.StringValue(container.Image)

	state.Registry = processRegistryName(container)

	state.Resources = types.StringValue(string(container.Resources))

	// Environment variables (refresh state)
	if container.EnvironmentVariables != nil {
		setVal, d := buildEnvSetFromAPI(ctx, container.EnvironmentVariables, nil, state.EnvironmentVariables, secretPreservePrev)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.EnvironmentVariables = setVal
	}

	// Ports
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
	ingressesTF, diags := buildIngressesFromApi(container)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Ingresses = ingressesTF

	// Health check
	state.HealthCheck = buildHealthCheckState(container)

	// Scaling
	// Declare autoInputType once
	autoInputType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"minimal_replicas": types.Int64Type,
			"maximal_replicas": types.Int64Type,
			"triggers": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":      types.StringType,
						"threshold": types.Int64Type,
					},
				},
			},
		},
	}

	// Declare outer scaling type
	scalingType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type":         types.StringType,
			"manual_input": types.Int64Type,
			"auto_input":   autoInputType,
		},
	}

	// Initialize scalingObj to null (failsafe default)
	var scalingObj attr.Value = types.ObjectNull(scalingType.AttrTypes)

	if container.AutoScaling != nil {
		var triggerVals []attr.Value
		for _, t := range container.AutoScaling.Triggers {
			triggerObj := types.ObjectValueMust(
				map[string]attr.Type{
					"type":      types.StringType,
					"threshold": types.Int64Type,
				},
				map[string]attr.Value{
					"type":      types.StringValue(strings.ToUpper(t.Type)),
					"threshold": types.Int64Value(int64(t.Threshold)),
				},
			)
			triggerVals = append(triggerVals, triggerObj)
		}

		triggersList, diags := types.ListValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"type":      types.StringType,
					"threshold": types.Int64Type,
				},
			}, triggerVals)
		if diags.HasError() {
			return
		}

		autoInput := types.ObjectValueMust(
			autoInputType.AttrTypes,
			map[string]attr.Value{
				"minimal_replicas": types.Int64Value(int64(container.AutoScaling.Replicas.Minimum)),
				"maximal_replicas": types.Int64Value(int64(container.AutoScaling.Replicas.Maximum)),
				"triggers":         triggersList,
			},
		)

		scalingObj = types.ObjectValueMust(
			scalingType.AttrTypes,
			map[string]attr.Value{
				"type":         types.StringValue("auto"),
				"manual_input": types.Int64Null(),
				"auto_input":   autoInput,
			},
		)
	} else if container.NumberOfReplicas > 0 {
		scalingObj = types.ObjectValueMust(
			scalingType.AttrTypes,
			map[string]attr.Value{
				"type":         types.StringValue("manual"),
				"manual_input": types.Int64Value(int64(container.NumberOfReplicas)),
				"auto_input":   types.ObjectNull(autoInputType.AttrTypes),
			},
		)
	}

	if obj, ok := scalingObj.(types.Object); ok {
		state.Scaling = obj
	} else {
		resp.Diagnostics.AddError("Error creating container", "Could not read container: "+err.Error())
		return
	}

	state.Status = types.StringValue(container.State)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *containerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan containerResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Registry.IsNull() || plan.Registry.IsUnknown() {
		plan.Registry = types.StringNull()
	}

	containerResources := api.ContainerResources(plan.Resources.ValueString())

	// Build input struct
	input := api.ContainerModifyInput{
		Namespace: plan.Namespace.ValueString(),
		Name:      plan.Name.ValueString(),
		Image:     plan.Image.ValueStringPointer(),
		Registry:  plan.Registry.ValueStringPointer(),
		Resources: &containerResources,
	}

	// Ports
	ports, diags := buildPortsInput(ctx, plan.Ports)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input.Ports = ports

	// Mounts - get previous state for comparison
	var prev containerResource
	prevMounts := types.ListNull(MountsObjectType())
	if !req.State.Raw.IsNull() && req.State.Raw.IsKnown() {
		diags := req.State.Get(ctx, &prev)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		prevMounts = prev.Mounts
	}

	mounts, diags := buildMountsUpdateInput(ctx, plan.Mounts, prevMounts)
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

	// Scaling
	var scaling scalingResource
	diags = plan.Scaling.As(ctx, &scaling, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch scaling.Type.ValueString() {
	case "auto":
		if !scaling.AutoInput.IsNull() && !scaling.AutoInput.IsUnknown() {
			var autoInput autoscaleResource
			diags := scaling.AutoInput.As(ctx, &autoInput, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			var triggers []triggerResource
			diags = autoInput.Triggers.ElementsAs(ctx, &triggers, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			auto := api.AutoScalingInput{
				Replicas: api.ReplicasInput{
					Minimum: int(autoInput.MinimalReplicas.ValueInt64()),
					Maximum: int(autoInput.MaximalReplicas.ValueInt64()),
				},
			}

			for _, t := range triggers {
				auto.Triggers = append(auto.Triggers, api.AutoScalingTriggerInput{
					Type:      api.AutoScalingType(t.Type.ValueString()),
					Threshold: int(t.Threshold.ValueInt64()),
				})
			}

			input.Scaling = &api.ScalingInput{Auto: &auto}
		}

	case "manual":
		if !scaling.Manualinput.IsNull() && !scaling.Manualinput.IsUnknown() {
			replicas := int(scaling.Manualinput.ValueInt64())
			input.Scaling = &api.ScalingInput{
				Manual: &api.ManualScalingInput{
					Replicas: replicas,
				},
			}
		}
	}

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
		resp.Diagnostics.AddError("Error creating containerResult", "Could not reach a running state: "+err.Error())
	}

	// modify containerResult
	containerResult, err := client.ContainerModify(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating containerResult", "Could not create containerResult: "+err.Error())
		return
	}

	err = waitForUnlocked(ctx, containerLocked(), *client, plan.Namespace.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating containerResult", "Could not reach a running state: "+err.Error())
	}

	containerResult, err = client.ListContainerByName(plan.Namespace.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating containerResult", "Could not update containerResult: "+err.Error())
		return
	}

	// Set all fields in plan from returned containerResult
	plan.ID = types.StringValue(containerResult.Name)
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(containerResult.Name)
	plan.Image = types.StringValue(containerResult.Image)

	plan.Registry = processRegistryName(containerResult)

	plan.Resources = types.StringValue(string(containerResult.Resources))

	// Environment variables (update state)
	if containerResult.EnvironmentVariables != nil {
		setVal, d := buildEnvSetFromAPI(ctx, containerResult.EnvironmentVariables, input.EnvironmentVariables, plan.EnvironmentVariables, secretUseProvided)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.EnvironmentVariables = setVal
	}

	// Ports
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

	// Scaling
	autoInputType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"minimal_replicas": types.Int64Type,
			"maximal_replicas": types.Int64Type,
			"triggers": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":      types.StringType,
						"threshold": types.Int64Type,
					},
				},
			},
		},
	}

	scalingType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type":         types.StringType,
			"manual_input": types.Int64Type,
			"auto_input":   autoInputType,
		},
	}

	var scalingObj attr.Value = types.ObjectNull(scalingType.AttrTypes)

	if containerResult.AutoScaling != nil {
		var triggerVals []attr.Value
		for _, t := range containerResult.AutoScaling.Triggers {
			triggerObj := types.ObjectValueMust(
				map[string]attr.Type{
					"type":      types.StringType,
					"threshold": types.Int64Type,
				},
				map[string]attr.Value{
					"type":      types.StringValue(strings.ToUpper(t.Type)),
					"threshold": types.Int64Value(int64(t.Threshold)),
				},
			)
			triggerVals = append(triggerVals, triggerObj)
		}

		triggersList, diags := types.ListValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"type":      types.StringType,
					"threshold": types.Int64Type,
				},
			}, triggerVals)
		if diags.HasError() {
			resp.Diagnostics.AddError("Scaling triggers error", "Failed to build scaling triggers list")
			return
		}

		autoInput := types.ObjectValueMust(
			autoInputType.AttrTypes,
			map[string]attr.Value{
				"minimal_replicas": types.Int64Value(int64(containerResult.AutoScaling.Replicas.Minimum)),
				"maximal_replicas": types.Int64Value(int64(containerResult.AutoScaling.Replicas.Maximum)),
				"triggers":         triggersList,
			},
		)

		scalingObj = types.ObjectValueMust(
			scalingType.AttrTypes,
			map[string]attr.Value{
				"type":         types.StringValue("auto"),
				"manual_input": types.Int64Null(),
				"auto_input":   autoInput,
			},
		)
	} else if containerResult.NumberOfReplicas > 0 {
		scalingObj = types.ObjectValueMust(
			scalingType.AttrTypes,
			map[string]attr.Value{
				"type":         types.StringValue("manual"),
				"manual_input": types.Int64Value(int64(containerResult.NumberOfReplicas)),
				"auto_input":   types.ObjectNull(autoInputType.AttrTypes),
			},
		)
	}

	if obj, ok := scalingObj.(types.Object); ok {
		plan.Scaling = obj
	} else {
		resp.Diagnostics.AddError("Error updating containerResult", "Could not update containerResult: "+err.Error())
		return
	}

	plan.Status = types.StringValue(containerResult.State)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *containerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan containerResource
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
		resp.Diagnostics.AddError("Error creating containerResult", "Could not reach a running plan: "+err.Error())
	}

	_, err = client.ContainerDelete(plan.Namespace.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting container",
			fmt.Sprintf("Failed to delete container %q: %s", plan.Name.ValueString(), err.Error()),
		)
		return
	}
}

func (r *containerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
			"Error importing container",
			fmt.Sprintf("Unable to fetch container %q in namespace %q: %s", name, namespace, err.Error()),
		)
		return
	}

	// Use common function to build basic state
	stateValues, diags := buildContainerImportState(ctx, container, namespace, name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build scaling state (not included in common function since starter containers don't have scaling)
	autoInputType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"minimal_replicas": types.Int64Type,
			"maximal_replicas": types.Int64Type,
			"triggers": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":      types.StringType,
						"threshold": types.Int64Type,
					},
				},
			},
		},
	}

	scalingType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type":         types.StringType,
			"manual_input": types.Int64Type,
			"auto_input":   autoInputType,
		},
	}

	var scalingObj attr.Value = types.ObjectNull(scalingType.AttrTypes)

	if container.AutoScaling != nil {
		var triggerVals []attr.Value
		for _, t := range container.AutoScaling.Triggers {
			triggerObj := types.ObjectValueMust(
				map[string]attr.Type{
					"type":      types.StringType,
					"threshold": types.Int64Type,
				},
				map[string]attr.Value{
					"type":      types.StringValue(strings.ToUpper(t.Type)),
					"threshold": types.Int64Value(int64(t.Threshold)),
				},
			)
			triggerVals = append(triggerVals, triggerObj)
		}

		triggersList, diags := types.ListValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"type":      types.StringType,
					"threshold": types.Int64Type,
				},
			}, triggerVals)
		if diags.HasError() {
			resp.Diagnostics.AddError("Scaling triggers error", "Failed to build scaling triggers list")
			return
		}

		autoInput := types.ObjectValueMust(
			autoInputType.AttrTypes,
			map[string]attr.Value{
				"minimal_replicas": types.Int64Value(int64(container.AutoScaling.Replicas.Minimum)),
				"maximal_replicas": types.Int64Value(int64(container.AutoScaling.Replicas.Maximum)),
				"triggers":         triggersList,
			},
		)

		scalingObj = types.ObjectValueMust(
			scalingType.AttrTypes,
			map[string]attr.Value{
				"type":         types.StringValue("auto"),
				"manual_input": types.Int64Null(),
				"auto_input":   autoInput,
			},
		)
	} else if container.NumberOfReplicas > 0 {
		scalingObj = types.ObjectValueMust(
			scalingType.AttrTypes,
			map[string]attr.Value{
				"type":         types.StringValue("manual"),
				"manual_input": types.Int64Value(int64(container.NumberOfReplicas)),
				"auto_input":   types.ObjectNull(autoInputType.AttrTypes),
			},
		)
	}

	// Create state using common values and add scaling + timeouts
	state := containerResource{
		ID:                   stateValues["id"].(types.String),
		Name:                 stateValues["name"].(types.String),
		Namespace:            stateValues["namespace"].(types.String),
		Image:                stateValues["image"].(types.String),
		Registry:             stateValues["registry"].(types.String),
		Resources:            types.StringValue(string(container.Resources)),
		EnvironmentVariables: stateValues["environment_variables"].(types.Set),
		Ports:                stateValues["ports"].(types.List),
		Ingresses:            stateValues["ingresses"].(types.List),
		Mounts:               stateValues["mounts"].(types.List),
		HealthCheck:          stateValues["health_check"].(types.Object),
		Status:               stateValues["status"].(types.String),
		LastUpdated:          stateValues["last_updated"].(types.String),
	}

	// Add scaling (specific to regular containers)
	if obj, ok := scalingObj.(types.Object); ok {
		state.Scaling = obj
	} else {
		resp.Diagnostics.AddError("Error importing container", "Failed to build scaling object")
		return
	}

	// Add timeouts
	state.Timeouts = timeouts.Value{
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func processRegistryName(containerResult api.ContainerResult) types.String {
	if containerResult.PrivateRegistry != nil {
		return types.StringValue(containerResult.PrivateRegistry.Name)
	}
	return types.StringNull()
}
