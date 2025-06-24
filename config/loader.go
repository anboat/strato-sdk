package config

import (
	"fmt"
	"github.com/anboat/strato-sdk/config/types"
	"github.com/spf13/viper"
	"strings"
)

// Global config instance
var config types.Config

// LoadConfig loads the configuration from a given path.
// It uses Viper to support multiple configuration formats and environment variables.
func LoadConfig(configPath string) *types.Config {
	// If a configPath is specified, use it directly.
	if configPath != "" {
		// Set Viper to use the specified configPath.
		viper.SetConfigFile(configPath) // Set the full path including filename and extension.
	} else {
		// If no path is specified, set default search paths and name.
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("./config/examples")
	}

	// Set environment variable prefix.
	viper.SetEnvPrefix("strato")
	// Replace "." with "_" in environment variable keys.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// Read environment variables.
	viper.AutomaticEnv()

	// Read the configuration file.
	if err := viper.ReadInConfig(); err != nil {
		if configPath != "" {
			// If a config file path was specified but reading failed, panic.
			panic(fmt.Errorf("failed to read config file: %w", err))
		}
		// If no path was specified and the file doesn't exist, use default settings.
		fmt.Printf("Config file not found, using default settings: %v\n", err)
	} else {
		fmt.Printf("Loaded config file: %s\n", viper.ConfigFileUsed())
	}

	// Unmarshal the configuration into the struct.
	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Errorf("failed to unmarshal config: %w", err))
	}

	return &config
}

// GetConfig returns the global configuration instance.
func GetConfig() *types.Config {
	return &config
}

// GetAppConfig returns the application configuration.
func GetAppConfig() *types.AppConfig {
	return &config.App
}

// GetWebConfig returns the web configuration.
func GetWebConfig() *types.WebConfig {
	return &config.Web
}

// GetSearchConfig returns the search configuration.
func GetSearchConfig() *types.SearchConfig {
	return &config.Search
}

// GetModelsConfig returns the models configuration.
func GetModelsConfig() *types.ModelsConfig {
	return &config.Models
}

// GetAgentConfig returns the agent configuration.
func GetAgentConfig() *types.AgentConfig {
	return &config.Agent
}

// GetResearchConfig returns the research agent configuration.
func GetResearchConfig() *types.ResearchConfig {
	return &config.Agent.Research
}

// UpdateConfig updates the global configuration with a new config object.
func UpdateConfig(newConfig *types.Config) {
	config = *newConfig
}

// IsDebugMode checks if the application is in debug mode.
func IsDebugMode() bool {
	return config.App.Debug
}

// IsProduction checks if the application is in production environment.
func IsProduction() bool {
	return config.App.Environment == "production"
}

// IsDevelopment checks if the application is in development environment.
func IsDevelopment() bool {
	return config.App.Environment == "development"
}

// IsTesting checks if the application is in testing environment.
func IsTesting() bool {
	return config.App.Environment == "testing"
}

// GetEnvironment returns the current running environment.
func GetEnvironment() string {
	return config.App.Environment
}
