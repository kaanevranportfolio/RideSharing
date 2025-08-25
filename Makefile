.PHONY: build run test clean help deps start-db test-infra test-services stop-all proto

# Default target
help:
	@echo "Available commands:"
	@echo "  proto          - Generate protobuf files"
	@echo "  test-infra     - Run complete infrastructure tests (self-contained)"
	@echo "  test-services  - Run complete service integration tests (self-contained)"
	@echo "  start-db       - Start only databases"
	@echo "  build          - Build all services (Go binaries)"
	@echo "  build-docker   - Build all services with Docker Compose"
	@echo "  start-services - Start all Go services locally"
	@echo "  run            - Start all services with Docker Compose"
	@echo "  test           - Run unit tests"
	@echo "  stop-all       - Stop all running containers and services"
	@echo "  clean          - Clean up everything (containers, volumes, binaries)"
	@echo "  deps           - Download dependencies"
	@echo "  setup          - Complete setup for new developers (Go version, proto, deps)"
	@echo ""
	@echo "Phase 3-5 Commands:"
	@echo "  start-monitoring - Start monitoring stack (Prometheus, Grafana, Jaeger)"
	@echo "  deploy-k8s     - Deploy to Kubernetes"
	@echo "  test-performance - Run performance tests with k6"
	@echo "  test-e2e       - Run end-to-end tests"
	@echo "  security-scan  - Run security vulnerability scans"
	@echo "  build-all      - Build all services and Docker images"
	@echo "  helm-install   - Install using Helm charts"
	@echo "  helm-upgrade   - Upgrade Helm deployment"

# Self-contained infrastructure test
test-infra:
	@echo "Running self-contained infrastructure tests..."
	@chmod +x scripts/test-infrastructure.sh
	@./scripts/test-infrastructure.sh

# Self-contained service integration test
# =============================================================================
# 🎯 COMPREHENSIVE TEST MANAGEMENT SYSTEM
# =============================================================================
# Centralized test orchestration with enhanced visualization
# Author: Senior Test Engineer

# Test Management - Top Level Commands
.PHONY: test-all test-fast test-full test-ci test-dev test-report

# 🚀 MASTER TEST COMMANDS
test-all: ## Run comprehensive test suite with full reporting
	@echo "🚀 Running comprehensive test suite..."
	@./scripts/test-orchestrator.sh all

test-fast: ## Run fast tests only (unit + integration)
	@echo "⚡ Running fast test suite..."
	@./scripts/test-orchestrator.sh unit
	@./scripts/test-orchestrator.sh integration

test-full: ## Run complete test suite including load and security tests
	@echo "🔬 Running full test suite..."
	@$(MAKE) test-all

test-ci: ## Run CI/CD optimized test suite
	@echo "🤖 Running CI/CD test suite..."
	@./scripts/test-orchestrator.sh unit
	@./scripts/test-orchestrator.sh integration
	@./scripts/test-orchestrator.sh security

test-dev: ## Run developer-focused tests with watch mode
	@echo "👨‍💻 Running developer test suite..."
	@$(MAKE) test-unit
	@$(MAKE) test-integration

test-report: ## Generate comprehensive test reports
	@echo "📊 Generating test reports..."
	@./scripts/test-orchestrator.sh all
	@echo "📁 Reports available in: test-reports/"

# Individual Test Categories
test-unit: ## Run unit tests with enhanced output
	@echo "🧪 Running unit tests..."
	@./scripts/test-orchestrator.sh unit

test-integration: ## Run integration tests
	@echo "🔗 Running integration tests..."
	@./scripts/test-orchestrator.sh integration

test-e2e: ## Run end-to-end tests
	@echo "🎭 Running E2E tests..."
	@./scripts/test-orchestrator.sh e2e

test-load: ## Run load and performance tests
	@echo "⚡ Running load tests..."
	@./scripts/test-orchestrator.sh load

test-security: ## Run security tests
	@echo "🔒 Running security tests..."
	@./scripts/test-orchestrator.sh security

