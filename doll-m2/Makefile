CC = gcc
CFLAGS = -Wall -Wextra -std=c99 -Iinclude
LDFLAGS = 

# Directories
SRCDIR = src
INCDIR = include
LIBDIR = lib
TESTDIR = tests
BUILDDIR = build
OBJDIR = $(BUILDDIR)/obj

# Source files
SOURCES = $(wildcard $(SRCDIR)/*.c)
OBJECTS = $(SOURCES:$(SRCDIR)/%.c=$(OBJDIR)/%.o)
TARGET = $(BUILDDIR)/arunika-doll-m2

# Test files
TEST_SOURCES = $(wildcard $(TESTDIR)/*.c)
TEST_OBJECTS = $(TEST_SOURCES:$(TESTDIR)/%.c=$(OBJDIR)/test_%.o)
TEST_TARGET = $(BUILDDIR)/test_runner

# Default target
all: $(TARGET)

# Create build directories
$(BUILDDIR):
	mkdir -p $(BUILDDIR)

$(OBJDIR):
	mkdir -p $(OBJDIR)

# Build main target
$(TARGET): $(OBJECTS) | $(BUILDDIR)
	$(CC) $(OBJECTS) -o $@ $(LDFLAGS)

# Build object files
$(OBJDIR)/%.o: $(SRCDIR)/%.c | $(OBJDIR)
	$(CC) $(CFLAGS) -c $< -o $@

# Build test object files
$(OBJDIR)/test_%.o: $(TESTDIR)/%.c | $(OBJDIR)
	$(CC) $(CFLAGS) -c $< -o $@

# Build and run tests
test: $(TEST_TARGET)
	./$(TEST_TARGET)

$(TEST_TARGET): $(TEST_OBJECTS) $(filter-out $(OBJDIR)/main.o,$(OBJECTS)) | $(BUILDDIR)
	$(CC) $^ -o $@ $(LDFLAGS)

# Clean build files
clean:
	rm -rf $(BUILDDIR)

# Install dependencies (for ESP32 development)
install-deps:
	@echo "Installing ESP-IDF dependencies..."
	@echo "Please follow ESP-IDF installation guide for your platform"
	@echo "https://docs.espressif.com/projects/esp-idf/en/latest/esp32/get-started/"

# ESP32 specific targets (will be added when porting to ESP32)
esp32-build:
	@echo "ESP32 build target - to be implemented when porting to ESP-IDF"

esp32-flash:
	@echo "ESP32 flash target - to be implemented when porting to ESP-IDF"

esp32-monitor:
	@echo "ESP32 monitor target - to be implemented when porting to ESP-IDF"

# Help target
help:
	@echo "Available targets:"
	@echo "  all          - Build the main application"
	@echo "  test         - Build and run tests"
	@echo "  clean        - Clean build files"
	@echo "  install-deps - Install development dependencies"
	@echo "  esp32-build  - Build for ESP32 (future)"
	@echo "  esp32-flash  - Flash to ESP32 (future)"
	@echo "  esp32-monitor- Monitor ESP32 output (future)"
	@echo "  help         - Show this help message"

.PHONY: all test clean install-deps esp32-build esp32-flash esp32-monitor help
