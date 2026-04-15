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
make deps                       # Install all dev tools (goose, buf, mockgen, protoc, linters, go-test-coverage)
make generate                   # Run protobuf/gRPC codegen for all Go services + auth lib
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

Go services follow a layered architecture under `internal/`:
- **grpc/** - gRPC service implementations (entry points)
- **usecase/** - Business logic layer
- **repo/** - Database repositories (PostgreSQL)
- **domain/** - Domain models

Tasker additionally has:
- **handlers/** - Message handlers (SQS event processing)
- **services/** - External service integrations (LLM, embeddings, push, scheduler)
- **workers/** - Background workers (SQS consumers, scheduled jobs)

Notificator additionally has:
- **worker/** - Background worker (SQS consumer)

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
- **auth** - Auth service gRPC client stubs (proto-generated)
- **errors** - Domain error types
- **log** - Structured logging wrapper
- **slices** - Slice utility helpers
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
- `screens/` - Page-level components (Auth, ResetPassword, Tasks, Schedule, Notifications, Settings)
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

Protobuf codegen in CI uses the custom action at `.github/actions/codegen/`. Shared Go service templates: `go-service-ci.yml`, `go-service-release.yml`.

## Linting

Go linter config (`.golangci.yml`): errcheck, goconst, gosec, govet, ineffassign, staticcheck, unused, protogetter. Formatter: goimports.

## Testing

Go tests use `testcontainers-go` for integration tests with real PostgreSQL. Environment variable `TESTCONTAINERS_RYUK_DISABLED=true` is set in the Makefile. Run a single test with:

```bash
cd backend && go test ./tasker/internal/... -run TestName -race
```

## Save decisions
After successful feature implementations save relevant decisions here. Save only things that matters for next sessions. 
If you are not sure that you would need this info to make less research of project do not save. You must update info if it becomes irrelevant after the feature was developed.

## Decisions

### Domain errors pattern
Define sentinel errors with `errors.Error("...")` in domain packages (see `tasker/internal/domain/repo.go` `ErrTaskNotFound`, `notificator/internal/domain/notification/setting.go` `ErrInvalidSettingType`). Map them to gRPC status codes in the grpc layer with `errors.Is()`.

### Notification types
Available notification setting types are defined in code at `notificator/internal/domain/notification/setting.go` (`AvailableSettingTypes`), not in a DB table. Tasker sets the `Type` field on `domain.Notification` when sending via SQS (`"upcoming_event"`, `"schedule_change"`). Both lists must stay in sync.

### Tasker → Notificator communication
Tasker sends push notifications via SQS using protobuf `pushpb.Notification` messages (see `tasker/internal/services/notifications/service.go`). The notificator consumes these from SQS. Three send sites: upcoming event notifier, schedule change notifier (both in `scenarios.go`), and scheduling usecase (`usecase/scheduling/usecase.go`).