package service

import (
	"fmt"

	"tv-tracker/internal/models"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/timeutil"
)

// DashboardData contains the data for the task dashboard
type DashboardData struct {
	UpdateTasks   []models.Task `json:"update_tasks"`
	OrganizeTasks []models.Task `json:"organize_tasks"`
}

// TaskBoardService handles task board operations
type TaskBoardService struct {
	taskRepo *repository.TaskRepository
	showRepo *repository.TVShowRepository
}

// NewTaskBoardService creates a new TaskBoardService
func NewTaskBoardService(taskRepo *repository.TaskRepository, showRepo *repository.TVShowRepository) *TaskBoardService {
	return &TaskBoardService{
		taskRepo: taskRepo,
		showRepo: showRepo,
	}
}

// GetDashboardData retrieves all pending tasks grouped by type
// Requirements: 6.4, 7.1, 7.2
func (s *TaskBoardService) GetDashboardData() (*DashboardData, error) {
	// Get pending UPDATE tasks
	updateTasks, err := s.taskRepo.GetPendingByType(models.TaskTypeUpdate)
	if err != nil {
		return nil, fmt.Errorf("failed to get update tasks: %w", err)
	}

	// Get pending ORGANIZE tasks
	organizeTasks, err := s.taskRepo.GetPendingByType(models.TaskTypeOrganize)
	if err != nil {
		return nil, fmt.Errorf("failed to get organize tasks: %w", err)
	}

	// Ensure non-nil slices for JSON serialization
	if updateTasks == nil {
		updateTasks = []models.Task{}
	}
	if organizeTasks == nil {
		organizeTasks = []models.Task{}
	}

	return &DashboardData{
		UpdateTasks:   updateTasks,
		OrganizeTasks: organizeTasks,
	}, nil
}

// CompleteTask marks a task as completed
// For ORGANIZE tasks, it also archives the associated TV show
// Requirements: 6.1, 6.2
func (s *TaskBoardService) CompleteTask(taskID int64) error {
	// Get the task first to check its type
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %d", taskID)
	}

	tx, err := s.taskRepo.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	taskRepo := s.taskRepo.WithTx(tx)
	showRepo := s.showRepo.WithTx(tx)

	// Mark the task as completed
	if err := taskRepo.Complete(taskID); err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	// For ORGANIZE tasks, also archive the associated TV show
	// Requirement 6.2: WHEN a user marks an ORGANIZE_Task as complete,
	// THE Task_Board SHALL set is_completed to True AND set the associated TVShow.is_archived to True
	if task.TaskType == models.TaskTypeOrganize {
		if err := showRepo.Archive(task.TVShowID); err != nil {
			return fmt.Errorf("failed to archive show: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// PostponeTask postpones a task to tomorrow by deleting it and recreating it with tomorrow's date
func (s *TaskBoardService) PostponeTask(taskID int64) error {
	// Get the task first
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %d", taskID)
	}

	tx, err := s.taskRepo.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	taskRepo := s.taskRepo.WithTx(tx)

	// Calculate tomorrow's date based on current time
	tomorrow := timeutil.Now().AddDate(0, 0, 1).Format("2006-01-02 15:04:05")

	// Create a new task for tomorrow
	newTask := &models.Task{
		TVShowID:    task.TVShowID,
		TaskType:    task.TaskType,
		Description: task.Description,
		IsCompleted: false,
	}

	if err := taskRepo.CreateWithDate(newTask, tomorrow); err != nil {
		return fmt.Errorf("failed to create postponed task: %w", err)
	}

	// Delete the original task
	if err := taskRepo.Delete(taskID); err != nil {
		return fmt.Errorf("failed to delete original task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
