// Package service provides business logic services for the container registry.
package service

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AutomationEngine provides automated task scheduling and execution.
type AutomationEngine struct {
	tasks      map[string]*ScheduledTask
	running    map[string]context.CancelFunc
	logger     *zap.Logger
	mu         sync.RWMutex
	isRunning  bool
	stopCh     chan struct{}
}

// ScheduledTask represents a scheduled automation task.
type ScheduledTask struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schedule    string                 `json:"schedule"` // cron expression
	Enabled     bool                   `json:"enabled"`
	TaskType    string                 `json:"task_type"`
	Config      map[string]interface{} `json:"config"`
	LastRun     time.Time              `json:"last_run"`
	NextRun     time.Time              `json:"next_run"`
	LastStatus  string                 `json:"last_status"`
	LastError   string                 `json:"last_error,omitempty"`
	RunCount    int64                  `json:"run_count"`
	FailCount   int64                  `json:"fail_count"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// TaskResult represents the result of a task execution.
type TaskResult struct {
	TaskID    string        `json:"task_id"`
	Success   bool          `json:"success"`
	Message   string        `json:"message"`
	Duration  time.Duration `json:"duration"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// AutomationConfig holds automation engine configuration.
type AutomationConfig struct {
	Enabled       bool
	MaxConcurrent int
	RetryAttempts int
	RetryDelay    time.Duration
}

// NewAutomationEngine creates a new AutomationEngine instance.
func NewAutomationEngine(config *AutomationConfig, logger *zap.Logger) *AutomationEngine {
	if config == nil {
		config = &AutomationConfig{
			Enabled:       true,
			MaxConcurrent: 5,
			RetryAttempts: 3,
			RetryDelay:    time.Minute,
		}
	}

	return &AutomationEngine{
		tasks:   make(map[string]*ScheduledTask),
		running: make(map[string]context.CancelFunc),
		logger:  logger,
		stopCh:  make(chan struct{}),
	}
}

// Start starts the automation engine.
func (e *AutomationEngine) Start() error {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return nil
	}
	e.isRunning = true
	e.mu.Unlock()

	// Register default tasks
	e.registerDefaultTasks()

	// Start scheduler
	go e.scheduler()

	if e.logger != nil {
		e.logger.Info("Automation engine started")
	}

	return nil
}

// Stop stops the automation engine.
func (e *AutomationEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return
	}

	// Cancel all running tasks
	for _, cancel := range e.running {
		cancel()
	}

	close(e.stopCh)
	e.isRunning = false

	if e.logger != nil {
		e.logger.Info("Automation engine stopped")
	}
}

// RegisterTask registers a new scheduled task.
func (e *AutomationEngine) RegisterTask(task *ScheduledTask) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.NextRun = e.calculateNextRun(task.Schedule)

	e.tasks[task.ID] = task

	if e.logger != nil {
		e.logger.Info("Task registered",
			zap.String("task_id", task.ID),
			zap.String("name", task.Name),
			zap.String("schedule", task.Schedule),
		)
	}

	return nil
}

// UnregisterTask removes a scheduled task.
func (e *AutomationEngine) UnregisterTask(taskID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Cancel if running
	if cancel, ok := e.running[taskID]; ok {
		cancel()
		delete(e.running, taskID)
	}

	delete(e.tasks, taskID)

	return nil
}

// GetTask returns a task by ID.
func (e *AutomationEngine) GetTask(taskID string) (*ScheduledTask, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	task, ok := e.tasks[taskID]
	return task, ok
}

// ListTasks returns all registered tasks.
func (e *AutomationEngine) ListTasks() []*ScheduledTask {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tasks := make([]*ScheduledTask, 0, len(e.tasks))
	for _, task := range e.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// RunTask manually triggers a task execution.
func (e *AutomationEngine) RunTask(taskID string) (*TaskResult, error) {
	e.mu.RLock()
	task, ok := e.tasks[taskID]
	e.mu.RUnlock()

	if !ok {
		return nil, ErrTaskNotFound
	}

	return e.executeTask(task)
}

// EnableTask enables a task.
func (e *AutomationEngine) EnableTask(taskID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	task, ok := e.tasks[taskID]
	if !ok {
		return ErrTaskNotFound
	}

	task.Enabled = true
	task.UpdatedAt = time.Now()
	task.NextRun = e.calculateNextRun(task.Schedule)

	return nil
}

// DisableTask disables a task.
func (e *AutomationEngine) DisableTask(taskID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	task, ok := e.tasks[taskID]
	if !ok {
		return ErrTaskNotFound
	}

	task.Enabled = false
	task.UpdatedAt = time.Now()

	return nil
}

// scheduler is the main scheduling loop.
func (e *AutomationEngine) scheduler() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.checkAndRunTasks()
		}
	}
}

// checkAndRunTasks checks for tasks that need to run.
func (e *AutomationEngine) checkAndRunTasks() {
	e.mu.RLock()
	tasks := make([]*ScheduledTask, 0)
	now := time.Now()

	for _, task := range e.tasks {
		if task.Enabled && !task.NextRun.IsZero() && now.After(task.NextRun) {
			tasks = append(tasks, task)
		}
	}
	e.mu.RUnlock()

	for _, task := range tasks {
		go func(t *ScheduledTask) {
			e.executeTask(t)
		}(task)
	}
}

