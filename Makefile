MAIN_PATH := cmd/server/main.go

APP_NAME := goAdBlock

BUILD_DIR := build

.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

.PHONY: run
run: build
	@echo "Running $(APP_NAME) with default settings..."
	@$(BUILD_DIR)/$(APP_NAME)

