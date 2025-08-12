package saga

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager manages saga execution and coordination
type Manager struct {
	logger      *zap.Logger
	instances   map[SagaID]*SagaInstance
	definitions map[string]SagaDefinition
	eventChan   chan SagaEvent
	mu          sync.RWMutex
}

// NewManager creates a new saga manager
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		logger:      logger,
		instances:   make(map[SagaID]*SagaInstance),
		definitions: make(map[string]SagaDefinition),
		eventChan:   make(chan SagaEvent, 100),
	}
}

// RegisterDefinition registers a saga definition
func (m *Manager) RegisterDefinition(def SagaDefinition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.definitions[def.ID()] = def
	m.logger.Info("Saga definition registered", zap.String("id", def.ID()))
}

// StartSaga starts a new saga instance
func (m *Manager) StartSaga(ctx context.Context, definitionID string, data SagaData) (SagaID, error) {
	m.mu.Lock()
	def, exists := m.definitions[definitionID]
	if !exists {
		m.mu.Unlock()
		return "", fmt.Errorf("saga definition not found: %s", definitionID)
	}

	sagaID := SagaID(fmt.Sprintf("%s_%d", definitionID, time.Now().UnixNano()))

	// Initialize step executions
	stepExecs := make([]StepExecution, len(def.Steps()))
	for i, step := range def.Steps() {
		stepExecs[i] = StepExecution{
			ID:    step.ID(),
			State: StepStatePending,
		}
	}

	instance := &SagaInstance{
		ID:         sagaID,
		Definition: definitionID,
		State:      SagaStateStarted,
		Data:       data,
		Steps:      stepExecs,
		StartedAt:  time.Now(),
	}

	m.instances[sagaID] = instance
	m.mu.Unlock()

	// Emit saga started event
	m.emitEvent(SagaEvent{
		SagaID:    sagaID,
		Type:      EventSagaStarted,
		Timestamp: time.Now(),
		Data:      data,
	})

	// Start execution in a goroutine
	go m.executeSaga(ctx, sagaID, def)

	m.logger.Info("Saga started", zap.String("sagaID", string(sagaID)), zap.String("definition", definitionID))
	return sagaID, nil
}

// GetSaga returns a saga instance by ID
func (m *Manager) GetSaga(sagaID SagaID) (*SagaInstance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	instance, exists := m.instances[sagaID]
	return instance, exists
}

// executeSaga executes a saga instance
func (m *Manager) executeSaga(ctx context.Context, sagaID SagaID, def SagaDefinition) {
	m.updateSagaState(sagaID, SagaStateRunning)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, def.Timeout())
	defer cancel()

	// Execute steps sequentially
	compensationNeeded := false
	lastCompletedStep := -1

	for i, step := range def.Steps() {
		if err := m.executeStep(ctx, sagaID, i, step); err != nil {
			m.logger.Error("Step failed",
				zap.String("sagaID", string(sagaID)),
				zap.String("stepID", string(step.ID())),
				zap.Error(err))

			compensationNeeded = true
			break
		}
		lastCompletedStep = i
	}

	if compensationNeeded {
		m.logger.Info("Starting compensation", zap.String("sagaID", string(sagaID)))
		m.compensateSaga(ctx, sagaID, def, lastCompletedStep)
	} else {
		m.completeSaga(sagaID)
	}
}

// executeStep executes a single step
func (m *Manager) executeStep(ctx context.Context, sagaID SagaID, stepIndex int, step Step) error {
	m.updateStepState(sagaID, stepIndex, StepStateRunning)

	now := time.Now()
	m.setStepStartTime(sagaID, stepIndex, now)

	// Emit step started event
	m.emitEvent(SagaEvent{
		SagaID:    sagaID,
		StepID:    step.ID(),
		Type:      EventStepStarted,
		Timestamp: now,
	})

	// Get current saga data
	instance, _ := m.GetSaga(sagaID)

	// Execute the step
	result := step.Execute(ctx, instance.Data)

	if result.Success {
		// Update step result and complete
		m.setStepResult(sagaID, stepIndex, result.Data)
		m.updateStepState(sagaID, stepIndex, StepStateCompleted)

		now = time.Now()
		m.setStepCompletionTime(sagaID, stepIndex, now)

		// Emit step completed event
		m.emitEvent(SagaEvent{
			SagaID:    sagaID,
			StepID:    step.ID(),
			Type:      EventStepCompleted,
			Timestamp: now,
			Data:      result.Data,
		})

		m.logger.Info("Step completed",
			zap.String("sagaID", string(sagaID)),
			zap.String("stepID", string(step.ID())))

		return nil
	} else {
		// Handle step failure
		m.updateStepState(sagaID, stepIndex, StepStateFailed)
		m.setStepError(sagaID, stepIndex, result.Error.Error())

		now = time.Now()
		m.setStepCompletionTime(sagaID, stepIndex, now)

		// Emit step failed event
		m.emitEvent(SagaEvent{
			SagaID:    sagaID,
			StepID:    step.ID(),
			Type:      EventStepFailed,
			Timestamp: now,
			Data:      result.Error.Error(),
		})

		return result.Error
	}
}

