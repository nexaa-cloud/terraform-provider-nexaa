// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

func IngressObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"domain_name": types.StringType,
		"port":        types.Int64Type,
		"tls":         types.BoolType,
		"allow_list":  types.ListType{ElemType: types.StringType},
	}
}

func IngressObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: IngressObjectAttributeTypes()}
}

func buildIngressesFromApi(containerResult api.ContainerResult) (types.List, diag.Diagnostics) {
	var ingressElems []attr.Value
	for _, ing := range containerResult.Ingresses {

		if ing.State == "to_be_deleted" || ing.State == "deleting" {
			// Continue if ingress is getting deleted
			continue
		}

		allowListElems := make([]attr.Value, len(ing.Allowlist))
		for i, a := range ing.Allowlist {
			allowListElems[i] = types.StringValue(a)
		}
		allowList, diags := types.ListValue(
			types.StringType,
			allowListElems,
		)

		if diags.HasError() {
			return types.ListNull(IngressObjectType()), diags
		}

		var ingDomain types.String

		if containerResult.Ingresses == nil {
			ingDomain = types.StringNull()
		} else {
			ingDomain = types.StringValue(ing.DomainName)
		}

		ingressObj := types.ObjectValueMust(
			IngressObjectAttributeTypes(),
			map[string]attr.Value{
				"domain_name": ingDomain,
				"port":        types.Int64Value(int64(ing.Port)),
				"tls":         types.BoolValue(ing.EnableTLS),
				"allow_list":  allowList,
			})
		ingressElems = append(ingressElems, ingressObj)
	}

	ingressesList, diags := types.ListValue(
		IngressObjectType(),
		ingressElems,
	)

	if diags.HasError() {
		return types.ListNull(IngressObjectType()), diags
	}

	return ingressesList, diags
}
