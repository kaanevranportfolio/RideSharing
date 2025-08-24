FROM golang:1.21-alpine AS builder
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
COPY --from=builder /app/services/trip-service/trip-service .
EXPOSE 8085
CMD ["./trip-service"]
