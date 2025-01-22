# Build configuration
BINARY_NAME=quantai
VERSION=1.0.0
GOARCH=amd64
BUILD_DIR=./build

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOLINT=golangci-lint
GOMOD=$(GOCMD) mod
GOTOOL=$(GOCMD) tool
GOFMT=gofmt

# Database configuration
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=quantai
DB_HOST=localhost
DB_PORT=5432
MIGRATION_DIR=./migrations

# Docker configuration
DOCKER_COMPOSE=docker-compose
DOCKER_IMAGE=quantai
DOCKER_TAG=latest

# Default command
.DEFAULT_GOAL := help

# HELP
# This will output the help for each task
help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# Development
.PHONY: run
run: ## Run the application locally
	$(GORUN) cmd/server/main.go

.PHONY: build
build: ## Build the application
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v ./cmd/server

.PHONY: clean
clean: ## Clean build directory
	rm -rf $(BUILD_DIR)

.PHONY: test
test: ## Run tests
	$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOTOOL) cover -html=coverage.out

.PHONY: bench
bench: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

# Code quality
.PHONY: lint
lint: ## Run linter
	$(GOLINT) run ./...

.PHONY: vet
vet: ## Run go vet
	$(GOVET) ./...

.PHONY: fmt
fmt: ## Format code
	$(GOFMT) -s -w .

.PHONY: check
check: fmt lint vet test ## Run all checks

# Dependencies
.PHONY: deps
deps: ## Download dependencies
	$(GOMOD) download

.PHONY: deps-upgrade
deps-upgrade: ## Upgrade dependencies
	$(GOMOD) tidy
	$(GOMOD) download

# Database
.PHONY: migrate-up
migrate-up: ## Run database migrations
	migrate -path $(MIGRATION_DIR) -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable" up

.PHONY: migrate-down
migrate-down: ## Rollback database migrations
	migrate -path $(MIGRATION_DIR) -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable" down

.PHONY: migrate-create
migrate-create: ## Create a new migration file
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir $(MIGRATION_DIR) -seq $$name

# Docker
.PHONY: docker-build
docker-build: ## Build docker image
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: docker-push
docker-push: ## Push docker image
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-up
docker-up: ## Start docker containers
	$(DOCKER_COMPOSE) up -d

.PHONY: docker-down
docker-down: ## Stop docker containers
	$(DOCKER_COMPOSE) down

# Protobuf
.PHONY: proto
proto: ## Generate protobuf code
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/*.proto

# Documentation
.PHONY: swagger
swagger: ## Generate swagger documentation
	swag init -g cmd/server/main.go -o api/swagger

# Development environment
.PHONY: dev-deps
dev-deps: ## Install development dependencies
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/golang/mock/mockgen@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/swaggo/swag/cmd/swag@latest

.PHONY: generate-mocks
generate-mocks: ## Generate mocks for testing
	mockgen -source=internal/services/portfolio/service.go -destination=internal/services/portfolio/mock/service_mock.go
	mockgen -source=internal/services/analytics/service.go -destination=internal/services/analytics/mock/service_mock.go
	mockgen -source=internal/services/ai/service.go -destination=internal/services/ai/mock/service_mock.go

# AI Model Management
.PHONY: train-models
train-models: ## Train AI models
	$(GORUN) cmd/trainer/main.go

.PHONY: evaluate-models
evaluate-models: ## Evaluate AI models
	$(GORUN) cmd/evaluator/main.go

.PHONY: export-models
export-models: ## Export trained models
	$(GORUN) cmd/exporter/main.go

# Deployment
.PHONY: deploy-prod
deploy-prod: check build docker-build docker-push ## Deploy to production
	kubectl apply -f deployments/production

.PHONY: deploy-staging
deploy-staging: check build docker-build docker-push ## Deploy to staging
	kubectl apply -f deployments/staging

# Monitoring
.PHONY: prometheus-up
prometheus-up: ## Start Prometheus monitoring
	docker-compose -f monitoring/docker-compose.yml up -d prometheus

.PHONY: grafana-up
grafana-up: ## Start Grafana dashboard
	docker-compose -f monitoring/docker-compose.yml up -d grafana

.PHONY: monitoring-up
monitoring-up: prometheus-up grafana-up ## Start all monitoring services

.PHONY: monitoring-down
monitoring-down: ## Stop all monitoring services
	docker-compose -f monitoring/docker-compose.yml down

# Performance Testing
.PHONY: bench-cpu
bench-cpu: ## Run CPU profiling
	$(GORUN) cmd/server/main.go -cpuprofile=cpu.prof

.PHONY: bench-mem
bench-mem: ## Run memory profiling
	$(GORUN) cmd/server/main.go -memprofile=mem.prof

.PHONY: bench-trace
bench-trace: ## Run execution tracing
	$(GORUN) cmd/server/main.go -trace=trace.out

.PHONY: analyze-profile
analyze-profile: ## Analyze performance profile
	go tool pprof -http=:8080 cpu.prof

# Version Management
.PHONY: version
version: ## Print version information
	@echo "Version: $(VERSION)"
	@git describe --tags --abbrev=0 2>/dev/null || echo "No tags found"

.PHONY: release
release: ## Create a new release
	@read -p "Enter release version: " version; \
	git tag -a $$version -m "Release $$version" && \
	git push origin $$version

# CI Tasks
.PHONY: ci-test
ci-test: lint test ## Run CI tests
	$(GOTEST) -race -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: ci-build
ci-build: ## Run CI build
	$(MAKE) build
	$(MAKE) docker-build

# Maintenance
.PHONY: clean-all
clean-all: clean docker-down monitoring-down ## Clean everything
	rm -f cpu.prof mem.prof trace.out
	rm -f coverage.out coverage.txt
	rm -rf vendor/

.PHONY: update-deps
update-deps: ## Update all dependencies
	$(GOMOD) tidy
	$(GOMOD) verify
	$(GOMOD) download
	@echo "Dependencies updated successfully"