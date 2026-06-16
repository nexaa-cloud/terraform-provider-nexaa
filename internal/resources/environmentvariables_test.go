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

// --- extractEnvInputsFromSet ---

func makeEnvSet(envs []attr.Value) types.Set {
	return types.SetValueMust(envVarObjectType(), envs)
}

func makeEnvObj(name, value string, secret bool) types.Object {
	return types.ObjectValueMust(envVarObjectType().AttrTypes, map[string]attr.Value{
		"name":   types.StringValue(name),
		"value":  types.StringValue(value),
		"secret": types.BoolValue(secret),
	})
}

func Test_ExtractEnvInputs_null_set(t *testing.T) {
	inputs, diags := extractEnvInputsFromSet(context.Background(), types.SetNull(envVarObjectType()))
	assert.False(t, diags.HasError())
	assert.Empty(t, inputs)
}

func Test_ExtractEnvInputs_single_plain_var(t *testing.T) {
	set := makeEnvSet([]attr.Value{makeEnvObj("KEY", "value", false)})
	inputs, diags := extractEnvInputsFromSet(context.Background(), set)
	assert.False(t, diags.HasError())
	assert.Len(t, inputs, 1)
	assert.Equal(t, "KEY", inputs[0].Name)
	assert.Equal(t, "value", inputs[0].Value)
	assert.False(t, inputs[0].Secret)
	assert.Equal(t, api.StatePresent, inputs[0].State)
}

func Test_ExtractEnvInputs_secret_var(t *testing.T) {
	set := makeEnvSet([]attr.Value{makeEnvObj("SECRET_KEY", "s3cr3t", true)})
	inputs, diags := extractEnvInputsFromSet(context.Background(), set)
	assert.False(t, diags.HasError())
	assert.Len(t, inputs, 1)
	assert.True(t, inputs[0].Secret)
}

func Test_ExtractEnvInputs_null_value_errors(t *testing.T) {
	obj := types.ObjectValueMust(envVarObjectType().AttrTypes, map[string]attr.Value{
		"name":   types.StringValue("BAD_VAR"),
		"value":  types.StringNull(),
		"secret": types.BoolValue(false),
	})
	set := makeEnvSet([]attr.Value{obj})
	_, diags := extractEnvInputsFromSet(context.Background(), set)
	assert.True(t, diags.HasError())
}

// --- buildPrevSecretMap ---

func Test_BuildPrevSecretMap_null_set(t *testing.T) {
	m := buildPrevSecretMap(context.Background(), types.SetNull(envVarObjectType()))
	assert.Empty(t, m)
}

func Test_BuildPrevSecretMap_only_secrets_included(t *testing.T) {
	set := makeEnvSet([]attr.Value{
		makeEnvObj("PLAIN", "visible", false),
		makeEnvObj("SECRET", "hidden", true),
	})
	m := buildPrevSecretMap(context.Background(), set)
	assert.Len(t, m, 1)
	assert.Equal(t, "hidden", m["SECRET"])
	assert.NotContains(t, m, "PLAIN")
}

// --- buildEnvSetFromAPI ---

func makeAPIVar(name string, value *string, secret bool) api.EnvironmentVariableResult {
	return api.EnvironmentVariableResult{Name: name, Value: value, Secret: secret}
}

func strVal(s string) *string { return &s }

func Test_BuildEnvSetFromAPI_secretUseProvided(t *testing.T) {
	apiVars := []api.EnvironmentVariableResult{
		makeAPIVar("PLAIN", strVal("visible"), false),
		makeAPIVar("SECRET", nil, true),
	}
	provided := []api.EnvironmentVariableInput{
		{Name: "SECRET", Value: "provided_secret", Secret: true},
	}
	set, diags := buildEnvSetFromAPI(context.Background(), apiVars, provided, types.SetNull(envVarObjectType()), secretUseProvided)
	assert.False(t, diags.HasError())
	assert.Equal(t, 2, len(set.Elements()))

	var envs []environmentVariableResource
	set.ElementsAs(context.Background(), &envs, false)
	byName := map[string]environmentVariableResource{}
	for _, e := range envs {
		byName[e.Name.ValueString()] = e
	}
	assert.Equal(t, "visible", byName["PLAIN"].Value.ValueString())
	assert.Equal(t, "provided_secret", byName["SECRET"].Value.ValueString())
}

func Test_BuildEnvSetFromAPI_secretUseProvided_missing_falls_back_to_mask(t *testing.T) {
	apiVars := []api.EnvironmentVariableResult{
		makeAPIVar("SECRET", nil, true),
	}
	set, diags := buildEnvSetFromAPI(context.Background(), apiVars, nil, types.SetNull(envVarObjectType()), secretUseProvided)
	assert.False(t, diags.HasError())

	var envs []environmentVariableResource
	set.ElementsAs(context.Background(), &envs, false)
	assert.Equal(t, "***", envs[0].Value.ValueString())
}

func Test_BuildEnvSetFromAPI_secretPreservePrev(t *testing.T) {
	prev := makeEnvSet([]attr.Value{makeEnvObj("SECRET", "old_secret", true)})
	apiVars := []api.EnvironmentVariableResult{
		makeAPIVar("SECRET", nil, true),
	}
	set, diags := buildEnvSetFromAPI(context.Background(), apiVars, nil, prev, secretPreservePrev)
	assert.False(t, diags.HasError())

	var envs []environmentVariableResource
	set.ElementsAs(context.Background(), &envs, false)
	assert.Equal(t, "old_secret", envs[0].Value.ValueString())
}

