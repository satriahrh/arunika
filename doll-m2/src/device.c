#include "arunika.h"

// Global device state
static device_state_t current_state = DEVICE_STATE_INIT;

int device_init(void) {
    printf("Initializing Arunika device...\n");
    
    // Load configuration
    device_config_t config;
    if (config_load(&config) != ARUNIKA_OK) {
        printf("Failed to load configuration\n");
        return ARUNIKA_ERROR_CONFIG;
    }
    
    // Initialize audio subsystem
    if (audio_init() != ARUNIKA_OK) {
        printf("Failed to initialize audio\n");
        return ARUNIKA_ERROR_AUDIO;
    }
    
    // Initialize network subsystem
    if (network_init() != ARUNIKA_OK) {
        printf("Failed to initialize network\n");
        return ARUNIKA_ERROR_NETWORK;
    }
    
    // Initialize power management
    if (power_init() != ARUNIKA_OK) {
        printf("Failed to initialize power management\n");
        return ARUNIKA_ERROR_INIT;
    }
    
    device_set_state(DEVICE_STATE_IDLE);
    printf("Device initialization complete\n");
    
    return ARUNIKA_OK;
}

int device_set_state(device_state_t state) {
    printf("Device state transition: %d -> %d\n", current_state, state);
    current_state = state;
    return ARUNIKA_OK;
}

device_state_t device_get_state(void) {
    return current_state;
}

int device_handle_button_press(void) {
    printf("Button press detected\n");
    
    switch (current_state) {
        case DEVICE_STATE_IDLE:
            // Start recording
            if (audio_start_recording() == ARUNIKA_OK) {
                device_set_state(DEVICE_STATE_RECORDING);
            }
            break;
            
        case DEVICE_STATE_RECORDING:
            // Stop recording and process
            audio_stop_recording();
            device_set_state(DEVICE_STATE_PROCESSING);
            break;
            
        default:
            printf("Button press ignored in current state: %d\n", current_state);
            break;
    }
    
    return ARUNIKA_OK;
}

int device_process_incoming_message(const char *message) {
    printf("Processing incoming message: %s\n", message);
    
    // TODO: Parse JSON message and handle different types
    // For now, assume it's an AI response with audio data
    
    if (strstr(message, "ai_response") != NULL) {
        // Extract audio data and play it
        device_set_state(DEVICE_STATE_PLAYING);
        
        // TODO: Parse JSON, extract base64 audio data, decode and play
        
        // Simulate audio playback delay
        delay_ms(2000);
        
        device_set_state(DEVICE_STATE_IDLE);
    }
    
    return ARUNIKA_OK;
}
