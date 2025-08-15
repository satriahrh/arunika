#include "arunika.h"
#include <time.h>

// Base64 encoding table
static const char base64_table[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

int base64_encode(const uint8_t *input, size_t input_len, char *output, size_t output_len) {
    if (!input || !output || input_len == 0) {
        return ARUNIKA_ERROR_INVALID_PARAM;
    }
    
    size_t encoded_len = 4 * ((input_len + 2) / 3);
    if (output_len < encoded_len + 1) {
        return ARUNIKA_ERROR_MEMORY;
    }
    
    size_t i, j;
    for (i = 0, j = 0; i < input_len; ) {
        uint32_t octet_a = i < input_len ? input[i++] : 0;
        uint32_t octet_b = i < input_len ? input[i++] : 0;
        uint32_t octet_c = i < input_len ? input[i++] : 0;
        
        uint32_t triple = (octet_a << 0x10) + (octet_b << 0x08) + octet_c;
        
        output[j++] = base64_table[(triple >> 3 * 6) & 0x3F];
        output[j++] = base64_table[(triple >> 2 * 6) & 0x3F];
        output[j++] = base64_table[(triple >> 1 * 6) & 0x3F];
        output[j++] = base64_table[(triple >> 0 * 6) & 0x3F];
    }
    
    // Add padding
    for (i = 0; i < (3 - input_len % 3) % 3; i++) {
        output[encoded_len - 1 - i] = '=';
    }
    
    output[encoded_len] = '\0';
    return encoded_len;
}

int base64_decode(const char *input, uint8_t *output, size_t output_len) {
    if (!input || !output) {
        return ARUNIKA_ERROR_INVALID_PARAM;
    }
    
    // TODO: Implement base64 decoding
    // For now, return error as not implemented
    return ARUNIKA_ERROR_INIT;
}

uint32_t get_timestamp_ms(void) {
    // TODO: Use proper ESP32 timer/RTC
    // For now, use system clock
    
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (uint32_t)(ts.tv_sec * 1000 + ts.tv_nsec / 1000000);
}

void delay_ms(uint32_t ms) {
    // TODO: Use proper ESP32 delay function
    // For now, use standard library
    
    struct timespec ts;
    ts.tv_sec = ms / 1000;
    ts.tv_nsec = (ms % 1000) * 1000000;
    nanosleep(&ts, NULL);
}

const char* arunika_error_string(arunika_error_t error) {
    switch (error) {
        case ARUNIKA_OK:
            return "Success";
        case ARUNIKA_ERROR_INIT:
            return "Initialization error";
        case ARUNIKA_ERROR_CONFIG:
            return "Configuration error";
        case ARUNIKA_ERROR_NETWORK:
            return "Network error";
        case ARUNIKA_ERROR_AUDIO:
            return "Audio error";
        case ARUNIKA_ERROR_WEBSOCKET:
            return "WebSocket error";
        case ARUNIKA_ERROR_MEMORY:
            return "Memory error";
        case ARUNIKA_ERROR_TIMEOUT:
            return "Timeout error";
        case ARUNIKA_ERROR_INVALID_PARAM:
            return "Invalid parameter";
        default:
            return "Unknown error";
    }
}
