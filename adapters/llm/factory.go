package llm

import (
	"context"
	"fmt"
	"github.com/anboat/strato-sdk/config"
	"github.com/anboat/strato-sdk/config/types"
	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	openaiLib "github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/ollama/ollama/api"
	"net/http"
	"sync"
	"time"
)

// ModelOption defines an option function for configuring a model.
type ModelOption func(*ModelOptions)

// ModelOptions stores optional parameters for model creation.
type ModelOptions struct {
	JSONFormat bool // Whether to return JSON format.
	HTTPClient *http.Client
}

// WithHTTPClient sets a custom HTTP client for the model.
func WithHTTPClient(client *http.Client) ModelOption {
	return func(opts *ModelOptions) {
		opts.HTTPClient = client
	}
}

// WithJSONFormat sets the model to return responses in JSON format.
func WithJSONFormat() ModelOption {
	return func(opts *ModelOptions) {
		opts.JSONFormat = true
	}
}

// applyOptions applies the given options and returns a ModelOptions struct.
func applyOptions(options ...ModelOption) *ModelOptions {
	opts := &ModelOptions{} // Default values
	for _, opt := range options {
		opt(opts)
	}
	return opts
}

var (
	// factories is a global registry for chat model factories.
	factories   = make(map[string]ChatModelFactory)
	factoriesMu sync.RWMutex

	// instances is a cache for created model instances.
	instances   = make(map[string]model.ToolCallingChatModel)
	instancesMu sync.RWMutex
)

// ChatModelFactory defines the function signature for a model factory.
type ChatModelFactory func(ctx context.Context, modelConfig *types.ModelConfig, options ...ModelOption) (model.ToolCallingChatModel, error)

// RegisterChatModelFactory registers a ChatModel factory for a given model type.
func RegisterChatModelFactory(modelType string, factory ChatModelFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[modelType] = factory
}

// CreateOpenAI creates an OpenAI ChatModel.
func CreateOpenAI(ctx context.Context, modelConfig *types.ModelConfig, options ...ModelOption) (model.ToolCallingChatModel, error) {
	openaiConfig := &openai.ChatModelConfig{
		APIKey:  modelConfig.APIKey,
		BaseURL: modelConfig.BaseURL,
		Model:   modelConfig.Model,
		Timeout: GetModelTimeout(modelConfig),
	}

	// Handle optional parameters
	if modelConfig.Temperature > 0 {
		temp := float32(modelConfig.Temperature)
		openaiConfig.Temperature = &temp
	}
	if modelConfig.MaxTokens > 0 {
		maxTokens := modelConfig.MaxTokens
		openaiConfig.MaxTokens = &maxTokens
	}
	if modelConfig.TopP > 0 {
		topP := modelConfig.TopP
		openaiConfig.TopP = &topP
	}
	if modelConfig.FrequencyPenalty != 0 {
		fp := modelConfig.FrequencyPenalty
		openaiConfig.FrequencyPenalty = &fp
	}
	if modelConfig.PresencePenalty != 0 {
		pp := modelConfig.PresencePenalty
		openaiConfig.PresencePenalty = &pp
	}

	modelOptions := applyOptions(options...)
	if modelOptions.JSONFormat {
		openaiConfig.ResponseFormat = &openaiLib.ChatCompletionResponseFormat{
			Type: openaiLib.ChatCompletionResponseFormatTypeJSONObject,
		}
	}

	// Support for Azure OpenAI
	if azure, ok := modelConfig.Config["azure"].(map[string]interface{}); ok {
		openaiConfig.ByAzure = true
		if v, ok := azure["api_version"].(string); ok {
			openaiConfig.APIVersion = v
		}
	}

	chatModel, err := openai.NewChatModel(ctx, openaiConfig)
	return chatModel, err
}

