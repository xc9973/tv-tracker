package repository

import (
	"database/sql"
	"time"

	"tv-tracker/internal/models"
)

// TaskRepository handles Task database operations
type TaskRepository struct {
	db *sql.DB
}

// NewTaskRepository creates a new TaskRepository
func NewTaskRepository(sqliteDB *SQLiteDB) *TaskRepository {
	return &TaskRepository{db: sqliteDB.db}
}

// Create inserts a new Task into the database
func (r *TaskRepository) Create(task *models.Task) error {
	now := time.Now()
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
	task := &models.Task{}
	err := r.db.QueryRow(`
		SELECT t.id, t.tv_show_id, s.name, s.resource_time, t.task_type, t.description, t.is_completed, t.created_at
		FROM tasks t
		JOIN tv_shows s ON t.tv_show_id = s.id
		WHERE t.tv_show_id = ? AND t.description LIKE ?
	`, showID, "%"+episode+"%").Scan(
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
