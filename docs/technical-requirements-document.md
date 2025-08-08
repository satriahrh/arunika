# Technical Requirements Document

## Project Title: **Arunika - AI-Powered Interactive Talking Dolls**

---

### **1. Overview**

Arunika transforms ordinary dolls into interactive, intelligent companions through an integrated AI-powered conversational module. The system enables seamless voice interactions between children and dolls, providing educational, entertaining, and emotionally supportive experiences. Unlike DIY or assembly-required products, Arunika delivers complete, ready-to-use talking dolls manufactured with embedded AI technology.

The architecture leverages cloud-based AI services for natural language processing, speech recognition, and text-to-speech synthesis, while maintaining child-safe interaction protocols and parental controls for a secure, engaging experience.

---

### **2. Functional Requirements**

#### **Doll (IoT Device)**
1. **Voice Input:**
   - Capture child voice using high-quality digital microphone (INMP441 or equivalent)
   - Push-to-talk activation
   - Process ambient noise filtering for clear audio capture
   - Real-time audio streaming to cloud services via Wi-Fi

2. **Voice Output:**
   - Play AI-generated responses through integrated speaker system
   - Support multiple voice personalities and character adaptations
   - Implement volume control and audio quality optimization
   - Enable emotional tone variations in speech output

3. **Communication:**
   - Maintain persistent Wi-Fi connection with automatic reconnection
   - WebSocket-first approach for all real-time communication (audio streaming, responses, control messages)
   - **Session Management:** Single persistent connection with automatic reconnection and exponential backoff
   - **Connection Limits:** One WebSocket connection per device ID to prevent conflicts

4. **Power Management:**
   - 3.7V LiPo battery with 6-8 hours continuous operation
   - USB-C charging with LED status indicators
   - Sleep mode activation during inactivity to preserve battery
   - Low-battery warning system with graceful shutdown

5. **Physical Integration:**
   - Seamless integration into doll body without external modifications
   - Water-resistant components for child safety
   - Durable construction to withstand normal play activities
   - Child-safe materials and components certification

#### **Cloud Server**
1. **Speech Processing:**
   - Real-time speech-to-text conversion using Google Speech-to-Text or Azure Speech
   - Support for multiple languages and child speech recognition optimization
   - Noise cancellation and audio enhancement algorithms
   - Context-aware transcription with conversation history

2. **AI Conversation Engine:**
   - Integration with Google Gemini, OpenAI GPT, or similar LLM
   - Child-safe content filtering and response validation
   - Personality customization based on doll character and child preferences
   - Educational content integration and age-appropriate responses

3. **Text-to-Speech Synthesis:**
   - High-quality voice synthesis using Google TTS or Azure Speech
   - Multiple voice options and character-specific personalities
   - Emotional tone adaptation based on conversation context
   - Real-time audio streaming back to doll device

4. **Session Management:**
   - User authentication and child profile management
   - Conversation history tracking and context preservation
   - Multi-device support for family accounts
   - Privacy-compliant data handling with parental controls

5. **Content Management:**
   - Educational content library with curriculum alignment
   - Story generation and interactive storytelling capabilities
   - Character-specific dialogue and personality traits
   - Parental dashboard for monitoring and customization

#### **Manufacturing Integration**
1. **Supply Chain:**
   - Partnership agreements with doll manufacturers
   - Component sourcing and quality assurance protocols
   - Bulk production and assembly line integration
   - Quality control testing for each assembled unit

2. **Product Assembly:**
   - Standardized module installation process
   - Compatibility testing with various doll designs
   - Final product packaging and branding
   - User manual and setup guide creation

---

### **3. Non-Functional Requirements**

#### **Performance:**
- End-to-end response time (child input to doll response) must be under **2 seconds**
- Audio streaming latency should not exceed **200ms**
- 99.5% successful voice recognition accuracy for child speech
- Support for **10+ concurrent conversations** per server instance

