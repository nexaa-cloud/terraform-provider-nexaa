// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

// Tests that API errors returned during CRUD operations are surfaced in
// resp.Diagnostics rather than silently swallowed or causing a panic.
// Each test injects a MockNexaaAPI, exercises one error path in a resource
// method, and asserts resp.Diagnostics.HasError().

package resources

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
	nexaaclient "github.com/nexaa-cloud/terraform-provider-nexaa/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func namespaceTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{"delete": types.StringType}),
	}
}

func registryTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"delete": types.StringType,
		}),
	}
}

func buildNamespacePlan(t *testing.T, name string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	r := &namespaceResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	p := tfsdk.Plan{Schema: sr.Schema}
	diags := p.Set(ctx, &namespaceResource{
		ID:          types.StringNull(),
		Name:        types.StringValue(name),
		Description: types.StringNull(),
		Timeouts:    namespaceTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildNamespacePlan: %v", diags))
	return p
}

func buildNamespaceState(t *testing.T, name string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	r := &namespaceResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := tfsdk.State{Schema: sr.Schema}
	diags := s.Set(ctx, &namespaceResource{
		ID:          types.StringValue(name),
		Name:        types.StringValue(name),
		Description: types.StringNull(),
		Timeouts:    namespaceTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildNamespaceState: %v", diags))
	return s
}

func buildRegistryPlan(t *testing.T, namespace, name string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	r := &registryResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	p := tfsdk.Plan{Schema: sr.Schema}
	diags := p.Set(ctx, &registryResource{
		ID:        types.StringNull(),
		Namespace: types.StringValue(namespace),
		Name:      types.StringValue(name),
		Source:    types.StringValue("docker.io"),
		Username:  types.StringValue("user"),
		Password:  types.StringValue("pass"),
		Verify:    types.BoolValue(true),
		Locked:    types.BoolNull(),
		Status:    types.StringNull(),
		Timeouts:  registryTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildRegistryPlan: %v", diags))
	return p
}

func buildRegistryState(t *testing.T, namespace, name string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	r := &registryResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := tfsdk.State{Schema: sr.Schema}
	diags := s.Set(ctx, &registryResource{
		ID:        types.StringValue(name),
		Namespace: types.StringValue(namespace),
		Name:      types.StringValue(name),
		Source:    types.StringValue("docker.io"),
		Username:  types.StringValue("user"),
		Password:  types.StringValue("pass"),
		Verify:    types.BoolValue(true),
		Locked:    types.BoolValue(false),
		Status:    types.StringValue("active"),
		Timeouts:  registryTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildRegistryState: %v", diags))
	return s
}

// ── namespace ─────────────────────────────────────────────────────────────────

func Test_NamespaceCreate_preflight_unexpected_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("NamespaceListByName", "test-ns").Return(api.NamespaceResult{}, errors.New("connection refused"))

	r := &namespaceResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildNamespacePlan(t, "test-ns")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "connection refused")
}

func Test_NamespaceCreate_already_exists_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("NamespaceListByName", "test-ns").Return(api.NamespaceResult{Name: "test-ns"}, nil)

	r := &namespaceResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildNamespacePlan(t, "test-ns")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "already exists")
}

func Test_NamespaceCreate_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("NamespaceListByName", "test-ns").Return(api.NamespaceResult{}, errors.New("not found"))
	m.On("NamespaceCreate", mock.Anything).Return(api.NamespaceResult{}, errors.New("validation failed: name too long"))

	r := &namespaceResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildNamespacePlan(t, "test-ns")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "validation failed")
}

func Test_NamespaceRead_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("NamespaceListByName", "test-ns").Return(api.NamespaceResult{}, errors.New("internal server error"))

	r := &namespaceResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.ReadResponse{}
	r.Read(context.Background(), resource.ReadRequest{State: buildNamespaceState(t, "test-ns")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "internal server error")
}

func Test_NamespaceUpdate_always_surfaces_error(t *testing.T) {
	r := &namespaceResource{nexaaClient: nexaaclient.NewWithAPI(nil)}
	resp := &resource.UpdateResponse{}
	r.Update(context.Background(), resource.UpdateRequest{Plan: buildNamespacePlan(t, "test-ns")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "namespace")
}

// ── registry ──────────────────────────────────────────────────────────────────

func Test_RegistryCreate_preflight_unexpected_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListRegistryByName", "test-ns", "my-reg").Return((*api.RegistryResult)(nil), errors.New("internal server error"))

	r := &registryResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildRegistryPlan(t, "test-ns", "my-reg")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "internal server error")
}

func Test_RegistryCreate_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListRegistryByName", "test-ns", "my-reg").Return((*api.RegistryResult)(nil), errors.New("not found"))
	m.On("RegistryCreate", mock.Anything).Return(api.RegistryResult{}, errors.New("validation failed: invalid source URL"))

	r := &registryResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildRegistryPlan(t, "test-ns", "my-reg")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "validation failed")
}

