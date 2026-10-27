package views

import (
	"fmt"
	"sync"
)

// Registry manages all registered views
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ViewFactory
	order     []string // Maintains registration order
}

var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// GetRegistry returns the global view registry
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = &Registry{
			factories: make(map[string]ViewFactory),
			order:     make([]string, 0),
		}
	})
	return globalRegistry
}

// Register registers a view factory
func Register(factory ViewFactory) {
	registry := GetRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	// Create a temporary view instance to get metadata
	view := factory()
	metadata := view.Metadata()

	// Register the factory
	if _, exists := registry.factories[metadata.ID]; !exists {
		registry.order = append(registry.order, metadata.ID)
	}
	registry.factories[metadata.ID] = factory
}

// Get creates a new view instance by ID
func (r *Registry) Get(id string) (View, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[id]
	if !exists {
		return nil, fmt.Errorf("view not found: %s", id)
	}

	return factory(), nil
}

// GetAll returns metadata for all registered views in registration order
func (r *Registry) GetAll() []Metadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadatas := make([]Metadata, 0, len(r.order))
	for _, id := range r.order {
		factory := r.factories[id]
		view := factory()
		metadatas = append(metadatas, view.Metadata())
	}

	return metadatas
}

// Count returns the number of registered views
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.factories)
}