#### **Reliability:**
- **99.9% cloud service uptime** with automatic failover
- Robust error handling and graceful degradation
- Automatic reconnection for lost network connections
- Local backup responses for critical system failures

#### **Scalability:**
- Support **50+ concurrent doll connections** initially
- Horizontal scaling to support 100+ is out of scope for MVP; plan upgrade path (broker, sharding) post-pilot

#### **Security & Privacy:**
- End-to-end encryption for all voice communications
- No persistent storage of child voice recordings
- Parental controls for data management and privacy settings

#### **Compatibility:**
- Support for various doll sizes and designs (12-18 inch dolls)
- Wi-Fi compatibility across common home networks
- Mobile app compatibility (iOS 14+, Android 8+)

---

### **4. Technical Architecture**

#### **Current Architecture (Cloud-Native with Message Broker)**
```
Doll Device ──WebSocket──▶ Load Balancer ──▶ API Gateway
     ↓                            ↓
Audio Chunks              Authentication Service
     ↓                            ↓
WebSocket Handler ─────────▶ Message Broker ◄──── STT Service
     ↓                      (Channel-based)        ↓
Device-Specific       ┌─────────┼─────────┐   Conversation
Routing              ↓         ↓         ↓   Engine (LLM)
                WebSocket  WebSocket  WebSocket     ↓
                Client 1   Client 2   Client 3   TTS Service
                                                     ↓
                Session Management & Analytics ◄─────┘
```

#### **MVP Architecture (Single Service, No Broker)**
```
Doll Device ──WebSocket──▶ Realtime API (Single Service)
     │                          │
     │                          ├──▶ Streaming STT (managed: Google/Azure)
     │                          ├──▶ LLM (Gemini/OpenAI) + safety filter
     │                          └──▶ TTS (managed) → audio chunks
     └─────────────◀───────────── Audio (streamed back to device)

Auth: mTLS device cert + JWT
State: In-memory session map per device (optional Redis for resume)
Deployment: Single region, 1–2 replicas behind managed LB (Cloud Run/ALB)
```

MVP Simplifications:
- No message broker, no API gateway; single service handles WS + pipeline.
- In-memory device routing; one WS per device; backpressure with bounded queues.
- Vendor streaming APIs for STT/TTS; send partial transcripts and partial audio.
- Minimal persistence (only auth/profile); no conversation history at rest (privacy by default).

#### **Device Architecture**
- **Microcontroller:** WEMOS LOLIN32 Lite (ESP32, Wi-Fi 802.11 b/g/n, dual-core)
- **Storage:** 4MB flash for firmware and local cache
- **Power:** 3.7V 1200mAh LiPo battery with USB-C charging
- **Connectivity:** Wi-Fi 802.11 b/g/n, Bluetooth 4.2 for setup and diagnostics
- **Audio Input:** INMP441 I2S digital microphone with noise cancellation
- **Audio Output:** MAX98357A I2S amplifier + 4Ω, 3W speaker

#### **Cloud Infrastructure**
- **Platform:** AWS, Google Cloud, or Azure single region deployment
- **Compute:** Container based deployment
- **Database:** MySQL for user data, Redis for session caching
- **Storage:** S3/Cloud Storage for audio processing and content
- **CDN:** CloudFront/CloudFlare for global audio delivery
- **Monitoring:** CloudWatch/Datadog for system observability

---

### **5. Data Flow**

1. **Doll Activation:**
   - Child speaks to doll (button activation)
   - Device initiates secure connection, can be used for upcoming interactions
   - Device captures audio
   - Authentication using device certificate and user session

2. **Audio Processing:**
   - Real-time audio chunk streaming via WebSocket
   - Streaming speech-to-text conversion with incremental results
   - Server forwards to managed STT; emits incremental transcripts to the device
   - In-memory device session routing (no broker), with ordering and backpressure
   - Context enrichment using conversation history and child profile

