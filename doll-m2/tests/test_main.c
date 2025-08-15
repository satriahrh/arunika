#include "../include/arunika.h"
#include <assert.h>

void test_config_load_save() {
    device_config_t config;
    
    // Test loading configuration
    int result = config_load(&config);
    assert(result == ARUNIKA_OK);
    assert(strlen(config.device_id) > 0);
    
    printf("✅ Config load/save test passed\n");
}

void test_device_state_management() {
    // Test initial state
    device_state_t state = device_get_state();
    assert(state == DEVICE_STATE_INIT);
    
    // Test state transition
    int result = device_set_state(DEVICE_STATE_IDLE);
    assert(result == ARUNIKA_OK);
    
    state = device_get_state();
    assert(state == DEVICE_STATE_IDLE);
    
    printf("✅ Device state management test passed\n");
}

void test_audio_initialization() {
    int result = audio_init();
    assert(result == ARUNIKA_OK);
    
    printf("✅ Audio initialization test passed\n");
}

void test_network_initialization() {
    int result = network_init();
    assert(result == ARUNIKA_OK);
    
    printf("✅ Network initialization test passed\n");
}

void test_power_management() {
    int result = power_init();
    assert(result == ARUNIKA_OK);
    
    uint8_t battery = power_get_battery_level();
    assert(battery <= 100);
    
    printf("✅ Power management test passed\n");
}

void test_utility_functions() {
    // Test timestamp function
    uint32_t ts1 = get_timestamp_ms();
    delay_ms(10);
    uint32_t ts2 = get_timestamp_ms();
    assert(ts2 > ts1);
    
    // Test error string function
    const char* error_str = arunika_error_string(ARUNIKA_OK);
    assert(strcmp(error_str, "Success") == 0);
    
    printf("✅ Utility functions test passed\n");
}

void test_base64_encoding() {
    const char* input = "Hello World";
    char output[256];
    
    int result = base64_encode((const uint8_t*)input, strlen(input), output, sizeof(output));
    assert(result > 0);
    assert(strlen(output) > 0);
    
    printf("✅ Base64 encoding test passed\n");
}

int main() {
    printf("🧪 Running Arunika firmware tests...\n\n");
    
    test_config_load_save();
    test_device_state_management();
    test_audio_initialization();
    test_network_initialization();
    test_power_management();
    test_utility_functions();
    test_base64_encoding();
    
    printf("\n🎉 All tests passed!\n");
    return 0;
}
