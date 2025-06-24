package search

import (
	"context"
	"fmt"
	"github.com/anboat/strato-sdk/config"
	"sync"
)

// SearchStrategy defines the interface for a search strategy.
// It provides a method to execute a search based on a given request.
type SearchStrategy interface {
	// Execute performs the search according to the implemented strategy.
	Execute(ctx context.Context, request *SearchRequest) (*SearchResponse, error)
}

// SearchStrategyConfig holds the configuration for a search strategy.
type SearchStrategyConfig struct {
	// Breadth specifies the number of search results to retrieve from each search engine.
	Breadth int `json:"breadth"`

	// MixedEngines is a list of search engines to be used for a mixed search,
	// where results are aggregated from multiple engines concurrently.
	MixedEngines []SearchEngine `json:"mixed_engines,omitempty"`

	// DefaultEngine is the primary search engine to use if no specific order is provided.
	DefaultEngine SearchEngine `json:"default_engine,omitempty"`

	// DefaultFallbackOrder defines the sequence of search engines to try in case of failure.
	DefaultFallbackOrder []SearchEngine `json:"default_fallback_order,omitempty"`

	// EnableFallback, if true, allows the strategy to try the next engine in the order upon failure.
	EnableFallback bool `json:"enable_fallback"`

	// FailFast, if true, causes the strategy to stop immediately after the first engine failure.
	FailFast bool `json:"fail_fast"`
}

// DefaultSearchStrategy provides a default implementation for the search strategy.
// It supports both fallback and mixed search modes.
type DefaultSearchStrategy struct {
	mu       sync.RWMutex
	config   *SearchStrategyConfig
	adapters map[SearchEngine]SearchAdapter
}

// NewDefaultSearchStrategy creates a new instance of the default search strategy
// with the provided configuration. It also sets default values for any missing configuration.
func NewDefaultSearchStrategy(config *SearchStrategyConfig) *DefaultSearchStrategy {
	if config == nil {
		config = &SearchStrategyConfig{}
	}

	// Set default values.
	if config.DefaultEngine == "" {
		config.DefaultEngine = "searxng"
	}

	if len(config.DefaultFallbackOrder) == 0 {
		config.DefaultFallbackOrder = []SearchEngine{
			"searxng",
			"firecrawl",
		}
	}

	if config.EnableFallback == false {
		config.EnableFallback = true // Enable fallback by default.
	}

	// Set default search breadth.
	if config.Breadth <= 0 {
		config.Breadth = 10 // Default to 10 results per engine.
	}

	return &DefaultSearchStrategy{
		config:   config,
		adapters: make(map[SearchEngine]SearchAdapter),
	}
}

// NewDefaultSearchStrategyFromConfig creates a default search strategy instance
// from the global application configuration. It ensures that only enabled engines are used.
func NewDefaultSearchStrategyFromConfig() (SearchStrategy, error) {
	searchConfig := config.GetSearchConfig()
	if searchConfig == nil {
		return nil, fmt.Errorf("search engine configuration not found")
	}

	// Check if there are any enabled search engines.
	enabledEngines := searchConfig.Engines
	if len(enabledEngines) == 0 {
		return nil, fmt.Errorf("no enabled search engines found")
	}

	// Build the strategy configuration from the global config.
	strategyConfig := &SearchStrategyConfig{
		Breadth:              searchConfig.Strategy.Breadth,
		MixedEngines:         convertStringSliceToSearchEngines(searchConfig.Strategy.MixedEngines),
		DefaultEngine:        SearchEngine(searchConfig.Strategy.DefaultEngine),
		DefaultFallbackOrder: convertStringSliceToSearchEngines(searchConfig.Strategy.DefaultFallbackOrder),
		EnableFallback:       searchConfig.Strategy.EnableFallback,
		FailFast:             searchConfig.Strategy.FailFast,
	}

	// Filter the fallback order list to include only enabled engines.
	if len(searchConfig.Engines) > 0 && len(strategyConfig.DefaultFallbackOrder) > 0 {
		var engines []SearchEngine
		for _, engineName := range searchConfig.Strategy.DefaultFallbackOrder {
			if engineConfig, exists := searchConfig.Engines[engineName]; exists && engineConfig.Enabled {
				engines = append(engines, SearchEngine(engineName))
			}
		}
		strategyConfig.DefaultFallbackOrder = engines
	}

	// If mixed search engines are configured, filter them to include only enabled ones.
	if len(strategyConfig.MixedEngines) > 0 {
		var enabledMixedEngines []SearchEngine
		for _, engine := range strategyConfig.MixedEngines {
			engineName := string(engine)
			if engineConfig, exists := searchConfig.Engines[engineName]; exists && engineConfig.Enabled {
				enabledMixedEngines = append(enabledMixedEngines, engine)
			}
		}
		strategyConfig.MixedEngines = enabledMixedEngines
	}

	return NewDefaultSearchStrategy(strategyConfig), nil
}

