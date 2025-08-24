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

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/services/user-service/user-service .

# Expose port
EXPOSE 8051 50051

# Command to run the application
CMD ["./user-service"]
