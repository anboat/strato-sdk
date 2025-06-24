package searxng

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/anboat/strato-sdk/adapters/search"
	"github.com/anboat/strato-sdk/pkg/logging"
)

// SearXNGAdapter is the search adapter for SearXNG.
type SearXNGAdapter struct {
	config *SearXNGConfig
	client *http.Client
}

// SearXNGConfig holds the configuration for the SearXNG search adapter.
type SearXNGConfig struct {
	BaseURL    string        `json:"base_url"`    // SearXNG instance URL.
	UserAgent  string        `json:"user_agent"`  // User agent for requests.
	Timeout    time.Duration `json:"timeout"`     // Request timeout.
	Language   string        `json:"language"`    // Search language.
	SafeSearch string        `json:"safe_search"` // Safe search level (e.g., "0" none, "1" moderate, "2" strict).
	Categories string        `json:"categories"`  // Search categories (e.g., "general", "news").
	Engines    string        `json:"engines"`     // Specific search engines to use.
}

// SearXNGResponse represents the response from the SearXNG API.
type SearXNGResponse struct {
	Query               string          `json:"query"`
	NumberOfResults     int             `json:"number_of_results"`
	Results             []SearXNGResult `json:"results"`
	Answers             []interface{}   `json:"answers"`
	Corrections         []interface{}   `json:"corrections"`
	Infoboxes           []interface{}   `json:"infoboxes"`
	Suggestions         []interface{}   `json:"suggestions"`
	UnresponsiveEngines []interface{}   `json:"unresponsive_engines"`
}

// SearXNGResult represents a single search result from SearXNG.
type SearXNGResult struct {
	URL         string                 `json:"url"`
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	Engine      string                 `json:"engine"`
	ParsedURL   []string               `json:"parsed_url"`
	Template    string                 `json:"template"`
	Engines     []string               `json:"engines"`
	Positions   []int                  `json:"positions"`
	Score       float64                `json:"score"`
	Category    string                 `json:"category"`
	PublishDate string                 `json:"publishedDate,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewSearXNGAdapter creates a new SearXNG search adapter.
func NewSearXNGAdapter(config *SearXNGConfig) *SearXNGAdapter {
	// Set default values.
	if config.BaseURL == "" {
		config.BaseURL = "https://searx.be" // Default to a public instance.
	}
	if config.UserAgent == "" {
		config.UserAgent = "SearXNG-Go-Client/1.0"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.Language == "" {
		config.Language = "zh-CN"
	}
	if config.SafeSearch == "" {
		config.SafeSearch = "1" // Moderate safe search level.
	}
	if config.Categories == "" {
		config.Categories = "general"
	}

	// Remove trailing slash from BaseURL.
	config.BaseURL = strings.TrimSuffix(config.BaseURL, "/")

	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &SearXNGAdapter{
		config: config,
		client: client,
	}
}

// Search implements the search.SearchAdapter interface.
func (s *SearXNGAdapter) Search(ctx context.Context, request *search.SearchRequest) (*search.SearchResponse, error) {
	startTime := time.Now()

	// Build the request URL.
	requestURL, err := s.buildRequestURL(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build request URL: %w", err)
	}

	// Create the HTTP request.
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set request headers.
	req.Header.Set("User-Agent", s.config.UserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", s.config.Language)

	// Send the request.
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logging.Warnf("failed to close response body: %v", err)
		}
	}(resp.Body)

	// Check the response status code.
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response.
	var searxngResp SearXNGResponse
	if err := json.Unmarshal(body, &searxngResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	// Convert to the standard response format.
	return s.convertResponse(&searxngResp, time.Since(startTime)), nil
}

// buildRequestURL builds the request URL for the SearXNG API.
func (s *SearXNGAdapter) buildRequestURL(request *search.SearchRequest) (string, error) {
	if request.Query == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	// Build query parameters.
	params := url.Values{}
	params.Set("q", request.Query)
	params.Set("format", "json")

	// Set categories.
	if categories := s.getCategories(request); categories != "" {
		params.Set("categories", categories)
	}

	// Set language.
	if lang := s.getLanguage(request); lang != "" {
		params.Set("language", lang)
	}

	// Set safe search.
	if safeSearch := s.getSafeSearch(request); safeSearch != "" {
		params.Set("safesearch", safeSearch)
	}

	// Set pagination.
	if request.Offset > 0 {
		page := (request.Offset / max(request.Num, 10)) + 1
		params.Set("pageno", strconv.Itoa(page))
	}

	// Set specific search engines.
	if s.config.Engines != "" {
		params.Set("engines", s.config.Engines)
	}

	// Handle engine-specific parameters.
	if request.EngineParams != nil {
		if engines, ok := request.EngineParams["engines"].(string); ok && engines != "" {
			params.Set("engines", engines)
		}
		if timeRange, ok := request.EngineParams["time_range"].(string); ok && timeRange != "" {
			params.Set("time_range", timeRange)
		}
	}

	return fmt.Sprintf("%s/search?%s", s.config.BaseURL, params.Encode()), nil
}

// getCategories determines the search categories to use.
func (s *SearXNGAdapter) getCategories(request *search.SearchRequest) string {
	if request.EngineParams != nil {
		if categories, ok := request.EngineParams["categories"].(string); ok {
			return categories
		}
	}
	return s.config.Categories
}

// getLanguage determines the search language to use.
func (s *SearXNGAdapter) getLanguage(request *search.SearchRequest) string {
	if request.Lang != "" {
		return request.Lang
	}
	return s.config.Language
}

// getSafeSearch determines the safe search level to use.
func (s *SearXNGAdapter) getSafeSearch(request *search.SearchRequest) string {
	if request.SafeSearch != "" {
		// Map standard safe search terms to SearXNG values if necessary.
		switch strings.ToLower(request.SafeSearch) {
		case "off", "false", "0":
			return "0"
		case "moderate", "1":
			return "1"
		case "strict", "on", "true", "2":
			return "2"
		}
	}
	return s.config.SafeSearch
}

// convertResponse converts a SearXNG response to the standard search.SearchResponse format.
func (s *SearXNGAdapter) convertResponse(resp *SearXNGResponse, timeTaken time.Duration) *search.SearchResponse {
	results := make([]*search.SearchResultItem, len(resp.Results))
	for i, item := range resp.Results {
		results[i] = &search.SearchResultItem{
			Title:       item.Title,
			URL:         item.URL,
			Description: item.Content,
			Link:        item.URL,
			Snippet:     item.Content,
			Rank:        i + 1,
			Score:       item.Score,
			PublishDate: item.PublishDate,
			Metadata: map[string]interface{}{
				"engine":   item.Engine,
				"category": item.Category,
			},
		}
	}

	return &search.SearchResponse{
		Query:      resp.Query,
		Results:    results,
		TotalCount: resp.NumberOfResults,
		TimeTaken:  timeTaken.Milliseconds(),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
