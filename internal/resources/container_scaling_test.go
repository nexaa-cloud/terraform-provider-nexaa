// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

var emptyObjectAttrTypes = map[string]attr.Type{}
var emptyObjectAttrValues = map[string]attr.Value{}

func scalingManual(replicas int64) scalingResource {
	return scalingResource{
		Type:        types.StringValue("manual"),
		Manualinput: types.Int64Value(replicas),
		AutoInput:   types.ObjectNull(emptyObjectAttrTypes),
	}
}

func scalingAuto() scalingResource {
	return scalingResource{
		Type:        types.StringValue("auto"),
		Manualinput: types.Int64Null(),
		AutoInput:   types.ObjectValueMust(emptyObjectAttrTypes, emptyObjectAttrValues),
	}
}

func Test_ValidateScalingConfig_manual_valid(t *testing.T) {
	assert.NoError(t, validateScalingConfig(scalingManual(3)))
}

func Test_ValidateScalingConfig_auto_valid(t *testing.T) {
	assert.NoError(t, validateScalingConfig(scalingAuto()))
}

func Test_ValidateScalingConfig_manual_with_auto_input_errors(t *testing.T) {
	s := scalingResource{
		Type:        types.StringValue("manual"),
		Manualinput: types.Int64Value(2),
		AutoInput:   types.ObjectValueMust(emptyObjectAttrTypes, emptyObjectAttrValues),
	}
	assert.ErrorContains(t, validateScalingConfig(s), "auto_input must not be set")
}

func Test_ValidateScalingConfig_auto_with_manual_input_errors(t *testing.T) {
	s := scalingResource{
		Type:        types.StringValue("auto"),
		Manualinput: types.Int64Value(2),
		AutoInput:   types.ObjectValueMust(emptyObjectAttrTypes, emptyObjectAttrValues),
	}
	assert.ErrorContains(t, validateScalingConfig(s), "manual_input must not be set")
}

func Test_ValidateScalingConfig_manual_missing_manual_input_errors(t *testing.T) {
	s := scalingResource{
		Type:        types.StringValue("manual"),
		Manualinput: types.Int64Null(),
		AutoInput:   types.ObjectNull(emptyObjectAttrTypes),
	}
	assert.ErrorContains(t, validateScalingConfig(s), "manual_input is required")
}

func Test_ValidateScalingConfig_auto_missing_auto_input_errors(t *testing.T) {
	s := scalingResource{
		Type:        types.StringValue("auto"),
		Manualinput: types.Int64Null(),
		AutoInput:   types.ObjectNull(emptyObjectAttrTypes),
	}
	assert.ErrorContains(t, validateScalingConfig(s), "auto_input is required")
}
