// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
	nexaaclient "github.com/nexaa-cloud/terraform-provider-nexaa/internal/client"
)

// MessageQueueAdminUserObjectAttributeTypes describes the read-only admin_user
// nested object: the Nexaa-managed administrative user and its credentials.
func MessageQueueAdminUserObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":     types.StringType,
		"role":     types.StringType,
		"status":   types.StringType,
		"password": types.StringType,
		"dsn":      types.StringType,
	}
}

func translateApiToMessageQueueResource(ctx context.Context, client nexaaclient.NexaaAPI, queue api.MessageQueueResult, timeout timeouts.Value) (messageQueueResource, diag.Diagnostics) {
	plan := messageQueueResource{}

	namespace := queue.GetNamespace()
	plan.ID = types.StringValue(generateMessageQueueId(namespace.GetName(), queue.GetName()))
	plan.Namespace = types.StringValue(namespace.GetName())
	plan.Name = types.StringValue(queue.GetName())
	plan.Type = types.StringValue(queue.Spec.GetType())
	plan.Version = types.StringValue(queue.Spec.GetVersion())
	plan.Plan = types.StringValue(queue.Plan.GetId())
	plan.State = types.StringValue(queue.GetState())
	plan.Locked = types.BoolValue(queue.GetLocked())
	plan.Timeouts = timeout

	adminUser, diags := buildAdminUserFromApi(client, namespace.GetName(), queue.GetName(), queue.GetAdminUser())
	if diags.HasError() {
		return plan, diags
	}
	plan.AdminUser = adminUser

	allowlist, diags := toTypesStringList(ctx, queue.Ingress.GetAllowList())
	if diags.HasError() {
		allowlist = types.ListNull(types.StringType)
	}
	plan.Allowlist = allowlist

	if queue.GetExternalConnection() == nil {
		plan.ExternalConnection = types.ObjectNull(ExternalConnectionObjectAttributeTypes())
		return plan, nil
	}
	externalConnection, diags := buildExternalConnectionFromApi(ctx, queue.GetExternalConnection().ExternalConnectionResult)
	if diags.HasError() {
		return plan, diags
	}

	plan.ExternalConnection = externalConnection

	return plan, nil
}

// buildAdminUserFromApi translates the message queue's admin user into the
// read-only admin_user object. The shared MessageQueueResult only carries the
// admin user's name/role/status; the password and dsn are fetched from the
// dedicated admin-credentials endpoint.
func buildAdminUserFromApi(client nexaaclient.NexaaAPI, namespace, queueName string, adminUser *api.MessageQueueResultAdminUserMessageQueueUser) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if adminUser == nil {
		return types.ObjectNull(MessageQueueAdminUserObjectAttributeTypes()), diags
	}

	password := types.StringNull()
	dsn := types.StringNull()
	creds, err := client.MessageQueueAdminCredentials(api.MessageQueueResourceInput{
		Name:      queueName,
		Namespace: namespace,
	}, adminUser.GetName())
	if err != nil {
		diags.AddWarning(
			"Could not fetch message queue admin credentials",
			fmt.Sprintf("The admin user %q is reported but its password/dsn could not be retrieved: %s", adminUser.GetName(), err.Error()),
		)
	} else {
		password = types.StringValue(creds.GetPassword())
		dsn = types.StringValue(creds.GetDsn())
	}

	obj, d := types.ObjectValue(MessageQueueAdminUserObjectAttributeTypes(), map[string]attr.Value{
		"name":     types.StringValue(adminUser.GetName()),
		"role":     types.StringValue(adminUser.GetRole()),
		"status":   types.StringValue(adminUser.GetStatus()),
		"password": password,
		"dsn":      dsn,
	})
	diags.Append(d...)
	return obj, diags
}

func generateMessageQueueId(namespace string, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}
