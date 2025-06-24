package jina

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/anboat/strato-sdk/adapters/web"
)

// Constants for the Jina adapter.
const (
	// JinaReaderBaseURL is the base URL for the Jina Reader API.
	JinaReaderBaseURL = "https://r.jina.ai"

	// DefaultTimeout is the default timeout for requests.
	DefaultTimeout = 30 * time.Second
)

// Client is a client for the Jina AI API.
type Client struct {
	// Authentication information.
	apiKey string

	// HTTP client.
	httpClient *http.Client

	// Basic configuration.
	baseURL string
	timeout time.Duration
}

// ClientConfig holds the configuration for the Jina client.
type ClientConfig struct {
	APIKey     string        `json:"api_key,omitempty"`
	BaseURL    string        `json:"base_url,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`
	HTTPClient *http.Client  `json:"-"`
}

// JinaResponse is the structure for a Jina API response.
type JinaResponse struct {
	Code   int              `json:"code"`
	Status int              `json:"status"`
	Data   JinaResponseData `json:"data"`
	Meta   JinaResponseMeta `json:"meta"`
}

// JinaResponseData is the data part of a Jina API response.
type JinaResponseData struct {
	Images        map[string]string `json:"images"`
	Links         map[string]string `json:"links"`
	Text          string            `json:"text"`
	HTML          string            `json:"html"`
	Content       string            `json:"content"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	URL           string            `json:"url"`
	PublishedTime string            `json:"publishedTime"`
	Usage         JinaUsage         `json:"usage"`
}

// JinaResponseMeta is the metadata part of a Jina API response.
type JinaResponseMeta struct {
	Usage JinaUsage `json:"usage"`
}

// JinaUsage holds usage statistics.
type JinaUsage struct {
	Tokens int `json:"tokens"`
}

// NewClient creates a new Jina client.
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = &ClientConfig{}
	}

	// Set default values.
	if config.BaseURL == "" {
		config.BaseURL = JinaReaderBaseURL
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
		timeout:    config.Timeout,
	}
}

// Scrape implements the single-page scraping functionality.
func (c *Client) Scrape(ctx context.Context, targetURL string, options *web.ScrapeOptions) (*web.WebContent, error) {
	// Build the request URL.
	requestURL := fmt.Sprintf("%s/%s", c.baseURL, targetURL)

	// Create an HTTP request.
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set request headers.
	c.setRequestHeaders(req, options)

	// Send the request.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status.
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response.
	return c.parseResponse(resp, options)
}

// ScrapeMultiple implements batch scraping.
func (c *Client) ScrapeMultiple(ctx context.Context, urls []string, options *web.ScrapeOptions) ([]*web.WebContent, error) {
	results := make([]*web.WebContent, 0, len(urls))

	// Concurrent scraping (simplified version, a goroutine pool could be used in practice).
	for _, targetURL := range urls {
		result, err := c.Scrape(ctx, targetURL, options)
		if err != nil {
			// Log the error but continue processing other URLs.
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// setRequestHeaders sets the request headers.
func (c *Client) setRequestHeaders(req *http.Request, options *web.ScrapeOptions) {
	// Set basic headers.
	req.Header.Set("Accept", "application/json")
	// Set API Key.
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	if options != nil {
		// Set return format.
		if options.Format.IsValid() {
			req.Header.Set("X-Return-Format", options.Format.String())
		} else {
			req.Header.Set("X-Return-Format", "text")
		}

		// Set timeout.
		if options.Timeout > 0 {
			req.Header.Set("X-Timeout", strconv.Itoa(options.Timeout))
		}

		// Set image and link summaries.
		if options.ImagesSummary {
			req.Header.Set("X-With-Images-Summary", "true")
		}
		if options.LinksSummary {
			req.Header.Set("X-With-Links-Summary", "true")
		}

		// Set Cookies.
		if len(options.Cookies) > 0 {
			var cookies []string
			for name, value := range options.Cookies {
				cookies = append(cookies, fmt.Sprintf("%s=%s", name, value))
			}
			req.Header.Set("Cookie", strings.Join(cookies, "; "))
		}
	}
}

// parseResponse parses the Jina API response.
func (c *Client) parseResponse(resp *http.Response, options *web.ScrapeOptions) (*web.WebContent, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 解析 JSON 响应
	var jinaResp JinaResponse
	if err := json.Unmarshal(body, &jinaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	if jinaResp.Code != http.StatusOK {
		return nil, fmt.Errorf("Jina API returned an error (code: %d, status: %d)", jinaResp.Code, jinaResp.Status)
	}

	// 构建 WebContent 响应
	content := &web.WebContent{
		URL:   jinaResp.Data.URL,
		Title: jinaResp.Data.Title,
	}

	// Parse links and images if requested
	if options != nil {
		switch options.Format {
		case web.FormatHTML:
			content.Content = jinaResp.Data.HTML
		case web.FormatMarkdown:
			content.Content = jinaResp.Data.Content
		case web.FormatText:
			fallthrough
		default:
			content.Content = jinaResp.Data.Text
		}
	} else {
		content.Content = jinaResp.Data.Text
	}

	if len(jinaResp.Data.Links) > 0 {
		content.Links = c.parseLinks(jinaResp.Data.Links)
	}

	if len(jinaResp.Data.Images) > 0 {
		content.Images = c.parseImages(jinaResp.Data.Images)
	}

	return content, nil
}

// parseLinks parses the links data from the Jina response.
func (c *Client) parseLinks(linksData map[string]string) []web.Link {
	var links []web.Link

	for text, url := range linksData {
		// Skip empty link or text
		if text == "" || url == "" {
			continue
		}

		links = append(links, web.Link{
			URL:  url,
			Text: text,
		})
	}
	return links
}

// parseImages parses the images data from the Jina response.
func (c *Client) parseImages(imagesData map[string]string) []web.Image {
	var images []web.Image

	for title, url := range imagesData {
		if url == "" {
			continue
		}

		images = append(images, web.Image{
			URL:   url,
			Title: title,
		})
	}

	return images
}
