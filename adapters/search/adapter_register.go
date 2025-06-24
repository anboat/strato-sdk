package search

import (
	"fmt"

	"github.com/anboat/strato-sdk/config"
	"github.com/anboat/strato-sdk/config/types"
)

// AdapterCreatorFunc defines the function signature for an adapter creator.
type AdapterCreatorFunc func(engineConfig types.EngineConfig) (SearchAdapter, error)

// adapterCreators is a global registry for adapter creator functions.
var adapterCreators = make(map[string]AdapterCreatorFunc)

// RegisterAdapterCreator registers an adapter creator function for a given engine name.
func RegisterAdapterCreator(engineName string, creator AdapterCreatorFunc) {
	adapterCreators[engineName] = creator
}

// RegisterAllSearchAdapters registers all search adapters based on the application configuration.
func RegisterAllSearchAdapters() error {
	searchConfig := config.GetSearchConfig()
	if searchConfig == nil {
		return fmt.Errorf("search configuration not initialized")
	}

	// Iterate through the engines in the config and dynamically register enabled adapters.
	for engineName, engineConfig := range searchConfig.Engines {
		if engineConfig.Enabled {
			// Use a local variable to avoid closure capture issues.
			name := engineName
			// Register the adapter using a generic factory function.
			RegisterSearchAdapter(SearchEngine(name), func() (SearchAdapter, error) {
				return createAdapterFromConfig(name)
			})
		}
	}

	return nil
}

// createAdapterFromConfig is a generic adapter creation function.
// It creates a search adapter instance based on the configuration.
func createAdapterFromConfig(engineName string) (SearchAdapter, error) {
	searchConfig := config.GetSearchConfig()
	if searchConfig == nil {
		return nil, fmt.Errorf("search configuration not initialized")
	}

	engineConfig, exists := searchConfig.Engines[engineName]
	if !exists {
		return nil, fmt.Errorf("engine configuration not found for %s", engineName)
	}

	if !engineConfig.Enabled {
		return nil, fmt.Errorf("engine %s is disabled", engineName)
	}

	// Find the corresponding adapter creator.
	creator, exists := adapterCreators[engineName]
	if !exists {
		return nil, fmt.Errorf("adapter creator not registered for %s", engineName)
	}

	// Use the creator to create the adapter instance.
	return creator(engineConfig)
}
