FROM golang:1.23-alpine AS builder
WORKDIR /app

# Copy root go mod files and shared module
COPY go.mod go.sum ./
COPY shared/ ./shared/

# Copy service source code
COPY services/api-gateway/ ./services/api-gateway/

# Build the application
WORKDIR /app/services/api-gateway
RUN go build -o api-gateway main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/services/api-gateway/api-gateway .
EXPOSE 8080
CMD ["./api-gateway"]
