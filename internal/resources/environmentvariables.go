// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

// secretMode defines how secret environment variable values are populated in state.
type secretMode int

const (
	secretUseProvided  secretMode = iota // store provided secret value (Create/Update)
	secretPreservePrev                   // preserve previous secret value or mask if absent (Read)
	secretMaskOnly                       // mask all secrets (Import)
)

// envVarObjectType returns the ObjectType used for environment variable elements.
func envVarObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":   types.StringType,
		"value":  types.StringType,
		"secret": types.BoolType,
	}}
}

// extractEnvInputsFromSet converts a Terraform Set of environment variable objects into API inputs.
func extractEnvInputsFromSet(ctx context.Context, set types.Set) ([]api.EnvironmentVariableInput, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	var envs []environvariableResource
	if d := set.ElementsAs(ctx, &envs, false); d.HasError() {
		return nil, d
	}
	inputs := make([]api.EnvironmentVariableInput, 0, len(envs))
	for _, ev := range envs {
		if ev.Value.IsNull() || ev.Value.IsUnknown() {
			diags.AddError("Invalid environment variable", "Value for env var "+ev.Name.ValueString()+" is null or unknown")
			continue
		}
		inputs = append(inputs, api.EnvironmentVariableInput{
			Name:   ev.Name.ValueString(),
			Value:  ev.Value.ValueString(),
			Secret: ev.Secret.ValueBool(),
			State:  api.StatePresent,
		})
	}
	return inputs, diags
}

// buildPrevSecretMap extracts existing secret values from prior state for reuse.
func buildPrevSecretMap(ctx context.Context, prevSet types.Set) map[string]string {
	m := map[string]string{}
	if prevSet.IsNull() || prevSet.IsUnknown() {
		return m
	}
	var prev []environvariableResource
	_ = prevSet.ElementsAs(ctx, &prev, false)
	for _, p := range prev {
		if p.Secret.ValueBool() && !p.Value.IsNull() && !p.Value.IsUnknown() {
			m[p.Name.ValueString()] = p.Value.ValueString()
		}
	}
	return m
}

// buildEnvSetFromAPI converts API env vars to a Terraform Set with appropriate secret handling based on mode.
func buildEnvSetFromAPI(ctx context.Context, apiVars []api.ContainerResultEnvironmentVariablesEnvironmentVariable, provided []api.EnvironmentVariableInput, prevSet types.Set, mode secretMode) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	objType := envVarObjectType()
	providedMap := map[string]string{}
	if mode == secretUseProvided {
		for _, p := range provided {
			if p.Secret {
				providedMap[p.Name] = p.Value
			}
		}
	}
	prevSecrets := map[string]string{}
	if mode == secretPreservePrev {
		prevSecrets = buildPrevSecretMap(ctx, prevSet)
	}

	values := make([]attr.Value, 0, len(apiVars))
	for _, ev := range apiVars {
		var val types.String
		if ev.Secret {
			switch mode {
			case secretUseProvided:
				if v, ok := providedMap[ev.Name]; ok {
					val = types.StringValue(v)
				} else {
					val = types.StringValue("***")
				}
			case secretPreservePrev:
				if v, ok := prevSecrets[ev.Name]; ok {
					val = types.StringValue(v)
				} else {
					val = types.StringValue("***")
				}
			case secretMaskOnly:
				val = types.StringValue("***")
			}
		} else if ev.Value != nil {
			val = types.StringValue(*ev.Value)
		} else {
			val = types.StringNull()
		}
		obj := types.ObjectValueMust(objType.AttrTypes, map[string]attr.Value{
			"name":   types.StringValue(ev.Name),
			"value":  val,
			"secret": types.BoolValue(ev.Secret),
		})
		values = append(values, obj)
	}
	setVal, d := types.SetValue(objType, values)
	diags.Append(d...)
	return setVal, diags
}
