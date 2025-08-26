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
# Install curl for healthcheck support
RUN apk add --no-cache curl
COPY --from=builder /app/services/payment-service/payment-service .
EXPOSE 9087
CMD ["./payment-service"]
