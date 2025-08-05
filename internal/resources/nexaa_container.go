// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nexaa-cloud/nexaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nexaa-cloud/terraform-provider-nexaa/internal/enums"
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
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Namespace            types.String `tfsdk:"namespace"`
	Image                types.String `tfsdk:"image"`
	Registry             types.String `tfsdk:"registry"`
	Resources            types.Object `tfsdk:"resources"`
	EnvironmentVariables types.List   `tfsdk:"environment_variables"`
	Ports                types.List   `tfsdk:"ports"`
	Ingresses            types.List   `tfsdk:"ingresses"`
	Mounts               types.List   `tfsdk:"mounts"`
	HealthCheck          types.Object `tfsdk:"health_check"`
	Scaling              types.Object `tfsdk:"scaling"`
	LastUpdated          types.String `tfsdk:"last_updated"`
	Status               types.String `tfsdk:"status"`
}

type resourcesResource struct {
	CPU types.Float64 `tfsdk:"cpu"`
	RAM types.Float64 `tfsdk:"ram"`
}

type mountResource struct {
	Path   types.String `tfsdk:"path"`
	Volume types.String `tfsdk:"volume"`
}

type environvariableResource struct {
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

func (r *containerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Container resource representing a container that will be deployed on nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the container, equal to the name",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the container",
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
			"resources": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"cpu": schema.Float64Attribute{
						Required:    true,
						Description: "The amount of cpu used for the container, can be the following values: 0.25, 0.5, 0.75, 1, 2, 3, 4",
						Validators: []validator.Float64{
							float64validator.OneOf(enums.CPU...),
						},
					},
					"ram": schema.Float64Attribute{
						Required:    true,
						Description: "The amount of ram used for the container (in GB), can be the following values: 0.5, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16",
						Validators: []validator.Float64{
							float64validator.OneOf(enums.RAM...),
						},
					},
				},
				Required:    true,
				Description: "The resources used for running the container",
			},
			"ports": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "The ports used to expose for traffic, format as from:to",
			},
			"environment_variables": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The name used for the environment variable",
						},
						"value": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The value used for the environment variable, is required",
						},
						"secret": schema.BoolAttribute{
							Optional:    true,
							Description: "A boolean to represent if the environment variable is a secret or not",
						},
					},
				},
				Optional:    true,
				Computed:    true,
				Description: "Environment variables used in the container, write the non-secrets first the the secrets",
			},
			"ingresses": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"domain_name": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The domain used for the ingress, defaults to https://101010-{namespaceName}-{containerName}.container.tilaa.cloud",
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
								Description: "Used as condition as to when the container needs to add a replica, you can have 2 triggers, one for eacht type",
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Required:    true,
											Description: "The type of metric used for specifying what the triggers monitors, is eihter MEMORY or CPU",
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
func (r *containerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan containerResource
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
	input := api.ContainerCreateInput{
		Namespace: plan.Namespace.ValueString(),
		Name:      plan.Name.ValueString(),
		Image:     plan.Image.ValueString(),
		Registry:  plan.Registry.ValueStringPointer(),
		Resources: api.ContainerResources(cpuRam),
	}

	//Ports
	if !plan.Ports.IsNull() && !plan.Ports.IsUnknown() {
		var ports []string
		diags = plan.Ports.ElementsAs(ctx, &ports, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Ports = ports
	} else {
		input.Ports = make([]string, 0)
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

	// Ingress
	if !plan.Ingresses.IsNull() && !plan.Ingresses.IsUnknown() {
		var ingresses []ingresResource
		diags = plan.Ingresses.ElementsAs(ctx, &ingresses, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Ingresses = make([]api.IngressInput, 0, len(ingresses))
		for _, ing := range ingresses {
			if !ing.Port.IsNull() {
				allowList := []string{}
				if !ing.AllowList.IsNull() && !ing.AllowList.IsUnknown() {
					var rawAllowList []types.String
					_ = ing.AllowList.ElementsAs(ctx, &rawAllowList, false)
					for _, ip := range rawAllowList {
						allowList = append(allowList, ip.ValueString())
					}
				}
				var domain string
				if !ing.DomainName.IsNull() || !ing.DomainName.IsUnknown() {
					domain = ing.DomainName.ValueString()
				} else {
					domain = "	"
				}
				input.Ingresses = append(input.Ingresses, api.IngressInput{
					DomainName: &domain,
					Port:       int(ing.Port.ValueInt64()),
					EnableTLS:  ing.TLS.ValueBool(),
					Whitelist:  allowList,
					State:      api.StatePresent,
				})
			}
		}
	} else {
		input.Ingresses = []api.IngressInput{}
	}

	// Environment variables
	if !plan.EnvironmentVariables.IsNull() && !plan.EnvironmentVariables.IsUnknown() {
		var envVars []environvariableResource
		diags = plan.EnvironmentVariables.ElementsAs(ctx, &envVars, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, ev := range envVars {
			if ev.Value.IsNull() || ev.Value.IsUnknown() {
				resp.Diagnostics.AddError(
					"Error Creating Container",
					"The value for this Env variable is null or unknown, "+ev.Name.ValueString(),
				)
				return
			}
			input.EnvironmentVariables = append(input.EnvironmentVariables, api.EnvironmentVariableInput{
				Name:   ev.Name.ValueString(),
				Value:  ev.Value.ValueString(),
				Secret: ev.Secret.ValueBool(),
				State:  api.StatePresent,
			})
		}
	}

	// Healthcheck
	if !plan.HealthCheck.IsNull() && !plan.HealthCheck.IsUnknown() {
		var hc healthcheckResource
		diags = plan.HealthCheck.As(ctx, &hc, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		input.HealthCheck = &api.HealthCheckInput{
			Port: int(hc.Port.ValueInt64()),
			Path: hc.Path.ValueString(),
		}
	}

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

	// Create container
	client := api.NewClient()
	container, err := client.ContainerCreate(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating container", "Could not create container: "+err.Error())
		return
	}

	// Set all fields in plan from returned container
	plan.ID = types.StringValue(container.Name)
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(container.Name)
	plan.Image = types.StringValue(container.Image)

	if container.PrivateRegistry == nil || container.PrivateRegistry.Name == "public" {
		plan.Registry = types.StringNull()
	} else {
		plan.Registry = types.StringValue(*input.Registry)
	}

	// Parse CPU and RAM from the string
	resParts := strings.Split(string(container.Resources), "_")
	if len(resParts) == 4 {
		cpu, err := strconv.ParseFloat(resParts[1], 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing container resources",
				"Failed to parse CPU value: "+err.Error(),
			)
			return
		}

		ram, err := strconv.ParseFloat(resParts[3], 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing container resources",
				"Failed to parse RAM value: "+err.Error(),
			)
			return
		}

		// Create a new types.Object with CPU and RAM fields set
		resourcesObj := types.ObjectValueMust(
			map[string]attr.Type{
				"cpu": types.Float64Type,
				"ram": types.Float64Type,
			},
			map[string]attr.Value{
				"cpu": types.Float64Value(cpu / 1000),
				"ram": types.Float64Value(ram / 1000),
			},
		)

		// Assign it to plan.Resources
		plan.Resources = resourcesObj
	}

	// Environment variables
	if container.EnvironmentVariables != nil {
		envVars := make([]attr.Value, len(container.EnvironmentVariables))
		for i, ev := range container.EnvironmentVariables {
			var val types.String
			if ev.Secret {
				for _, env := range input.EnvironmentVariables {
					if ev.Name == env.Name {
						val = types.StringValue(env.Value)
					}
				}
			} else {
				val = types.StringValue(*ev.Value)
			}
			obj := types.ObjectValueMust(
				map[string]attr.Type{
					"name":   types.StringType,
					"value":  types.StringType,
					"secret": types.BoolType,
				},
				map[string]attr.Value{
					"name":   types.StringValue(ev.Name),
					"value":  val,
					"secret": types.BoolValue(ev.Secret),
				},
			)
			envVars[i] = obj
		}
		listVal, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":   types.StringType,
				"value":  types.StringType,
				"secret": types.BoolType,
			},
		}, envVars)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.EnvironmentVariables = listVal
	}

	// Ports
	if container.Ports != nil {
		ports := make([]attr.Value, len(container.Ports))
		for i, p := range container.Ports {
			ports[i] = types.StringValue(p)
		}

		portList, diags := types.ListValue(types.StringType, ports)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Ports = portList
	}

	// Mounts
	if container.Mounts != nil {
		mounts := make([]attr.Value, len(container.Mounts))
		for i, m := range container.Mounts {
			obj := types.ObjectValueMust(
				map[string]attr.Type{
					"path":   types.StringType,
					"volume": types.StringType,
				},
				map[string]attr.Value{
					"path":   types.StringValue(m.Path),
					"volume": types.StringValue(m.Volume.Name),
				})
			mounts[i] = obj
		}
		mountList, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"path":   types.StringType,
				"volume": types.StringType,
			},
		}, mounts)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Mounts = mountList
	}

	// Ingresses
	var ingressElems []attr.Value
	for _, ing := range container.Ingresses {
		allowListElems := make([]attr.Value, len(ing.Allowlist))
		for i, a := range ing.Allowlist {
			allowListElems[i] = types.StringValue(a)
		}
		allowList, diags := types.ListValue(types.StringType, allowListElems)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var ingDomain types.String

		if container.Ingresses == nil || strings.Contains(ing.DomainName, ".tilaa.cloud") {
			ingDomain = types.StringNull()
		} else {
			ingDomain = types.StringValue(ing.DomainName)
		}

		ingressObj := types.ObjectValueMust(
			map[string]attr.Type{
				"domain_name": types.StringType,
				"port":        types.Int64Type,
				"tls":         types.BoolType,
				"allow_list":  types.ListType{ElemType: types.StringType},
			},
			map[string]attr.Value{
				"domain_name": ingDomain,
				"port":        types.Int64Value(int64(ing.Port)),
				"tls":         types.BoolValue(ing.EnableTLS),
				"allow_list":  allowList,
			})
		ingressElems = append(ingressElems, ingressObj)
	}

	ingressesList, diags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"domain_name": types.StringType,
				"port":        types.Int64Type,
				"tls":         types.BoolType,
				"allow_list":  types.ListType{ElemType: types.StringType},
			},
		},
		ingressElems,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Ingresses = ingressesList

	// Health check
	if container.HealthCheck != nil {
		hc := types.ObjectValueMust(map[string]attr.Type{
			"port": types.Int64Type,
			"path": types.StringType,
		},
			map[string]attr.Value{
				"port": types.Int64Value(int64(container.HealthCheck.Port)),
				"path": types.StringValue(container.HealthCheck.Path),
			})
		plan.HealthCheck = hc
	}

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
				"manual_input": types.Int64Null(), // explicitly null
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
		plan.Scaling = obj
	} else {
		resp.Diagnostics.AddError("Error creating container", "Could not create container: "+err.Error())
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	plan.Status = types.StringValue(container.State)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
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

	if container.PrivateRegistry == nil || container.PrivateRegistry.Name == "public" {
		state.Registry = types.StringNull()
	} else {
		state.Registry = types.StringValue(container.PrivateRegistry.Name)
	}

	// Parse CPU and RAM from the string
	resParts := strings.Split(string(container.Resources), "_")
	if len(resParts) == 4 {
		cpu, err := strconv.ParseFloat(resParts[1], 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing container resources",
				"Failed to parse CPU value: "+err.Error(),
			)
			return
		}

		ram, err := strconv.ParseFloat(resParts[3], 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing container resources",
				"Failed to parse RAM value: "+err.Error(),
			)
			return
		}

		// Create a new types.Object with CPU and RAM fields set
		resourcesObj := types.ObjectValueMust(
			map[string]attr.Type{
				"cpu": types.Float64Type,
				"ram": types.Float64Type,
			},
			map[string]attr.Value{
				"cpu": types.Float64Value(cpu / 1000),
				"ram": types.Float64Value(ram / 1000),
			},
		)

		// Assign it to plan.Resources
		state.Resources = resourcesObj
	}

	// Environment variables
	if container.EnvironmentVariables != nil {
		envVars := make([]attr.Value, len(container.EnvironmentVariables))
		for i, ev := range container.EnvironmentVariables {
			var val types.String
			if ev.Secret {
				val = types.StringNull()
			} else {
				val = types.StringValue(*ev.Value)
			}
			obj := types.ObjectValueMust(
				map[string]attr.Type{
					"name":   types.StringType,
					"value":  types.StringType,
					"secret": types.BoolType,
				},
				map[string]attr.Value{
					"name":   types.StringValue(ev.Name),
					"value":  val,
					"secret": types.BoolValue(ev.Secret),
				},
			)
			envVars[i] = obj
		}
		listVal, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":   types.StringType,
				"value":  types.StringType,
				"secret": types.BoolType,
			},
		}, envVars)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.EnvironmentVariables = listVal
	}

	// Ports
	if container.Ports != nil {
		ports := make([]attr.Value, len(container.Ports))
		for i, p := range container.Ports {
			ports[i] = types.StringValue(p)
		}

		portList, diags := types.ListValue(types.StringType, ports)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Ports = portList
	}

	// Mounts
	if container.Mounts != nil {
		mounts := make([]attr.Value, len(container.Mounts))
		for i, m := range container.Mounts {
			obj := types.ObjectValueMust(
				map[string]attr.Type{
					"path":   types.StringType,
					"volume": types.StringType,
				},
				map[string]attr.Value{
					"path":   types.StringValue(m.Path),
					"volume": types.StringValue(m.Volume.Name),
				})
			mounts[i] = obj
		}
		mountList, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"path":   types.StringType,
				"volume": types.StringType,
			},
		}, mounts)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Mounts = mountList
	}

	// Ingresses
	var ingressElems []attr.Value
	for _, ing := range container.Ingresses {
		allowListElems := make([]attr.Value, len(ing.Allowlist))
		for i, a := range ing.Allowlist {
			allowListElems[i] = types.StringValue(a)
		}
		allowList, diags := types.ListValue(types.StringType, allowListElems)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var ingDomain types.String

		if container.Ingresses == nil || strings.Contains(ing.DomainName, ".tilaa.cloud") {
			ingDomain = types.StringNull()
		} else {
			ingDomain = types.StringValue(ing.DomainName)
		}

		ingressObj := types.ObjectValueMust(
			map[string]attr.Type{
				"domain_name": types.StringType,
				"port":        types.Int64Type,
				"tls":         types.BoolType,
				"allow_list":  types.ListType{ElemType: types.StringType},
			},
			map[string]attr.Value{
				"domain_name": ingDomain,
				"port":        types.Int64Value(int64(ing.Port)),
				"tls":         types.BoolValue(ing.EnableTLS),
				"allow_list":  allowList,
			})
		ingressElems = append(ingressElems, ingressObj)
	}

	ingressesList, diags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"domain_name": types.StringType,
				"port":        types.Int64Type,
				"tls":         types.BoolType,
				"allow_list":  types.ListType{ElemType: types.StringType},
			},
		},
		ingressElems,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Ingresses = ingressesList

	// Health check
	if container.HealthCheck != nil {
		hc := types.ObjectValueMust(map[string]attr.Type{
			"port": types.Int64Type,
			"path": types.StringType,
		},
			map[string]attr.Value{
				"port": types.Int64Value(int64(container.HealthCheck.Port)),
				"path": types.StringValue(container.HealthCheck.Path),
			})
		state.HealthCheck = hc
	}

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
				"manual_input": types.Int64Null(), // explicitly null
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
	input := api.ContainerModifyInput{
		Namespace: plan.Namespace.ValueString(),
		Name:      plan.Name.ValueString(),
		Image:     plan.Image.ValueStringPointer(),
		Registry:  plan.Registry.ValueStringPointer(),
		Resources: (*api.ContainerResources)(&cpuRam),
	}

	//Ports
	if !plan.Ports.IsNull() && !plan.Ports.IsUnknown() {
		var ports []string
		diags = plan.Ports.ElementsAs(ctx, &ports, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Ports = ports
	} else {
		input.Ports = make([]string, 0)
	}

	// Mounts
	var previousMounts []mountResource
	if !req.State.Raw.IsNull() && req.State.Raw.IsKnown() {
		var prev containerResource
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

	// Ingress
	if !plan.Ingresses.IsNull() && !plan.Ingresses.IsUnknown() {
		var ingresses []ingresResource
		diags = plan.Ingresses.ElementsAs(ctx, &ingresses, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Ingresses = make([]api.IngressInput, 0, len(ingresses))
		for _, ing := range ingresses {
			if !ing.Port.IsNull() {
				allowList := []string{}
				if !ing.AllowList.IsNull() && !ing.AllowList.IsUnknown() {
					var rawAllowList []types.String
					_ = ing.AllowList.ElementsAs(ctx, &rawAllowList, false)
					for _, ip := range rawAllowList {
						allowList = append(allowList, ip.ValueString())
					}
				}
				if !ing.DomainName.IsNull() && !ing.DomainName.IsUnknown() {
					input.Ingresses = append(input.Ingresses, api.IngressInput{
						DomainName: ing.DomainName.ValueStringPointer(),
						Port:       int(ing.Port.ValueInt64()),
						EnableTLS:  ing.TLS.ValueBool(),
						Whitelist:  allowList,
						State:      api.StatePresent,
					})
				} else {
					input.Ingresses = append(input.Ingresses, api.IngressInput{
						DomainName: nil,
						Port:       int(ing.Port.ValueInt64()),
						EnableTLS:  ing.TLS.ValueBool(),
						Whitelist:  allowList,
						State:      api.StatePresent,
					})
				}

			}
		}
	} else {
		input.Ingresses = []api.IngressInput{}
	}

	// Environment variables
	if !plan.EnvironmentVariables.IsNull() && !plan.EnvironmentVariables.IsUnknown() {
		var envVars []environvariableResource
		diags = plan.EnvironmentVariables.ElementsAs(ctx, &envVars, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, ev := range envVars {
			input.EnvironmentVariables = append(input.EnvironmentVariables, api.EnvironmentVariableInput{
				Name:   ev.Name.ValueString(),
				Value:  ev.Value.ValueString(),
				Secret: ev.Secret.ValueBool(),
				State:  api.StatePresent,
			})
		}
	}

	// Healthcheck
	if !plan.HealthCheck.IsNull() && !plan.HealthCheck.IsUnknown() {
		var hc healthcheckResource
		diags = plan.HealthCheck.As(ctx, &hc, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		input.HealthCheck = &api.HealthCheckInput{
			Port: int(hc.Port.ValueInt64()),
			Path: hc.Path.ValueString(),
		}
	}

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

	// modify container
	client := api.NewClient()
	container, err := client.ContainerModify(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating container", "Could not create container: "+err.Error())
		return
	}

	// Set all fields in plan from returned container
	plan.ID = types.StringValue(container.Name)
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(container.Name)
	plan.Image = types.StringValue(container.Image)

	if container.PrivateRegistry == nil {
		plan.Registry = types.StringNull()
	} else {
		plan.Registry = types.StringValue(container.PrivateRegistry.Name)
	}

	// Parse CPU and RAM from the string
	resParts := strings.Split(string(container.Resources), "_")
	if len(resParts) == 4 {
		cpu, err := strconv.ParseFloat(resParts[1], 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing container resources",
				"Failed to parse CPU value: "+err.Error(),
			)
			return
		}

		ram, err := strconv.ParseFloat(resParts[3], 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing container resources",
				"Failed to parse RAM value: "+err.Error(),
			)
			return
		}

		// Create a new types.Object with CPU and RAM fields set
		resourcesObj := types.ObjectValueMust(
			map[string]attr.Type{
				"cpu": types.Float64Type,
				"ram": types.Float64Type,
			},
			map[string]attr.Value{
				"cpu": types.Float64Value(cpu / 1000),
				"ram": types.Float64Value(ram / 1000),
			},
		)

		// Assign it to plan.Resources
		plan.Resources = resourcesObj
	}

	// Environment variables
	if container.EnvironmentVariables != nil {
		envVars := make([]attr.Value, len(container.EnvironmentVariables))
		for i, ev := range container.EnvironmentVariables {
			var val types.String
			if !ev.Secret {
				val = types.StringValue(*ev.Value)
			} else {
				for _, env := range input.EnvironmentVariables {
					if env.Name == ev.Name {
						val = types.StringValue(env.Value)
					}
				}
			}
			obj := types.ObjectValueMust(
				map[string]attr.Type{
					"name":   types.StringType,
					"value":  types.StringType,
					"secret": types.BoolType,
				},
				map[string]attr.Value{
					"name":   types.StringValue(ev.Name),
					"value":  val,
					"secret": types.BoolValue(ev.Secret),
				},
			)
			envVars[i] = obj
		}
		listVal, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":   types.StringType,
				"value":  types.StringType,
				"secret": types.BoolType,
			},
		}, envVars)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.EnvironmentVariables = listVal
	}

	// Ports
	if container.Ports != nil {
		ports := make([]attr.Value, len(container.Ports))
		for i, p := range container.Ports {
			ports[i] = types.StringValue(p)
		}

		portList, diags := types.ListValue(types.StringType, ports)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Ports = portList
	}

	// Mounts
	if container.Mounts != nil {
		mounts := make([]attr.Value, len(container.Mounts))
		for i, m := range container.Mounts {
			obj := types.ObjectValueMust(
				map[string]attr.Type{
					"path":   types.StringType,
					"volume": types.StringType,
				},
				map[string]attr.Value{
					"path":   types.StringValue(m.Path),
					"volume": types.StringValue(m.Volume.Name),
				})
			mounts[i] = obj
		}
		mountList, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"path":   types.StringType,
				"volume": types.StringType,
			},
		}, mounts)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Mounts = mountList
	}

	// Ingresses
	var ingressElems []attr.Value
	for _, ing := range container.Ingresses {
		allowListElems := make([]attr.Value, len(ing.Allowlist))
		for i, a := range ing.Allowlist {
			allowListElems[i] = types.StringValue(a)
		}
		allowList, diags := types.ListValue(types.StringType, allowListElems)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var ingDomain types.String

		if container.Ingresses == nil || strings.Contains(ing.DomainName, ".tilaa.cloud") {
			ingDomain = types.StringNull()
		} else {
			ingDomain = types.StringValue(ing.DomainName)
		}

		ingressObj := types.ObjectValueMust(
			map[string]attr.Type{
				"domain_name": types.StringType,
				"port":        types.Int64Type,
				"tls":         types.BoolType,
				"allow_list":  types.ListType{ElemType: types.StringType},
			},
			map[string]attr.Value{
				"domain_name": ingDomain,
				"port":        types.Int64Value(int64(ing.Port)),
				"tls":         types.BoolValue(ing.EnableTLS),
				"allow_list":  allowList,
			})
		ingressElems = append(ingressElems, ingressObj)
	}

	ingressesList, diags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"domain_name": types.StringType,
				"port":        types.Int64Type,
				"tls":         types.BoolType,
				"allow_list":  types.ListType{ElemType: types.StringType},
			},
		},
		ingressElems,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Ingresses = ingressesList

	// Health check
	if container.HealthCheck != nil {
		hc := types.ObjectValueMust(map[string]attr.Type{
			"port": types.Int64Type,
			"path": types.StringType,
		},
			map[string]attr.Value{
				"port": types.Int64Value(int64(container.HealthCheck.Port)),
				"path": types.StringValue(container.HealthCheck.Path),
			})
		plan.HealthCheck = hc
	}

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
		plan.Scaling = obj
	} else {
		resp.Diagnostics.AddError("Error updating container", "Could not update container: "+err.Error())
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	plan.Status = types.StringValue(container.State)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
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
		maxRetries   = 10
		initialDelay = 2 * time.Second
	)
	delay := initialDelay

	client := api.NewClient()

	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		container, err := client.ListContainerByName(state.Namespace.ValueString(), state.Name.ValueString())
		if err != nil {
			lastErr = err
			resp.Diagnostics.AddError(
				"Error looking up container",
				fmt.Sprintf("Could not find container with name %q: %s", state.Name.ValueString(), err.Error()),
			)
			return
		}
		if container.State == "created" {
			_, err := client.VolumeDelete(state.Namespace.ValueString(), state.Name.ValueString())
			if err != nil {
				lastErr = err
				resp.Diagnostics.AddError(
					"Error deleting container",
					fmt.Sprintf("Failed to delete container %q: %s", state.Name.ValueString(), err.Error()),
				)
				return
			}
			return
		}
		if !container.Locked {
			_, err := client.VolumeDelete(state.Namespace.ValueString(), state.Name.ValueString())
			if err != nil {
				lastErr = err
				resp.Diagnostics.AddError(
					"Error deleting container",
					fmt.Sprintf("Failed to delete container %q: %s", state.Name.ValueString(), err.Error()),
				)
				return
			}
			return
		}

		time.Sleep(delay)
		delay *= 2
	}

	if lastErr != nil {
		resp.Diagnostics.AddError(
			"Failed to delete container",
			fmt.Sprintf("Container could not be deleted after retries. Last error: %s", lastErr.Error()),
		)
	} else {
		resp.Diagnostics.AddError(
			"Failed to delete container",
			"Container could not be deleted after retries, but no specific error was returned.",
		)
	}
}

