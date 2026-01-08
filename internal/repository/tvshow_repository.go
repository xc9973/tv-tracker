package repository

import (
	"database/sql"

	"tv-tracker/internal/models"
	"tv-tracker/internal/timeutil"
)

// TVShowRepository handles TVShow database operations
type TVShowRepository struct {
	db *sql.DB
}

// NewTVShowRepository creates a new TVShowRepository
func NewTVShowRepository(sqliteDB *SQLiteDB) *TVShowRepository {
	return &TVShowRepository{db: sqliteDB.db}
}

// Create inserts a new TVShow into the database
func (r *TVShowRepository) Create(show *models.TVShow) error {
	now := timeutil.Now()
	result, err := r.db.Exec(`
		INSERT INTO tv_shows (tmdb_id, name, total_seasons, status, origin_country, resource_time, is_archived, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, show.TMDBID, show.Name, show.TotalSeasons, show.Status, show.OriginCountry, show.ResourceTime, show.IsArchived, now, now)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	show.ID = id
	show.CreatedAt = now
	show.UpdatedAt = now
	return nil
}

// GetByTMDBID retrieves a TVShow by its TMDB ID
func (r *TVShowRepository) GetByTMDBID(tmdbID int) (*models.TVShow, error) {
	show := &models.TVShow{}
	err := r.db.QueryRow(`
		SELECT id, tmdb_id, name, total_seasons, status, origin_country, resource_time, is_archived, created_at, updated_at
		FROM tv_shows WHERE tmdb_id = ?
	`, tmdbID).Scan(
		&show.ID, &show.TMDBID, &show.Name, &show.TotalSeasons, &show.Status,
		&show.OriginCountry, &show.ResourceTime, &show.IsArchived, &show.CreatedAt, &show.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return show, nil
}

// GetByID retrieves a TVShow by its ID
func (r *TVShowRepository) GetByID(id int64) (*models.TVShow, error) {
	show := &models.TVShow{}
	err := r.db.QueryRow(`
		SELECT id, tmdb_id, name, total_seasons, status, origin_country, resource_time, is_archived, created_at, updated_at
		FROM tv_shows WHERE id = ?
	`, id).Scan(
		&show.ID, &show.TMDBID, &show.Name, &show.TotalSeasons, &show.Status,
		&show.OriginCountry, &show.ResourceTime, &show.IsArchived, &show.CreatedAt, &show.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return show, nil
}

// GetAllActive retrieves all non-archived TVShows
func (r *TVShowRepository) GetAllActive() ([]models.TVShow, error) {
	rows, err := r.db.Query(`
		SELECT id, tmdb_id, name, total_seasons, status, origin_country, resource_time, is_archived, created_at, updated_at
		FROM tv_shows WHERE is_archived = FALSE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shows []models.TVShow
	for rows.Next() {
		var show models.TVShow
		err := rows.Scan(
			&show.ID, &show.TMDBID, &show.Name, &show.TotalSeasons, &show.Status,
			&show.OriginCountry, &show.ResourceTime, &show.IsArchived, &show.CreatedAt, &show.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		shows = append(shows, show)
	}
	return shows, rows.Err()
}

// GetAll retrieves all TVShows
func (r *TVShowRepository) GetAll() ([]models.TVShow, error) {
	rows, err := r.db.Query(`
		SELECT id, tmdb_id, name, total_seasons, status, origin_country, resource_time, is_archived, created_at, updated_at
		FROM tv_shows
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shows []models.TVShow
	for rows.Next() {
		var show models.TVShow
		err := rows.Scan(
			&show.ID, &show.TMDBID, &show.Name, &show.TotalSeasons, &show.Status,
			&show.OriginCountry, &show.ResourceTime, &show.IsArchived, &show.CreatedAt, &show.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		shows = append(shows, show)
	}
	return shows, rows.Err()
}

// Update updates an existing TVShow in the database
func (r *TVShowRepository) Update(show *models.TVShow) error {
	now := timeutil.Now()
	_, err := r.db.Exec(`
		UPDATE tv_shows 
		SET name = ?, total_seasons = ?, status = ?, origin_country = ?, resource_time = ?, is_archived = ?, updated_at = ?
		WHERE id = ?
	`, show.Name, show.TotalSeasons, show.Status, show.OriginCountry, show.ResourceTime, show.IsArchived, now, show.ID)
	if err != nil {
		return err
	}
	show.UpdatedAt = now
	return nil
}

// Archive sets a TVShow as archived
func (r *TVShowRepository) Archive(showID int64) error {
	now := timeutil.Now()
	_, err := r.db.Exec(`
		UPDATE tv_shows SET is_archived = TRUE, updated_at = ? WHERE id = ?
	`, now, showID)
	return err
}
