// Package service provides business logic services for the container registry.
package service

import (
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// WorkflowService provides workflow management services.
type WorkflowService struct {
	workflows  sync.Map // map[string]*Workflow
	jobs       sync.Map // map[string]*Job
	logger     *zap.Logger
	isPaused   bool
	mu         sync.RWMutex
}

// Workflow represents an automated workflow.
type Workflow struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Trigger     WorkflowTrigger   `json:"trigger"`
	Steps       []WorkflowStep    `json:"steps"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	LastRunAt   time.Time         `json:"last_run_at,omitempty"`
	LastStatus  string            `json:"last_status,omitempty"`
}

// WorkflowTrigger defines when a workflow should run.
type WorkflowTrigger struct {
	Type     string            `json:"type"` // schedule, event, manual
	Schedule string            `json:"schedule,omitempty"` // cron expression
	Event    string            `json:"event,omitempty"` // push, pull, delete
	Filter   map[string]string `json:"filter,omitempty"`
}

// WorkflowStep represents a step in a workflow.
type WorkflowStep struct {
	Name       string            `json:"name"`
	Action     string            `json:"action"` // sign, scan, notify, cleanup, sync
	Parameters map[string]string `json:"parameters,omitempty"`
	OnFailure  string            `json:"on_failure,omitempty"` // continue, stop, retry
	Timeout    string            `json:"timeout,omitempty"`
}

// Job represents a running workflow job.
type Job struct {
	ID          string       `json:"id"`
	WorkflowID  string       `json:"workflow_id"`
	Status      string       `json:"status"` // pending, running, completed, failed, cancelled
	StartedAt   time.Time    `json:"started_at"`
	CompletedAt time.Time    `json:"completed_at,omitempty"`
	Steps       []JobStep    `json:"steps"`
	Error       string       `json:"error,omitempty"`
	Logs        []string     `json:"logs,omitempty"`
}

// JobStep represents a step execution in a job.
type JobStep struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Output      string    `json:"output,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// CreateWorkflowRequest represents a request to create a workflow.
type CreateWorkflowRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description,omitempty"`
	Trigger     WorkflowTrigger `json:"trigger" binding:"required"`
	Steps       []WorkflowStep  `json:"steps" binding:"required"`
}

// NewWorkflowService creates a new WorkflowService instance.
func NewWorkflowService(logger *zap.Logger) *WorkflowService {
	return &WorkflowService{
		logger: logger,
	}
}

