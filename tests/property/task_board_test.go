package property

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"tv-tracker/internal/models"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/service"
	"tv-tracker/internal/timeutil"
)

// Feature: tv-tracker, Property 16: Episode ID partial match does not dedupe
// Validates: A shorter episode id (e.g. S01E01) must not match longer ones (e.g. S01E10).
func TestEpisodeIDDoesNotPartiallyMatch(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50

	properties := gopter.NewProperties(parameters)

	properties.Property("SxxEyy does not match SxxEyy0", prop.ForAll(
		func(tmdbID int, showName string, season int) bool {
			if tmdbID <= 0 || showName == "" || season < 1 {
				return true
			}

			dbPath := fmt.Sprintf("test_episode_match_%d.db", tmdbID)
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
				t.Logf("Failed to create show: %v", err)
				return false
			}

			episodeShort := service.FormatEpisodeID(season, 1)
			episodeLong := service.FormatEpisodeID(season, 10)

			// Insert a legacy style task that includes only the long episode token.
			longTask := &models.Task{
				TVShowID:    show.ID,
				TaskType:    models.TaskTypeUpdate,
				Description: fmt.Sprintf("新剧集 %s 已更新", episodeLong),
				IsCompleted: false,
			}
			if err := taskRepo.Create(longTask); err != nil {
				t.Logf("Failed to create task: %v", err)
				return false
			}

			got, err := taskRepo.GetByShowAndEpisode(show.ID, episodeShort)
			if err != nil {
				t.Logf("GetByShowAndEpisode error: %v", err)
				return false
			}
			if got != nil {
				t.Logf("Short episode %s must not match task for %s", episodeShort, episodeLong)
				return false
			}

			return true
		},
		gen.IntRange(1, 1000),
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t)
}

// Feature: tv-tracker, Property 13: UPDATE_Task Completion
// Validates: Requirements 6.1
// For any UPDATE_Task, marking it complete SHALL set is_completed to True
// and SHALL NOT modify the associated TVShow.is_archived.
func TestUpdateTaskCompletion(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("completing UPDATE_Task sets is_completed but does not archive show", prop.ForAll(
		func(tmdbID int, showName string, season, episode int) bool {
			// Skip invalid inputs
			if tmdbID <= 0 || showName == "" || season < 1 || episode < 1 {
				return true
			}

			// Create temporary database
			dbPath := fmt.Sprintf("test_update_completion_%d.db", tmdbID)
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
			taskRepo := repository.NewTaskRepository(db)

			// Create a show (not archived)
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

			// Record original archived state
			originalIsArchived := show.IsArchived

			// Create an UPDATE_Task
			episodeID := service.FormatEpisodeID(season, episode)
			task := &models.Task{
				TVShowID:    show.ID,
				TaskType:    models.TaskTypeUpdate,
				Description: fmt.Sprintf("新剧集 %s 已更新", episodeID),
				IsCompleted: false,
			}
			if err := taskRepo.Create(task); err != nil {
				t.Logf("Failed to create task: %v", err)
				return false
			}

			// Create TaskBoardService and complete the task
			taskBoard := service.NewTaskBoardService(taskRepo, showRepo)
			if err := taskBoard.CompleteTask(task.ID); err != nil {
				t.Logf("Failed to complete task: %v", err)
				return false
			}

			// Verify task is completed
			completedTask, err := taskRepo.GetByID(task.ID)
			if err != nil {
				t.Logf("Failed to get task: %v", err)
				return false
			}
			if completedTask == nil {
				t.Logf("Task not found after completion")
				return false
			}
			if !completedTask.IsCompleted {
				t.Logf("Task should be completed but is_completed is false")
				return false
			}

			// Verify show is NOT archived (UPDATE_Task should not archive)
			updatedShow, err := showRepo.GetByTMDBID(tmdbID)
			if err != nil {
				t.Logf("Failed to get show: %v", err)
				return false
			}
			if updatedShow == nil {
				t.Logf("Show not found after task completion")
				return false
			}
			if updatedShow.IsArchived != originalIsArchived {
				t.Logf("Show archived state changed: expected %v, got %v", originalIsArchived, updatedShow.IsArchived)
				return false
			}

			return true
		},
		gen.IntRange(1, 1000), // tmdbID
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }), // showName
		gen.IntRange(1, 10), // season
		gen.IntRange(1, 24), // episode
	))

	properties.TestingRun(t)
}

