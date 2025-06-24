package types

// AgentConfig holds the configuration for intelligent agents.
type AgentConfig struct {
	// Research process configuration.
	Research ResearchConfig `json:"research" yaml:"research" mapstructure:"research"`
}

// ResearchConfig holds the configuration for the research agent.
type ResearchConfig struct {
	// Maximum number of iterations.
	MaxIterations int `json:"max_iterations" yaml:"max_iterations" mapstructure:"max_iterations"`

	// Maximum number of steps for the Eino workflow graph.
	MaxSteps int `json:"max_steps" yaml:"max_steps" mapstructure:"max_steps"`

	// Minimum number of questions to ensure research depth.
	MinQuestions int `json:"min_questions" yaml:"min_questions" mapstructure:"min_questions"`

	// Maximum content length.
	MaxContentLength int `json:"max_content_length" yaml:"max_content_length" mapstructure:"max_content_length"`

	// Maximum length of a single piece of content.
	MaxSingleContent int `json:"max_single_content" yaml:"max_single_content" mapstructure:"max_single_content"`

	// Channel buffer size.
	ChannelBuffer int `json:"channel_buffer" yaml:"channel_buffer" mapstructure:"channel_buffer"`
}
