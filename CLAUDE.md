# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Personage is a personal assistant web application with intelligent task management, scheduling, and notifications. It uses a microservices architecture deployed to Yandex Cloud.

## Repository Structure

```
personage/
├── backend/          # All backend services (Go monorepo + C# auth + Python services)
│   ├── tasker/       # Go - Task management, event clustering, LLM-based task generation
│   ├── notificator/  # Go - Web push notifications via VAPID
│   ├── Personage.Auth/  # C#/.NET - Authentication & JWT
│   ├── telegram-auth/   # Python - Telegram OAuth
│   ├── traitex/         # Python - Trait extraction/NLP
│   └── libs/            # Shared Go libraries (errors, log, postgres, sqs, token, webapp, tx)
├── frontend/         # React 19 + TypeScript + Vite + Tailwind PWA
└── terraform/        # Yandex Cloud IaC
```

## Build & Development Commands

### Backend (run from `backend/`)

```bash
make deps                       # Install all dev tools (goose, buf, mockgen, protoc, linters)
make generate                   # Run protobuf/gRPC codegen for all Go services
make services/deploy            # Start docker-compose (PostgreSQL + services)
make lint                       # Run golangci-lint
make tests                      # Run all Go tests with race detection

# Per-service
make tasker/run                 # Build and run tasker in Docker
make notificator/run            # Build and run notificator in Docker
make {service}/generate         # Code generation for a specific service
make {service}/migrate/up       # Run DB migrations
make {service}/migrate/down     # Reset DB migrations
make {service}/migrate/down-one # Roll back one migration
make {service}/migrate/create name=<name>  # Create new migration
```

Services: `tasker`, `notificator`, `auth`, `traitex`

### Frontend (run from `frontend/`)

```bash
npm run dev          # Vite dev server
npm run build        # Production build
npm run lint         # ESLint
npm run typecheck    # TypeScript type checking (tsc --noEmit)
```

### Root-level

```bash
make secrets          # Fetch Yandex Cloud tokens and AWS credentials into secrets.env
make frontend-release # Build + deploy frontend to S3
```

## Architecture

### Backend - Go Services

Both Go services (tasker, notificator) follow the same layered architecture under `internal/`:
- **grpc/** - gRPC service implementations (entry points)
- **handlers/** - Message handlers (SQS event processing)
- **usecase/** - Business logic layer
- **services/** - External service integrations (LLM, embeddings, push, scheduler)
- **workers/** - Background workers (SQS consumers, scheduled jobs)
- **repo/** - Database repositories (PostgreSQL)
- **domain/** - Domain models

### API Design

APIs are protobuf-first. Proto files live in each service's `api/` directory. Code generation produces:
- Go gRPC server/client stubs
- gRPC-Gateway HTTP handlers (REST endpoints)
- OpenAPI v2 specs
- Validation code (protoc-gen-validate)

Generation pipeline: proto files → `buf generate` (via `go generate ./...`) → `gen/` and `swagger/` output directories.

### Shared Libraries (`backend/libs/go/`)

- **webapp** - Base web application framework: wires up gRPC server + HTTP gateway + CORS + health checks
- **postgres** - Connection pooling via pgx, helpers
- **sqs** - AWS SQS message consumer/producer
- **token** - JWT verification middleware
- **errors** - Domain error types
- **log** - Structured logging wrapper
- **tx** - Database transaction helpers

### Inter-Service Communication

- **SQS** for async messaging between services (event processing, notification dispatch)
- **gRPC** for synchronous service-to-service calls
- HTTP gateway auto-generated from gRPC definitions for frontend consumption

### Database

- PostgreSQL 18 with pgvector extension (for AI embedding similarity search)
- Separate databases per service: `tasker`, `notificator`, `auth`, `traitex`
- Migrations managed by goose (SQL files in `{service}/migrations/`)
- Local dev DB: `postgres://user:pass@localhost:5432/{db}?sslmode=disable`

### Frontend

- React 19 SPA with PWA support (service worker in `sw.ts`)
- `screens/` - Page-level components (Auth, Tasks, Schedule, Notifications, Settings)
- `components/` - Reusable UI components
- `utils/` - API service clients (authService, taskerService, notificatorService, pushNotifications)
- Styling: Tailwind CSS 4.2
- Backend communication via HTTP (gRPC-Gateway endpoints)

### AI/ML Pipeline (Tasker)

Tasker uses AI for intelligent task management:
1. Events arrive via SQS → stored in PostgreSQL
2. Events are embedded using OpenAI embeddings and clustered via pgvector similarity
3. Closed clusters are processed by an LLM (OpenRouter) to generate actionable tasks
4. Tasks are scheduled and trigger notifications via the notificator service

## CI/CD

GitHub Actions workflows per service follow the pattern:
- **CI** (`{service}-ci.yml`): codegen → lint → test → build (on PR/push)
- **Release** (`{service}-release.yml`): full pipeline + Docker push to Yandex Container Registry + DB migration + Terraform deploy

Protobuf codegen in CI uses the custom action at `.github/actions/codegen/`.

## Linting

Go linter config (`.golangci.yml`): errcheck, goconst, gosec, govet, ineffassign, staticcheck, unused, protogetter. Formatter: goimports.

## Testing

Go tests use `testcontainers-go` for integration tests with real PostgreSQL. Environment variable `TESTCONTAINERS_RYUK_DISABLED=true` is set in the Makefile. Run a single test with:

```bash
cd backend && go test ./tasker/internal/... -run TestName -race
```