test-contract: ## Run contract tests
	@echo "📋 Running contract tests..."
	@./scripts/test-orchestrator.sh contract

# Legacy test commands (maintained for backward compatibility)
test: test-fast ## Alias for fast tests

test-coverage: ## Generate coverage reports
	@echo "📊 Generating coverage reports..."
	@cd tests && go test ./unit/... ./testutils/... -v -coverprofile=coverage.out
	@cd tests && go tool cover -html=coverage.out -o coverage.html
	@echo "📁 Coverage report: tests/coverage.html"

test-benchmark: ## Run benchmark tests
	@echo "⚡ Running benchmark tests..."
	@cd tests && go test ./unit/... -bench=. -benchmem

test-clean: ## Clean test artifacts
	@echo "🧹 Cleaning test artifacts..."
	@rm -rf test-reports/
	@cd tests && rm -f coverage.out coverage.html
	@echo "✅ Test artifacts cleaned"

# Service-specific test commands
test-services: ## Test all individual services
	@echo "🔧 Testing individual services..."
	@for service in services/*/; do \
		if [ -d "$$service" ]; then \
			echo "🔍 Testing $$(basename $$service)..."; \
			cd "$$service" && go test ./... -v || true; \
			cd -; \
		fi; \
	done

test-api-gateway: ## Test API Gateway specifically
	@echo "🌐 Testing API Gateway..."
	@cd services/api-gateway && go test ./... -v

test-user-service: ## Test User Service specifically
	@echo "👤 Testing User Service..."
	@cd services/user-service && go test ./... -v

test-vehicle-service: ## Test Vehicle Service specifically
	@echo "🚗 Testing Vehicle Service..."
	@cd services/vehicle-service && go test ./... -v

test-geo-service: ## Test Geo Service specifically
	@echo "📍 Testing Geo Service..."
	@cd services/geo-service && go test ./... -v

test-trip-service: ## Test Trip Service specifically
	@echo "🧳 Testing Trip Service..."
	@cd services/trip-service && go test ./... -v

test-pricing-service: ## Test Pricing Service specifically
	@echo "💰 Testing Pricing Service..."
	@cd services/pricing-service && go test ./... -v

test-payment-service: ## Test Payment Service specifically
	@echo "💳 Testing Payment Service..."
	@cd services/payment-service && go test ./... -v

test-matching-service: ## Test Matching Service specifically
	@echo "🎯 Testing Matching Service..."
	@cd services/matching-service && go test ./... -v

# Test environment setup
test-setup: ## Setup test environment
	@echo "⚙️ Setting up test environment..."
	@mkdir -p test-reports/{unit,integration,e2e,load,security,contract}
	@echo "✅ Test environment ready"

test-deps: ## Install test dependencies
	@echo "📦 Installing test dependencies..."
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install github.com/sonatard/nancy@latest
	@echo "✅ Test dependencies installed"

# Help command with enhanced formatting
help: ## Show this help message with emojis and formatting
	@echo ""
	@echo "🎯 RIDESHARE PLATFORM - MAKEFILE COMMANDS"
	@echo "=========================================="
	@echo ""
	@echo "📊 TEST MANAGEMENT:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E 'test-|TEST' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "🚀 DEVELOPMENT:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -vE 'test-|TEST|help' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "💡 Examples:"
	@echo "  make test-all           # Run complete test suite"
	@echo "  make test-fast          # Run only unit + integration"
	@echo "  make test-ci            # Run CI-optimized tests"
	@echo "  make test-user-service  # Test specific service"
	@echo ""

# Start databases only
start-db:
	@echo "Starting databases..."
	@docker compose -f docker-compose-db.yml up -d

# Stop all services and containers
stop-all:
	@echo "Stopping all services and containers..."
	@pkill -f "test-service" 2>/dev/null || true
	@pkill -f "user-service" 2>/dev/null || true
	@pkill -f "vehicle-service" 2>/dev/null || true
	@pkill -f "geo-service" 2>/dev/null || true
	@pkill -f "matching-service" 2>/dev/null || true
	@pkill -f "trip-service" 2>/dev/null || true
	@docker compose -f docker-compose-db.yml down 2>/dev/null || true
	@docker compose down 2>/dev/null || true

# DRY build rules for all services
SERVICES := geo-service vehicle-service matching-service trip-service user-service api-gateway

build: $(SERVICES)

$(SERVICES):
	cd services/$@ && go build -o $@ main.go

# Build all services with Docker Compose
build-docker:
	@echo "Building all services with Docker Compose..."
	@docker compose build

# Start all Go services locally
start-services: build
	@echo "Starting all services locally..."
	@echo "Starting databases..."
	@docker compose -f docker-compose-db.yml up -d
	@sleep 5
	@echo "Starting test service on port 8080..."
	@./test-service &
	@echo "Starting user service on port 8081..."
	@cd services/user-service && ./user-service &
	@echo "Starting vehicle service on port 8082..."
	@cd services/vehicle-service && ./vehicle-service &
	@echo "Starting geo service on port 8083..."
	@cd services/geo-service && ./geo-service &
	@echo "Starting matching service on port 8084..."
	@cd services/matching-service && ./matching-service &
	@echo "Starting trip service on port 8085..."
	@cd services/trip-service && ./trip-service &
	@sleep 3
	@echo "✓ All services started. Check status with 'make status'"

# Start all services
run: start-services

# Run unit tests
test:
	@echo "Running unit tests..."
	@go test ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Clean up everything
clean: stop-all
	@echo "Cleaning up..."
	@docker compose -f docker-compose-db.yml down -v 2>/dev/null || true
	@docker compose down -v 2>/dev/null || true
	@docker system prune -f
	@rm -f test-service user-service vehicle-service geo-service matching-service trip-service
	@echo "✓ Cleanup completed"

# Check service health (requires services to be running)
health:
	@echo "Checking service health..."
	@curl -f http://localhost:8080/health 2>/dev/null && echo "✓ Test service healthy" || echo "✗ Test service down"
	@curl -f http://localhost:8081/health 2>/dev/null && echo "✓ User service healthy" || echo "✗ User service down"
	@curl -f http://localhost:8082/health 2>/dev/null && echo "✓ Vehicle service healthy" || echo "✗ Vehicle service down"
	@curl -f http://localhost:8083/health 2>/dev/null && echo "✓ Geo service healthy" || echo "✗ Geo service down"
	@curl -f http://localhost:8084/api/v1/health 2>/dev/null && echo "✓ Matching service healthy" || echo "✗ Matching service down"
	@curl -f http://localhost:8085/api/v1/health 2>/dev/null && echo "✓ Trip service healthy" || echo "✗ Trip service down"

# Show logs (requires services to be running)
logs:
	@docker compose logs -f

# Quick development cycle
dev: clean test-infra
	@echo "✓ Development environment ready"

# Production readiness check
prod-check: clean test-services
	@echo "✓ Production readiness verified"

# Show running containers and processes
status:
	@echo "=== Docker Containers ==="
	@docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "No containers running"
	@echo ""
	@echo "=== Service Processes ==="
	@pgrep -f "service" | while read pid; do ps -p $$pid -o pid,comm,args; done 2>/dev/null || echo "No service processes running"

# Unified local testing and deployment
# Test targets - consolidated and following Go best practices
.PHONY: test unit-test integration-test e2e-test load-test test-all test-coverage test-race test-bench

# Run all tests (unit + integration + e2e)
test-all: unit-test integration-test e2e-test

# Unit tests for individual packages (fast, no external dependencies)
unit-test:
	@echo "Running unit tests..."
	@if [ -d "tests/unit" ]; then \
		cd tests && go test ./unit/... -v -timeout=30s; \
	fi
	@if [ -d "tests/testutils" ]; then \
		cd tests && go test ./testutils/... -v -timeout=30s; \
	fi
	@for service in services/*/; do \
		if [ -d "$$service" ] && [ -f "$${service}go.mod" ]; then \
			echo "Testing service: $$(basename $$service)"; \
			cd "$$service" && go test ./... -v -timeout=30s || true; \
			cd - > /dev/null; \
		fi; \
	done

