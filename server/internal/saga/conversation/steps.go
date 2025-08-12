package conversation

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/internal/ai"
	"github.com/satriahrh/arunika/server/internal/saga"
)

// ConversationSagaDefinition defines the conversation processing saga
type ConversationSagaDefinition struct {
	aiService *ai.AIService
	logger    *zap.Logger
}

// NewConversationSagaDefinition creates a new conversation saga definition
func NewConversationSagaDefinition(aiService *ai.AIService, logger *zap.Logger) *ConversationSagaDefinition {
	return &ConversationSagaDefinition{
		aiService: aiService,
		logger:    logger,
	}
}

func (d *ConversationSagaDefinition) ID() string {
	return "conversation_processing"
}

func (d *ConversationSagaDefinition) Timeout() time.Duration {
	return 30 * time.Second // 30 seconds timeout for entire conversation flow
}

func (d *ConversationSagaDefinition) Steps() []saga.Step {
	return []saga.Step{
		NewSpeechToTextStep(d.aiService, d.logger),
		NewContentValidationStep(d.aiService, d.logger),
		NewLLMProcessingStep(d.aiService, d.logger),
		NewTextToSpeechStep(d.aiService, d.logger),
	}
}

// Data keys for conversation saga
const (
	DataKeyDeviceID      = "device_id"
	DataKeySessionID     = "session_id"
	DataKeyAudioData     = "audio_data"
	DataKeyTranscription = "transcription"
	DataKeyIsContentSafe = "is_content_safe"
	DataKeyLLMResponse   = "llm_response"
	DataKeyResponseAudio = "response_audio"
	DataKeyContext       = "context"
)

// SpeechToTextStep converts audio to text
type SpeechToTextStep struct {
	aiService *ai.AIService
	logger    *zap.Logger
}

func NewSpeechToTextStep(aiService *ai.AIService, logger *zap.Logger) *SpeechToTextStep {
	return &SpeechToTextStep{
		aiService: aiService,
		logger:    logger,
	}
}

func (s *SpeechToTextStep) ID() saga.StepID {
	return "speech_to_text"
}

func (s *SpeechToTextStep) Execute(ctx context.Context, data saga.SagaData) saga.StepResult {
	s.logger.Info("Executing speech-to-text step")

	// Extract audio data
	audioDataStr, ok := data[DataKeyAudioData].(string)
	if !ok {
		return saga.StepResult{
			Success: false,
			Error:   fmt.Errorf("missing or invalid audio data"),
		}
	}

	// Decode base64 audio data
	audioData, err := base64.StdEncoding.DecodeString(audioDataStr)
	if err != nil {
		return saga.StepResult{
			Success: false,
			Error:   fmt.Errorf("failed to decode audio data: %w", err),
		}
	}

	// Process speech to text
	transcription, err := s.aiService.ProcessSpeechToText(audioData)
	if err != nil {
		return saga.StepResult{
			Success: false,
			Error:   fmt.Errorf("speech-to-text failed: %w", err),
		}
	}

	// Store transcription in saga data
	data[DataKeyTranscription] = transcription

	s.logger.Info("Speech-to-text completed", zap.String("transcription", transcription))

	return saga.StepResult{
		Success: true,
		Data:    transcription,
	}
}

func (s *SpeechToTextStep) Compensate(ctx context.Context, data saga.SagaData) error {
	// No compensation needed for STT - it's read-only
	s.logger.Info("Speech-to-text step compensation (no-op)")
	return nil
}

// ContentValidationStep validates content safety
type ContentValidationStep struct {
	aiService *ai.AIService
	logger    *zap.Logger
}

func NewContentValidationStep(aiService *ai.AIService, logger *zap.Logger) *ContentValidationStep {
	return &ContentValidationStep{
		aiService: aiService,
		logger:    logger,
	}
}

func (s *ContentValidationStep) ID() saga.StepID {
	return "content_validation"
}