// compensateSaga runs compensation for completed steps in reverse order
func (m *Manager) compensateSaga(ctx context.Context, sagaID SagaID, def SagaDefinition, lastCompletedStep int) {
	instance, _ := m.GetSaga(sagaID)
	steps := def.Steps()

	// Compensate in reverse order
	for i := lastCompletedStep; i >= 0; i-- {
		step := steps[i]

		m.logger.Info("Compensating step",
			zap.String("sagaID", string(sagaID)),
			zap.String("stepID", string(step.ID())))

		if err := step.Compensate(ctx, instance.Data); err != nil {
			m.logger.Error("Compensation failed",
				zap.String("sagaID", string(sagaID)),
				zap.String("stepID", string(step.ID())),
				zap.Error(err))
		} else {
			m.updateStepState(sagaID, i, StepStateCompensated)

			// Emit step compensated event
			m.emitEvent(SagaEvent{
				SagaID:    sagaID,
				StepID:    step.ID(),
				Type:      EventStepCompensated,
				Timestamp: time.Now(),
			})
		}
	}

	m.updateSagaState(sagaID, SagaStateCompensated)
	now := time.Now()
	m.setSagaCompletionTime(sagaID, now)

	// Emit saga compensated event
	m.emitEvent(SagaEvent{
		SagaID:    sagaID,
		Type:      EventSagaCompensated,
		Timestamp: now,
	})

	m.logger.Info("Saga compensated", zap.String("sagaID", string(sagaID)))
}

// completeSaga marks a saga as completed
func (m *Manager) completeSaga(sagaID SagaID) {
	m.updateSagaState(sagaID, SagaStateCompleted)
	now := time.Now()
	m.setSagaCompletionTime(sagaID, now)

	// Emit saga completed event
	m.emitEvent(SagaEvent{
		SagaID:    sagaID,
		Type:      EventSagaCompleted,
		Timestamp: now,
	})

	m.logger.Info("Saga completed", zap.String("sagaID", string(sagaID)))
}

// Helper methods for updating saga state
func (m *Manager) updateSagaState(sagaID SagaID, state SagaState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if instance, exists := m.instances[sagaID]; exists {
		instance.State = state
	}
}

func (m *Manager) updateStepState(sagaID SagaID, stepIndex int, state StepState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if instance, exists := m.instances[sagaID]; exists && stepIndex < len(instance.Steps) {
		instance.Steps[stepIndex].State = state
	}
}

func (m *Manager) setStepStartTime(sagaID SagaID, stepIndex int, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if instance, exists := m.instances[sagaID]; exists && stepIndex < len(instance.Steps) {
		instance.Steps[stepIndex].StartedAt = &t
	}
}

func (m *Manager) setStepCompletionTime(sagaID SagaID, stepIndex int, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if instance, exists := m.instances[sagaID]; exists && stepIndex < len(instance.Steps) {
		instance.Steps[stepIndex].CompletedAt = &t
	}
}

func (m *Manager) setStepResult(sagaID SagaID, stepIndex int, result interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if instance, exists := m.instances[sagaID]; exists && stepIndex < len(instance.Steps) {
		instance.Steps[stepIndex].Result = result
	}
}

func (m *Manager) setStepError(sagaID SagaID, stepIndex int, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if instance, exists := m.instances[sagaID]; exists && stepIndex < len(instance.Steps) {
		instance.Steps[stepIndex].Error = errMsg
	}
}

func (m *Manager) setSagaCompletionTime(sagaID SagaID, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if instance, exists := m.instances[sagaID]; exists {
		instance.CompletedAt = &t
	}
}

func (m *Manager) emitEvent(event SagaEvent) {
	select {
	case m.eventChan <- event:
	default:
		m.logger.Warn("Event channel full, dropping event", zap.String("type", event.Type))
	}
}

// EventChannel returns the event channel for listening to saga events
func (m *Manager) EventChannel() <-chan SagaEvent {
	return m.eventChan
}
