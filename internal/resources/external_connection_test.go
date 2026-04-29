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

func Test_ExternalConnection_null(t *testing.T) {

	var queryResult *api.ContainerResultExternalConnection

	externalConnection, _ := buildExternalConnectionWithPortsListFromApi(
		context.Background(),
		queryResult,
	)

	assert.Equal(
		t,
		types.ObjectNull(ExternalConnectionWithPortsObjectAttributeTypes()),
		externalConnection,
	)
}

func Test_ExternalConnection_with_ports(t *testing.T) {
	// This test is to verify that the ports list is correctly built when the API returns a non-null result.

	// Mock API response with ports
	// 	Ipv4  string                                                `json:"ipv4"`
	// Ipv6  string                                                `json:"ipv6"`
	// Ports []ExternalConnectionResultPortsExternalConnectionPort `json:"ports"`

	metricPort := 9001
	webServerPort := 80

	var ports []api.ExternalConnectionResultPortsExternalConnectionPort
	ports = append(ports, api.ExternalConnectionResultPortsExternalConnectionPort{
		AllowList: []string{
			"0.0.0.0/0",
			"::/0",
		},
		ExternalPort: 10022,
		InternalPort: &webServerPort,
		Protocol:     api.ProtocolTcp,
	})

	ports = append(ports, api.ExternalConnectionResultPortsExternalConnectionPort{
		AllowList: []string{
			"0.0.0.0/0",
			"::/0",
		},
		ExternalPort: 10023,
		InternalPort: &metricPort,
		Protocol:     api.ProtocolTcp,
	})

	queryResult := &api.ExternalConnectionResult{
		Ipv4:  "203.0.113.5",
		Ipv6:  "2001:db8::1",
		Ports: ports,
	}

	externalConnection, _ := buildExternalConnectionWithPortsListFromApi(
		context.Background(),
		&api.ContainerResultExternalConnection{
			ExternalConnectionResult: *queryResult,
		},
	)

	expectedPorts := types.ListValueMust(types.ObjectType{AttrTypes: ExternalConnectionPortsContainerObjectAttributeTypes()}, []attr.Value{
		types.ObjectValueMust(
			ExternalConnectionPortsContainerObjectAttributeTypes(),
			map[string]attr.Value{
				"external_port": types.Int64Value(10022),
				"internal_port": types.Int64Value(80),
				"protocol":      types.StringValue("TCP"),
				"allowlist": types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("0.0.0.0/0"),
					types.StringValue("::/0"),
				}),
			},
		),
		types.ObjectValueMust(
			ExternalConnectionPortsContainerObjectAttributeTypes(),
			map[string]attr.Value{
				"external_port": types.Int64Value(10023),
				"internal_port": types.Int64Value(9001),
				"protocol":      types.StringValue("TCP"),
				"allowlist": types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("0.0.0.0/0"),
					types.StringValue("::/0"),
				}),
			},
		),
	})

	expected := types.ObjectValueMust(ExternalConnectionWithPortsObjectAttributeTypes(), map[string]attr.Value{
		"ipv4":  types.StringValue("203.0.113.5"),
		"ipv6":  types.StringValue("2001:db8::1"),
		"ports": expectedPorts,
	})

	assert.Equal(
		t,
		expected,
		externalConnection,
	)
}
