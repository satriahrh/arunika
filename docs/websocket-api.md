# WebSocket API Documentation

## Overview

The Arunika WebSocket API provides real-time bidirectional communication between devices and the server. It supports audio streaming, device management, system messaging, and authentication.

## Connection

### Endpoint
```
ws://localhost:8080/ws
```

### Authentication

All WebSocket connections require authentication via JWT tokens. The token can be provided in:

1. **Query Parameter** (Preferred)
   ```
   ws://localhost:8080/ws?token=<JWT_TOKEN>
   ```

2. **Authorization Header**
   ```
   Authorization: Bearer <JWT_TOKEN>
   ```

### Token Types

#### Device Token
For device connections (e.g., smart dolls):
```json
{
  "device_id": "device-123",
  "role": "device",
  "exp": 1640995200,
  "iat": 1640908800
}
```

#### User Token
For user/parent connections (e.g., mobile apps):
```json
{
  "user_id": "user-456",
  "role": "user", 
  "exp": 1641514800,
  "iat": 1640908800
}
```

## Message Format

All messages follow a standardized JSON format:

```json
{
  "type": "message_type",
  "timestamp": "2023-12-01T10:00:00Z",
  "message_id": "optional-uuid",
  // type-specific fields...
}
```

## Message Types

### 1. Audio Chunk (Client → Server)

Used for streaming audio data from devices to the server for processing.

```json
{
  "type": "audio_chunk",
  "timestamp": "2023-12-01T10:00:00Z",
  "device_id": "device-123",
  "session_id": "session-456",
  "audio_data": "SGVsbG8gV29ybGQ=",
  "sample_rate": 16000,
  "encoding": "pcm",
  "chunk_sequence": 1,
  "is_final": false,
  "duration_ms": 1000,
  "content_type": "audio/pcm"
}
```

**Fields:**
- `device_id` (string, required): Unique device identifier
- `session_id` (string, required): Conversation session ID
- `audio_data` (string, required): Base64-encoded audio data
- `sample_rate` (int, required): Audio sample rate (8000-48000 Hz)
- `encoding` (string, required): Audio encoding (`pcm`, `wav`, `mp3`, `opus`)
- `chunk_sequence` (int): Sequential chunk number (starts from 0)
- `is_final` (bool): Whether this is the final chunk in the sequence
- `duration_ms` (int, optional): Chunk duration in milliseconds
- `content_type` (string, optional): MIME type of the audio data

### 2. AI Response (Server → Client)

Response from the AI system after processing audio input.

```json
{
  "type": "ai_response",
  "timestamp": "2023-12-01T10:00:05Z",
  "session_id": "session-456",
  "response_text": "Hello! How can I help you today?",
  "audio_data": "SGVsbG8gSG93IGNhbiBJIGhlbHAgeW91IHRvZGF5Pw==",
  "emotion": "friendly",
  "confidence": 0.95,
  "processing_time_ms": 1500,
  "conversation_id": "conv-789"
}
```

**Fields:**
- `session_id` (string, required): Conversation session ID
- `response_text` (string): AI response as text
- `audio_data` (string): Base64-encoded audio response
- `emotion` (string): Detected/assigned emotion (`friendly`, `excited`, `calm`, etc.)
- `confidence` (float): AI confidence score (0.0-1.0)
- `processing_time_ms` (int): Processing time in milliseconds
- `conversation_id` (string): Unique conversation identifier

### 3. Ping/Pong (Bidirectional)

Connection health check mechanism.

**Ping (Client → Server):**
```json
{
  "type": "ping",
  "timestamp": "2023-12-01T10:00:00Z",
  "data": "optional-ping-data"
}
```

**Pong (Server → Client):**
```json
{
  "type": "pong",
  "timestamp": "2023-12-01T10:00:00Z",
  "data": "optional-ping-data"
}
```

### 4. Device Status (Client → Server)

Device status updates and telemetry.

```json
{
  "type": "device_status",
  "timestamp": "2023-12-01T10:00:00Z",
  "device_id": "device-123",
  "status": "online",
  "battery_level": 85,
  "metadata": {
    "firmware_version": "1.2.3",
    "signal_strength": -45,
    "temperature": 22.5
  }
}
```

**Fields:**
- `device_id` (string, required): Unique device identifier
- `status` (string, required): Device status (`online`, `offline`, `sleeping`, `error`)
- `battery_level` (int, optional): Battery percentage (0-100)
- `metadata` (object, optional): Additional device information

### 5. System Message (Server → Client)

System-wide announcements and notifications.

```json
{
  "type": "system_message",
  "timestamp": "2023-12-01T10:00:00Z",
  "priority": "normal",
  "title": "System Maintenance",
  "content": "Scheduled maintenance will occur at 2 AM UTC.",
  "actions": [
    {
      "id": "acknowledge",
      "label": "OK",
      "type": "button"
    },
    {
      "id": "learn_more",
      "label": "Learn More",
      "type": "link",
      "url": "https://example.com/maintenance"
    }
  ]
}
```

**Fields:**
- `priority` (string, required): Message priority (`low`, `normal`, `high`, `critical`)
- `title` (string, required): Message title
- `content` (string, required): Message content
- `actions` (array, optional): Available user actions