func (s *ContentValidationStep) Execute(ctx context.Context, data saga.SagaData) saga.StepResult {
	s.logger.Info("Executing content validation step")

	// Get transcription from previous step
	transcription, ok := data[DataKeyTranscription].(string)
	if !ok {
		return saga.StepResult{
			Success: false,
			Error:   fmt.Errorf("missing transcription from previous step"),
		}
	}

	// Validate content safety
	isSafe, err := s.aiService.ValidateContent(transcription)
	if err != nil {
		return saga.StepResult{
			Success: false,
			Error:   fmt.Errorf("content validation failed: %w", err),
		}
	}

	if !isSafe {
		return saga.StepResult{
			Success: false,
			Error:   fmt.Errorf("content is not child-safe"),
		}
	}

	// Store validation result
	data[DataKeyIsContentSafe] = isSafe

	s.logger.Info("Content validation completed", zap.Bool("is_safe", isSafe))

	return saga.StepResult{
		Success: true,
		Data:    isSafe,
	}
}

func (s *ContentValidationStep) Compensate(ctx context.Context, data saga.SagaData) error {
	// No compensation needed for validation - it's read-only
	s.logger.Info("Content validation step compensation (no-op)")
	return nil
}

// LLMProcessingStep generates AI response
type LLMProcessingStep struct {
	aiService *ai.AIService
	logger    *zap.Logger
}

func NewLLMProcessingStep(aiService *ai.AIService, logger *zap.Logger) *LLMProcessingStep {
	return &LLMProcessingStep{
		aiService: aiService,
		logger:    logger,
	}
}

func (s *LLMProcessingStep) ID() saga.StepID {
	return "llm_processing"
}

func (s *LLMProcessingStep) Execute(ctx context.Context, data saga.SagaData) saga.StepResult {
	s.logger.Info("Executing LLM processing step")

	// Get transcription and context
	transcription, ok := data[DataKeyTranscription].(string)
	if !ok {
		return saga.StepResult{
			Success: false,
			Error:   fmt.Errorf("missing transcription from previous step"),
		}
	}

	// Build context for LLM
	context := make(map[string]interface{})
	if deviceID, exists := data[DataKeyDeviceID]; exists {
		context["device_id"] = deviceID
	}
	if sessionID, exists := data[DataKeySessionID]; exists {
		context["session_id"] = sessionID
	}
	if existingContext, exists := data[DataKeyContext]; exists {
		if ctx, ok := existingContext.(map[string]interface{}); ok {
			for k, v := range ctx {
				context[k] = v
			}
		}
	}

	// Generate AI response
	response, err := s.aiService.GenerateResponse(transcription, context)
	if err != nil {
		return saga.StepResult{
			Success: false,
			Error:   fmt.Errorf("LLM processing failed: %w", err),
		}
	}

	// Store response in saga data
	data[DataKeyLLMResponse] = response

	s.logger.Info("LLM processing completed", zap.String("response", response))

	return saga.StepResult{
		Success: true,
		Data:    response,
	}
}

func (s *LLMProcessingStep) Compensate(ctx context.Context, data saga.SagaData) error {
	// No compensation needed for LLM - it's stateless
	s.logger.Info("LLM processing step compensation (no-op)")
	return nil
}

// TextToSpeechStep converts text response to audio
type TextToSpeechStep struct {
	aiService *ai.AIService
	logger    *zap.Logger
}

func NewTextToSpeechStep(aiService *ai.AIService, logger *zap.Logger) *TextToSpeechStep {
	return &TextToSpeechStep{
		aiService: aiService,
		logger:    logger,
	}
}

func (s *TextToSpeechStep) ID() saga.StepID {
	return "text_to_speech"
}

func (s *TextToSpeechStep) Execute(ctx context.Context, data saga.SagaData) saga.StepResult {
	s.logger.Info("Executing text-to-speech step")

	// Get LLM response from previous step
	response, ok := data[DataKeyLLMResponse].(string)
	if !ok {
		return saga.StepResult{
			Success: false,
			Error:   fmt.Errorf("missing LLM response from previous step"),
		}
	}

	// Convert to speech
	audioData, err := s.aiService.ProcessTextToSpeech(response, "child_friendly")
	if err != nil {
		return saga.StepResult{
			Success: false,
			Error:   fmt.Errorf("text-to-speech failed: %w", err),
		}
	}

	// Encode audio as base64
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)

	// Store audio in saga data
	data[DataKeyResponseAudio] = audioBase64

	s.logger.Info("Text-to-speech completed", zap.Int("audio_size", len(audioData)))

	return saga.StepResult{
		Success: true,
		Data:    audioBase64,
	}
}

func (s *TextToSpeechStep) Compensate(ctx context.Context, data saga.SagaData) error {
	// No compensation needed for TTS - it's stateless
	s.logger.Info("Text-to-speech step compensation (no-op)")
	return nil
}