// CreateWorkflow creates a new workflow.
func (s *WorkflowService) CreateWorkflow(req *CreateWorkflowRequest) (*Workflow, error) {
	workflow := &Workflow{
		ID:          generateID(),
		Name:        req.Name,
		Description: req.Description,
		Trigger:     req.Trigger,
		Steps:       req.Steps,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.workflows.Store(workflow.ID, workflow)

	if s.logger != nil {
		s.logger.Info("Workflow created",
			zap.String("id", workflow.ID),
			zap.String("name", workflow.Name),
		)
	}

	return workflow, nil
}

// GetWorkflow retrieves a workflow by ID.
func (s *WorkflowService) GetWorkflow(id string) (*Workflow, error) {
	workflow, ok := s.workflows.Load(id)
	if !ok {
		return nil, errors.New("workflow not found")
	}
	return workflow.(*Workflow), nil
}

// ListWorkflows lists all workflows.
func (s *WorkflowService) ListWorkflows() ([]*Workflow, error) {
	var workflows []*Workflow

	s.workflows.Range(func(key, value interface{}) bool {
		workflows = append(workflows, value.(*Workflow))
		return true
	})

	return workflows, nil
}

// UpdateWorkflow updates a workflow.
func (s *WorkflowService) UpdateWorkflow(id string, req *CreateWorkflowRequest) (*Workflow, error) {
	existing, ok := s.workflows.Load(id)
	if !ok {
		return nil, errors.New("workflow not found")
	}

	workflow := existing.(*Workflow)
	workflow.Name = req.Name
	workflow.Description = req.Description
	workflow.Trigger = req.Trigger
	workflow.Steps = req.Steps
	workflow.UpdatedAt = time.Now()

	s.workflows.Store(id, workflow)

	return workflow, nil
}

// DeleteWorkflow deletes a workflow.
func (s *WorkflowService) DeleteWorkflow(id string) error {
	s.workflows.Delete(id)
	return nil
}

// EnableWorkflow enables a workflow.
func (s *WorkflowService) EnableWorkflow(id string) error {
	workflow, ok := s.workflows.Load(id)
	if !ok {
		return errors.New("workflow not found")
	}

	w := workflow.(*Workflow)
	w.Enabled = true
	w.UpdatedAt = time.Now()

	return nil
}

// DisableWorkflow disables a workflow.
func (s *WorkflowService) DisableWorkflow(id string) error {
	workflow, ok := s.workflows.Load(id)
	if !ok {
		return errors.New("workflow not found")
	}

	w := workflow.(*Workflow)
	w.Enabled = false
	w.UpdatedAt = time.Now()

	return nil
}

// TriggerWorkflow manually triggers a workflow.
func (s *WorkflowService) TriggerWorkflow(id string) (*Job, error) {
	s.mu.RLock()
	if s.isPaused {
		s.mu.RUnlock()
		return nil, errors.New("workflow service is paused")
	}
	s.mu.RUnlock()

	workflow, ok := s.workflows.Load(id)
	if !ok {
		return nil, errors.New("workflow not found")
	}

	w := workflow.(*Workflow)
	if !w.Enabled {
		return nil, errors.New("workflow is disabled")
	}

	// Create job
	job := &Job{
		ID:         generateID(),
		WorkflowID: id,
		Status:     "pending",
		StartedAt:  time.Now(),
		Steps:      make([]JobStep, len(w.Steps)),
	}

	for i, step := range w.Steps {
		job.Steps[i] = JobStep{
			Name:   step.Name,
			Status: "pending",
		}
	}

	s.jobs.Store(job.ID, job)

	// Execute job asynchronously
	go s.executeJob(job, w)

	return job, nil
}

// GetJob retrieves a job by ID.
func (s *WorkflowService) GetJob(id string) (*Job, error) {
	job, ok := s.jobs.Load(id)
	if !ok {
		return nil, errors.New("job not found")
	}
	return job.(*Job), nil
}

// ListJobs lists all jobs.
func (s *WorkflowService) ListJobs(workflowID string) ([]*Job, error) {
	var jobs []*Job

	s.jobs.Range(func(key, value interface{}) bool {
		job := value.(*Job)
		if workflowID == "" || job.WorkflowID == workflowID {
			jobs = append(jobs, job)
		}
		return true
	})

	return jobs, nil
}

// CancelJob cancels a running job.
func (s *WorkflowService) CancelJob(id string) error {
	job, ok := s.jobs.Load(id)
	if !ok {
		return errors.New("job not found")
	}

	j := job.(*Job)
	if j.Status != "running" && j.Status != "pending" {
		return errors.New("job is not running")
	}

	j.Status = "cancelled"
	j.CompletedAt = time.Now()

	return nil
}

// PauseAll pauses all workflows.
func (s *WorkflowService) PauseAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isPaused = true

	if s.logger != nil {
		s.logger.Info("All workflows paused")
	}
}

// ResumeAll resumes all workflows.
func (s *WorkflowService) ResumeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isPaused = false

	if s.logger != nil {
		s.logger.Info("All workflows resumed")
	}
}

// IsPaused returns whether workflows are paused.
func (s *WorkflowService) IsPaused() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isPaused
}

// executeJob executes a workflow job.
func (s *WorkflowService) executeJob(job *Job, workflow *Workflow) {
	job.Status = "running"

	for i, step := range workflow.Steps {
		// Check if paused
		s.mu.RLock()
		if s.isPaused {
			s.mu.RUnlock()
			job.Status = "cancelled"
			job.Error = "workflow service paused"
			job.CompletedAt = time.Now()
			return
		}
		s.mu.RUnlock()

		// Check if cancelled
		if job.Status == "cancelled" {
			return
		}

		// Execute step
		job.Steps[i].Status = "running"
		job.Steps[i].StartedAt = time.Now()

		err := s.executeStep(&step)

		job.Steps[i].CompletedAt = time.Now()

		if err != nil {
			job.Steps[i].Status = "failed"
			job.Steps[i].Error = err.Error()

			if step.OnFailure != "continue" {
				job.Status = "failed"
				job.Error = err.Error()
				job.CompletedAt = time.Now()
				return
			}
		} else {
			job.Steps[i].Status = "completed"
		}
	}

	job.Status = "completed"
	job.CompletedAt = time.Now()

	// Update workflow last run
	workflow.LastRunAt = time.Now()
	workflow.LastStatus = job.Status
}

// executeStep executes a single workflow step.
func (s *WorkflowService) executeStep(step *WorkflowStep) error {
	if s.logger != nil {
		s.logger.Info("Executing step",
			zap.String("name", step.Name),
			zap.String("action", step.Action),
		)
	}

	// Simulate step execution
	time.Sleep(100 * time.Millisecond)

	switch step.Action {
	case "sign":
		// Sign image
		return nil
	case "scan":
		// Scan for vulnerabilities
		return nil
	case "notify":
		// Send notification
		return nil
	case "cleanup":
		// Cleanup old images
		return nil
	case "sync":
		// Sync images
		return nil
	default:
		return errors.New("unknown action: " + step.Action)
	}
}

// generateID generates a unique ID.
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
