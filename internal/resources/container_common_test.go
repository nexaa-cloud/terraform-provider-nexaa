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

// --- parseContainerImportID ---

func Test_ParseContainerImportID_valid(t *testing.T) {
	ns, name, err := parseContainerImportID("my-namespace/my-container")
	assert.NoError(t, err)
	assert.Equal(t, "my-namespace", ns)
	assert.Equal(t, "my-container", name)
}

func Test_ParseContainerImportID_missing_slash_errors(t *testing.T) {
	_, _, err := parseContainerImportID("no-slash-here")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "namespace")
}

func Test_ParseContainerImportID_empty_namespace_errors(t *testing.T) {
	_, _, err := parseContainerImportID("/container-name")
	assert.Error(t, err)
}

func Test_ParseContainerImportID_empty_name_errors(t *testing.T) {
	_, _, err := parseContainerImportID("namespace/")
	assert.Error(t, err)
}

// --- buildMountsUpdateInput ---

func makeMountList(mounts ...map[string]string) types.List {
	elems := make([]attr.Value, len(mounts))
	for i, m := range mounts {
		elems[i] = types.ObjectValueMust(MountsObjectAttributeTypes(), map[string]attr.Value{
			"path":   types.StringValue(m["path"]),
			"volume": types.StringValue(m["volume"]),
		})
	}
	return types.ListValueMust(MountsObjectType(), elems)
}

func Test_BuildMountsUpdateInput_both_null(t *testing.T) {
	result, diags := buildMountsUpdateInput(context.Background(),
		types.ListNull(MountsObjectType()),
		types.ListNull(MountsObjectType()),
	)
	assert.False(t, diags.HasError())
	assert.Empty(t, result)
}

func Test_BuildMountsUpdateInput_new_mount_present(t *testing.T) {
	current := makeMountList(map[string]string{"path": "/data", "volume": "vol-a"})
	result, diags := buildMountsUpdateInput(context.Background(), current, types.ListNull(MountsObjectType()))
	assert.False(t, diags.HasError())
	assert.Len(t, result, 1)
	assert.Equal(t, "/data", result[0].Path)
	assert.Equal(t, api.StatePresent, result[0].State)
}

func Test_BuildMountsUpdateInput_removed_mount_marked_absent(t *testing.T) {
	previous := makeMountList(map[string]string{"path": "/old", "volume": "vol-old"})
	result, diags := buildMountsUpdateInput(context.Background(), types.ListNull(MountsObjectType()), previous)
	assert.False(t, diags.HasError())
	assert.Len(t, result, 1)
	assert.Equal(t, "/old", result[0].Path)
	assert.Equal(t, api.StateAbsent, result[0].State)
}

func Test_BuildMountsUpdateInput_unchanged_mount_stays_present(t *testing.T) {
	mount := map[string]string{"path": "/data", "volume": "vol-a"}
	current := makeMountList(mount)
	previous := makeMountList(mount)
	result, diags := buildMountsUpdateInput(context.Background(), current, previous)
	assert.False(t, diags.HasError())
	assert.Len(t, result, 1)
	assert.Equal(t, api.StatePresent, result[0].State)
}

func Test_BuildMountsUpdateInput_add_and_remove(t *testing.T) {
	current := makeMountList(map[string]string{"path": "/new", "volume": "vol-new"})
	previous := makeMountList(map[string]string{"path": "/old", "volume": "vol-old"})
	result, diags := buildMountsUpdateInput(context.Background(), current, previous)
	assert.False(t, diags.HasError())
	assert.Len(t, result, 2)

	byPath := map[string]api.MountInput{}
	for _, r := range result {
		byPath[r.Path] = r
	}
	assert.Equal(t, api.StatePresent, byPath["/new"].State)
	assert.Equal(t, api.StateAbsent, byPath["/old"].State)
}
