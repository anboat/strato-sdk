package types

// WebConfig holds the configuration for the Web module.
// It includes all settings related to web scraping.
type WebConfig struct {
	// Strategy configuration.
	Strategy WebStrategyConfig `json:"strategy" yaml:"strategy" mapstructure:"strategy"`

	// Scrapers is a map of web scraper configurations.
	Scrapers map[string]WebScraperConfig `json:"scrapers" yaml:"scrapers" mapstructure:"scrapers"`
}

// WebStrategyConfig holds the configuration for the web scraping strategy.
type WebStrategyConfig struct {
	// DefaultScraper is the default engine to use.
	DefaultScraper string `json:"default_scraper" yaml:"default_scraper" mapstructure:"default_scraper"`

	// DefaultFallbackOrder is the default fallback order.
	DefaultFallbackOrder []string `json:"default_fallback_order" yaml:"default_fallback_order" mapstructure:"default_fallback_order"`

	// EnableFallback determines whether to enable fallback.
	EnableFallback bool `json:"enable_fallback" yaml:"enable_fallback" mapstructure:"enable_fallback"`

	// FailFast determines whether to fall back immediately if a single engine fails.
	FailFast bool `json:"fail_fast" yaml:"fail_fast" mapstructure:"fail_fast"`
}

// WebScraperConfig holds the configuration for a single web scraper (simplified).
type WebScraperConfig struct {
	// Enabled indicates whether this scraper is enabled.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// API configuration.
	APIKey  string `json:"api_key" yaml:"api_key" mapstructure:"api_key"`
	BaseURL string `json:"base_url" yaml:"base_url" mapstructure:"base_url"`

	// Scraper-specific parameters.
	Config map[string]interface{} `json:"config" yaml:"config" mapstructure:"config"`
}
