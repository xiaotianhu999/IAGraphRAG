# aiplusall-kb Project Structure

## Root Directory Organization

```
aiplusall-kb/
├── cmd/                    # Application entry points
│   └── server/            # Main server application
├── internal/              # Private application code (Go)
├── client/                # Go client SDK
├── frontend/              # Vue.js frontend application
├── docreader/             # Python document processing service
├── mcp-server/            # MCP (Model Context Protocol) server
├── config/                # Configuration files
├── migrations/            # Database migration scripts
├── scripts/               # Build and deployment scripts
├── docs/                  # Project documentation
├── helm/                  # Kubernetes Helm charts
└── docker/                # Docker build files
```

## Backend Structure (`internal/`)

```
internal/
├── application/           # Application layer
│   ├── repository/       # Data access interfaces
│   └── service/          # Business logic services
├── handler/              # HTTP request handlers (controllers)
├── middleware/           # HTTP middleware (auth, logging, etc.)
├── models/               # AI model integrations (chat, embedding, rerank)
├── types/                # Type definitions and interfaces
├── config/               # Configuration management
├── database/             # Database utilities and migrations
├── errors/               # Custom error types
├── logger/               # Logging utilities
├── router/               # Route definitions
├── utils/                # Utility functions
├── agent/                # AI agent functionality
├── mcp/                  # MCP client implementation
├── stream/               # Streaming response management
└── tracing/              # OpenTelemetry tracing
```

## Frontend Structure (`frontend/`)

```
frontend/
├── src/
│   ├── api/              # API client functions
│   ├── components/       # Reusable Vue components
│   ├── views/            # Page-level components
│   ├── router/           # Vue Router configuration
│   ├── stores/           # Pinia state management
│   ├── types/            # TypeScript type definitions
│   ├── utils/            # Utility functions
│   ├── hooks/            # Vue composition functions
│   ├── i18n/             # Internationalization
│   └── assets/           # Static assets (images, fonts, styles)
├── public/               # Public static files
└── packages/             # Local package dependencies
```

## Document Reader Structure (`docreader/`)

```
docreader/
├── parser/               # Document parsing implementations
│   ├── base_parser.py   # Base parser interface
│   ├── pdf_parser.py    # PDF document parser
│   ├── docx_parser.py   # Word document parser
│   ├── image_parser.py  # Image processing with OCR
│   └── web_parser.py    # Web content parser
├── models/               # Data models and configurations
├── proto/                # gRPC protocol definitions
├── utils/                # Utility functions
├── splitter/             # Text chunking and splitting
└── tests/                # Unit and integration tests
```

## Configuration Structure (`config/`)

- `config.yaml` - Main application configuration
- `config-*.yaml` - Environment-specific configurations
- Supports environment variable substitution
- YAML format with nested sections for different components

## Migration Structure (`migrations/`)

```
migrations/
├── versioned/            # Versioned migrations (golang-migrate)
├── mysql/                # MySQL-specific migrations
└── paradedb/             # ParadeDB-specific migrations
```

## Key Architectural Patterns

### Go Backend Patterns
- **Dependency Injection**: Uses Uber Dig for clean dependency management
- **Repository Pattern**: Data access abstraction in `internal/application/repository/`
- **Service Layer**: Business logic in `internal/application/service/`
- **Middleware Chain**: Authentication, logging, tracing in `internal/middleware/`
- **Clean Architecture**: Separation of concerns with clear boundaries

### Frontend Patterns
- **Component-Based**: Reusable Vue 3 components with TypeScript
- **State Management**: Centralized state with Pinia stores
- **API Layer**: Dedicated API client functions in `src/api/`
- **Route-Based Code Splitting**: Lazy loading for optimal performance

### Microservice Communication
- **gRPC**: Backend ↔ Document Reader communication
- **REST API**: Frontend ↔ Backend communication
- **Message Queues**: Redis-based async task processing
- **Event-Driven**: Internal event system for loose coupling

## File Naming Conventions

### Go Files
- Snake case for files: `user_service.go`, `auth_middleware.go`
- Package names match directory names
- Test files: `*_test.go`
- Interface files often end with `_interface.go`

### Frontend Files
- Kebab case for components: `user-menu.vue`, `knowledge-base-selector.vue`
- PascalCase for TypeScript types: `UserInfo.ts`, `ApiResponse.ts`
- Camel case for utilities: `apiClient.ts`, `dateUtils.ts`

### Python Files
- Snake case: `pdf_parser.py`, `ocr_engine.py`
- Test files: `test_*.py`
- Protocol buffers: `*_pb2.py`, `*_pb2_grpc.py`

## Import/Export Patterns

### Go
- Use relative imports within the project: `github.com/aiplusall/aiplusall-kb/internal/...`
- Group imports: standard library, third-party, internal
- Dependency injection for service dependencies

### Frontend
- Use `@/` alias for src directory imports
- Barrel exports in index files for clean imports
- Dynamic imports for route-based code splitting

### Python
- Relative imports within docreader package
- Absolute imports for external dependencies
- Protocol buffer imports for gRPC communication