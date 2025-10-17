.PHONY: build run clean deps test install

# Переменные
BINARY_NAME=ovpn-bot
ADMIN_BINARY_NAME=ovpn-admin
BUILD_DIR=bin
MAIN_PATH=cmd/bot/main.go
ADMIN_PATH=cmd/admin/main.go

# Сборка приложения
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@go build -o $(BUILD_DIR)/$(ADMIN_BINARY_NAME) $(ADMIN_PATH)
	@echo "Build completed: $(BUILD_DIR)/$(BINARY_NAME) and $(BUILD_DIR)/$(ADMIN_BINARY_NAME)"

# Запуск приложения
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

# Запуск в режиме отладки
debug: build
	@echo "Running $(BINARY_NAME) in DEBUG mode..."
	@DEBUG=true ./$(BUILD_DIR)/$(BINARY_NAME)

# Установка зависимостей
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Очистка
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

# Тесты
test:
	@echo "Running tests..."
	@go test -v ./...

# Установка в систему
install: build
	@echo "Installing $(BINARY_NAME)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installed: /usr/local/bin/$(BINARY_NAME)"

# Создание директорий
setup:
	@echo "Creating directories..."
	@mkdir -p data
	@mkdir -p .ovpn
	@echo "Directories created"

# Копирование примера конфигурации
config:
	@echo "Creating .env file from example..."
	@cp .env.example .env
	@echo "Please edit .env file with your configuration"

# Полная настройка
init: setup config deps
	@echo "Initialization completed"
	@echo "Please edit .env file with your bot token and run 'make run'"

# Генерация кодов активации
generate-codes:
	@echo "Generating activation codes..."
	@./$(BUILD_DIR)/$(ADMIN_BINARY_NAME) -limit=1 -count=5

# Генерация кодов с параметрами
generate-codes-custom:
	@echo "Usage: make generate-codes-custom LIMIT=5 COUNT=10"
	@./$(BUILD_DIR)/$(ADMIN_BINARY_NAME) -limit=$(LIMIT) -count=$(COUNT)

# Помощь
help:
	@echo "Available commands:"
	@echo "  build                    - Build the application"
	@echo "  run                      - Build and run the application"
	@echo "  debug                    - Build and run in DEBUG mode"
	@echo "  deps                     - Install dependencies"
	@echo "  clean                    - Clean build artifacts"
	@echo "  test                     - Run tests"
	@echo "  install                  - Install to system"
	@echo "  setup                    - Create necessary directories"
	@echo "  config                   - Create .env from example"
	@echo "  init                     - Full initialization"
	@echo "  generate-codes           - Generate 5 activation codes with limit 1"
	@echo "  generate-codes-custom    - Generate codes with custom limit and count"
	@echo "  help                     - Show this help"
