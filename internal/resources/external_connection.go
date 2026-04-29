// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

func ExternalConnectionObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: ExternalConnectionObjectAttributeTypes()}
}

func ExternalConnectionObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"ipv4":  types.StringType,
		"ipv6":  types.StringType,
		"ports": types.ObjectType{AttrTypes: ExternalConnectionPortsObjectAttributeTypes()},
	}
}

func ExternalConnectionWithPortsObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"ipv4":  types.StringType,
		"ipv6":  types.StringType,
		"ports": types.ListType{ElemType: types.ObjectType{AttrTypes: ExternalConnectionPortsContainerObjectAttributeTypes()}},
	}
}

func ExternalConnectionPortsObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"external_port": types.Int64Type,
		"allowlist":     types.ListType{ElemType: types.StringType},
	}
}

func ExternalConnectionPortsContainerObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"external_port": types.Int64Type,
		"internal_port": types.Int64Type,
		"protocol":      types.StringType,
		"allowlist":     types.ListType{ElemType: types.StringType},
	}
}

func buildExternalConnectionInputCloudDb(ctx context.Context, plan cloudDatabaseClusterResource, state *cloudDatabaseClusterResource) *api.ExternalConnectionInput {
	var externalConnectionInputs api.ExternalConnectionInput

	if plan.ExternalConnection.IsNull() {
		externalConnectionInputs.State = api.StateAbsent
		externalConnectionInputs.Ports = []api.ExternalConnectionPortInput{}
		return &externalConnectionInputs
	}
	if plan.ExternalConnection.IsUnknown() {
		externalConnectionInputs.State = api.StateAbsent
		externalConnectionInputs.Ports = []api.ExternalConnectionPortInput{}
		return &externalConnectionInputs
	}

	var externalConnectionData cloudDatabaseClusterExternalConnectionResource
	diags := plan.ExternalConnection.As(ctx, &externalConnectionData, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil
	}

	var oldExternalConnectionData cloudDatabaseClusterExternalConnectionResource
	if state != nil {
		if !state.ExternalConnection.IsNull() && !state.ExternalConnection.IsUnknown() {
			diags = state.ExternalConnection.As(ctx, &oldExternalConnectionData, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil
			}
		}
	}

	var ports api.ExternalConnectionPortInput
	var externalConnectionPortsData cloudDatabaseClusterExternalConnectionPortsResource
	diags = externalConnectionData.Ports.As(ctx, &externalConnectionPortsData, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil
	}

	var oldExternalConnectionPortsData cloudDatabaseClusterExternalConnectionPortsResource
	if !oldExternalConnectionData.Ports.IsNull() && !oldExternalConnectionData.Ports.IsUnknown() {
		diags = oldExternalConnectionData.Ports.As(ctx, &oldExternalConnectionPortsData, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil
		}

		externalport := int(oldExternalConnectionPortsData.ExternalPort.ValueInt64())
		ports.ExternalPort = &externalport
	}

	var allowlist []api.AllowListInput
	if !oldExternalConnectionPortsData.ExternalPort.IsNull() && !oldExternalConnectionPortsData.ExternalPort.IsUnknown() {
		allowlist = buildAllowlistInput(ctx, &oldExternalConnectionPortsData.Allowlist, externalConnectionPortsData.Allowlist)
	}
	if oldExternalConnectionPortsData.ExternalPort.IsNull() {
		// Create ports object with correct allowlist and empty port in case of a new external connection
		ports.ExternalPort = nil

		newAllowlist := toStringArray(ctx, externalConnectionPortsData.Allowlist)
		for _, newIp := range newAllowlist {
			allowlist = append(allowlist, api.AllowListInput{
				Ip:    newIp,
				State: api.StatePresent,
			})
		}
	}
	if oldExternalConnectionPortsData.ExternalPort.IsUnknown() {
		// Create ports object with correct allowlist and empty port in case of a new external connection
		ports.ExternalPort = nil

		newAllowlist := toStringArray(ctx, externalConnectionPortsData.Allowlist)
		for _, newIp := range newAllowlist {
			allowlist = append(allowlist, api.AllowListInput{
				Ip:    newIp,
				State: api.StatePresent,
			})
		}
	}

	ports.AllowList = allowlist
	ports.State = api.StatePresent
	ports.Protocol = api.ProtocolTcp
	externalConnectionInputs.Ports = []api.ExternalConnectionPortInput{ports}
	externalConnectionInputs.SharedIp = true
	externalConnectionInputs.State = api.StatePresent

	return &externalConnectionInputs
}

