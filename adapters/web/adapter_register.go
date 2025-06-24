package web

import (
	"fmt"

	"github.com/anboat/strato-sdk/config"
	"github.com/anboat/strato-sdk/config/types"
)

// AdapterCreatorFunc defines the function signature for an adapter creator.
type AdapterCreatorFunc func(scraperConfig types.WebScraperConfig) (WebAdapter, error)

// adapterCreators is a global registry for adapter creator functions.
var adapterCreators = make(map[string]AdapterCreatorFunc)

// RegisterAdapterCreator registers an adapter creator function for a given scraper name.
func RegisterAdapterCreator(scraperName string, creator AdapterCreatorFunc) {
	adapterCreators[scraperName] = creator
}

// RegisterAllWebAdapters registers all web adapters based on the application configuration.
func RegisterAllWebAdapters() error {
	webConfig := config.GetWebConfig()
	if webConfig == nil {
		return fmt.Errorf("web adapter configuration not initialized")
	}

	// Iterate through the scrapers in the config and dynamically register enabled adapters.
	for scraperName, scraperConfig := range webConfig.Scrapers {
		if scraperConfig.Enabled {
			// Use a local variable to avoid closure capture issues.
			name := scraperName
			// Register the adapter using a generic factory function.
			RegisterWebAdapter(WebScraper(name), func() (WebAdapter, error) {
				return createAdapterFromConfig(name)
			})
		}
	}

	return nil
}

// createAdapterFromConfig is a generic adapter creation function.
// It creates a web adapter instance based on the configuration.
func createAdapterFromConfig(scraperName string) (WebAdapter, error) {
	webConfig := config.GetWebConfig()
	if webConfig == nil {
		return nil, fmt.Errorf("web adapter configuration not initialized")
	}

	scraperConfig, exists := webConfig.Scrapers[scraperName]
	if !exists {
		return nil, fmt.Errorf("scraper configuration not found for %s", scraperName)
	}

	if !scraperConfig.Enabled {
		return nil, fmt.Errorf("scraper %s is disabled", scraperName)
	}

	// Find the corresponding adapter creator.
	creator, exists := adapterCreators[scraperName]
	if !exists {
		return nil, fmt.Errorf("adapter creator not registered for %s", scraperName)
	}

	// Use the creator to create the adapter instance.
	return creator(scraperConfig)
}
