# Deployment Guide

This document provides comprehensive deployment instructions for the rideshare platform across different environments.

## Prerequisites

### Required Tools
- Docker 20.10+
- Docker Compose 2.0+
- Kubernetes 1.24+
- Helm 3.8+
- kubectl configured for your cluster
- Go 1.21+
- Node.js 18+ (for development tools)

### Infrastructure Requirements

#### Minimum Resources (Development)
- CPU: 4 cores
- Memory: 8GB RAM
- Storage: 50GB SSD
- Network: 100Mbps

#### Production Resources (Recommended)
- CPU: 16+ cores
- Memory: 32GB+ RAM
- Storage: 500GB+ SSD with high IOPS
- Network: 1Gbps+
- Load Balancer with SSL termination

## Local Development Setup

### 1. Clone and Setup

```bash
# Clone the repository
git clone <repository-url>
cd rideshare-platform

# Copy environment template
cp .env.example .env

# Edit environment variables
vim .env
```

### 2. Environment Configuration

```bash
# .env file
# Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=rideshare_platform
POSTGRES_USER=rideshare
POSTGRES_PASSWORD=your_secure_password

MONGO_HOST=localhost
MONGO_PORT=27017
MONGO_DB=rideshare_geo
MONGO_USER=rideshare
MONGO_PASSWORD=your_secure_password

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_secure_password

# Service Configuration
USER_SERVICE_PORT=8001
VEHICLE_SERVICE_PORT=8002
GEO_SERVICE_PORT=8003
MATCHING_SERVICE_PORT=8004
PRICING_SERVICE_PORT=8005
TRIP_SERVICE_PORT=8006
PAYMENT_SERVICE_PORT=8007
API_GATEWAY_PORT=8080

# JWT Configuration
JWT_SECRET=your_jwt_secret_key_here
JWT_EXPIRY=24h
JWT_REFRESH_EXPIRY=168h

# External Services
STRIPE_SECRET_KEY=sk_test_your_stripe_key
GOOGLE_MAPS_API_KEY=your_google_maps_key

# Monitoring
PROMETHEUS_PORT=9090
GRAFANA_PORT=3000
JAEGER_PORT=16686
```

### 3. Start Development Environment

```bash
# Start infrastructure services
make dev-infra-up

# Wait for services to be ready
make wait-for-services

# Run database migrations
make migrate-up

# Seed test data
make seed-data

# Start all microservices
make dev-services-up

# Start API Gateway
make dev-gateway-up
```

### 4. Verify Installation

```bash
# Check service health
make health-check

# Access GraphQL Playground
open http://localhost:8080/playground

# Access monitoring dashboards
open http://localhost:3000  # Grafana
open http://localhost:9090  # Prometheus
open http://localhost:16686 # Jaeger
```

## Docker Compose Deployment

### Development Environment

