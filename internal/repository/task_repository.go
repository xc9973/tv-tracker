package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"tv-tracker/internal/models"
	"tv-tracker/internal/timeutil"
)

type dbtx interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// TaskRepository handles Task database operations
type TaskRepository struct {
	db   dbtx
	base *sql.DB
}

// NewTaskRepository creates a new TaskRepository
func NewTaskRepository(sqliteDB *SQLiteDB) *TaskRepository {
	return &TaskRepository{db: sqliteDB.db, base: sqliteDB.db}
}

func (r *TaskRepository) BeginTx() (*sql.Tx, error) {
	if r.base == nil {
		return nil, errors.New("task repository: transactions not supported on tx-scoped repo")
	}
	return r.base.Begin()
}

func (r *TaskRepository) WithTx(tx *sql.Tx) *TaskRepository {
	return &TaskRepository{db: tx}
}

// Create inserts a new Task into the database
func (r *TaskRepository) Create(task *models.Task) error {
	now := timeutil.Now()
	result, err := r.db.Exec(`
		INSERT INTO tasks (tv_show_id, task_type, description, is_completed, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, task.TVShowID, task.TaskType, task.Description, task.IsCompleted, now)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	task.ID = id
	task.CreatedAt = now
	return nil
}

// GetPendingByType retrieves all pending tasks of a specific type
func (r *TaskRepository) GetPendingByType(taskType models.TaskType) ([]models.Task, error) {
	rows, err := r.db.Query(`
		SELECT t.id, t.tv_show_id, s.name, s.resource_time, t.task_type, t.description, t.is_completed, t.created_at
		FROM tasks t
		JOIN tv_shows s ON t.tv_show_id = s.id
		WHERE t.is_completed = FALSE AND t.task_type = ?
		ORDER BY t.created_at DESC
	`, taskType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		err := rows.Scan(
			&task.ID, &task.TVShowID, &task.TVShowName, &task.ResourceTime,
			&task.TaskType, &task.Description, &task.IsCompleted, &task.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// GetByShowAndEpisode retrieves a task by show ID and episode description
func (r *TaskRepository) GetByShowAndEpisode(showID int64, episode string) (*models.Task, error) {
	// Prefer stable prefix match: "SxxExx|..." to avoid partial matches.
	// For legacy tasks, fall back to safer matching (including unpadded "S01E1" cases).
	normalized, season, epNum, normalizedOK := normalizeEpisodeID(episode)

	if normalizedOK {
		prefixPattern := normalized + "|%"
		if task, found, err := r.getByShowAndDescriptionLike(showID, prefixPattern); err != nil {
			return nil, err
		} else if found {
			return task, nil
		}

		// Legacy: padded token anywhere.
		if task, found, err := r.getByShowAndDescriptionLike(showID, "%"+normalized+"%"); err != nil {
			return nil, err
		} else if found {
			return task, nil
		}

		// Legacy: unpadded token with digit boundary to avoid S01E1 matching S01E10.
		unpadded := fmt.Sprintf("S%dE%d", season, epNum)
		glob1 := "*" + unpadded + "[^0-9]*"
		glob2 := "*" + unpadded
		if task, found, err := r.getByShowAndDescriptionGlob(showID, glob1, glob2); err != nil {
			return nil, err
		} else if found {
			return task, nil
		}

		return nil, nil
	}

	// Unknown episode ID format: keep previous behavior.
	if task, found, err := r.getByShowAndDescriptionLike(showID, "%"+episode+"%"); err != nil {
		return nil, err
	} else if found {
		return task, nil
	}
	return nil, nil
}

func (r *TaskRepository) getByShowAndDescriptionLike(showID int64, pattern string) (*models.Task, bool, error) {
	task := &models.Task{}
	err := r.db.QueryRow(`
		SELECT t.id, t.tv_show_id, s.name, s.resource_time, t.task_type, t.description, t.is_completed, t.created_at
		FROM tasks t
		JOIN tv_shows s ON t.tv_show_id = s.id
		WHERE t.tv_show_id = ? AND t.description LIKE ?
		LIMIT 1
	`, showID, pattern).Scan(
		&task.ID, &task.TVShowID, &task.TVShowName, &task.ResourceTime,
		&task.TaskType, &task.Description, &task.IsCompleted, &task.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return task, true, nil
}

func (r *TaskRepository) getByShowAndDescriptionGlob(showID int64, patterns ...string) (*models.Task, bool, error) {
	if len(patterns) == 0 {
		return nil, false, nil
	}

	// Build a small OR chain without risking SQL injection (patterns are bound args).
	where := "t.tv_show_id = ? AND ("
	args := make([]any, 0, 1+len(patterns))
	args = append(args, showID)
	for i := range patterns {
		if i > 0 {
			where += " OR "
		}
		where += "t.description GLOB ?"
		args = append(args, patterns[i])
	}
	where += ")"

	query := fmt.Sprintf(`
		SELECT t.id, t.tv_show_id, s.name, s.resource_time, t.task_type, t.description, t.is_completed, t.created_at
		FROM tasks t
		JOIN tv_shows s ON t.tv_show_id = s.id
		WHERE %s
		LIMIT 1
	`, where)

	task := &models.Task{}
	err := r.db.QueryRow(query, args...).Scan(
		&task.ID, &task.TVShowID, &task.TVShowName, &task.ResourceTime,
		&task.TaskType, &task.Description, &task.IsCompleted, &task.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return task, true, nil
}

var episodeIDRe = regexp.MustCompile(`^S(\d+)E(\d+)$`)

func normalizeEpisodeID(input string) (normalized string, season int, episode int, ok bool) {
	m := episodeIDRe.FindStringSubmatch(input)
	if m == nil {
		return "", 0, 0, false
	}

	season, err := strconv.Atoi(m[1])
	if err != nil {
		return "", 0, 0, false
	}
	episode, err = strconv.Atoi(m[2])
	if err != nil {
		return "", 0, 0, false
	}

	if season < 0 || episode < 0 {
		return "", 0, 0, false
	}

	return fmt.Sprintf("S%02dE%02d", season, episode), season, episode, true
}

// ExistsOrganizeTask checks if an ORGANIZE task exists for a show
func (r *TaskRepository) ExistsOrganizeTask(showID int64) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM tasks 
		WHERE tv_show_id = ? AND task_type = ?
	`, showID, models.TaskTypeOrganize).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Complete marks a task as completed
func (r *TaskRepository) Complete(taskID int64) error {
	_, err := r.db.Exec(`
		UPDATE tasks SET is_completed = TRUE WHERE id = ?
	`, taskID)
	return err
}

// GetAllPending retrieves all pending tasks
func (r *TaskRepository) GetAllPending() ([]models.Task, error) {
	rows, err := r.db.Query(`
		SELECT t.id, t.tv_show_id, s.name, s.resource_time, t.task_type, t.description, t.is_completed, t.created_at
		FROM tasks t
		JOIN tv_shows s ON t.tv_show_id = s.id
		WHERE t.is_completed = FALSE
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		err := rows.Scan(
			&task.ID, &task.TVShowID, &task.TVShowName, &task.ResourceTime,
			&task.TaskType, &task.Description, &task.IsCompleted, &task.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// GetByID retrieves a task by its ID
func (r *TaskRepository) GetByID(taskID int64) (*models.Task, error) {
	task := &models.Task{}
	err := r.db.QueryRow(`
		SELECT t.id, t.tv_show_id, s.name, s.resource_time, t.task_type, t.description, t.is_completed, t.created_at
		FROM tasks t
		JOIN tv_shows s ON t.tv_show_id = s.id
		WHERE t.id = ?
	`, taskID).Scan(
		&task.ID, &task.TVShowID, &task.TVShowName, &task.ResourceTime,
		&task.TaskType, &task.Description, &task.IsCompleted, &task.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return task, nil
}

// Delete removes a task by its ID
func (r *TaskRepository) Delete(taskID int64) error {
	_, err := r.db.Exec(`DELETE FROM tasks WHERE id = ?`, taskID)
	return err
}

// CreateWithDate inserts a new Task with a specific created_at date
func (r *TaskRepository) CreateWithDate(task *models.Task, createdAt string) error {
	result, err := r.db.Exec(`
		INSERT INTO tasks (tv_show_id, task_type, description, is_completed, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, task.TVShowID, task.TaskType, task.Description, task.IsCompleted, createdAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	task.ID = id
	return nil
}