// CreateDeepSeek creates a DeepSeek ChatModel.
func CreateDeepSeek(ctx context.Context, modelConfig *types.ModelConfig, options ...ModelOption) (model.ToolCallingChatModel, error) {

	dsConfig := &deepseek.ChatModelConfig{
		APIKey:           modelConfig.APIKey,
		BaseURL:          modelConfig.BaseURL,
		Model:            modelConfig.Model,
		Temperature:      modelConfig.Temperature,
		MaxTokens:        modelConfig.MaxTokens,
		TopP:             modelConfig.TopP,
		FrequencyPenalty: modelConfig.FrequencyPenalty,
		PresencePenalty:  modelConfig.PresencePenalty,
		Timeout:          GetModelTimeout(modelConfig),
	}

	modelOptions := applyOptions(options...)

	if modelOptions.JSONFormat {
		dsConfig.ResponseFormatType = deepseek.ResponseFormatTypeJSONObject
	}

	chatModel, err := deepseek.NewChatModel(ctx, dsConfig)

	return chatModel, err
}

// CreateQwen creates a Qwen ChatModel.
func CreateQwen(ctx context.Context, modelConfig *types.ModelConfig, options ...ModelOption) (model.ToolCallingChatModel, error) {
	qwenConfig := &qwen.ChatModelConfig{
		APIKey:  modelConfig.APIKey,
		BaseURL: modelConfig.BaseURL,
		Model:   modelConfig.Model,
		Timeout: GetModelTimeout(modelConfig),
	}

	// Handle optional parameters
	if modelConfig.Temperature > 0 {
		temp := float32(modelConfig.Temperature)
		qwenConfig.Temperature = &temp
	}
	if modelConfig.MaxTokens > 0 {
		maxTokens := modelConfig.MaxTokens
		qwenConfig.MaxTokens = &maxTokens
	}
	if modelConfig.TopP > 0 {
		topP := modelConfig.TopP
		qwenConfig.TopP = &topP
	}
	if modelConfig.FrequencyPenalty != 0 {
		fp := modelConfig.FrequencyPenalty
		qwenConfig.FrequencyPenalty = &fp
	}
	if modelConfig.PresencePenalty != 0 {
		pp := modelConfig.PresencePenalty
		qwenConfig.PresencePenalty = &pp
	}

	modelOptions := applyOptions(options...)
	if modelOptions.JSONFormat {
		qwenConfig.ResponseFormat = &openaiLib.ChatCompletionResponseFormat{
			Type: openaiLib.ChatCompletionResponseFormatTypeJSONObject,
		}
	}
	chatModel, err := qwen.NewChatModel(ctx, qwenConfig)
	return chatModel, err
}

// CreateClaude creates a Claude ChatModel.
func CreateClaude(ctx context.Context, modelConfig *types.ModelConfig, options ...ModelOption) (model.ToolCallingChatModel, error) {
	claudeConfig := &claude.Config{
		APIKey:  modelConfig.APIKey,
		BaseURL: &modelConfig.BaseURL,
		Model:   modelConfig.Model,
	}

	// Handle optional parameters
	if modelConfig.Temperature > 0 {
		temp := float32(modelConfig.Temperature)
		claudeConfig.Temperature = &temp
	}
	if modelConfig.TopP > 0 {
		topP := float32(modelConfig.TopP)
		claudeConfig.TopP = &topP
	}
	if modelConfig.MaxTokens > 0 {
		claudeConfig.MaxTokens = modelConfig.MaxTokens
	}
	if modelConfig.Config != nil {
		if v, ok := modelConfig.Config["top_k"].(int32); ok {
			claudeConfig.TopK = &v
		}
		if v, ok := modelConfig.Config["stop_sequences"].([]string); ok {
			claudeConfig.StopSequences = v
		}
	}
	chatModel, err := claude.NewChatModel(ctx, claudeConfig)
	return chatModel, err
}

