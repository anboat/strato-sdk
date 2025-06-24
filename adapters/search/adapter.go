package search

import (
	"context"
)

// SearchEngine represents a search engine identifier.
type SearchEngine string

// String implements the fmt.Stringer interface.
func (e SearchEngine) String() string {
	return string(e)
}

// SearchAdapter is the interface for a search adapter.
type SearchAdapter interface {
	// Search is the core search function.
	Search(ctx context.Context, request *SearchRequest) (*SearchResponse, error)
}

// SearchRequest represents a search request.
type SearchRequest struct {
	// Basic parameters
	Query string `json:"query"`
	Num   int    `json:"num,omitempty"`

	// Common options
	Offset     int    `json:"offset,omitempty"`
	Lang       string `json:"lang,omitempty"`
	Region     string `json:"region,omitempty"`
	SafeSearch string `json:"safe_search,omitempty"`
	TimeRange  string `json:"time_range,omitempty"`

	// Filtering options
	Site     string `json:"site,omitempty"`
	FileType string `json:"file_type,omitempty"`

	// Engine-specific parameters
	EngineParams map[string]interface{} `json:"engine_params,omitempty"`
}

// SearchResponse represents a search response.
type SearchResponse struct {
	Query      string              `json:"query"`
	Results    []*SearchResultItem `json:"results"`
	TotalCount int                 `json:"total_count,omitempty"`
	TimeTaken  int64               `json:"time_taken_ms,omitempty"`
}

// SearchResultItem represents a single search result item.
type SearchResultItem struct {
	// Core information
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`

	// Compatibility fields
	Link    string `json:"link,omitempty"`
	Snippet string `json:"snippet,omitempty"`

	// Extended information
	Rank        int                    `json:"rank,omitempty"`
	Score       float64                `json:"score,omitempty"`
	PublishDate string                 `json:"publish_date,omitempty"`
	FileType    string                 `json:"file_type,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
