// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"fmt"
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
	plan.Plan = types.StringValue(cluster.Plan.GetId())
	plan.Spec = Spec{
		Type:    types.StringValue(cluster.Spec.GetType()),
		Version: types.StringValue(cluster.Spec.GetVersion()),
	}
	plan.State = types.StringValue(cluster.GetState())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	return plan
}

func generateCloudDatabaseClusterChildId(namespace string, cluster string, name string) string {
	return fmt.Sprintf("%s/%s/%s", namespace, cluster, name)
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

type PermissionListType struct {
	basetypes.ListType
}

func NewPermissionListType() PermissionListType {
	return PermissionListType{
		ListType: types.ListType{
			ElemType: NewPermissionType(),
		},
	}
}

type PermissionType struct {
	basetypes.ObjectType
}

func NewPermissionType() PermissionType {
	return PermissionType{
		ObjectType: types.ObjectType{
			AttrTypes: PermissionAttributes(),
		},
	}
}

type Permission struct {
	Type    types.String `tfsdk:"type"`
	Version types.String `tfsdk:"version"`
}

func PermissionAttributes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":    types.StringType,
		"version": types.StringType,
	}
}