func Test_RegistryUpdate_always_surfaces_error(t *testing.T) {
	r := &registryResource{nexaaClient: nexaaclient.NewWithAPI(nil)}
	resp := &resource.UpdateResponse{}
	r.Update(context.Background(), resource.UpdateRequest{Plan: buildRegistryPlan(t, "test-ns", "my-reg")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "update")
}

func Test_RegistryRead_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListRegistryByName", "test-ns", "my-reg").Return((*api.RegistryResult)(nil), errors.New("network timeout"))

	r := &registryResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.ReadResponse{}
	r.Read(context.Background(), resource.ReadRequest{State: buildRegistryState(t, "test-ns", "my-reg")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "network timeout")
}

// ── volume ────────────────────────────────────────────────────────────────────

func volumeTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{"delete": types.StringType}),
	}
}

func buildVolumePlan(t *testing.T, namespace, name string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	r := &volumeResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	p := tfsdk.Plan{Schema: sr.Schema}
	diags := p.Set(ctx, &volumeResource{
		ID:        types.StringNull(),
		Namespace: types.StringValue(namespace),
		Name:      types.StringValue(name),
		Size:      types.Int64Value(1),
		Usage:     types.Float64Value(0),
		Locked:    types.BoolNull(),
		Status:    types.StringNull(),
		Timeouts:  volumeTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildVolumePlan: %v", diags))
	return p
}

func buildVolumeState(t *testing.T, namespace, name string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	r := &volumeResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := tfsdk.State{Schema: sr.Schema}
	diags := s.Set(ctx, &volumeResource{
		ID:        types.StringValue(name),
		Namespace: types.StringValue(namespace),
		Name:      types.StringValue(name),
		Size:      types.Int64Value(1),
		Usage:     types.Float64Value(0),
		Locked:    types.BoolValue(false),
		Status:    types.StringValue("active"),
		Timeouts:  volumeTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildVolumeState: %v", diags))
	return s
}

func Test_VolumeCreate_preflight_unexpected_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListVolumeByName", "test-ns", "my-vol").Return((*api.VolumeResult)(nil), errors.New("connection refused"))

	r := &volumeResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildVolumePlan(t, "test-ns", "my-vol")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "connection refused")
}

func Test_VolumeCreate_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListVolumeByName", "test-ns", "my-vol").Return((*api.VolumeResult)(nil), nil)
	m.On("VolumeCreate", mock.Anything).Return(api.VolumeResult{}, errors.New("quota exceeded"))

	r := &volumeResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildVolumePlan(t, "test-ns", "my-vol")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "quota exceeded")
}

func Test_VolumeRead_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListVolumeByName", "test-ns", "my-vol").Return((*api.VolumeResult)(nil), errors.New("internal server error"))

	r := &volumeResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.ReadResponse{}
	r.Read(context.Background(), resource.ReadRequest{State: buildVolumeState(t, "test-ns", "my-vol")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "internal server error")
}

// ── message queue ─────────────────────────────────────────────────────────────

func messageQueueTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		}),
	}
}

