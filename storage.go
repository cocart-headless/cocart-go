package cocart

import "sync"

// Storage defines the interface for persisting cart keys and JWT tokens.
type Storage interface {
	// Get retrieves a value by key. Returns ErrKeyNotFound if the key does not exist.
	Get(key string) (string, error)
	// Set stores a key-value pair.
	Set(key string, value string) error
	// Delete removes a key.
	Delete(key string) error
}

// MemoryStorage is an in-memory Storage implementation.
// It is safe for concurrent use.
type MemoryStorage struct {
	mu    sync.RWMutex
	store map[string]string
}

// NewMemoryStorage creates a new MemoryStorage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		store: make(map[string]string),
	}
}

// Get retrieves a value by key.
func (m *MemoryStorage) Get(key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.store[key]
	if !ok {
		return "", ErrKeyNotFound
	}
	return v, nil
}

// Set stores a key-value pair.
func (m *MemoryStorage) Set(key string, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = value
	return nil
}

// Delete removes a key.
func (m *MemoryStorage) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, key)
	return nil
}
