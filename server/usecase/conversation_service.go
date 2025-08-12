package usecase

import (
	"context"
	"encoding/base64"
	"fmt"

	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/domain"
	"github.com/satriahrh/arunika/server/repository"
)

// ConversationService orchestrates the conversation flow
type ConversationService struct {
	speechToText repository.SpeechToText
	textToSpeech repository.TextToSpeech
	chatService  *ChatService
	logger       *zap.Logger
}

// NewConversationService creates a new conversation service
func NewConversationService(
	stt repository.SpeechToText,
	tts repository.TextToSpeech,
	chatService *ChatService,
	logger *zap.Logger,
) *ConversationService {
	return &ConversationService{
		speechToText: stt,
		textToSpeech: tts,
		chatService:  chatService,
		logger:       logger,
	}
}

// ProcessAudioChunk processes an audio chunk and returns AI response
func (s *ConversationService) ProcessAudioChunk(ctx context.Context, msg *domain.AudioChunkMessage) (*domain.AIResponseMessage, error) {
	s.logger.Info("Processing audio chunk",
		zap.String("deviceID", msg.DeviceID),
		zap.String("sessionID", msg.SessionID),
		zap.Bool("isFinal", msg.IsFinal))

	// Decode base64 audio data
	audioData, err := base64.StdEncoding.DecodeString(msg.AudioData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio data: %w", err)
	}

	// Configure audio for transcription
	audioConfig := repository.AudioConfig{
		SampleRate: msg.SampleRate,
		Encoding:   msg.Encoding,
		Language:   "id-ID", // Indonesian
	}

	// Step 1: Speech to Text
	transcription, err := s.speechToText.TranscribeAudio(ctx, audioData, audioConfig)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	s.logger.Info("Transcription completed", zap.String("text", transcription))

	// Step 2: Generate AI response using chat service
	input := make(chan string, 1)
	output := make(chan string, 1)

	// Extract user ID from session (for now, use a default)
	userID := 1 // TODO: Extract from session/JWT

	// Start chat service in goroutine
	go func() {
		err := s.chatService.Execute(ctx, userID, input, output)
		if err != nil {
			s.logger.Error("Chat service error", zap.Error(err))
		}
	}()

	// Send transcription to chat service
	input <- transcription

	// Wait for response
	aiResponseText := <-output

	s.logger.Info("AI response generated", zap.String("response", aiResponseText))

	// Step 3: Text to Speech
	voiceConfig := repository.VoiceConfig{
		Voice:     "id-ID-ArdiNeural",
		Language:  "id-ID",
		Gender:    "female",
		SpeakRate: "medium",
	}

	audioResponse, err := s.textToSpeech.SynthesizeAudio(ctx, aiResponseText, voiceConfig)
	if err != nil {
		return nil, fmt.Errorf("text-to-speech failed: %w", err)
	}

	// Encode audio response as base64
	audioBase64 := base64.StdEncoding.EncodeToString(audioResponse)

	s.logger.Info("TTS completed", zap.Int("audioSize", len(audioResponse)))

	// Build response message
	response := &domain.AIResponseMessage{
		Type:      "ai_response",
		SessionID: msg.SessionID,
		Text:      aiResponseText,
		AudioData: audioBase64,
		Emotion:   "friendly",
		Timestamp: msg.Timestamp,
	}

	return response, nil
}
