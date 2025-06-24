package types

// LogConfig holds the configuration for logging.
type LogConfig struct {
	// Level Log level: debug/info/warn/error
	Level string `json:"level" yaml:"level" mapstructure:"level"`

	// FilePath Log file path
	FilePath string `json:"file_path" yaml:"file_path" mapstructure:"file_path"`

	// MaxSize Maximum size of a single log file in MB
	MaxSize int `json:"max_size" yaml:"max_size" mapstructure:"max_size"`

	// MaxBackups Number of old log files to keep
	MaxBackups int `json:"max_backups" yaml:"max_backups" mapstructure:"max_backups"`

	// MaxAge Days to retain old log files
	MaxAge int `json:"max_age" yaml:"max_age" mapstructure:"max_age"`

	// Compress Whether to compress old log files
	Compress bool `json:"compress" yaml:"compress" mapstructure:"compress"`

	// Env Environment: development/production
	Env string `json:"env" yaml:"env" mapstructure:"env"`

	// EnableConsole Whether to output logs to the console
	EnableConsole bool `json:"enable_console" yaml:"enable_console" mapstructure:"enable_console"`
}
