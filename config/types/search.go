package types

// SearchConfig holds the configuration related to search functionalities.
type SearchConfig struct {
	// Search strategy configuration.
	Strategy SearchStrategyConfig `json:"strategy" yaml:"strategy" mapstructure:"strategy"`

	// Specific configurations for each search engine.
	Engines map[string]EngineConfig `json:"engines" yaml:"engines" mapstructure:"engines"`
}

// SearchStrategyConfig holds the configuration for the search strategy.
type SearchStrategyConfig struct {
	// Search breadth - the number of results to take from each search engine.
	Breadth int `json:"breadth" yaml:"breadth" mapstructure:"breadth"`

	// List of search engines to use for mixed search.
	MixedEngines []string `json:"mixed_engines" yaml:"mixed_engines" mapstructure:"mixed_engines"`

	// Default engine to use if no engine list is specified.
	DefaultEngine string `json:"default_engine" yaml:"default_engine" mapstructure:"default_engine"`

	// Default fallback order if no engine list is specified.
	DefaultFallbackOrder []string `json:"default_fallback_order" yaml:"default_fallback_order" mapstructure:"default_fallback_order"`

	// Whether to enable fallback.
	EnableFallback bool `json:"enable_fallback" yaml:"enable_fallback" mapstructure:"enable_fallback"`

	// Whether to fall back immediately if a single engine fails.
	FailFast bool `json:"fail_fast" yaml:"fail_fast" mapstructure:"fail_fast"`

	// Maximum number of retries.
	MaxRetries int `json:"max_retries" yaml:"max_retries" mapstructure:"max_retries"`

	// Timeout in seconds.
	TimeoutSeconds int `json:"timeout_seconds" yaml:"timeout_seconds" mapstructure:"timeout_seconds"`
}

// EngineConfig holds the configuration for a single search engine.
type EngineConfig struct {
	// Whether this engine is enabled.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// API configuration.
	APIKey    string `json:"api_key" yaml:"api_key" mapstructure:"api_key"`
	SecretKey string `json:"secret_key" yaml:"secret_key" mapstructure:"secret_key"`
	BaseURL   string `json:"base_url" yaml:"base_url" mapstructure:"base_url"`

	// Engine-specific parameters.
	Config map[string]interface{} `json:"config" yaml:"config" mapstructure:"config"`

	// Rate limit configuration.
	RateLimit RateLimitConfig `json:"rate_limit" yaml:"rate_limit" mapstructure:"rate_limit"`
}

// RateLimitConfig holds the rate limiting configuration.
type RateLimitConfig struct {
	// Whether to enable rate limiting.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// Maximum requests per second.
	RequestsPerSecond int `json:"requests_per_second" yaml:"requests_per_second" mapstructure:"requests_per_second"`

	// Burst size for requests.
	BurstSize int `json:"burst_size" yaml:"burst_size" mapstructure:"burst_size"`
}
