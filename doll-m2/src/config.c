#include "arunika.h"

// Configuration storage (in real implementation, this would be in EEPROM/Flash)
static device_config_t default_config = {
    .wifi_ssid = "YourWiFiNetwork",
    .wifi_password = "YourWiFiPassword", 
    .server_url = "wss://api.arunika.com",
    .device_id = "ARUN_DEV_001234",
    .server_port = 443,
    .audio_format = AUDIO_FORMAT_MULAW
};

int config_load(device_config_t *config) {
    if (config == NULL) {
        return ARUNIKA_ERROR_INVALID_PARAM;
    }
    
    // TODO: Load from persistent storage (EEPROM, Flash, etc.)
    // For now, use default configuration
    memcpy(config, &default_config, sizeof(device_config_t));
    
    printf("Configuration loaded:\n");
    printf("  WiFi SSID: %s\n", config->wifi_ssid);
    printf("  Server URL: %s\n", config->server_url);
    printf("  Device ID: %s\n", config->device_id);
    printf("  Server Port: %d\n", config->server_port);
    
    return ARUNIKA_OK;
}

int config_save(const device_config_t *config) {
    if (config == NULL) {
        return ARUNIKA_ERROR_INVALID_PARAM;
    }
    
    // TODO: Save to persistent storage (EEPROM, Flash, etc.)
    memcpy(&default_config, config, sizeof(device_config_t));
    
    printf("Configuration saved\n");
    return ARUNIKA_OK;
}
