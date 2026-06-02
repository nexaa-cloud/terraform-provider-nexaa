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
	"github.com/stretchr/testify/require"
)

// --- ExternalConnectionObjectType ---

func Test_ExternalConnectionObjectType_has_expected_keys(t *testing.T) {
	attrs := ExternalConnectionObjectType().AttrTypes
	assert.Contains(t, attrs, "ipv4")
	assert.Contains(t, attrs, "ipv6")
	assert.Contains(t, attrs, "ports")
}

// --- buildExternalConnectionFromApi ---

func Test_BuildExternalConnectionFromApi_empty_ports_returns_null(t *testing.T) {
	conn := api.ExternalConnectionResult{Ipv4: "1.2.3.4", Ipv6: "::1"}
	obj, diags := buildExternalConnectionFromApi(context.Background(), conn)
	assert.False(t, diags.HasError())
	assert.True(t, obj.IsNull())
}

func Test_BuildExternalConnectionFromApi_with_port_returns_object(t *testing.T) {
	conn := api.ExternalConnectionResult{
		Ipv4: "1.2.3.4",
		Ipv6: "::1",
		Ports: []api.ExternalConnectionResultPortsExternalConnectionPort{
			{ExternalPort: 5432, AllowList: []string{"0.0.0.0/0"}, Protocol: api.ProtocolTcp},
		},
	}
	obj, diags := buildExternalConnectionFromApi(context.Background(), conn)
	assert.False(t, diags.HasError())
	assert.False(t, obj.IsNull())
	assert.Equal(t, types.StringValue("1.2.3.4"), obj.Attributes()["ipv4"])
}

// --- buildExternalConnectionInputContainer helpers ---

func makeExtConnNullContainer() containerResource {
	return containerResource{
		ExternalConnection: types.ObjectNull(ExternalConnectionWithPortsObjectAttributeTypes()),
	}
}

func makeExtConnUnknownContainer() containerResource {
	return containerResource{
		ExternalConnection: types.ObjectUnknown(ExternalConnectionWithPortsObjectAttributeTypes()),
	}
}

func makeExtConnContainerWithPort(internalPort int64, protocol string, externalPort *int64) containerResource {
	portAttrTypes := ExternalConnectionPortsContainerObjectAttributeTypes()
	var epVal attr.Value
	if externalPort != nil {
		epVal = types.Int64Value(*externalPort)
	} else {
		epVal = types.Int64Null()
	}
	portObj := types.ObjectValueMust(portAttrTypes, map[string]attr.Value{
		"external_port": epVal,
		"internal_port": types.Int64Value(internalPort),
		"protocol":      types.StringValue(protocol),
		"allowlist":     types.ListValueMust(types.StringType, []attr.Value{types.StringValue("0.0.0.0/0")}),
	})
	portsList := types.ListValueMust(types.ObjectType{AttrTypes: portAttrTypes}, []attr.Value{portObj})
	extConnObj := types.ObjectValueMust(
		ExternalConnectionWithPortsObjectAttributeTypes(),
		map[string]attr.Value{
			"ipv4":  types.StringNull(),
			"ipv6":  types.StringNull(),
			"ports": portsList,
		},
	)
	return containerResource{ExternalConnection: extConnObj}
}

func makeExtConnContainerEmptyPorts() containerResource {
	portAttrTypes := ExternalConnectionPortsContainerObjectAttributeTypes()
	emptyPorts := types.ListValueMust(types.ObjectType{AttrTypes: portAttrTypes}, []attr.Value{})
	extConnObj := types.ObjectValueMust(
		ExternalConnectionWithPortsObjectAttributeTypes(),
		map[string]attr.Value{
			"ipv4":  types.StringNull(),
			"ipv6":  types.StringNull(),
			"ports": emptyPorts,
		},
	)
	return containerResource{ExternalConnection: extConnObj}
}

// --- buildExternalConnectionInputContainer ---

func Test_BuildExternalConnectionInputContainer_null_plan_is_absent(t *testing.T) {
	input, diags := buildExternalConnectionInputContainer(context.Background(), makeExtConnNullContainer(), nil)
	assert.False(t, diags.HasError())
	require.NotNil(t, input)
	assert.Equal(t, api.StateAbsent, input.State)
}

func Test_BuildExternalConnectionInputContainer_unknown_plan_is_absent(t *testing.T) {
	input, diags := buildExternalConnectionInputContainer(context.Background(), makeExtConnUnknownContainer(), nil)
	assert.False(t, diags.HasError())
	require.NotNil(t, input)
	assert.Equal(t, api.StateAbsent, input.State)
}

func Test_BuildExternalConnectionInputContainer_new_port_is_present(t *testing.T) {
	plan := makeExtConnContainerWithPort(8080, "TCP", nil)
	input, diags := buildExternalConnectionInputContainer(context.Background(), plan, nil)
	assert.False(t, diags.HasError())
	require.NotNil(t, input)
	assert.Equal(t, api.StatePresent, input.State)
	require.Len(t, input.Ports, 1)
	assert.Equal(t, api.StatePresent, input.Ports[0].State)
	assert.Equal(t, api.ProtocolTcp, input.Ports[0].Protocol)
	assert.Nil(t, input.Ports[0].ExternalPort)
}