func Test_BuildEnvSetFromAPI_secretPreservePrev_unknown_secret_masked(t *testing.T) {
	apiVars := []api.EnvironmentVariableResult{
		makeAPIVar("NEW_SECRET", nil, true),
	}
	set, diags := buildEnvSetFromAPI(context.Background(), apiVars, nil, types.SetNull(envVarObjectType()), secretPreservePrev)
	assert.False(t, diags.HasError())

	var envs []environmentVariableResource
	set.ElementsAs(context.Background(), &envs, false)
	assert.Equal(t, "***", envs[0].Value.ValueString())
}

// --- buildEnvUpdateInputs ---

func Test_BuildEnvUpdateInputs_null_prev_and_plan(t *testing.T) {
	inputs, diags := buildEnvUpdateInputs(context.Background(), types.SetNull(envVarObjectType()), types.SetNull(envVarObjectType()))
	assert.False(t, diags.HasError())
	assert.Empty(t, inputs)
}

func Test_BuildEnvUpdateInputs_no_removals(t *testing.T) {
	set := makeEnvSet([]attr.Value{
		makeEnvObj("FOO", "bar", false),
		makeEnvObj("BAZ", "qux", false),
	})
	inputs, diags := buildEnvUpdateInputs(context.Background(), set, set)
	assert.False(t, diags.HasError())
	assert.Len(t, inputs, 2)
	for _, inp := range inputs {
		assert.Equal(t, api.StatePresent, inp.State)
	}
}

func Test_BuildEnvUpdateInputs_one_var_removed(t *testing.T) {
	prev := makeEnvSet([]attr.Value{
		makeEnvObj("KEEP", "val", false),
		makeEnvObj("REMOVE", "old", false),
	})
	plan := makeEnvSet([]attr.Value{
		makeEnvObj("KEEP", "val", false),
	})

	inputs, diags := buildEnvUpdateInputs(context.Background(), plan, prev)
	assert.False(t, diags.HasError())
	assert.Len(t, inputs, 2)

	byName := map[string]api.EnvironmentVariableInput{}
	for _, inp := range inputs {
		byName[inp.Name] = inp
	}
	assert.Equal(t, api.StatePresent, byName["KEEP"].State)
	assert.Equal(t, api.StateAbsent, byName["REMOVE"].State)
}

func Test_BuildEnvUpdateInputs_all_vars_removed(t *testing.T) {
	prev := makeEnvSet([]attr.Value{
		makeEnvObj("A", "1", false),
		makeEnvObj("B", "2", false),
	})

	inputs, diags := buildEnvUpdateInputs(context.Background(), types.SetNull(envVarObjectType()), prev)
	assert.False(t, diags.HasError())
	assert.Len(t, inputs, 2)
	for _, inp := range inputs {
		assert.Equal(t, api.StateAbsent, inp.State)
	}
}

func Test_BuildEnvUpdateInputs_secret_var_removed(t *testing.T) {
	prev := makeEnvSet([]attr.Value{
		makeEnvObj("PLAIN", "visible", false),
		makeEnvObj("SECRET", "hidden", true),
	})
	plan := makeEnvSet([]attr.Value{
		makeEnvObj("PLAIN", "visible", false),
	})

	inputs, diags := buildEnvUpdateInputs(context.Background(), plan, prev)
	assert.False(t, diags.HasError())
	assert.Len(t, inputs, 2)

	byName := map[string]api.EnvironmentVariableInput{}
	for _, inp := range inputs {
		byName[inp.Name] = inp
	}
	assert.Equal(t, api.StatePresent, byName["PLAIN"].State)
	assert.Equal(t, api.StateAbsent, byName["SECRET"].State)
}

func Test_BuildEnvUpdateInputs_var_added(t *testing.T) {
	prev := makeEnvSet([]attr.Value{makeEnvObj("OLD", "v", false)})
	plan := makeEnvSet([]attr.Value{
		makeEnvObj("OLD", "v", false),
		makeEnvObj("NEW", "w", false),
	})

	inputs, diags := buildEnvUpdateInputs(context.Background(), plan, prev)
	assert.False(t, diags.HasError())
	assert.Len(t, inputs, 2)
	for _, inp := range inputs {
		assert.Equal(t, api.StatePresent, inp.State)
	}
}

func Test_BuildEnvUpdateInputs_null_prev_behaves_like_extract(t *testing.T) {
	plan := makeEnvSet([]attr.Value{makeEnvObj("FOO", "bar", false)})
	inputs, diags := buildEnvUpdateInputs(context.Background(), plan, types.SetNull(envVarObjectType()))
	assert.False(t, diags.HasError())
	assert.Len(t, inputs, 1)
	assert.Equal(t, api.StatePresent, inputs[0].State)
	assert.Equal(t, "FOO", inputs[0].Name)
}

func Test_BuildEnvSetFromAPI_secretMaskOnly(t *testing.T) {
	apiVars := []api.EnvironmentVariableResult{
		makeAPIVar("SECRET", strVal("should_be_masked"), true),
		makeAPIVar("PLAIN", strVal("visible"), false),
	}
	set, diags := buildEnvSetFromAPI(context.Background(), apiVars, nil, types.SetNull(envVarObjectType()), secretMaskOnly)
	assert.False(t, diags.HasError())

	var envs []environmentVariableResource
	set.ElementsAs(context.Background(), &envs, false)
	byName := map[string]environmentVariableResource{}
	for _, e := range envs {
		byName[e.Name.ValueString()] = e
	}
	assert.Equal(t, "***", byName["SECRET"].Value.ValueString())
	assert.Equal(t, "visible", byName["PLAIN"].Value.ValueString())
}
