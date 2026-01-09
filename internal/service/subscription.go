package service

import (
	"fmt"
	"strings"

	"tv-tracker/internal/models"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/tmdb"
)

// SubscriptionManager manages TV show subscriptions
type SubscriptionManager struct {
	tmdbClient  *tmdb.Client
	cacheSvc    *TMDBCacheService
	showRepo    *repository.TVShowRepository
	episodeRepo *repository.EpisodeRepository
}

// NewSubscriptionManager creates a new SubscriptionManager
func NewSubscriptionManager(
	tmdbClient *tmdb.Client,
	cacheSvc *TMDBCacheService,
	showRepo *repository.TVShowRepository,
	episodeRepo *repository.EpisodeRepository,
) *SubscriptionManager {
	return &SubscriptionManager{
		tmdbClient:  tmdbClient,
		cacheSvc:    cacheSvc,
		showRepo:    showRepo,
		episodeRepo: episodeRepo,
	}
}

// Subscribe subscribes to a TV show by TMDB ID
// Fetches show details from TMDB, stores the show, and syncs the latest season episodes
func (s *SubscriptionManager) Subscribe(tmdbID int) (*models.TVShow, bool, error) {
	// Check if already subscribed
	existing, err := s.showRepo.GetByTMDBID(tmdbID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check existing subscription: %w", err)
	}
	if existing != nil {
		return existing, true, nil // Already subscribed, return existing with flag
	}

	// Fetch details from cache or TMDB (manual refresh only when cache is empty)
	details, _, err := s.cacheSvc.GetOrRefresh(tmdbID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch TV details: %w", err)
	}

	// Determine origin country
	originCountry := ""
	if len(details.OriginCountry) > 0 {
		originCountry = details.OriginCountry[0]
	}

	// Create TVShow record
	show := &models.TVShow{
		TMDBID:               details.ID,
		Name:                 details.Name,
		TotalSeasons:         details.NumberOfSeasons,
		Status:               details.Status,
		OriginCountry:        originCountry,
		ResourceTime:         InferResourceTime(originCountry),
		ResourceTimeIsManual: false,
		IsArchived:           false,
	}

	if err := s.showRepo.Create(show); err != nil {
		return nil, false, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Sync latest season episodes (manual refresh only when cache is empty)
	if details.NumberOfSeasons > 0 {
		if err := s.syncSeasonEpisodes(tmdbID, details.NumberOfSeasons); err != nil {
			// Log error but don't fail the subscription
			fmt.Printf("Warning: failed to sync episodes for show %d: %v\n", tmdbID, err)
		}
	}

	return show, false, nil
}

// syncSeasonEpisodes syncs episodes for a specific season
func (s *SubscriptionManager) syncSeasonEpisodes(tmdbID, seasonNumber int) error {
	episodes, err := s.tmdbClient.GetSeasonEpisodes(tmdbID, seasonNumber)
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
		if err := s.episodeRepo.Upsert(episode); err != nil {
			return fmt.Errorf("failed to upsert episode S%02dE%02d: %w", ep.SeasonNumber, ep.EpisodeNumber, err)
		}
	}

	return nil
}

// IsSubscribed checks if a show is already subscribed
func (s *SubscriptionManager) IsSubscribed(tmdbID int) bool {
	show, err := s.showRepo.GetByTMDBID(tmdbID)
	if err != nil {
		return false
	}
	return show != nil
}

// GetAllSubscriptions returns all subscribed shows
func (s *SubscriptionManager) GetAllSubscriptions() ([]models.TVShow, error) {
	return s.showRepo.GetAll()
}

// Unsubscribe removes a subscription by show ID
func (s *SubscriptionManager) Unsubscribe(showID int64) error {
	// Get the show to find its TMDB ID
	show, err := s.showRepo.GetByID(showID)
	if err != nil {
		return fmt.Errorf("failed to get show: %w", err)
	}
	if show == nil {
		return fmt.Errorf("show not found: %d", showID)
	}

	// Delete associated episodes
	if err := s.episodeRepo.DeleteByTMDBID(show.TMDBID); err != nil {
		return fmt.Errorf("failed to delete episodes: %w", err)
	}

	// Archive the show (soft delete)
	if err := s.showRepo.Archive(showID); err != nil {
		return fmt.Errorf("failed to archive show: %w", err)
	}

	return nil
}

// InferResourceTime infers the expected resource availability time based on origin country
// US/UK/CA -> "18:00"
// CN/TW -> "20:00"
// JP/KR -> "23:00"
// Others -> "待定"
func InferResourceTime(originCountry string) string {
	country := strings.ToUpper(strings.TrimSpace(originCountry))

	switch country {
	case "US", "UK", "CA", "GB": // GB is the ISO code for UK
		return "18:00"
	case "CN", "TW":
		return "20:00"
	case "JP", "KR":
		return "23:00"
	default:
		return "待定"
	}
}
