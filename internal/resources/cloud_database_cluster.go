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

func translateReplicasToGroup(Replicas int64) string {
	switch Replicas {
	case 1:
		return "Single (1 node)"
	case 2:
		return "Redundant (2 nodes)"
	case 3:
		return "Highly available (3 nodes)"
	default:
		return "Single (1 node)" // fallback
	}
}

func translateGroupToReplicas(Group string) int {
	switch Group {
	case "Single (1 node)":
		return 1
	case "Redundant (2 nodes)":
		return 2
	case "Highly available (3 nodes)":
		return 3
	default:
		return 1 // fallback
	}
}

func translateApiToCloudDatabaseClusterResource(plan cloudDatabaseClusterResource, cluster api.CloudDatabaseClusterResult) cloudDatabaseClusterResource {
	namespace := cluster.GetNamespace()
	plan.ID = types.StringValue(generateCloudDatabaseClusterId(namespace.GetName(), cluster.GetName()))
	plan.Cluster = ClusterRef{
		Name:      types.StringValue(cluster.Name),
		Namespace: types.StringValue(namespace.GetName()),
	}
	plan.Plan = Plan{
		Cpu:      types.Int64Value(int64(cluster.Plan.GetCpu())),
		Memory:   types.Int64Value(int64(cluster.Plan.GetMemory())),
		Storage:  types.Int64Value(int64(cluster.Plan.GetStorage())),
		Replicas: types.Int64Value(int64(translateGroupToReplicas(cluster.Plan.GetGroup()))),
	}
	plan.Spec = Spec{
		Type:    types.StringValue(cluster.Spec.GetType()),
		Version: types.StringValue(cluster.Spec.GetVersion()),
	}
	plan.State = types.StringValue(cluster.GetState())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	return plan
}

func getPlanId(client *api.Client, Replicas int64, Cpu int64, Memory int64, Storage int64) (string, error) {
	plans, err := client.CloudDatabaseClusterListPlans()
	if err != nil {
		return "", err
	}

	Group := translateReplicasToGroup(Replicas)
	var planId string
	for _, plan := range plans {
		if plan.Group != Group {
			continue
		}

		if plan.Cpu != int(Cpu) {
			continue
		}

		if int(plan.Memory) != int(Memory) {
			continue
		}

		if plan.Storage != int(Storage) {
			continue
		}

		planId = plan.Id
	}

	if planId == "" {
		var sb strings.Builder
		sb.WriteString("No plan found for the given parameters, These are the available plans: \n")
		sb.WriteString("ID\tNAME\tCPU\tSTORAGE\tRAM\tGROUP\t")
		for _, plan := range plans {
			sb.WriteString(fmt.Sprintf("%q \t%q \t%d \t%d \t%g \t%q \n", plan.Id, plan.Name, plan.Cpu, plan.Storage, plan.Memory, plan.Group))
		}

		return "", errors.New(sb.String())
	}

	return planId, nil
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
