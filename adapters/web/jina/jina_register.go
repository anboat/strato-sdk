package jina

import (
	"time"

	"github.com/anboat/strato-sdk/adapters/web"
	"github.com/anboat/strato-sdk/config/types"
)

// JinaAdapterCreator creates Jina adapter instances.
type JinaAdapterCreator struct{}

// CreateAdapter creates a new Jina adapter instance.
func (c *JinaAdapterCreator) CreateAdapter(scraperConfig types.WebScraperConfig) (web.WebAdapter, error) {
	// Create a Jina client configuration object.
	clientConfig := &ClientConfig{}

	// Read the API key from the scraper configuration.
	clientConfig.APIKey = scraperConfig.APIKey

	// Use the BaseURL field.
	if scraperConfig.BaseURL != "" {
		clientConfig.BaseURL = scraperConfig.BaseURL
	} else {
		clientConfig.BaseURL = JinaReaderBaseURL
	}

	// Read other configurations from the 'config' section.
	if scraperConfig.Config != nil {
		// Handle various timeout formats.
		if timeout, ok := scraperConfig.Config["timeout"]; ok {
			switch t := timeout.(type) {
			case int:
				clientConfig.Timeout = time.Duration(t) * time.Second
			case float64:
				clientConfig.Timeout = time.Duration(t) * time.Second
			case string:
				if duration, err := time.ParseDuration(t); err == nil {
					clientConfig.Timeout = duration
				}
			}
		}
	}

	// Set a default timeout if not provided.
	if clientConfig.Timeout == 0 {
		clientConfig.Timeout = DefaultTimeout
	}

	// Create and return the Client, which already implements the WebAdapter interface.
	return NewClient(clientConfig), nil
}

// init registers the Jina adapter creator.
func init() {
	creator := &JinaAdapterCreator{}
	web.RegisterAdapterCreator("jina", creator.CreateAdapter)
}
