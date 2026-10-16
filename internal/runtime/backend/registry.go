// Package backend defines the extension contract for external inference backends.
package backend

import (
	"fmt"
	"sync"
)

// Registry stores available backend implementations by ID.
type Registry struct {
	mu       sync.RWMutex
	backends map[string]Backend
}

// NewRegistry creates an empty backend registry.
func NewRegistry() *Registry {
	return &Registry{backends: map[string]Backend{}}
}

// Register adds or replaces a backend implementation.
func (r *Registry) Register(b Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends[b.ID()] = b
}

// Get returns a backend by ID.
func (r *Registry) Get(id string) (Backend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.backends[id]
	if !ok {
		return nil, fmt.Errorf("backend %q is not registered", id)
	}
	return b, nil
}

// List returns all registered backend IDs.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.backends))
	for id := range r.backends {
		ids = append(ids, id)
	}
	return ids
}

