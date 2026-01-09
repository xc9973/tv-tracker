package service

import (
	"fmt"
	"time"

	"tv-tracker/internal/models"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/timeutil"
	"tv-tracker/internal/tmdb"
)

// SyncResult contains the results of a sync operation
type SyncResult struct {
	UpdateTasks   int `json:"update_tasks"`
	OrganizeTasks int `json:"organize_tasks"`
	Errors        int `json:"errors"`
}

// TaskGenerator handles task generation based on show status and episode updates
type TaskGenerator struct {
	tmdbClient  *tmdb.Client
	cacheSvc    *TMDBCacheService
	showRepo    *repository.TVShowRepository
	episodeRepo *repository.EpisodeRepository
	taskRepo    *repository.TaskRepository
}

// NewTaskGenerator creates a new TaskGenerator
func NewTaskGenerator(
	tmdbClient *tmdb.Client,
	cacheSvc *TMDBCacheService,
	showRepo *repository.TVShowRepository,
	episodeRepo *repository.EpisodeRepository,
	taskRepo *repository.TaskRepository,
) *TaskGenerator {
	return &TaskGenerator{
		tmdbClient:  tmdbClient,
		cacheSvc:    cacheSvc,
		showRepo:    showRepo,
		episodeRepo: episodeRepo,
		taskRepo:    taskRepo,
	}
}

// FormatEpisodeID formats season and episode numbers into SxxExx format
// e.g., season 1, episode 5 -> "S01E05"
func FormatEpisodeID(season, episode int) string {
	return fmt.Sprintf("S%02dE%02d", season, episode)
}

// SyncAll iterates through all non-archived subscriptions, syncs latest season episodes,
// and generates tasks as needed
func (t *TaskGenerator) SyncAll() (*SyncResult, error) {
	result := &SyncResult{}

	// Get all active (non-archived) shows
	shows, err := t.showRepo.GetAllActive()
	if err != nil {
		return nil, fmt.Errorf("failed to get active shows: %w", err)
	}

	for _, show := range shows {
		// Fetch latest data from cache only.
		tmdbData, ok, err := t.cacheSvc.GetCached(show.TMDBID)
		if err != nil {
			fmt.Printf("Warning: failed to load cached TMDB data for show %d (%s): %v\n", show.TMDBID, show.Name, err)
			result.Errors++
			continue
		}
		if !ok {
			// No cache yet; skip until manual refresh is triggered.
			continue
		}

		// Update local show data
		if err := t.updateShowData(&show, tmdbData); err != nil {
			fmt.Printf("Warning: failed to update show data for %s: %v\n", show.Name, err)
			result.Errors++
		}

		// Sync latest season episodes (manual refresh only)
		if tmdbData.NumberOfSeasons > 0 {
			if err := t.syncSeasonEpisodes(show.TMDBID, tmdbData.NumberOfSeasons); err != nil {
				fmt.Printf("Warning: failed to sync episodes for show %d: %v\n", show.TMDBID, err)
			}
		}

		// Check for episode updates and generate UPDATE_Task if needed
		task, err := t.checkEpisodeUpdate(&show, tmdbData)
		if err != nil {
			fmt.Printf("Warning: failed to check episode update for %s: %v\n", show.Name, err)
			result.Errors++
		} else if task != nil {
			result.UpdateTasks++
		}

		// Check if show ended and generate ORGANIZE_Task if needed
		task, err = t.checkShowEnded(&show, tmdbData)
		if err != nil {
			fmt.Printf("Warning: failed to check show ended for %s: %v\n", show.Name, err)
			result.Errors++
		} else if task != nil {
			result.OrganizeTasks++
		}
	}

	return result, nil
}

// syncSeasonEpisodes syncs episodes for a specific season from TMDB
func (t *TaskGenerator) syncSeasonEpisodes(tmdbID, seasonNumber int) error {
	episodes, err := t.tmdbClient.GetSeasonEpisodes(tmdbID, seasonNumber)
	if err != nil {
		return err
	}

	for _, ep := range episodes {
		episode := &models.Episode{
			TMDBID:   tmdbID,
			Season:   ep.SeasonNumber,
			Episode:  ep.EpisodeNumber,
			Title:    ep.Name,
			Overview: ep.Overview,
			AirDate:  ep.AirDate,
		}
		if err := t.episodeRepo.Upsert(episode); err != nil {
			return fmt.Errorf("failed to upsert episode %s: %w", FormatEpisodeID(ep.SeasonNumber, ep.EpisodeNumber), err)
		}
	}

	return nil
}

