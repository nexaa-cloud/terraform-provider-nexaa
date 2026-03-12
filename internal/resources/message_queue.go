// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
)


func translateApiToMessageQueueResource(ctx context.Context, queue api.MessageQueueResult, timeout timeouts.Value) (messageQueueResource, diag.Diagnostics) {
	plan := messageQueueResource{}

	namespace := queue.GetNamespace()
	plan.ID = types.StringValue(generateMessageQueueId(namespace.GetName(), queue.GetName()))
	plan.Namespace = types.StringValue(namespace.GetName())
	plan.Name = types.StringValue(queue.GetName())
	plan.Type = types.StringValue(queue.Spec.GetType()) 
	plan.Version = types.StringValue(queue.Spec.GetVersion()) 
	plan.Plan = types.StringValue(queue.Plan.GetId()) 
	plan.State = types.StringValue(queue.GetState())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.Locked = types.BoolValue(queue.GetLocked())
	plan.Timeouts = timeout

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

func generateMessageQueueId(namespace string, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}





