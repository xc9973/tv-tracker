package repository

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDB wraps the database connection
type SQLiteDB struct {
	db *sql.DB
}

// NewSQLiteDB creates a new SQLite database connection with connection pool settings
func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Configure connection pool for optimal performance
	// SQLite benefits from limited connections due to write locking
	db.SetMaxOpenConns(25)                 // Maximum number of open connections
	db.SetMaxIdleConns(5)                  // Maximum number of idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum lifetime of a connection

	return &SQLiteDB{db: db}, nil
}

// Close closes the database connection
func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

// InitSchema creates the database tables
func (s *SQLiteDB) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tv_shows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tmdb_id INTEGER UNIQUE NOT NULL,
		name TEXT NOT NULL,
		total_seasons INTEGER DEFAULT 1,
		status TEXT DEFAULT 'Unknown',
		origin_country TEXT DEFAULT '',
		resource_time TEXT DEFAULT '待定',
		is_archived BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS episodes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tmdb_id INTEGER NOT NULL,
		season INTEGER NOT NULL,
		episode INTEGER NOT NULL,
		title TEXT,
		overview TEXT,
		air_date DATE,
		UNIQUE(tmdb_id, season, episode)
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tv_show_id INTEGER NOT NULL,
		task_type TEXT NOT NULL,
		description TEXT NOT NULL,
		is_completed BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (tv_show_id) REFERENCES tv_shows(id)
	);

	CREATE INDEX IF NOT EXISTS idx_episodes_air_date ON episodes(air_date);
	CREATE INDEX IF NOT EXISTS idx_episodes_tmdb ON episodes(tmdb_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_completed ON tasks(is_completed);
	CREATE INDEX IF NOT EXISTS idx_shows_archived ON tv_shows(is_archived);
	`
	_, err := s.db.Exec(schema)
	return err
}