3. **AI Response Generation:**
   - Processed text sent to conversation engine (LLM)
   - Response generation with safety filtering and personality adaptation
   - Educational content integration and age-appropriate customization

4. **Audio Response:**
   - Generated text converted to speech with character voice
   - Audio stream compressed and delivered to doll device
   - Real-time playback through doll speaker system

5. **Session Management:**
   - Conversation logging for parental dashboard (with privacy controls)
   - Context preservation for ongoing conversations
   - Analytics and usage tracking for product improvement

---

### **6. API Specification**

#### **Device APIs**
- **Authentication:** `POST /api/v1/device/auth`
- **Health Check:** `GET /api/v1/device/health`
- **Session Management:** `POST /api/v1/session/start`, `DELETE /api/v1/session/end`

#### **Parent Dashboard APIs**
- **User Management:** `POST /api/v1/users/register`, `POST /api/v1/users/login`
- **Child Profiles:** `GET/POST/PUT /api/v1/children`
- **Conversation History:** `GET /api/v1/conversations`
- **Privacy Controls:** `PUT /api/v1/privacy/settings`

#### **WebSocket Communication (Primary)**
- **Endpoint:** `wss://api.arunika.com/ws?token=<device_jwt>`
- **Authentication:** Device certificate + JWT token in query parameter
- **Connection Management:** Single connection per device ID, automatic reconnection with exponential backoff
- **Message Types:** Audio chunks, AI responses, control messages, status updates
- **Compression:** WebSocket per-message deflate for audio data
- **Heartbeat:** 30-second ping/pong for connection monitoring

#### **WebSocket Message Formats**
- **Audio Chunk (Device → Server):**
  ```json
  {
    "type": "audio_chunk",
    "device_id": "ARUN_DEV_001234",
    "session_id": "sess_abc123def456",
    "audio_data": "base64_encoded_mulaw_audio",
    "sample_rate": 8000,
    "encoding": "MULAW",
    "timestamp": "2024-08-08T10:30:00Z",
    "chunk_sequence": 15,
    "is_final": false
  }
  ```

- **Streaming Configuration (Device → Server):**
  ```json
  {
    "type": "stream_config",
    "device_id": "ARUN_DEV_001234",
    "config": {
      "encoding": "MULAW",
      "sample_rate": 8000,
      "language": "en-US",
      "child_mode": true,
      "personality": "friendly_companion"
    }
  }
  ```

- **AI Response (Server → Device):**
  ```json
  {
    "type": "ai_response",
    "session_id": "sess_abc123def456",
    "response_text": "Hello! How are you feeling today?",
    "audio_data": "base64_encoded_response_audio",
    "emotion": "cheerful",
    "educational_content": true,
    "timestamp": "2024-08-08T10:30:02Z"
  }
  ```

- **Transcription Update (Server → Device):**
  ```json
  {
    "type": "transcription",
    "session_id": "sess_abc123def456",
    "text": "Hello, how are you",
    "is_final": false,
    "confidence": 0.85,
    "timestamp": "2024-08-08T10:30:01Z"
  }
  ```

---

### **7. Development Milestones**

#### **Phase 1: Core System (Weeks 1-4)**
- ESP32-S3 firmware development with audio processing
- Cloud infrastructure setup and basic API development
- Speech-to-text and text-to-speech integration
- Secure device authentication and communication

#### **Phase 2: AI Integration (Weeks 5-8)**
- LLM integration with child-safe filtering
- Personality engine and character voice development
- Conversation context management
- Basic parental dashboard and controls

#### **Phase 3: Manufacturing Integration (Weeks 9-12)**
- Doll manufacturing partnership agreements
- Module integration and assembly process
- Quality assurance and testing protocols
- First batch production (100 units)

#### **Phase 4: Pilot Launch (Weeks 13-16)**
- Alpha testing with target families
- User feedback collection and analysis
- Product iteration and improvement
- Preparation for scaled production