```yaml
# docker-compose.dev.yml
version: '3.8'

services:
  # Infrastructure Services
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: rideshare_platform
      POSTGRES_USER: rideshare
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./infrastructure/database/init:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U rideshare"]
      interval: 10s
      timeout: 5s
      retries: 5

  mongodb:
    image: mongo:6.0
    environment:
      MONGO_INITDB_ROOT_USERNAME: rideshare
      MONGO_INITDB_ROOT_PASSWORD: password
      MONGO_INITDB_DATABASE: rideshare_geo
    ports:
      - "27017:27017"
    volumes:
      - mongodb_data:/data/db
      - ./infrastructure/database/mongo-init:/docker-entrypoint-initdb.d
    healthcheck:
      test: echo 'db.runCommand("ping").ok' | mongosh localhost:27017/test --quiet
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass password
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "--raw", "incr", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Message Queue
  kafka:
    image: confluentinc/cp-kafka:latest
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    ports:
      - "9092:9092"
    depends_on:
      - zookeeper

  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    ports:
      - "2181:2181"

  # Monitoring Services
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./infrastructure/monitoring/prometheus:/etc/prometheus
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
    volumes:
      - grafana_data:/var/lib/grafana
      - ./infrastructure/monitoring/grafana:/etc/grafana/provisioning

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
      - "14268:14268"
    environment:
      COLLECTOR_OTLP_ENABLED: true

  # Application Services
  user-service:
    build:
      context: .
      dockerfile: services/user-service/Dockerfile
    ports:
      - "8001:8001"
    environment:
      - DATABASE_URL=postgres://rideshare:password@postgres:5432/rideshare_platform
      - REDIS_URL=redis://:password@redis:6379
      - KAFKA_BROKERS=kafka:9092
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      kafka:
        condition: service_started

  vehicle-service:
    build:
      context: .
      dockerfile: services/vehicle-service/Dockerfile
    ports:
      - "8002:8002"
    environment:
      - DATABASE_URL=postgres://rideshare:password@postgres:5432/rideshare_platform
      - REDIS_URL=redis://:password@redis:6379
      - KAFKA_BROKERS=kafka:9092
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  geo-service:
    build:
      context: .
      dockerfile: services/geo-service/Dockerfile
    ports:
      - "8003:8003"
    environment:
      - MONGODB_URL=mongodb://rideshare:password@mongodb:27017/rideshare_geo
      - REDIS_URL=redis://:password@redis:6379
      - KAFKA_BROKERS=kafka:9092
    depends_on:
      mongodb:
        condition: service_healthy
      redis:
        condition: service_healthy

  matching-service:
    build:
      context: .
      dockerfile: services/matching-service/Dockerfile
    ports:
      - "8004:8004"
    environment:
      - REDIS_URL=redis://:password@redis:6379
      - KAFKA_BROKERS=kafka:9092
      - GEO_SERVICE_URL=geo-service:8003
      - PRICING_SERVICE_URL=pricing-service:8005
    depends_on:
      - redis
      - geo-service
      - pricing-service

  pricing-service:
    build:
      context: .
      dockerfile: services/pricing-service/Dockerfile
    ports:
      - "8005:8005"
    environment:
      - DATABASE_URL=postgres://rideshare:password@postgres:5432/rideshare_platform
      - REDIS_URL=redis://:password@redis:6379
      - KAFKA_BROKERS=kafka:9092
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  trip-service:
    build:
      context: .
      dockerfile: services/trip-service/Dockerfile
    ports:
      - "8006:8006"
    environment:
      - DATABASE_URL=postgres://rideshare:password@postgres:5432/rideshare_platform
      - REDIS_URL=redis://:password@redis:6379
      - KAFKA_BROKERS=kafka:9092
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  payment-service:
    build:
      context: .
      dockerfile: services/payment-service/Dockerfile
    ports:
      - "8007:8007"
    environment:
      - DATABASE_URL=postgres://rideshare:password@postgres:5432/rideshare_platform
      - REDIS_URL=redis://:password@redis:6379
      - KAFKA_BROKERS=kafka:9092
      - STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  api-gateway:
    build:
      context: .
      dockerfile: services/api-gateway/Dockerfile
    ports:
      - "8080:8080"
    environment:
      - USER_SERVICE_URL=user-service:8001
      - VEHICLE_SERVICE_URL=vehicle-service:8002
      - GEO_SERVICE_URL=geo-service:8003
      - MATCHING_SERVICE_URL=matching-service:8004
      - PRICING_SERVICE_URL=pricing-service:8005
      - TRIP_SERVICE_URL=trip-service:8006
      - PAYMENT_SERVICE_URL=payment-service:8007
      - REDIS_URL=redis://:password@redis:6379
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - user-service
      - vehicle-service
      - geo-service
      - matching-service
      - pricing-service
      - trip-service
      - payment-service

volumes:
  postgres_data:
  mongodb_data:
  redis_data:
  prometheus_data:
  grafana_data:
```

### Production Environment

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  # Use external managed databases in production
  # Only include application services here
  
  user-service:
    image: rideshare/user-service:${VERSION}
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
      - KAFKA_BROKERS=${KAFKA_BROKERS}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8001/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  # ... other services with similar production configurations
```

## Kubernetes Deployment

### Namespace Setup

```yaml
# infrastructure/kubernetes/namespaces/rideshare.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: rideshare
  labels:
    name: rideshare
    environment: production
```

### ConfigMaps and Secrets

```yaml
# infrastructure/kubernetes/configmaps/app-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: rideshare
data:
  USER_SERVICE_PORT: "8001"
  VEHICLE_SERVICE_PORT: "8002"
  GEO_SERVICE_PORT: "8003"
  MATCHING_SERVICE_PORT: "8004"
  PRICING_SERVICE_PORT: "8005"
  TRIP_SERVICE_PORT: "8006"
  PAYMENT_SERVICE_PORT: "8007"
  API_GATEWAY_PORT: "8080"
  JWT_EXPIRY: "24h"
  JWT_REFRESH_EXPIRY: "168h"

---
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
  namespace: rideshare
type: Opaque
data:
  DATABASE_URL: <base64-encoded-database-url>
  REDIS_URL: <base64-encoded-redis-url>
  JWT_SECRET: <base64-encoded-jwt-secret>
  STRIPE_SECRET_KEY: <base64-encoded-stripe-key>
  GOOGLE_MAPS_API_KEY: <base64-encoded-maps-key>
```

### Service Deployments

```yaml
# infrastructure/kubernetes/deployments/user-service.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
  namespace: rideshare
  labels:
    app: user-service
    version: v1
