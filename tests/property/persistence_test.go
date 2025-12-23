package property

import (
	"os"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"tv-tracker/internal/models"
	"tv-tracker/internal/repository"
)

// Feature: tv-tracker, Property 16: TVShow Persistence Round-Trip
// Validates: Requirements 8.1, 8.4
// For any valid TVShow object, saving to the database and reading back SHALL produce an equivalent object.
func TestTVShowPersistenceRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("TVShow persistence round-trip preserves data", prop.ForAll(
		func(tmdbID int, name string, totalSeasons int, status string, originCountry string, resourceTime string, isArchived bool) bool {
			// Skip invalid inputs
			if tmdbID <= 0 || name == "" || totalSeasons < 0 {
				return true // Skip this test case
			}

			// Create a temporary database for each test
			dbPath := "test_roundtrip.db"
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

			repo := repository.NewTVShowRepository(db)

			// Create original TVShow
			original := &models.TVShow{
				TMDBID:        tmdbID,
				Name:          name,
				TotalSeasons:  totalSeasons,
				Status:        status,
				OriginCountry: originCountry,
				ResourceTime:  resourceTime,
				IsArchived:    isArchived,
			}

			// Save to database
			if err := repo.Create(original); err != nil {
				t.Logf("Failed to create TVShow: %v", err)
				return false
			}

			// Read back from database
			retrieved, err := repo.GetByTMDBID(tmdbID)
			if err != nil {
				t.Logf("Failed to retrieve TVShow: %v", err)
				return false
			}

			if retrieved == nil {
				t.Log("Retrieved TVShow is nil")
				return false
			}

			// Verify round-trip preserves data
			return retrieved.TMDBID == original.TMDBID &&
				retrieved.Name == original.Name &&
				retrieved.TotalSeasons == original.TotalSeasons &&
				retrieved.Status == original.Status &&
				retrieved.OriginCountry == original.OriginCountry &&
				retrieved.ResourceTime == original.ResourceTime &&
				retrieved.IsArchived == original.IsArchived &&
				retrieved.ID == original.ID
		},
		gen.IntRange(1, 1000000),                                                  // tmdbID
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),       // name
		gen.IntRange(0, 50),                                                       // totalSeasons
		gen.OneConstOf("Returning Series", "Ended", "Canceled", "Unknown"),        // status
		gen.OneConstOf("US", "UK", "CA", "CN", "TW", "JP", "KR", ""),               // originCountry
		gen.OneConstOf("18:00", "20:00", "23:00", "待定"),                           // resourceTime
		gen.Bool(),                                                                // isArchived
	))

	properties.TestingRun(t)
}


// Feature: tv-tracker, Property 17: Task Foreign Key Integrity
// Validates: Requirements 8.2
// For any Task in the database, its tv_show_id SHALL reference an existing TVShow record.
func TestTaskForeignKeyIntegrity(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Task tv_show_id references existing TVShow", prop.ForAll(
		func(tmdbID int, showName string, taskType string, description string) bool {
			// Skip invalid inputs
			if tmdbID <= 0 || showName == "" || description == "" {
				return true
			}

			// Create a temporary database for each test
			dbPath := "test_fk_integrity.db"
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

			showRepo := repository.NewTVShowRepository(db)
			taskRepo := repository.NewTaskRepository(db)

			// Create a TVShow first
			show := &models.TVShow{
				TMDBID:        tmdbID,
				Name:          showName,
				TotalSeasons:  1,
				Status:        "Returning Series",
				OriginCountry: "US",
				ResourceTime:  "18:00",
				IsArchived:    false,
			}

			if err := showRepo.Create(show); err != nil {
				t.Logf("Failed to create TVShow: %v", err)
				return false
			}

			// Create a Task referencing the TVShow
			var tt models.TaskType
			if taskType == "UPDATE" {
				tt = models.TaskTypeUpdate
			} else {
				tt = models.TaskTypeOrganize
			}

			task := &models.Task{
				TVShowID:    show.ID,
				TaskType:    tt,
				Description: description,
				IsCompleted: false,
			}

			if err := taskRepo.Create(task); err != nil {
				t.Logf("Failed to create Task: %v", err)
				return false
			}

			// Verify the task's tv_show_id references an existing TVShow
			retrievedTask, err := taskRepo.GetByID(task.ID)
			if err != nil {
				t.Logf("Failed to retrieve Task: %v", err)
				return false
			}

			if retrievedTask == nil {
				t.Log("Retrieved Task is nil")
				return false
			}

			// Verify the referenced TVShow exists
			referencedShow, err := showRepo.GetByID(retrievedTask.TVShowID)
			if err != nil {
				t.Logf("Failed to retrieve referenced TVShow: %v", err)
				return false
			}

			if referencedShow == nil {
				t.Log("Referenced TVShow does not exist - foreign key integrity violated")
				return false
			}

			// Verify the TVShowName is populated correctly from the join
			return retrievedTask.TVShowName == referencedShow.Name &&
				retrievedTask.TVShowID == show.ID
		},
		gen.IntRange(1, 1000000),                                            // tmdbID
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }), // showName
		gen.OneConstOf("UPDATE", "ORGANIZE"),                                // taskType
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }), // description
	))

	properties.TestingRun(t)
}
