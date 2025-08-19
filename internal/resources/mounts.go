// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

func MountsObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"path":   types.StringType,
		"volume": types.StringType,
	}
}

func MountsObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: MountsObjectAttributeTypes()}
}

func buildMountsFromApi(mounts []api.ContainerMounts) (basetypes.ListValue, diag.Diagnostics) {
	result := make([]attr.Value, len(mounts))
	for i, m := range mounts {
		obj := types.ObjectValueMust(
			MountsObjectAttributeTypes(),
			map[string]attr.Value{
				"path":   types.StringValue(m.Path),
				"volume": types.StringValue(m.Volume.Name),
			})
		result[i] = obj
	}

	return types.ListValue(
		MountsObjectType(),
		result,
	)
}
