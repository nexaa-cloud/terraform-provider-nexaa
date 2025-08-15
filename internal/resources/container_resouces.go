// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"errors"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

func ContainerResourceObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cpu": types.Float64Type,
		"ram": types.Float64Type,
	}
}

func buildResourcesFromAPI(resources api.ContainerResources) (types.Object, error) {
	resParts := strings.Split(string(resources), "_")
	if len(resParts) != 4 {
		return types.ObjectNull(ContainerResourceObjectAttributeTypes()), errors.New("error parsing container resources")
	}
	cpu, err := strconv.ParseFloat(resParts[1], 64)
	if err != nil {
		return types.ObjectNull(ContainerResourceObjectAttributeTypes()), errors.New("error parsing container resources")
	}

	ram, err := strconv.ParseFloat(resParts[3], 64)
	if err != nil {
		return types.ObjectNull(ContainerResourceObjectAttributeTypes()), errors.New("error parsing container resources")
	}

	// Create a new types.Object with CPU and RAM fields set
	return types.ObjectValueMust(
		ContainerResourceObjectAttributeTypes(),
		map[string]attr.Value{
			"cpu": types.Float64Value(cpu / 1000),
			"ram": types.Float64Value(ram / 1000),
		},
	), nil
}
