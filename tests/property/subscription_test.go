package property

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"tv-tracker/internal/models"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/service"
	"tv-tracker/internal/tmdb"
)

// Feature: tv-tracker, Property 3: Subscription Data Round-Trip
// Validates: Requirements 2.2
// For any valid TMDB show ID, subscribing to the show and then retrieving it from the database
// SHALL return a TVShow with matching tmdb_id, name, status, and origin_country.
func TestSubscriptionDataRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("subscription data round-trip preserves data", prop.ForAll(
		func(tmdbID int, name string, status string, originCountry string, numSeasons int) bool {
			// Skip invalid inputs
			if tmdbID <= 0 || name == "" || numSeasons < 1 {
				return true
			}

			// Create mock TMDB server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Handle TV details request
				if r.URL.Path == "/tv/"+string(rune(tmdbID)) || true {
					response := map[string]interface{}{
						"id":                tmdbID,
						"name":              name,
						"status":            status,
						"poster_path":       "/test.jpg",
						"origin_country":    []string{originCountry},
						"number_of_seasons": numSeasons,
					}
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
					return
				}
			}))
			defer server.Close()

			// Create temporary database
			dbPath := "test_subscription_roundtrip.db"
			defer os.Remove(dbPath)

			db, err := repository.NewSQLiteDB(dbPath)
			if err != nil {
				t.Logf("Failed to create database: %v", err)
				return false
			}
			defer db.Close()

			if err := db.InitSchema(); err != nil {
				t.Logf("Failed to init schema: %v", err)
				return false
			}

			// Create repositories and services
			showRepo := repository.NewTVShowRepository(db)
			episodeRepo := repository.NewEpisodeRepository(db)
			tmdbClient := tmdb.NewClient("test-api-key")
			tmdbClient.SetBaseURL(server.URL)

			subManager := service.NewSubscriptionManager(tmdbClient, showRepo, episodeRepo)

			// Subscribe to the show
			show, err := subManager.Subscribe(tmdbID)
			if err != nil {
				t.Logf("Failed to subscribe: %v", err)
				return false
			}

			if show == nil {
				t.Log("Subscribed show is nil")
				return false
			}

			// Retrieve from database
			retrieved, err := showRepo.GetByTMDBID(tmdbID)
			if err != nil {
				t.Logf("Failed to retrieve show: %v", err)
				return false
			}

			if retrieved == nil {
				t.Log("Retrieved show is nil")
				return false
			}

			// Verify round-trip preserves data
			return retrieved.TMDBID == tmdbID &&
				retrieved.Name == name &&
				retrieved.Status == status &&
				retrieved.OriginCountry == originCountry
		},
		gen.IntRange(1, 1000000),                                            // tmdbID
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }), // name
		gen.OneConstOf("Returning Series", "Ended", "Canceled"),             // status
		gen.OneConstOf("US", "UK", "CA", "CN", "TW", "JP", "KR"),             // originCountry
		gen.IntRange(1, 20),                                                 // numSeasons
	))

	properties.TestingRun(t)
}

