package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/akshithere/task-scheduler/internal/database"
	"github.com/akshithere/task-scheduler/internal/models"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type TaskService struct {
	db *gorm.DB
}

func NewTaskService() *TaskService {
	return &TaskService{
		db: database.DB,
	}
}

func (s *TaskService) CreateTask(task *models.Task) error {
	if err := s.validateTrigger(&task.Trigger); err != nil {
		return fmt.Errorf("invalid trigger: %w", err)
	}

	nextRun, err := s.calculateNextRun(&task.Trigger)
	if err != nil {
		return fmt.Errorf("failed to calculate next run: %w", err)
	}
	task.NextRun = nextRun

	task.Status = models.TaskStatusScheduled

	if err := s.db.Create(task).Error; err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

func (s *TaskService) GetTask(id uuid.UUID) (*models.Task, error) {
	var task models.Task
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return &task, nil
}

func (s *TaskService) ListTasks(status *string, limit, offset int) ([]models.Task, int64, error) {
	var tasks []models.Task
	var total int64

	query := s.db.Model(&models.Task{})

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	return tasks, total, nil
}

func (s *TaskService) UpdateTask(id uuid.UUID, updates *models.Task) (*models.Task, error) {
	task, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}

	if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusCancelled {
		return nil, fmt.Errorf("cannot update task with status: %s", task.Status)
	}

	if updates.Name != "" {
		task.Name = updates.Name
	}

	if updates.Trigger.Type != "" {
		if err := s.validateTrigger(&updates.Trigger); err != nil {
			return nil, fmt.Errorf("invalid trigger: %w", err)
		}
		task.Trigger = updates.Trigger

		nextRun, err := s.calculateNextRun(&task.Trigger)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate next run: %w", err)
		}
		task.NextRun = nextRun
	}

	if updates.Action.Method != "" {
		task.Action = updates.Action
	}

	if err := s.db.Save(task).Error; err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

func (s *TaskService) CancelTask(id uuid.UUID) error {
	result := s.db.Model(&models.Task{}).
		Where("id = ? AND status = ?", id, models.TaskStatusScheduled).
		Update("status", models.TaskStatusCancelled)

	if result.Error != nil {
		return fmt.Errorf("failed to cancel task: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found or already cancelled/completed")
	}

	return nil
}

func (s *TaskService) GetTaskResults(taskID uuid.UUID, limit, offset int) ([]models.TaskResult, int64, error) {
	var results []models.TaskResult
	var total int64

	query := s.db.Model(&models.TaskResult{}).Where("task_id = ?", taskID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count task results: %w", err)
	}

	if err := query.Order("run_at DESC").Limit(limit).Offset(offset).Find(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get task results: %w", err)
	}

	return results, total, nil
}

func (s *TaskService) ListAllResults(taskID *uuid.UUID, success *bool, limit, offset int) ([]models.TaskResult, int64, error) {
	var results []models.TaskResult
	var total int64

	query := s.db.Model(&models.TaskResult{})

	if taskID != nil {
		query = query.Where("task_id = ?", *taskID)
	}
	if success != nil {
		query = query.Where("success = ?", *success)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count results: %w", err)
	}

	if err := query.Preload("Task").Order("run_at DESC").Limit(limit).Offset(offset).Find(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list results: %w", err)
	}

	return results, total, nil
}

func (s *TaskService) validateTrigger(trigger *models.Trigger) error {
	switch trigger.Type {
	case models.TriggerTypeOneOff:
		if trigger.DateTime == nil {
			return errors.New("datetime is required for one-off trigger")
		}
		if trigger.DateTime.Before(time.Now().UTC()) {
			return errors.New("datetime must be in the future")
		}
	case models.TriggerTypeCron:
		if trigger.Cron == nil || *trigger.Cron == "" {
			return errors.New("cron expression is required for cron trigger")
		}

		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err := parser.Parse(*trigger.Cron); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	default:
		return errors.New("trigger type must be 'one-off' or 'cron'")
	}
	return nil
}

func (s *TaskService) calculateNextRun(trigger *models.Trigger) (*time.Time, error) {
	switch trigger.Type {
	case models.TriggerTypeOneOff:
		return trigger.DateTime, nil
	case models.TriggerTypeCron:
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(*trigger.Cron)
		if err != nil {
			return nil, err
		}
		nextRun := schedule.Next(time.Now().UTC())
		return &nextRun, nil
	default:
		return nil, errors.New("invalid trigger type")
	}
}
