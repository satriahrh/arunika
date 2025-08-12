package domain

// AudioChunkMessage represents an incoming audio chunk from device
type AudioChunkMessage struct {
	Type       string `json:"type"`
	DeviceID   string `json:"device_id"`
	SessionID  string `json:"session_id"`
	AudioData  string `json:"audio_data"` // base64 encoded
	SampleRate int    `json:"sample_rate"`
	Encoding   string `json:"encoding"`
	Timestamp  string `json:"timestamp"`
	ChunkSeq   int    `json:"chunk_sequence"`
	IsFinal    bool   `json:"is_final"`
}

// AIResponseMessage represents a response from the AI system
type AIResponseMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Text      string `json:"response_text"`
	AudioData string `json:"audio_data"` // base64 encoded
	Emotion   string `json:"emotion"`
	Timestamp string `json:"timestamp"`
}

// TranscriptionMessage represents a transcription result
type TranscriptionMessage struct {
	SessionID string `json:"session_id"`
	UserID    int    `json:"user_id"`
	DeviceID  string `json:"device_id"`
	Text      string `json:"text"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}
