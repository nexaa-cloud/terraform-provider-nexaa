// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

func translateApiToVolumeResource(plan volumeResource, volume api.VolumeResult) volumeResource {
	plan.ID = types.StringValue(generateNamespaceVolumeId(plan.Namespace.ValueString(), volume.GetName()))
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(volume.Name)
	plan.Size = types.Int64Value(int64(volume.Size))
	plan.Status = types.StringValue(volume.State)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	return plan
}
