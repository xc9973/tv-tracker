package property

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"tv-tracker/internal/models"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/service"
	"tv-tracker/internal/tmdb"
)

// Feature: tv-tracker, Property 9: Episode ID Format
// Validates: Requirements 4.2
// For any season number S and episode number E, the formatted episode ID SHALL be
// "S{SS}E{EE}" where SS and EE are zero-padded to 2 digits.
func TestEpisodeIDFormat(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Pattern for SxxExx format
	pattern := regexp.MustCompile(`^S\d{2}E\d{2}$`)

	properties.Property("episode ID follows SxxExx format", prop.ForAll(
		func(season, episode int) bool {
			// Skip invalid inputs (negative numbers)
			if season < 0 || episode < 0 {
				return true
			}

			result := service.FormatEpisodeID(season, episode)

			// Verify format matches SxxExx pattern
			if !pattern.MatchString(result) {
				t.Logf("Format mismatch: got %s for S%d E%d", result, season, episode)
				return false
			}

			// Extract season and episode from result
			var parsedSeason, parsedEpisode int
			_, err := fmt.Sscanf(result, "S%02dE%02d", &parsedSeason, &parsedEpisode)
			if err != nil {
				t.Logf("Failed to parse result %s: %v", result, err)
				return false
			}

			// For values 0-99, they should match exactly
			if season <= 99 && parsedSeason != season {
				t.Logf("Season mismatch: expected %d, got %d", season, parsedSeason)
				return false
			}
			if episode <= 99 && parsedEpisode != episode {
				t.Logf("Episode mismatch: expected %d, got %d", episode, parsedEpisode)
				return false
			}

			return true
		},
		gen.IntRange(0, 99),  // season (typical range)
		gen.IntRange(0, 99),  // episode (typical range)
	))

	// Test specific edge cases
	properties.Property("single digit numbers are zero-padded", prop.ForAll(
		func(season, episode int) bool {
			result := service.FormatEpisodeID(season, episode)

			// Should always be exactly 6 characters: S + 2 digits + E + 2 digits
			if len(result) != 6 {
				t.Logf("Length mismatch: expected 6, got %d for %s", len(result), result)
				return false
			}

			// First character should be 'S'
			if result[0] != 'S' {
				return false
			}

			// Fourth character should be 'E'
			if result[3] != 'E' {
				return false
			}

			return true
		},
		gen.IntRange(0, 9),  // single digit season
		gen.IntRange(0, 9),  // single digit episode
	))

	properties.TestingRun(t)
}