spec:
  replicas: 3
  selector:
    matchLabels:
      app: user-service
  template:
    metadata:
      labels:
        app: user-service
        version: v1
    spec:
      containers:
      - name: user-service
        image: rideshare/user-service:latest
        ports:
        - containerPort: 8001
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: DATABASE_URL
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: REDIS_URL
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: JWT_SECRET
        envFrom:
        - configMapRef:
            name: app-config
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8001
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8001
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: user-service
  namespace: rideshare
  labels:
    app: user-service
spec:
  selector:
    app: user-service
  ports:
  - port: 8001
    targetPort: 8001
    protocol: TCP
  type: ClusterIP
```

### Ingress Configuration

```yaml
# infrastructure/kubernetes/ingress/api-gateway.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-gateway
  namespace: rideshare
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
spec:
  tls:
  - hosts:
    - api.rideshare.com
    secretName: api-gateway-tls
  rules:
  - host: api.rideshare.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: api-gateway
            port:
              number: 8080
```

## Helm Chart Deployment

### Chart Structure

```
deployments/helm/rideshare-platform/
├── Chart.yaml
├── values.yaml
├── values-dev.yaml
├── values-staging.yaml
├── values-prod.yaml
└── templates/
    ├── configmap.yaml
    ├── secret.yaml
    ├── deployment.yaml
    ├── service.yaml
    ├── ingress.yaml
    ├── hpa.yaml
    └── servicemonitor.yaml
```

### Chart.yaml

```yaml
apiVersion: v2
name: rideshare-platform
description: A comprehensive rideshare platform
type: application
version: 1.0.0
appVersion: "1.0.0"
dependencies:
- name: postgresql
  version: 12.1.9
  repository: https://charts.bitnami.com/bitnami
  condition: postgresql.enabled
- name: mongodb
  version: 13.6.8
  repository: https://charts.bitnami.com/bitnami
  condition: mongodb.enabled
- name: redis
  version: 17.4.7
  repository: https://charts.bitnami.com/bitnami
  condition: redis.enabled
- name: kafka
  version: 20.0.6
  repository: https://charts.bitnami.com/bitnami
  condition: kafka.enabled
```

### values.yaml

```yaml
# Global configuration
global:
  imageRegistry: ""
  imagePullSecrets: []
  storageClass: ""

# Application configuration
app:
  name: rideshare-platform
  version: "1.0.0"
  environment: production

# Service configurations
services:
  userService:
    enabled: true
    image:
      repository: rideshare/user-service
      tag: "1.0.0"
      pullPolicy: IfNotPresent
    replicaCount: 3
    resources:
      requests:
        memory: "256Mi"
        cpu: "250m"
      limits:
        memory: "512Mi"
        cpu: "500m"
    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 10
      targetCPUUtilizationPercentage: 70

  vehicleService:
    enabled: true
    image:
      repository: rideshare/vehicle-service
      tag: "1.0.0"
      pullPolicy: IfNotPresent
    replicaCount: 2
    resources:
      requests:
        memory: "256Mi"
        cpu: "250m"
      limits:
        memory: "512Mi"
        cpu: "500m"

  geoService:
    enabled: true
    image:
      repository: rideshare/geo-service
      tag: "1.0.0"
      pullPolicy: IfNotPresent
    replicaCount: 3
    resources:
      requests:
        memory: "512Mi"
        cpu: "500m"
      limits:
        memory: "1Gi"
        cpu: "1000m"

  matchingService:
    enabled: true
    image:
      repository: rideshare/matching-service
      tag: "1.0.0"
      pullPolicy: IfNotPresent
    replicaCount: 3
    resources:
      requests:
        memory: "512Mi"
        cpu: "500m"
      limits:
        memory: "1Gi"
        cpu: "1000m"

  pricingService:
    enabled: true
    image:
      repository: rideshare/pricing-service
      tag: "1.0.0"
      pullPolicy: IfNotPresent
    replicaCount: 2
    resources:
      requests:
        memory: "256Mi"
        cpu: "250m"
      limits:
        memory: "512Mi"
        cpu: "500m"

  tripService:
    enabled: true
    image:
      repository: rideshare/trip-service
      tag: "1.0.0"
      pullPolicy: IfNotPresent
    replicaCount: 3
    resources:
      requests:
        memory: "512Mi"
        cpu: "500m"
      limits:
        memory: "1Gi"
        cpu: "1000m"

  paymentService:
    enabled: true
    image:
      repository: rideshare/payment-service
      tag: "1.0.0"
      pullPolicy: IfNotPresent
    replicaCount: 2
    resources:
      requests:
        memory: "256Mi"
        cpu: "250m"
      limits:
        memory: "512Mi"
        cpu: "500m"

  apiGateway:
    enabled: true
    image:
      repository: rideshare/api-gateway
      tag: "1.0.0"
      pullPolicy: IfNotPresent
    replicaCount: 3
    resources:
      requests:
        memory: "512Mi"
        cpu: "500m"
      limits:
        memory: "1Gi"
        cpu: "1000m"
    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 20
      targetCPUUtilizationPercentage: 70

