package web

import (
	"fmt"
	"sync"
)

// AdapterFactory defines the function signature for a web adapter factory.
type AdapterFactory func() (WebAdapter, error)

// WebRegistry is a registry for web adapters.
type WebRegistry struct {
	mu        sync.RWMutex
	factories map[WebScraper]AdapterFactory
}

// NewWebRegistry creates a new web adapter registry.
func NewWebRegistry() *WebRegistry {
	return &WebRegistry{
		factories: make(map[WebScraper]AdapterFactory),
	}
}

// Register registers a web adapter factory.
func (r *WebRegistry) Register(scraper WebScraper, factory AdapterFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[scraper] = factory
}

// Create creates a web adapter instance.
func (r *WebRegistry) Create(scraper WebScraper) (WebAdapter, error) {
	r.mu.RLock()
	factory, exists := r.factories[scraper]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unregistered web scraper: %s", scraper)
	}

	return factory()
}

// globalRegistry is the global instance of the web adapter registry.
var globalRegistry = NewWebRegistry()

// RegisterWebAdapter registers a web adapter globally.
func RegisterWebAdapter(engine WebScraper, factory AdapterFactory) {
	globalRegistry.Register(engine, factory)
}

// CreateWebAdapter creates a web adapter instance from the global registry.
func CreateWebAdapter(engine WebScraper) (WebAdapter, error) {
	return globalRegistry.Create(engine)
}