func (r *containerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format \"<namespace>/<container_name>\", got: "+req.ID,
		)
		return
	}
	namespace := parts[0]
	name := parts[1]

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

	// resources
	resParts := strings.Split(string(container.Resources), "_")
	if len(resParts) != 4 {
		resp.Diagnostics.AddError(
			"Error importing container",
			"Error while importing a container, err: "+err.Error(),
		)
		return
	}
	cpu, err := strconv.ParseFloat(resParts[1], 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing container resources",
			"Failed to parse CPU value: "+err.Error(),
		)
		return
	}

	ram, err := strconv.ParseFloat(resParts[3], 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing container resources",
			"Failed to parse RAM value: "+err.Error(),
		)
		return
	}

	// Create a new types.Object with CPU and RAM fields set
	resourcesObj := types.ObjectValueMust(
		map[string]attr.Type{
			"cpu": types.Float64Type,
			"ram": types.Float64Type,
		},
		map[string]attr.Value{
			"cpu": types.Float64Value(cpu / 1000),
			"ram": types.Float64Value(ram / 1000),
		},
	)

	// Environment Variables
	var envList []attr.Value
	for _, env := range container.EnvironmentVariables {
		envList = append(envList, types.ObjectValueMust(
			map[string]attr.Type{
				"name":   types.StringType,
				"value":  types.StringType,
				"secret": types.BoolType,
			},
			map[string]attr.Value{
				"name":   types.StringValue(env.Name),
				"value":  types.StringPointerValue(env.Value),
				"secret": types.BoolValue(env.Secret),
			},
		))
	}
	envTF := types.ListValueMust(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":   types.StringType,
				"value":  types.StringType,
				"secret": types.BoolType,
			},
		}, envList)

	// Ports
	ports := make([]attr.Value, len(container.Ports))
	for i, p := range container.Ports {
		ports[i] = types.StringValue(p)
	}

	portList, diags := types.ListValue(types.StringType, ports)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Mounts
	var mountList []attr.Value
	for _, m := range container.Mounts {
		mountList = append(mountList, types.ObjectValueMust(
			map[string]attr.Type{
				"path":   types.StringType,
				"volume": types.StringType,
			},
			map[string]attr.Value{
				"path":   types.StringValue(m.Path),
				"volume": types.StringValue(m.Volume.Name),
			},
		))
	}
	mountTF := types.ListValueMust(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"path":   types.StringType,
				"volume": types.StringType,
			},
		}, mountList)

	// Ingresses
	var ingressList []attr.Value
	for _, ing := range container.Ingresses {
		var allowList []attr.Value
		for _, ip := range ing.Allowlist {
			allowList = append(allowList, types.StringValue(ip))
		}
		ingressList = append(ingressList, types.ObjectValueMust(
			map[string]attr.Type{
				"domain_name": types.StringType,
				"port":        types.Int64Type,
				"tls":         types.BoolType,
				"allow_list":  types.ListType{ElemType: types.StringType},
			},
			map[string]attr.Value{
				"domain_name": types.StringValue(ing.DomainName),
				"port":        types.Int64Value(int64(ing.Port)),
				"tls":         types.BoolValue(ing.EnableTLS),
				"allow_list":  types.ListValueMust(types.StringType, allowList),
			},
		))
	}
	ingressesTF := types.ListValueMust(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"domain_name": types.StringType,
				"port":        types.Int64Type,
				"tls":         types.BoolType,
				"allow_list":  types.ListType{ElemType: types.StringType},
			},
		}, ingressList)

	// Health Check
	healthTF := types.ObjectValueMust(
		map[string]attr.Type{
			"port": types.Int64Type,
			"path": types.StringType,
		},
		map[string]attr.Value{
			"port": types.Int64Value(int64(container.HealthCheck.Port)),
			"path": types.StringValue(container.HealthCheck.Path),
		},
	)

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

	state := containerResource{
		ID:                   types.StringValue(container.Name),
		Name:                 types.StringValue(container.Name),
		Namespace:            types.StringValue(namespace),
		Image:                types.StringValue(container.Image),
		Registry:             types.StringValue(container.PrivateRegistry.Name),
		Resources:            resourcesObj,
		EnvironmentVariables: envTF,
		Ports:                portList,
		Ingresses:            ingressesTF,
		Mounts:               mountTF,
		HealthCheck:          healthTF,
		Status:               types.StringValue(container.State),
		LastUpdated:          types.StringValue(time.Now().Format(time.RFC3339)),
	}

	if obj, ok := scalingObj.(types.Object); ok {
		state.Scaling = obj
	} else {
		resp.Diagnostics.AddError("Error importing container", "Could not import container: "+err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
