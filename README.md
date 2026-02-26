# Go Boilerplate

A production-ready Golang REST API boilerplate with common features and best practices built-in.

## Features

- **Web Framework**: Gin - High-performance HTTP framework
- **Database**: GORM with PostgreSQL and MySQL support
- **Caching**: Redis integration
- **Message Queue**: RabbitMQ support
- **Authentication**: JWT middleware
- **Validation**: Request validation with custom validators
- **Logging**: Structured logging
- **Worker Pool**: Background job processing with ants
- **AI Integration**: Google Gemini AI client (optional)
- **Storage**: AWS S3 integration (optional)
- **Docker**: Multi-stage Docker builds
- **Docker Compose**: Ready-to-run local development

## Prerequisites

- Go 1.24+
- PostgreSQL 14+ (or MySQL 8+)
- Redis 7+
- RabbitMQ 3.12+

## Quick Start

### 1. Clone and Install

```bash
git clone <your-repo-url>
cd go-boilerplate
go mod tidy
```

### 2. Configure Environment

Create a `.env` file in the project root:

```bash
# Application
APP_ENV=development
APP_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=boilerplate_db
DB_USER=postgres
DB_PASS=postgres

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_USER=default
REDIS_PASS=
REDIS_POOL_SIZE=10

# RabbitMQ
RABBIT_HOST=localhost
RABBIT_PORT=5672
RABBIT_USER=guest
RABBIT_PASS=guest

# AI (optional)
GEMINI_API_KEY=your_api_key
GEMINI_MODEL=gemini-2.0-flash-exp
```

### 3. Run Development Server

```bash
go run cmd/api/main.go
```

The server will start on `http://localhost:8080`

### 4. Health Check

```bash
curl http://localhost:8080/health
```

## Project Structure

```
go-boilerplate/
├── cmd/
│   ├── api/          # Main application entry point
│   └── migrate/      # Database migration tool
├── configs/          # Configuration management
├── internal/
│   ├── common/       # Common types, enums, models
│   ├── handler/      # HTTP handlers/controllers
│   ├── pkg/          # Internal packages
│   │   ├── ai-connector/  # AI client
│   │   ├── db/            # Database utilities
│   │   ├── helper/        # Helper functions
│   │   ├── jwt/           # JWT authentication
│   │   ├── logger/        # Logging
│   │   ├── middleware/    # HTTP middleware
│   │   ├── rabbitmq/      # RabbitMQ client
│   │   ├── redis/         # Redis client
│   │   ├── storage/       # S3 storage
│   │   └── validation/    # Custom validators
│   ├── repository/   # Data access layer
│   ├── server/       # Server setup
│   └── service/      # Business logic layer
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── go.mod
```

## Adding Your First Feature

### 1. Create a Model

Create `internal/common/models/user.go`:

```go
package models

import "time"

type User struct {
    ID        uint      `json:"id" gorm:"primaryKey"`
    Name      string    `json:"name" gorm:"not null"`
    Email     string    `json:"email" gorm:"uniqueIndex"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 2. Create Repository

Create `internal/repository/user/repository.go`:

```go
package user

import (
    "context"
    "go-boilerplate/internal/common/models"
    "go-boilerplate/internal/pkg/db"
    "go-boilerplate/internal/pkg/redis"
)

type IRepository interface {
    Create(ctx context.Context, user *models.User) error
    // Add more methods
}

type Repository struct {
    redis redis.IRedis
    db    *database.Database
}

func NewRepo(ctx context.Context, redis redis.IRedis, db *database.Database) IRepository {
    return &Repository{redis: redis, db: db}
}

func (r *Repository) Create(ctx context.Context, user *models.User) error {
    return r.db.Db.WithContext(ctx).Create(user).Error
}
```

### 3. Create Service

Create `internal/service/user/service.go`:

```go
package user

import (
    "context"
    "go-boilerplate/internal/common/models"
    "go-boilerplate/internal/repository"
)

type IService interface {
    CreateUser(ctx context.Context, user *models.User) error
}

type Service struct {
    redis redis.IRedis
    repo  user.IRepository
}

func NewService(ctx context.Context, redis redis.IRedis, repo repository.IRepository) IService {
    return &Service{redis: redis, repo: repo.User}
}

func (s *Service) CreateUser(ctx context.Context, user *models.User) error {
    return s.repo.Create(ctx, user)
}
```

### 4. Create Handler

Create `internal/handler/user/handler.go`:

```go
package user

import (
    "github.com/gin-gonic/gin"
)

type Handler struct {
    service user.IService
}

func NewHandler(ctx context.Context, service user.IService) *Handler {
    return &Handler{service: service}
}

func (h *Handler) NewRoutes(r *gin.RouterGroup) {
    r.POST("/users", h.CreateUser)
}

func (h *Handler) CreateUser(c *gin.Context) {
    // Implementation
}
```

### 5. Register Routes

Update `internal/server/api.server.go`:

```go
func InitRoutes(...) {
    // Initialize
    userHandler := user.NewHandler(ctx, userService)
    userHandler.NewRoutes(e)
}
```

## Docker

### Build

```bash
docker build -t go-boilerplate .
```

### Run with Docker Compose

```bash
docker-compose up -d
```

## Makefile Commands

```bash
make build      # Build the application
make run        # Run the application
make test       # Run tests
make clean      # Clean build artifacts
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| APP_ENV | Application environment | development |
| APP_PORT | Server port | 8080 |
| DB_HOST | Database host | localhost |
| DB_PORT | Database port | 5432 |
| DB_NAME | Database name | - |
| DB_USER | Database user | - |
| DB_PASS | Database password | - |
| REDIS_HOST | Redis host | localhost |
| REDIS_PORT | Redis port | 6379 |
| RABBIT_HOST | RabbitMQ host | localhost |
| RABBIT_PORT | RabbitMQ port | 5672 |

## License

MIT License - feel free to use this boilerplate for your projects.
