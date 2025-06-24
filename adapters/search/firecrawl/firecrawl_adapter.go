package firecrawl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/anboat/strato-sdk/adapters/search"
	"io"
	"net/http"
	"time"
)

// FirecrawlAdapter is the search adapter for Firecrawl.
type FirecrawlAdapter struct {
	client *http.Client
	config *FirecrawlConfig
}

// FirecrawlConfig holds the configuration for the Firecrawl search adapter.
type FirecrawlConfig struct {
	APIKey    string        `json:"api_key"`    // Required: Firecrawl API key
	BaseURL   string        `json:"base_url"`   // API base URL
	Timeout   time.Duration `json:"timeout"`    // Request timeout
	UserAgent string        `json:"user_agent"` // User agent
}

// FirecrawlSearchResponse is the response from the Firecrawl API.
type FirecrawlSearchResponse struct {
	Success bool                    `json:"success"`
	Data    []FirecrawlSearchResult `json:"data"`
}

// FirecrawlSearchResult is a single search result item from Firecrawl.
type FirecrawlSearchResult struct {
	Title       string                 `json:"title"`
	URL         string                 `json:"url"`
	Description string                 `json:"description"`
	Markdown    string                 `json:"markdown,omitempty"`
	Links       []string               `json:"links,omitempty"`
	Content     string                 `json:"content,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ScrapeOptions defines the scraping options for Firecrawl.
type ScrapeOptions struct {
	Formats []string `json:"formats,omitempty"`
}

// FirecrawlSearchRequest is a struct for the request body of Firecrawl.
type FirecrawlSearchRequest struct {
	Query             string         `json:"query"`
	Limit             int            `json:"limit,omitempty"`
	Location          string         `json:"location,omitempty"`
	Tbs               string         `json:"tbs,omitempty"`
	Timeout           int            `json:"timeout,omitempty"`
	IgnoreInvalidURLs bool           `json:"ignoreInvalidURLs,omitempty"`
	ScrapeOptions     *ScrapeOptions `json:"scrapeOptions,omitempty"`
}

// NewFirecrawlAdapter creates a new Firecrawl search adapter.
func NewFirecrawlAdapter(config *FirecrawlConfig) *FirecrawlAdapter {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.firecrawl.dev/v1/search"
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.UserAgent == "" {
		config.UserAgent = "Strato-SDK-Bot/1.0"
	}

	return &FirecrawlAdapter{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Search implements the search interface.
func (f *FirecrawlAdapter) Search(ctx context.Context, request *search.SearchRequest) (*search.SearchResponse, error) {
	startTime := time.Now()

	firecrawlReq := FirecrawlSearchRequest{
		Query:    request.Query,
		Limit:    request.Num,
		Location: request.Region,
		Tbs:      request.TimeRange,
	}

	// Parse scrape_options and other Firecrawl-specific parameters from EngineParams.
	if request.EngineParams != nil {
		if params, ok := request.EngineParams["scrape_options"]; ok {
			if scrapeOptionsMap, ok := params.(map[string]interface{}); ok {
				opts := &ScrapeOptions{}
				if formats, ok := scrapeOptionsMap["formats"].([]interface{}); ok {
					for _, fVal := range formats {
						if formatStr, ok := fVal.(string); ok {
							opts.Formats = append(opts.Formats, formatStr)
						}
					}
				}
				firecrawlReq.ScrapeOptions = opts
			}
		}

		if timeout, ok := request.EngineParams["timeout"]; ok {
			switch t := timeout.(type) {
			case int:
				firecrawlReq.Timeout = t
			case float64:
				firecrawlReq.Timeout = int(t)
			}
		}

		if ignore, ok := request.EngineParams["ignore_invalid_urls"].(bool); ok {
			firecrawlReq.IgnoreInvalidURLs = ignore
		}
	}

	reqBody, err := json.Marshal(firecrawlReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal firecrawl request body: %w", err)
	}

	reqURL := f.config.BaseURL

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+f.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", f.config.UserAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to firecrawl: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read firecrawl response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("firecrawl API returned error status: %d, body: %s", resp.StatusCode, string(body))
	}

	var firecrawlResp FirecrawlSearchResponse
	if err := json.Unmarshal(body, &firecrawlResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal firecrawl response: %w", err)
	}

	if !firecrawlResp.Success {
		return nil, fmt.Errorf("firecrawl API returned success=false")
	}

	response := f.convertToSearchResponse(&firecrawlResp, request.Query, time.Since(startTime))
	return response, nil
}

func (f *FirecrawlAdapter) convertToSearchResponse(firecrawlResp *FirecrawlSearchResponse, query string, timeTaken time.Duration) *search.SearchResponse {
	results := make([]*search.SearchResultItem, 0, len(firecrawlResp.Data))
	for i, item := range firecrawlResp.Data {
		metadata := item.Metadata
		if metadata == nil {
			metadata = make(map[string]interface{})
		}

		if item.Markdown != "" {
			metadata["markdown"] = item.Markdown
		}
		if len(item.Links) > 0 {
			metadata["links"] = item.Links
		}

		results = append(results, &search.SearchResultItem{
			Title:       item.Title,
			URL:         item.URL,
			Description: item.Description,
			Link:        item.URL,
			Snippet:     item.Description,
			Rank:        i + 1,
			Metadata:    metadata,
		})
	}

	return &search.SearchResponse{
		Query:     query,
		Results:   results,
		TimeTaken: timeTaken.Milliseconds(),
	}
}
