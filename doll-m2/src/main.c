#include "arunika.h"

// Main application loop
int main(void) {
    printf("Starting Arunika Doll M2 Firmware\n");
    
    // Initialize device
    if (device_init() != ARUNIKA_OK) {
        printf("Device initialization failed\n");
        return -1;
    }
    
    // Connect to WiFi
    printf("Connecting to WiFi...\n");
    device_config_t device_config;
    config_load(&device_config);
    if (network_connect_wifi(device_config.wifi_ssid, device_config.wifi_password) != ARUNIKA_OK) {
        printf("WiFi connection failed\n");
        return -1;
    }
    
    // Main application loop
    while (1) {
        // Check for button press
        // TODO: Implement actual button reading
        
        // Handle WebSocket messages
        if (websocket_is_connected()) {
            char message_buffer[1024];
            if (websocket_receive_message(message_buffer, sizeof(message_buffer)) > 0) {
                device_process_incoming_message(message_buffer);
            }
        } else {
            // Try to reconnect
            printf("Attempting WebSocket connection...\n");
            char ws_path[128];
            snprintf(ws_path, sizeof(ws_path), "/ws?device_id=%s", device_config.device_id);
            websocket_connect(device_config.server_url, device_config.server_port, ws_path);
        }
        
        // Handle audio recording
        if (device_get_state() == DEVICE_STATE_RECORDING) {
            audio_buffer_t buffer;
            if (audio_read_buffer(&buffer) == ARUNIKA_OK) {
                static int sequence = 0;
                websocket_send_audio_chunk(&buffer, sequence++);
            }
        }
        
        // Power management
        uint8_t battery_level = power_get_battery_level();
        if (battery_level < 10) {
            printf("Low battery: %d%%\n", battery_level);
            // TODO: Implement low battery handling
        }
        
        // Small delay to prevent busy waiting
        delay_ms(10);
    }
    
    return 0;
}
