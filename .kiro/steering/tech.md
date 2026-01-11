# aiplusall-kb Technology Stack

## Backend Stack
- **Language**: Go 1.24+ with modules
- **Framework**: Gin web framework
- **Database**: PostgreSQL with pgvector extension (ParadeDB image)
- **ORM**: GORM for database operations
- **Authentication**: JWT tokens with role-based access control
- **API Documentation**: Swagger/OpenAPI with swag

## Document Processing
- **Language**: Python 3.x
- **Service**: gRPC-based document reader microservice
- **OCR**: PaddleOCR for text extraction from images
- **Parsers**: Support for PDF, DOCX, DOC, Excel, Markdown, HTML, images
- **Multimodal**: Vision-Language Model integration for image understanding

## Frontend Stack
- **Framework**: Vue 3 with TypeScript
- **Build Tool**: Vite
- **UI Library**: TDesign Vue Next
- **State Management**: Pinia
- **Styling**: Less CSS preprocessor
- **Internationalization**: Vue I18n

## Infrastructure & Storage
- **Containerization**: Docker with multi-stage builds
- **Orchestration**: Docker Compose with profiles
- **Vector Database**: PostgreSQL pgvector or Qdrant
- **Search Engine**: Elasticsearch (optional)
- **Knowledge Graph**: Neo4j (optional)
- **Object Storage**: Minio or Tencent COS
- **Cache & Queue**: Redis for caching and async tasks (Asynq)
- **Tracing**: OpenTelemetry with Jaeger

## Development Tools
- **Dependency Injection**: Uber Dig
- **Migration**: golang-migrate
- **Code Quality**: golangci-lint
- **Hot Reload**: Air for Go backend development
- **Package Management**: Go modules, npm/pnpm for frontend

## Common Build Commands

### Development Mode (Recommended)
```bash
# Start infrastructure only
make dev-start

# Start backend (new terminal)
make dev-app

# Start frontend (new terminal) 
make dev-frontend

# Or use one-click start
./scripts/quick-dev.sh
```

### Production Build
```bash
# Build all Docker images
make docker-build-all

# Start all services
make start-all

# Stop all services
make stop-all
```

### Database Operations
```bash
# Run migrations
make migrate-up

# Rollback migrations
make migrate-down

# Clean database (development)
make clean-db
```

### Code Quality
```bash
# Format code
make fmt

# Run linter
make lint

# Generate API docs
make docs

# Run tests
make test
```

## Environment Configuration
- Use `.env` file for local development (copy from `.env.example`)
- Docker environment variables for containerized deployment
- Configuration files in `config/` directory with YAML format
- Multi-profile Docker Compose for different feature sets