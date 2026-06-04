SHELL := /bin/bash

# Module directories
STREAM_GATEWAY_DIR   := stream-gateway
DORMITORY_SERVICE_DIR := dormitory-service-go
FACE_RECOGNITION_DIR  := face-recognition
FRONTEND_DIR          := frontend
BIN_DIR               := bin

.PHONY: help infra-up infra-down infra-logs build build-go test test-go test-py test-frontend lint lint-go lint-py lint-frontend clean models dev docker-build

help: ## List all available targets
	@echo "CampusVision AI - Development Makefile"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

infra-up: ## Start infrastructure services (Kafka, Redis, MariaDB)
	docker compose up -d kafka redis mariadb

infra-down: ## Stop infrastructure services
	docker compose down

infra-logs: ## Tail infrastructure logs
	docker compose logs -f

build: build-go ## Build all services (Go + frontend)
	cd $(FRONTEND_DIR) && pnpm build:prod

build-go: ## Build both Go services
	@mkdir -p $(STREAM_GATEWAY_DIR)/$(BIN_DIR) $(DORMITORY_SERVICE_DIR)/$(BIN_DIR)
	cd $(STREAM_GATEWAY_DIR) && go build -o $(BIN_DIR)/stream-gateway ./cmd/main.go
	cd $(DORMITORY_SERVICE_DIR) && go build -o $(BIN_DIR)/dormitory-service ./cmd/dormitory-service/

test: test-go test-py test-frontend ## Run all tests

test-go: ## Run Go tests (stream-gateway + dormitory-service-go)
	cd $(STREAM_GATEWAY_DIR) && go test ./... && echo "✅ stream-gateway tests passed"
	cd $(DORMITORY_SERVICE_DIR) && go test ./... && echo "✅ dormitory-service-go tests passed"

test-py: ## Run Python tests (face-recognition)
	cd $(FACE_RECOGNITION_DIR) && python -m pytest tests/ && echo "✅ face-recognition tests passed"

test-frontend: ## Run frontend tests
	cd $(FRONTEND_DIR) && pnpm test && echo "✅ frontend tests passed"

lint: lint-go lint-py lint-frontend ## Run all linters

lint-go: ## Run Go linters (golangci-lint)
	cd $(STREAM_GATEWAY_DIR) && golangci-lint run ./...
	cd $(DORMITORY_SERVICE_DIR) && golangci-lint run ./...

lint-py: ## Run Python linter (ruff)
	cd $(FACE_RECOGNITION_DIR) && ruff check .

lint-frontend: ## Run frontend linter (ESLint)
	cd $(FRONTEND_DIR) && npx eslint src/ --ext .ts,.vue

clean: ## Clean build artifacts
	rm -rf $(STREAM_GATEWAY_DIR)/$(BIN_DIR) $(DORMITORY_SERVICE_DIR)/$(BIN_DIR)
	rm -rf $(FRONTEND_DIR)/dist
	rm -rf $(STREAM_GATEWAY_DIR)/dist $(DORMITORY_SERVICE_DIR)/dist
	@echo "✅ Cleaned build artifacts"

models: ## Download ONNX models for face-recognition
	cd $(FACE_RECOGNITION_DIR) && python -m app.download_models

dev: ## Print instructions for starting all services in dev mode
	@echo "╔══════════════════════════════════════════════════════╗"
	@echo "║        CampusVision AI - Development Mode           ║"
	@echo "╚══════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Start each service in a separate terminal:"
	@echo ""
	@echo "  1. Infrastructure:"
	@echo "     $$ make infra-up"
	@echo ""
	@echo "  2. Stream Gateway (:8080 health, :8081 mgmt):"
	@echo "     $$ cd $(STREAM_GATEWAY_DIR) && go run cmd/main.go --config config.yaml"
	@echo ""
	@echo "  3. Face Recognition (Kafka consumer):"
	@echo "     $$ cd $(FACE_RECOGNITION_DIR) && python -m app.main --config config.yaml"
	@echo ""
	@echo "  4. Dormitory Service (:8083 API):"
	@echo "     $$ cd $(DORMITORY_SERVICE_DIR) && CONFIG_PATH=config.yaml go run ./cmd/dormitory-service/"
	@echo ""
	@echo "  5. Frontend (:80 dev server):"
	@echo "     $$ cd $(FRONTEND_DIR) && pnpm dev"
	@echo ""
	@echo "  Ports:  8080 (health)  8081 (mgmt)  8083 (API)  80 (frontend)"

docker-build: ## Build all Docker images
	docker compose build
