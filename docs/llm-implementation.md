# LLM Interface Implementation

## Overview
This document describes the Gemini LLM interface implementation for the Arunika project.

## Implementation Details

### Files Created
1. `adapters/llm/gemini.go` - Main Gemini LLM implementation
2. `adapters/llm/gemini_chat_session.go` - Chat session management

### Features Implemented

#### GeminiLLM
- **Child-friendly safety settings**: Configured with appropriate harm thresholds
- **Retry logic**: Automatic retry with exponential backoff for failed API calls
- **Fallback responses**: Child-friendly fallback messages when API fails
- **Environment-based configuration**: Uses `GEMINI_API_KEY` environment variable

#### GeminiChatSession
- **Conversation history management**: Maintains full chat history
- **Context conversion**: Converts internal chat format to Gemini API format
- **Session-based conversations**: Each session maintains its own history
- **Graceful error handling**: Returns fallback responses instead of errors

### Configuration

#### Environment Variables
```bash
export GEMINI_API_KEY="your_gemini_api_key_here"
```

#### Safety Settings
- **Harassment**: Block medium and above
- **Hate Speech**: Block medium and above  
- **Sexually Explicit**: Block low and above
- **Dangerous Content**: Block low and above

#### Generation Parameters
- **Temperature**: 0.7 (slightly creative but controlled)
- **TopP**: 0.8
- **TopK**: 40
- **Max Output Tokens**: 500

### Usage Example

```go
import (
    "context"
    "go.uber.org/zap"
    "github.com/satriahrh/arunika/server/adapters/llm"
    "github.com/satriahrh/arunika/server/domain/repositories"
)

// Create LLM instance
logger, _ := zap.NewDevelopment()
geminiLLM, err := llm.NewGeminiLLM(logger)
if err != nil {
    // Handle error
}

// Create chat session
ctx := context.Background()
history := []repositories.ChatMessage{} // existing history
session, err := geminiLLM.GenerateChat(ctx, history)
if err != nil {
    // Handle error
}

// Send message
userMessage := repositories.ChatMessage{
    Role: repositories.UserRole,
    Content: "Hello, how are you?",
}

response, err := session.SendMessage(ctx, userMessage)
if err != nil {
    // Handle error (though implementation returns fallback instead of error)
}

fmt.Printf("Doll response: %s\n", response.Content)
```

### Fallback Responses
When the API fails or returns empty responses, the system uses these child-friendly fallbacks:
- "I'm thinking really hard about that, can you ask me again?"
- "My brain needs a little rest, let's try talking about something else!"
- "I'm having trouble understanding right now, but I'm still here with you!"
- "Let me think about that... maybe you can help me by asking in a different way?"
- "I'm learning new things every day! Can you tell me more about what you're thinking?"

### Integration Notes

This implementation is designed to be integrated with:
1. **Session Management**: Use session IDs to maintain separate conversations
2. **Audio Sessions**: Each audio session can have its own chat session
3. **WebSocket Handlers**: Integrate with existing websocket message processing
4. **Error Handling**: All errors are gracefully handled with fallback responses

### Next Steps for Integration

1. **Add to audio session workflow**: Include LLM chat session in `AudioSession` struct
2. **Environment configuration**: Set up `GEMINI_API_KEY` in deployment environment
3. **Session persistence**: Consider persisting chat history for longer conversations
4. **Rate limiting**: Add rate limiting for API calls if needed
5. **Monitoring**: Add metrics and monitoring for LLM usage

### Dependencies Added
- `google.golang.org/genai` - Official Google Generative AI Go SDK
