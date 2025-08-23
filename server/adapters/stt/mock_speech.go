package stt

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/domain/repositories"
)

// MockSpeechToText is a placeholder implementation for speech recognition
type MockSpeechToText struct {
	logger *zap.Logger
}

// MockSpeechToTextStream is a mock implementation of streaming speech recognition
type MockSpeechToTextStream struct {
	logger        *zap.Logger
	audioReceived bool
	transcription string
}

// NewMockSpeechToText creates a new mock speech-to-text service
func NewMockSpeechToText(logger *zap.Logger) repositories.SpeechToText {
	return &MockSpeechToText{
		logger: logger,
	}
}

// InitTranscribeStreaming creates a new mock streaming session
func (s *MockSpeechToText) InitTranscribeStreaming(ctx context.Context, config repositories.AudioConfig) (repositories.SpeechToTextStreaming, error) {
	s.logger.Info("Initializing mock streaming transcription",
		zap.Int("sampleRate", config.SampleRate),
		zap.String("encoding", config.Encoding),
		zap.String("language", config.Language))

	return &MockSpeechToTextStream{
		logger:        s.logger,
		audioReceived: false,
		transcription: "",
	}, nil
}

// Stream implements mock streaming audio processing
func (m *MockSpeechToTextStream) Stream(data []byte) error {
	m.logger.Info("Processing mock audio chunk", zap.Int("size", len(data)))

	if len(data) > 0 {
		m.audioReceived = true
		// Mock different responses based on cumulative audio size
		switch {
		case len(data) > 10000:
			m.transcription = "Halo Arunika, apa kabar? Saya ingin bercerita tentang hari ini."
		case len(data) > 5000:
			m.transcription = "Terima kasih sudah mendengarkan."
		case len(data) > 1000:
			m.transcription = "Halo Arunika!"
		default:
			m.transcription = "Hai"
		}
	}

	return nil
}

// End returns the mock transcription result
func (m *MockSpeechToTextStream) End() (string, error) {
	m.logger.Info("Ending mock transcription stream", zap.String("result", m.transcription))

	if !m.audioReceived {
		return "", fmt.Errorf("no audio data received")
	}

	if m.transcription == "" {
		return "", fmt.Errorf("no speech detected in audio")
	}

	return m.transcription, nil
}

// TranscribeAudio implements repository.SpeechToText
func (s *MockSpeechToText) TranscribeAudio(ctx context.Context, audioData []byte, config repositories.AudioConfig) (string, error) {
	s.logger.Info("Processing speech-to-text",
		zap.Int("audioSize", len(audioData)),
		zap.Int("sampleRate", config.SampleRate),
		zap.String("encoding", config.Encoding))

	// Mock transcription based on audio size
	switch {
	case len(audioData) > 10000:
		return "Halo Arunika, apa kabar? Saya ingin bercerita tentang hari ini.", nil
	case len(audioData) > 5000:
		return "Terima kasih sudah mendengarkan.", nil
	case len(audioData) > 1000:
		return "Halo Arunika!", nil
	default:
		return "Hai", nil
	}
}
