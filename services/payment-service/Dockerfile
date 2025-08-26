FROM golang:1.23-alpine AS builder
WORKDIR /app

# Copy root go mod files and shared module
COPY go.mod go.sum ./
COPY shared/ ./shared/

# Copy service go mod files
COPY services/payment-service/go.mod ./services/payment-service/

# Download dependencies
WORKDIR /app/services/payment-service
RUN go mod download

# Copy service source code
WORKDIR /app
COPY services/payment-service/ ./services/payment-service/

# Build the application
WORKDIR /app/services/payment-service
RUN go build -o payment-service main.go

FROM alpine:latest
WORKDIR /app
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
COPY --from=builder /app/services/payment-service/payment-service .
EXPOSE 8055
CMD ["./payment-service"]