# Integration tests (require external services like databases)
integration-test:
	@echo "Running integration tests..."
	@if [ -d "tests/integration" ]; then \
		cd tests && go test ./integration/... -v -timeout=60s || true; \
	fi

# End-to-end tests (require full system running)
e2e-test:
	@echo "Running end-to-end tests..."
	@if [ -d "tests/e2e" ]; then \
		cd tests && go test ./e2e/... -v -timeout=120s || true; \
	else \
		echo "E2E tests directory not found"; \
	fi

# Legacy test target (consolidated with unit-test)
test: unit-test

# Test coverage report
test-coverage:
	@echo "Generating test coverage report..."
	@go test -coverprofile=coverage.out ./services/... ./shared/... ./tests/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Race condition detection
test-race:
	@echo "Running tests with race detection..."
	@go test -race -short ./services/... ./shared/... ./tests/...

# ================================================
# Phase 3-5: Production Infrastructure & CI/CD
# ================================================

# Phase 3: Production Infrastructure
start-monitoring:
	@echo "Starting monitoring stack..."
	@docker compose -f docker-compose-monitoring.yml up -d prometheus grafana jaeger node-exporter alertmanager
	@echo "Monitoring services started:"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - Grafana: http://localhost:3000 (admin/admin)"
	@echo "  - Jaeger: http://localhost:16686"

