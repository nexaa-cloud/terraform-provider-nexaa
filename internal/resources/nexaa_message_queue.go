// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &messageQueueResource{}
	_ resource.ResourceWithImportState = &messageQueueResource{}
)

// NewMessageQueueResource is a helper function to simplify the provider implementation.
func NewMessageQueueResource() resource.Resource {
	return &messageQueueResource{}
}

// messageQueueResource is the resource implementation.
type messageQueueResource struct {
	ID          		types.String `tfsdk:"id"`
	Namespace   		types.String `tfsdk:"namespace"`
	Name        		types.String `tfsdk:"name"`
	Plan        		types.String `tfsdk:"plan"`
	Type        		types.String `tfsdk:"type"`
	Version     		types.String `tfsdk:"version"`
	ExternalConnection	types.Object `tfsdk:"external_connection"`
	State       		types.String `tfsdk:"state"`
	Locked      		types.Bool   `tfsdk:"locked"`
	LastUpdated 		types.String `tfsdk:"last_updated"`
	Allowlist   		types.List   `tfsdk:"allowlist"`
	Timeouts    		timeouts.Value `tfsdk:"timeouts"`
}

type messageQueueExternalConnectionResource struct {
	Ipv6  types.String `tfsdk:"ipv6"`
	Ipv4  types.String `tfsdk:"ipv4"`
	Ports types.Object `tfsdk:"ports"`
}

type messageQueueExternalConnectionPortsResource struct {
	ExternalPort types.Int64 `tfsdk:"external_port"`
	Allowlist    types.List  `tfsdk:"allowlist"`
}

// Metadata returns the resource type name.
func (r *messageQueueResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_message_queue"
}

