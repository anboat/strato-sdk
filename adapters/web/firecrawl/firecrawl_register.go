package firecrawl

import (
	"github.com/anboat/strato-sdk/adapters/web"
	"github.com/anboat/strato-sdk/config/types"
	"time"
)

// FirecrawlAdapterCreator creates Firecrawl adapter instances.
type FirecrawlAdapterCreator struct{}

// CreateAdapter creates a new Firecrawl adapter.
func (c *FirecrawlAdapterCreator) CreateAdapter(scraperConfig types.WebScraperConfig) (web.WebAdapter, error) {
	// Create Firecrawl config object.
	firecrawlConfig := &ClientConfig{}

	// Read API key from credentials.
	firecrawlConfig.APIKey = scraperConfig.APIKey

	// Use the BaseURL field.
	if scraperConfig.BaseURL != "" {
		firecrawlConfig.BaseURL = scraperConfig.BaseURL
	} else {
		firecrawlConfig.BaseURL = FirecrawlAPIBaseURL
	}

	// Read other configurations if they exist.
	if scraperConfig.Config != nil {
		if timeout, ok := scraperConfig.Config["timeout"]; ok {
			switch t := timeout.(type) {
			case int:
				firecrawlConfig.Timeout = time.Duration(t) * time.Second
			case float64:
				firecrawlConfig.Timeout = time.Duration(t) * time.Second
			case string:
				if duration, err := time.ParseDuration(t); err == nil {
					firecrawlConfig.Timeout = duration
				}
			}
		}
	}

	// Set a default timeout if not provided.
	if firecrawlConfig.Timeout == 0 {
		firecrawlConfig.Timeout = DefaultTimeout
	}

	return NewClient(firecrawlConfig), nil
}

// init registers the Firecrawl adapter creator.
func init() {
	// The web adapter registry is in the parent `web` package.
	web.RegisterAdapterCreator("firecrawl", (&FirecrawlAdapterCreator{}).CreateAdapter)
}
