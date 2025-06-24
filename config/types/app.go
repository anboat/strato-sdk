package types

// AppConfig holds the basic application configuration.
type AppConfig struct {
	Name        string `json:"name" yaml:"name" mapstructure:"name"`                      // Application name
	Version     string `json:"version" yaml:"version" mapstructure:"version"`             // Application version
	Environment string `json:"environment" yaml:"environment" mapstructure:"environment"` // Running environment: development/testing/staging/production
	Debug       bool   `json:"debug" yaml:"debug" mapstructure:"debug"`                   // Whether to enable debug mode
}
