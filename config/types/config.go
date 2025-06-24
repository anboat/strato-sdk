package types

// Config is the main configuration structure, containing settings for all modules.
type Config struct {

	// Log module configuration.
	Log LogConfig `json:"log" yaml:"log" mapstructure:"log"` // Logging configuration.

	// Web module configuration.
	Web WebConfig `json:"web" yaml:"web" mapstructure:"web"`

	// Basic application configuration.
	App AppConfig `json:"app" yaml:"app" mapstructure:"app"`

	// Search engine configuration.
	Search SearchConfig `json:"search" yaml:"search" mapstructure:"search"`

	// Models configuration.
	Models ModelsConfig `json:"models" yaml:"models" mapstructure:"models"`

	// Agent configuration.
	Agent AgentConfig `json:"agent" yaml:"agent" mapstructure:"agent"`
}
