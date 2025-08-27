# Dedicated Dockerfile for unit-test-runner
FROM golang:1.23-alpine AS unit-test
RUN apk add --no-cache bash

WORKDIR /app

# Copy root go mod files and shared module
COPY go.mod go.sum ./
COPY shared/ ./shared/

# Copy all service source and test files
COPY services/ ./services/
# Copy the testutils directory for utility tests
COPY tests/testutils/ ./tests/testutils/


# Copy only the test orchestrator script needed for unit tests
COPY scripts/test-orchestrator.sh ./scripts/test-orchestrator.sh

# Download dependencies for all services
RUN for d in services/*; do if [ -d "$d" ]; then cd "$d" && go mod download; fi; done

# Run the test orchestrator script for unit tests
CMD ["/bin/sh", "./scripts/test-orchestrator.sh", "unit"]