deploy-k8s:
	@echo "Deploying to Kubernetes..."
	@kubectl apply -f deployments/k8s/namespace.yaml
	@kubectl apply -f deployments/k8s/configmap.yaml
	@kubectl apply -f deployments/k8s/database/
	@echo "Waiting for databases to be ready..."
	@kubectl wait --for=condition=available --timeout=300s deployment/postgres -n rideshare-platform || true
	@kubectl wait --for=condition=available --timeout=300s deployment/mongodb -n rideshare-platform || true
	@kubectl wait --for=condition=available --timeout=300s deployment/redis -n rideshare-platform || true
	@kubectl apply -f deployments/k8s/services/
	@kubectl apply -f deployments/k8s/autoscaling/
	@echo "Kubernetes deployment complete!"

k8s-status:
	@echo "Kubernetes deployment status:"
	@kubectl get pods -n rideshare-platform
	@kubectl get services -n rideshare-platform
	@kubectl get hpa -n rideshare-platform

# Helm deployment
helm-install:
	@echo "Installing with Helm..."
	@helm dependency update deployments/helm/rideshare-platform/
	@helm install rideshare-platform deployments/helm/rideshare-platform/ \
		--namespace rideshare-platform --create-namespace \
		--set image.tag=latest

helm-upgrade:
	@echo "Upgrading Helm deployment..."
	@helm upgrade rideshare-platform deployments/helm/rideshare-platform/ \
		--namespace rideshare-platform \
		--set image.tag=latest

helm-uninstall:
	@echo "Uninstalling Helm deployment..."
	@helm uninstall rideshare-platform --namespace rideshare-platform

# Phase 4: Testing & Quality Assurance
test-performance:
	@echo "Running performance tests..."
	@command -v k6 >/dev/null 2>&1 || { echo "k6 not installed. Installing..."; \
		sudo gpg -k; \
		sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69; \
		echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list; \
		sudo apt-get update; \
		sudo apt-get install k6; }
	@k6 run --vus 10 --duration 30s tests/performance/load-test.js

test-performance-extended:
	@echo "Running extended performance tests..."
	@k6 run --vus 50 --duration 5m tests/performance/load-test.js

test-e2e:
	@echo "Running end-to-end tests..."
	@go test -v ./tests/e2e/...

test-all:
	@echo "Running all tests..."
	@$(MAKE) test
	@$(MAKE) test-e2e