func buildMessageQueuePlan(t *testing.T, namespace, name string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	r := &messageQueueResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	p := tfsdk.Plan{Schema: sr.Schema}

	mqExtConnAttrTypes := map[string]attr.Type{
		"ipv4": types.StringType,
		"ipv6": types.StringType,
		"ports": types.ObjectType{AttrTypes: map[string]attr.Type{
			"external_port": types.Int64Type,
			"allowlist":     types.ListType{ElemType: types.StringType},
		}},
	}

	diags := p.Set(ctx, &messageQueueResource{
		ID:                 types.StringNull(),
		Namespace:          types.StringValue(namespace),
		Name:               types.StringValue(name),
		Plan:               types.StringValue("small"),
		Type:               types.StringValue("RabbitMQ"),
		Version:            types.StringValue("3.13"),
		ExternalConnection: types.ObjectNull(mqExtConnAttrTypes),
		State:              types.StringNull(),
		Locked:             types.BoolNull(),
		Allowlist:          types.ListNull(types.StringType),
		AdminUser:          types.ObjectNull(MessageQueueAdminUserObjectAttributeTypes()),
		Timeouts:           messageQueueTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildMessageQueuePlan: %v", diags))
	return p
}

func buildMessageQueueState(t *testing.T, namespace, name string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	r := &messageQueueResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := tfsdk.State{Schema: sr.Schema}

	mqExtConnAttrTypes := map[string]attr.Type{
		"ipv4": types.StringType,
		"ipv6": types.StringType,
		"ports": types.ObjectType{AttrTypes: map[string]attr.Type{
			"external_port": types.Int64Type,
			"allowlist":     types.ListType{ElemType: types.StringType},
		}},
	}

	diags := s.Set(ctx, &messageQueueResource{
		ID:                 types.StringValue(namespace + "/" + name),
		Namespace:          types.StringValue(namespace),
		Name:               types.StringValue(name),
		Plan:               types.StringValue("small"),
		Type:               types.StringValue("RabbitMQ"),
		Version:            types.StringValue("3.13"),
		ExternalConnection: types.ObjectNull(mqExtConnAttrTypes),
		State:              types.StringValue("active"),
		Locked:             types.BoolValue(false),
		Allowlist:          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("0.0.0.0/0"), types.StringValue("::/0")}),
		AdminUser:          types.ObjectNull(MessageQueueAdminUserObjectAttributeTypes()),
		Timeouts:           messageQueueTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildMessageQueueState: %v", diags))
	return s
}

func Test_MessageQueueCreate_preflight_unexpected_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("MessageQueueGet", api.MessageQueueResourceInput{Name: "my-mq", Namespace: "test-ns"}).
		Return(api.MessageQueueResult{}, errors.New("connection refused"))

	r := &messageQueueResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildMessageQueuePlan(t, "test-ns", "my-mq")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "connection refused")
}

func Test_MessageQueueCreate_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("MessageQueueGet", api.MessageQueueResourceInput{Name: "my-mq", Namespace: "test-ns"}).
		Return(api.MessageQueueResult{}, errors.New("not found"))
	m.On("MessageQueueCreate", mock.Anything).Return(api.MessageQueueResult{}, errors.New("quota exceeded"))

	r := &messageQueueResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildMessageQueuePlan(t, "test-ns", "my-mq")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "quota exceeded")
}

func Test_MessageQueueRead_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("MessageQueueGet", api.MessageQueueResourceInput{Name: "my-mq", Namespace: "test-ns"}).
		Return(api.MessageQueueResult{}, errors.New("internal server error"))

	r := &messageQueueResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.ReadResponse{}
	r.Read(context.Background(), resource.ReadRequest{State: buildMessageQueueState(t, "test-ns", "my-mq")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "internal server error")
}

// ── container job ─────────────────────────────────────────────────────────────

func containerJobTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		}),
	}
}

