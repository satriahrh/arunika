package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/domain/repositories"
)

const (
	elevenLabsAPIBaseURL = "https://api.elevenlabs.io/v1"
	defaultVoiceID       = "21m00Tcm4TlvDq8ikWAM" // Rachel voice
	chunkSize            = 1024                   // Size of audio chunks to stream
	defaultOutputFormat  = "pcm_24000"            // PCM format for real-time applications
)

// ElevenLabsTTS implements TextToSpeech interface using Eleven Labs API
type ElevenLabsTTS struct {
	apiKey       string
	voiceID      string
	modelID      string
	outputFormat string
	stability    float64
	clarity      float64
	logger       *zap.Logger
}

// Ensure ElevenLabsTTS implements the TextToSpeech interface
var _ repositories.TextToSpeech = (*ElevenLabsTTS)(nil)

// ElevenLabsVoiceSettings represents voice settings for Eleven Labs API
type ElevenLabsVoiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
	Style           float64 `json:"style,omitempty"`
	UseSpeakerBoost bool    `json:"use_speaker_boost,omitempty"`
}

// ElevenLabsRequest represents the request payload for Eleven Labs TTS API
type ElevenLabsRequest struct {
	Text                   string                  `json:"text"`
	ModelID                string                  `json:"model_id"`
	LanguageCode           string                  `json:"language_code,omitempty"`
	VoiceSettings          ElevenLabsVoiceSettings `json:"voice_settings"`
	ApplyTextNormalization string                  `json:"apply_text_normalization,omitempty"`
}

// NewElevenLabsTTS creates a new Eleven Labs TTS instance
func NewElevenLabsTTS(logger *zap.Logger) (*ElevenLabsTTS, error) {
	apiKey := os.Getenv("ELEVEN_LABS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ELEVEN_LABS_API_KEY environment variable is required")
	}

	voiceID := os.Getenv("ELEVEN_LABS_VOICE_ID")
	if voiceID == "" {
		voiceID = defaultVoiceID
		logger.Info("Using default voice ID", zap.String("voiceID", voiceID))
	}

	modelID := os.Getenv("ELEVEN_LABS_MODEL_ID")
	if modelID == "" {
		modelID = "eleven_multilingual_v2"
		logger.Info("Using default model ID", zap.String("modelID", modelID))
	}

	outputFormat := os.Getenv("ELEVEN_LABS_OUTPUT_FORMAT")
	if outputFormat == "" {
		outputFormat = defaultOutputFormat
		logger.Info("Using default output format", zap.String("outputFormat", outputFormat))
	}

	return &ElevenLabsTTS{
		apiKey:       apiKey,
		voiceID:      voiceID,
		modelID:      modelID,
		outputFormat: outputFormat,
		stability:    0.5,  // Default stability
		clarity:      0.75, // Default clarity (similarity_boost)
		logger:       logger,
	}, nil
}

