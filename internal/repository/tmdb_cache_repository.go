package repository

import (
	"database/sql"
)

// TMDBCacheRepository stores raw TMDB TV details snapshots.
type TMDBCacheRepository struct {
	db *sql.DB
}

// NewTMDBCacheRepository creates a new TMDBCacheRepository.
func NewTMDBCacheRepository(sqliteDB *SQLiteDB) *TMDBCacheRepository {
	return &TMDBCacheRepository{db: sqliteDB.db}
}

// Get returns cached payload JSON for a TMDB ID.
func (r *TMDBCacheRepository) Get(tmdbID int) (string, bool, error) {
	var payload string
	err := r.db.QueryRow(`
		SELECT payload_json
		FROM tmdb_cache
		WHERE tmdb_id = ?
	`, tmdbID).Scan(&payload)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return payload, true, nil
}

// Upsert writes the latest TMDB payload JSON.
func (r *TMDBCacheRepository) Upsert(tmdbID int, payloadJSON string, fetchedAt string, language string) error {
	_, err := r.db.Exec(`
		INSERT INTO tmdb_cache (tmdb_id, payload_json, fetched_at, language)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(tmdb_id) DO UPDATE SET
			payload_json = excluded.payload_json,
			fetched_at = excluded.fetched_at,
			language = excluded.language
	`, tmdbID, payloadJSON, fetchedAt, language)
	return err
}
