package types

// ModelsConfig holds the configuration related to models.
type ModelsConfig struct {
	// Default model to use.
	DefaultModel string `json:"default_model" yaml:"default_model" mapstructure:"default_model"`

	// Configuration for all available models.
	Models map[string]ModelConfig `json:"models" yaml:"models" mapstructure:"models"`
}

// ModelConfig holds the configuration for a single model.
type ModelConfig struct {
	// Model type (e.g., openai, claude, gemini, qwen, deepseek, ollama, ark, qianfan).
	Type string `json:"type" yaml:"type" mapstructure:"type"`

	// Whether this model is enabled.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// API configuration.
	APIKey  string `json:"api_key" yaml:"api_key" mapstructure:"api_key"`
	BaseURL string `json:"base_url" yaml:"base_url" mapstructure:"base_url"`

	// Model name.
	Model string `json:"model" yaml:"model" mapstructure:"model"`

	// Model parameters.
	Temperature      float32 `json:"temperature" yaml:"temperature" mapstructure:"temperature"`
	MaxTokens        int     `json:"max_tokens" yaml:"max_tokens" mapstructure:"max_tokens"`
	TopP             float32 `json:"top_p" yaml:"top_p" mapstructure:"top_p"`
	FrequencyPenalty float32 `json:"frequency_penalty" yaml:"frequency_penalty" mapstructure:"frequency_penalty"`
	PresencePenalty  float32 `json:"presence_penalty" yaml:"presence_penalty" mapstructure:"presence_penalty"`

	// Timeout configuration in seconds.
	TimeoutSeconds int `json:"timeout_seconds" yaml:"timeout_seconds" mapstructure:"timeout_seconds"`

	// Model-specific configuration.
	Config map[string]interface{} `json:"config" yaml:"config" mapstructure:"config"`
}
