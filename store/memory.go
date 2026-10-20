package store

import (
	"context"
	"sync"
)

type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]string)}
}

func (m *MemoryStore) Save(_ context.Context, code, originalURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[code] = originalURL
	return nil
}

func (m *MemoryStore) Retrieve(_ context.Context, code string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.data[code]
	if !ok {
		return "", ErrNotFound
	}
	return u, nil
}