// Feature: tv-tracker, Property 14: ORGANIZE_Task Completion Cascades to Archive
// Validates: Requirements 6.2
// For any ORGANIZE_Task, marking it complete SHALL set is_completed to True
// AND set the associated TVShow.is_archived to True.
func TestOrganizeTaskCompletionCascadesToArchive(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("completing ORGANIZE_Task sets is_completed and archives show", prop.ForAll(
		func(tmdbID int, showName string, status string) bool {
			// Skip invalid inputs
			if tmdbID <= 0 || showName == "" {
				return true
			}

			// Create temporary database
			dbPath := fmt.Sprintf("test_organize_completion_%d.db", tmdbID)
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
			taskRepo := repository.NewTaskRepository(db)

			// Create a show (not archived, but ended/canceled)
			show := &models.TVShow{
				TMDBID:        tmdbID,
				Name:          showName,
				TotalSeasons:  1,
				Status:        status, // "Ended" or "Canceled"
				OriginCountry: "US",
				ResourceTime:  "18:00",
				IsArchived:    false, // Initially not archived
			}
			if err := showRepo.Create(show); err != nil {
				t.Logf("Failed to create show: %v", err)
				return false
			}

			// Create an ORGANIZE_Task
			task := &models.Task{
				TVShowID:    show.ID,
				TaskType:    models.TaskTypeOrganize,
				Description: fmt.Sprintf("《%s》已完结，请整理归档", showName),
				IsCompleted: false,
			}
			if err := taskRepo.Create(task); err != nil {
				t.Logf("Failed to create task: %v", err)
				return false
			}

			// Create TaskBoardService and complete the task
			taskBoard := service.NewTaskBoardService(taskRepo, showRepo)
			if err := taskBoard.CompleteTask(task.ID); err != nil {
				t.Logf("Failed to complete task: %v", err)
				return false
			}

			// Verify task is completed
			completedTask, err := taskRepo.GetByID(task.ID)
			if err != nil {
				t.Logf("Failed to get task: %v", err)
				return false
			}
			if completedTask == nil {
				t.Logf("Task not found after completion")
				return false
			}
			if !completedTask.IsCompleted {
				t.Logf("Task should be completed but is_completed is false")
				return false
			}

			// Verify show IS archived (ORGANIZE_Task should archive the show)
			updatedShow, err := showRepo.GetByTMDBID(tmdbID)
			if err != nil {
				t.Logf("Failed to get show: %v", err)
				return false
			}
			if updatedShow == nil {
				t.Logf("Show not found after task completion")
				return false
			}
			if !updatedShow.IsArchived {
				t.Logf("Show should be archived after ORGANIZE_Task completion but is_archived is false")
				return false
			}

			return true
		},
		gen.IntRange(1, 1000), // tmdbID
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }), // showName
		gen.OneConstOf("Ended", "Canceled"),                                 // status
	))

	properties.TestingRun(t)
}

// Feature: tv-tracker, Property 15: UPDATE_Task Postpone is atomic
// Validates: Requirement 6.3 (postpone moves task to tomorrow)
// Postponing a task SHALL result in exactly one pending task remaining for the show/type,
// even if the operation involves multiple DB steps.
func TestPostponeTaskIsAtomic(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50

	properties := gopter.NewProperties(parameters)

	properties.Property("postponing task leaves exactly one pending task", prop.ForAll(
		func(tmdbID int, showName string, season, episode int) bool {
			if tmdbID <= 0 || showName == "" || season < 1 || episode < 1 {
				return true
			}

			now := time.Date(2026, 1, 9, 10, 0, 0, 0, time.FixedZone("CST", 8*3600))
			timeutil.SetNowFunc(func() time.Time { return now })
			defer timeutil.SetNowFunc(nil)

			dbPath := fmt.Sprintf("test_postpone_atomic_%d.db", tmdbID)
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

			episodeID := service.FormatEpisodeID(season, episode)
			task := &models.Task{
				TVShowID:    show.ID,
				TaskType:    models.TaskTypeUpdate,
				Description: fmt.Sprintf("%s|新剧集更新: %s", episodeID, episodeID),
				IsCompleted: false,
			}
			if err := taskRepo.Create(task); err != nil {
				t.Logf("Failed to create task: %v", err)
				return false
			}

			taskBoard := service.NewTaskBoardService(taskRepo, showRepo)
			if err := taskBoard.PostponeTask(task.ID); err != nil {
				t.Logf("Failed to postpone task: %v", err)
				return false
			}

			pending, err := taskRepo.GetPendingByType(models.TaskTypeUpdate)
			if err != nil {
				t.Logf("Failed to list pending tasks: %v", err)
				return false
			}
			count := 0
			for _, p := range pending {
				if p.TVShowID == show.ID {
					count++
				}
			}
			if count != 1 {
				t.Logf("Expected exactly 1 pending task for show after postpone, got %d", count)
				return false
			}

			return true
		},
		gen.IntRange(1, 1000),
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(1, 10),
		gen.IntRange(1, 24),
	))

	properties.TestingRun(t)
}
