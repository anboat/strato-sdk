package web

import (
	"context"
)

// WebScraper represents a web scraping engine identifier.
type WebScraper string

// String implements the fmt.Stringer interface.
func (e WebScraper) String() string {
	return string(e)
}

// Format represents the scraping format enumeration.
type Format string

const (
	FormatText     Format = "text"
	FormatHTML     Format = "html"
	FormatMarkdown Format = "markdown"
)

// String implements the fmt.Stringer interface.
func (f Format) String() string {
	return string(f)
}

// IsValid checks if the format is valid.
func (f Format) IsValid() bool {
	switch f {
	case FormatText, FormatHTML, FormatMarkdown:
		return true
	default:
		return false
	}
}

// WebAdapter is the interface for a web adapter.
type WebAdapter interface {
	// Scrape is the core function for scraping a single page.
	Scrape(ctx context.Context, url string, options *ScrapeOptions) (*WebContent, error)

	// ScrapeMultiple scrapes multiple pages concurrently.
	ScrapeMultiple(ctx context.Context, urls []string, options *ScrapeOptions) ([]*WebContent, error)
}

// ScrapeOptions holds the options for a scraping request.
type ScrapeOptions struct {
	// Basic options
	Format        Format            `json:"format,omitempty"`         // Scraping format enumeration
	Timeout       int               `json:"timeout,omitempty"`        // Timeout in seconds
	Cookies       map[string]string `json:"cookies,omitempty"`        // Cookies to use for the request
	ImagesSummary bool              `json:"images_summary,omitempty"` // Whether to include a summary of unique images at the end
	LinksSummary  bool              `json:"links_summary,omitempty"`  // Whether to include a summary of unique links at the end
}

// WebContent represents the content of a web page (simplified response).
type WebContent struct {
	// Basic information
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"` // Main content

	// Extracted structured data
	Links  []Link  `json:"links,omitempty"`
	Images []Image `json:"images,omitempty"`
}

// Link represents a hyperlink.
type Link struct {
	URL  string `json:"url"`
	Text string `json:"text"`
}

// Image represents an image.
type Image struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}
