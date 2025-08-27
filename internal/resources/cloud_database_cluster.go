// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type ClusterRefType struct {
	basetypes.ObjectType
}

// NewClusterRefType returns the concrete custom type for the cluster object.
func NewClusterRefType() ClusterRefType {
	return ClusterRefType{
		ObjectType: types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"namespace": types.StringType,
				"name":      types.StringType,
			},
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

func generateCloudDatabaseClusterChildId(namespace string, cluster string, name string) string {
	return fmt.Sprintf("%s/%s/%s", namespace, cluster, name)
}
