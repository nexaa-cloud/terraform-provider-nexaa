// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"

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
		"allowlist":   types.ListType{ElemType: types.StringType},
	}
}

func IngressObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: IngressObjectAttributeTypes()}
}

func buildIngressElem(ing api.ContainerResultIngressesIngress) (attr.Value, diag.Diagnostics) {
	allowListElems := make([]attr.Value, len(ing.Allowlist))
	for i, a := range ing.Allowlist {
		allowListElems[i] = types.StringValue(a)
	}
	allowList, diags := types.ListValue(types.StringType, allowListElems)
	if diags.HasError() {
		return nil, diags
	}
	return types.ObjectValueMust(
		IngressObjectAttributeTypes(),
		map[string]attr.Value{
			"domain_name": types.StringValue(ing.DomainName),
			"port":        types.Int64Value(int64(ing.Port)),
			"tls":         types.BoolValue(ing.EnableTLS),
			"allowlist":   allowList,
		},
	), diags
}

func buildIngressesFromApi(containerResult api.ContainerResult) (types.List, diag.Diagnostics) {
	var ingressElems []attr.Value
	for _, ing := range containerResult.Ingresses {
		if ing.State == "to_be_deleted" || ing.State == "deleting" {
			continue
		}
		elem, diags := buildIngressElem(ing)
		if diags.HasError() {
			return types.ListNull(IngressObjectType()), diags
		}
		ingressElems = append(ingressElems, elem)
	}

	ingressesList, diags := types.ListValue(IngressObjectType(), ingressElems)
	if diags.HasError() {
		return types.ListNull(IngressObjectType()), diags
	}
	return ingressesList, diags
}

// buildIngressesFromApiInPlanOrder returns API ingresses in the same order as knownIngresses,
// matched by domain_name. Falls back to plain API order when any domain_name is unknown.
func buildIngressesFromApiInPlanOrder(ctx context.Context, containerResult api.ContainerResult, knownIngresses types.List) (types.List, diag.Diagnostics) {
	if knownIngresses.IsNull() || knownIngresses.IsUnknown() {
		return buildIngressesFromApi(containerResult)
	}

	var knownData []ingresResource
	diags := knownIngresses.ElementsAs(ctx, &knownData, false)
	if diags.HasError() {
		return types.ListNull(IngressObjectType()), diags
	}

	for _, k := range knownData {
		if k.DomainName.IsNull() || k.DomainName.IsUnknown() {
			return buildIngressesFromApi(containerResult)
		}
	}

	apiByDomain := make(map[string]api.ContainerResultIngressesIngress)
	for _, ing := range containerResult.Ingresses {
		if ing.State == "to_be_deleted" || ing.State == "deleting" {
			continue
		}
		apiByDomain[ing.DomainName] = ing
	}

	seen := make(map[string]bool)
	var ingressElems []attr.Value

	for _, known := range knownData {
		domain := known.DomainName.ValueString()
		apiIng, ok := apiByDomain[domain]
		if !ok {
			continue
		}
		seen[domain] = true
		elem, d := buildIngressElem(apiIng)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(IngressObjectType()), diags
		}
		ingressElems = append(ingressElems, elem)
	}

	for domain, apiIng := range apiByDomain {
		if seen[domain] {
			continue
		}
		elem, d := buildIngressElem(apiIng)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(IngressObjectType()), diags
		}
		ingressElems = append(ingressElems, elem)
	}

	list, d := types.ListValue(IngressObjectType(), ingressElems)
	diags.Append(d...)
	return list, diags
}
