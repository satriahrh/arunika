# WebSocket Session Management Protocol

## Overview

The Arunika WebSocket API now uses an intuitive session-based protocol for voice conversations with persistent session management backed by MongoDB.

## Authentication

WebSocket connections require JWT authentication via the `Authorization` header:

```
Authorization: Bearer <jwt_token>
```

## Message Protocol

### Client Messages

#### 1. Start Listening
**Message Type**: `listening_start`

```json
{
  "type": "listening_start"
}
```

**Purpose**: Signals that the client is ready to capture and send audio from the user.

**Server Response**:
```json
{
  "type": "listening_started",
  "session_id": "507f1f77bcf86cd799439011",
  "timestamp": 1640995200,
  "status": "ready"
}
```

#### 2. Audio Data
**Message Type**: Binary WebSocket message

**Purpose**: Streams raw audio data while listening is active.

**Format**: Binary PCM audio data (LINEAR16, 48kHz)

#### 3. Stop Listening
**Message Type**: `listening_end`

```json
{
  "type": "listening_end"
}
```

**Purpose**: Signals that the user has finished speaking.

**Server Response**:
```json
{
  "type": "listening_ended",
  "session_id": "507f1f77bcf86cd799439011",
  "timestamp": 1640995200,
  "status": "completed"
}
```

### Server Messages

#### 1. Start Speaking
**Message Type**: `speaking_start`

```json
{
  "type": "speaking_start",
  "session_id": "507f1f77bcf86cd799439011",
  "timestamp": 1640995200
}
```

**Purpose**: Indicates that the server is about to send synthesized speech audio.

#### 2. Audio Response
**Message Type**: Binary WebSocket message

**Purpose**: Streams synthesized speech audio to the client.

**Format**: Binary audio data in PCM format

#### 3. Stop Speaking
**Message Type**: `speaking_end`

```json
{
  "type": "speaking_end",
  "session_id": "507f1f77bcf86cd799439011",
  "timestamp": 1640995200
}
```

**Purpose**: Indicates that all audio for the current response has been sent.

## Session Management

### Session Lifecycle

1. **Session Creation**: Automatically created when a device sends `listening_start` for the first time
2. **Session Continuation**: If the last message was within 30 minutes, the existing session continues
3. **Session Renewal**: If the last message was more than 30 minutes ago, a new session is created
4. **Session Expiration**: Sessions automatically expire 24 hours after the last activity

### Session Rules

- **One Session Per Device**: Each device can only have one active session at a time
- **Persistent History**: Conversation history is maintained within each session
- **Automatic Cleanup**: Expired sessions are automatically cleaned up by a background service

### Session Data Structure

Sessions are stored in MongoDB with the following structure:

```json
{
  "_id": "ObjectId",
  "device_id": "device-ARUNIKA001",
  "created_at": "2024-01-01T10:00:00Z",
  "last_active_at": "2024-01-01T10:30:00Z",
  "last_message_at": "2024-01-01T10:29:00Z",
  "expires_at": "2024-01-02T10:30:00Z",
  "status": "active",
  "messages": [
    {
      "timestamp": "2024-01-01T10:25:00Z",
      "role": "user",
      "content": "Hello, how are you?",
      "duration_ms": 1500,
      "metadata": {
        "transcription_confidence": 0.95
      }
    },
    {
      "timestamp": "2024-01-01T10:25:30Z",
      "role": "assistant",
      "content": "I'm doing well, thank you! How can I help you today?",
      "duration_ms": 2200,
      "metadata": {}
    }
  ],
  "metadata": {
    "language": "id-ID",
    "user_preferences": {}
  }
}
```

## Error Handling

### Error Response Format

```json
{
  "type": "error",
  "timestamp": 1640995200,
  "message": "Error description"
}
```

### Common Error Scenarios

1. **Already Listening**: Sent `listening_start` while already listening
2. **Not Listening**: Sent `listening_end` while not listening
3. **Session Creation Failed**: Unable to create or retrieve session
4. **Speech Recognition Error**: STT service unavailable
5. **Speech Synthesis Error**: TTS service unavailable

## Example Client Implementation

```javascript
class ArunikaClient {
  constructor(serverUrl, authToken) {
    this.serverUrl = serverUrl;
    this.authToken = authToken;
    this.isListening = false;
    this.currentSessionId = null;
  }

  connect() {
    this.ws = new WebSocket(this.serverUrl, [], {
      headers: {
        'Authorization': `Bearer ${this.authToken}`
      }
    });

    this.ws.onmessage = (event) => {
      if (event.data instanceof Blob) {
        this.handleAudioData(event.data);
      } else {
        const message = JSON.parse(event.data);
        this.handleTextMessage(message);
      }
    };
  }

  startListening() {
    if (this.isListening) return;
    
    this.ws.send(JSON.stringify({
      type: 'listening_start'
    }));
  }

  sendAudioChunk(audioData) {
    if (this.isListening) {
      this.ws.send(audioData);
    }
  }

  stopListening() {
    if (!this.isListening) return;
    
    this.ws.send(JSON.stringify({
      type: 'listening_end'
    }));
  }

  handleTextMessage(message) {
    switch (message.type) {
      case 'listening_started':
        this.isListening = true;
        this.currentSessionId = message.session_id;
        console.log('Started listening, session:', message.session_id);
        break;
        
      case 'listening_ended':
        this.isListening = false;
        console.log('Stopped listening');
        break;
        
      case 'speaking_start':
        console.log('Assistant starting to speak');
        this.onSpeakingStart?.(message.session_id);
        break;
        
      case 'speaking_end':
        console.log('Assistant finished speaking');
        this.onSpeakingEnd?.(message.session_id);
        break;
        
      case 'error':
        console.error('Error:', message.message);
        this.onError?.(message.message);
        break;
    }
  }

  handleAudioData(audioBlob) {
    this.onAudioReceived?.(audioBlob);
  }
}

// Usage
const client = new ArunikaClient('ws://localhost:8080/ws', 'your-jwt-token');
client.connect();

// Set up audio handling
client.onAudioReceived = (audioBlob) => {
  // Play the received audio
  const audio = new Audio(URL.createObjectURL(audioBlob));
  audio.play();
};

// Start a conversation
client.startListening();
// ... capture and send audio data
client.sendAudioChunk(audioData);
client.stopListening();
```

## Migration from Legacy Protocol

The system maintains backward compatibility with the legacy `audio_session_start` and `audio_session_end` messages while encouraging migration to the new protocol:

### Legacy Support

```json
// Still supported
{"type": "audio_session_start", "session_id": "legacy-id"}
{"type": "audio_session_end", "session_id": "legacy-id"}
```

### Recommended Migration

```json
// New protocol
{"type": "listening_start"}
{"type": "listening_end"}
```

The new protocol provides:
- Automatic session management
- Persistent conversation history
- Better error handling
- Intuitive message naming
- Device isolation guarantees