func buildExternalConnectionFromApi(ctx context.Context, conn api.ExternalConnectionResult) (types.Object, diag.Diagnostics) {

	allowlist, diags := toTypesStringList(ctx, conn.GetPorts()[0].GetAllowList())
	if diags.HasError() {
		return types.ObjectNull(ExternalConnectionObjectAttributeTypes()), diags
	}

	ports := types.ObjectValueMust(
		ExternalConnectionPortsObjectAttributeTypes(),
		map[string]attr.Value{
			"external_port": types.Int64Value(int64(conn.GetPorts()[0].GetExternalPort())),
			"allowlist":     allowlist,
		})

	externalConnectionObj := types.ObjectValueMust(
		ExternalConnectionObjectAttributeTypes(),
		map[string]attr.Value{
			"ipv4":  types.StringValue(conn.GetIpv4()),
			"ipv6":  types.StringValue(conn.GetIpv6()),
			"ports": ports,
		})

	return externalConnectionObj, nil
}

func buildExternalConnectionWithPortsListFromApi(ctx context.Context, conn *api.ContainerResultExternalConnection) (types.Object, diag.Diagnostics) {
	if conn == nil {
		return types.ObjectNull(ExternalConnectionWithPortsObjectAttributeTypes()), nil
	}

	var ports []attr.Value
	for _, port := range conn.GetPorts() {
		allowlist, diags := toTypesStringList(ctx, port.GetAllowList())
		if diags.HasError() {
			return types.ObjectNull(ExternalConnectionWithPortsObjectAttributeTypes()), diags
		}

		internalPort := port.GetInternalPort()
		var protocol string
		p := port.GetProtocol()
		switch p {
		case api.ProtocolTcp:
			protocol = "TCP"
		case api.ProtocolUdp:
			protocol = "UDP"
		}

		externalPortObj := types.ObjectValueMust(
			ExternalConnectionPortsContainerObjectAttributeTypes(),
			map[string]attr.Value{
				"external_port": types.Int64Value(int64(port.GetExternalPort())),
				"internal_port": types.Int64Value(int64(*internalPort)),
				"protocol":      types.StringValue(protocol),
				"allowlist":     allowlist,
			})

		ports = append(ports, externalPortObj)

	}

	portsList := types.ListValueMust(types.ObjectType{AttrTypes: ExternalConnectionPortsContainerObjectAttributeTypes()}, ports)

	externalConnectionObj := types.ObjectValueMust(
		ExternalConnectionWithPortsObjectAttributeTypes(),
		map[string]attr.Value{
			"ipv4":  types.StringValue(conn.GetIpv4()),
			"ipv6":  types.StringValue(conn.GetIpv6()),
			"ports": portsList,
		})
	return externalConnectionObj, nil
}

