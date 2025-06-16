// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gitlab.com/tilaa/tilaa-cli/api"

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
	_ resource.Resource = &containerResource{}
	//_ resource.ResourceWithImportState = &containerResource{}
)

// NewContainerResource is a helper function to simplify the provider implementation.
func NewContainerResource() resource.Resource {
	return &containerResource{}
}

// containerResource is the resource implementation.
type containerResource struct {
	ID                   types.String      `tfsdk:"id"`
	Name                 types.String      `tfsdk:"name"`
	Namespace            types.String      `tfsdk:"namespace"`
	Image                types.String      `tfsdk:"image"`
	Registry             types.String      `tfsdk:"registry"`
	Resources            resourcesResource `tfsdk:"resources"`
	EnvironmentVariables types.List        `tfsdk:"environment_variables"`
	Ports                types.List        `tfsdk:"ports"`
	Ingresses            types.List        `tfsdk:"ingresses"`
	Mounts               types.List        `tfsdk:"mounts"`
	HealthCheck          types.Object      `tfsdk:"health_check"`
	Scaling              types.Object      `tfsdk:"scaling"`
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
		Description: "Container resource representing a deployable service.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"namespace": schema.StringAttribute{
				Required: true,
			},
			"image": schema.StringAttribute{
				Required: true,
			},
			"registry": schema.StringAttribute{
				Optional: true,
			},
			"resources": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"cpu": schema.Float64Attribute{
						Required: true,
						Validators: []validator.Float64{
							float64validator.OneOf(enums.CPU...),
						},
					},
					"ram": schema.Float64Attribute{
						Required: true,
						Validators: []validator.Float64{
							float64validator.OneOf(enums.RAM...),
						},
					},
				},
				Required: true,
			},
			"ports": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"environment_variables": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"value": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"secret": schema.BoolAttribute{
							Optional: true,
						},
					},
				},
				Optional: true,
				Computed: true,
			},
			"ingresses": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"domain_name": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"port": schema.Int64Attribute{
							Required: true,
						},
						"tls": schema.BoolAttribute{
							Required: true,
						},
						"allow_list": schema.ListAttribute{
							ElementType: types.StringType,
							Optional:    true,
						},
					},
				},
				Optional: true,
				Computed: true,
			},
			"mounts": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Required: true,
						},
						"volume": schema.StringAttribute{
							Required: true,
						},
					},
				},
				Optional: true,
				Computed: true,
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
				Required: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required: true,
					},
					"manual_input": schema.Int64Attribute{
						Optional: true,
					},
					"auto_input": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"minimal_replicas": schema.Int64Attribute{
								Required: true,
							},
							"maximal_replicas": schema.Int64Attribute{
								Required: true,
							},
							"triggers": schema.ListNestedAttribute{
								Optional: true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Required: true,
											Validators: []validator.String{
												stringvalidator.OneOf("MEMORY", "CPU"),
											},
										},
										"threshold": schema.Int64Attribute{
											Required: true,
										},
									},
								},
							},
						},
					},
				},
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
	cpu := int(plan.Resources.CPU.ValueFloat64() * 1000)
	ram := int(plan.Resources.RAM.ValueFloat64() * 1000)
	cpuRam := fmt.Sprintf("CPU_%d_RAM_%d", cpu, ram)

	if plan.Registry.IsNull() || plan.Registry.IsUnknown() {
		plan.Registry = types.StringPointerValue(nil)
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
	plan.Registry = types.StringValue(container.PrivateRegistry.Name)

	// Parse CPU and RAM back from the resources string
	resParts := strings.Split(string(container.Resources), "_")
	if len(resParts) == 4 {
		cpu, err := strconv.ParseFloat(resParts[1], 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating container",
				"Something went wrong while creating a container",
			)
			return
		}

		ram, err := strconv.ParseFloat(resParts[3], 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating container",
				"Something went wrong while creating a container",
			)
			return
		}

		plan.Resources.CPU = types.Float64Value(cpu / 1000)
		plan.Resources.RAM = types.Float64Value(ram / 1000)
	}

	// Environment variables
	if container.EnvironmentVariables != nil {
		envVars := make([]attr.Value, len(container.EnvironmentVariables))
		for i, ev := range container.EnvironmentVariables {
			var val types.String
			if ev.Secret == false {
				val = types.StringValue(*ev.Value)
			} else {
				val = types.StringValue("******")
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

		ingressObj := types.ObjectValueMust(
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

	plan.Scaling = scalingObj.(types.Object)

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
	state.Registry = types.StringValue(container.PrivateRegistry.Name)

	// Parse CPU and RAM back from the resources string
	resParts := strings.Split(string(container.Resources), "_")
	if len(resParts) == 4 {
		cpu, err := strconv.ParseFloat(resParts[1], 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating container",
				"Something went wrong while creating a container",
			)
			return
		}

		ram, err := strconv.ParseFloat(resParts[3], 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating container",
				"Something went wrong while creating a container",
			)
			return
		}

		state.Resources.CPU = types.Float64Value(cpu / 1000)
		state.Resources.RAM = types.Float64Value(ram / 1000)
	}

	// Environment variables
	if container.EnvironmentVariables != nil {
		envVars := make([]attr.Value, len(container.EnvironmentVariables))
		for i, ev := range container.EnvironmentVariables {
			var val types.String
			if ev.Secret == false {
				val = types.StringValue(*ev.Value)
			} else {
				val = types.StringValue("******")
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

		ingressObj := types.ObjectValueMust(
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

	state.Scaling = scalingObj.(types.Object)

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
	cpu := int(plan.Resources.CPU.ValueFloat64() * 1000)
	ram := int(plan.Resources.RAM.ValueFloat64() * 1000)
	cpuRam := fmt.Sprintf("CPU_%d_RAM_%d", cpu, ram)


	if plan.Registry.IsNull() || plan.Registry.IsUnknown() {
		plan.Registry = types.StringPointerValue(nil)
	}

	// Build input struct
	input := api.ContainerModifyInput{
		Namespace: plan.Namespace.ValueString(),
		Name:      plan.Name.ValueString(),
		Image:	   plan.Image.ValueStringPointer(),
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


	resp.Diagnostics.AddWarning(
		"Value for input",
		"Value: "+ fmt.Sprintf("%#v\n", input),
	)

	// Create container
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
	plan.Registry = types.StringValue(container.PrivateRegistry.Name)

	// Parse CPU and RAM back from the resources string
	resParts := strings.Split(string(container.Resources), "_")
	if len(resParts) == 4 {
		cpu, err := strconv.ParseFloat(resParts[1], 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating container",
				"Something went wrong while creating a container",
			)
			return
		}

		ram, err := strconv.ParseFloat(resParts[3], 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating container",
				"Something went wrong while creating a container",
			)
			return
		}

		plan.Resources.CPU = types.Float64Value(cpu / 1000)
		plan.Resources.RAM = types.Float64Value(ram / 1000)
	}

	// Environment variables
	if container.EnvironmentVariables != nil {
		envVars := make([]attr.Value, len(container.EnvironmentVariables))
		for i, ev := range container.EnvironmentVariables {
			var val types.String
			if ev.Secret == false {
				val = types.StringValue(*ev.Value)
			} else {
				val = types.StringValue("******")
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

		ingressObj := types.ObjectValueMust(
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

	plan.Scaling = scalingObj.(types.Object)

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
		maxRetries   = 4
		initialDelay = 10 * time.Second
	)
	delay := initialDelay

	var err error

	// Retry DeleteContainer until it no longer complains about "locked"
	for i := 0; i <= maxRetries; i++ {
		client := api.NewClient()
		_, err = client.ContainerDelete(state.Namespace.ValueString(), state.Name.ValueString())
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

// func (r *containerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
//     resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

//     id := req.ID
//     list, err := api.ListContainers()
//     if err != nil {
//         resp.Diagnostics.AddError("Error listing containers", err.Error())
//         return
//     }
//     for _, item := range list {
//         if item.Name == id {
//             resp.State.SetAttribute(ctx, path.Root("name"), item.Name)
//             resp.State.SetAttribute(ctx, path.Root("description"), item.Description)
//             resp.State.SetAttribute(ctx, path.Root("last_updated"), time.Now().Format(time.RFC850))
//             return
//         }
//     }
//     resp.Diagnostics.AddError(
//         "Error importing container",
//         "Could not find container with name: "+id,
//     )
// }
