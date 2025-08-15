#ifndef ARUNIKA_H
#define ARUNIKA_H

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stdbool.h>

// Audio configuration
#define SAMPLE_RATE 8000
#define BITS_PER_SAMPLE 16
#define CHANNELS 1
#define AUDIO_BUFFER_SIZE 1024

// Network configuration
#define MAX_SSID_LENGTH 32
#define MAX_PASSWORD_LENGTH 64
#define MAX_URL_LENGTH 256
#define MAX_DEVICE_ID_LENGTH 32

// WebSocket message types
#define MSG_TYPE_AUDIO_CHUNK "audio_chunk"
#define MSG_TYPE_PING "ping"
#define MSG_TYPE_PONG "pong"
#define MSG_TYPE_AI_RESPONSE "ai_response"

// Device states
typedef enum {
    DEVICE_STATE_INIT,
    DEVICE_STATE_CONNECTING,
    DEVICE_STATE_CONNECTED,
    DEVICE_STATE_RECORDING,
    DEVICE_STATE_PROCESSING,
    DEVICE_STATE_PLAYING,
    DEVICE_STATE_IDLE,
    DEVICE_STATE_ERROR
} device_state_t;

// Audio formats
typedef enum {
    AUDIO_FORMAT_PCM,
    AUDIO_FORMAT_MULAW,
    AUDIO_FORMAT_ALAW
} audio_format_t;

// Configuration structure
typedef struct {
    char wifi_ssid[MAX_SSID_LENGTH];
    char wifi_password[MAX_PASSWORD_LENGTH];
    char server_url[MAX_URL_LENGTH];
    char device_id[MAX_DEVICE_ID_LENGTH];
    uint16_t server_port;
    audio_format_t audio_format;
} device_config_t;

// Audio buffer structure
typedef struct {
    uint8_t *data;
    size_t size;
    size_t capacity;
    uint32_t sample_rate;
    audio_format_t format;
} audio_buffer_t;

// Function declarations

// Initialization
int device_init(void);
int audio_init(void);
int network_init(void);

// Configuration
int config_load(device_config_t *config);
int config_save(const device_config_t *config);

// Audio functions
int audio_start_recording(void);
int audio_stop_recording(void);
int audio_read_buffer(audio_buffer_t *buffer);
int audio_play_buffer(const audio_buffer_t *buffer);
int audio_set_volume(uint8_t volume);

// Network functions
int network_connect_wifi(const char *ssid, const char *password);
int network_disconnect_wifi(void);
bool network_is_connected(void);

// WebSocket functions
int websocket_connect(const char *url, uint16_t port, const char *path);
int websocket_disconnect(void);
int websocket_send_audio_chunk(const audio_buffer_t *buffer, int sequence);
int websocket_send_ping(void);
int websocket_receive_message(char *buffer, size_t buffer_size);
bool websocket_is_connected(void);

// Device control
int device_set_state(device_state_t state);
device_state_t device_get_state(void);
int device_handle_button_press(void);
int device_process_incoming_message(const char *message);

// Power management
int power_init(void);
int power_enter_sleep_mode(void);
int power_wake_up(void);
uint8_t power_get_battery_level(void);
bool power_is_charging(void);

// Utility functions
int base64_encode(const uint8_t *input, size_t input_len, char *output, size_t output_len);
int base64_decode(const char *input, uint8_t *output, size_t output_len);
uint32_t get_timestamp_ms(void);
void delay_ms(uint32_t ms);

// Error handling
typedef enum {
    ARUNIKA_OK = 0,
    ARUNIKA_ERROR_INIT = -1,
    ARUNIKA_ERROR_CONFIG = -2,
    ARUNIKA_ERROR_NETWORK = -3,
    ARUNIKA_ERROR_AUDIO = -4,
    ARUNIKA_ERROR_WEBSOCKET = -5,
    ARUNIKA_ERROR_MEMORY = -6,
    ARUNIKA_ERROR_TIMEOUT = -7,
    ARUNIKA_ERROR_INVALID_PARAM = -8
} arunika_error_t;

const char* arunika_error_string(arunika_error_t error);

#endif // ARUNIKA_H
