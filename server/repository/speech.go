package repository

import "context"

// SpeechToText abstracts speech recognition services
type SpeechToText interface {
	// TranscribeAudio converts audio data to text
	TranscribeAudio(ctx context.Context, audioData []byte, config AudioConfig) (string, error)
}

// TextToSpeech abstracts text-to-speech services
type TextToSpeech interface {
	// SynthesizeAudio converts text to audio data
	SynthesizeAudio(ctx context.Context, text string, config VoiceConfig) ([]byte, error)
}

// AudioConfig represents audio configuration for speech recognition
type AudioConfig struct {
	SampleRate int    `json:"sample_rate"`
	Encoding   string `json:"encoding"`
	Language   string `json:"language"`
}

// VoiceConfig represents voice configuration for TTS
type VoiceConfig struct {
	Voice     string `json:"voice"`
	Language  string `json:"language"`
	Gender    string `json:"gender"`
	SpeakRate string `json:"speak_rate"`
}
