# Dedicated Dockerfile for integration-test-runner
FROM golang:1.23-alpine AS integration-test
RUN apk add --no-cache bash curl

WORKDIR /app

# Copy root go mod files and shared module
COPY go.mod go.sum ./
COPY shared/ ./shared/

# Copy all service source files
COPY services/ ./services/

# Copy integration tests and orchestrator script
COPY tests/integration/ ./tests/integration/
COPY scripts/test-orchestrator.sh ./scripts/test-orchestrator.sh

# Download dependencies for all services
RUN for d in services/*; do if [ -d "$d" ]; then cd "$d" && go mod download; fi; done

# Run the minimal test orchestrator script for integration tests
ENTRYPOINT ["./scripts/test-orchestrator.sh", "integration"]
