package models

import "time"

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeUpdate   TaskType = "UPDATE"
	TaskTypeOrganize TaskType = "ORGANIZE"
)

// TVShow represents a subscribed TV show
type TVShow struct {
	ID                   int64     `json:"id"`
	TMDBID               int       `json:"tmdb_id"`
	Name                 string    `json:"name"`
	TotalSeasons         int       `json:"total_seasons"`
	Status               string    `json:"status"`         // Returning Series, Ended, Canceled
	OriginCountry        string    `json:"origin_country"` // Country code (US, CN, JP, etc.)
	ResourceTime         string    `json:"resource_time"`  // Expected resource availability time
	ResourceTimeIsManual bool      `json:"resource_time_is_manual"`
	IsArchived           bool      `json:"is_archived"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Episode represents a cached episode from TMDB
type Episode struct {
	ID       int64  `json:"id"`
	TMDBID   int    `json:"tmdb_id"`
	Season   int    `json:"season"`
	Episode  int    `json:"episode"`
	Title    string `json:"title"`
	Overview string `json:"overview"`
	AirDate  string `json:"air_date"` // YYYY-MM-DD format
}

// Task represents a pending task (update reminder or organize task)
type Task struct {
	ID           int64     `json:"id"`
	TVShowID     int64     `json:"tv_show_id"`
	TVShowName   string    `json:"tv_show_name"`  // For display purposes
	ResourceTime string    `json:"resource_time"` // Expected resource availability time
	TaskType     TaskType  `json:"task_type"`
	Description  string    `json:"description"`
	IsCompleted  bool      `json:"is_completed"`
	CreatedAt    time.Time `json:"created_at"`
}