func buildExternalConnectionInputMQ(ctx context.Context, plan messageQueueResource, state *messageQueueResource) *api.ExternalConnectionInput {
	var externalConnectionInputs api.ExternalConnectionInput

	if plan.ExternalConnection.IsNull() {
		externalConnectionInputs.State = api.StateAbsent
		externalConnectionInputs.Ports = []api.ExternalConnectionPortInput{}
		return &externalConnectionInputs
	}
	if plan.ExternalConnection.IsUnknown() {
		externalConnectionInputs.State = api.StateAbsent
		externalConnectionInputs.Ports = []api.ExternalConnectionPortInput{}
		return &externalConnectionInputs
	}

	var externalConnectionData messageQueueExternalConnectionResource
	diags := plan.ExternalConnection.As(ctx, &externalConnectionData, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil
	}

	var oldExternalConnectionData messageQueueExternalConnectionResource
	if state != nil {
		if !state.ExternalConnection.IsNull() && !state.ExternalConnection.IsUnknown() {
			diags = state.ExternalConnection.As(ctx, &oldExternalConnectionData, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil
			}
		}
	}

	var ports api.ExternalConnectionPortInput
	var externalConnectionPortsData messageQueueExternalConnectionPortsResource
	diags = externalConnectionData.Ports.As(ctx, &externalConnectionPortsData, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil
	}

	var oldExternalConnectionPortsData messageQueueExternalConnectionPortsResource
	if !oldExternalConnectionData.Ports.IsNull() && !oldExternalConnectionData.Ports.IsUnknown() {
		diags = oldExternalConnectionData.Ports.As(ctx, &oldExternalConnectionPortsData, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil
		}

		externalport := int(oldExternalConnectionPortsData.ExternalPort.ValueInt64())
		ports.ExternalPort = &externalport
	}

	var allowlist []api.AllowListInput
	if !oldExternalConnectionPortsData.ExternalPort.IsNull() && !oldExternalConnectionPortsData.ExternalPort.IsUnknown() {
		allowlist = buildAllowlistInput(ctx, &oldExternalConnectionPortsData.Allowlist, externalConnectionPortsData.Allowlist)
	}
	if oldExternalConnectionPortsData.ExternalPort.IsNull() {
		// Create ports object with correct allowlist and empty port
		ports.ExternalPort = nil

		newAllowlist := toStringArray(ctx, externalConnectionPortsData.Allowlist)
		for _, newIp := range newAllowlist {
			allowlist = append(allowlist, api.AllowListInput{
				Ip:    newIp,
				State: api.StatePresent,
			})
		}
	}
	if oldExternalConnectionPortsData.ExternalPort.IsUnknown() {
		// Create ports object with correct allowlist and empty port
		ports.ExternalPort = nil

		newAllowlist := toStringArray(ctx, externalConnectionPortsData.Allowlist)
		for _, newIp := range newAllowlist {
			allowlist = append(allowlist, api.AllowListInput{
				Ip:    newIp,
				State: api.StatePresent,
			})
		}
	}

	ports.AllowList = allowlist
	ports.Protocol = api.ProtocolTcp
	ports.State = api.StatePresent
	externalConnectionInputs.Ports = []api.ExternalConnectionPortInput{ports}
	externalConnectionInputs.SharedIp = true
	externalConnectionInputs.State = api.StatePresent

	return &externalConnectionInputs
}

