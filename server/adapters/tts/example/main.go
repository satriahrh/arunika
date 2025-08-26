package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"go.uber.org/zap"

	"github.com/joho/godotenv"
	"github.com/satriahrh/arunika/server/adapters/tts"
)

func main() {
	godotenv.Load()

	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Check if API key is set
	if os.Getenv("ELEVEN_LABS_API_KEY") == "" {
		logger.Fatal("ELEVEN_LABS_API_KEY environment variable is required")
	}

	// Create TTS service
	ttsService, err := tts.NewElevenLabsTTS(logger)
	if err != nil {
		logger.Fatal("Failed to create TTS service", zap.Error(err))
	}

	// Indonesian text to convert
	text := "Halo! Ini adalah demonstrasi dari integrasi text to speech Eleven Labs dalam proyek Arunika."

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Converting text to speech", zap.String("text", text))

	// Convert text to speech
	audioChan, err := ttsService.ConvertTextToSpeech(ctx, text)
	if err != nil {
		logger.Fatal("Failed to convert text to speech", zap.Error(err))
	}

	// Create output file - use .pcm extension for PCM format
	outputFile := "example_output.pcm"
	file, err := os.Create(outputFile)
	if err != nil {
		logger.Fatal("Failed to create output file", zap.Error(err))
	}
	defer file.Close()

	// Process audio chunks
	totalBytes := 0
	chunkCount := 0

	logger.Info("Receiving and saving audio chunks...")

	for audioChunk := range audioChan {
		if len(audioChunk) == 0 {
			logger.Warn("Received empty audio chunk")
			continue
		}

		// Write chunk to file
		n, err := file.Write(audioChunk)
		if err != nil {
			logger.Error("Failed to write audio chunk", zap.Error(err))
			break
		}

		totalBytes += n
		chunkCount++

		logger.Debug("Received audio chunk",
			zap.Int("chunkNumber", chunkCount),
			zap.Int("chunkSize", len(audioChunk)),
			zap.Int("totalBytes", totalBytes))
	}

	logger.Info("Audio conversion completed",
		zap.Int("totalChunks", chunkCount),
		zap.Int("totalBytes", totalBytes),
		zap.String("outputFile", outputFile))

	fmt.Printf("✅ Audio successfully saved to %s (%d bytes in %d chunks)\n", outputFile, totalBytes, chunkCount)

	// Close the file before playing it
	file.Close()

	// Play the audio file automatically
	if os.Getenv("NO_AUTOPLAY") != "true" {
		logger.Info("Playing audio file automatically...")
		err := playAudioFile(outputFile, logger)
		if err != nil {
			logger.Warn("Failed to play audio automatically", zap.Error(err))
			fmt.Printf("⚠️  Could not auto-play audio. You can manually play it with:\n")
			printPlaybackInstructions(outputFile)
		} else {
			fmt.Printf("🎵 Audio played successfully!\n")
		}
	} else {
		fmt.Printf("🎵 To play the audio file, use:\n")
		printPlaybackInstructions(outputFile)
	}

	// Optional: Get available voices
	if os.Getenv("SHOW_VOICES") == "true" {
		logger.Info("Fetching available voices...")
		voices, err := ttsService.GetAvailableVoices(ctx)
		if err != nil {
			logger.Warn("Failed to get available voices", zap.Error(err))
		} else {
			fmt.Printf("\n📢 Available voices (%d):\n", len(voices))
			for i, voice := range voices {
				if i >= 10 {
					fmt.Printf("... and %d more voices\n", len(voices)-10)
					break
				}
				fmt.Printf("  - %s (ID: %s)\n", voice["name"], voice["voice_id"])
			}
		}
	}
}

// playAudioFile attempts to play a PCM audio file using available system tools
func playAudioFile(filename string, logger *zap.Logger) error {
	var cmd *exec.Cmd

	// Try different audio players based on the operating system and availability
	players := getAudioPlayers()

	for _, player := range players {
		if isCommandAvailable(player.command) {
			args := append(player.args, filename)
			cmd = exec.Command(player.command, args...)
			logger.Info("Attempting to play audio",
				zap.String("player", player.command),
				zap.Strings("args", args))

			err := cmd.Run()
			if err == nil {
				return nil
			}
			logger.Debug("Player failed",
				zap.String("player", player.command),
				zap.Error(err))
		}
	}

	return fmt.Errorf("no suitable audio player found")
}

// audioPlayer represents an audio player command and its arguments
type audioPlayer struct {
	command string
	args    []string
}

// getAudioPlayers returns a list of audio players to try, with PCM-specific arguments
func getAudioPlayers() []audioPlayer {
	// For pcm_24000 format: 24kHz, signed 16-bit, mono
	return []audioPlayer{
		// SoX play command (most common)
		{"play", []string{"-t", "raw", "-r", "24000", "-e", "signed", "-b", "16", "-c", "1"}},
		// FFplay (part of FFmpeg)
		{"ffplay", []string{"-f", "s16le", "-ar", "24000", "-ac", "1", "-nodisp", "-autoexit"}},
		// ALSA aplay (Linux)
		{"aplay", []string{"-f", "S16_LE", "-r", "24000", "-c", "1"}},
		// macOS afplay (won't work with raw PCM, but trying anyway)
		{"afplay", []string{}},
	}
}

// isCommandAvailable checks if a command is available in the system PATH
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// printPlaybackInstructions prints manual playback instructions for different platforms
func printPlaybackInstructions(filename string) {
	fmt.Printf("  # Using SoX (recommended):\n")
	fmt.Printf("  play -t raw -r 24000 -e signed -b 16 -c 1 %s\n\n", filename)

	fmt.Printf("  # Using FFplay:\n")
	fmt.Printf("  ffplay -f s16le -ar 24000 -ac 1 -nodisp -autoexit %s\n\n", filename)

	if runtime.GOOS == "linux" {
		fmt.Printf("  # Using ALSA (Linux):\n")
		fmt.Printf("  aplay -f S16_LE -r 24000 -c 1 %s\n\n", filename)
	}

	fmt.Printf("  # Convert to WAV for easier playback:\n")
	fmt.Printf("  ffmpeg -f s16le -ar 24000 -ac 1 -i %s %s.wav\n", filename, filename[:len(filename)-4])
}
