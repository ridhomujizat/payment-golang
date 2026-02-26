# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make run              # Run the API server (go run cmd/api/main.go)
make build            # Build binary to ./build/api
make test             # Run all tests (go test -v ./...)
make test-coverage    # Tests with HTML coverage report
make lint             # Run golangci-lint
make fmt              # Format code (go fmt ./...)
make tidy             # go mod tidy
make migrate          # Run GORM AutoMigrate (go run cmd/migrate/main.go)
make dev              # Hot reload with air
make docker-up        # Start containers via docker-compose
make docker-down      # Stop containers
```

Run a single test: `go test -v -run TestName ./internal/service/example/...`

## Architecture

Go 1.24 REST API using **Gin** framework, **GORM** ORM (PostgreSQL/MySQL), **Redis** caching, **RabbitMQ** messaging, and optional **Google Gemini AI** / **AWS S3** integrations.

Module name: `go-boilerplate`

### Layered structure

**Handler → Service → Repository** pattern with interface-based dependency injection throughout.

- `cmd/api/main.go` — Entry point. Bootstraps all infrastructure (Redis, RabbitMQ, DB, AI), then calls `serverApp.Setup()`.
- `cmd/migrate/main.go` — Standalone migration runner using GORM AutoMigrate.
- `configs/` — Config struct with `env` struct tags, loaded via reflection from `.env` file (godotenv).
- `internal/server/api.server.go` — `Setup()` initializes middleware and routes. `InitRoutes()` wires handler→service→repository. All routes are under `/api` base path.
- `internal/server/worker.server.go` — Background worker pool using `ants`. Workers run in non-development environments and consume from RabbitMQ.

### Adding a new feature

1. **Model** in `internal/common/models/`
2. **Repository** — Create package under `internal/repository/<domain>/`, define `IRepository` interface. Register it in `internal/repository/regist.repo.go` (`IRepository` struct).
3. **Service** — Create package under `internal/service/<domain>/`, define `IService` interface and `Service` struct. Constructor takes `context.Context`, `redis.IRedis`, `rabbitmq.ConnectionManager`, `rabbitmq.Publisher`, and `repository.IRepository`.
4. **Handler** — Create package under `internal/handler/<domain>/`, split into `controller.handler.go` (struct + constructor + handler methods) and `router.handler.go` (route registration via `NewRoutes(*gin.RouterGroup)`).
5. **Wire up** in `internal/server/api.server.go` `InitRoutes()`: instantiate service, handler, call `handler.NewRoutes(e)`.

### Response pattern

Handlers retrieve a `send` function from Gin context (`c.MustGet("send")`) set by `middleware.ResponseInit()`. Services return `*types.Response` which is passed to `send()`. Use `helper.ParseResponse()` to construct responses.

### Key internal packages (`internal/pkg/`)

- `middleware/` — CORS, request/response init, JWT auth, encryption, multipart file handling
- `jwt/` — Token generation/validation
- `db/` — GORM setup, migrations, Redis-based query caching (`go-gorm/caches`), transaction helpers
- `redis/` — Redis client with `IRedis` interface
- `rabbitmq/` — Connection manager with auto-reconnect, Publisher for message publishing
- `helper/` — JSON parsing, HTTP utilities, encryption, env helpers
- `validation/` — Request validation using `go-playground/validator`
- `ai-connector/` — Google Gemini AI client
- `storage/s3/` — AWS S3 client

### Naming conventions

- Files use dot-separated names indicating purpose: `<name>.<layer>.<scope>.go` (e.g., `cors.middleware.pkg.go`, `response.type.common.go`)
- Handler packages contain `controller.handler.go` (struct/logic) and `router.handler.go` (route definitions)
- Service packages contain `regist.service.go` (interface/constructor) and `<name>.service.go` (method implementations)

### Environment

All config is loaded from `.env` via struct tags on `configs.Config`. See `.env.example` for required variables. The config loader uses reflection and will panic if a tagged env var is missing.