func buildContainerJobPlan(t *testing.T, namespace, name string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	r := &containerJobResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	p := tfsdk.Plan{Schema: sr.Schema}
	diags := p.Set(ctx, &containerJobResource{
		ID:                   types.StringNull(),
		Name:                 types.StringValue(name),
		Namespace:            types.StringValue(namespace),
		Image:                types.StringValue("nginx:latest"),
		Registry:             types.StringNull(),
		Resources:            types.StringValue("cpu250-ram500"),
		EnvironmentVariables: types.SetNull(envVarObjectType()),
		Command:              types.ListNull(types.StringType),
		Entrypoint:           types.ListNull(types.StringType),
		Mounts:               types.ListNull(MountsObjectType()),
		Schedule:             types.StringValue("0 4 * * *"),
		Enabled:              types.BoolValue(true),
		State:                types.StringNull(),
		Timeouts:             containerJobTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildContainerJobPlan: %v", diags))
	return p
}

func buildContainerJobState(t *testing.T, namespace, name string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	r := &containerJobResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := tfsdk.State{Schema: sr.Schema}
	diags := s.Set(ctx, &containerJobResource{
		ID:                   types.StringValue(name),
		Name:                 types.StringValue(name),
		Namespace:            types.StringValue(namespace),
		Image:                types.StringValue("nginx:latest"),
		Registry:             types.StringNull(),
		Resources:            types.StringValue("cpu250-ram500"),
		EnvironmentVariables: types.SetNull(envVarObjectType()),
		Command:              types.ListNull(types.StringType),
		Entrypoint:           types.ListNull(types.StringType),
		Mounts:               types.ListNull(MountsObjectType()),
		Schedule:             types.StringValue("0 4 * * *"),
		Enabled:              types.BoolValue(true),
		State:                types.StringValue("active"),
		Timeouts:             containerJobTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildContainerJobState: %v", diags))
	return s
}

func Test_ContainerJobCreate_already_exists_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ContainerJobByName", "test-ns", "my-job").Return(api.ContainerJobResult{Name: "my-job"}, nil)

	r := &containerJobResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildContainerJobPlan(t, "test-ns", "my-job")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "already exists")
}

func Test_ContainerJobCreate_preflight_unexpected_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ContainerJobByName", "test-ns", "my-job").Return(api.ContainerJobResult{}, errors.New("connection refused"))

	r := &containerJobResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildContainerJobPlan(t, "test-ns", "my-job")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "connection refused")
}

func Test_ContainerJobCreate_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ContainerJobByName", "test-ns", "my-job").Return(api.ContainerJobResult{}, errors.New("not found"))
	m.On("ContainerJobCreate", mock.Anything).Return(api.ContainerJobResult{}, errors.New("image not found"))

	r := &containerJobResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildContainerJobPlan(t, "test-ns", "my-job")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "image not found")
}

func Test_ContainerJobRead_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ContainerJobByName", "test-ns", "my-job").Return(api.ContainerJobResult{}, errors.New("internal server error"))

	r := &containerJobResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.ReadResponse{}
	r.Read(context.Background(), resource.ReadRequest{State: buildContainerJobState(t, "test-ns", "my-job")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "internal server error")
}

// ── cloud database cluster ────────────────────────────────────────────────────

func cloudDBClusterTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		}),
	}
}

func buildCloudDBClusterPlan(t *testing.T, namespace, name string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	r := &cloudDatabaseClusterResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	p := tfsdk.Plan{Schema: sr.Schema}
	diags := p.Set(ctx, &cloudDatabaseClusterResource{
		ID: types.StringNull(),
		Cluster: ClusterRef{
			Namespace: types.StringValue(namespace),
			Name:      types.StringValue(name),
		},
		Spec: Spec{
			Type:    types.StringValue("mysql"),
			Version: types.StringValue("8.0"),
		},
		Plan:               types.StringValue("starter"),
		Hostname:           types.StringNull(),
		ExternalConnection: types.ObjectNull(ExternalConnectionObjectAttributeTypes()),
		State:              types.StringNull(),
		Timeouts:           cloudDBClusterTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildCloudDBClusterPlan: %v", diags))
	return p
}

func buildCloudDBClusterState(t *testing.T, namespace, name string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	r := &cloudDatabaseClusterResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := tfsdk.State{Schema: sr.Schema}
	diags := s.Set(ctx, &cloudDatabaseClusterResource{
		ID: types.StringValue(namespace + "/" + name),
		Cluster: ClusterRef{
			Namespace: types.StringValue(namespace),
			Name:      types.StringValue(name),
		},
		Spec: Spec{
			Type:    types.StringValue("mysql"),
			Version: types.StringValue("8.0"),
		},
		Plan:               types.StringValue("starter"),
		Hostname:           types.StringValue("db.example.com"),
		ExternalConnection: types.ObjectNull(ExternalConnectionObjectAttributeTypes()),
		State:              types.StringValue("active"),
		Timeouts:           cloudDBClusterTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildCloudDBClusterState: %v", diags))
	return s
}

func Test_CloudDatabaseClusterCreate_preflight_unexpected_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("CloudDatabaseClusterGet", api.CloudDatabaseClusterResourceInput{Name: "my-cluster", Namespace: "test-ns"}).
		Return(api.CloudDatabaseClusterResult{}, errors.New("connection refused"))

	r := &cloudDatabaseClusterResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildCloudDBClusterPlan(t, "test-ns", "my-cluster")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "connection refused")
}

func Test_CloudDatabaseClusterCreate_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("CloudDatabaseClusterGet", api.CloudDatabaseClusterResourceInput{Name: "my-cluster", Namespace: "test-ns"}).
		Return(api.CloudDatabaseClusterResult{}, errors.New("not found"))
	m.On("CloudDatabaseClusterCreate", mock.Anything).Return(api.CloudDatabaseClusterResult{}, errors.New("quota exceeded"))

	r := &cloudDatabaseClusterResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildCloudDBClusterPlan(t, "test-ns", "my-cluster")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "quota exceeded")
}

