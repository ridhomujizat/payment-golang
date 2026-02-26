# Go Boilerplate Makefile

# Variables
APP_NAME=go-boilerplate
CMD_DIR=./cmd/api
BUILD_DIR=./build
BINARY_NAME=api

# Database connection from .env if available
include .env
DB_URL=postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

.PHONY: help build run test clean docker-build docker-up docker-down migrate

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go

run: ## Run the application
	@echo "Running $(APP_NAME)..."
	@go run $(CMD_DIR)/main.go

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	@go mod tidy

mod-download: ## Download go modules
	@echo "Downloading go modules..."
	@go mod download

# Database migration commands
migrate: ## Run GORM AutoMigrate
	@echo "Running GORM AutoMigrate..."
	@go run cmd/migrate/main.go

# Docker commands
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(APP_NAME):latest .

docker-up: ## Start Docker containers
	@echo "Starting Docker containers..."
	@docker-compose up -d

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	@docker-compose down

docker-logs: ## Show Docker logs
	@docker-compose logs -f

# Development
dev: ## Run in development mode with hot reload (requires air)
	@echo "Running in development mode..."
	@air

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@go install github.com/swaggo/swag/cmd/swag@latest

# Swagger
swagger: ## Generate swagger documentation
	@echo "Generating swagger docs..."
	@swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

# lint
lint: ## Run golangci-lint
	@echo "Running linter..."
	@golangci-lint run

# format
fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...

run-app:
	@echo "Running application..."
	go run cmd/api/main.go


.DEFAULT_GOAL := help
