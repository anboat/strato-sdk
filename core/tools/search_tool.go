package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/anboat/strato-sdk/adapters/search"
	"github.com/anboat/strato-sdk/config"
)

// Global variables to avoid repeated initialization.
var (
	searchStrategy  search.SearchStrategy
	searchInitOnce  sync.Once
	searchInitError error
)

// initSearchAdapters initializes the search adapters and strategy.
// This function is executed only once.
func initSearchAdapters() {
	searchInitOnce.Do(func() {
		// 1. Register all search adapters.
		if err := search.RegisterAllSearchAdapters(); err != nil {
			searchInitError = fmt.Errorf("failed to initialize search adapters: %w", err)
			return
		}

		// 2. Create the search strategy from the configuration.
		strategy, err := search.NewDefaultSearchStrategyFromConfig()
		if err != nil {
			searchInitError = fmt.Errorf("failed to create search strategy: %w", err)
			return
		}

		searchStrategy = strategy
	})
}

// SearchRequest defines the parameters for a search request.
// It uses jsonschema tags to define parameter constraints for the Eino framework.
type SearchRequest struct {
	Query      string `json:"query" jsonschema:"required,description=The search query string (required)."`
	Num        int    `json:"num" jsonschema:"minimum=1,maximum=50,description=The number of results to return, default is 10."`
	Lang       string `json:"lang" jsonschema:"description=The search language, e.g., zh-CN, en-US."`
	Region     string `json:"region" jsonschema:"description=The search region, e.g., CN, US."`
	SafeSearch string `json:"safe_search" jsonschema:"enum=off,enum=moderate,enum=strict,description=The safe search level."`
	TimeRange  string `json:"time_range" jsonschema:"description=The time range for the search, e.g., past_day, past_week."`
}

// SearchResponse defines the structure of the search response.
type SearchResponse struct {
	Success    bool                       `json:"success"`
	Message    string                     `json:"message,omitempty"`
	Query      string                     `json:"query"`
	Engine     string                     `json:"engine,omitempty"`
	Results    []*search.SearchResultItem `json:"results,omitempty"`
	TotalCount int                        `json:"total_count"`
	TimeTaken  int64                      `json:"time_taken_ms"`
	Error      string                     `json:"error,omitempty"`
}

// searchFunc is the underlying implementation of the search tool.
func searchFunc(ctx context.Context, request *SearchRequest, opts ...tool.Option) (*SearchResponse, error) {
	startTime := time.Now()

	// Validate the request parameters.
	if err := validateSearchRequest(request); err != nil {
		return &SearchResponse{
			Success:   false,
			Query:     request.Query,
			TimeTaken: time.Since(startTime).Milliseconds(),
			Error:     fmt.Sprintf("parameter validation failed: %v", err),
		}, nil
	}

	// Ensure search adapters are initialized (executes only once).
	initSearchAdapters()
	if searchInitError != nil {
		return &SearchResponse{
			Success:   false,
			Query:     request.Query,
			TimeTaken: time.Since(startTime).Milliseconds(),
			Error:     searchInitError.Error(),
		}, nil
	}

	// Build the internal search request.
	searchRequest := buildSearchRequest(request)

	// Execute the search using the configured strategy.
	result, err := searchStrategy.Execute(ctx, searchRequest)
	if err != nil {
		return &SearchResponse{
			Success:   false,
			Query:     request.Query,
			TimeTaken: time.Since(startTime).Milliseconds(),
			Error:     fmt.Sprintf("search execution failed: %v", err),
		}, nil
	}

	// Build the final response.
	response := buildSearchResponse(result, request.Query, startTime)
	return response, nil
}

// validateSearchRequest validates the search request parameters.
func validateSearchRequest(request *SearchRequest) error {
	if request.Query == "" {
		return fmt.Errorf("search query cannot be empty")
	}

	// Set default values.
	if request.Num <= 0 || request.Num > 50 {
		request.Num = 10
	}

	return nil
}

// buildSearchRequest builds the internal search.SearchRequest from the tool's SearchRequest.
func buildSearchRequest(request *SearchRequest) *search.SearchRequest {
	searchRequest := &search.SearchRequest{
		Query:      request.Query,
		Num:        request.Num,
		Offset:     0, // Default to the first page.
		Lang:       request.Lang,
		Region:     request.Region,
		SafeSearch: request.SafeSearch,
		TimeRange:  request.TimeRange,
	}

	return searchRequest
}

// buildSearchResponse builds the final SearchResponse from the internal search.SearchResponse.
func buildSearchResponse(result *search.SearchResponse, query string, startTime time.Time) *SearchResponse {
	response := &SearchResponse{
		Success:    true,
		Query:      query,
		Results:    result.Results,
		TotalCount: result.TotalCount,
		TimeTaken:  time.Since(startTime).Milliseconds(),
	}

	// Extract the engine name from metadata.
	if len(result.Results) > 0 && result.Results[0].Metadata != nil {
		if engineName, ok := result.Results[0].Metadata["search_engine"].(string); ok {
			response.Engine = engineName
		}

		// Check if a fallback strategy was used.
		if strategy, ok := result.Results[0].Metadata["strategy"].(string); ok && strategy == "fallback" {
			if attempt, ok := result.Results[0].Metadata["attempt"].(int); ok && attempt > 1 {
				response.Message = fmt.Sprintf("Fallback strategy was used, succeeded on attempt %d", attempt)
			}
		}
	}

	return response
}

// NewSearchTool creates a new invokable search tool.
// It initializes all search adapters and strategies based on the application configuration.
func NewSearchTool() (tool.InvokableTool, error) {
	// Check for basic search configuration.
	searchConfig := config.GetSearchConfig()
	if searchConfig == nil {
		return nil, fmt.Errorf("search engine configuration not found")
	}

	// Pre-initialize search adapters.
	initSearchAdapters()
	if searchInitError != nil {
		return nil, searchInitError
	}

	// Create the InvokableTool using Eino's utility function.
	return utils.InferOptionableTool(
		"search",
		"An intelligent search tool that supports multiple search engines (e.g., web, academic) and features a multi-engine fallback strategy. SDK users can extend it with custom search adapters.",
		searchFunc,
	)
}
