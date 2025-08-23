# Project Structure and Layout

This document outlines the complete project structure for the rideshare platform, including all directories, key files, and their purposes.

## Root Directory Structure

```
rideshare-platform/
├── README.md                    # Project overview and quick start
├── Makefile                     # Build and development commands
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
├── .gitignore                   # Git ignore patterns
├── .env.example                 # Environment variables template
├── docker-compose.yml           # Local development setup
├── docker-compose.prod.yml      # Production Docker setup
├── services/                    # Microservices directory
├── shared/                      # Shared libraries and utilities
├── infrastructure/              # Infrastructure as code
├── tests/                       # Test suites
├── docs/                        # Documentation
├── scripts/                     # Build and deployment scripts
└── deployments/                 # Deployment configurations
```

## Services Directory

Each service follows a consistent structure with domain-driven design principles:

```
services/
├── user-service/
│   ├── cmd/
│   │   └── main.go             # Service entry point
│   ├── internal/
│   │   ├── config/             # Configuration management
│   │   ├── domain/             # Domain models and interfaces
│   │   ├── repository/         # Data access layer
│   │   ├── service/            # Business logic layer
│   │   ├── handler/            # HTTP/gRPC handlers
│   │   └── middleware/         # Service-specific middleware
│   ├── proto/                  # Service-specific protobuf files
│   ├── migrations/             # Database migrations
│   ├── Dockerfile              # Container definition
│   ├── go.mod                  # Service module
│   └── README.md               # Service documentation
├── vehicle-service/            # Same structure as user-service
├── geo-service/                # Same structure as user-service
├── matching-service/           # Same structure as user-service
├── pricing-service/            # Same structure as user-service
├── trip-service/               # Same structure as user-service
├── payment-service/            # Same structure as user-service
└── api-gateway/
    ├── cmd/
    │   └── main.go
    ├── internal/
    │   ├── config/
    │   ├── resolver/           # GraphQL resolvers
    │   ├── middleware/
    │   └── client/             # gRPC clients
    ├── schema/                 # GraphQL schema definitions
    ├── Dockerfile
    ├── go.mod
    └── README.md
```

## Shared Directory

Common libraries and utilities used across services:

```
shared/
├── proto/                      # Shared protocol buffer definitions
│   ├── common/
│   │   ├── types.proto         # Common data types
│   │   ├── errors.proto        # Error definitions
│   │   └── events.proto        # Event definitions
│   ├── user/
│   │   └── user.proto          # User service definitions
│   ├── vehicle/
│   │   └── vehicle.proto       # Vehicle service definitions
│   ├── geo/
│   │   └── geo.proto           # Geo service definitions
│   ├── matching/
│   │   └── matching.proto      # Matching service definitions
│   ├── pricing/
│   │   └── pricing.proto       # Pricing service definitions
│   ├── trip/
│   │   └── trip.proto          # Trip service definitions
│   └── payment/
│       └── payment.proto       # Payment service definitions
├── models/                     # Shared domain models
│   ├── user.go
│   ├── vehicle.go
│   ├── trip.go
│   ├── location.go
│   └── pricing.go
├── utils/                      # Utility functions
│   ├── logger/                 # Structured logging
│   ├── crypto/                 # Cryptographic utilities
│   ├── geo/                    # Geospatial calculations
│   ├── validation/             # Input validation
│   └── time/                   # Time utilities
├── middleware/                 # Common middleware
│   ├── auth.go                 # Authentication middleware
│   ├── cors.go                 # CORS middleware
│   ├── logging.go              # Request logging
│   ├── metrics.go              # Metrics collection
│   └── ratelimit.go            # Rate limiting
├── config/                     # Configuration management
│   ├── config.go               # Configuration structures
│   └── loader.go               # Configuration loading
└── events/                     # Event definitions and handlers
    ├── types.go                # Event type definitions
    ├── publisher.go            # Event publishing
    └── subscriber.go           # Event subscription
```

## Infrastructure Directory

Infrastructure as code and deployment configurations:

```
infrastructure/
├── docker/                     # Docker configurations
│   ├── Dockerfile.base         # Base Go image
│   ├── Dockerfile.service      # Service template
│   └── docker-compose/
│       ├── development.yml     # Development environment
│       ├── testing.yml         # Testing environment
│       └── production.yml      # Production environment
├── kubernetes/                 # Kubernetes manifests
│   ├── namespaces/
│   ├── configmaps/
│   ├── secrets/
│   ├── services/
│   ├── deployments/
│   ├── ingress/
│   └── rbac/
├── helm/                       # Helm charts
│   └── rideshare-platform/
│       ├── Chart.yaml
│       ├── values.yaml
│       ├── values-dev.yaml
│       ├── values-prod.yaml
│       └── templates/
└── monitoring/                 # Monitoring configurations
    ├── prometheus/
    │   ├── prometheus.yml
    │   └── rules/
    ├── grafana/
    │   ├── dashboards/
    │   └── provisioning/
    └── jaeger/
        └── jaeger.yml
```

