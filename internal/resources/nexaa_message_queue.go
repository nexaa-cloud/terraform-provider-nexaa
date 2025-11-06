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
	_ resource.Resource                = &messageQueueResource{}
	_ resource.ResourceWithImportState = &messageQueueResource{}
)

// NewMessageQueueResource is a helper function to simplify the provider implementation.
func NewMessageQueueResource() resource.Resource {
	return &messageQueueResource{}
}

// messageQueueResource is the resource implementation.
type messageQueueResource struct {
	ID          types.String `tfsdk:"id"`
	Namespace   types.String `tfsdk:"namespace"`
	Name        types.String `tfsdk:"name"`
	Plan        types.String `tfsdk:"plan"`
	Type        types.String `tfsdk:"type"`
	Version     types.String `tfsdk:"version"`
	State       types.String `tfsdk:"state"`
	Locked      types.Bool   `tfsdk:"locked"`
	LastUpdated types.String `tfsdk:"last_updated"`
	Allowlist   types.List   `tfsdk:"allowlist"`
}

// Metadata returns the resource type name.
func (r *messageQueueResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_message_queue"
}

// Schema defines the schema for the resource.
func (r *messageQueueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Description: "List of IP addresses allowed to access the message queue",
				ElementType: types.StringType,
				Optional:    true,
			},
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

	// Convert allowlist from Terraform types.List to []AllowListInput
	var allowList []api.AllowListInput
	if !plan.Allowlist.IsNull() && !plan.Allowlist.IsUnknown() {
		var allowlistIPs []string
		diags = plan.Allowlist.ElementsAs(ctx, &allowlistIPs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, ip := range allowlistIPs {
			allowList = append(allowList, api.AllowListInput{
				Ip:    ip,
				State: api.StatePresent,
			})
		}
	}

	input := api.MessageQueueCreateInput{
		Name:      plan.Name.ValueString(),
		Namespace: plan.Namespace.ValueString(),
		Plan:      plan.Plan.ValueString(),
		Spec: api.MessageQueueSpecInput{
			Type:    plan.Type.ValueString(),
			Version: plan.Version.ValueString(),
		},
		AllowList: allowList,
	}

	client := api.NewClient()

	const (
		maxRetries   = 4
		initialDelay = 3 * time.Second
	)
	delay := initialDelay
	var err error
	var queue api.MessageQueueResult

	for i := 0; i <= maxRetries; i++ {
		queue, err = client.MessageQueueCreate(input)
		if err == nil {
			break
		}

		time.Sleep(delay)
		delay *= 2
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating message queue",
			"Could not create message queue, error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", plan.Namespace.ValueString(), queue.Name))
	plan.Namespace = types.StringValue(queue.Namespace.Name)
	plan.Name = types.StringValue(queue.Name)
	plan.Plan = types.StringValue(plan.Plan.ValueString())
	plan.Type = types.StringValue(plan.Type.ValueString())
	plan.Version = types.StringValue(plan.Version.ValueString())
	plan.State = types.StringValue(queue.State)
	plan.Locked = types.BoolValue(queue.Locked)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
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

	state.ID = types.StringValue(fmt.Sprintf("%s/%s", queue.Namespace.Name, queue.Name))
	state.Namespace = types.StringValue(queue.Namespace.Name)
	state.Name = types.StringValue(queue.Name)
	state.State = types.StringValue(queue.State)
	state.Locked = types.BoolValue(queue.Locked)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *messageQueueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"You can't update a message queue",
		"You can't change a message queue. You can only create and delete a message queue",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *messageQueueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state messageQueueResource
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

	// Retry DeleteMessageQueue with context timeout
	for i := 0; i <= maxRetries; i++ {
		// Check context timeout
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError(
				"Timeout deleting message queue",
				fmt.Sprintf("Context timeout while waiting to delete message queue %q", state.Name.ValueString()),
			)
			return
		default:
		}

		input := api.MessageQueueResourceInput{
			Name:      state.Name.ValueString(),
			Namespace: state.Namespace.ValueString(),
		}

		queue, err := client.MessageQueueGet(input)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error fetching message queue",
				fmt.Sprintf("Could not find message queue with name %q: %s", state.Name.ValueString(), err.Error()),
			)
			return
		}

		if queue.State == "created" {
			deleteInput := api.ResourceNameInput{
				Name:      state.Name.ValueString(),
				Namespace: state.Namespace.ValueString(),
			}
			_, err := client.MessageQueueDelete(deleteInput)

			if err != nil {
				resp.Diagnostics.AddError(
					"Error deleting message queue",
					fmt.Sprintf("Failed to delete message queue %q: %s", state.Name.ValueString(), err.Error()),
				)
				return
			}
			return
		}
		if queue.State == "failed" && queue.Locked {
			resp.Diagnostics.AddError(
				"Error deleting message queue",
				fmt.Sprintf("Failed to delete message queue %q, the message queue is locked and could not be deleted", state.Name.ValueString()),
			)
			return
		}

		// Sleep with context timeout
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError(
				"Timeout deleting message queue",
				fmt.Sprintf("Context timeout while waiting to delete message queue %q", state.Name.ValueString()),
			)
			return
		case <-time.After(delay):
		}
		delay *= 2
	}

	// If we reach here, we exhausted all retries without successfully deleting
	resp.Diagnostics.AddError(
		"Timeout waiting for message queue to unlock",
		"Message queue could not be deleted after retries.",
	)
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

	// Fetch the message queue using the namespace and queue name
	input := api.MessageQueueResourceInput{
		Name:      queueName,
		Namespace: ns,
	}
	queue, err := client.MessageQueueGet(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Message Queue",
			"Could not read message queue "+queueName+": "+err.Error(),
		)
		return
	}

	// Set the message queue attributes in the state
	resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%s/%s", ns, queue.Name))
	resp.State.SetAttribute(ctx, path.Root("namespace"), ns)
	resp.State.SetAttribute(ctx, path.Root("name"), queue.Name)
	resp.State.SetAttribute(ctx, path.Root("state"), queue.State)
	resp.State.SetAttribute(ctx, path.Root("locked"), queue.Locked)
	resp.State.SetAttribute(ctx, path.Root("last_updated"), time.Now().Format(time.RFC850))
}
