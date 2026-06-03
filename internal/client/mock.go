// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"github.com/nexaa-cloud/nexaa-cli/api"
	"github.com/stretchr/testify/mock"
)

// MockNexaaAPI is a testify mock implementation of NexaaAPI for use in unit tests.
type MockNexaaAPI struct {
	mock.Mock
}

func (m *MockNexaaAPI) NamespaceCreate(input api.NamespaceCreateInput) (api.NamespaceResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.NamespaceResult), args.Error(1)
}

func (m *MockNexaaAPI) NamespaceListByName(name string) (api.NamespaceResult, error) {
	args := m.Called(name)
	return args.Get(0).(api.NamespaceResult), args.Error(1)
}

func (m *MockNexaaAPI) NamespaceDelete(name string) (bool, error) {
	args := m.Called(name)
	return args.Bool(0), args.Error(1)
}

func (m *MockNexaaAPI) ContainerCreate(input api.ContainerCreateInput) (api.ContainerResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.ContainerResult), args.Error(1)
}

func (m *MockNexaaAPI) ContainerModify(input api.ContainerModifyInput) (api.ContainerResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.ContainerResult), args.Error(1)
}

func (m *MockNexaaAPI) ContainerDelete(namespace string, containerName string) (bool, error) {
	args := m.Called(namespace, containerName)
	return args.Bool(0), args.Error(1)
}

func (m *MockNexaaAPI) ListContainerByName(namespace string, containerName string) (api.ContainerResult, error) {
	args := m.Called(namespace, containerName)
	return args.Get(0).(api.ContainerResult), args.Error(1)
}

func (m *MockNexaaAPI) ContainerJobCreate(input api.ContainerJobCreateInput) (api.ContainerJobResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.ContainerJobResult), args.Error(1)
}

func (m *MockNexaaAPI) ContainerJobModify(input api.ContainerJobModifyInput) (api.ContainerJobResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.ContainerJobResult), args.Error(1)
}

func (m *MockNexaaAPI) ContainerJobDelete(namespace string, containerJobName string) (bool, error) {
	args := m.Called(namespace, containerJobName)
	return args.Bool(0), args.Error(1)
}

func (m *MockNexaaAPI) ContainerJobByName(namespace string, name string) (api.ContainerJobResult, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(api.ContainerJobResult), args.Error(1)
}

func (m *MockNexaaAPI) RegistryCreate(input api.RegistryCreateInput) (api.RegistryResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.RegistryResult), args.Error(1)
}

func (m *MockNexaaAPI) RegistryDelete(namespace string, registryName string) (bool, error) {
	args := m.Called(namespace, registryName)
	return args.Bool(0), args.Error(1)
}

func (m *MockNexaaAPI) ListRegistryByName(namespace string, registryName string) (*api.RegistryResult, error) {
	args := m.Called(namespace, registryName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.RegistryResult), args.Error(1)
}

func (m *MockNexaaAPI) VolumeCreate(input api.VolumeCreateInput) (api.VolumeResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.VolumeResult), args.Error(1)
}

func (m *MockNexaaAPI) VolumeIncrease(input api.VolumeModifyInput) (api.VolumeResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.VolumeResult), args.Error(1)
}

func (m *MockNexaaAPI) VolumeDelete(namespace string, volumeName string) (bool, error) {
	args := m.Called(namespace, volumeName)
	return args.Bool(0), args.Error(1)
}

func (m *MockNexaaAPI) ListVolumeByName(namespace string, volumeName string) (*api.VolumeResult, error) {
	args := m.Called(namespace, volumeName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.VolumeResult), args.Error(1)
}

func (m *MockNexaaAPI) MessageQueueCreate(input api.MessageQueueCreateInput) (api.MessageQueueResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.MessageQueueResult), args.Error(1)
}

func (m *MockNexaaAPI) MessageQueueModify(input api.MessageQueueModifyInput) (api.MessageQueueResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.MessageQueueResult), args.Error(1)
}

func (m *MockNexaaAPI) MessageQueueDelete(input api.MessageQueueResourceInput) (bool, error) {
	args := m.Called(input)
	return args.Bool(0), args.Error(1)
}

func (m *MockNexaaAPI) MessageQueueGet(input api.MessageQueueResourceInput) (api.MessageQueueResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.MessageQueueResult), args.Error(1)
}

func (m *MockNexaaAPI) MessageQueueList() ([]api.MessageQueueResult, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]api.MessageQueueResult), args.Error(1)
}

func (m *MockNexaaAPI) MessageQueuePlans() ([]api.MessageQueuePlanResult, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]api.MessageQueuePlanResult), args.Error(1)
}

func (m *MockNexaaAPI) CloudDatabaseClusterCreate(input api.CloudDatabaseClusterCreateInput) (api.CloudDatabaseClusterResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.CloudDatabaseClusterResult), args.Error(1)
}

func (m *MockNexaaAPI) CloudDatabaseClusterModify(input api.CloudDatabaseClusterModifyInput) (api.CloudDatabaseClusterResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.CloudDatabaseClusterResult), args.Error(1)
}

func (m *MockNexaaAPI) CloudDatabaseClusterDelete(input api.CloudDatabaseClusterResourceInput) (bool, error) {
	args := m.Called(input)
	return args.Bool(0), args.Error(1)
}

func (m *MockNexaaAPI) CloudDatabaseClusterGet(input api.CloudDatabaseClusterResourceInput) (api.CloudDatabaseClusterResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.CloudDatabaseClusterResult), args.Error(1)
}

func (m *MockNexaaAPI) CloudDatabaseClusterListPlans() ([]api.CloudDatabaseClusterPlan, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]api.CloudDatabaseClusterPlan), args.Error(1)
}

func (m *MockNexaaAPI) CloudDatabaseClusterDatabaseCreate(input api.CloudDatabaseClusterDatabaseCreateInput) (api.CloudDatabaseClusterDatabaseResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.CloudDatabaseClusterDatabaseResult), args.Error(1)
}

func (m *MockNexaaAPI) CloudDatabaseClusterDatabaseDelete(input api.CloudDatabaseClusterDatabaseResourceInput) (bool, error) {
	args := m.Called(input)
	return args.Bool(0), args.Error(1)
}

func (m *MockNexaaAPI) CloudDatabaseClusterUserCreate(input api.CloudDatabaseClusterUserCreateInput) (api.CloudDatabaseClusterUserResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.CloudDatabaseClusterUserResult), args.Error(1)
}

func (m *MockNexaaAPI) CloudDatabaseClusterUserModify(input api.CloudDatabaseClusterUserModifyInput) (api.CloudDatabaseClusterUserResult, error) {
	args := m.Called(input)
	return args.Get(0).(api.CloudDatabaseClusterUserResult), args.Error(1)
}

func (m *MockNexaaAPI) CloudDatabaseClusterUserGet(input api.CloudDatabaseClusterResourceInput, name string) (api.CloudDatabaseClusterUserResult, error) {
	args := m.Called(input, name)
	return args.Get(0).(api.CloudDatabaseClusterUserResult), args.Error(1)
}

func (m *MockNexaaAPI) CloudDatabaseClusterUserList(input api.CloudDatabaseClusterResourceInput) ([]api.CloudDatabaseClusterUserResult, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]api.CloudDatabaseClusterUserResult), args.Error(1)
}