// executeTask executes a single task.
func (e *AutomationEngine) executeTask(task *ScheduledTask) (*TaskResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Track running task
	e.mu.Lock()
	e.running[task.ID] = cancel
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.running, task.ID)
		e.mu.Unlock()
	}()

	start := time.Now()
	result := &TaskResult{
		TaskID:    task.ID,
		Timestamp: start,
	}

	// Execute based on task type
	var err error
	switch task.TaskType {
	case "cleanup":
		err = e.runCleanupTask(ctx, task)
	case "sync":
		err = e.runSyncTask(ctx, task)
	case "scan":
		err = e.runScanTask(ctx, task)
	case "backup":
		err = e.runBackupTask(ctx, task)
	case "sign":
		err = e.runSignTask(ctx, task)
	case "sbom":
		err = e.runSBOMTask(ctx, task)
	default:
		err = ErrUnknownTaskType
	}

	result.Duration = time.Since(start)

	// Update task status
	e.mu.Lock()
	task.LastRun = start
	task.RunCount++
	task.NextRun = e.calculateNextRun(task.Schedule)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		task.LastStatus = "failed"
		task.LastError = err.Error()
		task.FailCount++
	} else {
		result.Success = true
		result.Message = "Task completed successfully"
		task.LastStatus = "success"
		task.LastError = ""
	}
	task.UpdatedAt = time.Now()
	e.mu.Unlock()

	if e.logger != nil {
		if result.Success {
			e.logger.Info("Task completed",
				zap.String("task_id", task.ID),
				zap.Duration("duration", result.Duration),
			)
		} else {
			e.logger.Error("Task failed",
				zap.String("task_id", task.ID),
				zap.Error(err),
			)
		}
	}

	return result, err
}

// registerDefaultTasks registers default automation tasks.
func (e *AutomationEngine) registerDefaultTasks() {
	// Storage cleanup task
	e.RegisterTask(&ScheduledTask{
		ID:          "cleanup-storage",
		Name:        "Storage Cleanup",
		Description: "Clean up old images and cache",
		Schedule:    "0 2 * * *", // Daily at 2 AM
		Enabled:     true,
		TaskType:    "cleanup",
		Config: map[string]interface{}{
			"keep_days":  30,
			"keep_count": 5,
		},
	})

	// Vulnerability scan task
	e.RegisterTask(&ScheduledTask{
		ID:          "vuln-scan",
		Name:        "Vulnerability Scan",
		Description: "Scan images for vulnerabilities",
		Schedule:    "0 3 * * *", // Daily at 3 AM
		Enabled:     true,
		TaskType:    "scan",
		Config: map[string]interface{}{
			"scanner": "trivy",
		},
	})

	// SBOM generation task
	e.RegisterTask(&ScheduledTask{
		ID:          "sbom-generate",
		Name:        "SBOM Generation",
		Description: "Generate SBOM for new images",
		Schedule:    "0 4 * * *", // Daily at 4 AM
		Enabled:     true,
		TaskType:    "sbom",
		Config: map[string]interface{}{
			"format": "spdx-json",
		},
	})
}

// calculateNextRun calculates the next run time based on cron expression.
func (e *AutomationEngine) calculateNextRun(schedule string) time.Time {
	// Simplified cron parsing - in production use a proper cron library
	// Format: minute hour day month weekday
	now := time.Now()

	// Default to next day at the same time
	return now.Add(24 * time.Hour)
}

// Task execution implementations
func (e *AutomationEngine) runCleanupTask(ctx context.Context, task *ScheduledTask) error {
	// Implementation for cleanup task
	if e.logger != nil {
		e.logger.Info("Running cleanup task", zap.String("task_id", task.ID))
	}
	return nil
}

func (e *AutomationEngine) runSyncTask(ctx context.Context, task *ScheduledTask) error {
	// Implementation for sync task
	if e.logger != nil {
		e.logger.Info("Running sync task", zap.String("task_id", task.ID))
	}
	return nil
}

func (e *AutomationEngine) runScanTask(ctx context.Context, task *ScheduledTask) error {
	// Implementation for vulnerability scan task
	if e.logger != nil {
		e.logger.Info("Running scan task", zap.String("task_id", task.ID))
	}
	return nil
}

func (e *AutomationEngine) runBackupTask(ctx context.Context, task *ScheduledTask) error {
	// Implementation for backup task
	if e.logger != nil {
		e.logger.Info("Running backup task", zap.String("task_id", task.ID))
	}
	return nil
}

func (e *AutomationEngine) runSignTask(ctx context.Context, task *ScheduledTask) error {
	// Implementation for auto-sign task
	if e.logger != nil {
		e.logger.Info("Running sign task", zap.String("task_id", task.ID))
	}
	return nil
}

func (e *AutomationEngine) runSBOMTask(ctx context.Context, task *ScheduledTask) error {
	// Implementation for SBOM generation task
	if e.logger != nil {
		e.logger.Info("Running SBOM task", zap.String("task_id", task.ID))
	}
	return nil
}

// Error definitions
var (
	ErrTaskNotFound    = &TaskError{Message: "task not found"}
	ErrUnknownTaskType = &TaskError{Message: "unknown task type"}
)

// TaskError represents a task-related error.
type TaskError struct {
	Message string
}

func (e *TaskError) Error() string {
	return e.Message
}