func Test_CloudDatabaseClusterRead_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("CloudDatabaseClusterGet", api.CloudDatabaseClusterResourceInput{Name: "my-cluster", Namespace: "test-ns"}).
		Return(api.CloudDatabaseClusterResult{}, errors.New("internal server error"))

	r := &cloudDatabaseClusterResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.ReadResponse{}
	r.Read(context.Background(), resource.ReadRequest{State: buildCloudDBClusterState(t, "test-ns", "my-cluster")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "internal server error")
}

// ── cloud database cluster database ──────────────────────────────────────────

func cloudDBClusterDatabaseTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		}),
	}
}

func buildCloudDBClusterDatabasePlan(t *testing.T, namespace, clusterName, dbName string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	r := &cloudDatabaseClusterDatabaseResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	p := tfsdk.Plan{Schema: sr.Schema}
	diags := p.Set(ctx, &cloudDatabaseClusterDatabaseResource{
		ID: types.StringNull(),
		Cluster: ClusterRef{
			Namespace: types.StringValue(namespace),
			Name:      types.StringValue(clusterName),
		},
		Name:        types.StringValue(dbName),
		Description: types.StringNull(),
		Timeouts:    cloudDBClusterDatabaseTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildCloudDBClusterDatabasePlan: %v", diags))
	return p
}

func buildCloudDBClusterDatabaseState(t *testing.T, namespace, clusterName, dbName string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	r := &cloudDatabaseClusterDatabaseResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := tfsdk.State{Schema: sr.Schema}
	diags := s.Set(ctx, &cloudDatabaseClusterDatabaseResource{
		ID: types.StringValue(generateCloudDatabaseClusterDatabaseId(namespace, clusterName, dbName)),
		Cluster: ClusterRef{
			Namespace: types.StringValue(namespace),
			Name:      types.StringValue(clusterName),
		},
		Name:        types.StringValue(dbName),
		Description: types.StringNull(),
		Timeouts:    cloudDBClusterDatabaseTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildCloudDBClusterDatabaseState: %v", diags))
	return s
}

func Test_CloudDatabaseClusterDatabaseCreate_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	// waitForUnlocked calls CloudDatabaseClusterGet; return an unlocked cluster so it passes immediately.
	m.On("CloudDatabaseClusterGet", mock.Anything).Return(api.CloudDatabaseClusterResult{Id: "123", Locked: false}, nil)
	m.On("CloudDatabaseClusterDatabaseCreate", mock.Anything).Return(api.CloudDatabaseClusterDatabaseResult{}, errors.New("quota exceeded"))

	r := &cloudDatabaseClusterDatabaseResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildCloudDBClusterDatabasePlan(t, "test-ns", "my-cluster", "mydb")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "quota exceeded")
}

func Test_CloudDatabaseClusterDatabaseRead_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("CloudDatabaseClusterGet", mock.Anything).Return(api.CloudDatabaseClusterResult{}, errors.New("internal server error"))

	r := &cloudDatabaseClusterDatabaseResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.ReadResponse{}
	r.Read(context.Background(), resource.ReadRequest{State: buildCloudDBClusterDatabaseState(t, "test-ns", "my-cluster", "mydb")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "internal server error")
}

// ── cloud database cluster user ───────────────────────────────────────────────

func cloudDBClusterUserTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		}),
	}
}

