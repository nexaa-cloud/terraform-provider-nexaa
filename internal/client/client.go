// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"sync"

	"github.com/nexaa-cloud/nexaa-cli/api"
)

// NexaaAPI is the interface covering all Nexaa API methods used by resources
// and data sources. It is implemented by *api.Client and can be replaced with
// a mock in unit tests.
type NexaaAPI interface {
	// Namespace
	NamespaceCreate(input api.NamespaceCreateInput) (api.NamespaceResult, error)
	NamespaceListByName(name string) (api.NamespaceResult, error)
	NamespaceDelete(name string) (bool, error)

	// Container
	ContainerCreate(input api.ContainerCreateInput) (api.ContainerResult, error)
	ContainerModify(input api.ContainerModifyInput) (api.ContainerResult, error)
	ContainerDelete(namespace string, containerName string) (bool, error)
	ListContainerByName(namespace string, containerName string) (api.ContainerResult, error)

	// Container Job
	ContainerJobCreate(input api.ContainerJobCreateInput) (api.ContainerJobResult, error)
	ContainerJobModify(input api.ContainerJobModifyInput) (api.ContainerJobResult, error)
	ContainerJobDelete(namespace string, containerJobName string) (bool, error)
	ContainerJobByName(namespace string, name string) (api.ContainerJobResult, error)

	// Registry
	RegistryCreate(input api.RegistryCreateInput) (api.RegistryResult, error)
	RegistryDelete(namespace string, registryName string) (bool, error)
	ListRegistryByName(namespace string, registryName string) (*api.RegistryResult, error)

	// Volume
	VolumeCreate(input api.VolumeCreateInput) (api.VolumeResult, error)
	VolumeIncrease(input api.VolumeModifyInput) (api.VolumeResult, error)
	VolumeDelete(namespace string, volumeName string) (bool, error)
	ListVolumeByName(namespace string, volumeName string) (*api.VolumeResult, error)

	// Message Queue
	MessageQueueCreate(input api.MessageQueueCreateInput) (api.MessageQueueResult, error)
	MessageQueueModify(input api.MessageQueueModifyInput) (api.MessageQueueResult, error)
	MessageQueueDelete(input api.MessageQueueResourceInput) (bool, error)
	MessageQueueGet(input api.MessageQueueResourceInput) (api.MessageQueueResult, error)
	MessageQueueList() ([]api.MessageQueueResult, error)
	MessageQueuePlans() ([]api.MessageQueuePlanResult, error)

	// Cloud Database Cluster
	CloudDatabaseClusterCreate(input api.CloudDatabaseClusterCreateInput) (api.CloudDatabaseClusterResult, error)
	CloudDatabaseClusterModify(input api.CloudDatabaseClusterModifyInput) (api.CloudDatabaseClusterResult, error)
	CloudDatabaseClusterDelete(input api.CloudDatabaseClusterResourceInput) (bool, error)
	CloudDatabaseClusterGet(input api.CloudDatabaseClusterResourceInput) (api.CloudDatabaseClusterResult, error)
	CloudDatabaseClusterListPlans() ([]api.CloudDatabaseClusterPlan, error)

	// Cloud Database Cluster Database
	CloudDatabaseClusterDatabaseCreate(input api.CloudDatabaseClusterDatabaseCreateInput) (api.CloudDatabaseClusterDatabaseResult, error)
	CloudDatabaseClusterDatabaseDelete(input api.CloudDatabaseClusterDatabaseResourceInput) (bool, error)

	// Cloud Database Cluster User
	CloudDatabaseClusterUserCreate(input api.CloudDatabaseClusterUserCreateInput) (api.CloudDatabaseClusterUserResult, error)
	CloudDatabaseClusterUserModify(input api.CloudDatabaseClusterUserModifyInput) (api.CloudDatabaseClusterUserResult, error)
	CloudDatabaseClusterUserGet(input api.CloudDatabaseClusterResourceInput, name string) (api.CloudDatabaseClusterUserResult, error)
	CloudDatabaseClusterUserList(input api.CloudDatabaseClusterResourceInput) ([]api.CloudDatabaseClusterUserResult, error)
}

// mutexKV is a key-value store of mutexes. It serializes concurrent operations
// that share the same key (e.g., two Create calls for a resource with the same name).
type mutexKV struct {
	mu    sync.Mutex
	store map[string]*sync.Mutex
}

func newMutexKV() *mutexKV {
	return &mutexKV{store: make(map[string]*sync.Mutex)}
}

func (m *mutexKV) lock(key string) {
	m.get(key).Lock()
}

func (m *mutexKV) unlock(key string) {
	m.get(key).Unlock()
}

func (m *mutexKV) get(key string) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[key]; !ok {
		m.store[key] = &sync.Mutex{}
	}
	return m.store[key]
}

// NexaaClient wraps the Nexaa API client and a shared mutex store.
// Resources receive a single NexaaClient instance via Configure so that
// concurrent Create calls for the same resource name are serialized,
// preventing backend unique-constraint violations.
type NexaaClient struct {
	API NexaaAPI
	mu  *mutexKV
}

func New(apiClient *api.Client) *NexaaClient {
	return &NexaaClient{
		API: apiClient,
		mu:  newMutexKV(),
	}
}

// NewWithAPI creates a NexaaClient with an arbitrary NexaaAPI implementation.
// Use this in unit tests to inject a MockNexaaAPI.
func NewWithAPI(apiClient NexaaAPI) *NexaaClient {
	return &NexaaClient{
		API: apiClient,
		mu:  newMutexKV(),
	}
}

func (c *NexaaClient) Lock(key string) {
	c.mu.lock(key)
}

func (c *NexaaClient) Unlock(key string) {
	c.mu.unlock(key)
}
