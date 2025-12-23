package repository

import (
	"database/sql"

	"tv-tracker/internal/models"
)

// EpisodeRepository handles Episode database operations
type EpisodeRepository struct {
	db *sql.DB
}

// NewEpisodeRepository creates a new EpisodeRepository
func NewEpisodeRepository(sqliteDB *SQLiteDB) *EpisodeRepository {
	return &EpisodeRepository{db: sqliteDB.db}
}

// Upsert inserts or updates an episode
func (r *EpisodeRepository) Upsert(episode *models.Episode) error {
	result, err := r.db.Exec(`
		INSERT INTO episodes (tmdb_id, season, episode, title, overview, air_date)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(tmdb_id, season, episode) DO UPDATE SET
			title = excluded.title,
			overview = excluded.overview,
			air_date = excluded.air_date
	`, episode.TMDBID, episode.Season, episode.Episode, episode.Title, episode.Overview, episode.AirDate)
	if err != nil {
		return err
	}
	if episode.ID == 0 {
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		episode.ID = id
	}
	return nil
}

// GetByTMDBID retrieves all episodes for a show by TMDB ID
func (r *EpisodeRepository) GetByTMDBID(tmdbID int) ([]models.Episode, error) {
	rows, err := r.db.Query(`
		SELECT id, tmdb_id, season, episode, title, overview, air_date
		FROM episodes WHERE tmdb_id = ?
		ORDER BY season, episode
	`, tmdbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []models.Episode
	for rows.Next() {
		var ep models.Episode
		var airDate sql.NullString
		err := rows.Scan(&ep.ID, &ep.TMDBID, &ep.Season, &ep.Episode, &ep.Title, &ep.Overview, &airDate)
		if err != nil {
			return nil, err
		}
		if airDate.Valid {
			ep.AirDate = airDate.String
		}
		episodes = append(episodes, ep)
	}
	return episodes, rows.Err()
}

// GetByAirDate retrieves all episodes airing on a specific date
func (r *EpisodeRepository) GetByAirDate(date string) ([]models.Episode, error) {
	rows, err := r.db.Query(`
		SELECT id, tmdb_id, season, episode, title, overview, air_date
		FROM episodes WHERE air_date = ?
		ORDER BY tmdb_id, season, episode
	`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []models.Episode
	for rows.Next() {
		var ep models.Episode
		var airDate sql.NullString
		err := rows.Scan(&ep.ID, &ep.TMDBID, &ep.Season, &ep.Episode, &ep.Title, &ep.Overview, &airDate)
		if err != nil {
			return nil, err
		}
		if airDate.Valid {
			ep.AirDate = airDate.String
		}
		episodes = append(episodes, ep)
	}
	return episodes, rows.Err()
}

// DeleteByTMDBID deletes all episodes for a show
func (r *EpisodeRepository) DeleteByTMDBID(tmdbID int) error {
	_, err := r.db.Exec(`DELETE FROM episodes WHERE tmdb_id = ?`, tmdbID)
	return err
}
