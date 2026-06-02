// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package data_sources

import (
	"testing"

	"github.com/nexaa-cloud/nexaa-cli/api"
	"github.com/stretchr/testify/assert"
)

// --- getMessageQueuePlan ---

func Test_GetMessageQueuePlan_match_found(t *testing.T) {
	plans := []api.MessageQueuePlanResult{
		{Id: "plan-a", Cpu: 0.5, Memory: 1.0, Storage: 10.0, Replicas: 1},
		{Id: "plan-b", Cpu: 1.0, Memory: 2.0, Storage: 20.0, Replicas: 3},
	}
	result, err := getMessageQueuePlan(plans, 3, 1.0, 2.0, 20.0)
	assert.NoError(t, err)
	assert.Equal(t, "plan-b", result.Id.ValueString())
}

func Test_GetMessageQueuePlan_no_match_returns_error(t *testing.T) {
	plans := []api.MessageQueuePlanResult{
		{Id: "plan-a", Cpu: 0.5, Memory: 1.0, Storage: 10.0, Replicas: 1},
	}
	_, err := getMessageQueuePlan(plans, 3, 99.0, 99.0, 99.0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No plan found")
	assert.Contains(t, err.Error(), "plan-a")
}

func Test_GetMessageQueuePlan_empty_plans_returns_error(t *testing.T) {
	_, err := getMessageQueuePlan([]api.MessageQueuePlanResult{}, 1, 1.0, 1.0, 1.0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No plan found")
}

// --- getPlan ---

func Test_GetPlan_match_found(t *testing.T) {
	plans := []api.CloudDatabaseClusterPlan{
		{Id: "db-plan-1", Group: "Single (1 node)", Cpu: 2, Memory: 4, Storage: 20},
		{Id: "db-plan-2", Group: "Highly available (3 nodes)", Cpu: 4, Memory: 8, Storage: 40},
	}
	result, err := getPlan(plans, 1, 2, 4, 20)
	assert.NoError(t, err)
	assert.Equal(t, "db-plan-1", result.Id.ValueString())
}

func Test_GetPlan_no_match_returns_error(t *testing.T) {
	plans := []api.CloudDatabaseClusterPlan{
		{Id: "db-plan-1", Group: "Single (1 node)", Cpu: 2, Memory: 4, Storage: 20},
	}
	_, err := getPlan(plans, 1, 99, 99, 99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No plan found")
	assert.Contains(t, err.Error(), "db-plan-1")
}

func Test_GetPlan_replicas_mismatch_skips_plan(t *testing.T) {
	plans := []api.CloudDatabaseClusterPlan{
		{Id: "db-plan-ha", Group: "Highly available (3 nodes)", Cpu: 4, Memory: 8, Storage: 40},
	}
	// replicas=1 → Group="Single (1 node)", but the plan is HA
	_, err := getPlan(plans, 1, 4, 8, 40)
	assert.Error(t, err)
}

// --- translateReplicasToGroup ---

func Test_TranslateReplicasToGroup_1_is_single(t *testing.T) {
	assert.Equal(t, "Single (1 node)", translateReplicasToGroup(1))
}

func Test_TranslateReplicasToGroup_2_is_redundant(t *testing.T) {
	assert.Equal(t, "Redundant (2 nodes)", translateReplicasToGroup(2))
}

func Test_TranslateReplicasToGroup_3_is_highly_available(t *testing.T) {
	assert.Equal(t, "Highly available (3 nodes)", translateReplicasToGroup(3))
}

func Test_TranslateReplicasToGroup_default_falls_back_to_single(t *testing.T) {
	assert.Equal(t, "Single (1 node)", translateReplicasToGroup(99))
}
