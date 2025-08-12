package saga

import (
	"context"
	"time"
)

// SagaState represents the current state of a saga execution
type SagaState string

const (
	SagaStateStarted     SagaState = "started"
	SagaStateRunning     SagaState = "running"
	SagaStateCompleted   SagaState = "completed"
	SagaStateFailed      SagaState = "failed"
	SagaStateCompensated SagaState = "compensated"
)

// StepState represents the state of an individual step
type StepState string

const (
	StepStatePending     StepState = "pending"
	StepStateRunning     StepState = "running"
	StepStateCompleted   StepState = "completed"
	StepStateFailed      StepState = "failed"
	StepStateCompensated StepState = "compensated"
)

// SagaID uniquely identifies a saga instance
type SagaID string

// StepID uniquely identifies a step within a saga
type StepID string

// SagaData holds the shared data for a saga execution
type SagaData map[string]interface{}

// StepResult represents the result of a step execution
type StepResult struct {
	Success bool
	Data    interface{}
	Error   error
}

// Step represents a single step in a saga
type Step interface {
	ID() StepID
	Execute(ctx context.Context, data SagaData) StepResult
	Compensate(ctx context.Context, data SagaData) error
}

// SagaDefinition defines the steps and flow of a saga
type SagaDefinition interface {
	ID() string
	Steps() []Step
	Timeout() time.Duration
}

// SagaInstance represents a running instance of a saga
type SagaInstance struct {
	ID          SagaID          `json:"id"`
	Definition  string          `json:"definition"`
	State       SagaState       `json:"state"`
	Data        SagaData        `json:"data"`
	Steps       []StepExecution `json:"steps"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Error       string          `json:"error,omitempty"`
}

// StepExecution represents the execution state of a step
type StepExecution struct {
	ID          StepID      `json:"id"`
	State       StepState   `json:"state"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	Error       string      `json:"error,omitempty"`
	Result      interface{} `json:"result,omitempty"`
}

// SagaEvent represents an event in the saga lifecycle
type SagaEvent struct {
	SagaID    SagaID      `json:"saga_id"`
	StepID    StepID      `json:"step_id,omitempty"`
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// Event types
const (
	EventSagaStarted     = "saga_started"
	EventSagaCompleted   = "saga_completed"
	EventSagaFailed      = "saga_failed"
	EventSagaCompensated = "saga_compensated"
	EventStepStarted     = "step_started"
	EventStepCompleted   = "step_completed"
	EventStepFailed      = "step_failed"
	EventStepCompensated = "step_compensated"
)
