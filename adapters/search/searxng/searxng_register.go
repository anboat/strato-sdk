package searxng

import (
	"time"

	"github.com/anboat/strato-sdk/adapters/search"
	"github.com/anboat/strato-sdk/config/types"
)

// SearXNGAdapterCreator creates SearXNG adapter instances.
type SearXNGAdapterCreator struct{}

// CreateAdapter creates a new SearXNG adapter instance.
func (c *SearXNGAdapterCreator) CreateAdapter(engineConfig types.EngineConfig) (search.SearchAdapter, error) {
	// Create a SearXNG configuration object.
	searxngConfig := &SearXNGConfig{}

	// Use the BaseURL field instead of the Config map.
	if engineConfig.BaseURL != "" {
		searxngConfig.BaseURL = engineConfig.BaseURL
	} else {
		searxngConfig.BaseURL = "https://searx.be" // Default public instance.
	}

	// Read other configurations from the engine's 'config' section.
	if engineConfig.Config != nil {
		if userAgent, ok := engineConfig.Config["user_agent"].(string); ok {
			searxngConfig.UserAgent = userAgent
		} else {
			searxngConfig.UserAgent = "SearXNG-Go-Client/1.0"
		}

		if language, ok := engineConfig.Config["language"].(string); ok {
			searxngConfig.Language = language
		} else {
			searxngConfig.Language = "zh-CN"
		}

		if safeSearch, ok := engineConfig.Config["safe_search"].(string); ok {
			searxngConfig.SafeSearch = safeSearch
		} else {
			searxngConfig.SafeSearch = "1" // Moderate level.
		}

		if categories, ok := engineConfig.Config["categories"].(string); ok {
			searxngConfig.Categories = categories
		} else {
			searxngConfig.Categories = "general"
		}

		if engines, ok := engineConfig.Config["engines"].(string); ok {
			searxngConfig.Engines = engines
		}

		// Handle various timeout formats.
		if timeout, ok := engineConfig.Config["timeout"]; ok {
			switch t := timeout.(type) {
			case int:
				searxngConfig.Timeout = time.Duration(t) * time.Second
			case float64:
				searxngConfig.Timeout = time.Duration(t) * time.Second
			case string:
				if duration, err := time.ParseDuration(t); err == nil {
					searxngConfig.Timeout = duration
				}
			}
		}
	} else {
		// Provide default configurations.
		searxngConfig.UserAgent = "SearXNG-Go-Client/1.0"
		searxngConfig.Language = "zh-CN"
		searxngConfig.SafeSearch = "1"
		searxngConfig.Categories = "general"
	}

	// Set a default timeout if not provided.
	if searxngConfig.Timeout == 0 {
		searxngConfig.Timeout = 30 * time.Second
	}

	return NewSearXNGAdapter(searxngConfig), nil
}

// init registers the SearXNG adapter creator.
func init() {
	creator := &SearXNGAdapterCreator{}
	search.RegisterAdapterCreator("searxng", creator.CreateAdapter)
}