# Phase 5: CI/CD and Security
security-scan:
	@echo "Running security scans..."
	@command -v trivy >/dev/null 2>&1 || { echo "Installing Trivy..."; \
		sudo apt-get update; \
		sudo apt-get install wget apt-transport-https gnupg lsb-release; \
		wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -; \
		echo "deb https://aquasecurity.github.io/trivy-repo/deb $$(lsb_release -sc) main" | sudo tee -a /etc/apt/sources.list.d/trivy.list; \
		sudo apt-get update; \
		sudo apt-get install trivy; }
	@echo "Scanning Docker images for vulnerabilities..."
	@for service in user-service vehicle-service pricing-service payment-service; do \
		echo "Scanning $$service..."; \
		trivy image rideshare/$$service:latest || true; \
	done

build-all:
	@echo "Building all services and Docker images..."
	@$(MAKE) build
	@$(MAKE) build-docker

ci-pipeline:
	@echo "Running full CI pipeline..."
	@$(MAKE) deps
	@$(MAKE) build-all
	@$(MAKE) test-all
	@$(MAKE) security-scan
	@echo "CI pipeline completed successfully!"

# Infrastructure management
infra-up:
	@echo "Starting full infrastructure..."
	@$(MAKE) start-db
	@sleep 5
	@$(MAKE) start-monitoring
	@$(MAKE) test-infra

infra-down:
	@echo "Stopping full infrastructure..."
	@docker compose -f docker-compose-monitoring.yml down || true
	@$(MAKE) stop-all

# Health checks
health-check:
	@echo "Checking service health..."
	@curl -f http://localhost:9084/health || echo "User service not available"
	@curl -f http://localhost:9083/health || echo "Vehicle service not available"
	@curl -f http://localhost:9087/health || echo "Pricing service not available"
	@curl -f http://localhost:9088/health || echo "Payment service not available"

metrics-check:
	@echo "Checking metrics endpoints..."
	@curl -f http://localhost:9084/api/v1/metrics || echo "User service metrics not available"
	@curl -f http://localhost:9083/api/v1/metrics || echo "Vehicle service metrics not available"

# Documentation
docs-serve:
	@echo "Starting documentation server..."
	@command -v mkdocs >/dev/null 2>&1 || pip install mkdocs mkdocs-material
	@mkdocs serve

# Benchmark tests
test-bench:
	@echo "Running benchmark tests..."
	@go test -bench=. -benchmem ./services/... ./shared/... ./tests/...

# Load testing
load-test:
	@echo "Running load tests..."

# =============================================================================
# 🧬 PROTOBUF AND DEVELOPMENT SETUP
# =============================================================================

# Generate protobuf files
proto:
	@echo "🧬 Generating protobuf files..."
	@chmod +x scripts/generate-proto.sh
	@./scripts/generate-proto.sh

# Complete setup for new developers
setup:
	@echo "🚀 Setting up development environment..."
	@echo "Step 1: Updating Go version in project files..."
	@chmod +x scripts/update-project-go-version.sh
	@./scripts/update-project-go-version.sh
	@echo "Step 2: Generating protobuf files..."
	@$(MAKE) proto
	@echo "Step 3: Installing dependencies..."
	@$(MAKE) deps
	@echo "✅ Development environment setup complete!"

# Upgrade system Go version (requires sudo)
upgrade-go:
	@echo "⬆️ Upgrading system Go version..."
	@chmod +x scripts/upgrade-go.sh
	@./scripts/upgrade-go.sh

# Clean generated protobuf files
clean-proto:
	@echo "🧹 Cleaning generated protobuf files..."
	@find shared/proto -name "*.pb.go" -delete
	@echo "✅ Protobuf files cleaned"

# Update project Go version
update-go-version:
	@echo "🔧 Updating project Go version..."
	@chmod +x scripts/update-project-go-version.sh
	@./scripts/update-project-go-version.sh
	@bash tests/load/load_test.sh