func buildCloudDBClusterUserPlan(t *testing.T, namespace, clusterName, userName string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	r := &cloudDatabaseClusterUserResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	p := tfsdk.Plan{Schema: sr.Schema}

	permissionsAttrTypes := map[string]attr.Type{
		"database_name": types.StringType,
		"permission":    types.StringType,
		"state":         types.StringType,
	}

	diags := p.Set(ctx, &cloudDatabaseClusterUserResource{
		ID: types.StringNull(),
		Cluster: ClusterRef{
			Namespace: types.StringValue(namespace),
			Name:      types.StringValue(clusterName),
		},
		Name:        types.StringValue(userName),
		Password:    types.StringNull(),
		Permissions: types.SetNull(types.ObjectType{AttrTypes: permissionsAttrTypes}),
		Timeouts:    cloudDBClusterUserTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildCloudDBClusterUserPlan: %v", diags))
	return p
}

func buildCloudDBClusterUserState(t *testing.T, namespace, clusterName, userName string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	r := &cloudDatabaseClusterUserResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := tfsdk.State{Schema: sr.Schema}

	permissionsAttrTypes := map[string]attr.Type{
		"database_name": types.StringType,
		"permission":    types.StringType,
		"state":         types.StringType,
	}

	diags := s.Set(ctx, &cloudDatabaseClusterUserResource{
		ID: types.StringValue(generateCloudDatabaseClusterUserId(namespace, clusterName, userName)),
		Cluster: ClusterRef{
			Namespace: types.StringValue(namespace),
			Name:      types.StringValue(clusterName),
		},
		Name:        types.StringValue(userName),
		Password:    types.StringNull(),
		Permissions: types.SetValueMust(types.ObjectType{AttrTypes: permissionsAttrTypes}, []attr.Value{}),
		Timeouts:    cloudDBClusterUserTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildCloudDBClusterUserState: %v", diags))
	return s
}

func Test_CloudDatabaseClusterUserCreate_userlist_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	// waitForUnlocked calls CloudDatabaseClusterGet; return an unlocked cluster so it passes.
	m.On("CloudDatabaseClusterGet", mock.Anything).Return(api.CloudDatabaseClusterResult{Id: "123", Locked: false}, nil)
	m.On("CloudDatabaseClusterUserList", mock.Anything).Return(nil, errors.New("connection refused"))

	r := &cloudDatabaseClusterUserResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildCloudDBClusterUserPlan(t, "test-ns", "my-cluster", "myuser")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "connection refused")
}

func Test_CloudDatabaseClusterUserCreate_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	// waitForUnlocked calls CloudDatabaseClusterGet; return an unlocked cluster so it passes.
	m.On("CloudDatabaseClusterGet", mock.Anything).Return(api.CloudDatabaseClusterResult{Id: "123", Locked: false}, nil)
	m.On("CloudDatabaseClusterUserList", mock.Anything).Return([]api.CloudDatabaseClusterUserResult{}, nil)
	m.On("CloudDatabaseClusterUserCreate", mock.Anything).Return(api.CloudDatabaseClusterUserResult{}, errors.New("create failed"))

	r := &cloudDatabaseClusterUserResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildCloudDBClusterUserPlan(t, "test-ns", "my-cluster", "myuser")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "create failed")
}

func Test_CloudDatabaseClusterUserRead_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("CloudDatabaseClusterUserGet", mock.Anything, "myuser").Return(api.CloudDatabaseClusterUserResult{}, errors.New("internal server error"))

	r := &cloudDatabaseClusterUserResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.ReadResponse{}
	r.Read(context.Background(), resource.ReadRequest{State: buildCloudDBClusterUserState(t, "test-ns", "my-cluster", "myuser")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "internal server error")
}

// ── container ─────────────────────────────────────────────────────────────────

func containerTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		}),
	}
}