// convertStringSliceToSearchEngines converts a slice of strings to a slice of SearchEngine type.
func convertStringSliceToSearchEngines(engines []string) []SearchEngine {
	var result []SearchEngine
	for _, engine := range engines {
		result = append(result, SearchEngine(engine))
	}
	return result
}

// Execute determines whether to perform a mixed search or a fallback search
// based on the configuration.
func (s *DefaultSearchStrategy) Execute(ctx context.Context, request *SearchRequest) (*SearchResponse, error) {
	// If mixed search engines are configured, perform a mixed search.
	if len(s.config.MixedEngines) > 0 {
		return s.executeMixedSearch(ctx, request)
	}

	// Otherwise, perform the fallback search logic.
	return s.executeFallbackSearch(ctx, request)
}

// executeMixedSearch performs a search across multiple engines concurrently and aggregates the results.
func (s *DefaultSearchStrategy) executeMixedSearch(ctx context.Context, request *SearchRequest) (*SearchResponse, error) {
	engines := s.config.MixedEngines
	if len(engines) == 0 {
		return nil, fmt.Errorf("no mixed search engines are configured")
	}

	// A channel to collect results from different engines.
	type engineResult struct {
		engine   SearchEngine
		response *SearchResponse
		err      error
	}

	resultChan := make(chan engineResult, len(engines))
	var wg sync.WaitGroup

	// Perform search on each engine concurrently.
	for _, engine := range engines {
		wg.Add(1)
		go func(eng SearchEngine) {
			defer wg.Done()

			adapter, err := s.getOrCreateAdapter(eng)
			if err != nil {
				resultChan <- engineResult{engine: eng, err: fmt.Errorf("failed to create adapter for engine %s: %w", eng, err)}
				return
			}

			response, err := adapter.Search(ctx, request)
			resultChan <- engineResult{engine: eng, response: response, err: err}
		}(engine)
	}

	// Wait for all searches to complete.
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect and merge the results.
	var allResults []*SearchResultItem
	var errors []error
	usedEngines := make(map[string]int)
	seenURLs := make(map[string]bool) // A set of URLs for deduplication.

	for result := range resultChan {
		if result.err != nil {
			errors = append(errors, fmt.Errorf("engine %s failed: %w", result.engine, result.err))
			continue
		}

		if result.response != nil && len(result.response.Results) > 0 {
			// Get the top 'breadth' results.
			maxResults := s.config.Breadth
			if maxResults <= 0 {
				maxResults = len(result.response.Results)
			} else if maxResults > len(result.response.Results) {
				maxResults = len(result.response.Results)
			}

			engineResults := result.response.Results[:maxResults]

			// Deduplicate results and add engine identifier to each result.
			var uniqueResults []*SearchResultItem
			for _, searchResult := range engineResults {
				// Check for URL duplicates.
				if searchResult.URL != "" && seenURLs[searchResult.URL] {
					continue // Skip duplicate URL.
				}

				if searchResult.Metadata == nil {
					searchResult.Metadata = make(map[string]interface{})
				}
				searchResult.Metadata["search_engine"] = string(result.engine)
				searchResult.Metadata["strategy"] = "mixed"

				// Mark the URL as seen.
				if searchResult.URL != "" {
					seenURLs[searchResult.URL] = true
				}

				uniqueResults = append(uniqueResults, searchResult)
			}

			allResults = append(allResults, uniqueResults...)
			usedEngines[string(result.engine)] = len(uniqueResults)
		}
	}

	// Handle case where all engines fail.
	if len(allResults) == 0 {
		if len(errors) > 0 {
			return nil, fmt.Errorf("all mixed search engines failed: %v", errors)
		}
		return nil, fmt.Errorf("no results from any mixed search engine")
	}

	// Build the final mixed search response.
	response := &SearchResponse{
		Query:      request.Query,
		Results:    allResults,
		TotalCount: len(allResults),
	}

	return response, nil
}

