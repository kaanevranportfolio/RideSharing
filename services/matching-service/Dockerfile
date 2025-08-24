FROM golang:1.21-alpine AS builder
WORKDIR /app

# Copy root go mod files and shared module
COPY go.mod go.sum ./
COPY shared/ ./shared/

# Copy service go mod files
COPY services/matching-service/go.mod services/matching-service/go.sum ./services/matching-service/

# Download dependencies
WORKDIR /app/services/matching-service
RUN go mod download

# Copy service source code
WORKDIR /app
COPY services/matching-service/ ./services/matching-service/

# Build the application
WORKDIR /app/services/matching-service
RUN go build -o matching-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/services/matching-service/matching-service .
EXPOSE 8084
CMD ["./matching-service"]
