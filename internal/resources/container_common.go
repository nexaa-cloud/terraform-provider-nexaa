// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nexaa-cloud/nexaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Common container input building functions

func buildPortsInput(ctx context.Context, ports types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if ports.IsNull() || ports.IsUnknown() {
		return make([]string, 0), diags
	}

	var portsList []string
	diags = ports.ElementsAs(ctx, &portsList, false)
	return portsList, diags
}

func buildMountsInput(ctx context.Context, mounts types.List) ([]api.MountInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	if mounts.IsNull() || mounts.IsUnknown() {
		return []api.MountInput{}, diags
	}

	var mountsData []mountResource
	diags = mounts.ElementsAs(ctx, &mountsData, false)
	if diags.HasError() {
		return nil, diags
	}

	var mountInputs []api.MountInput
	for _, m := range mountsData {
		mountInputs = append(mountInputs, api.MountInput{
			Path: m.Path.ValueString(),
			Volume: api.MountVolumeInput{
				Name:       m.Volume.ValueString(),
				AutoCreate: false,
				Increase:   false,
				Size:       nil,
			},
			State: api.StatePresent,
		})
	}

	return mountInputs, diags
}

func buildIngressesInput(ctx context.Context, ingresses types.List) ([]api.IngressInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	if ingresses.IsNull() || ingresses.IsUnknown() {
		return []api.IngressInput{}, diags
	}

	var ingressesData []ingresResource
	diags = ingresses.ElementsAs(ctx, &ingressesData, false)
	if diags.HasError() {
		return nil, diags
	}

	var ingressInputs []api.IngressInput
	for _, ing := range ingressesData {
		if !ing.Port.IsNull() {
			allowList := []string{}
			if !ing.AllowList.IsNull() && !ing.AllowList.IsUnknown() {
				var rawAllowList []types.String
				_ = ing.AllowList.ElementsAs(ctx, &rawAllowList, false)
				for _, ip := range rawAllowList {
					allowList = append(allowList, ip.ValueString())
				}
			}
			var domainPtr *string
			if !ing.DomainName.IsNull() && !ing.DomainName.IsUnknown() {
				domain := ing.DomainName.ValueString()
				domainPtr = &domain
			} else {
				domainPtr = nil
			}
			ingressInputs = append(ingressInputs, api.IngressInput{
				DomainName: domainPtr,
				Port:       int(ing.Port.ValueInt64()),
				EnableTLS:  ing.TLS.ValueBool(),
				Whitelist:  allowList,
				State:      api.StatePresent,
			})
		}
	}

	return ingressInputs, diags
}

func buildHealthCheckInput(ctx context.Context, healthCheck types.Object) (*api.HealthCheckInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	if healthCheck.IsNull() || healthCheck.IsUnknown() {
		return nil, diags
	}

	var hc healthcheckResource
	diags = healthCheck.As(ctx, &hc, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}

	return &api.HealthCheckInput{
		Port: int(hc.Port.ValueInt64()),
		Path: hc.Path.ValueString(),
	}, diags
}

// Common container state building functions

func buildPortsState(containerResult api.ContainerResult) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if containerResult.Ports == nil {
		return types.ListNull(types.StringType), diags
	}

	ports := make([]attr.Value, len(containerResult.Ports))
	for i, p := range containerResult.Ports {
		ports[i] = types.StringValue(p)
	}

	portList, d := types.ListValue(types.StringType, ports)
	diags.Append(d...)
	return portList, diags
}

func buildHealthCheckState(containerResult api.ContainerResult) types.Object {
	if containerResult.HealthCheck == nil {
		return types.ObjectNull(map[string]attr.Type{
			"port": types.Int64Type,
			"path": types.StringType,
		})
	}

	return types.ObjectValueMust(map[string]attr.Type{
		"port": types.Int64Type,
		"path": types.StringType,
	}, map[string]attr.Value{
		"port": types.Int64Value(int64(containerResult.HealthCheck.Port)),
		"path": types.StringValue(containerResult.HealthCheck.Path),
	})
}

// Common update functions for mounts and ingresses

