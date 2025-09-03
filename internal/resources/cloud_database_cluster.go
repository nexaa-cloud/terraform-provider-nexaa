// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

func translateApiToCloudDatabaseClusterResource(plan cloudDatabaseClusterResource, cluster api.CloudDatabaseClusterResult) cloudDatabaseClusterResource {
	namespace := cluster.GetNamespace()
	plan.ID = types.StringValue(generateCloudDatabaseClusterId(namespace.GetName(), cluster.GetName()))
	plan.Cluster = ClusterRef{
		Name:      types.StringValue(cluster.Name),
		Namespace: types.StringValue(namespace.GetName()),
	}
	plan.Hostname = types.StringValue(cluster.Hostname)
	plan.Plan = types.StringValue(cluster.Plan.GetId())
	plan.Spec = Spec{
		Type:    types.StringValue(cluster.Spec.GetType()),
		Version: types.StringValue(cluster.Spec.GetVersion()),
	}
	plan.State = types.StringValue(cluster.GetState())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	return plan
}

func translateApiToCloudDatabaseClusterUserResource(plan cloudDatabaseClusterUserResource, cluster ClusterRef, user api.CloudDatabaseClusterUserResult) cloudDatabaseClusterUserResource {
	plan.ID = types.StringValue(generateCloudDatabaseClusterUserId(cluster.Namespace.ValueString(), cluster.Name.ValueString(), user.GetName()))
	plan.Name = types.StringValue(user.GetName())
	plan.Cluster = ClusterRef{
		Namespace: types.StringValue(cluster.Namespace.ValueString()),
		Name:      types.StringValue(cluster.Name.ValueString()),
	}
	var apiPermissions []map[string]attr.Value
	for _, permission := range user.Permissions {
		var databasePermission = "read_write"
		if permission.Permission == api.DatabasePermissionReadOnly {
			databasePermission = "read_only"
		}

		apiPermissions = append(apiPermissions, map[string]attr.Value{
			"database_name": types.StringValue(permission.DatabaseName),
			"permission":    types.StringValue(databasePermission),
			"state":         types.StringValue("present"),
		})
	}
	elementType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"database_name": types.StringType,
			"permission":    types.StringType,
			"state":         types.StringType,
		},
	}

	values := make([]attr.Value, 0, len(apiPermissions))
	for _, m := range apiPermissions {
		values = append(values, types.ObjectValueMust(elementType.AttrTypes, m))
	}
	plan.Permissions = types.SetValueMust(elementType, values)

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	return plan
}

type cloudDatabaseClusterChildId struct {
	Namespace string
	Cluster   string
	Name      string
}

func generateCloudDatabaseClusterChildId(namespace string, cluster string, typeName string, name string) string {
	return fmt.Sprintf("%s/%s/%s/%s", namespace, cluster, typeName, name)
}

func generateCloudDatabaseClusterDatabaseId(namespace string, cluster string, name string) string {
	return generateCloudDatabaseClusterChildId(namespace, cluster, "database", name)
}

func generateCloudDatabaseClusterUserId(namespace string, cluster string, name string) string {
	return generateCloudDatabaseClusterChildId(namespace, cluster, "user", name)
}

func unpackCloudDatabaseClusterChildId(id string) (cloudDatabaseClusterChildId, error) {
	parts := strings.SplitN(id, "/", 4)
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		return cloudDatabaseClusterChildId{}, errors.New(
			"Expected import ID in the format \"<namespace>/<cluster_name>/<type_name>/<child_name>\", got: " + id,
		)
	}

	namespace := parts[0]
	clusterName := parts[1]
	childName := parts[3]

	return cloudDatabaseClusterChildId{
		Namespace: namespace,
		Cluster:   clusterName,
		Name:      childName,
	}, nil
}

func generateCloudDatabaseClusterId(namespace string, cluster string) string {
	return fmt.Sprintf("%s/%s", namespace, cluster)
}

type ClusterRefType struct {
	basetypes.ObjectType
}

// NewClusterRefType returns the concrete custom type for the cluster object.
func NewClusterRefType() ClusterRefType {
	return ClusterRefType{
		ObjectType: types.ObjectType{
			AttrTypes: ClusterRefAttributes(),
		},
	}
}

func ClusterRefAttributes() map[string]attr.Type {
	return map[string]attr.Type{
		"namespace": types.StringType,
		"name":      types.StringType,
	}
}

// ClusterRef is a helper model for (de)serializing the cluster object value.
type ClusterRef struct {
	Namespace types.String `tfsdk:"namespace"`
	Name      types.String `tfsdk:"name"`
}

type PlanType struct {
	basetypes.ObjectType
}

func NewPlanType() PlanType {
	return PlanType{
		ObjectType: types.ObjectType{
			AttrTypes: PlanAttributes(),
		},
	}
}

type Plan struct {
	Replicas types.Int64 `tfsdk:"replicas"`
	Cpu      types.Int64 `tfsdk:"cpu"`
	Memory   types.Int64 `tfsdk:"memory"`
	Storage  types.Int64 `tfsdk:"storage"`
}

func PlanAttributes() map[string]attr.Type {
	return map[string]attr.Type{
		"replicas": types.Int64Type,
		"memory":   types.Int64Type,
		"storage":  types.Int64Type,
		"cpu":      types.Int64Type,
	}
}

type SpecType struct {
	basetypes.ObjectType
}

func NewSpecType() SpecType {
	return SpecType{
		ObjectType: types.ObjectType{
			AttrTypes: SpecAttributes(),
		},
	}
}

type Spec struct {
	Type    types.String `tfsdk:"type"`
	Version types.String `tfsdk:"version"`
}

func SpecAttributes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":    types.StringType,
		"version": types.StringType,
	}
}
