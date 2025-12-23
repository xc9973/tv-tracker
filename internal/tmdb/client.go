package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultBaseURL    = "https://api.themoviedb.org/3"
	defaultTimeout    = 10 * time.Second
	requestInterval   = 100 * time.Millisecond // 请求间隔，避免触发限流
)

// Client handles all interactions with the TMDB API
type Client struct {
	apiKey      string
	baseURL     string
	httpClient  *http.Client
	lastRequest time.Time
}

// SearchResult represents a single TV show from search results
type SearchResult struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	PosterPath    string   `json:"poster_path"`
	FirstAirDate  string   `json:"first_air_date"`
	OriginCountry []string `json:"origin_country"`
}

// EpisodeInfo represents episode information from TMDB
type EpisodeInfo struct {
	AirDate       string `json:"air_date"`
	EpisodeNumber int    `json:"episode_number"`
	SeasonNumber  int    `json:"season_number"`
	Name          string `json:"name"`
	Overview      string `json:"overview"`
}

// TVDetails represents detailed TV show information
type TVDetails struct {
	ID               int          `json:"id"`
	Name             string       `json:"name"`
	Status           string       `json:"status"`
	PosterPath       string       `json:"poster_path"`
	OriginCountry    []string     `json:"origin_country"`
	NumberOfSeasons  int          `json:"number_of_seasons"`
	NextEpisodeToAir *EpisodeInfo `json:"next_episode_to_air"`
	LastEpisodeToAir *EpisodeInfo `json:"last_episode_to_air"`
}

// SeasonDetail represents season information with episodes
type SeasonDetail struct {
	Episodes []EpisodeInfo `json:"episodes"`
}


// searchResponse wraps the TMDB search API response
type searchResponse struct {
	Results []SearchResult `json:"results"`
}

// APIError represents an error returned by the TMDB API
type APIError struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("TMDB API error (code %d): %s", e.StatusCode, e.StatusMessage)
}

// NewClient creates a new TMDB API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// NewClientWithHTTP creates a new TMDB API client with a custom HTTP client
func NewClientWithHTTP(apiKey string, httpClient *http.Client) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: httpClient,
	}
}

// SetBaseURL allows overriding the base URL (useful for testing)
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// SearchTV searches for TV shows by query string
// Calls TMDB /search/tv API with Chinese language
func (c *Client) SearchTV(query string) ([]SearchResult, error) {
	if query == "" {
		return []SearchResult{}, nil
	}

	c.rateLimit() // 限流

	endpoint := fmt.Sprintf("%s/search/tv?api_key=%s&query=%s&language=zh-CN",
		c.baseURL, c.apiKey, url.QueryEscape(query))

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to search TV shows: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return result.Results, nil
}


// GetTVDetails fetches detailed information for a TV show
// Calls TMDB /tv/{id} API with Chinese language
func (c *Client) GetTVDetails(tmdbID int) (*TVDetails, error) {
	if tmdbID <= 0 {
		return nil, fmt.Errorf("invalid TMDB ID: %d", tmdbID)
	}

	c.rateLimit() // 限流

	endpoint := fmt.Sprintf("%s/tv/%d?api_key=%s&language=zh-CN", c.baseURL, tmdbID, c.apiKey)

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get TV details: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var details TVDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("failed to decode TV details response: %w", err)
	}

	return &details, nil
}

// GetSeasonEpisodes fetches all episodes for a specific season
// Calls TMDB /tv/{id}/season/{season} API with Chinese language
func (c *Client) GetSeasonEpisodes(tmdbID, seasonNumber int) ([]EpisodeInfo, error) {
	if tmdbID <= 0 {
		return nil, fmt.Errorf("invalid TMDB ID: %d", tmdbID)
	}
	if seasonNumber < 0 {
		return nil, fmt.Errorf("invalid season number: %d", seasonNumber)
	}

	c.rateLimit() // 限流

	endpoint := fmt.Sprintf("%s/tv/%d/season/%d?api_key=%s&language=zh-CN",
		c.baseURL, tmdbID, seasonNumber, c.apiKey)

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get season episodes: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var season SeasonDetail
	if err := json.NewDecoder(resp.Body).Decode(&season); err != nil {
		return nil, fmt.Errorf("failed to decode season response: %w", err)
	}

	return season.Episodes, nil
}

// checkResponse checks the HTTP response for errors
func (c *Client) checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode:    resp.StatusCode,
			StatusMessage: fmt.Sprintf("HTTP %d: failed to read error response", resp.StatusCode),
		}
	}

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return &APIError{
			StatusCode:    resp.StatusCode,
			StatusMessage: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		}
	}

	if apiErr.StatusCode == 0 {
		apiErr.StatusCode = resp.StatusCode
	}
	if apiErr.StatusMessage == "" {
		apiErr.StatusMessage = fmt.Sprintf("HTTP %d error", resp.StatusCode)
	}

	return &apiErr
}

// rateLimit ensures requests are spaced out to avoid hitting API limits
func (c *Client) rateLimit() {
	elapsed := time.Since(c.lastRequest)
	if elapsed < requestInterval {
		time.Sleep(requestInterval - elapsed)
	}
	c.lastRequest = time.Now()
}
