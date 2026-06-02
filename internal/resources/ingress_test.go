// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
	"github.com/stretchr/testify/assert"
)

func makeAPIIngress(domain string, port int, state string) api.ContainerResultIngressesIngress {
	return api.ContainerResultIngressesIngress{
		DomainName: domain,
		Port:       port,
		EnableTLS:  false,
		Allowlist:  []string{"0.0.0.0/0"},
		State:      state,
	}
}

func makeContainerResult(ingresses ...api.ContainerResultIngressesIngress) api.ContainerResult {
	return api.ContainerResult{Ingresses: ingresses}
}

// --- buildIngressesFromApi ---

func Test_BuildIngressesFromApi_empty(t *testing.T) {
	result, diags := buildIngressesFromApi(makeContainerResult())
	assert.False(t, diags.HasError())
	assert.Equal(t, 0, len(result.Elements()))
}

func Test_BuildIngressesFromApi_normal_ingress_included(t *testing.T) {
	result, diags := buildIngressesFromApi(makeContainerResult(
		makeAPIIngress("app.example.com", 80, "present"),
	))
	assert.False(t, diags.HasError())
	assert.Equal(t, 1, len(result.Elements()))
}

func Test_BuildIngressesFromApi_to_be_deleted_skipped(t *testing.T) {
	result, diags := buildIngressesFromApi(makeContainerResult(
		makeAPIIngress("app.example.com", 80, "present"),
		makeAPIIngress("old.example.com", 443, "to_be_deleted"),
	))
	assert.False(t, diags.HasError())
	assert.Equal(t, 1, len(result.Elements()))
}

func Test_BuildIngressesFromApi_deleting_skipped(t *testing.T) {
	result, diags := buildIngressesFromApi(makeContainerResult(
		makeAPIIngress("deleting.example.com", 80, "deleting"),
	))
	assert.False(t, diags.HasError())
	assert.Equal(t, 0, len(result.Elements()))
}

func Test_BuildIngressesFromApi_multiple_mixed_states(t *testing.T) {
	result, diags := buildIngressesFromApi(makeContainerResult(
		makeAPIIngress("keep-1.example.com", 80, "present"),
		makeAPIIngress("skip-1.example.com", 443, "to_be_deleted"),
		makeAPIIngress("keep-2.example.com", 8080, "creating"),
		makeAPIIngress("skip-2.example.com", 8443, "deleting"),
	))
	assert.False(t, diags.HasError())
	assert.Equal(t, 2, len(result.Elements()))
}

// --- buildIngressesFromApiInPlanOrder ---

func makeKnownIngressList(domains ...string) types.List {
	elems := make([]attr.Value, len(domains))
	for i, d := range domains {
		allowlist := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("0.0.0.0/0")})
		elems[i] = types.ObjectValueMust(IngressObjectAttributeTypes(), map[string]attr.Value{
			"domain_name": types.StringValue(d),
			"port":        types.Int64Value(80),
			"tls":         types.BoolValue(false),
			"allowlist":   allowlist,
		})
	}
	return types.ListValueMust(IngressObjectType(), elems)
}

func Test_BuildIngressesFromApiInPlanOrder_null_known_falls_back(t *testing.T) {
	cr := makeContainerResult(makeAPIIngress("a.example.com", 80, "present"))
	result, diags := buildIngressesFromApiInPlanOrder(context.Background(), cr, types.ListNull(IngressObjectType()))
	assert.False(t, diags.HasError())
	assert.Equal(t, 1, len(result.Elements()))
}

func Test_BuildIngressesFromApiInPlanOrder_unknown_domain_falls_back_to_plain_order(t *testing.T) {
	allowlist := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("0.0.0.0/0")})
	unknownElem := types.ObjectValueMust(IngressObjectAttributeTypes(), map[string]attr.Value{
		"domain_name": types.StringUnknown(),
		"port":        types.Int64Value(80),
		"tls":         types.BoolValue(false),
		"allowlist":   allowlist,
	})
	knownList := types.ListValueMust(IngressObjectType(), []attr.Value{unknownElem})

	cr := makeContainerResult(makeAPIIngress("a.example.com", 80, "present"))
	result, diags := buildIngressesFromApiInPlanOrder(context.Background(), cr, knownList)
	assert.False(t, diags.HasError())
	assert.Equal(t, 1, len(result.Elements()))
}

func Test_BuildIngressesFromApiInPlanOrder_respects_plan_order(t *testing.T) {
	cr := makeContainerResult(
		makeAPIIngress("b.example.com", 443, "present"),
		makeAPIIngress("a.example.com", 80, "present"),
	)
	known := makeKnownIngressList("a.example.com", "b.example.com")
	result, diags := buildIngressesFromApiInPlanOrder(context.Background(), cr, known)
	assert.False(t, diags.HasError())
	elems := result.Elements()
	assert.Equal(t, 2, len(elems))
	first := elems[0].(types.Object)
	assert.Equal(t, types.StringValue("a.example.com"), first.Attributes()["domain_name"])
}

func Test_BuildIngressesFromApiInPlanOrder_api_only_ingress_appended(t *testing.T) {
	cr := makeContainerResult(
		makeAPIIngress("a.example.com", 80, "present"),
		makeAPIIngress("new.example.com", 9000, "present"),
	)
	known := makeKnownIngressList("a.example.com")
	result, diags := buildIngressesFromApiInPlanOrder(context.Background(), cr, known)
	assert.False(t, diags.HasError())
	assert.Equal(t, 2, len(result.Elements()))
}

func Test_BuildIngressesFromApiInPlanOrder_plan_entry_missing_from_api_dropped(t *testing.T) {
	cr := makeContainerResult(makeAPIIngress("a.example.com", 80, "present"))
	// plan references two domains but only one is in the API response
	known := makeKnownIngressList("a.example.com", "gone.example.com")
	result, diags := buildIngressesFromApiInPlanOrder(context.Background(), cr, known)
	assert.False(t, diags.HasError())
	assert.Equal(t, 1, len(result.Elements()))
}