#### **Phase 5: Market Launch (Weeks 17-20)**
- Brand partnerships and licensing agreements
- Marketing and distribution channel setup
- Customer support and warranty programs
- Performance monitoring and optimization

---

### **8. Risks and Mitigation**

| **Risk**                          | **Impact**   | **Probability** | **Mitigation Strategy**                                |
|-----------------------------------|--------------|-----------------|-------------------------------------------------------|
| Child safety and content concerns | High         | Medium          | Robust content filtering, parental controls, regular audits |
| Privacy regulation compliance     | High         | Medium          | GDPR/COPPA compliance by design, data minimization   |
| Manufacturing partnership delays  | Medium       | High            | Multiple vendor relationships, phased rollout        |
| Cloud service latency/downtime    | Medium       | Low             | Multi-region deployment, edge computing, offline mode |
| Component supply chain issues     | Medium       | Medium          | Diverse supplier base, inventory buffers             |
| Competition from tech giants      | High         | Medium          | Focus on manufacturing partnerships, IP protection   |
| Battery life and charging issues  | Low          | Medium          | Extensive testing, user education, warranty coverage |

---

### **9. Tools and Resources**

#### **Hardware Development**
- **Development Boards:** ESP32-S3-DevKitC-1, audio development shields
- **Components:** INMP441 microphones, MAX98357A amplifiers, speakers
- **Tools:** Oscilloscope, audio analyzers, 3D printing for prototypes
- **Testing:** Anechoic chamber for audio testing, child safety lab

#### **Software Development**
- **Embedded:** ESP-IDF, FreeRTOS, audio processing libraries
- **Cloud Backend:** Go/Node.js with Echo/Express frameworks
- **AI Services:** Google Cloud AI, Azure Cognitive Services, OpenAI API
- **Mobile/Web:** React Native for mobile app, React for dashboard
- **Testing:** Jest, Cypress for frontend; Go test, Postman for backend

#### **Manufacturing**
- **CAD Software:** SolidWorks, Fusion 360 for mechanical design
- **PCB Design:** Altium Designer, KiCad for electronics
- **Quality Assurance:** Statistical process control, automated testing
- **Supply Chain:** ERP systems for inventory and production management

#### **Infrastructure**
- **Cloud Platform:** AWS/GCP with CDN and global load balancing
- **Monitoring:** Datadog, CloudWatch for system observability
- **Security:** Vault for secrets management, security scanning tools
- **Analytics:** Mixpanel, Google Analytics for user behavior tracking

---

### **10. Compliance and Certification**

#### **Safety Standards**
- **CPSC (Consumer Product Safety Commission)** compliance for toys
- **CE marking** for European market entry
- **FCC certification** for wireless device operation
- **UL listing** for electrical safety standards

#### **Privacy Regulations**
- **GDPR** compliance for European users
- **COPPA** compliance for children under 13
- **California Consumer Privacy Act (CCPA)** compliance
- **Data localization** requirements per region

#### **Quality Certifications**
- **ISO 9001** for quality management systems
- **ISO 27001** for information security management
- **SOC 2 Type II** for service organization controls
- **WCAG 2.1** for accessibility compliance

---

### **11. Glossary**

- **STT:** Speech-to-Text - Converting spoken words to written text
- **TTS:** Text-to-Speech - Converting written text to spoken words
- **LLM:** Large Language Model - AI system for natural language understanding
- **COPPA:** Children's Online Privacy Protection Act
- **GDPR:** General Data Protection Regulation
- **ESP32-S3:** Advanced microcontroller with Wi-Fi and AI acceleration
- **I2S:** Inter-IC Sound - Digital audio interface protocol
- **WebSocket:** Protocol for real-time bidirectional communication
- **CDN:** Content Delivery Network for global content distribution
- **API Gateway:** Entry point for managing and routing API requests
