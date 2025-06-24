package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/anboat/strato-sdk/adapters/search"
)

// Constants for the Twitter adapter.
const (
	TwitterAPIBaseURL = "https://api.twitter.com/2"
	DefaultTimeout    = 30 * time.Second
)

// Client is a client for the Twitter API.
type Client struct {
	bearerToken string
	httpClient  *http.Client
	baseURL     string
}

// ClientConfig holds the configuration for the Twitter client.
type ClientConfig struct {
	BearerToken string        `json:"bearer_token"`
	BaseURL     string        `json:"base_url,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`
	HTTPClient  *http.Client  `json:"-"`
}

// TwitterSearchResponse represents the main structure of the Twitter API search response.
type TwitterSearchResponse struct {
	Data     []TweetObject           `json:"data"`
	Includes map[string][]UserObject `json:"includes"`
	Meta     MetaObject              `json:"meta"`
}

// TweetObject represents a single tweet.
type TweetObject struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}

// UserObject represents a Twitter user.
type UserObject struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

// MetaObject contains metadata about the search results.
type MetaObject struct {
	NewestID    string `json:"newest_id"`
	OldestID    string `json:"oldest_id"`
	ResultCount int    `json:"result_count"`
	NextToken   string `json:"next_token"`
}

// NewClient creates a new Twitter client.
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = &ClientConfig{}
	}

	if config.BaseURL == "" {
		config.BaseURL = TwitterAPIBaseURL
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
		bearerToken: config.BearerToken,
		httpClient:  config.HTTPClient,
		baseURL:     config.BaseURL,
	}
}

// Search implements the search.SearchAdapter interface for Twitter.
func (c *Client) Search(ctx context.Context, request *search.SearchRequest) (*search.SearchResponse, error) {
	req, err := c.buildRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to build twitter search request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to twitter: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read twitter response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("twitter api returned an error. status: %d, body: %s", resp.StatusCode, string(body))
	}

	return c.parseResponse(body, request.Query)
}

// buildRequest creates an HTTP request for the Twitter search API.
func (c *Client) buildRequest(ctx context.Context, request *search.SearchRequest) (*http.Request, error) {
	endpoint := fmt.Sprintf("%s/tweets/search/recent", c.baseURL)

	params := url.Values{}
	params.Set("query", request.Query)
	// You can add more parameters here like 'tweet.fields', 'expansions', etc.
	params.Set("expansions", "author_id")

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	req.Header.Set("Accept", "application/json")

	return req, nil
}

// parseResponse parses the JSON response from the Twitter API.
func (c *Client) parseResponse(body []byte, query string) (*search.SearchResponse, error) {
	var twitterResp TwitterSearchResponse
	if err := json.Unmarshal(body, &twitterResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal twitter json response: %w", err)
	}

	userMap := make(map[string]UserObject)
	if users, ok := twitterResp.Includes["users"]; ok {
		for _, user := range users {
			userMap[user.ID] = user
		}
	}

	var results []*search.SearchResultItem
	for _, tweet := range twitterResp.Data {
		user, ok := userMap[tweet.AuthorID]
		if !ok {
			// Skip tweet if author info is not found.
			continue
		}

		item := &search.SearchResultItem{
			Title:       fmt.Sprintf("Tweet from %s (@%s)", user.Name, user.Username),
			URL:         fmt.Sprintf("https://twitter.com/%s/status/%s", user.Username, tweet.ID),
			Description: tweet.Text,
			PublishDate: tweet.CreatedAt.Format(time.RFC3339),
			Rank:        len(results) + 1,
			Metadata: map[string]interface{}{
				"author_name":     user.Name,
				"author_username": user.Username,
				"tweet_id":        tweet.ID,
			},
		}
		results = append(results, item)
	}

	response := &search.SearchResponse{
		Query:      query,
		Results:    results,
		TotalCount: twitterResp.Meta.ResultCount,
	}

	return response, nil
}
