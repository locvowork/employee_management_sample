# Makefile for apigateway

# Variables
BINARY_NAME=apigateway
BUILD_DIR=bin
MAIN_FILE=main.go
DOCKER_COMPOSE=docker-compose
DOCKER_COMPOSE_FILE=docker-compose.yml

# Go commands
GOBUILD=go build
GORUN=go run
GOTEST=go test
GOMOD=go mod

# Default environment
ENV?=development

.PHONY: all build run test clean deps tidy help \
        docker-build docker-up docker-down docker-logs docker-restart \
        run_dev run_prod run_up stop logs restart

all: help

# Environment commands
run_dev: ## Run in development mode (default)
	@echo "Starting development environment..."
	@$(MAKE) docker-build docker-up

run_up: ## Start the application in the current environment
	@echo "Starting $(ENV) environment..."
	@$(MAKE) docker-up

run_prod: ## Run in production mode
	@echo "Starting production environment..."
	@ENV=production $(MAKE) docker-build docker-up

# Docker Compose commands
docker-build: ## Build the Docker image
	@echo "Building Docker image..."
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) build --no-cache

docker-up: ## Start the containers
	@echo "Starting containers..."
	APP_ENV=$(ENV) $(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) up -d

stop: ## Stop and remove containers
	@echo "Stopping and removing containers..."
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) down

docker-logs: ## Show container logs
	@echo "Showing logs..."
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) logs -f

docker-restart: ## Restart the containers
	@echo "Restarting containers..."
	@$(MAKE) docker-down docker-up

# Go commands
build: ## Build the application
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)

run: ## Run the application
	@echo "Running..."
	$(GORUN) $(MAIN_FILE)

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

clean: ## Clean build directory and Docker resources
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)	

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download

tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	$(GOMOD) tidy

# Help command
help: ## Display this help screen
	@echo "\nAvailable commands:"
	@echo "\nEnvironment (Docker):"
	@echo "  make run_dev       Start development environment (default)"
	@echo "  make run_prod      Start production environment"
	@echo "  make run_up        Start the current environment"
	@echo "  make stop          Stop and remove containers"
	@echo "  make logs          Show container logs"
	@echo "  make restart       Restart the containers"
	@echo "\nDevelopment:"
	@echo "  make build         Build the application"
	@echo "  make run           Run the application locally"
	@echo "  make test          Run tests"
	@echo "  make deps          Download dependencies"
	@echo "  make tidy          Tidy go.mod"
	@echo "  make clean         Clean build directory and Docker resources"
	@echo ""
