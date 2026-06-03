// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// --- toStringArray ---

func Test_ToStringArray_null_list_returns_empty(t *testing.T) {
	result := toStringArray(context.Background(), types.ListNull(types.StringType))
	assert.Empty(t, result)
}

func Test_ToStringArray_unknown_list_returns_empty(t *testing.T) {
	result := toStringArray(context.Background(), types.ListUnknown(types.StringType))
	assert.Empty(t, result)
}

func Test_ToStringArray_normal_list_returns_sorted(t *testing.T) {
	list := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("c"),
		types.StringValue("a"),
		types.StringValue("b"),
	})
	result := toStringArray(context.Background(), list)
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func Test_ToStringArray_single_element(t *testing.T) {
	list := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("only"),
	})
	result := toStringArray(context.Background(), list)
	assert.Equal(t, []string{"only"}, result)
}

// --- toTypesStringList ---

func Test_ToTypesStringList_empty_slice(t *testing.T) {
	result, diags := toTypesStringList(context.Background(), []string{})
	assert.False(t, diags.HasError())
	assert.Equal(t, 0, len(result.Elements()))
}

func Test_ToTypesStringList_non_empty_slice(t *testing.T) {
	result, diags := toTypesStringList(context.Background(), []string{"x", "y"})
	assert.False(t, diags.HasError())
	assert.Equal(t, 2, len(result.Elements()))
}