// Feature: tv-tracker, Property 5: Sync Processes Only Active Shows
// Validates: Requirements 3.1, 5.4, 6.3
// For any set of TVShow records, the sync operation SHALL process only those where
// is_archived = False, and SHALL skip all archived shows.
func TestSyncProcessesOnlyActiveShows(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("sync only processes non-archived shows", prop.ForAll(
		func(activeCount, archivedCount int) bool {
			// Skip invalid inputs
			if activeCount < 0 || archivedCount < 0 || activeCount+archivedCount == 0 {
				return true
			}

			// Track which shows were processed by TMDB API calls
			processedTMDBIDs := make(map[int]bool)

			// Create mock TMDB server that tracks which shows are queried
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract TMDB ID from URL path
				path := r.URL.Path
				var tmdbID int
				if _, err := fmt.Sscanf(path, "/tv/%d", &tmdbID); err == nil {
					processedTMDBIDs[tmdbID] = true
				}

				// Return a valid response
				response := map[string]interface{}{
					"id":                1,
					"name":              "Test Show",
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
			dbPath := fmt.Sprintf("test_sync_active_%d_%d.db", activeCount, archivedCount)
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

			// Create repositories
			showRepo := repository.NewTVShowRepository(db)
			episodeRepo := repository.NewEpisodeRepository(db)
			taskRepo := repository.NewTaskRepository(db)
			tmdbClient := tmdb.NewClient("test-api-key")
			tmdbClient.SetBaseURL(server.URL)

			// Create active shows (is_archived = false)
			activeTMDBIDs := make([]int, 0, activeCount)
			for i := 0; i < activeCount; i++ {
				tmdbID := 1000 + i
				show := &models.TVShow{
					TMDBID:        tmdbID,
					Name:          fmt.Sprintf("Active Show %d", i),
					TotalSeasons:  1,
					Status:        "Returning Series",
					OriginCountry: "US",
					ResourceTime:  "18:00",
					IsArchived:    false,
				}
				if err := showRepo.Create(show); err != nil {
					t.Logf("Failed to create active show: %v", err)
					return false
				}
				activeTMDBIDs = append(activeTMDBIDs, tmdbID)
			}

			// Create archived shows (is_archived = true)
			archivedTMDBIDs := make([]int, 0, archivedCount)
			for i := 0; i < archivedCount; i++ {
				tmdbID := 2000 + i
				show := &models.TVShow{
					TMDBID:        tmdbID,
					Name:          fmt.Sprintf("Archived Show %d", i),
					TotalSeasons:  1,
					Status:        "Ended",
					OriginCountry: "US",
					ResourceTime:  "18:00",
					IsArchived:    true,
				}
				if err := showRepo.Create(show); err != nil {
					t.Logf("Failed to create archived show: %v", err)
					return false
				}
				archivedTMDBIDs = append(archivedTMDBIDs, tmdbID)
			}

			// Create TaskGenerator and run sync
			taskGen := service.NewTaskGenerator(tmdbClient, showRepo, episodeRepo, taskRepo)
			_, err = taskGen.SyncAll()
			if err != nil {
				t.Logf("SyncAll failed: %v", err)
				return false
			}

			// Verify: all active shows should have been processed
			for _, tmdbID := range activeTMDBIDs {
				if !processedTMDBIDs[tmdbID] {
					t.Logf("Active show %d was not processed", tmdbID)
					return false
				}
			}

			// Verify: no archived shows should have been processed
			for _, tmdbID := range archivedTMDBIDs {
				if processedTMDBIDs[tmdbID] {
					t.Logf("Archived show %d was incorrectly processed", tmdbID)
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 5),  // activeCount
		gen.IntRange(0, 5),  // archivedCount
	))

	properties.TestingRun(t)
}


// Feature: tv-tracker, Property 10: UPDATE_Task Idempotence
// Validates: Requirements 4.3
// For any TVShow and episode combination, running sync multiple times SHALL create
// at most one UPDATE_Task.
func TestUpdateTaskIdempotence(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("multiple syncs create at most one UPDATE_Task per episode", prop.ForAll(
		func(tmdbID int, showName string, season, episode int, syncCount int) bool {
			// Skip invalid inputs
			if tmdbID <= 0 || showName == "" || season < 1 || episode < 1 || syncCount < 1 || syncCount > 5 {
				return true
			}

			// Today's date for air_date
			today := time.Now().Format("2006-01-02")

			// Create mock TMDB server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path

				// Handle TV details request
				if strings.HasPrefix(path, "/tv/") && !strings.Contains(path, "/season/") {
					response := map[string]interface{}{
						"id":                tmdbID,
						"name":              showName,
						"status":            "Returning Series",
						"poster_path":       "/test.jpg",
						"origin_country":    []string{"US"},
						"number_of_seasons": season,
						"next_episode_to_air": map[string]interface{}{
							"air_date":       today,
							"episode_number": episode,
							"season_number":  season,
							"name":           "Test Episode",
						},
					}
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
					return
				}

				// Handle season episodes request
				if strings.Contains(path, "/season/") {
					response := map[string]interface{}{
						"episodes": []map[string]interface{}{
							{
								"air_date":       today,
								"episode_number": episode,
								"season_number":  season,
								"name":           "Test Episode",
								"overview":       "Test overview",
							},
						},
					}
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			// Create temporary database
			dbPath := fmt.Sprintf("test_update_idempotence_%d.db", tmdbID)
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

			// Create repositories
			showRepo := repository.NewTVShowRepository(db)
			episodeRepo := repository.NewEpisodeRepository(db)
			taskRepo := repository.NewTaskRepository(db)
			tmdbClient := tmdb.NewClient("test-api-key")
			tmdbClient.SetBaseURL(server.URL)

			// Create a show
			show := &models.TVShow{
				TMDBID:        tmdbID,
				Name:          showName,
				TotalSeasons:  season,
				Status:        "Returning Series",
				OriginCountry: "US",
				ResourceTime:  "18:00",
				IsArchived:    false,
			}
			if err := showRepo.Create(show); err != nil {
				t.Logf("Failed to create show: %v", err)
				return false
			}

			// Create TaskGenerator
			taskGen := service.NewTaskGenerator(tmdbClient, showRepo, episodeRepo, taskRepo)

			// Run sync multiple times
			for i := 0; i < syncCount; i++ {
				_, err := taskGen.SyncAll()
				if err != nil {
					t.Logf("SyncAll failed on iteration %d: %v", i+1, err)
					return false
				}
			}

			// Count UPDATE_Tasks for this show
			episodeID := service.FormatEpisodeID(season, episode)
			updateTasks, err := taskRepo.GetPendingByType(models.TaskTypeUpdate)
			if err != nil {
				t.Logf("Failed to get pending tasks: %v", err)
				return false
			}

			// Count tasks for this specific show and episode
			count := 0
			for _, task := range updateTasks {
				if task.TVShowID == show.ID && strings.Contains(task.Description, episodeID) {
					count++
				}
			}

			// Should have at most one UPDATE_Task
			if count > 1 {
				t.Logf("Expected at most 1 UPDATE_Task, got %d", count)
				return false
			}

			return true
		},
		gen.IntRange(1, 1000),                                               // tmdbID
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }), // showName
		gen.IntRange(1, 10),                                                 // season
		gen.IntRange(1, 24),                                                 // episode
		gen.IntRange(1, 3),                                                  // syncCount
	))

	properties.TestingRun(t)
}


// Feature: tv-tracker, Property 12: ORGANIZE_Task Idempotence
// Validates: Requirements 5.3
// For any ended/canceled TVShow, running sync multiple times SHALL create
// at most one ORGANIZE_Task.
func TestOrganizeTaskIdempotence(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("multiple syncs create at most one ORGANIZE_Task per ended show", prop.ForAll(
		func(tmdbID int, showName string, status string, syncCount int) bool {
			// Skip invalid inputs
			if tmdbID <= 0 || showName == "" || syncCount < 1 || syncCount > 5 {
				return true
			}

			// Create mock TMDB server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path

				// Handle TV details request
				if strings.HasPrefix(path, "/tv/") && !strings.Contains(path, "/season/") {
					response := map[string]interface{}{
						"id":                tmdbID,
						"name":              showName,
						"status":            status, // "Ended" or "Canceled"
						"poster_path":       "/test.jpg",
						"origin_country":    []string{"US"},
						"number_of_seasons": 1,
					}
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
					return
				}

				// Handle season episodes request
				if strings.Contains(path, "/season/") {
					response := map[string]interface{}{
						"episodes": []map[string]interface{}{},
					}
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			// Create temporary database
			dbPath := fmt.Sprintf("test_organize_idempotence_%d.db", tmdbID)
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

			// Create repositories
			showRepo := repository.NewTVShowRepository(db)
			episodeRepo := repository.NewEpisodeRepository(db)
			taskRepo := repository.NewTaskRepository(db)
			tmdbClient := tmdb.NewClient("test-api-key")
			tmdbClient.SetBaseURL(server.URL)

			// Create a show (initially with "Returning Series" status, will be updated by sync)
			show := &models.TVShow{
				TMDBID:        tmdbID,
				Name:          showName,
				TotalSeasons:  1,
				Status:        "Returning Series", // Will be updated to Ended/Canceled by sync
				OriginCountry: "US",
				ResourceTime:  "18:00",
				IsArchived:    false,
			}
			if err := showRepo.Create(show); err != nil {
				t.Logf("Failed to create show: %v", err)
				return false
			}

			// Create TaskGenerator
			taskGen := service.NewTaskGenerator(tmdbClient, showRepo, episodeRepo, taskRepo)

			// Run sync multiple times
			for i := 0; i < syncCount; i++ {
				_, err := taskGen.SyncAll()
				if err != nil {
					t.Logf("SyncAll failed on iteration %d: %v", i+1, err)
					return false
				}
			}

			// Count ORGANIZE_Tasks for this show
			organizeTasks, err := taskRepo.GetPendingByType(models.TaskTypeOrganize)
			if err != nil {
				t.Logf("Failed to get pending tasks: %v", err)
				return false
			}

			// Count tasks for this specific show
			count := 0
			for _, task := range organizeTasks {
				if task.TVShowID == show.ID {
					count++
				}
			}

			// Should have at most one ORGANIZE_Task
			if count > 1 {
				t.Logf("Expected at most 1 ORGANIZE_Task, got %d", count)
				return false
			}

			// If status is Ended or Canceled, should have exactly one ORGANIZE_Task
			if (status == "Ended" || status == "Canceled") && count != 1 {
				t.Logf("Expected exactly 1 ORGANIZE_Task for %s show, got %d", status, count)
				return false
			}

			return true
		},
		gen.IntRange(1, 1000),                                               // tmdbID
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }), // showName
		gen.OneConstOf("Ended", "Canceled"),                                 // status (only ended/canceled shows)
		gen.IntRange(1, 3),                                                  // syncCount
	))

	properties.TestingRun(t)
}