func buildContainerScalingObj() types.Object {
	autoInputAttrTypes := map[string]attr.Type{
		"minimal_replicas": types.Int64Type,
		"maximal_replicas": types.Int64Type,
		"triggers": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
			"type":      types.StringType,
			"threshold": types.Int64Type,
		}}},
	}
	scalingAttrTypes := map[string]attr.Type{
		"type":         types.StringType,
		"manual_input": types.Int64Type,
		"auto_input":   types.ObjectType{AttrTypes: autoInputAttrTypes},
	}
	return types.ObjectValueMust(scalingAttrTypes, map[string]attr.Value{
		"type":         types.StringValue("manual"),
		"manual_input": types.Int64Value(1),
		"auto_input":   types.ObjectNull(autoInputAttrTypes),
	})
}

func buildContainerPlan(t *testing.T, namespace, name string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	r := &containerResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	p := tfsdk.Plan{Schema: sr.Schema}
	diags := p.Set(ctx, &containerResource{
		ID:                   types.StringNull(),
		Name:                 types.StringValue(name),
		Namespace:            types.StringValue(namespace),
		Image:                types.StringValue("nginx:latest"),
		Registry:             types.StringNull(),
		Resources:            types.StringValue("cpu250-ram500"),
		Command:              types.ListNull(types.StringType),
		Entrypoint:           types.ListNull(types.StringType),
		EnvironmentVariables: types.SetNull(envVarObjectType()),
		Ports:                types.ListNull(types.StringType),
		Ingresses:            types.ListNull(IngressObjectType()),
		ExternalConnection:   types.ObjectNull(ExternalConnectionWithPortsObjectAttributeTypes()),
		Mounts:               types.ListNull(MountsObjectType()),
		HealthCheck:          types.ObjectNull(map[string]attr.Type{"port": types.Int64Type, "path": types.StringType}),
		Scaling:              buildContainerScalingObj(),
		Status:               types.StringNull(),
		Timeouts:             containerTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildContainerPlan: %v", diags))
	return p
}

func buildContainerState(t *testing.T, namespace, name string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	r := &containerResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := tfsdk.State{Schema: sr.Schema}
	diags := s.Set(ctx, &containerResource{
		ID:                   types.StringValue(name),
		Name:                 types.StringValue(name),
		Namespace:            types.StringValue(namespace),
		Image:                types.StringValue("nginx:latest"),
		Registry:             types.StringNull(),
		Resources:            types.StringValue("cpu250-ram500"),
		Command:              types.ListNull(types.StringType),
		Entrypoint:           types.ListNull(types.StringType),
		EnvironmentVariables: types.SetNull(envVarObjectType()),
		Ports:                types.ListNull(types.StringType),
		Ingresses:            types.ListNull(IngressObjectType()),
		ExternalConnection:   types.ObjectNull(ExternalConnectionWithPortsObjectAttributeTypes()),
		Mounts:               types.ListNull(MountsObjectType()),
		HealthCheck:          types.ObjectNull(map[string]attr.Type{"port": types.Int64Type, "path": types.StringType}),
		Scaling:              buildContainerScalingObj(),
		Status:               types.StringValue("running"),
		Timeouts:             containerTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildContainerState: %v", diags))
	return s
}

func Test_ContainerCreate_already_exists_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListContainerByName", "test-ns", "my-container").Return(api.ContainerResult{Name: "my-container"}, nil)

	r := &containerResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildContainerPlan(t, "test-ns", "my-container")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "already exists")
}

func Test_ContainerCreate_preflight_unexpected_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListContainerByName", "test-ns", "my-container").Return(api.ContainerResult{}, errors.New("connection refused"))

	r := &containerResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildContainerPlan(t, "test-ns", "my-container")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "connection refused")
}

func Test_ContainerCreate_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListContainerByName", "test-ns", "my-container").Return(api.ContainerResult{}, errors.New("not found"))
	m.On("ContainerCreate", mock.Anything).Return(api.ContainerResult{}, errors.New("image not found"))

	r := &containerResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildContainerPlan(t, "test-ns", "my-container")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "image not found")
}