// updateShowData updates local show data with TMDB data
func (t *TaskGenerator) updateShowData(show *models.TVShow, tmdbData *tmdb.TVDetails) error {
	show.Name = tmdbData.Name
	show.TotalSeasons = tmdbData.NumberOfSeasons
	show.Status = tmdbData.Status

	// Update origin country if available
	if len(tmdbData.OriginCountry) > 0 {
		originCountry := tmdbData.OriginCountry[0]
		if show.OriginCountry != originCountry {
			show.OriginCountry = originCountry
			if !show.ResourceTimeIsManual {
				show.ResourceTime = InferResourceTime(originCountry)
			}
		}
	}

	return t.showRepo.Update(show)
}

// checkEpisodeUpdate checks if an UPDATE_Task should be generated for a show
// Creates a task if next_episode_to_air has air_date equal to today or in the past
// and no task exists for that episode yet
func (t *TaskGenerator) checkEpisodeUpdate(show *models.TVShow, tmdbData *tmdb.TVDetails) (*models.Task, error) {
	// Check next episode to air
	if tmdbData.NextEpisodeToAir != nil {
		task, err := t.createUpdateTaskIfNeeded(show, tmdbData.NextEpisodeToAir)
		if err != nil {
			return nil, err
		}
		if task != nil {
			return task, nil
		}
	}

	// Also check last episode to air (for missed episodes)
	if tmdbData.LastEpisodeToAir != nil {
		task, err := t.createUpdateTaskIfNeeded(show, tmdbData.LastEpisodeToAir)
		if err != nil {
			return nil, err
		}
		if task != nil {
			return task, nil
		}
	}

	return nil, nil
}

// createUpdateTaskIfNeeded creates an UPDATE_Task if the episode air date is today or in the past
// and no task exists for that episode yet
func (t *TaskGenerator) createUpdateTaskIfNeeded(show *models.TVShow, episode *tmdb.EpisodeInfo) (*models.Task, error) {
	if episode == nil || episode.AirDate == "" {
		return nil, nil
	}

	// Parse air date in local timezone to match timeutil.Now().
	airDate, err := time.ParseInLocation("2006-01-02", episode.AirDate, timeutil.Now().Location())
	if err != nil {
		return nil, fmt.Errorf("failed to parse air date %s: %w", episode.AirDate, err)
	}

	// Check if air date is today or in the past (local date comparison).
	now := timeutil.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	airDateDay := time.Date(airDate.Year(), airDate.Month(), airDate.Day(), 0, 0, 0, 0, airDate.Location())

	if airDateDay.After(today) {
		// Episode hasn't aired yet
		return nil, nil
	}

	// Format episode ID
	episodeID := FormatEpisodeID(episode.SeasonNumber, episode.EpisodeNumber)

	// Check if task already exists for this episode
	existingTask, err := t.taskRepo.GetByShowAndEpisode(show.ID, episodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing task: %w", err)
	}
	if existingTask != nil {
		// Task already exists, don't create duplicate
		return nil, nil
	}

	// Create UPDATE_Task. Use stable prefix "SxxExx|" for exact matching.
	description := fmt.Sprintf("%s|新剧集更新: %s - %s", episodeID, episodeID, episode.Name)
	task := &models.Task{
		TVShowID:    show.ID,
		TaskType:    models.TaskTypeUpdate,
		Description: description,
		IsCompleted: false,
	}

	if err := t.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("failed to create UPDATE_Task: %w", err)
	}

	return task, nil
}

// checkShowEnded checks if an ORGANIZE_Task should be generated for a show
// Creates a task if status is "Ended" or "Canceled" and no ORGANIZE_Task exists
func (t *TaskGenerator) checkShowEnded(show *models.TVShow, tmdbData *tmdb.TVDetails) (*models.Task, error) {
	// Check if show has ended or been canceled
	if tmdbData.Status != "Ended" && tmdbData.Status != "Canceled" {
		return nil, nil
	}

	// Check if ORGANIZE_Task already exists
	exists, err := t.taskRepo.ExistsOrganizeTask(show.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing ORGANIZE_Task: %w", err)
	}
	if exists {
		// Task already exists, don't create duplicate
		return nil, nil
	}

	// Create ORGANIZE_Task
	description := fmt.Sprintf("剧集已完结，请整理归档本地文件")
	task := &models.Task{
		TVShowID:    show.ID,
		TaskType:    models.TaskTypeOrganize,
		Description: description,
		IsCompleted: false,
	}

	if err := t.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("failed to create ORGANIZE_Task: %w", err)
	}

	return task, nil
}
