package web

import (
	"context"
	"fmt"
	"sync"

	"github.com/anboat/strato-sdk/config"
)

// WebStrategy defines the interface for a web scraping strategy.
type WebStrategy interface {
	// Execute executes the web scraping strategy for a single URL.
	Execute(ctx context.Context, url string, options *ScrapeOptions) (*WebContent, error)

	// ExecuteMultiple executes the web scraping strategy for multiple URLs.
	ExecuteMultiple(ctx context.Context, urls []string, options *ScrapeOptions) ([]*WebContent, error)
}

// WebStrategyConfig holds the configuration for a web scraping strategy.
type WebStrategyConfig struct {
	// DefaultScraper is the default scraper to use if no list is specified.
	DefaultScraper WebScraper `json:"default_scraper,omitempty"`

	// DefaultFallbackOrder is the default order of scrapers to try.
	DefaultFallbackOrder []WebScraper `json:"default_fallback_order,omitempty"`

	// EnableFallback determines whether to try the next scraper on failure.
	EnableFallback bool `json:"enable_fallback"`

	// FailFast determines whether to stop immediately after the first failure.
	FailFast bool `json:"fail_fast"`
}

// DefaultWebStrategy is the default implementation of the web scraping strategy.
type DefaultWebStrategy struct {
	mu       sync.RWMutex
	config   *WebStrategyConfig
	adapters map[WebScraper]WebAdapter
}

// NewDefaultWebStrategy creates a new default web scraping strategy.
func NewDefaultWebStrategy(config *WebStrategyConfig) *DefaultWebStrategy {
	if config == nil {
		config = &WebStrategyConfig{}
	}

	// Set default values
	if config.DefaultScraper == "" {
		config.DefaultScraper = "jina"
	}

	if len(config.DefaultFallbackOrder) == 0 {
		config.DefaultFallbackOrder = []WebScraper{
			"jina",
			"firecrawl",
		}
	}

	if !config.EnableFallback {
		config.EnableFallback = true // Enable fallback by default
	}

	return &DefaultWebStrategy{
		config:   config,
		adapters: make(map[WebScraper]WebAdapter),
	}
}

// NewDefaultWebStrategyFromConfig creates a default web scraping strategy from the global configuration.
func NewDefaultWebStrategyFromConfig() (WebStrategy, error) {
	webConfig := config.GetWebConfig()
	if webConfig == nil {
		return nil, fmt.Errorf("web scraper configuration not found")
	}

	// Check for enabled web scrapers
	enabledScrapers := webConfig.Scrapers
	if len(enabledScrapers) == 0 {
		return nil, fmt.Errorf("no enabled web scrapers found")
	}

	// Build strategy configuration
	strategyConfig := &WebStrategyConfig{
		DefaultScraper:       WebScraper(webConfig.Strategy.DefaultScraper),
		DefaultFallbackOrder: convertStringSliceToWebScrapers(webConfig.Strategy.DefaultFallbackOrder),
		EnableFallback:       webConfig.Strategy.EnableFallback,
		FailFast:             webConfig.Strategy.FailFast,
	}

	// Build the list of scraper configurations (only enabled ones)
	if len(webConfig.Scrapers) > 0 && len(strategyConfig.DefaultFallbackOrder) > 0 {
		var scrapers []WebScraper
		for _, scraperName := range webConfig.Strategy.DefaultFallbackOrder {
			if scraperConfig, exists := webConfig.Scrapers[scraperName]; exists && scraperConfig.Enabled {
				scrapers = append(scrapers, WebScraper(scraperName))
			}
		}
		strategyConfig.DefaultFallbackOrder = scrapers
	}

	return NewDefaultWebStrategy(strategyConfig), nil
}

// convertStringSliceToWebScrapers converts a slice of strings to a slice of WebScrapers.
func convertStringSliceToWebScrapers(scrapers []string) []WebScraper {
	var result []WebScraper
	for _, scraper := range scrapers {
		result = append(result, WebScraper(scraper))
	}
	return result
}

// Execute executes the web scraping strategy for a single URL.
func (s *DefaultWebStrategy) Execute(ctx context.Context, url string, options *ScrapeOptions) (*WebContent, error) {
	scrapers := s.getScraperOrder()

	var lastErr error
	for i, scraper := range scrapers {
		adapter, err := s.getOrCreateAdapter(scraper)
		if err != nil {
			lastErr = fmt.Errorf("failed to create adapter for %s: %w", scraper, err)
			if s.config.FailFast || i == len(scrapers)-1 {
				continue
			}
		}

		result, err := adapter.Scrape(ctx, url, options)
		if err == nil {
			// On success, you can mark the used engine in the result metadata if needed.
			if result != nil {
				if result.Images == nil {
					result.Images = make([]Image, 0)
				}
				if result.Links == nil {
					result.Links = make([]Link, 0)
				}
			}
			return result, nil
		}

		lastErr = fmt.Errorf("scraper %s failed: %w", scraper, err)

		// If fallback is disabled or this is the last scraper, return the error.
		if !s.config.EnableFallback || i == len(scrapers)-1 {
			break
		}
	}

	return nil, fmt.Errorf("all web scrapers failed, last error: %w", lastErr)
}

// ExecuteMultiple executes the web scraping strategy for multiple URLs.
func (s *DefaultWebStrategy) ExecuteMultiple(ctx context.Context, urls []string, options *ScrapeOptions) ([]*WebContent, error) {
	scrapers := s.getScraperOrder()

	var lastErr error
	for i, scraper := range scrapers {
		adapter, err := s.getOrCreateAdapter(scraper)
		if err != nil {
			lastErr = fmt.Errorf("failed to create adapter for %s: %w", scraper, err)
			if s.config.FailFast || i == len(scrapers)-1 {
				continue
			}
		}

		results, err := adapter.ScrapeMultiple(ctx, urls, options)
		if err == nil {
			return results, nil
		}

		lastErr = fmt.Errorf("scraper %s failed to scrape multiple URLs: %w", scraper, err)

		// If fallback is disabled or this is the last scraper, return the error.
		if !s.config.EnableFallback || i == len(scrapers)-1 {
			break
		}
	}

	return nil, fmt.Errorf("all web scrapers failed, last error: %w", lastErr)
}

// getScraperOrder returns the order of scrapers to be executed.
func (s *DefaultWebStrategy) getScraperOrder() []WebScraper {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 1. Use the scraper list from the configuration.
	if len(s.config.DefaultFallbackOrder) > 0 {
		// Return a copy of the scraper list from the config.
		scrapers := make([]WebScraper, len(s.config.DefaultFallbackOrder))
		copy(scrapers, s.config.DefaultFallbackOrder)
		return scrapers
	}

	// 2. Use the default scraper as a final fallback.
	return []WebScraper{s.config.DefaultScraper}
}

// getOrCreateAdapter gets or creates an adapter for the given scraper.
func (s *DefaultWebStrategy) getOrCreateAdapter(scraper WebScraper) (WebAdapter, error) {
	s.mu.RLock()
	adapter, exists := s.adapters[scraper]
	s.mu.RUnlock()

	if exists {
		return adapter, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double check
	adapter, exists = s.adapters[scraper]
	if exists {
		return adapter, nil
	}

	newAdapter, err := CreateWebAdapter(scraper)
	if err != nil {
		return nil, err
	}

	s.adapters[scraper] = newAdapter
	return newAdapter, nil
}
