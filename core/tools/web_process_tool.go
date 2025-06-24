package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/anboat/strato-sdk/adapters/web"
	"github.com/anboat/strato-sdk/config"
)

// Global variables to avoid repeated initialization.
var (
	webStrategy  web.WebStrategy
	webInitOnce  sync.Once
	webInitError error
)

// initWebAdapters initializes the web adapters and strategy.
// This function is executed only once.
func initWebAdapters() {
	webInitOnce.Do(func() {
		// 1. Register all web adapters.
		if err := web.RegisterAllWebAdapters(); err != nil {
			webInitError = fmt.Errorf("failed to initialize web adapters: %w", err)
			return
		}

		// 2. Create the web strategy from the configuration.
		strategy, err := web.NewDefaultWebStrategyFromConfig()
		if err != nil {
			webInitError = fmt.Errorf("failed to create web strategy: %w", err)
			return
		}

		webStrategy = strategy
	})
}

// WebScrapeRequest defines the parameters for a web scraping request.
// It uses jsonschema tags to define parameter constraints for the Eino framework.
type WebScrapeRequest struct {
	URL     string   `json:"url" jsonschema:"description=The URL of the web page to scrape (mutually exclusive with 'urls')."`
	URLs    []string `json:"urls" jsonschema:"description=A list of web page URLs to scrape in batch (mutually exclusive with 'url')."`
	Format  string   `json:"format" jsonschema:"enum=text,enum=markdown,enum=html,description=The desired format for the returned content, default is 'text'."`
	Timeout int      `json:"timeout" jsonschema:"description=Request timeout in seconds, default is 30."`
}

// WebScrapeResponse defines the structure of the web scraping response.
type WebScrapeResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message,omitempty"`
	Results []*web.WebContent `json:"results,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// webProcessFunc is the underlying implementation of the web processing tool.
func webProcessFunc(ctx context.Context, request *WebScrapeRequest) (*WebScrapeResponse, error) {
	startTime := time.Now()

	// Validate the request parameters.
	if err := validateWebScrapeRequest(request); err != nil {
		return &WebScrapeResponse{
			Success: false,
			Error:   fmt.Sprintf("parameter validation failed: %v", err),
		}, nil
	}

	// Ensure web adapters are initialized (executes only once).
	initWebAdapters()
	if webInitError != nil {
		return &WebScrapeResponse{
			Success: false,
			Error:   webInitError.Error(),
		}, nil
	}

	// Build the scrape options.
	scrapeOptions := buildScrapeOptions(request)

	// Execute the scraping operation.
	var results []*web.WebContent
	var scrapeErr error

	if request.URL != "" {
		// Single URL scrape.
		result, err := webStrategy.Execute(ctx, request.URL, scrapeOptions)
		if err != nil {
			scrapeErr = err
		} else {
			results = []*web.WebContent{result}
		}
	} else if len(request.URLs) > 0 {
		// Batch URL scrape.
		results, scrapeErr = webStrategy.ExecuteMultiple(ctx, request.URLs, scrapeOptions)
	}

	// Handle scraping errors.
	if scrapeErr != nil {
		return &WebScrapeResponse{
			Success: false,
			Error:   fmt.Sprintf("scraping failed: %v", scrapeErr),
		}, nil
	}

	// Build the final response.
	response := buildWebScrapeResponse(results, startTime)
	return response, nil
}

// validateWebScrapeRequest validates the web scrape request parameters.
func validateWebScrapeRequest(request *WebScrapeRequest) error {
	// Check URL and URLs parameters.
	if request.URL == "" && len(request.URLs) == 0 {
		return fmt.Errorf("either 'url' or 'urls' parameter must be provided")
	}

	if request.URL != "" && len(request.URLs) > 0 {
		return fmt.Errorf("'url' and 'urls' parameters cannot be provided simultaneously")
	}

	// Validate the format parameter.
	if request.Format != "" {
		format := web.Format(request.Format)
		if !format.IsValid() {
			return fmt.Errorf("unsupported format: %s, supported formats are: text, markdown, html", request.Format)
		}
	}

	// Validate the timeout parameter.
	if request.Timeout < 0 {
		return fmt.Errorf("timeout cannot be a negative number")
	}

	return nil
}

// buildScrapeOptions builds the internal web.ScrapeOptions from the tool's WebScrapeRequest.
func buildScrapeOptions(request *WebScrapeRequest) *web.ScrapeOptions {
	options := &web.ScrapeOptions{}

	// Set the format.
	if request.Format != "" {
		options.Format = web.Format(request.Format)
	} else {
		options.Format = web.FormatText // Default format.
	}

	// Set the timeout.
	if request.Timeout > 0 {
		options.Timeout = request.Timeout
	} else {
		// Use the default timeout.
		options.Timeout = 30
	}

	// By default, enable summaries for links and images.
	options.LinksSummary = true
	options.ImagesSummary = true

	return options
}

// buildWebScrapeResponse builds the final WebScrapeResponse.
func buildWebScrapeResponse(results []*web.WebContent, startTime time.Time) *WebScrapeResponse {
	response := &WebScrapeResponse{
		Success: true,
		Results: results,
		Message: fmt.Sprintf("Successfully scraped %d pages in %dms", len(results), time.Since(startTime).Milliseconds()),
	}

	return response
}

// NewWebProcessTool creates a new invokable web processing tool.
// It initializes all web scraping adapters and strategies based on the application configuration.
func NewWebProcessTool() (tool.InvokableTool, error) {
	// Check for basic web configuration.
	webConfig := config.GetWebConfig()
	if webConfig == nil {
		return nil, fmt.Errorf("web scraper configuration not found")
	}

	// Pre-initialize web adapters.
	initWebAdapters()
	if webInitError != nil {
		return nil, webInitError
	}

	// Create the InvokableTool using Eino's utility function.
	return utils.InferTool(
		"web_scrape",
		"An intelligent web scraping tool that supports multiple scraping engines for single-page or batch processing. It features a multi-engine fallback strategy, and SDK users can extend it with custom scraping adapters.",
		webProcessFunc,
	)
}