func Test_ContainerRead_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListContainerByName", "test-ns", "my-container").Return(api.ContainerResult{}, errors.New("internal server error"))

	r := &containerResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.ReadResponse{}
	r.Read(context.Background(), resource.ReadRequest{State: buildContainerState(t, "test-ns", "my-container")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "internal server error")
}

// ── starter container ─────────────────────────────────────────────────────────

func buildStarterContainerPlan(t *testing.T, namespace, name string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	r := &starterContainerResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	p := tfsdk.Plan{Schema: sr.Schema}
	diags := p.Set(ctx, &starterContainerResource{
		ID:                   types.StringNull(),
		Name:                 types.StringValue(name),
		Namespace:            types.StringValue(namespace),
		Image:                types.StringValue("nginx:latest"),
		Registry:             types.StringNull(),
		Command:              types.ListNull(types.StringType),
		Entrypoint:           types.ListNull(types.StringType),
		EnvironmentVariables: types.SetNull(envVarObjectType()),
		Ports:                types.ListNull(types.StringType),
		Ingresses:            types.ListNull(IngressObjectType()),
		ExternalConnection:   types.ObjectNull(ExternalConnectionWithPortsObjectAttributeTypes()),
		Mounts:               types.ListNull(MountsObjectType()),
		HealthCheck:          types.ObjectNull(map[string]attr.Type{"port": types.Int64Type, "path": types.StringType}),
		Status:               types.StringNull(),
		Timeouts:             containerTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildStarterContainerPlan: %v", diags))
	return p
}

func buildStarterContainerState(t *testing.T, namespace, name string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	r := &starterContainerResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := tfsdk.State{Schema: sr.Schema}
	diags := s.Set(ctx, &starterContainerResource{
		ID:                   types.StringValue(name),
		Name:                 types.StringValue(name),
		Namespace:            types.StringValue(namespace),
		Image:                types.StringValue("nginx:latest"),
		Registry:             types.StringNull(),
		Command:              types.ListNull(types.StringType),
		Entrypoint:           types.ListNull(types.StringType),
		EnvironmentVariables: types.SetNull(envVarObjectType()),
		Ports:                types.ListNull(types.StringType),
		Ingresses:            types.ListNull(IngressObjectType()),
		ExternalConnection:   types.ObjectNull(ExternalConnectionWithPortsObjectAttributeTypes()),
		Mounts:               types.ListNull(MountsObjectType()),
		HealthCheck:          types.ObjectNull(map[string]attr.Type{"port": types.Int64Type, "path": types.StringType}),
		Status:               types.StringValue("running"),
		Timeouts:             containerTimeouts(),
	})
	require.False(t, diags.HasError(), fmt.Sprintf("buildStarterContainerState: %v", diags))
	return s
}

func Test_StarterContainerCreate_already_exists_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListContainerByName", "test-ns", "my-starter").Return(api.ContainerResult{Name: "my-starter"}, nil)

	r := &starterContainerResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildStarterContainerPlan(t, "test-ns", "my-starter")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "already exists")
}

func Test_StarterContainerCreate_preflight_unexpected_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListContainerByName", "test-ns", "my-starter").Return(api.ContainerResult{}, errors.New("connection refused"))

	r := &starterContainerResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildStarterContainerPlan(t, "test-ns", "my-starter")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "connection refused")
}

func Test_StarterContainerCreate_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListContainerByName", "test-ns", "my-starter").Return(api.ContainerResult{}, errors.New("not found"))
	m.On("ContainerCreate", mock.Anything).Return(api.ContainerResult{}, errors.New("image not found"))

	r := &starterContainerResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{Plan: buildStarterContainerPlan(t, "test-ns", "my-starter")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "image not found")
}

func Test_StarterContainerRead_api_error_surfaced(t *testing.T) {
	m := new(nexaaclient.MockNexaaAPI)
	m.On("ListContainerByName", "test-ns", "my-starter").Return(api.ContainerResult{}, errors.New("internal server error"))

	r := &starterContainerResource{nexaaClient: nexaaclient.NewWithAPI(m)}
	resp := &resource.ReadResponse{}
	r.Read(context.Background(), resource.ReadRequest{State: buildStarterContainerState(t, "test-ns", "my-starter")}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "internal server error")
}
