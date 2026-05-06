// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"sync"

	"github.com/nexaa-cloud/nexaa-cli/api"
)

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
	API *api.Client
	mu  *mutexKV
}

func New(apiClient *api.Client) *NexaaClient {
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
