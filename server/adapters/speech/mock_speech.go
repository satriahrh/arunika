package speech

import (
	"context"

	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/repository"
)

// MockSpeechToText is a placeholder implementation for speech recognition
type MockSpeechToText struct {
	logger *zap.Logger
}

// NewMockSpeechToText creates a new mock speech-to-text service
func NewMockSpeechToText(logger *zap.Logger) repository.SpeechToText {
	return &MockSpeechToText{
		logger: logger,
	}
}

// TranscribeAudio implements repository.SpeechToText
func (s *MockSpeechToText) TranscribeAudio(ctx context.Context, audioData []byte, config repository.AudioConfig) (string, error) {
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

// MockTextToSpeech is a placeholder implementation for text-to-speech
type MockTextToSpeech struct {
	logger *zap.Logger
}

// NewMockTextToSpeech creates a new mock text-to-speech service
func NewMockTextToSpeech(logger *zap.Logger) repository.TextToSpeech {
	return &MockTextToSpeech{
		logger: logger,
	}
}

// SynthesizeAudio implements repository.TextToSpeech
func (t *MockTextToSpeech) SynthesizeAudio(ctx context.Context, text string, config repository.VoiceConfig) ([]byte, error) {
	t.logger.Info("Processing text-to-speech",
		zap.String("text", text),
		zap.String("voice", config.Voice))

	// Mock audio data - generate based on text length
	audioSize := len(text) * 100 // Simulate audio size
	mockAudio := make([]byte, audioSize)

	// Fill with some pattern to simulate audio data
	for i := range mockAudio {
		mockAudio[i] = byte(i % 256)
	}

	return mockAudio, nil
}