// executeFallbackSearch performs a search using a fallback mechanism.
// It tries engines one by one from a predefined order until a successful result is obtained.
func (s *DefaultSearchStrategy) executeFallbackSearch(ctx context.Context, request *SearchRequest) (*SearchResponse, error) {
	engines := s.getEngineOrder()
	seenURLs := make(map[string]bool) // A set for deduplicating URLs.

	var lastErr error
	for i, engine := range engines {
		adapter, err := s.getOrCreateAdapter(engine)
		if err != nil {
			lastErr = fmt.Errorf("failed to create adapter for engine %s: %w", engine, err)
			if s.config.FailFast || i == len(engines)-1 {
				continue
			}
		}

		response, err := adapter.Search(ctx, request)
		if err == nil {
			// On success, mark the response with the engine used.
			if response.Results != nil {
				// Handle deduplication.
				var uniqueResults []*SearchResultItem
				for _, result := range response.Results {
					// Check for URL duplicates.
					if result.URL != "" && seenURLs[result.URL] {
						continue // Skip duplicate URL.
					}

					if result.Metadata == nil {
						result.Metadata = make(map[string]interface{})
					}
					result.Metadata["search_engine"] = string(engine)
					result.Metadata["strategy"] = "fallback"
					result.Metadata["attempt"] = i + 1

					// Mark URL as seen.
					if result.URL != "" {
						seenURLs[result.URL] = true
					}

					uniqueResults = append(uniqueResults, result)
				}

				// Limit the number of results if breadth is set.
				if s.config.Breadth > 0 && len(uniqueResults) > s.config.Breadth {
					uniqueResults = uniqueResults[:s.config.Breadth]
				}

				response.Results = uniqueResults
				response.TotalCount = len(uniqueResults)
			}
			return response, nil
		}

		lastErr = fmt.Errorf("engine %s search failed: %w", engine, err)

		// If fallback is disabled or this is the last engine, break the loop.
		if !s.config.EnableFallback || i == len(engines)-1 {
			break
		}
	}

	return nil, fmt.Errorf("all search engines failed, last error: %w", lastErr)
}

// getEngineOrder determines the order of execution for search engines.
func (s *DefaultSearchStrategy) getEngineOrder() []SearchEngine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 1. Use the engine list from the configuration if available.
	if len(s.config.DefaultFallbackOrder) > 0 {
		// Return a copy of the list from the config.
		engines := make([]SearchEngine, len(s.config.DefaultFallbackOrder))
		copy(engines, s.config.DefaultFallbackOrder)
		return engines
	}

	// 2. Use the default engine as a final fallback.
	return []SearchEngine{s.config.DefaultEngine}
}

// getOrCreateAdapter retrieves a cached adapter instance or creates a new one if not available.
func (s *DefaultSearchStrategy) getOrCreateAdapter(engine SearchEngine) (SearchAdapter, error) {
	s.mu.RLock()
	if adapter, exists := s.adapters[engine]; exists {
		s.mu.RUnlock()
		return adapter, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check in case another goroutine created the adapter.
	if adapter, exists := s.adapters[engine]; exists {
		return adapter, nil
	}

	// Create a new adapter. Configuration is read when the adapter is registered.
	adapter, err := CreateSearchAdapter(engine)
	if err != nil {
		return nil, err
	}

	s.adapters[engine] = adapter
	return adapter, nil
}
