package twitter

import (
	"fmt"
	"time"

	"github.com/anboat/strato-sdk/adapters/search"
	"github.com/anboat/strato-sdk/config/types"
)

// TwitterAdapterCreator creates Twitter adapter instances.
type TwitterAdapterCreator struct{}

// CreateAdapter creates a new Twitter adapter from the engine configuration.
func (c *TwitterAdapterCreator) CreateAdapter(engineConfig types.EngineConfig) (search.SearchAdapter, error) {
	bearerToken := engineConfig.SecretKey
	if bearerToken == "" {
		// The BearerToken is often stored in the 'secret_key' field for consistency.
		return nil, fmt.Errorf("twitter bearer_token (secret_key) is missing from credentials")
	}

	config := &ClientConfig{
		BearerToken: bearerToken,
	}

	if engineConfig.Config != nil {
		if baseURL, ok := engineConfig.Config["base_url"].(string); ok {
			config.BaseURL = baseURL
		}
		if timeout, ok := engineConfig.Config["timeout"]; ok {
			switch t := timeout.(type) {
			case int:
				config.Timeout = time.Duration(t) * time.Second
			case float64:
				config.Timeout = time.Duration(t) * time.Second
			}
		}
	}

	return NewClient(config), nil
}

// init registers the Twitter adapter creator with the global registry.
func init() {
	search.RegisterAdapterCreator("twitter", (&TwitterAdapterCreator{}).CreateAdapter)
}
