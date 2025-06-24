package firecrawl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/anboat/strato-sdk/adapters/web"
	"github.com/anboat/strato-sdk/pkg/logging"
	"io"
	"net/http"
	"time"
)

// Constants for the Firecrawl adapter.
const (
	FirecrawlAPIBaseURL = "https://api.firecrawl.dev/v1/scrape"
	DefaultTimeout      = 90 * time.Second
)

// Client is a client for the Firecrawl API.
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// ClientConfig holds the configuration for the Firecrawl client.
type ClientConfig struct {
	APIKey     string        `json:"api_key"`
	BaseURL    string        `json:"base_url,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`
	HTTPClient *http.Client  `json:"-"`
}

// ScrapeRequest defines the request body for the /v0/scrape endpoint.
type ScrapeRequest struct {
	URL              string            `json:"url"`
	PageOptions      *PageOptions      `json:"pageOptions,omitempty"`
	ExtractorOptions *ExtractorOptions `json:"extractorOptions,omitempty"`
}

// PageOptions defines page-specific options for scraping.
type PageOptions struct {
	Screenshots bool `json:"screenshots,omitempty"`
	WaitFor     int  `json:"waitFor,omitempty"` // in milliseconds
}

// ExtractorOptions defines what to extract from the page.
type ExtractorOptions struct {
	Mode             string `json:"mode,omitempty"` // e.g., "llm-extraction"
	ExtractionPrompt string `json:"extractionPrompt,omitempty"`
}

// ScrapeResponse defines the successful response from the /v0/scrape endpoint.
type ScrapeResponse struct {
	Success bool        `json:"success"`
	Data    *ScrapeData `json:"data"`
}

// ScrapeData contains the actual scraped content.
type ScrapeData struct {
	Content     string                 `json:"content"`
	Markdown    string                 `json:"markdown"`
	HTML        string                 `json:"html"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	URL         string                 `json:"url"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewClient creates a new Firecrawl client.
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = &ClientConfig{}
	}

	if config.BaseURL == "" {
		config.BaseURL = FirecrawlAPIBaseURL
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Timeout: config.Timeout,
		}
	}

	return &Client{
		apiKey:     config.APIKey,
		httpClient: config.HTTPClient,
		baseURL:    config.BaseURL,
	}
}

// Scrape implements the web.WebAdapter interface for a single URL.
func (c *Client) Scrape(ctx context.Context, targetURL string, options *web.ScrapeOptions) (*web.WebContent, error) {
	// Build request body
	reqBody := ScrapeRequest{
		URL: targetURL,
	}

	if options != nil {
		if options.Timeout > 0 {
			reqBody.PageOptions = &PageOptions{
				WaitFor: options.Timeout * 1000, // Firecrawl expects milliseconds
			}
		}
	}

	// Marshal request body to JSON
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	requestURL := c.baseURL
	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	c.setRequestHeaders(req)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logging.Warnf("failed to close response body: %v", err)
		}
	}(resp.Body)

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	return c.parseResponse(resp, options)
}

// ScrapeMultiple implements the web.WebAdapter interface for multiple URLs.
func (c *Client) ScrapeMultiple(ctx context.Context, urls []string, options *web.ScrapeOptions) ([]*web.WebContent, error) {
	// The Firecrawl scrape API does not support multiple URLs in a single request.
	// We will iterate and scrape them one by one.
	var results []*web.WebContent
	for _, url := range urls {
		result, err := c.Scrape(ctx, url, options)
		if err != nil {
			// In a real-world scenario, you might want to collect errors.
			// For now, we'll just log or handle the error for the individual URL and skip it.
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

func (c *Client) setRequestHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

func (c *Client) parseResponse(resp *http.Response, options *web.ScrapeOptions) (*web.WebContent, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var firecrawlResp ScrapeResponse
	if err := json.Unmarshal(body, &firecrawlResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	if !firecrawlResp.Success || firecrawlResp.Data == nil {
		return nil, fmt.Errorf("firecrawl API returned an unsuccessful response")
	}

	data := firecrawlResp.Data
	content := &web.WebContent{
		URL:   data.URL,
		Title: data.Title,
	}

	format := web.FormatText
	if options != nil && options.Format.IsValid() {
		format = options.Format
	}

	switch format {
	case web.FormatMarkdown:
		content.Content = data.Markdown
	case web.FormatHTML:
		content.Content = data.HTML
	default: // web.FormatText
		content.Content = data.Content
	}

	return content, nil
}
