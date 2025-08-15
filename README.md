# Arunika - AI-Powered Interactive Talking Dolls

Arunika transforms ordinary dolls into interactive, intelligent companions through an integrated AI-powered conversational module. This monorepo contains the complete system including server backend and device firmware.

## Project Structure

```
arunika/
├── docs/                          # Documentation
│   ├── technical-requirements-document.md
│   └── product-requirement/
├── server/                        # Go backend server
│   ├── cmd/                       # Application entrypoints
│   ├── internal/                  # Private application code
│   │   ├── api/                   # REST API handlers
│   │   ├── auth/                  # Authentication logic
│   │   ├── websocket/             # WebSocket communication
│   │   ├── ai/                    # AI service integration
│   │   └── models/                # Data models
│   ├── pkg/                       # Public packages
│   └── go.mod                     # Go module definition
├── doll-m2/                       # C firmware for doll devices
│   ├── include/                   # Header files
│   ├── src/                       # Source files
│   ├── lib/                       # External libraries
│   ├── tests/                     # Unit tests
│   ├── Makefile                   # Build configuration
│   └── README.md                  # Firmware documentation
├── scripts/                       # Build and deployment scripts
└── README.md                      # This file
```

## Getting Started

### Prerequisites

- **Go 1.21+** (for server development)
- **GCC** (for firmware development)
- **Make** (for building)
- **Git** (for version control)

### Quick Start

1. **Clone the repository:**
   ```bash
   git clone https://github.com/satriahrh/arunika.git
   cd arunika
   ```

2. **Start the server:**
   ```bash
   cd server
   go mod tidy
   go run cmd/main.go
   ```

3. **Build the firmware:**
   ```bash
   cd doll-m2
   make all
   ```

## Components

### 1. Server (Go)

The backend server handles:
- RESTful APIs for user management and device authentication
- WebSocket communication with doll devices
- AI service integration (STT, LLM, TTS)
- Real-time audio processing pipeline

**Key Features:**
- Device authentication and session management
- Parent dashboard APIs
- Child-safe AI conversation engine
- Streaming audio processing

**Technology Stack:**
- **Framework:** Echo (Golang web framework)
- **WebSocket:** Gorilla WebSocket
- **Authentication:** JWT tokens
- **Logging:** Zap (structured logging)

### 2. Doll Firmware (C)

The embedded firmware for ESP32-based doll devices:
- Audio input/output handling
- WiFi connectivity and WebSocket communication
- Device state management and power optimization
- Button input and interaction handling

**Key Features:**
- Real-time audio streaming
- Low-power operation modes
- Automatic reconnection and error recovery
- Hardware abstraction layer for portability

**Target Hardware:**
- **MCU:** ESP32-S3 (dual-core, WiFi)
- **Audio:** I2S digital microphone and amplifier
- **Power:** 3.7V LiPo battery with USB-C charging

## Architecture Overview

```
Child ↔ Doll Device ↔ WebSocket ↔ Go Server ↔ AI Services
                                        ↓
                               Parent Dashboard
```

### Communication Flow

1. **Child speaks to doll** → Audio captured by microphone
2. **Device streams audio** → WebSocket connection to server
3. **Server processes audio** → Speech-to-Text → LLM → Text-to-Speech
4. **Server streams response** → WebSocket back to device
5. **Device plays response** → Speaker output to child

## Development

### Server Development

```bash
cd server

# Install dependencies
go mod tidy

# Run server
go run cmd/main.go

# Run tests
go test ./...

# Build binary
go build -o bin/arunika-server cmd/main.go
```

### Firmware Development

```bash
cd doll-m2

# Build for development
make all

# Run tests
make test

# Clean build
make clean

# Future ESP32 development
make esp32-build
```

## API Documentation

### Device APIs
- `POST /api/v1/device/auth` - Device authentication

### Parent Dashboard APIs
- `POST /api/v1/users/register` - User registration
- `POST /api/v1/users/login` - User login
- `GET/POST/PUT /api/v1/children` - Child profile management
- `GET /api/v1/conversations` - Conversation history

### WebSocket Communication
- **Endpoint:** `/ws?device_id=<device_id>`
- **Authentication:** JWT token in query parameter
- **Message Types:** Audio chunks, AI responses, control messages
- **Heartbeat:** 5-second ping/pong for connection monitoring

## Configuration

### Server Configuration

Environment variables:
```bash
PORT=8080                    # Server port
JWT_SECRET=your-secret-key   # JWT signing secret
LOG_LEVEL=info              # Logging level
```

### Device Configuration

Device configuration (stored in firmware):
```c
wifi_ssid = "YourWiFiNetwork"
wifi_password = "YourWiFiPassword"
server_url = "wss://api.arunika.com"
device_id = "ARUN_DEV_001234"
```

## Deployment

### Server Deployment

```bash
# Build for production
cd server
go build -ldflags="-s -w" -o bin/arunika-server cmd/main.go

# Docker deployment (future)
docker build -t arunika-server .
docker run -p 8080:8080 arunika-server
```

### Device Firmware

```bash
# Flash to ESP32 (future)
cd doll-m2
make esp32-flash
```

## Testing

### Unit Tests

```bash
# Server tests
cd server
go test ./...

# Firmware tests
cd doll-m2
make test
```

### Integration Tests

```bash
# End-to-end testing
./scripts/test-integration.sh
```

## Monitoring and Observability

- **Structured Logging:** Zap logger with JSON format
- **Health Checks:** `/health` endpoint
- **WebSocket Metrics:** Connection count, message throughput
- **Device Metrics:** Battery level, connection status

## Security

- **Device Authentication:** mTLS certificates + JWT tokens
- **Content Filtering:** Child-safe AI response validation
- **Data Privacy:** No persistent voice recording storage
- **Network Security:** WSS (WebSocket Secure) communication

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style

- **Go:** Follow `gofmt` and `golint` standards
- **C:** Follow Linux kernel coding style
- **Commit Messages:** Use conventional commits format

## Roadmap

### Phase 1: MVP Development (Current)
- [x] Basic server structure and API endpoints
- [x] Firmware architecture and build system
- [ ] WebSocket communication implementation
- [ ] Audio processing pipeline
- [ ] Device authentication

### Phase 2: Hardware Integration
- [ ] ESP32 firmware porting
- [ ] I2S audio drivers
- [ ] Power management optimization
- [ ] Hardware testing and validation

### Phase 3: AI Integration
- [ ] Speech-to-Text service integration
- [ ] LLM conversation engine
- [ ] Text-to-Speech synthesis
- [ ] Content safety filtering

### Phase 4: Production Ready
- [ ] Manufacturing integration
- [ ] Quality assurance testing
- [ ] Performance optimization
- [ ] Deployment automation

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For technical support and questions:
- **Issues:** GitHub Issues
- **Documentation:** [Technical Requirements Document](docs/technical-requirements-document.md)
- **Contact:** Open an issue for support

---

**Arunika** - Making dolls come to life with AI 🤖✨
