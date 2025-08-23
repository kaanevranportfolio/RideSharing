# Rideshare Platform

A comprehensive, production-grade rideshare simulation platform built with Go, GraphQL, and gRPC microservices architecture.

## Architecture Overview

This platform implements a sophisticated rideshare system with the following core services:

- **User Management Service**: Authentication, driver/rider profiles
- **Vehicle Management Service**: Vehicle registration and availability
- **Geospatial/ETA Service**: Location tracking and route optimization
- **Matching/Dispatch Service**: Intelligent driver-rider matching
- **Pricing Service**: Dynamic fare calculation with surge pricing
- **Trip Lifecycle Service**: Complete trip state management with event sourcing
- **Payment Mock Service**: Simulated payment processing
- **GraphQL API Gateway**: Unified client-facing API

## Technology Stack

- **Backend**: Go 1.21+, Gin, gRPC
- **API Layer**: GraphQL (gqlgen)
- **Databases**: PostgreSQL, MongoDB, Redis
- **Infrastructure**: Docker, Kubernetes, Helm
- **Monitoring**: Prometheus, Grafana, Jaeger
- **Event Streaming**: Apache Kafka

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Kubernetes cluster (local or cloud)
- Helm 3.x
- Go 1.21+

### Local Development

1. Clone the repository:
```bash
git clone <repository-url>
cd rideshare-platform
```

2. Start the development environment:
```bash
make dev-up
```

3. Run database migrations:
```bash
make migrate-up
```

4. Start all services:
```bash
make services-up
```

5. Access the GraphQL playground:
```
http://localhost:8080/playground
```

### Production Deployment

1. Deploy to Kubernetes:
```bash
helm install rideshare ./deployments/helm/rideshare-platform
```

2. Monitor the deployment:
```bash
kubectl get pods -n rideshare
```

## Project Structure

```
rideshare-platform/
├── services/                 # Microservices
│   ├── user-service/        # User management
│   ├── vehicle-service/     # Vehicle management
│   ├── geo-service/         # Geospatial operations
│   ├── matching-service/    # Driver-rider matching
│   ├── pricing-service/     # Fare calculation
│   ├── trip-service/        # Trip lifecycle
│   ├── payment-service/     # Payment processing
│   └── api-gateway/         # GraphQL gateway
├── shared/                  # Shared libraries
│   ├── proto/              # Protocol buffer definitions
│   ├── models/             # Domain models
│   ├── utils/              # Utility functions
│   └── middleware/         # Common middleware
├── infrastructure/          # Infrastructure as code
│   ├── docker/             # Docker configurations
│   ├── kubernetes/         # K8s manifests
│   ├── helm/               # Helm charts
│   └── monitoring/         # Monitoring configs
├── tests/                  # Test suites
│   ├── unit/               # Unit tests
│   ├── integration/        # Integration tests
│   └── e2e/                # End-to-end tests
├── docs/                   # Documentation
├── scripts/                # Build and deployment scripts
└── deployments/            # Deployment configurations
```

## API Documentation

### GraphQL Schema

The platform exposes a comprehensive GraphQL API with the following main types:

- `User`: Rider and driver profiles
- `Vehicle`: Vehicle information and availability
- `Trip`: Complete trip lifecycle
- `Location`: Geospatial data
- `Pricing`: Fare calculations and surge pricing

### Real-time Subscriptions

- `tripUpdates`: Live trip status updates
- `driverLocation`: Real-time driver location tracking
- `pricingChanges`: Dynamic pricing updates

## Development

### Running Tests

```bash
# Unit tests
make test-unit

# Integration tests
make test-integration

# End-to-end tests
make test-e2e

# All tests
make test-all
```

### Code Generation

```bash
# Generate Protocol Buffers
make proto-gen

# Generate GraphQL resolvers
make graphql-gen

# Generate mocks
make mock-gen
```

### Database Migrations

```bash
# Create new migration
make migrate-create name=add_user_table

# Run migrations
make migrate-up

# Rollback migrations
make migrate-down
```

## Monitoring and Observability

### Metrics

- **Prometheus**: Metrics collection and alerting
- **Grafana**: Visualization dashboards
- **Custom Metrics**: Business and technical KPIs

### Tracing

- **Jaeger**: Distributed tracing
- **Correlation IDs**: Request tracking across services

### Logging

- **Structured Logging**: JSON format with correlation IDs
- **Log Levels**: Configurable per service
- **Centralized**: ELK stack for log aggregation

## Security

- **Authentication**: JWT-based with refresh tokens
- **Authorization**: Role-based access control (RBAC)
- **API Security**: Rate limiting, input validation
- **Inter-service**: mTLS for gRPC communication
- **Data Protection**: Encryption at rest and in transit

## Performance

- **Caching**: Multi-level caching strategy
- **Database**: Connection pooling and query optimization
- **Load Balancing**: Service-level load balancing
- **Horizontal Scaling**: Stateless service design

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For questions and support, please open an issue in the GitHub repository.