// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// --- ImmutableString ---

func Test_ImmutableString_null_state_allows_change(t *testing.T) {
	req := planmodifier.StringRequest{
		Path:       path.Root("test"),
		StateValue: types.StringNull(),
		PlanValue:  types.StringValue("new"),
	}
	var resp planmodifier.StringResponse
	ImmutableString().PlanModifyString(context.Background(), req, &resp)
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_ImmutableString_unknown_state_allows_change(t *testing.T) {
	req := planmodifier.StringRequest{
		Path:       path.Root("test"),
		StateValue: types.StringUnknown(),
		PlanValue:  types.StringValue("new"),
	}
	var resp planmodifier.StringResponse
	ImmutableString().PlanModifyString(context.Background(), req, &resp)
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_ImmutableString_equal_values_no_error(t *testing.T) {
	req := planmodifier.StringRequest{
		Path:       path.Root("test"),
		StateValue: types.StringValue("same"),
		PlanValue:  types.StringValue("same"),
	}
	var resp planmodifier.StringResponse
	ImmutableString().PlanModifyString(context.Background(), req, &resp)
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_ImmutableString_changed_value_errors(t *testing.T) {
	req := planmodifier.StringRequest{
		Path:       path.Root("test"),
		StateValue: types.StringValue("old"),
		PlanValue:  types.StringValue("new"),
	}
	var resp planmodifier.StringResponse
	ImmutableString().PlanModifyString(context.Background(), req, &resp)
	assert.True(t, resp.Diagnostics.HasError())
}

// --- ImmutableObject ---

func Test_ImmutableObject_null_state_allows_change(t *testing.T) {
	attrTypes := map[string]attr.Type{"key": types.StringType}
	req := planmodifier.ObjectRequest{
		Path:       path.Root("test"),
		StateValue: types.ObjectNull(attrTypes),
		PlanValue:  types.ObjectValueMust(attrTypes, map[string]attr.Value{"key": types.StringValue("v")}),
	}
	var resp planmodifier.ObjectResponse
	ImmutableObject().PlanModifyObject(context.Background(), req, &resp)
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_ImmutableObject_equal_values_no_error(t *testing.T) {
	attrTypes := map[string]attr.Type{"key": types.StringType}
	val := types.ObjectValueMust(attrTypes, map[string]attr.Value{"key": types.StringValue("v")})
	req := planmodifier.ObjectRequest{
		Path:       path.Root("test"),
		StateValue: val,
		PlanValue:  val,
	}
	var resp planmodifier.ObjectResponse
	ImmutableObject().PlanModifyObject(context.Background(), req, &resp)
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_ImmutableObject_changed_value_errors(t *testing.T) {
	attrTypes := map[string]attr.Type{"key": types.StringType}
	req := planmodifier.ObjectRequest{
		Path:       path.Root("test"),
		StateValue: types.ObjectValueMust(attrTypes, map[string]attr.Value{"key": types.StringValue("old")}),
		PlanValue:  types.ObjectValueMust(attrTypes, map[string]attr.Value{"key": types.StringValue("new")}),
	}
	var resp planmodifier.ObjectResponse
	ImmutableObject().PlanModifyObject(context.Background(), req, &resp)
	assert.True(t, resp.Diagnostics.HasError())
}

// --- ImmutableList ---

func Test_ImmutableList_null_state_allows_change(t *testing.T) {
	req := planmodifier.ListRequest{
		Path:       path.Root("test"),
		StateValue: types.ListNull(types.StringType),
		PlanValue:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a")}),
	}
	var resp planmodifier.ListResponse
	ImmutableList().PlanModifyList(context.Background(), req, &resp)
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_ImmutableList_equal_values_no_error(t *testing.T) {
	val := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a")})
	req := planmodifier.ListRequest{
		Path:       path.Root("test"),
		StateValue: val,
		PlanValue:  val,
	}
	var resp planmodifier.ListResponse
	ImmutableList().PlanModifyList(context.Background(), req, &resp)
	assert.False(t, resp.Diagnostics.HasError())
}

func Test_ImmutableList_changed_value_errors(t *testing.T) {
	req := planmodifier.ListRequest{
		Path:       path.Root("test"),
		StateValue: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("old")}),
		PlanValue:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("new")}),
	}
	var resp planmodifier.ListResponse
	ImmutableList().PlanModifyList(context.Background(), req, &resp)
	assert.True(t, resp.Diagnostics.HasError())
}
