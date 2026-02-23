package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/akshithere/task-scheduler/internal/database"
	"github.com/akshithere/task-scheduler/internal/models"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Scheduler struct {
	scheduler *gocron.Scheduler
	db        *gorm.DB
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	httpClient *http.Client
	metrics   *Metrics
}

func NewScheduler(metrics *Metrics) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		scheduler: gocron.NewScheduler(time.UTC),
		db:        database.DB,
		ctx:       ctx,
		cancel:    cancel,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		metrics: metrics,
	}
}

func (s *Scheduler) Start() error {
	log.Println("Starting task scheduler")

	s.scheduler.Every(10).Seconds().Do(s.pollAndExecuteTasks)

	s.scheduler.StartAsync()

	log.Println("Task scheduler started successfully")
	return nil
}

func (s *Scheduler) Stop() {
	log.Println("Stopping task scheduler")
	s.cancel()
	s.scheduler.Stop()
	s.wg.Wait()
	log.Println("Task scheduler stopped")
}

func (s *Scheduler) pollAndExecuteTasks() {
	var tasks []models.Task

	now := time.Now().UTC()
	err := s.db.Where("status = ? AND next_run <= ?", models.TaskStatusScheduled, now).
		Find(&tasks).Error

	if err != nil {
		log.Printf("Error polling tasks: %v", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	log.Printf("Found %d task(s) ready for execution", len(tasks))

	for _, task := range tasks {
		task := task
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.executeTask(&task)
		}()
	}
}

func (s *Scheduler) executeTask(task *models.Task) {
	lockAcquired, err := s.acquireAdvisoryLock(task.ID)
	if err != nil {
		log.Printf("Error acquiring lock for task %s: %v", task.ID, err)
		return
	}

	if !lockAcquired {
		log.Printf("Task %s is already being executed by another instance", task.ID)
		return
	}

	defer s.releaseAdvisoryLock(task.ID)

	log.Printf("Executing task: %s (ID: %s)", task.Name, task.ID)

	startTime := time.Now()
	result := s.performHTTPRequest(task)
	result.DurationMs = time.Since(startTime).Milliseconds()
	result.RunAt = startTime.UTC()
	result.TaskID = task.ID

	if err := s.db.Create(&result).Error; err != nil {
		log.Printf("Error saving task result for task %s: %v", task.ID, err)
	}

	if err := s.updateTaskAfterExecution(task, result.Success); err != nil {
		log.Printf("Error updating task after execution %s: %v", task.ID, err)
	}

	s.metrics.RecordTaskExecution(task.ID.String(), result.Success, result.DurationMs)

	log.Printf("Task %s executed successfully, result saved", task.ID)
}

func (s *Scheduler) acquireAdvisoryLock(taskID uuid.UUID) (bool, error) {
	lockID := hashUUIDToInt64(taskID)

	var locked bool
	err := s.db.Raw("SELECT pg_try_advisory_lock(?)", lockID).Scan(&locked).Error
	if err != nil {
		return false, err
	}

	return locked, nil
}

func (s *Scheduler) releaseAdvisoryLock(taskID uuid.UUID) error {
	lockID := hashUUIDToInt64(taskID)

	var unlocked bool
	err := s.db.Raw("SELECT pg_advisory_unlock(?)", lockID).Scan(&unlocked).Error
	if err != nil {
		return err
	}

	if !unlocked {
		log.Printf("Warning: Failed to release lock for task %s", taskID)
	}

	return nil
}

func hashUUIDToInt64(id uuid.UUID) int64 {
	bytes := id[0:8]
	var result int64
	for i, b := range bytes {
		result |= int64(b) << (8 * i)
	}
	return result
}

func (s *Scheduler) performHTTPRequest(task *models.Task) models.TaskResult {
	result := models.TaskResult{
		TaskID: task.ID,
		RunAt:  time.Now().UTC(),
	}

	var reqBody io.Reader
	if task.Action.Payload != nil {
		reqBody = bytes.NewReader(task.Action.Payload)
	}

	req, err := http.NewRequestWithContext(s.ctx, task.Action.Method, task.Action.URL, reqBody)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create request: %v", err)
		result.ErrorMessage = &errMsg
		result.Success = false
		return result
	}

	for key, value := range task.Action.Headers {
		req.Header.Set(key, value)
	}

	if task.Action.Payload != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("Request failed: %v", err)
		result.ErrorMessage = &errMsg
		result.Success = false
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	headersJSON, _ := json.Marshal(resp.Header)
	result.ResponseHeaders = headersJSON

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		errMsg := fmt.Sprintf("Failed to read response body: %v", err)
		result.ErrorMessage = &errMsg
	} else {
		result.ResponseBody = string(bodyBytes)
	}

	if !result.Success {
		errMsg := fmt.Sprintf("HTTP request returned status code %d", resp.StatusCode)
		result.ErrorMessage = &errMsg
	}

	return result
}

func (s *Scheduler) updateTaskAfterExecution(task *models.Task, success bool) error {
	if task.Trigger.Type == models.TriggerTypeOneOff {
		return s.db.Model(task).Updates(map[string]interface{}{
			"status":   models.TaskStatusCompleted,
			"next_run": nil,
		}).Error
	}

	if task.Trigger.Type == models.TriggerTypeCron && task.Trigger.Cron != nil {
		nextRun, err := calculateNextCronRun(*task.Trigger.Cron)
		if err != nil {
			log.Printf("Error calculating next run for task %s: %v", task.ID, err)
			return err
		}

		return s.db.Model(task).Update("next_run", nextRun).Error
	}

	return nil
}

func calculateNextCronRun(cronExpr string) (*time.Time, error) {
	parser := gocron.NewScheduler(time.UTC)
	job, err := parser.Cron(cronExpr).Do(func() {})
	if err != nil {
		return nil, err
	}

	nextRun := job.NextRun()
	return &nextRun, nil
}