// Feature: tv-tracker, Property 4: Subscription Idempotence
// Validates: Requirements 2.3
// For any TMDB show ID, subscribing multiple times SHALL result in exactly one TVShow record in the database.
func TestSubscriptionIdempotence(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("subscribing multiple times creates exactly one record", prop.ForAll(
		func(tmdbID int, name string, subscribeCount int) bool {
			// Skip invalid inputs
			if tmdbID <= 0 || name == "" || subscribeCount < 1 || subscribeCount > 10 {
				return true
			}

			// Create mock TMDB server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := map[string]interface{}{
					"id":                tmdbID,
					"name":              name,
					"status":            "Returning Series",
					"poster_path":       "/test.jpg",
					"origin_country":    []string{"US"},
					"number_of_seasons": 1,
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create temporary database
			dbPath := "test_subscription_idempotence.db"
			defer os.Remove(dbPath)

			db, err := repository.NewSQLiteDB(dbPath)
			if err != nil {
				t.Logf("Failed to create database: %v", err)
				return false
			}
			defer db.Close()

			if err := db.InitSchema(); err != nil {
				t.Logf("Failed to init schema: %v", err)
				return false
			}

			// Create repositories and services
			showRepo := repository.NewTVShowRepository(db)
			episodeRepo := repository.NewEpisodeRepository(db)
			tmdbClient := tmdb.NewClient("test-api-key")
			tmdbClient.SetBaseURL(server.URL)

			subManager := service.NewSubscriptionManager(tmdbClient, showRepo, episodeRepo)

			// Subscribe multiple times
			var firstShow *models.TVShow
			for i := 0; i < subscribeCount; i++ {
				show, err := subManager.Subscribe(tmdbID)
				if err != nil {
					t.Logf("Failed to subscribe (attempt %d): %v", i+1, err)
					return false
				}
				if i == 0 {
					firstShow = show
				}
			}

			// Get all subscriptions
			allShows, err := subManager.GetAllSubscriptions()
			if err != nil {
				t.Logf("Failed to get all subscriptions: %v", err)
				return false
			}

			// Count shows with this TMDB ID
			count := 0
			for _, show := range allShows {
				if show.TMDBID == tmdbID {
					count++
				}
			}

			// Should have exactly one record
			if count != 1 {
				t.Logf("Expected 1 record, got %d", count)
				return false
			}

			// Verify IsSubscribed returns true
			if !subManager.IsSubscribed(tmdbID) {
				t.Log("IsSubscribed returned false after subscribing")
				return false
			}

			// Verify the ID is consistent
			retrieved, _ := showRepo.GetByTMDBID(tmdbID)
			if retrieved == nil || retrieved.ID != firstShow.ID {
				t.Log("Show ID changed after multiple subscriptions")
				return false
			}

			return true
		},
		gen.IntRange(1, 1000000),                                            // tmdbID
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }), // name
		gen.IntRange(1, 5),                                                  // subscribeCount
	))

	properties.TestingRun(t)
}

// Feature: tv-tracker, Property 18: Resource Time Inference
// Validates: Requirements 10.1, 10.2, 10.3, 10.4
// For any origin country code, InferResourceTime SHALL return:
// - "18:00" for US, UK, CA, GB
// - "20:00" for CN, TW
// - "23:00" for JP, KR
// - "待定" for all other countries
func TestResourceTimeInference(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Test US/UK/CA/GB countries return "18:00"
	properties.Property("US/UK/CA/GB countries return 18:00", prop.ForAll(
		func(country string) bool {
			result := service.InferResourceTime(country)
			return result == "18:00"
		},
		gen.OneConstOf("US", "UK", "CA", "GB", "us", "uk", "ca", "gb", " US ", " UK "),
	))

	// Test CN/TW countries return "20:00"
	properties.Property("CN/TW countries return 20:00", prop.ForAll(
		func(country string) bool {
			result := service.InferResourceTime(country)
			return result == "20:00"
		},
		gen.OneConstOf("CN", "TW", "cn", "tw", " CN ", " TW "),
	))

	// Test JP/KR countries return "23:00"
	properties.Property("JP/KR countries return 23:00", prop.ForAll(
		func(country string) bool {
			result := service.InferResourceTime(country)
			return result == "23:00"
		},
		gen.OneConstOf("JP", "KR", "jp", "kr", " JP ", " KR "),
	))

	// Test other countries return "待定"
	properties.Property("other countries return 待定", prop.ForAll(
		func(country string) bool {
			result := service.InferResourceTime(country)
			return result == "待定"
		},
		gen.OneConstOf("FR", "DE", "IT", "ES", "AU", "NZ", "BR", "MX", "IN", "RU", "", "XX", "UNKNOWN"),
	))

	properties.TestingRun(t)
}
