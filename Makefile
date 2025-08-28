# OneWay Support Bot - Simplified Makefile

# Variables
APP_NAME := oneway_subot
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Colors for output
GREEN := \033[0;32m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: help
help: ## Show available commands
	@echo '$(GREEN)OneWay Support Bot - Available commands:$(NC)'
	@echo ''
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(BLUE)%-15s$(NC) %s\n", $$1, $$2}'

# Development
.PHONY: deps
deps: ## Download dependencies
	@echo '$(BLUE)Downloading dependencies...$(NC)'
	go mod download
	go mod tidy

.PHONY: build
build: ## Build the application locally
	@echo '$(BLUE)Building $(APP_NAME)...$(NC)'
	go build -o bin/$(APP_NAME) ./cmd/bot

.PHONY: run
run: ## Run the application locally
	@echo '$(BLUE)Running $(APP_NAME)...$(NC)'
	go run ./cmd/bot

.PHONY: dev
dev: ## Run in development mode with hot reload
	@echo '$(BLUE)Running in development mode...$(NC)'
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo 'Installing air for hot reload...'; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# Testing
.PHONY: test
test: ## Run tests
	@echo '$(BLUE)Running tests...$(NC)'
	go test -v ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage
	@echo '$(BLUE)Running tests with coverage...$(NC)'
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo '$(GREEN)Coverage report: coverage.html$(NC)'

# Code quality
.PHONY: fmt
fmt: ## Format code
	@echo '$(BLUE)Formatting code...$(NC)'
	go fmt ./...

.PHONY: lint
lint: ## Run linter
	@echo '$(BLUE)Running linter...$(NC)'
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo 'Installing golangci-lint...'; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

.PHONY: vet
vet: ## Run go vet
	@echo '$(BLUE)Running go vet...$(NC)'
	go vet ./...

# Docker Compose shortcuts (main deployment method)
.PHONY: up
up: ## Start all services (main way to run the app)
	@echo '$(BLUE)Starting services with Docker Compose...$(NC)'
	docker-compose up -d

.PHONY: up-build
up-build: ## Start all services and rebuild
	@echo '$(BLUE)Building and starting services...$(NC)'
	docker-compose up -d --build

.PHONY: down
down: ## Stop all services
	@echo '$(BLUE)Stopping services...$(NC)'
	docker-compose down

.PHONY: logs
logs: ## Show all logs
	docker-compose logs -f

.PHONY: logs-bot
logs-bot: ## Show bot logs only
	docker-compose logs -f bot

.PHONY: status
status: ## Show containers status
	docker-compose ps

.PHONY: restart
restart: ## Restart bot service
	docker-compose restart bot

# Database shortcuts
.PHONY: db-shell
db-shell: ## Connect to MongoDB shell
	docker-compose exec mongo mongosh -u root -p

.PHONY: db-logs
db-logs: ## Show database logs
	docker-compose logs -f mongo

# Cleanup
.PHONY: clean
clean: ## Clean build artifacts
	@echo '$(BLUE)Cleaning...$(NC)'
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -cache

.PHONY: clean-docker
clean-docker: ## Clean Docker artifacts
	@echo '$(BLUE)Cleaning Docker...$(NC)'
	docker-compose down --volumes --remove-orphans
	docker system prune -f

# Setup
.PHONY: setup
setup: ## Setup development environment
	@echo '$(BLUE)Setting up development environment...$(NC)'
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo '$(GREEN)Setup completed! Use "make up" to start the app$(NC)'

# Default target
.DEFAULT_GOAL := help