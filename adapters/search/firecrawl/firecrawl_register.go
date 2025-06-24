package firecrawl

import (
	"fmt"
	"time"

	"github.com/anboat/strato-sdk/adapters/search"
	"github.com/anboat/strato-sdk/config/types"
)

// FirecrawlAdapterCreator creates Firecrawl search adapter instances.
type FirecrawlAdapterCreator struct{}

// CreateAdapter creates a new Firecrawl search adapter instance.
func (c *FirecrawlAdapterCreator) CreateAdapter(engineConfig types.EngineConfig) (search.SearchAdapter, error) {
	firecrawlConfig := &FirecrawlConfig{}

	firecrawlConfig.APIKey = engineConfig.APIKey
	if firecrawlConfig.APIKey == "" {
		return nil, fmt.Errorf("firecrawl engine is missing the required api_key parameter")
	}

	if engineConfig.BaseURL != "" {
		firecrawlConfig.BaseURL = engineConfig.BaseURL
	}

	if engineConfig.Config != nil {
		if userAgent, ok := engineConfig.Config["user_agent"].(string); ok {
			firecrawlConfig.UserAgent = userAgent
		}

		if timeout, ok := engineConfig.Config["timeout"]; ok {
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

	return NewFirecrawlAdapter(firecrawlConfig), nil
}

func init() {
	creator := &FirecrawlAdapterCreator{}
	search.RegisterAdapterCreator("firecrawl", creator.CreateAdapter)
}
