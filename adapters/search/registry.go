package search

import (
	"fmt"
	"sync"
)

// AdapterFactory defines the function signature for a search adapter factory.
type AdapterFactory func() (SearchAdapter, error)

// SearchRegistry is a registry for search adapters.
type SearchRegistry struct {
	mu        sync.RWMutex
	factories map[SearchEngine]AdapterFactory
}

// NewSearchRegistry creates a new search adapter registry.
func NewSearchRegistry() *SearchRegistry {
	return &SearchRegistry{
		factories: make(map[SearchEngine]AdapterFactory),
	}
}

// Register registers a search adapter factory.
func (r *SearchRegistry) Register(engine SearchEngine, factory AdapterFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[engine] = factory
}

// Create creates a search adapter instance.
func (r *SearchRegistry) Create(engine SearchEngine) (SearchAdapter, error) {
	r.mu.RLock()
	factory, exists := r.factories[engine]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unregistered search engine: %s", engine)
	}

	return factory()
}

// globalRegistry is the global instance of the search adapter registry.
var globalRegistry = NewSearchRegistry()

// RegisterSearchAdapter registers a search adapter globally.
func RegisterSearchAdapter(engine SearchEngine, factory AdapterFactory) {
	globalRegistry.Register(engine, factory)
}

// CreateSearchAdapter creates a search adapter instance from the global registry.
func CreateSearchAdapter(engine SearchEngine) (SearchAdapter, error) {
	return globalRegistry.Create(engine)
}
