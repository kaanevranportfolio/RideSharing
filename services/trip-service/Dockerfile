FROM golang:1.23-alpine AS builder
WORKDIR /app

# Copy root go mod files and shared module
COPY go.mod go.sum ./
COPY shared/ ./shared/

# Copy service go mod files
COPY services/trip-service/go.mod services/trip-service/go.sum ./services/trip-service/

# Download dependencies
WORKDIR /app/services/trip-service
RUN go mod download

# Copy service source code
WORKDIR /app
COPY services/trip-service/ ./services/trip-service/

# Build the application
WORKDIR /app/services/trip-service
RUN go build -o trip-service main.go

FROM alpine:latest
WORKDIR /app
ARG TARGETARCH
# Install curl and download correct grpc-health-probe for Linux x86_64 or arm64
RUN apk add --no-cache curl \
	&& if [ "$TARGETARCH" = "amd64" ]; then \
		PROBE_URL="https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.40/grpc_health_probe-linux-amd64"; \
	elif [ "$TARGETARCH" = "arm64" ]; then \
		PROBE_URL="https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.40/grpc_health_probe-linux-arm64"; \
	else \
		echo "Unsupported architecture: $TARGETARCH"; exit 1; \
	fi; \
	wget -O /usr/local/bin/grpc-health-probe "$PROBE_URL"; \
	chmod +x /usr/local/bin/grpc-health-probe
COPY --from=builder /app/services/trip-service/trip-service .
EXPOSE 50053 8085
CMD ["./trip-service"]
