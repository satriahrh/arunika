# Eleven Labs Text-to-Speech Integration

## Overview

This implementation adds text-to-speech functionality to the Arunika project using the Eleven Labs API. The integration follows the clean architecture pattern and is designed to work seamlessly with the existing websocket-based audio streaming system.

## Implementation Details

### 1. Core Components

#### `adapters/tts/elevenlabs.go`
- **Primary TTS Implementation**: Implements the `repositories.TextToSpeech` interface
- **Streaming Support**: Provides real-time audio streaming in chunks for optimal performance
- **Configuration**: Supports voice customization, model selection, and voice settings
- **Error Handling**: Comprehensive error handling with detailed logging

#### Key Features:
- ✅ Streaming audio response in configurable chunks
- ✅ Environment variable configuration
- ✅ Voice and model customization
- ✅ Context cancellation support
- ✅ Comprehensive error handling and logging
- ✅ Interface compliance verification

### 2. Integration Points

#### WebSocket Hub Integration
The TTS service is integrated into the websocket hub (`internal/websocket/hub.go`) to provide audio responses during voice conversations:

1. **Session Initialization**: TTS repository is created when audio sessions start
2. **Response Generation**: After speech-to-text and LLM processing, text responses are converted to audio
3. **Streaming Delivery**: Audio chunks are streamed back to the client via websocket

#### Audio Flow:
```
Device Audio Input → STT → LLM → TTS → Device Audio Output
```

### 3. Configuration

#### Required Environment Variables:
- `ELEVEN_LABS_API_KEY`: Your Eleven Labs API key

#### Optional Environment Variables:
- `ELEVEN_LABS_VOICE_ID`: Voice ID (defaults to Rachel voice)
- `ELEVEN_LABS_MODEL_ID`: Model ID (defaults to "eleven_monolingual_v1")

### 4. Usage Examples

#### Basic Usage in Code:
```go
ttsService, err := tts.NewElevenLabsTTS(logger)
if err != nil {
    return err
}

audioChan, err := ttsService.ConvertTextToSpeech(ctx, "Hello world!")
if err != nil {
    return err
}

for audioChunk := range audioChan {
    // Process audio chunk
    handleAudioChunk(audioChunk)
}
```

#### Example Application:
A complete example is provided in `adapters/tts/example/main.go` that demonstrates:
- TTS service initialization
- Text-to-speech conversion
- Audio file saving
- Voice listing functionality

#### Running the Example:
```bash
cd server
export ELEVEN_LABS_API_KEY="your_api_key_here"
go run ./adapters/tts/example/main.go
```

### 5. Testing

#### Unit Tests
- **Location**: `adapters/tts/elevenlabs_test.go`
- **Coverage**: Configuration, initialization, settings, error cases
- **Integration Test**: Available when real API key is provided

#### Running Tests:
```bash
# Unit tests only
go test ./adapters/tts/...

# With integration tests
ELEVEN_LABS_API_KEY="real_api_key" go test ./adapters/tts/... -v
```

### 6. Voice Customization

#### Available Parameters:
- **Stability**: Controls voice consistency (0.0 - 1.0)
- **Clarity**: Controls voice clarity/similarity boost (0.0 - 1.0)
- **Voice ID**: Selects different voice characters
- **Model ID**: Chooses the TTS model

#### Popular Voice IDs:
- `21m00Tcm4TlvDq8ikWAM` - Rachel (default, female)
- `pNInz6obpgDQGcFmaJgB` - Adam (male)
- `ErXwobaYiN019PkySvjV` - Antoni (male)

### 7. Error Handling

The implementation handles various error scenarios:
- ❌ Missing API key
- ❌ Empty/invalid text input
- ❌ Network connectivity issues
- ❌ API rate limiting
- ❌ Invalid voice/model IDs
- ❌ Context cancellation/timeouts

### 8. Performance Considerations

#### Streaming Benefits:
- **Reduced Latency**: Audio starts playing while generation continues
- **Memory Efficiency**: Large audio files are processed in chunks
- **Real-time Response**: Immediate feedback in conversation systems

#### Chunk Size:
- Default: 1024 bytes per chunk
- Configurable via `chunkSize` constant
- Balance between responsiveness and efficiency

### 9. Future Enhancements

#### Possible Improvements:
1. **Voice Caching**: Cache generated audio for repeated phrases
2. **SSML Support**: Add Speech Synthesis Markup Language support
3. **Voice Cloning**: Integration with Eleven Labs voice cloning features
4. **Batch Processing**: Support for multiple text inputs
5. **Quality Selection**: Different quality/bitrate options

### 10. Dependencies

#### New Dependencies Added:
- No new external dependencies required
- Uses standard Go HTTP client
- Leverages existing logging infrastructure (zap)

#### API Compatibility:
- Eleven Labs API v1
- RESTful HTTP interface
- JSON request/response format
- MP3 audio output format

## Conclusion

The Eleven Labs TTS integration provides a robust, scalable text-to-speech solution that integrates seamlessly with the existing Arunika architecture. The streaming approach ensures optimal performance for real-time voice interaction scenarios, while the comprehensive error handling and logging provide reliability in production environments.
