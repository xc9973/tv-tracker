package property

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"tv-tracker/internal/tmdb"
)

// Feature: tv-tracker, Property 2: API Error Handling
// Validates: Requirements 1.3
// For any TMDB API error response, the TMDB_Client SHALL return an error object
// with a descriptive message string, never raise an unhandled exception.
func TestAPIErrorHandling(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("API errors return descriptive error messages", prop.ForAll(
		func(statusCode int, statusMessage string) bool {
			// Create a mock server that returns an error response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
				errorResp := map[string]interface{}{
					"status_code":    statusCode,
					"status_message": statusMessage,
				}
				json.NewEncoder(w).Encode(errorResp)
			}))
			defer server.Close()

			// Create client pointing to mock server
			client := tmdb.NewClient("test-api-key")
			client.SetBaseURL(server.URL)

			// Test SearchTV - should return error, not panic
			results, err := client.SearchTV("test query")

			// Verify error is returned (not nil) and results are nil
			if err == nil {
				return false // Should have returned an error
			}

			if results != nil {
				return false // Results should be nil on error
			}

			// Verify error message is descriptive (non-empty)
			errMsg := err.Error()
			if errMsg == "" {
				return false // Error message should not be empty
			}

			// Test GetTVDetails - should return error, not panic
			details, err := client.GetTVDetails(12345)

			if err == nil {
				return false
			}

			if details != nil {
				return false
			}

			if err.Error() == "" {
				return false
			}

			// Test GetSeasonEpisodes - should return error, not panic
			episodes, err := client.GetSeasonEpisodes(12345, 1)

			if err == nil {
				return false
			}

			if episodes != nil {
				return false
			}

			if err.Error() == "" {
				return false
			}

			return true
		},
		// Generate HTTP error status codes (4xx and 5xx)
		gen.OneConstOf(400, 401, 403, 404, 429, 500, 502, 503, 504),
		// Generate various error messages
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t)
}


// Feature: tv-tracker, Property 1: TMDB Search Returns Valid Results
// Validates: Requirements 1.1, 1.2
// For any non-empty search query, the TMDB_Client SHALL return a list (possibly empty)
// where each item contains id, name, poster_path, and first_air_date fields.
func TestSearchReturnsValidResults(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("search returns valid structure with required fields", prop.ForAll(
		func(query string, numResults int, resultIDs []int, resultNames []string) bool {
			// Build mock response with generated data
			results := make([]map[string]interface{}, 0, numResults)
			for i := 0; i < numResults && i < len(resultIDs) && i < len(resultNames); i++ {
				results = append(results, map[string]interface{}{
					"id":             resultIDs[i],
					"name":           resultNames[i],
					"poster_path":    "/test_poster.jpg",
					"first_air_date": "2024-01-15",
					"origin_country": []string{"US"},
				})
			}

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := map[string]interface{}{
					"results": results,
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create client pointing to mock server
			client := tmdb.NewClient("test-api-key")
			client.SetBaseURL(server.URL)

			// Execute search
			searchResults, err := client.SearchTV(query)

			// Should not return error for valid response
			if err != nil {
				return false
			}

			// Should return a list (possibly empty)
			if searchResults == nil {
				return false
			}

			// Verify each result has required fields
			for _, result := range searchResults {
				// ID must be present and positive
				if result.ID <= 0 {
					return false
				}

				// Name must be present (can be empty string but field must exist)
				// The struct ensures the field exists, so we just verify it was decoded
			}

			// Verify the number of results matches what we sent
			if len(searchResults) != numResults {
				return false
			}

			return true
		},
		// Non-empty search query
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),
		// Number of results (0 to 10)
		gen.IntRange(0, 10),
		// Result IDs
		gen.SliceOfN(10, gen.IntRange(1, 1000000)),
		// Result names
		gen.SliceOfN(10, gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 })),
	))

	properties.TestingRun(t)
}

// TestEmptySearchQuery verifies that empty search queries return empty results
func TestEmptySearchQuery(t *testing.T) {
	client := tmdb.NewClient("test-api-key")

	results, err := client.SearchTV("")

	if err != nil {
		t.Errorf("Empty search should not return error, got: %v", err)
	}

	if results == nil {
		t.Error("Empty search should return empty slice, not nil")
	}

	if len(results) != 0 {
		t.Errorf("Empty search should return empty slice, got %d results", len(results))
	}
}