func buildMountsUpdateInput(ctx context.Context, currentMounts, previousMounts types.List) ([]api.MountInput, diag.Diagnostics) {
	var diags diag.Diagnostics
	var mountInputs []api.MountInput

	// Get previous mounts
	var prevMounts []mountResource
	if !previousMounts.IsNull() && !previousMounts.IsUnknown() {
		_ = previousMounts.ElementsAs(ctx, &prevMounts, false)
	}

	// Build planned mounts
	plannedMounts := map[string]struct{}{}
	if !currentMounts.IsNull() && !currentMounts.IsUnknown() {
		var mounts []mountResource
		diags = currentMounts.ElementsAs(ctx, &mounts, false)
		if diags.HasError() {
			return nil, diags
		}

		for _, m := range mounts {
			key := fmt.Sprintf("%s|%s", m.Path.ValueString(), m.Volume.ValueString())
			plannedMounts[key] = struct{}{}
			mountInputs = append(mountInputs, api.MountInput{
				Path: m.Path.ValueString(),
				Volume: api.MountVolumeInput{
					Name:       m.Volume.ValueString(),
					AutoCreate: false,
					Increase:   false,
					Size:       nil,
				},
				State: api.StatePresent,
			})
		}
	}

	// Mark removed mounts as absent
	for _, m := range prevMounts {
		key := fmt.Sprintf("%s|%s", m.Path.ValueString(), m.Volume.ValueString())
		if _, exists := plannedMounts[key]; !exists {
			mountInputs = append(mountInputs, api.MountInput{
				Path: m.Path.ValueString(),
				Volume: api.MountVolumeInput{
					Name:       m.Volume.ValueString(),
					AutoCreate: false,
					Increase:   false,
					Size:       nil,
				},
				State: api.StateAbsent,
			})
		}
	}

	return mountInputs, diags
}

func buildIngressesUpdateInput(ctx context.Context, currentIngresses, previousIngresses types.List) ([]api.IngressInput, diag.Diagnostics) {
	var diags diag.Diagnostics
	var ingressInputs []api.IngressInput

	if currentIngresses.IsNull() || currentIngresses.IsUnknown() {
		return ingressInputs, diags
	}

	var ingresses []ingresResource
	diags = currentIngresses.ElementsAs(ctx, &ingresses, false)
	if diags.HasError() {
		return nil, diags
	}

	// Get previous ingresses
	var prevIngresses []ingresResource
	if !previousIngresses.IsNull() && !previousIngresses.IsUnknown() {
		_ = previousIngresses.ElementsAs(ctx, &prevIngresses, false)
	}

	plannedIngresses := map[string]struct{}{}
	for _, ing := range ingresses {
		allowList := []string{}
		if !ing.AllowList.IsNull() && !ing.AllowList.IsUnknown() {
			var allowListVals []string
			d := ing.AllowList.ElementsAs(ctx, &allowListVals, false)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			allowList = allowListVals
		}

		key := ing.DomainName.ValueString()
		plannedIngresses[key] = struct{}{}
		ingressInputs = append(ingressInputs, api.IngressInput{
			DomainName: ing.DomainName.ValueStringPointer(),
			Port:       int(ing.Port.ValueInt64()),
			EnableTLS:  ing.TLS.ValueBool(),
			Whitelist:  allowList,
			State:      api.StatePresent,
		})
	}

	// Mark removed ingresses as absent
	for _, prevIng := range prevIngresses {
		key := prevIng.DomainName.ValueString()
		if _, exists := plannedIngresses[key]; !exists {
			ingressInputs = append(ingressInputs, api.IngressInput{
				DomainName: prevIng.DomainName.ValueStringPointer(),
				Port:       int(prevIng.Port.ValueInt64()),
				EnableTLS:  prevIng.TLS.ValueBool(),
				State:      api.StateAbsent,
			})
		}
	}

	return ingressInputs, diags
}

// Common import state builder
func buildContainerImportState(ctx context.Context, container api.ContainerResult, namespace, name string) (map[string]attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Environment Variables (import)
	envTF := types.SetNull(envVarObjectType())
	if container.EnvironmentVariables != nil {
		setVal, _ := buildEnvSetFromAPI(ctx, container.EnvironmentVariables, nil, types.SetNull(envVarObjectType()), secretMaskOnly)
		envTF = setVal
	}

	// Ports
	ports := make([]attr.Value, len(container.Ports))
	for i, p := range container.Ports {
		ports[i] = types.StringValue(p)
	}

	portList, d := types.ListValue(types.StringType, ports)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	// Mounts
	mountTF := types.ListNull(MountsObjectType())
	if container.Mounts != nil {
		mountList, d := buildMountsFromApi(container.Mounts)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		mountTF = mountList
	}

	// Ingresses
	ingressesTF, _ := buildIngressesFromApi(container)

	// Health Check
	healthTF := buildHealthCheckState(container)

	return map[string]attr.Value{
		"id":                    types.StringValue(container.Name),
		"name":                  types.StringValue(container.Name),
		"namespace":             types.StringValue(namespace),
		"image":                 types.StringValue(container.Image),
		"registry":              processRegistryName(container),
		"environment_variables": envTF,
		"ports":                 portList,
		"ingresses":             ingressesTF,
		"mounts":                mountTF,
		"health_check":          healthTF,
		"status":                types.StringValue(container.State),
		"last_updated":          types.StringValue(time.Now().Format(time.RFC3339)),
	}, diags
}

// Common validation for import ID format
func parseContainerImportID(importID string) (namespace, name string, err error) {
	parts := strings.SplitN(importID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected import ID in the format \"<namespace>/<container_name>\", got: %s", importID)
	}
	return parts[0], parts[1], nil
}
