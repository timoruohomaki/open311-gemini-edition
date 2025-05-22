# Project Structure

open311-api/
├── cmd/
│   └── api/
│       └── main.go                # Main application entry point, server setup, DI wiring
├── internal/
│   ├── config/
│   │   └── config.go              # Configuration loading and management (e.g., API keys, DB DSN)
│   ├── handler/
│   │   ├── http/                  # HTTP specific handlers
│   │   │   ├── open311_handler.go # Handlers for /services, /requests
│   │   │   ├── router.go          # HTTP router setup (e.g., using chi, gorilla/mux, or stdlib)
│   │   │   └── middleware.go      # Common HTTP middleware (auth, logging, recovery)
│   │   └── (other_protocols_if_any)/ # e.g., grpc/
│   ├── model/                     # Data structures (entities, DTOs)
│   │   ├── service.go             # Service type model
│   │   └── request.go             # ServiceRequestInput, ServiceRequestOutput models
│   ├── repository/                # Data access layer (interfaces and implementations)
│   │   ├── memory/                # In-memory implementation
│   │   │   └── open311_repo.go
│   │   ├── (postgres)/            # Example: PostgreSQL implementation
│   │   │   └── open311_repo.go
│   │   └── repository.go          # Interfaces for repositories (e.g., ServiceRepository, RequestRepository)
│   ├── service/                   # Business logic layer
│   │   └── open311_service.go     # Service logic (e.g., creating requests, fetching services)
│   └── platform/                  # Platform-specific concerns
│       └── logger/
│           └── syslog_logger.go   # Syslog logger implementation
├── api/                           # API specifications (e.g., OpenAPI/Swagger files)
│   └── openapi.yaml               # (Optional) OpenAPI spec for your API
├── pkg/                           # Public library code (reusable by other projects - keep minimal for apps)
│   └── (utils)/                   # Example: generic utility functions
├── configs/                       # Application configuration files (e.g., .env, .yaml, .json)
│   └── config.dev.yaml
├── scripts/                       # Helper scripts (build, deploy, test)
│   └── run_dev.sh
├── go.mod
├── go.sum
└── DEVELOPER_GUIDE.md
