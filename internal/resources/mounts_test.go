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

// --- buildMountsFromApi ---

func Test_BuildMountsFromApi_empty(t *testing.T) {
	result, diags := buildMountsFromApi([]api.ContainerMounts{})
	assert.False(t, diags.HasError())
	assert.Equal(t, 0, len(result.Elements()))
}

func Test_BuildMountsFromApi_single_mount(t *testing.T) {
	mounts := []api.ContainerMounts{
		{Path: "/data", Volume: api.ContainerMountsVolume{Name: "my-vol"}},
	}
	result, diags := buildMountsFromApi(mounts)
	assert.False(t, diags.HasError())
	assert.Equal(t, 1, len(result.Elements()))

	expected := types.ListValueMust(MountsObjectType(), []attr.Value{
		types.ObjectValueMust(MountsObjectAttributeTypes(), map[string]attr.Value{
			"path":   types.StringValue("/data"),
			"volume": types.StringValue("my-vol"),
		}),
	})
	assert.Equal(t, expected, result)
}

func Test_BuildMountsFromApi_multiple_mounts(t *testing.T) {
	mounts := []api.ContainerMounts{
		{Path: "/data", Volume: api.ContainerMountsVolume{Name: "vol-1"}},
		{Path: "/logs", Volume: api.ContainerMountsVolume{Name: "vol-2"}},
	}
	result, diags := buildMountsFromApi(mounts)
	assert.False(t, diags.HasError())
	assert.Equal(t, 2, len(result.Elements()))
}

// --- buildMountsInput ---

func Test_BuildMountsInput_null_list(t *testing.T) {
	result, diags := buildMountsInput(context.Background(), types.ListNull(MountsObjectType()))
	assert.False(t, diags.HasError())
	assert.Empty(t, result)
}

func Test_BuildMountsInput_single_mount(t *testing.T) {
	list := types.ListValueMust(MountsObjectType(), []attr.Value{
		types.ObjectValueMust(MountsObjectAttributeTypes(), map[string]attr.Value{
			"path":   types.StringValue("/data"),
			"volume": types.StringValue("my-vol"),
		}),
	})
	result, diags := buildMountsInput(context.Background(), list)
	assert.False(t, diags.HasError())
	assert.Len(t, result, 1)
	assert.Equal(t, "/data", result[0].Path)
	assert.Equal(t, "my-vol", result[0].Volume.Name)
	assert.Equal(t, api.StatePresent, result[0].State)
}

func Test_BuildMountsInput_multiple_mounts(t *testing.T) {
	list := types.ListValueMust(MountsObjectType(), []attr.Value{
		types.ObjectValueMust(MountsObjectAttributeTypes(), map[string]attr.Value{
			"path":   types.StringValue("/data"),
			"volume": types.StringValue("vol-1"),
		}),
		types.ObjectValueMust(MountsObjectAttributeTypes(), map[string]attr.Value{
			"path":   types.StringValue("/logs"),
			"volume": types.StringValue("vol-2"),
		}),
	})
	result, diags := buildMountsInput(context.Background(), list)
	assert.False(t, diags.HasError())
	assert.Len(t, result, 2)
	assert.Equal(t, "/data", result[0].Path)
	assert.Equal(t, "/logs", result[1].Path)
}