# Ingress configuration
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "1000"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
  hosts:
  - host: api.rideshare.com
    paths:
    - path: /
      pathType: Prefix
  tls:
  - secretName: api-gateway-tls
    hosts:
    - api.rideshare.com

# Database configurations
postgresql:
  enabled: true
  auth:
    postgresPassword: "secure_password"
    username: "rideshare"
    password: "secure_password"
    database: "rideshare_platform"
  primary:
    persistence:
      enabled: true
      size: 100Gi
    resources:
      requests:
        memory: "1Gi"
        cpu: "500m"
      limits:
        memory: "2Gi"
        cpu: "1000m"

mongodb:
  enabled: true
  auth:
    enabled: true
    rootPassword: "secure_password"
    username: "rideshare"
    password: "secure_password"
    database: "rideshare_geo"
  persistence:
    enabled: true
    size: 50Gi
  resources:
    requests:
      memory: "1Gi"
      cpu: "500m"
    limits:
      memory: "2Gi"
      cpu: "1000m"

redis:
  enabled: true
  auth:
    enabled: true
    password: "secure_password"
  master:
    persistence:
      enabled: true
      size: 20Gi
    resources:
      requests:
        memory: "512Mi"
        cpu: "250m"
      limits:
        memory: "1Gi"
        cpu: "500m"

kafka:
  enabled: true
  persistence:
    enabled: true
    size: 50Gi
  zookeeper:
    persistence:
      enabled: true
      size: 10Gi

# Monitoring
monitoring:
  enabled: true
  prometheus:
    enabled: true
  grafana:
    enabled: true
  jaeger:
    enabled: true
```

### Deployment Commands

```bash
# Add Helm repositories
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Install for development
helm install rideshare ./deployments/helm/rideshare-platform \
  -f ./deployments/helm/rideshare-platform/values-dev.yaml \
  --namespace rideshare \
  --create-namespace

# Install for production
helm install rideshare ./deployments/helm/rideshare-platform \
  -f ./deployments/helm/rideshare-platform/values-prod.yaml \
  --namespace rideshare \
  --create-namespace

# Upgrade deployment
helm upgrade rideshare ./deployments/helm/rideshare-platform \
  -f ./deployments/helm/rideshare-platform/values-prod.yaml \
  --namespace rideshare

# Rollback deployment
helm rollback rideshare 1 --namespace rideshare
```

## Production Deployment Checklist

### Pre-deployment
- [ ] Environment variables configured
- [ ] Secrets properly encrypted and stored
- [ ] Database migrations tested
- [ ] Load balancer configured
- [ ] SSL certificates installed
- [ ] Monitoring and alerting setup
- [ ] Backup strategy implemented
- [ ] Disaster recovery plan documented

### Deployment
- [ ] Blue-green deployment strategy
- [ ] Health checks passing
- [ ] Database connections verified
- [ ] External service integrations tested
- [ ] Performance benchmarks met
- [ ] Security scans completed

### Post-deployment
- [ ] Smoke tests passed
- [ ] Monitoring dashboards active
- [ ] Log aggregation working
- [ ] Alerts configured
- [ ] Documentation updated
- [ ] Team notified

## Monitoring and Observability

### Health Checks

```bash
# Service health endpoints
curl http://localhost:8001/health  # User Service
curl http://localhost:8002/health  # Vehicle Service
curl http://localhost:8003/health  # Geo Service
curl http://localhost:8004/health  # Matching Service
curl http://localhost:8005/health  # Pricing Service
curl http://localhost:8006/health  # Trip Service
curl http://localhost:8007/health  # Payment Service
curl http://localhost:8080/health  # API Gateway

# Database health
pg_isready -h localhost -p 5432 -U rideshare
mongosh --eval "db.adminCommand('ping')"
redis-cli ping
```

### Metrics and Alerts

Key metrics to monitor:
- Request latency (p95, p99)
- Error rates
- Database connection pool usage
- Memory and CPU utilization
- Active trips count
- Driver availability
- Payment success rate

### Log Aggregation

```bash
# View service logs
kubectl logs -f deployment/user-service -n rideshare
kubectl logs -f deployment/api-gateway -n rideshare

# View aggregated logs
# Configure ELK stack or similar for production
```

This comprehensive deployment guide provides everything needed to deploy the rideshare platform from local development to production Kubernetes clusters with proper monitoring, security, and scalability considerations.