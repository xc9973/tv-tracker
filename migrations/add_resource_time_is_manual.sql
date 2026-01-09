-- Migration: Add resource_time_is_manual column
-- Description: Adds a column to track whether resource_time was manually set
-- Run this migration if you get "no such column: resource_time_is_manual" error

-- Check if column exists, if not add it
-- SQLite doesn't support IF NOT EXISTS for ALTER TABLE, so we use a workaround

-- Step 1: Create a new table with the desired schema
CREATE TABLE IF NOT EXISTS tv_shows_new (
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

-- Step 2: Copy data from old table to new table
INSERT INTO tv_shows_new (
    id, tmdb_id, name, total_seasons, status, origin_country, 
    resource_time, is_archived, created_at, updated_at
)
SELECT 
    id, tmdb_id, name, total_seasons, status, origin_country, 
    resource_time, is_archived, created_at, updated_at
FROM tv_shows;

-- Step 3: Drop old table
DROP TABLE tv_shows;

-- Step 4: Rename new table to original name
ALTER TABLE tv_shows_new RENAME TO tv_shows;

-- Step 5: Recreate indexes
CREATE INDEX IF NOT EXISTS idx_shows_tmdb_archived ON tv_shows(tmdb_id, is_archived);