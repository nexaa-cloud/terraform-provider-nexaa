// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
	"github.com/stretchr/testify/assert"
)

func TestAcc_ExternalConnection_null(t *testing.T) {

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
