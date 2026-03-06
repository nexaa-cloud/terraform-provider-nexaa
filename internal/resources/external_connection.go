package resources

import (
	"context"

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

func ExternalConnectionPortsObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"external_port": types.Int64Type,
		"allowlist":     types.ListType{ElemType: types.StringType},
	}
}

func buildExternalConnectionUpdateInput(ctx context.Context, plan cloudDatabaseClusterResource, state *cloudDatabaseClusterResource) *api.ExternalConnectionInput {
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

	var ports api.ExternalConnectionPortInput
	if externalConnectionData.Ports.IsNull() {
		ports.State = api.StateAbsent
		ports.AllowList = []api.AllowListInput{}
		ports.ExternalPort = nil
		return &externalConnectionInputs
	}
	if externalConnectionData.Ports.IsUnknown() {
		ports.State = api.StateAbsent
		ports.AllowList = []api.AllowListInput{}
		ports.ExternalPort = nil
		return &externalConnectionInputs
	}

	var externalConnectionPortsData cloudDatabaseClusterExternalConnectionPortsResource
	diags= externalConnectionData.Ports.As(ctx, &externalConnectionPortsData, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil
	}

	var oldExternalConnectionData cloudDatabaseClusterExternalConnectionResource
	var oldExternalConnectionPortsData cloudDatabaseClusterExternalConnectionPortsResource
	var allowlist []api.AllowListInput
	if !externalConnectionPortsData.ExternalPort.IsNull() {
		if !state.ExternalConnection.IsNull() {
			diags = state.ExternalConnection.As(ctx, &oldExternalConnectionData, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil
			}

			diags = oldExternalConnectionData.Ports.As(ctx, &oldExternalConnectionPortsData, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil
			}

			externalport := int(oldExternalConnectionPortsData.ExternalPort.ValueInt64())
			ports.ExternalPort = &externalport
		}

		newAllowlist := toStringArray(ctx, externalConnectionPortsData.Allowlist)
		oldAllowlist := toStringArray(ctx, oldExternalConnectionPortsData.Allowlist)
		plannedIps := map[string]struct{}{}

		for _, newIp := range newAllowlist {
			plannedIps[newIp] = struct{}{}
			allowlist = append(allowlist, api.AllowListInput{
				Ip:	newIp,
				State: api.StatePresent,
			})
		}

		for _, oldIp := range oldAllowlist {
			if _, exists := plannedIps[oldIp]; !exists {
				allowlist = append(allowlist, api.AllowListInput{
					Ip:	oldIp,
					State: api.StateAbsent,
				})
			}
		}
	}
	if externalConnectionPortsData.ExternalPort.IsNull() {
		oldExternalConnectionPortsData = cloudDatabaseClusterExternalConnectionPortsResource{}
	}

	ports.AllowList = allowlist
	ports.State = api.StatePresent
	externalConnectionInputs.Ports = []api.ExternalConnectionPortInput{ports}
	externalConnectionInputs.SharedIp = true
	externalConnectionInputs.State = api.StatePresent

	return &externalConnectionInputs
}

func buildExternalConnectionFromApi(ctx context.Context, conn api.ExternalConnectionResult) (types.Object, diag.Diagnostics) {	

	allowlist, diags := types.ListValueFrom(
		ctx,
		types.StringType,
		conn.GetPorts()[0].GetAllowList(),
	)
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








