package property

import (
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"tv-tracker/internal/models"
	"tv-tracker/internal/notify"
	"tv-tracker/internal/service"
)

// Feature: tv-tracker, Property 19: Daily Report Contains All Today's Episodes
// Validates: Requirements 9.1, 9.2
// For any set of episodes where air_date equals today, the daily report SHALL include
// all of them with show name, episode info, and resource time.
func TestDailyReportContainsAllTodaysEpisodes(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("daily report contains all today's episodes with required info", prop.ForAll(
		func(tasks []testTask) bool {
			// Convert test tasks to models.Task
			var modelTasks []models.Task
			for _, tt := range tasks {
				episodeID := service.FormatEpisodeID(tt.Season, tt.Episode)
				task := models.Task{
					ID:           tt.ID,
					TVShowID:     tt.ShowID,
					TVShowName:   tt.ShowName,
					ResourceTime: tt.ResourceTime,
					TaskType:     models.TaskTypeUpdate,
					Description:  "新剧集更新: " + episodeID + " - " + tt.EpisodeName,
					IsCompleted:  false,
					CreatedAt:    time.Now(),
				}
				modelTasks = append(modelTasks, task)
			}

			// Generate the daily report
			report := notify.FormatDailyReport(modelTasks)

			// If no tasks, report should indicate no updates
			if len(tasks) == 0 {
				return strings.Contains(report, "今日暂无剧集更新")
			}

			// Verify each task's info is in the report
			for _, tt := range tasks {
				// Requirement 9.2: show name must be present
				if !strings.Contains(report, tt.ShowName) {
					t.Logf("Report missing show name: %s", tt.ShowName)
					return false
				}

				// Requirement 9.2: episode info (SxxExx format) must be present
				episodeID := service.FormatEpisodeID(tt.Season, tt.Episode)
				if !strings.Contains(report, episodeID) {
					t.Logf("Report missing episode ID: %s", episodeID)
					return false
				}

				// Requirement 9.2: resource time must be present
				if !strings.Contains(report, tt.ResourceTime) {
					t.Logf("Report missing resource time: %s", tt.ResourceTime)
					return false
				}
			}

			return true
		},
		genTestTasks(),
	))

	properties.TestingRun(t)
}

// testTask is a helper struct for generating test tasks
type testTask struct {
	ID           int64
	ShowID       int64
	ShowName     string
	Season       int
	Episode      int
	EpisodeName  string
	ResourceTime string
}

// genTestTasks generates a slice of test tasks
func genTestTasks() gopter.Gen {
	return gen.SliceOfN(5, genTestTask())
}

// genTestTask generates a single test task
func genTestTask() gopter.Gen {
	return gopter.CombineGens(
		gen.Int64Range(1, 1000),                                                 // ID
		gen.Int64Range(1, 1000),                                                 // ShowID
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),     // ShowName
		gen.IntRange(1, 20),                                                     // Season
		gen.IntRange(1, 30),                                                     // Episode
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),     // EpisodeName
		gen.OneConstOf("18:00", "20:00", "23:00", "待定"),                         // ResourceTime
	).Map(func(values []interface{}) testTask {
		return testTask{
			ID:           values[0].(int64),
			ShowID:       values[1].(int64),
			ShowName:     values[2].(string),
			Season:       values[3].(int),
			Episode:      values[4].(int),
			EpisodeName:  values[5].(string),
			ResourceTime: values[6].(string),
		}
	})
}

// TestDailyReportNoUpdatesMessage tests that when there are no updates,
// the report indicates this clearly (Requirement 9.3)
func TestDailyReportNoUpdatesMessage(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("empty task list produces no updates message", prop.ForAll(
		func(_ int) bool {
			// Generate report with empty task list
			report := notify.FormatDailyReport([]models.Task{})

			// Should contain the no updates message
			return strings.Contains(report, "今日暂无剧集更新")
		},
		gen.IntRange(1, 100), // dummy generator to run multiple times
	))

	properties.TestingRun(t)
}

// TestDailyReportOnlyIncludesUpdateTasks tests that the report only includes
// UPDATE tasks, not ORGANIZE tasks
func TestDailyReportOnlyIncludesUpdateTasks(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("daily report only includes UPDATE tasks", prop.ForAll(
		func(idx int) bool {
			// Use index to generate distinct show names
			updateShowName := "UpdateShow_" + string(rune('A'+idx%26))
			organizeShowName := "OrganizeShow_" + string(rune('Z'-idx%26))

			// Create mixed tasks
			tasks := []models.Task{
				{
					ID:           1,
					TVShowID:     1,
					TVShowName:   updateShowName,
					ResourceTime: "18:00",
					TaskType:     models.TaskTypeUpdate,
					Description:  "新剧集更新: S01E01 - Test Episode",
					IsCompleted:  false,
				},
				{
					ID:           2,
					TVShowID:     2,
					TVShowName:   organizeShowName,
					ResourceTime: "20:00",
					TaskType:     models.TaskTypeOrganize,
					Description:  "剧集已完结，请整理归档本地文件",
					IsCompleted:  false,
				},
			}

			report := notify.FormatDailyReport(tasks)

			// UPDATE task show name should be present
			if !strings.Contains(report, updateShowName) {
				t.Logf("Report missing UPDATE task show name: %s", updateShowName)
				return false
			}

			// ORGANIZE task show name should NOT be present (daily report is for updates only)
			if strings.Contains(report, organizeShowName) {
				t.Logf("Report incorrectly contains ORGANIZE task show name: %s", organizeShowName)
				return false
			}

			return true
		},
		gen.IntRange(0, 1000),
	))

	properties.TestingRun(t)
}
