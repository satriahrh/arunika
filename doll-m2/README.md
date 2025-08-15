# Arunika Doll M2 Firmware

This directory contains the firmware for the Arunika interactive doll (Model M2), written in C and designed to be portable for ESP32 development.

## Overview

The doll firmware handles:
- Audio input/output (microphone and speaker)
- WiFi connectivity
- WebSocket communication with the cloud server
- Device state management
- Power management
- Button input handling

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Audio Input   │    │  Button Input   │    │ Power Manager   │
│  (Microphone)   │    │                 │    │                 │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          v                      v                      v
┌─────────────────────────────────────────────────────────────────┐
│                    Main Application                             │
│                  (State Machine)                                │
└─────────────────────┬───────────────────────────────────────────┘
                      │
          ┌───────────┼───────────┐
          v           v           v
┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│   Network   │ │  WebSocket  │ │ Audio Output│
│    (WiFi)   │ │ Communication│ │  (Speaker)  │
└─────────────┘ └─────────────┘ └─────────────┘
```

## Directory Structure

```
doll-m2/
├── include/           # Header files
│   └── arunika.h     # Main API definitions
├── src/              # Source files
│   ├── main.c        # Main application
│   ├── config.c      # Configuration management
│   ├── audio.c       # Audio input/output (to be implemented)
│   ├── network.c     # WiFi and networking (to be implemented)
│   ├── websocket.c   # WebSocket communication (to be implemented)
│   └── power.c       # Power management (to be implemented)
├── lib/              # External libraries
├── tests/            # Unit tests
├── Makefile          # Build configuration
└── README.md         # This file
```

## Current Status

**Current Implementation:** 
- Basic C structure and API definitions
- Placeholder implementations for testing
- Makefile for development build

**To Be Implemented:**
- ESP32-specific audio drivers (I2S)
- WiFi connection management
- WebSocket client implementation
- Real-time audio streaming
- Hardware-specific power management
- Button debouncing and input handling

## Building

### For Development (Native C)

```bash
# Build the application
make all

# Run tests
make test

# Clean build files
make clean
```

### For ESP32 (Future)

When porting to ESP32, the following targets will be available:

```bash
# Build for ESP32
make esp32-build

# Flash to ESP32
make esp32-flash

# Monitor ESP32 output
make esp32-monitor
```

## ESP32 Migration Plan

### Phase 1: Setup ESP-IDF Environment
- Install ESP-IDF development framework
- Configure project structure for ESP-IDF
- Set up build system (idf.py)

### Phase 2: Hardware Abstraction Layer
- Implement I2S audio input/output
- WiFi connection management
- GPIO button handling
- Power management (battery monitoring, sleep modes)

### Phase 3: Communication Layer
- WebSocket client with SSL/TLS support
- Audio streaming and buffering
- Message queuing and retry logic
- Connection management and reconnection

### Phase 4: Application Layer
- State machine implementation
- Audio processing pipeline
- Error handling and recovery
- Performance optimization

## Hardware Requirements

### Target Hardware: ESP32-S3
- **MCU:** Dual-core Xtensa LX7 @240MHz
- **Memory:** 512KB SRAM, 384KB ROM
- **Flash:** 4MB (minimum)
- **Connectivity:** WiFi 802.11 b/g/n
- **Audio:** I2S interface for digital audio

### Audio Components
- **Microphone:** INMP441 I2S digital microphone
- **Amplifier:** MAX98357A I2S digital amplifier  
- **Speaker:** 4Ω, 3W speaker
- **Sample Rate:** 8kHz (for voice optimization)
- **Bit Depth:** 16-bit

### Power System
- **Battery:** 3.7V 1200mAh LiPo
- **Charging:** USB-C with TP4056 charge controller
- **Power Management:** Low-power modes, sleep management

## Communication Protocol

### WebSocket Messages

**Audio Chunk (Device → Server):**
```json
{
  "type": "audio_chunk",
  "device_id": "ARUN_DEV_001234",
  "audio_data": "base64_encoded_audio",
  "sample_rate": 8000,
  "encoding": "MULAW",
  "timestamp": "2024-08-08T10:30:00Z",
  "chunk_sequence": 15,
  "is_final": false
}
```

**AI Response (Server → Device):**
```json
{
  "type": "ai_response",
  "session_id": "sess_abc123def456",
  "response_text": "Hello! How are you feeling today?",
  "audio_data": "base64_encoded_response_audio",
  "emotion": "cheerful",
  "timestamp": "2024-08-08T10:30:02Z"
}
```

## Development Workflow

1. **Current Phase:** Develop and test core logic in native C
2. **Testing:** Unit tests for individual components
3. **Integration:** Combine components and test communication
4. **Migration:** Port to ESP-IDF and implement hardware drivers
5. **Optimization:** Performance tuning and power optimization

## Testing

```bash
# Run all tests
make test

# Test individual components (future)
make test-audio
make test-network
make test-websocket
```

## Dependencies

### Current (Native Development)
- GCC compiler
- Make build system
- Standard C library

### Future (ESP32 Development)
- ESP-IDF v5.0+
- WebSocket client library
- Audio processing libraries
- SSL/TLS support

## Contributing

1. Follow the existing code structure and naming conventions
2. Add unit tests for new functionality
3. Update documentation for any API changes
4. Test on both native and ESP32 platforms (when available)

## License

This firmware is part of the Arunika project and follows the project's licensing terms.