func buildExternalConnectionInputContainer(ctx context.Context, plan containerResource, state *containerResource) (*api.ExternalConnectionInput, diag.Diagnostics) {
	var externalConnectionInputs api.ExternalConnectionInput
	var diags diag.Diagnostics

	if plan.ExternalConnection.IsNull() {
		externalConnectionInputs.State = api.StateAbsent
		externalConnectionInputs.Ports = []api.ExternalConnectionPortInput{}
		return &externalConnectionInputs, nil
	}
	if plan.ExternalConnection.IsUnknown() {
		externalConnectionInputs.State = api.StateAbsent
		externalConnectionInputs.Ports = []api.ExternalConnectionPortInput{}
		return &externalConnectionInputs, nil
	}

	var externalConnectionData containerExternalConnectionResource
	diags = plan.ExternalConnection.As(ctx, &externalConnectionData, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}

	var oldExternalConnectionData containerExternalConnectionResource
	if state != nil {
		if !state.ExternalConnection.IsNull() && !state.ExternalConnection.IsUnknown() {
			diags = state.ExternalConnection.As(ctx, &oldExternalConnectionData, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}
		}
	}

	externalConnectionInputs.State = api.StatePresent
	externalConnectionInputs.SharedIp = true
	// State present

	var oldExternalConnectionPortsData []containerExternalConnectionPortsResource
	oldPortsArray := map[string]containerExternalConnectionPortsResource{}
	newPortsArray := map[string]containerExternalConnectionPortsResource{}
	var ports []api.ExternalConnectionPortInput
	if !oldExternalConnectionData.Ports.IsNull() && !oldExternalConnectionData.Ports.IsUnknown() {
		diags = oldExternalConnectionData.Ports.ElementsAs(ctx, &oldExternalConnectionPortsData, true)
		if diags.HasError() {
			return nil, diags
		}

		for _, port := range oldExternalConnectionPortsData {
			id := fmt.Sprintf("%v:%v", port.InternalPort.ValueInt64(), port.Protocol.ValueString())
			oldPortsArray[id] = port
		}
	}

	externalConnectionPortsData := make([]containerExternalConnectionPortsResource, len(externalConnectionData.Ports.Elements()))
	diags = externalConnectionData.Ports.ElementsAs(ctx, &externalConnectionPortsData, true)
	if diags.HasError() {
		return nil, diags
	}

	for _, port := range externalConnectionPortsData {
		var portInput api.ExternalConnectionPortInput

		internalPort := int(port.InternalPort.ValueInt64())
		portInput.InternalPort = &internalPort
		switch port.Protocol.ValueString() {
		case "TCP":
			portInput.Protocol = api.ProtocolTcp
		case "UDP":
			portInput.Protocol = api.ProtocolUdp
		}

		allowlist := buildAllowlistInput(ctx, nil, port.Allowlist)
		portInput.AllowList = allowlist
		portInput.State = api.StatePresent

		// Update allow list if port already exists, otherwise use the allowlist defined in the plan
		id := fmt.Sprintf("%v:%v", port.InternalPort.ValueInt64(), port.Protocol.ValueString())
		if _, exists := oldPortsArray[id]; exists {
			externalPort := int(oldPortsArray[id].ExternalPort.ValueInt64())
			portInput.ExternalPort = &externalPort

			oldAllowlist := oldPortsArray[id].Allowlist
			portInput.AllowList = buildAllowlistInput(ctx, &oldAllowlist, port.Allowlist)
		}

		newPortsArray[id] = port
		ports = append(ports, portInput)

	}

	for id, port := range oldPortsArray {
		if _, exists := newPortsArray[id]; !exists {
			var portInput api.ExternalConnectionPortInput
			internalPort := int(port.InternalPort.ValueInt64())
			portInput.InternalPort = &internalPort
			externalPort := int(port.ExternalPort.ValueInt64())
			portInput.ExternalPort = &externalPort
			switch port.Protocol.ValueString() {
			case "TCP":
				portInput.Protocol = api.ProtocolTcp
			case "UDP":
				portInput.Protocol = api.ProtocolUdp
			}
			portInput.State = api.StateAbsent
			portInput.AllowList = buildAllowlistInput(ctx, nil, port.Allowlist)
			ports = append(ports, portInput)
		}
	}
	externalConnectionInputs.Ports = ports

	return &externalConnectionInputs, nil
}

