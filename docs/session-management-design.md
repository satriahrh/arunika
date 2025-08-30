# Session Management Design

## WebSocket Communication Protocol

### Message Types

We will use the following intuitive message types for WebSocket communication:

- `listening_start`: Client starts listening to user's speech (sent by client)
- `listening_end`: Client stops listening and signals end of user input (sent by client)
- `speaking_start`: Server begins sending synthesized speech to client (sent by server)
- `speaking_end`: Server signals the end of synthesized speech output (sent by server)

### Session Management Flow

#### Session Initialization Flow

When a client connects and wants to start a conversation:

```
CLIENT                                  SERVER
  |                                       |
  |  WebSocket Connection + Auth Token    |
  |-------------------------------------->|
  |                                       | [Authenticate device]
  |       WebSocket Connected             |
  |<--------------------------------------|
  |                                       |
  |  {                                    |
  |    "type": "listening_start",         |
  |    "session_id": "<optional>",        | [If missing or invalid,
  |    "create_new": true|false           |  generate new session_id]
  |  }                                    |
  |-------------------------------------->|
  |                                       | [Create session in DB]
  |  {                                    |
  |    "type": "listening_started",       |
  |    "session_id": "<id>",              |
  |    "is_new_session": true|false,      |
  |    "expires_at": timestamp            |
  |  }                                    |
  |<--------------------------------------|
  |                                       |
```

#### Audio Processing Flow

```
CLIENT                                  SERVER
  |                                       |
  |  [Binary Audio Data]                  |
  |-------------------------------------->| [Process with STT]
  |  ...more chunks...                    | [Accumulate transcription]
  |-------------------------------------->|
  |                                       |
  |  {                                    |
  |    "type": "listening_end",           |
  |    "session_id": "<id>"               |
  |  }                                    |
  |-------------------------------------->|
  |                                       | [Finalize STT]
  |                                       | [Process with LLM]
  |                                       | [Generate TTS]
  |  {                                    |
  |    "type": "speaking_start",          |
  |    "session_id": "<id>",              |
  |    "response_text": "text version"    |
  |  }                                    |
  |<--------------------------------------|
  |                                       |
  |  [Binary Audio Data]                  |
  |<--------------------------------------|
  |  ...more chunks...                    |
  |<--------------------------------------|
  |                                       |
  |  {                                    |
  |    "type": "speaking_end",            |
  |    "session_id": "<id>"               |
  |  }                                    |
  |<--------------------------------------|
  |                                       |
```

## Database Integration

### NoSQL Database Selection

- **MongoDB** will be used for rapid development with flexible schema
- Collections needed:
  - `sessions`: Store active conversation sessions
  - `messages`: Store conversation history linked to sessions
  - `devices`: Store device information

### Data Structures

#### Session Document Structure
```json
{
  "_id": "ObjectId",
  "session_id": "UUID",
  "device_id": "DeviceIdentifier",
  "created_at": "Timestamp",
  "last_active_at": "Timestamp",
  "expires_at": "Timestamp",
  "status": "active|expired|terminated",
  "metadata": {
    "total_interactions": 0,
    "language": "id-ID",
    "user_preferences": {}
  }
}
```

#### Message Document Structure
```json
{
  "_id": "ObjectId",
  "session_id": "UUID",
  "device_id": "DeviceIdentifier",
  "timestamp": "Timestamp",
  "role": "user|assistant",
  "content": "Message text content",
  "audio_file_path": "optional/path/to/audio.wav",
  "duration_ms": 1500,
  "metadata": {
    "transcription_confidence": 0.95,
    "emotion": "neutral"
  }
}
```

## Implementation Considerations

### 1. Repository Interface for Sessions
- Define a `SessionRepository` interface in the domain layer
- Implement MongoDB adapter for this interface

### 2. Session Expiration & Cleanup
- Implement a background task to clean up expired sessions
- Consider TTL indexes in MongoDB for automatic expiration

### 3. Error Handling for Session Edge Cases
- Handle session not found scenarios
- Handle race conditions when multiple devices access same session
- Handle connection drops during conversation

### 4. Session Context Transfer
- How to maintain conversation context when re-using a session
- LLM integration for conversation history retrieval

### 5. Authentication Enhancements
- Session token in addition to device token
- Permission-based access to sessions

### 6. Monitoring and Analytics
- Session metrics collection
- Conversation quality analysis
- Performance tracking

### 7. Scaling Considerations
- Session sharding for high volume
- Connection pooling for MongoDB
- WebSocket connection distribution across multiple servers

### 8. Session Recovery Mechanism
- What happens if server crashes during a session
- How to resume interrupted conversations

### 9. Multiple Device Support
- Can multiple devices access the same conversation session?
- How to handle synchronization if yes

### 10. Rate Limiting and Abuse Prevention
- Limit number of active sessions per device
- Prevent session creation spam

## Next Steps

1. **Define MongoDB Schema and Indexes**
   - Create schema definition files
   - Plan appropriate indexes for query performance

2. **Implement Session Repository**
   - Create MongoDB adapter for session storage
   - Add CRUD operations for sessions

3. **Update WebSocket Handler**
   - Modify to use the new message types
   - Integrate with session repository
   - Implement session validation logic

4. **Enhance Client Communication**
   - Document the new protocol for client developers
   - Create example client code for testing

5. **Add Monitoring**
   - Session creation/usage metrics
   - Performance monitoring for database operations
