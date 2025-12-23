package property

import (
	"fmt"
	"os"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"tv-tracker/internal/models"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/service"
)

// Feature: tv-tracker, Property 15: Task Rendering Completeness
// Validates: Requirements 7.3
// For any Task with an associated TVShow, the rendered task view SHALL include
// the show name and task description.
func TestTaskRenderingCompleteness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("task rendering includes show name and description", prop.ForAll(
		func(tmdbID int, showName string, description string, taskType string) bool {
			// Skip invalid inputs
			if tmdbID <= 0 || showName == "" || description == "" {
				return true
			}

			// Create temporary database
			dbPath := fmt.Sprintf("test_task_rendering_%d.db", tmdbID)
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

			// Create a show
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

			// Determine task type
			var tt models.TaskType
			if taskType == "UPDATE" {
				tt = models.TaskTypeUpdate
			} else {
				tt = models.TaskTypeOrganize
			}

			// Create a task
			task := &models.Task{
				TVShowID:    show.ID,
				TaskType:    tt,
				Description: description,
				IsCompleted: false,
			}
			if err := taskRepo.Create(task); err != nil {
				t.Logf("Failed to create task: %v", err)
				return false
			}

			// Create TaskBoardService and get dashboard data
			taskBoard := service.NewTaskBoardService(taskRepo, showRepo)
			dashboardData, err := taskBoard.GetDashboardData()
			if err != nil {
				t.Logf("Failed to get dashboard data: %v", err)
				return false
			}

			// Find the task in the dashboard data
			var foundTask *models.Task
			if tt == models.TaskTypeUpdate {
				for i := range dashboardData.UpdateTasks {
					if dashboardData.UpdateTasks[i].ID == task.ID {
						foundTask = &dashboardData.UpdateTasks[i]
						break
					}
				}
			} else {
				for i := range dashboardData.OrganizeTasks {
					if dashboardData.OrganizeTasks[i].ID == task.ID {
						foundTask = &dashboardData.OrganizeTasks[i]
						break
					}
				}
			}

			if foundTask == nil {
				t.Logf("Task not found in dashboard data")
				return false
			}

			// Verify the task includes show name (Requirement 7.3)
			if foundTask.TVShowName != showName {
				t.Logf("Task TVShowName mismatch: expected %q, got %q", showName, foundTask.TVShowName)
				return false
			}

			// Verify the task includes description (Requirement 7.3)
			if foundTask.Description != description {
				t.Logf("Task Description mismatch: expected %q, got %q", description, foundTask.Description)
				return false
			}

			return true
		},
		gen.IntRange(1, 1000),                                                     // tmdbID
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),       // showName
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),       // description
		gen.OneConstOf("UPDATE", "ORGANIZE"),                                      // taskType
	))

	properties.TestingRun(t)
}
