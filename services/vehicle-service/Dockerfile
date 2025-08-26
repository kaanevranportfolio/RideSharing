# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy root go mod files and shared module
COPY go.mod go.sum ./
COPY shared/ ./shared/

# Copy service go mod files
COPY services/vehicle-service/go.mod services/vehicle-service/go.sum ./services/vehicle-service/

# Download dependencies and create go.sum
WORKDIR /app/services/vehicle-service
RUN go mod download && go mod tidy

# Copy service source code
WORKDIR /app
COPY services/vehicle-service/ ./services/vehicle-service/

# Download dependencies again after copying source
WORKDIR /app/services/vehicle-service
RUN go mod download

# Build the application
WORKDIR /app/services/vehicle-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o vehicle-service main.go

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata curl jq \
	&& ARCH=$(uname -m) \
	&& if [ "$ARCH" = "x86_64" ]; then \
		PROBE_URL=$(curl -s https://api.github.com/repos/grpc-ecosystem/grpc-health-probe/releases/latest | jq -r '.assets[] | select(.name | test("linux-amd64$")) | .browser_download_url'); \
	elif [ "$ARCH" = "aarch64" ]; then \
		PROBE_URL=$(curl -s https://api.github.com/repos/grpc-ecosystem/grpc-health-probe/releases/latest | jq -r '.assets[] | select(.name | test("linux-arm64$")) | .browser_download_url'); \
	else \
		echo "Unsupported architecture: $ARCH"; exit 1; \
	fi \
	&& wget -O /usr/local/bin/grpc-health-probe "$PROBE_URL" \
	&& chmod +x /usr/local/bin/grpc-health-probe
WORKDIR /root/
COPY --from=builder /app/services/vehicle-service/vehicle-service .
EXPOSE 8052
CMD ["./vehicle-service"]