func Test_BuildExternalConnectionInputContainer_existing_port_preserves_external_port(t *testing.T) {
	plan := makeExtConnContainerWithPort(8080, "TCP", nil)
	ep := int64(32500)
	state := makeExtConnContainerWithPort(8080, "TCP", &ep)
	input, diags := buildExternalConnectionInputContainer(context.Background(), plan, &state)
	assert.False(t, diags.HasError())
	require.NotNil(t, input)
	require.Len(t, input.Ports, 1)
	require.NotNil(t, input.Ports[0].ExternalPort)
	assert.Equal(t, 32500, *input.Ports[0].ExternalPort)
}

func Test_BuildExternalConnectionInputContainer_removed_port_marked_absent(t *testing.T) {
	plan := makeExtConnContainerEmptyPorts()
	ep := int64(32500)
	state := makeExtConnContainerWithPort(8080, "TCP", &ep)
	input, diags := buildExternalConnectionInputContainer(context.Background(), plan, &state)
	assert.False(t, diags.HasError())
	require.NotNil(t, input)
	assert.Equal(t, api.StatePresent, input.State)
	require.Len(t, input.Ports, 1)
	assert.Equal(t, api.StateAbsent, input.Ports[0].State)
}

// --- buildExternalConnectionInputStarterContainer ---

func makeExtConnNullStarter() starterContainerResource {
	return starterContainerResource{
		ExternalConnection: types.ObjectNull(ExternalConnectionWithPortsObjectAttributeTypes()),
	}
}

func makeExtConnUnknownStarter() starterContainerResource {
	return starterContainerResource{
		ExternalConnection: types.ObjectUnknown(ExternalConnectionWithPortsObjectAttributeTypes()),
	}
}

func makeExtConnStarterWithPort(internalPort int64, protocol string) starterContainerResource {
	portAttrTypes := ExternalConnectionPortsContainerObjectAttributeTypes()
	portObj := types.ObjectValueMust(portAttrTypes, map[string]attr.Value{
		"external_port": types.Int64Null(),
		"internal_port": types.Int64Value(internalPort),
		"protocol":      types.StringValue(protocol),
		"allowlist":     types.ListValueMust(types.StringType, []attr.Value{types.StringValue("0.0.0.0/0")}),
	})
	portsList := types.ListValueMust(types.ObjectType{AttrTypes: portAttrTypes}, []attr.Value{portObj})
	extConnObj := types.ObjectValueMust(
		ExternalConnectionWithPortsObjectAttributeTypes(),
		map[string]attr.Value{
			"ipv4":  types.StringNull(),
			"ipv6":  types.StringNull(),
			"ports": portsList,
		},
	)
	return starterContainerResource{ExternalConnection: extConnObj}
}

func Test_BuildExternalConnectionInputStarterContainer_null_plan_is_absent(t *testing.T) {
	input, diags := buildExternalConnectionInputStarterContainer(context.Background(), makeExtConnNullStarter(), nil)
	assert.False(t, diags.HasError())
	require.NotNil(t, input)
	assert.Equal(t, api.StateAbsent, input.State)
}

func Test_BuildExternalConnectionInputStarterContainer_unknown_plan_is_absent(t *testing.T) {
	input, diags := buildExternalConnectionInputStarterContainer(context.Background(), makeExtConnUnknownStarter(), nil)
	assert.False(t, diags.HasError())
	require.NotNil(t, input)
	assert.Equal(t, api.StateAbsent, input.State)
}

func Test_BuildExternalConnectionInputStarterContainer_new_port_is_present(t *testing.T) {
	plan := makeExtConnStarterWithPort(8080, "TCP")
	input, diags := buildExternalConnectionInputStarterContainer(context.Background(), plan, nil)
	assert.False(t, diags.HasError())
	require.NotNil(t, input)
	assert.Equal(t, api.StatePresent, input.State)
	require.Len(t, input.Ports, 1)
	assert.Equal(t, api.ProtocolTcp, input.Ports[0].Protocol)
}

// --- buildExternalConnectionWithPortsListFromApi (existing tests below) ---

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

func Test_ExternalConnection_udp_protocol(t *testing.T) {
	internalPort := 53

	queryResult := &api.ContainerResultExternalConnection{
		ExternalConnectionResult: api.ExternalConnectionResult{
			Ipv4: "203.0.113.5",
			Ipv6: "2001:db8::1",
			Ports: []api.ExternalConnectionResultPortsExternalConnectionPort{
				{
					AllowList:    []string{"0.0.0.0/0"},
					ExternalPort: 10053,
					InternalPort: &internalPort,
					Protocol:     api.ProtocolUdp,
				},
			},
		},
	}

	result, diags := buildExternalConnectionWithPortsListFromApi(context.Background(), queryResult)
	assert.False(t, diags.HasError())

	portsList := result.Attributes()["ports"].(types.List)
	portObj := portsList.Elements()[0].(types.Object)
	assert.Equal(t, types.StringValue("UDP"), portObj.Attributes()["protocol"])
}
