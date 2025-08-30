# ElevenLabs Text-to-Speech Implementation

This directory contains the implementation of the Text-to-Speech (TTS) service using the ElevenLabs API, optimized for real-time streaming with Bahasa Indonesia support.

## Features

- **Real-time PCM Streaming**: Low-latency audio streaming optimized for real-time applications
- **Bahasa Indonesia Support**: Language enforcement for better Indonesian pronunciation
- **Latency Optimization**: Configurable latency levels (0-4) for speed vs quality trade-offs
- **Configurable Voice Settings**: Allows customization of stability and clarity parameters
- **Multiple Output Formats**: Support for PCM, MP3, and other audio formats
- **Error Handling**: Comprehensive error handling with detailed logging
- **Context Support**: Full context cancellation support for timeout and cancellation scenarios

## Configuration

The following environment variables are supported:

- `ELEVEN_LABS_API_KEY`: Your Eleven Labs API key (required)
- `ELEVEN_LABS_VOICE_ID`: Voice ID to use (optional, defaults to Rachel voice)
- `ELEVEN_LABS_MODEL_ID`: Model ID to use (optional, defaults to "eleven_multilingual_v2")
- `ELEVEN_LABS_OUTPUT_FORMAT`: Audio output format (optional, defaults to "pcm_44100")

## Real-time Optimizations

### Default Configuration
- **Output Format**: `pcm_44100` - 44.1kHz PCM for immediate playback
- **Latency Optimization**: Level 3 (maximum) for ~75% latency reduction
- **Language**: Indonesian (`id`) for proper pronunciation
- **Model**: `eleven_multilingual_v2` for multilingual support

### Available Output Formats
| Format | Sample Rate | Use Case |
|--------|-------------|----------|
| `pcm_44100` | 44.1kHz | Real-time applications (default) |
| `pcm_22050` | 22.05kHz | Lower bandwidth requirements |
| `mp3_44100_128` | 44.1kHz | File storage |

### Latency Optimization Levels
| Level | Description | Latency Reduction |
|-------|-------------|-------------------|
| 0 | Default mode | No optimization |
| 1 | Normal optimization | ~50% improvement |
| 2 | Strong optimization | ~75% improvement |
| 3 | Maximum optimization | Maximum improvement (default) |
| 4 | Max + no text normalizer | Best latency, may mispronounce |

## Usage

### Basic Real-time Usage

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "go.uber.org/zap"
    "github.com/satriahrh/arunika/server/adapters/tts"
)

func main() {
    logger, _ := zap.NewDevelopment()
    
    // Create TTS instance with real-time optimizations
    ttsService, err := tts.NewElevenLabsTTS(logger)
    if err != nil {
        panic(err)
    }
    
    // Configure for real-time Indonesian speech
    ttsService.SetOutputFormat("pcm_44100")        // PCM for real-time
    ttsService.SetLatencyOptimization(3)           // Maximum optimization
    ttsService.SetVoiceSettings(0.7, 0.8)          // Tuned for Indonesian
    
    // Convert Indonesian text to speech
    ctx := context.Background()
    text := "Halo! Selamat datang di sistem Arunika."
    
    audioChan, err := ttsService.ConvertTextToSpeech(ctx, text)
    if err != nil {
        panic(err)
    }
    
    // Stream audio chunks for real-time playback
    file, err := os.Create("output.pcm")
    if err != nil {
        panic(err)
    }
    defer file.Close()
    
    for audioChunk := range audioChan {
        // Write to file (or send to audio player for real-time playback)
        _, err := file.Write(audioChunk)
        if err != nil {
            panic(err)
        }
        
        // For real-time applications, you would send audioChunk
        // directly to your audio player/streamer here
        // playAudioChunk(audioChunk)
    }
    
    fmt.Println("Audio saved to output.pcm")
}
```

### Advanced Configuration

```go
// Custom voice settings for Indonesian speech
ttsService.SetVoiceSettings(0.8, 0.9) // stability, clarity

// Change voice (ensure it supports Indonesian)
ttsService.SetVoiceID("your-voice-id")

// Adjust latency vs quality trade-off
ttsService.SetLatencyOptimization(2) // Medium optimization for better quality

// Change output format for different use cases
ttsService.SetOutputFormat("pcm_22050") // Lower sample rate for bandwidth
ttsService.SetOutputFormat("mp3_44100_128") // MP3 for file storage
```

## Available Voice IDs

Some popular voice IDs from Eleven Labs:

- `21m00Tcm4TlvDq8ikWAM` - Rachel (default)
- `pNInz6obpgDQGcFmaJgB` - Adam
- `ErXwobaYiN019PkySvjV` - Antoni
- `VR6AewLTigWG4xSOukaG` - Arnold
- `EXAVITQu4vr4xnSDxMaL` - Bella

You can also retrieve available voices programmatically:

```go
voices, err := ttsService.GetAvailableVoices(ctx)
if err != nil {
    panic(err)
}

for _, voice := range voices {
    fmt.Printf("Voice: %s (%s)\n", voice["name"], voice["voice_id"])
}
```

## Testing

### Unit Tests
```bash
go test ./adapters/tts -v
```

### Integration Tests
```bash
# Requires real API key for Indonesian speech testing
ELEVEN_LABS_API_KEY=your_real_api_key go test ./adapters/tts -v
```

### Example Usage
```bash
cd adapters/tts/example
ELEVEN_LABS_API_KEY=your_key go run main.go
```

## Real-time Application Notes

- **PCM Format**: Provides raw audio data ready for immediate playback without decoding overhead
- **Chunk Size**: Optimized at 1KB for low-latency streaming while maintaining efficiency
- **Context Handling**: Proper cancellation support for streaming termination
- **Error Recovery**: API-specific error responses for debugging connection issues
- **Performance Monitoring**: Detailed logging for streaming metrics and performance analysis

## Bahasa Indonesia Considerations

- **Multilingual Model**: Using `eleven_multilingual_v2` for optimal Indonesian pronunciation
- **Language Enforcement**: Language code `id` ensures proper Indonesian speech patterns
- **Text Normalization**: Automatic handling of numbers, dates, and special characters in Indonesian context
- **Voice Tuning**: Default voice settings optimized for Indonesian speech characteristics

## Error Handling

The implementation includes comprehensive error handling for:

- Missing or invalid API key
- Empty or invalid text input
- Network connectivity issues
- API rate limits and quota exceeded
- Invalid voice IDs or model selections
- Context cancellation and timeouts
- Audio format compatibility issues

All errors are logged with appropriate context using structured logging for debugging and monitoring.