### 6. Error (Server → Client)

Error responses and notifications.

```json
{
  "type": "error",
  "timestamp": "2023-12-01T10:00:00Z",
  "error_code": "VALIDATION_ERROR",
  "message": "Invalid audio format",
  "details": "Sample rate must be between 8000 and 48000 Hz"
}
```

**Fields:**
- `error_code` (string): Standardized error code
- `message` (string): Human-readable error message
- `details` (string, optional): Additional error details

### 7. Authentication (Bidirectional)

Authentication-related messages.

**Refresh Request (Client → Server):**
```json
{
  "type": "auth",
  "timestamp": "2023-12-01T10:00:00Z",
  "action": "refresh",
  "token": "current-jwt-token"
}
```

**Refresh Response (Server → Client):**
```json
{
  "type": "auth",
  "timestamp": "2023-12-01T10:00:00Z",
  "action": "refreshed",
  "token": "new-jwt-token",
  "expires_at": "2023-12-02T10:00:00Z"
}
```

**Logout (Client → Server):**
```json
{
  "type": "auth",
  "timestamp": "2023-12-01T10:00:00Z",
  "action": "logout"
}
```

## Error Codes

| Code | Description |
|------|-------------|
| `VALIDATION_ERROR` | Message validation failed |
| `AUTHENTICATION_ERROR` | Authentication failed |
| `AUTHORIZATION_ERROR` | Insufficient permissions |
| `PROCESSING_ERROR` | Server processing error |
| `RATE_LIMIT_ERROR` | Too many requests |
| `DEVICE_NOT_FOUND` | Device not registered |
| `SESSION_EXPIRED` | Session has expired |

## Connection States

### Device Sessions

Each connected device maintains a session with the following states:

- **Active**: Device is connected and responsive
- **Inactive**: Device is disconnected
- **Sleeping**: Device is in low-power mode
- **Error**: Device is experiencing issues

### Session Management

- Sessions are automatically created on device connection
- Sessions persist device information and conversation context
- Sessions are cleaned up after device disconnection
- Session state is synchronized across server instances (when Redis is enabled)

## Usage Examples

### Connecting a Device

```javascript
// Generate device token (typically done server-side)
const deviceToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...";

// Connect to WebSocket
const ws = new WebSocket(`ws://localhost:8080/ws?token=${deviceToken}`);

ws.onopen = function() {
    console.log('Connected to WebSocket');
    
    // Send device status
    ws.send(JSON.stringify({
        type: 'device_status',
        device_id: 'device-123',
        status: 'online',
        battery_level: 85
    }));
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
    
    if (message.type === 'ai_response') {
        // Play audio response
        playAudio(message.audio_data);
    }
};
```

### Streaming Audio

```javascript
// Stream audio chunks
function streamAudio(audioData, sessionId) {
    const chunk = {
        type: 'audio_chunk',
        device_id: 'device-123',
        session_id: sessionId,
        audio_data: btoa(audioData), // Base64 encode
        sample_rate: 16000,
        encoding: 'pcm',
        chunk_sequence: chunkCounter++,
        is_final: false
    };
    
    ws.send(JSON.stringify(chunk));
}

// Final chunk
function finishAudioStream(sessionId) {
    const finalChunk = {
        type: 'audio_chunk',
        device_id: 'device-123',
        session_id: sessionId,
        audio_data: '',
        sample_rate: 16000,
        encoding: 'pcm',
        chunk_sequence: chunkCounter++,
        is_final: true
    };
    
    ws.send(JSON.stringify(finalChunk));
}
```

### Health Check

```javascript
// Send ping every 30 seconds
setInterval(() => {
    ws.send(JSON.stringify({
        type: 'ping',
        data: Date.now().toString()
    }));
}, 30000);

// Handle pong responses
ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    
    if (message.type === 'pong') {
        const latency = Date.now() - parseInt(message.data);
        console.log(`Connection latency: ${latency}ms`);
    }
};
```

## Best Practices

### Audio Streaming
- Use chunk sizes of 1-2 seconds for optimal latency
- Send `is_final: true` to indicate end of audio sequence
- Include sequence numbers for ordered processing
- Use appropriate encoding (PCM for low latency, Opus for compression)

### Error Handling
- Always handle error messages gracefully
- Implement exponential backoff for reconnections
- Validate message format before sending
- Log errors for debugging

### Performance
- Implement connection pooling for multiple devices
- Use compression for large payloads
- Monitor connection health with ping/pong
- Implement proper rate limiting

### Security
- Always use valid JWT tokens
- Rotate tokens regularly
- Validate origin in production
- Use TLS/WSS in production environments

## Monitoring and Debugging

### Server Logs
The server provides structured logging for all WebSocket events:

```json
{
  "level": "info",
  "time": "2023-12-01T10:00:00Z",
  "msg": "Client registered",
  "deviceID": "device-123",
  "userID": "user-456", 
  "sessionID": "session-789"
}
```

### Metrics
- Connection count by device type
- Message processing latency
- Error rates by error code
- Audio chunk processing times

### Health Endpoints
- `/health`: General server health
- `/ws/stats`: WebSocket connection statistics
- `/ws/devices`: Active device list