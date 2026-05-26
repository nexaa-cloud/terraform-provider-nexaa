// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func makeIngressObject(domainName *string, port int64) types.Object {
	var domain attr.Value
	if domainName == nil {
		domain = types.StringNull()
	} else {
		domain = types.StringValue(*domainName)
	}
	allowlist := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("0.0.0.0/0"),
		types.StringValue("::/0"),
	})
	return types.ObjectValueMust(IngressObjectAttributeTypes(), map[string]attr.Value{
		"domain_name": domain,
		"port":        types.Int64Value(port),
		"tls":         types.BoolValue(true),
		"allowlist":   allowlist,
	})
}

func makeIngressObjectUnknownDomain(port int64) types.Object {
	allowlist := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("0.0.0.0/0"),
		types.StringValue("::/0"),
	})
	return types.ObjectValueMust(IngressObjectAttributeTypes(), map[string]attr.Value{
		"domain_name": types.StringUnknown(),
		"port":        types.Int64Value(port),
		"tls":         types.BoolValue(true),
		"allowlist":   allowlist,
	})
}

func strPtr(s string) *string { return &s }

func runIngressValidator(t *testing.T, elems []attr.Value) validator.ListResponse {
	t.Helper()
	list, diags := types.ListValue(IngressObjectType(), elems)
	assert.False(t, diags.HasError(), "failed to build ingress list")

	req := validator.ListRequest{ConfigValue: list}
	var resp validator.ListResponse
	noDuplicateDefaultIngressValidator{}.ValidateList(context.Background(), req, &resp)
	return resp
}

func Test_NoDuplicateDefaultIngress_null_list(t *testing.T) {
	req := validator.ListRequest{ConfigValue: types.ListNull(IngressObjectType())}
	var resp validator.ListResponse
	noDuplicateDefaultIngressValidator{}.ValidateList(context.Background(), req, &resp)
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_NoDuplicateDefaultIngress_single_null_domain(t *testing.T) {
	resp := runIngressValidator(t, []attr.Value{
		makeIngressObject(nil, 80),
	})
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_NoDuplicateDefaultIngress_two_explicit_domains(t *testing.T) {
	resp := runIngressValidator(t, []attr.Value{
		makeIngressObject(strPtr("a.example.com"), 80),
		makeIngressObject(strPtr("b.example.com"), 443),
	})
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_NoDuplicateDefaultIngress_one_null_one_explicit(t *testing.T) {
	resp := runIngressValidator(t, []attr.Value{
		makeIngressObject(nil, 80),
		makeIngressObject(strPtr("b.example.com"), 443),
	})
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_NoDuplicateDefaultIngress_two_null_domains(t *testing.T) {
	resp := runIngressValidator(t, []attr.Value{
		makeIngressObject(nil, 80),
		makeIngressObject(nil, 443),
	})
	assert.True(t, resp.Diagnostics.HasError())
}

func Test_NoDuplicateDefaultIngress_three_ingresses_two_null(t *testing.T) {
	resp := runIngressValidator(t, []attr.Value{
		makeIngressObject(strPtr("a.example.com"), 80),
		makeIngressObject(nil, 443),
		makeIngressObject(nil, 8080),
	})
	assert.True(t, resp.Diagnostics.HasError())
}

func Test_NoDuplicateDefaultIngress_two_same_explicit_domains(t *testing.T) {
	resp := runIngressValidator(t, []attr.Value{
		makeIngressObject(strPtr("a.example.com"), 80),
		makeIngressObject(strPtr("a.example.com"), 443),
	})
	assert.True(t, resp.Diagnostics.HasError())
}

func Test_NoDuplicateDefaultIngress_two_unknown_domains(t *testing.T) {
	resp := runIngressValidator(t, []attr.Value{
		makeIngressObjectUnknownDomain(80),
		makeIngressObjectUnknownDomain(443),
	})
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_NoDuplicateDefaultIngress_unknown_and_explicit_domain(t *testing.T) {
	resp := runIngressValidator(t, []attr.Value{
		makeIngressObjectUnknownDomain(80),
		makeIngressObject(strPtr("b.example.com"), 443),
	})
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_NoDuplicateDefaultIngress_unknown_does_not_count_as_null(t *testing.T) {
	resp := runIngressValidator(t, []attr.Value{
		makeIngressObjectUnknownDomain(80),
		makeIngressObject(nil, 443),
	})
	assert.False(t, resp.Diagnostics.HasError())
}
