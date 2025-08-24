# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy root go mod files and shared module
COPY go.mod go.sum ./
COPY shared/ ./shared/

# Copy service go mod files
COPY services/geo-service/go.mod services/geo-service/go.sum ./services/geo-service/

# Download dependencies
WORKDIR /app/services/geo-service
RUN go mod download

# Copy service source code
WORKDIR /app
COPY services/geo-service/ ./services/geo-service/

# Build the application
WORKDIR /app/services/geo-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o geo-service main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/services/geo-service/geo-service .

# Expose port
EXPOSE 8053

# Command to run
CMD ["./geo-service"]