// Schema defines the schema for the resource.
func (r *messageQueueResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Message Queue resource representing a managed message queue (e.g., RabbitMQ) on Nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the message queue in the format 'namespace/name'",
				Computed:    true,
			},
			"namespace": schema.StringAttribute{
				Description: "Name of the namespace the message queue belongs to",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the message queue",
				Required:    true,
			},
			"plan": schema.StringAttribute{
				Description: "The plan ID for the message queue",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of message queue (e.g., 'RabbitMQ')",
				Required:    true,
			},
			"version": schema.StringAttribute{
				Description: "The version of the message queue software",
				Required:    true,
			},
			"external_connection": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"ipv4": schema.StringAttribute{
						Computed:    true,
						Description: "The ipv4 address that can be used in combination with the external port to connect to your queue",
					},
					"ipv6": schema.StringAttribute{
						Computed:    true,
						Description: "The ipv6 address that can be used in combination with the external port to connect to your queue",
					},
					"ports": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"external_port": schema.Int64Attribute{
								Computed:    true,
								Description: "The port that is used in combination with your ipv4 or ipv6 address to connect to your queue",
							},
							"allowlist": schema.ListAttribute{
								ElementType: types.StringType,
								Optional:    true,
								Computed: 	 true,
								Description: "A list with the IP's that can access the message queue through the external connection, can be in ipv4 and/or ipv6 format. Defaults to 0.0.0.0/0 and ::/0, which means that the message queue can be accessed from any IP address.",
								Default: listdefault.StaticValue(
									types.ListValueMust(types.StringType, []attr.Value{
										types.StringValue("0.0.0.0/0"),
										types.StringValue("::/0"),
									}),
								),
								PlanModifiers: []planmodifier.List{
									listplanmodifier.UseStateForUnknown(),
								},
							},
						},
						Optional:    true,
						Description: "Used to define the connection parts of the external connection",
					},
				},
				Optional:    true,
				Description: "An external connection that can used to connect to a message queue",
			},
			"state": schema.StringAttribute{
				Description: "The current state of the message queue",
				Computed:    true,
			},
			"locked": schema.BoolAttribute{
				Description: "If the message queue is locked it can't be deleted",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the message queue",
				Computed:    true,
			},
			"allowlist": schema.ListAttribute{
				Description: "List of IP addresses allowed to access the management console of the message queue (defaults: '0.0.0.0/0' and '::/0')",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
                Default: listdefault.StaticValue(
                    types.ListValueMust(types.StringType, []attr.Value{
                        types.StringValue("0.0.0.0/0"),
                        types.StringValue("::/0"),
                    }),
                ),
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
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
func (r *messageQueueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan messageQueueResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 2*time.Minute)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()


	allowlist:= buildAllowlistInput(ctx, nil, plan.Allowlist)

	input := api.MessageQueueCreateInput{
		Name:      plan.Name.ValueString(),
		Namespace: plan.Namespace.ValueString(),
		Plan:      plan.Plan.ValueString(),
		ExternalConnection: buildExternalConnectionUpdateInputMQ(ctx, plan, nil),
		Spec: api.MessageQueueSpecInput{
			Type:    plan.Type.ValueString(),
			Version: plan.Version.ValueString(),
		},
		AllowList: allowlist,
	}

	client := api.NewClient()

	_, err := client.MessageQueueCreate(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating message queue",
			fmt.Sprintf("Failed to create message queue %q: %s", plan.Name.ValueString(), err.Error()),
		)
		return
	}

	err = waitForUnlocked(ctx, messageQueueLocked(), *client, plan.Namespace.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating message queue",
			fmt.Sprintf("Failed to wait for message queue %q to unlock: %s", plan.Name.ValueString(), err.Error()),
		)
		return
	}

	queue, err := client.MessageQueueGet(api.MessageQueueResourceInput{
		Name:      plan.Name.ValueString(),
		Namespace: plan.Namespace.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching created message queue",
			err.Error(),
		)
		return
	}

	plan, diags = translateApiToMessageQueueResource(ctx, queue, plan.Timeouts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *messageQueueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state messageQueueResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()

	input := api.MessageQueueResourceInput{
		Name:      state.Name.ValueString(),
		Namespace: state.Namespace.ValueString(),
	}

	queue, err := client.MessageQueueGet(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Message Queue",
			"Could not read message queue with name "+state.Name.ValueString()+", error: "+err.Error(),
		)
		return
	}

	state, diags = translateApiToMessageQueueResource(ctx, queue, state.Timeouts)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *messageQueueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan messageQueueResource
	var state messageQueueResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, 2*time.Minute)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	client := api.NewClient()

	allowList := buildAllowlistInput(ctx, &state.Allowlist, plan.Allowlist)

	input := api.MessageQueueModifyInput{
		Name:      plan.Name.ValueString(),
		Namespace: plan.Namespace.ValueString(),
		AllowList: allowList,
		ExternalConnection: buildExternalConnectionUpdateInputMQ(ctx, plan, &state),
	}

	_, err := client.MessageQueueModify(input)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating message queue",
			fmt.Sprintf("Failed to update message queue %q: %s", state.Name.ValueString(), err.Error()),
		)
		return
	}

	err = waitForUnlocked(ctx, messageQueueLocked(), *client, plan.Namespace.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for message queue to unlock",
			fmt.Sprintf("Failed to wait for message queue %q to unlock: %s", state.Name.ValueString(), err.Error()),
		)
		return
	}

	queue, err := client.MessageQueueGet(api.MessageQueueResourceInput{
		Name:      plan.Name.ValueString(),
		Namespace: plan.Namespace.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching updated message queue",
			err.Error(),
		)
		return
	}

	plan, diags = translateApiToMessageQueueResource(ctx, queue, plan.Timeouts)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *messageQueueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state messageQueueResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, 2*time.Minute)
	
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	client := api.NewClient()
	err := waitForUnlocked(ctx, messageQueueLocked(), *client, state.Namespace.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for message queue to unlock",
			fmt.Sprintf("Failed to wait for message queue %q to unlock: %s", state.Name.ValueString(), err.Error()),
		)
		return
	}

	input := api.MessageQueueResourceInput{
		Name:      state.Name.ValueString(),
		Namespace: state.Namespace.ValueString(),
	}

	_, err = client.MessageQueueDelete(input)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting message queue",
			fmt.Sprintf("Failed to delete message queue %q: %s", state.Name.ValueString(), err.Error()),
		)
		return
	}
}

// ImportState implements resource.ResourceWithImportState.
func (r *messageQueueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expect import ID as "namespace/queueName"
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format \"<namespace>/<queue_name>\", got: "+req.ID,
		)
		return
	}
	ns := parts[0]
	queueName := parts[1]

	client := api.NewClient()
	input := api.MessageQueueResourceInput{
		Name:      queueName,
		Namespace: ns,
	}
	queue, err := client.MessageQueueGet(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing Message Queue",
			"Could not list message queue "+queueName+": "+err.Error(),
		)
		return
	}

	var plan messageQueueResource

	plan.Timeouts = timeouts.Value{
		Object: types.ObjectValueMust(
			map[string]attr.Type{
				"create": types.StringType,
				"update": types.StringType,
				"delete": types.StringType,
			},
			map[string]attr.Value{
				"create": types.StringValue("2m"),
				"update": types.StringValue("2m"),
				"delete": types.StringValue("2m"),
			},
		),
	}

	plan, diags := translateApiToMessageQueueResource(ctx, queue, plan.Timeouts)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
