package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/domain/repositories"
)

const (
	defaultAPIBaseURL   = "https://api.elevenlabs.io/v1"
	defaultVoiceID      = "21m00Tcm4TlvDq8ikWAM"   // Rachel voice
	defaultChunkSize    = 1024                     // Size of audio chunks to stream
	defaultOutputFormat = "pcm_24000"              // PCM format for real-time applications
	defaultModelID      = "eleven_multilingual_v2" // Default model ID
	defaultStability    = 0.5                      // Default voice stability
	defaultClarity      = 0.75                     // Default voice clarity/similarity_boost
)

// ElevenLabsConfig holds configuration for the ElevenLabsTTS adapter
// This struct should be used to configure the ElevenLabsTTS adapter
// Required fields:
// - APIKey: Your Eleven Labs API key
// Optional fields with defaults:
// - APIBaseURL: The base URL for the Eleven Labs API (default: "https://api.elevenlabs.io/v1")
// - VoiceID: The voice ID to use (default: "21m00Tcm4TlvDq8ikWAM" - Rachel voice)
// - ModelID: The model ID to use (default: "eleven_multilingual_v2")
// - OutputFormat: The output format (default: "pcm_24000")
// - ChunkSize: The size of audio chunks to stream (default: 1024)
// - Stability: Voice stability value between 0 and 1 (default: 0.5)
// - Clarity: Voice clarity/similarity boost value between 0 and 1 (default: 0.75)
type ElevenLabsConfig struct {
	APIKey       string  // Required: Your Eleven Labs API key
	APIBaseURL   string  // Optional: The base URL for the Eleven Labs API
	VoiceID      string  // Optional: The voice ID to use
	ModelID      string  // Optional: The model ID to use
	OutputFormat string  // Optional: The output format
	ChunkSize    int     // Optional: The size of audio chunks to stream
	Stability    float64 // Optional: Voice stability value between 0 and 1
	Clarity      float64 // Optional: Voice clarity/similarity boost value between 0 and 1
}

// ElevenLabsTTS implements TextToSpeech interface using Eleven Labs API
type ElevenLabsTTS struct {
	apiKey       string
	apiBaseURL   string
	voiceID      string
	modelID      string
	outputFormat string
	chunkSize    int
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

// ValidateElevenLabsConfig validates the ElevenLabsConfig
func ValidateElevenLabsConfig(config ElevenLabsConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("eleven labs API key is required")
	}

	// Validate stability is in the valid range
	if config.Stability != 0 && (config.Stability < 0 || config.Stability > 1) {
		return fmt.Errorf("stability must be between 0 and 1, got %f", config.Stability)
	}

	// Validate clarity is in the valid range
	if config.Clarity != 0 && (config.Clarity < 0 || config.Clarity > 1) {
		return fmt.Errorf("clarity must be between 0 and 1, got %f", config.Clarity)
	}

	// Validate chunk size is reasonable if specified
	if config.ChunkSize < 0 {
		return fmt.Errorf("chunk size must be positive, got %d", config.ChunkSize)
	}

	return nil
}

// NewElevenLabsTTS creates a new Eleven Labs TTS instance
func NewElevenLabsTTS(config ElevenLabsConfig, logger *zap.Logger) (*ElevenLabsTTS, error) {
	// Validate required configuration
	if err := ValidateElevenLabsConfig(config); err != nil {
		return nil, err
	}

	// Apply defaults where needed
	apiBaseURL := config.APIBaseURL
	if apiBaseURL == "" {
		apiBaseURL = defaultAPIBaseURL
		logger.Info("Using default API base URL", zap.String("apiBaseURL", apiBaseURL))
	}

	voiceID := config.VoiceID
	if voiceID == "" {
		voiceID = defaultVoiceID
		logger.Info("Using default voice ID", zap.String("voiceID", voiceID))
	}

	modelID := config.ModelID
	if modelID == "" {
		modelID = defaultModelID
		logger.Info("Using default model ID", zap.String("modelID", modelID))
	}

	outputFormat := config.OutputFormat
	if outputFormat == "" {
		outputFormat = defaultOutputFormat
		logger.Info("Using default output format", zap.String("outputFormat", outputFormat))
	}

	chunkSize := config.ChunkSize
	if chunkSize == 0 {
		chunkSize = defaultChunkSize
		logger.Info("Using default chunk size", zap.Int("chunkSize", chunkSize))
	}

	// Use provided stability/clarity or defaults
	stability := config.Stability
	if stability == 0 {
		stability = defaultStability
		logger.Info("Using default stability", zap.Float64("stability", stability))
	}

	clarity := config.Clarity
	if clarity == 0 {
		clarity = defaultClarity
		logger.Info("Using default clarity", zap.Float64("clarity", clarity))
	}

	return &ElevenLabsTTS{
		apiKey:       config.APIKey,
		apiBaseURL:   apiBaseURL,
		voiceID:      voiceID,
		modelID:      modelID,
		outputFormat: outputFormat,
		chunkSize:    chunkSize,
		stability:    stability,
		clarity:      clarity,
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
		e.apiBaseURL, e.voiceID, e.outputFormat)
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
		buffer := make([]byte, e.chunkSize)
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

// NewElevenLabsConfigFromEnv creates a new ElevenLabsConfig from environment variables
// This is a helper function to simplify the creation of a properly configured ElevenLabsConfig
func NewElevenLabsConfigFromEnv() ElevenLabsConfig {
	// Read required API key
	apiKey := os.Getenv("ELEVEN_LABS_API_KEY")

	// Read optional parameters with defaults
	config := ElevenLabsConfig{
		APIKey:       apiKey,
		APIBaseURL:   os.Getenv("ELEVEN_LABS_API_BASE_URL"),
		VoiceID:      os.Getenv("ELEVEN_LABS_VOICE_ID"),
		ModelID:      os.Getenv("ELEVEN_LABS_MODEL_ID"),
		OutputFormat: os.Getenv("ELEVEN_LABS_OUTPUT_FORMAT"),
	}

	// Parse numeric values from environment
	if chunkSizeStr := os.Getenv("ELEVEN_LABS_CHUNK_SIZE"); chunkSizeStr != "" {
		if chunkSize, err := strconv.Atoi(chunkSizeStr); err == nil && chunkSize > 0 {
			config.ChunkSize = chunkSize
		}
	}

	if stabilityStr := os.Getenv("ELEVEN_LABS_STABILITY"); stabilityStr != "" {
		if stability, err := strconv.ParseFloat(stabilityStr, 64); err == nil && stability >= 0 && stability <= 1 {
			config.Stability = stability
		}
	}

	if clarityStr := os.Getenv("ELEVEN_LABS_CLARITY"); clarityStr != "" {
		if clarity, err := strconv.ParseFloat(clarityStr, 64); err == nil && clarity >= 0 && clarity <= 1 {
			config.Clarity = clarity
		}
	}

	return config
}

// GetAvailableVoices retrieves available voices from Eleven Labs API
func (e *ElevenLabsTTS) GetAvailableVoices(ctx context.Context) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/voices", e.apiBaseURL)

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
