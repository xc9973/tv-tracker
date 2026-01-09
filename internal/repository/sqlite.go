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

// InitSchema creates the database tables and runs migrations
func (s *SQLiteDB) InitSchema() error {
	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS tv_shows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tmdb_id INTEGER UNIQUE NOT NULL,
		name TEXT NOT NULL,
		total_seasons INTEGER DEFAULT 1,
		status TEXT DEFAULT 'Unknown',
		origin_country TEXT DEFAULT '',
		resource_time TEXT DEFAULT '待定',
		resource_time_is_manual BOOLEAN DEFAULT FALSE,
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

	CREATE TABLE IF NOT EXISTS tmdb_cache (
		tmdb_id INTEGER PRIMARY KEY,
		payload_json TEXT NOT NULL,
		fetched_at TIMESTAMP NOT NULL,
		language TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_episodes_air_date ON episodes(air_date);
	CREATE INDEX IF NOT EXISTS idx_episodes_tmdb ON episodes(tmdb_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_completed ON tasks(is_completed);
	CREATE INDEX IF NOT EXISTS idx_shows_archived ON tv_shows(is_archived);
	
	-- 复合索引优化 JOIN 查询性能
	CREATE INDEX IF NOT EXISTS idx_episodes_air_date_tmdb ON episodes(air_date, tmdb_id);
	CREATE INDEX IF NOT EXISTS idx_shows_tmdb_archived ON tv_shows(tmdb_id, is_archived);
	CREATE INDEX IF NOT EXISTS idx_tasks_show_completed ON tasks(tv_show_id, is_completed);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Run migrations
	return s.runMigrations()
}

// runMigrations executes pending database migrations
func (s *SQLiteDB) runMigrations() error {
	// Check if resource_time_is_manual column exists
	var result string
	err := s.db.QueryRow("SELECT resource_time_is_manual FROM tv_shows LIMIT 1").Scan(&result)

	if err != nil {
		// Column doesn't exist, need to migrate
		return s.migrateResourceTimeIsManual()
	}

	return nil
}

// migrateResourceTimeIsManual adds the resource_time_is_manual column
func (s *SQLiteDB) migrateResourceTimeIsManual() error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create new table with the column
	_, err = tx.Exec(`
		CREATE TABLE tv_shows_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tmdb_id INTEGER UNIQUE NOT NULL,
			name TEXT NOT NULL,
			total_seasons INTEGER DEFAULT 1,
			status TEXT DEFAULT 'Unknown',
			origin_country TEXT DEFAULT '',
			resource_time TEXT DEFAULT '待定',
			resource_time_is_manual BOOLEAN DEFAULT FALSE,
			is_archived BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Copy data from old table
	_, err = tx.Exec(`
		INSERT INTO tv_shows_new (
			id, tmdb_id, name, total_seasons, status, origin_country, 
			resource_time, is_archived, created_at, updated_at
		)
		SELECT 
			id, tmdb_id, name, total_seasons, status, origin_country, 
			resource_time, is_archived, created_at, updated_at
		FROM tv_shows
	`)
	if err != nil {
		return err
	}

	// Drop old table
	_, err = tx.Exec(`DROP TABLE tv_shows`)
	if err != nil {
		return err
	}

	// Rename new table
	_, err = tx.Exec(`ALTER TABLE tv_shows_new RENAME TO tv_shows`)
	if err != nil {
		return err
	}

	// Recreate index
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_shows_tmdb_archived ON tv_shows(tmdb_id, is_archived)`)
	if err != nil {
		return err
	}

	// Commit transaction
	return tx.Commit()
}
