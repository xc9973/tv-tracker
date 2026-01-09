package service

import (
	"encoding/json"
	"fmt"

	"tv-tracker/internal/repository"
	"tv-tracker/internal/timeutil"
	"tv-tracker/internal/tmdb"
)

const tmdbCacheLanguage = "zh-CN"

// TMDBCacheService provides manual-refresh caching for TMDB TV details.
type TMDBCacheService struct {
	client *tmdb.Client
	repo   *repository.TMDBCacheRepository
}

// NewTMDBCacheService creates a new TMDBCacheService.
func NewTMDBCacheService(client *tmdb.Client, repo *repository.TMDBCacheRepository) *TMDBCacheService {
	return &TMDBCacheService{
		client: client,
		repo:   repo,
	}
}

// GetCached returns cached details for a TMDB ID.
func (s *TMDBCacheService) GetCached(tmdbID int) (*tmdb.TVDetails, bool, error) {
	payload, ok, err := s.repo.Get(tmdbID)
	if err != nil || !ok {
		return nil, ok, err
	}

	var details tmdb.TVDetails
	if err := json.Unmarshal([]byte(payload), &details); err != nil {
		return nil, false, fmt.Errorf("failed to decode cached TMDB payload: %w", err)
	}
	return &details, true, nil
}

// Refresh fetches TMDB details and updates the cache.
func (s *TMDBCacheService) Refresh(tmdbID int) (*tmdb.TVDetails, error) {
	details, err := s.client.GetTVDetails(tmdbID)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(details)
	if err != nil {
		return nil, fmt.Errorf("failed to encode TMDB payload: %w", err)
	}

	fetchedAt := timeutil.Now().Format("2006-01-02 15:04:05")
	if err := s.repo.Upsert(tmdbID, string(payload), fetchedAt, tmdbCacheLanguage); err != nil {
		return nil, err
	}

	return details, nil
}

// GetOrRefresh returns cached data when present, otherwise refreshes.
func (s *TMDBCacheService) GetOrRefresh(tmdbID int) (*tmdb.TVDetails, bool, error) {
	cached, ok, err := s.GetCached(tmdbID)
	if err != nil {
		return nil, false, err
	}
	if ok {
		return cached, true, nil
	}

	refreshed, err := s.Refresh(tmdbID)
	if err != nil {
		return nil, false, err
	}
	return refreshed, false, nil
}
