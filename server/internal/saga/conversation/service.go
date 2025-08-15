package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/internal/ai"
	"github.com/satriahrh/arunika/server/domain"
	"github.com/satriahrh/arunika/server/internal/saga"
)

// Service handles conversation processing using saga pattern
type Service struct {
	sagaManager *saga.Manager
	aiService   *ai.AIService
	logger      *zap.Logger
}

// NewService creates a new conversation service
func NewService(sagaManager *saga.Manager, aiService *ai.AIService, logger *zap.Logger) *Service {
	service := &Service{
		sagaManager: sagaManager,
		aiService:   aiService,
		logger:      logger,
	}

	// Register conversation saga definition
	sagaDef := NewConversationSagaDefinition(aiService, logger)
	sagaManager.RegisterDefinition(sagaDef)

	return service
}

// ProcessAudioChunk processes an audio chunk using the conversation saga
func (s *Service) ProcessAudioChunk(ctx context.Context, msg *domain.AudioChunkMessage) (*domain.AIResponseMessage, error) {
	s.logger.Info("Processing audio chunk with saga",
		zap.String("deviceID", msg.DeviceID),
		zap.String("sessionID", msg.SessionID))

	// Prepare saga data
	sagaData := saga.SagaData{
		DataKeyDeviceID:  msg.DeviceID,
		DataKeySessionID: msg.SessionID,
		DataKeyAudioData: msg.AudioData,
		DataKeyContext: map[string]interface{}{
			"sample_rate": msg.SampleRate,
			"encoding":    msg.Encoding,
			"timestamp":   msg.Timestamp,
			"chunk_seq":   msg.ChunkSeq,
			"is_final":    msg.IsFinal,
		},
	}

	// Start the conversation saga
	sagaID, err := s.sagaManager.StartSaga(ctx, "conversation_processing", sagaData)
	if err != nil {
		return nil, fmt.Errorf("failed to start conversation saga: %w", err)
	}

	// Wait for saga completion (or timeout)
	result, err := s.waitForSagaCompletion(ctx, sagaID, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("saga execution failed: %w", err)
	}

	// Build response message from saga result
	response := &domain.AIResponseMessage{
		Type:      "ai_response",
		SessionID: msg.SessionID,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Extract data from saga result
	if transcription, ok := result.Data[DataKeyTranscription].(string); ok {
		// Store transcription for debugging/logging
		s.logger.Debug("Transcription", zap.String("text", transcription))
	}

	if llmResponse, ok := result.Data[DataKeyLLMResponse].(string); ok {
		response.Text = llmResponse
	}

	if audioData, ok := result.Data[DataKeyResponseAudio].(string); ok {
		response.AudioData = audioData
	}

	// Default emotion (could be extracted from LLM response in the future)
	response.Emotion = "friendly"

	s.logger.Info("Audio chunk processed successfully",
		zap.String("deviceID", msg.DeviceID),
		zap.String("sessionID", msg.SessionID),
		zap.String("sagaID", string(sagaID)))

	return response, nil
}

// waitForSagaCompletion waits for a saga to complete and returns the result
func (s *Service) waitForSagaCompletion(ctx context.Context, sagaID saga.SagaID, timeout time.Duration) (*saga.SagaInstance, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Poll for completion
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for saga completion")

		case <-ticker.C:
			instance, exists := s.sagaManager.GetSaga(sagaID)
			if !exists {
				return nil, fmt.Errorf("saga not found: %s", sagaID)
			}

			switch instance.State {
			case saga.SagaStateCompleted:
				return instance, nil
			case saga.SagaStateFailed, saga.SagaStateCompensated:
				return nil, fmt.Errorf("saga failed with state: %s, error: %s", instance.State, instance.Error)
			}
			// Continue waiting for running/started states
		}
	}
}

// GetSagaStatus returns the current status of a saga
func (s *Service) GetSagaStatus(sagaID saga.SagaID) (*SagaStatusResponse, error) {
	instance, exists := s.sagaManager.GetSaga(sagaID)
	if !exists {
		return nil, fmt.Errorf("saga not found: %s", sagaID)
	}

	return &SagaStatusResponse{
		SagaID:      string(sagaID),
		State:       string(instance.State),
		Steps:       convertStepExecutions(instance.Steps),
		StartedAt:   instance.StartedAt,
		CompletedAt: instance.CompletedAt,
		Error:       instance.Error,
	}, nil
}

// SagaStatusResponse represents the status of a saga
type SagaStatusResponse struct {
	SagaID      string                `json:"saga_id"`
	State       string                `json:"state"`
	Steps       []StepExecutionStatus `json:"steps"`
	StartedAt   time.Time             `json:"started_at"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
	Error       string                `json:"error,omitempty"`
}

// StepExecutionStatus represents the status of a step execution
type StepExecutionStatus struct {
	ID          string      `json:"id"`
	State       string      `json:"state"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	Error       string      `json:"error,omitempty"`
	Result      interface{} `json:"result,omitempty"`
}

// convertStepExecutions converts saga step executions to response format
func convertStepExecutions(steps []saga.StepExecution) []StepExecutionStatus {
	result := make([]StepExecutionStatus, len(steps))
	for i, step := range steps {
		result[i] = StepExecutionStatus{
			ID:          string(step.ID),
			State:       string(step.State),
			StartedAt:   step.StartedAt,
			CompletedAt: step.CompletedAt,
			Error:       step.Error,
			Result:      step.Result,
		}
	}
	return result
}

// StartEventListener starts listening to saga events for monitoring
func (s *Service) StartEventListener(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-s.sagaManager.EventChannel():
				s.handleSagaEvent(event)
			}
		}
	}()
}

// handleSagaEvent handles saga events for logging and monitoring
func (s *Service) handleSagaEvent(event saga.SagaEvent) {
	eventData, _ := json.Marshal(event)

	switch event.Type {
	case saga.EventSagaStarted:
		s.logger.Info("Saga started", zap.String("sagaID", string(event.SagaID)))
	case saga.EventSagaCompleted:
		s.logger.Info("Saga completed", zap.String("sagaID", string(event.SagaID)))
	case saga.EventSagaFailed:
		s.logger.Error("Saga failed", zap.String("sagaID", string(event.SagaID)), zap.ByteString("event", eventData))
	case saga.EventStepStarted:
		s.logger.Debug("Step started", zap.String("sagaID", string(event.SagaID)), zap.String("stepID", string(event.StepID)))
	case saga.EventStepCompleted:
		s.logger.Debug("Step completed", zap.String("sagaID", string(event.SagaID)), zap.String("stepID", string(event.StepID)))
	case saga.EventStepFailed:
		s.logger.Warn("Step failed", zap.String("sagaID", string(event.SagaID)), zap.String("stepID", string(event.StepID)), zap.ByteString("event", eventData))
	default:
		s.logger.Debug("Saga event", zap.String("type", event.Type), zap.ByteString("event", eventData))
	}
}
