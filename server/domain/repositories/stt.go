package repositories

import "context"

// SpeechToText abstracts speech recognition services
type SpeechToText interface {
	// TranscribeAudio converts audio data to text
	TranscribeAudio(ctx context.Context, audioData []byte, config AudioConfig) (string, error)
	// InitTranscribeStreaming initializes a streaming transcription session
	InitTranscribeStreaming(ctx context.Context, config AudioConfig) (SpeechToTextStreaming, error)
}

// AudioConfig represents audio configuration for speech recognition
type AudioConfig struct {
	SampleRate int    `json:"sample_rate"`
	Encoding   string `json:"encoding"`
	Language   string `json:"language"`
}

type SpeechToTextStreaming interface {
	Stream(data []byte) error
	End() (string, error)
}
