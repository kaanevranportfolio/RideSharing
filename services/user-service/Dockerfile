# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy root go mod files and shared module
COPY go.mod go.sum ./
COPY shared/ ./shared/

# Copy service go mod files  
COPY services/user-service/go.mod services/user-service/go.sum ./services/user-service/

# Download dependencies
WORKDIR /app/services/user-service
RUN go mod download

# Copy service source code
WORKDIR /app
COPY services/user-service/ ./services/user-service/

# Build the application
WORKDIR /app/services/user-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o user-service main.go

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
COPY --from=builder /app/services/user-service/user-service .
EXPOSE 8051 50051
CMD ["./user-service"]