// ConvertTextToSpeech converts text to speech using Eleven Labs API
func (e *ElevenLabsTTS) ConvertTextToSpeech(ctx context.Context, text string) (<-chan []byte, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	e.logger.Info("Converting text to speech",
		zap.String("text", text),
		zap.String("voiceID", e.voiceID),
		zap.String("modelID", e.modelID))

	// Create request payload
	request := ElevenLabsRequest{
		Text:                   text,
		ModelID:                e.modelID,
		ApplyTextNormalization: "auto",
		VoiceSettings: ElevenLabsVoiceSettings{
			Stability:       e.stability,
			SimilarityBoost: e.clarity,
			Style:           0.0,
			UseSpeakerBoost: true,
		},
	}

	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request with streaming optimizations
	url := fmt.Sprintf("%s/text-to-speech/%s/stream?output_format=%s&enable_logging=false",
		elevenLabsAPIBaseURL, e.voiceID, e.outputFormat)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers - Note: PCM format requires audio/pcm accept header
	acceptHeader := "audio/mpeg"
	if strings.HasPrefix(e.outputFormat, "pcm") {
		acceptHeader = "audio/pcm"
	}
	httpReq.Header.Set("Accept", acceptHeader)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("xi-api-key", e.apiKey)

	// Create HTTP client with optimized timeout for streaming
	client := &http.Client{
		Timeout: 60 * time.Second, // Increased timeout for real-time streaming
	}

	// Create channel for streaming audio data
	audioChan := make(chan []byte, 10)

	// Execute request in goroutine to stream response
	go func() {
		defer close(audioChan)

		e.logger.Debug("Sending request to Eleven Labs API", zap.String("url", url))

		resp, err := client.Do(httpReq)
		if err != nil {
			e.logger.Error("Failed to execute HTTP request", zap.Error(err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// Read error response
			errorBody, _ := io.ReadAll(resp.Body)
			e.logger.Error("Eleven Labs API returned error",
				zap.Int("statusCode", resp.StatusCode),
				zap.String("response", string(errorBody)))
			return
		}

		e.logger.Info("Successfully received response from Eleven Labs API",
			zap.String("contentType", resp.Header.Get("Content-Type")),
			zap.String("contentLength", resp.Header.Get("Content-Length")))

		// Stream the audio data in chunks
		buffer := make([]byte, chunkSize)
		totalBytes := 0
		chunkCount := 0

		for {
			select {
			case <-ctx.Done():
				e.logger.Warn("Context cancelled while streaming audio data")
				return
			default:
				n, err := resp.Body.Read(buffer)
				if n > 0 {
					totalBytes += n
					chunkCount++

					// Create a copy of the data to send
					chunk := make([]byte, n)
					copy(chunk, buffer[:n])

					e.logger.Debug("Sending audio chunk",
						zap.Int("chunkNumber", chunkCount),
						zap.Int("chunkSize", n),
						zap.Int("totalBytes", totalBytes))

					select {
					case audioChan <- chunk:
					case <-ctx.Done():
						e.logger.Warn("Context cancelled while sending audio chunk")
						return
					}
				}

				if err == io.EOF {
					e.logger.Info("Finished streaming audio data",
						zap.Int("totalChunks", chunkCount),
						zap.Int("totalBytes", totalBytes))
					return
				}

				if err != nil {
					e.logger.Error("Error reading response body", zap.Error(err))
					return
				}
			}
		}
	}()

	return audioChan, nil
}

// SetVoiceSettings allows customization of voice parameters
func (e *ElevenLabsTTS) SetVoiceSettings(stability, clarity float64) {
	e.stability = stability
	e.clarity = clarity
	e.logger.Info("Updated voice settings",
		zap.Float64("stability", stability),
		zap.Float64("clarity", clarity))
}

// SetVoiceID allows changing the voice used for TTS
func (e *ElevenLabsTTS) SetVoiceID(voiceID string) {
	e.voiceID = voiceID
	e.logger.Info("Updated voice ID", zap.String("voiceID", voiceID))
}

// SetModelID allows changing the model used for TTS
func (e *ElevenLabsTTS) SetModelID(modelID string) {
	e.modelID = modelID
	e.logger.Info("Updated model ID", zap.String("modelID", modelID))
}

// SetOutputFormat allows changing the output format for streaming
func (e *ElevenLabsTTS) SetOutputFormat(format string) {
	e.outputFormat = format
	e.logger.Info("Updated output format", zap.String("outputFormat", format))
}

// GetAvailableVoices retrieves available voices from Eleven Labs API
func (e *ElevenLabsTTS) GetAvailableVoices(ctx context.Context) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/voices", elevenLabsAPIBaseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("xi-api-key", e.apiKey)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned error %d: %s", resp.StatusCode, string(errorBody))
	}

	var voicesResponse struct {
		Voices []map[string]interface{} `json:"voices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&voicesResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	e.logger.Info("Retrieved available voices", zap.Int("count", len(voicesResponse.Voices)))
	return voicesResponse.Voices, nil
}
