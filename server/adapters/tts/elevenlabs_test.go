package tts

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestNewElevenLabsTTS(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Test without API key
	os.Unsetenv("ELEVEN_LABS_API_KEY")
	config := NewElevenLabsConfigFromEnv()
	_, err := NewElevenLabsTTS(config, logger)
	if err == nil {
		t.Error("Expected error when API key is not set")
	}

	// Test with API key
	os.Setenv("ELEVEN_LABS_API_KEY", "test-api-key")
	defer os.Unsetenv("ELEVEN_LABS_API_KEY")

	config = NewElevenLabsConfigFromEnv()
	tts, err := NewElevenLabsTTS(config, logger)
	if err != nil {
		t.Fatalf("Failed to create ElevenLabsTTS: %v", err)
	}

	if tts.apiKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got '%s'", tts.apiKey)
	}

	if tts.voiceID != defaultVoiceID {
		t.Errorf("Expected default voice ID '%s', got '%s'", defaultVoiceID, tts.voiceID)
	}
}

func TestElevenLabsTTS_SetVoiceSettings(t *testing.T) {
	logger := zaptest.NewLogger(t)
	os.Setenv("ELEVEN_LABS_API_KEY", "test-api-key")
	defer os.Unsetenv("ELEVEN_LABS_API_KEY")

	config := NewElevenLabsConfigFromEnv()
	tts, err := NewElevenLabsTTS(config, logger)
	if err != nil {
		t.Fatalf("Failed to create ElevenLabsTTS: %v", err)
	}

	tts.SetVoiceSettings(0.8, 0.9)

	if tts.stability != 0.8 {
		t.Errorf("Expected stability 0.8, got %f", tts.stability)
	}

	if tts.clarity != 0.9 {
		t.Errorf("Expected clarity 0.9, got %f", tts.clarity)
	}
}

func TestElevenLabsTTS_SetVoiceID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	os.Setenv("ELEVEN_LABS_API_KEY", "test-api-key")
	defer os.Unsetenv("ELEVEN_LABS_API_KEY")

	config := NewElevenLabsConfigFromEnv()
	tts, err := NewElevenLabsTTS(config, logger)
	if err != nil {
		t.Fatalf("Failed to create ElevenLabsTTS: %v", err)
	}

	newVoiceID := "new-voice-id"
	tts.SetVoiceID(newVoiceID)

	if tts.voiceID != newVoiceID {
		t.Errorf("Expected voice ID '%s', got '%s'", newVoiceID, tts.voiceID)
	}
}

func TestElevenLabsTTS_ConvertTextToSpeech_EmptyText(t *testing.T) {
	logger := zaptest.NewLogger(t)
	os.Setenv("ELEVEN_LABS_API_KEY", "test-api-key")
	defer os.Unsetenv("ELEVEN_LABS_API_KEY")

	config := NewElevenLabsConfigFromEnv()
	tts, err := NewElevenLabsTTS(config, logger)
	if err != nil {
		t.Fatalf("Failed to create ElevenLabsTTS: %v", err)
	}

	ctx := context.Background()
	_, err = tts.ConvertTextToSpeech(ctx, "")
	if err == nil {
		t.Error("Expected error for empty text")
	}

	_, err = tts.ConvertTextToSpeech(ctx, "   ")
	if err == nil {
		t.Error("Expected error for whitespace-only text")
	}
}

// Integration test - only runs if ELEVEN_LABS_API_KEY is set with real API key
func TestElevenLabsTTS_ConvertTextToSpeech_Integration(t *testing.T) {
	apiKey := os.Getenv("ELEVEN_LABS_API_KEY")
	if apiKey == "" || apiKey == "test-api-key" {
		t.Skip("Skipping integration test - set ELEVEN_LABS_API_KEY environment variable with real API key")
	}

	logger := zap.NewNop() // Use no-op logger for integration test

	config := NewElevenLabsConfigFromEnv()
	tts, err := NewElevenLabsTTS(config, logger)
	if err != nil {
		t.Fatalf("Failed to create ElevenLabsTTS: %v", err)
	}

	// Configure for Indonesian streaming with low latency
	tts.SetOutputFormat("pcm_44100")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	text := "Halo, ini adalah tes integrasi Eleven Labs untuk Bahasa Indonesia."
	audioChan, err := tts.ConvertTextToSpeech(ctx, text)
	if err != nil {
		t.Fatalf("Failed to convert text to speech: %v", err)
	}

	totalBytes := 0
	chunkCount := 0

	for chunk := range audioChan {
		if len(chunk) == 0 {
			t.Error("Received empty audio chunk")
		}
		totalBytes += len(chunk)
		chunkCount++
	}

	if totalBytes == 0 {
		t.Error("No audio data received")
	}

	if chunkCount == 0 {
		t.Error("No audio chunks received")
	}

	t.Logf("Integration test completed: received %d chunks, %d total bytes", chunkCount, totalBytes)
}
