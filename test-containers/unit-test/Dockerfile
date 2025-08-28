# Dedicated Dockerfile for unit-test-runner
FROM golang:1.23-alpine AS unit-test
RUN apk add --no-cache bash

WORKDIR /app


# Copy the entire project directory for full test coverage
COPY . .

# Download dependencies for all services
RUN for d in services/*; do if [ -d "$d" ]; then cd "$d" && go mod download; fi; done

# Run the test orchestrator script for unit tests
CMD ["./scripts/test-orchestrator.sh", "unit"]