func buildExternalConnectionInputStarterContainer(ctx context.Context, plan starterContainerResource, state *starterContainerResource) (*api.ExternalConnectionInput, diag.Diagnostics) {
	var externalConnectionInputs api.ExternalConnectionInput
	var diags diag.Diagnostics

	if plan.ExternalConnection.IsNull() {
		externalConnectionInputs.State = api.StateAbsent
		externalConnectionInputs.Ports = []api.ExternalConnectionPortInput{}
		return &externalConnectionInputs, nil
	}
	if plan.ExternalConnection.IsUnknown() {
		externalConnectionInputs.State = api.StateAbsent
		externalConnectionInputs.Ports = []api.ExternalConnectionPortInput{}
		return &externalConnectionInputs, nil
	}

	var externalConnectionData containerExternalConnectionResource
	diags = plan.ExternalConnection.As(ctx, &externalConnectionData, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}

	var oldExternalConnectionData containerExternalConnectionResource
	if state != nil {
		if !state.ExternalConnection.IsNull() && !state.ExternalConnection.IsUnknown() {
			diags = state.ExternalConnection.As(ctx, &oldExternalConnectionData, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}
		}
	}

	externalConnectionInputs.State = api.StatePresent
	externalConnectionInputs.SharedIp = true
	// State present

	var oldExternalConnectionPortsData []containerExternalConnectionPortsResource
	oldPortsArray := map[string]containerExternalConnectionPortsResource{}
	newPortsArray := map[string]containerExternalConnectionPortsResource{}
	var ports []api.ExternalConnectionPortInput
	if !oldExternalConnectionData.Ports.IsNull() && !oldExternalConnectionData.Ports.IsUnknown() {
		diags = oldExternalConnectionData.Ports.ElementsAs(ctx, &oldExternalConnectionPortsData, true)
		if diags.HasError() {
			return nil, diags
		}

		for _, port := range oldExternalConnectionPortsData {
			id := fmt.Sprintf("%v:%v", port.InternalPort.ValueInt64(), port.Protocol.ValueString())
			oldPortsArray[id] = port
		}
	}

	externalConnectionPortsData := make([]containerExternalConnectionPortsResource, len(externalConnectionData.Ports.Elements()))
	diags = externalConnectionData.Ports.ElementsAs(ctx, &externalConnectionPortsData, true)
	if diags.HasError() {
		return nil, diags
	}

	for _, port := range externalConnectionPortsData {
		var portInput api.ExternalConnectionPortInput

		internalPort := int(port.InternalPort.ValueInt64())
		portInput.InternalPort = &internalPort
		switch port.Protocol.ValueString() {
		case "TCP":
			portInput.Protocol = api.ProtocolTcp
		case "UDP":
			portInput.Protocol = api.ProtocolUdp
		}

		allowlist := buildAllowlistInput(ctx, nil, port.Allowlist)
		portInput.AllowList = allowlist
		portInput.State = api.StatePresent

		// Update allow list if port already exists, otherwise use the allowlist defined in the plan
		id := fmt.Sprintf("%v:%v", port.InternalPort.ValueInt64(), port.Protocol.ValueString())
		if _, exists := oldPortsArray[id]; exists {
			externalPort := int(oldPortsArray[id].ExternalPort.ValueInt64())
			portInput.ExternalPort = &externalPort

			oldAllowlist := oldPortsArray[id].Allowlist
			portInput.AllowList = buildAllowlistInput(ctx, &oldAllowlist, port.Allowlist)
		}

		newPortsArray[id] = port
		ports = append(ports, portInput)

	}

	for id, port := range oldPortsArray {
		if _, exists := newPortsArray[id]; !exists {
			var portInput api.ExternalConnectionPortInput
			internalPort := int(port.InternalPort.ValueInt64())
			portInput.InternalPort = &internalPort
			externalPort := int(port.ExternalPort.ValueInt64())
			portInput.ExternalPort = &externalPort
			switch port.Protocol.ValueString() {
			case "TCP":
				portInput.Protocol = api.ProtocolTcp
			case "UDP":
				portInput.Protocol = api.ProtocolUdp
			}
			portInput.State = api.StateAbsent
			portInput.AllowList = buildAllowlistInput(ctx, nil, port.Allowlist)
			ports = append(ports, portInput)
		}
	}
	externalConnectionInputs.Ports = ports

	return &externalConnectionInputs, nil
}