// CreateOllama creates an Ollama ChatModel.
func CreateOllama(ctx context.Context, modelConfig *types.ModelConfig, options ...ModelOption) (model.ToolCallingChatModel, error) {
	ollamaConfig := &ollama.ChatModelConfig{
		BaseURL: modelConfig.BaseURL,
		Model:   modelConfig.Model,
		Timeout: GetModelTimeout(modelConfig),
	}

	modelOptions := applyOptions(options...)
	// Handle optional parameters
	if modelConfig.Config != nil {
		if v, ok := modelConfig.Config["options"].(*api.Options); ok {
			ollamaConfig.Options = v
		}
		if v, ok := modelConfig.Config["format"].([]byte); ok {
			ollamaConfig.Format = v
		}
		if v, ok := modelConfig.Config["keep_alive"].(time.Duration); ok {
			ollamaConfig.KeepAlive = &v
		}
	}

	// Priority use HTTPClient from ModelOptions
	if modelOptions.HTTPClient != nil {
		ollamaConfig.HTTPClient = modelOptions.HTTPClient
	}

	chatModel, err := ollama.NewChatModel(ctx, ollamaConfig)
	if err != nil {
		return nil, err
	}
	return chatModel, nil
}

// CreateChatModel creates a chat model instance based on the model type and configuration.
// It uses a registered factory for the given model type.
func CreateChatModel(ctx context.Context, modelType string, config *types.ModelConfig, options ...ModelOption) (model.ToolCallingChatModel, error) {
	if config == nil {
		return nil, fmt.Errorf("model configuration cannot be nil")
	}

	// Check if the model is enabled
	if !config.Enabled {
		return nil, fmt.Errorf("model %s is not enabled", modelType)
	}

	// Check if the model type is registered
	factoriesMu.RLock()
	factory, exists := factories[config.Type]
	factoriesMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported model type: %s", config.Type)
	}

	// Create model instance
	chatModel, err := factory(ctx, config, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model '%s': %w", modelType, err)
	}

	return chatModel, nil
}

// GetChatModel retrieves a chat model instance by name.
// It first checks a cache of existing instances. If not found, it creates a new one
// using the configuration and caches it.
func GetChatModel(ctx context.Context, modelName string) (model.ToolCallingChatModel, error) {
	// First check cache
	instancesMu.RLock()
	if chatModel, exists := instances[modelName]; exists {
		instancesMu.RUnlock()
		return chatModel, nil
	}
	instancesMu.RUnlock()

	// Lock to create new instance
	instancesMu.Lock()
	defer instancesMu.Unlock()

	// Double check
	if chatModel, exists := instances[modelName]; exists {
		return chatModel, nil
	}

	// Get configuration
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		return nil, fmt.Errorf("model configuration not found")
	}

	modelConfig, exists := modelsConfig.Models[modelName]
	if !exists {
		return nil, fmt.Errorf("model configuration not found for: %s", modelName)
	}

	// Create model
	chatModel, err := CreateChatModel(ctx, modelConfig.Type, &modelConfig)
	if err != nil {
		return nil, err
	}

	// Cache model
	instances[modelName] = chatModel
	return chatModel, nil
}

// GetDefaultChatModel retrieves the default chat model instance.
// It uses the default model name from the application configuration.
func GetDefaultChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		return nil, fmt.Errorf("model configuration not found")
	}

	if modelsConfig.DefaultModel == "" {
		return nil, fmt.Errorf("no default model configured")
	}

	return GetChatModel(ctx, modelsConfig.DefaultModel)
}

// GetModelTimeout returns the timeout duration for a model.
// It defaults to 300 seconds if not specified in the configuration.
func GetModelTimeout(config *types.ModelConfig) time.Duration {
	if config.TimeoutSeconds > 0 {
		return time.Duration(config.TimeoutSeconds) * time.Second
	}
	return 300 * time.Second // Default timeout
}

// init automatically registers all supported ChatModel factories
func init() {
	// Register all model factories
	RegisterChatModelFactory("openai", CreateOpenAI)
	RegisterChatModelFactory("deepseek", CreateDeepSeek)
	RegisterChatModelFactory("qwen", CreateQwen)
	RegisterChatModelFactory("claude", CreateClaude)
	RegisterChatModelFactory("ollama", CreateOllama)
}