## Tests Directory

Comprehensive test suites:

```
tests/
├── unit/                       # Unit tests
│   ├── user/
│   ├── vehicle/
│   ├── geo/
│   ├── matching/
│   ├── pricing/
│   ├── trip/
│   └── payment/
├── integration/                # Integration tests
│   ├── api/                    # API integration tests
│   ├── database/               # Database integration tests
│   └── grpc/                   # gRPC integration tests
├── e2e/                        # End-to-end tests
│   ├── scenarios/              # Test scenarios
│   ├── fixtures/               # Test data fixtures
│   └── helpers/                # Test helper functions
├── load/                       # Load testing
│   ├── k6/                     # K6 load test scripts
│   └── artillery/              # Artillery load test configs
└── contract/                   # Contract testing
    ├── pact/                   # Pact contract tests
    └── schemas/                # Schema validation tests
```

## Documentation Directory

Comprehensive project documentation:

```
docs/
├── architecture.md             # System architecture overview
├── project-structure.md        # This file
├── api/                        # API documentation
│   ├── graphql/
│   │   ├── schema.md           # GraphQL schema documentation
│   │   └── queries.md          # Example queries and mutations
│   └── grpc/
│       └── services.md         # gRPC service documentation
├── deployment/                 # Deployment guides
│   ├── local.md                # Local development setup
│   ├── kubernetes.md           # Kubernetes deployment
│   └── production.md           # Production deployment guide
├── development/                # Development guides
│   ├── getting-started.md      # Developer onboarding
│   ├── coding-standards.md     # Code style and standards
│   ├── testing.md              # Testing guidelines
│   └── debugging.md            # Debugging guide
├── operations/                 # Operational guides
│   ├── monitoring.md           # Monitoring and alerting
│   ├── logging.md              # Logging best practices
│   ├── troubleshooting.md      # Common issues and solutions
│   └── performance.md          # Performance optimization
└── design/                     # Design documents
    ├── database-schema.md      # Database design
    ├── event-sourcing.md       # Event sourcing implementation
    ├── security.md             # Security considerations
    └── scalability.md          # Scalability patterns
```

## Scripts Directory

Build, deployment, and utility scripts:

```
scripts/
├── build/                      # Build scripts
│   ├── build-all.sh           # Build all services
│   ├── build-service.sh       # Build individual service
│   └── generate.sh            # Code generation
├── deploy/                     # Deployment scripts
│   ├── deploy-local.sh        # Local deployment
│   ├── deploy-k8s.sh          # Kubernetes deployment
│   └── rollback.sh            # Rollback deployment
├── database/                   # Database scripts
│   ├── migrate.sh             # Run migrations
│   ├── seed.sh                # Seed test data
│   └── backup.sh              # Database backup
├── testing/                    # Testing scripts
│   ├── run-tests.sh           # Run all tests
│   ├── integration-test.sh    # Integration tests
│   └── load-test.sh           # Load testing
└── utilities/                  # Utility scripts
    ├── cleanup.sh             # Cleanup resources
    ├── logs.sh                # Collect logs
    └── health-check.sh        # Health check script
```

## Deployments Directory

Environment-specific deployment configurations:

```
deployments/
├── local/                      # Local development
│   ├── docker-compose.yml
│   ├── .env
│   └── init-scripts/
├── development/                # Development environment
│   ├── kubernetes/
│   ├── helm/
│   └── configs/
├── staging/                    # Staging environment
│   ├── kubernetes/
│   ├── helm/
│   └── configs/
└── production/                 # Production environment
    ├── kubernetes/
    ├── helm/
    ├── configs/
    └── secrets/
```

## Key Configuration Files

### Root Level Files

- **Makefile**: Contains all build, test, and deployment commands
- **docker-compose.yml**: Local development environment setup
- **go.mod**: Main module definition with workspace configuration
- **.env.example**: Template for environment variables
- **.gitignore**: Git ignore patterns for Go, Docker, and IDE files

### Service Level Files

Each service contains:
- **cmd/main.go**: Service entry point with dependency injection
- **internal/config/**: Configuration management with environment variables
- **Dockerfile**: Multi-stage build for production optimization
- **go.mod**: Service-specific module definition
- **README.md**: Service-specific documentation

## Development Workflow

1. **Local Setup**: Use `make dev-setup` to initialize the development environment
2. **Code Generation**: Run `make generate` to generate protobuf and GraphQL code
3. **Testing**: Use `make test` for comprehensive testing
4. **Building**: Use `make build` to build all services
5. **Deployment**: Use `make deploy-local` for local deployment

This structure provides clear separation of concerns, maintainable code organization, and scalable architecture patterns suitable for a production-grade rideshare platform.