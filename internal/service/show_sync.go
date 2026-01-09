package service

import (
	"fmt"

	"tv-tracker/internal/models"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/tmdb"
)

// ShowSyncService handles single-show refresh flows.
type ShowSyncService struct {
	cacheSvc    *TMDBCacheService
	taskGen     *TaskGenerator
	showRepo    *repository.TVShowRepository
	episodeRepo *repository.EpisodeRepository
}

// NewShowSyncService creates a new ShowSyncService.
func NewShowSyncService(
	cacheSvc *TMDBCacheService,
	taskGen *TaskGenerator,
	showRepo *repository.TVShowRepository,
	episodeRepo *repository.EpisodeRepository,
) *ShowSyncService {
	return &ShowSyncService{
		cacheSvc:    cacheSvc,
		taskGen:     taskGen,
		showRepo:    showRepo,
		episodeRepo: episodeRepo,
	}
}

// RefreshShow fetches TMDB details and syncs local data for a show.
func (s *ShowSyncService) RefreshShow(tmdbID int) (*tmdb.TVDetails, error) {
	// Refresh cache with latest TMDB data.
	details, err := s.cacheSvc.Refresh(tmdbID)
	if err != nil {
		return nil, err
	}

	// Ensure show exists locally before syncing.
	show, err := s.showRepo.GetByTMDBID(tmdbID)
	if err != nil {
		return nil, fmt.Errorf("failed to load show: %w", err)
	}
	if show == nil {
		show = &models.TVShow{
			TMDBID:               details.ID,
			Name:                 details.Name,
			TotalSeasons:         details.NumberOfSeasons,
			Status:               details.Status,
			OriginCountry:        "",
			ResourceTime:         "待定",
			ResourceTimeIsManual: false,
			IsArchived:           false,
		}
		if len(details.OriginCountry) > 0 {
			show.OriginCountry = details.OriginCountry[0]
			if !show.ResourceTimeIsManual {
				show.ResourceTime = InferResourceTime(show.OriginCountry)
			}
		}
		if err := s.showRepo.Create(show); err != nil {
			return nil, fmt.Errorf("failed to create show: %w", err)
		}
	} else {
		show.Name = details.Name
		show.TotalSeasons = details.NumberOfSeasons
		show.Status = details.Status
		if len(details.OriginCountry) > 0 {
			originCountry := details.OriginCountry[0]
			if show.OriginCountry != originCountry {
				show.OriginCountry = originCountry
				if !show.ResourceTimeIsManual {
					show.ResourceTime = InferResourceTime(originCountry)
				}
			}
		}
		if err := s.showRepo.Update(show); err != nil {
			return nil, fmt.Errorf("failed to update show: %w", err)
		}
	}

	// Sync latest season episodes.
	if details.NumberOfSeasons > 0 {
		if err := s.taskGen.syncSeasonEpisodes(tmdbID, details.NumberOfSeasons); err != nil {
			return nil, fmt.Errorf("failed to sync episodes: %w", err)
		}
	}

	// Generate tasks based on refreshed data.
	if _, err := s.taskGen.checkEpisodeUpdate(show, details); err != nil {
		return nil, err
	}
	if _, err := s.taskGen.checkShowEnded(show, details); err != nil {
		return nil, err
	}

	return details, nil
}
