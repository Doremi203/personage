# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Workflow

### Git Worktrees
- **Never use a git worktree for any code change if user not explicitly said overwise.** Create and enter one before touching any file if user said **use worktree**.
- **Create worktrees under `.claude/worktrees/`.** Do not create worktrees outside this directory.
- After creating and entering a worktree, run `cd backend && make generate` to generate all protobuf/gRPC gen files before making changes.
- After finishing work in a worktree, always create a pull request before removing the worktree.
- When development is complete, ask the user "Ready to remove the worktree?" before calling ExitWorktree with `action: "remove"`.

### Commands
Always use `make` targets instead of invoking tools directly. Never run `go test`, `golangci-lint`, `go build`, `goose`, or similar tools as raw commands — use the corresponding `make` target (e.g. `make tests` not `go test ./...`, `make lint` not `golangci-lint ./...`).

### Codebase exploration
Before reading files in the main session, use the Explore agent whenever you are not yet sure which files or line ranges to read. Only read directly when you already know the exact file and relevant lines.

### Implementing plans
When executing a multi-task plan create a todo list and spawn a separate agent per task whenever you have enough context to write detailed instructions for it. This keeps the main session context lean and allows parallel execution.


## Project Overview

Personage is a personal assistant web application with intelligent task management, scheduling, and notifications. It uses a microservices architecture deployed to Yandex Cloud.

## Repository Structure

```
personage/
├── backend/          # All backend services (Go monorepo + C# auth + Python services)
│   ├── tasker/       # Go - Task management, event clustering, LLM-based task generation
│   ├── notificator/  # Go - Web push notifications via VAPID
│   ├── Personage.Auth/  # C#/.NET - Authentication & JWT
│   ├── telegram-auth/   # Python - Server-side Telegram client session manager (full MTProto client authenticated per user)
│   ├── traitex/         # Python - Trait extraction/NLP
│   └── libs/            # Shared Go libraries (errors, log, postgres, sqs, token, webapp, tx)
├── frontend/         # React 19 + TypeScript + Vite + Tailwind PWA
└── terraform/        # Yandex Cloud IaC
```

## Build & Development Commands

### Backend (run from `backend/`)

```bash
make deps                       # Install all dev tools (goose, buf, mockgen, protoc, linters, go-test-coverage)
make generate                   # Run protobuf/gRPC codegen for all Go services + auth lib (uses buf generate via go generate ./... — NEVER run protoc or buf directly)
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
make tasker/create-test-tasks   # Seed DB with test tasks (runs scripts/create_test_tasks.go)
make auth/proto/generate        # Regenerate auth client stubs in libs/go/auth
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
make frontend-build   # Vite production build only
make frontend-deploy  # Deploy dist/ to S3 (requires secrets.env)
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

Design docs live in `backend/tasker/`: `DEFINITIONS.md`, `FUNCTIONAL_REQUIREMENTS.md`, `USE_CASES.md`, `ARCHITECTURE.puml`, `SEQUENCE_EVENT_TO_TASK.puml`. Consult before non-trivial changes to event→task flow.

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

### Functional tests (`tests/`)

End-to-end/API tests in TypeScript + Jest, separate from Go unit/integration tests. Own `docker-compose.yml` and `init.sql` spin up dependencies; specs live under `tests/{tasker,notificator}/specs`, clients under `tests/{service}/client`, shared helpers under `tests/{service}/helpers`.

```bash
cd tests && npm test              # all functional tests
cd tests && npm run test:tasker   # tasker only
cd tests && npm run test:watch    # watch mode
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

### Tasker eval bootstrap fixtures
`backend/tasker/eval` fixtures may start with `expected_tasks: []` for the first curation run. The eval loader should allow empty expected tasks so the initial report can surface `unmatched_generated` items.

### Tasker eval Traitex transport
The documented remote Traitex endpoint `grpc-traitex.persomanage.ru:443` requires TLS. `tasker/eval/cmd/f1` should default to TLS and only use plaintext when explicitly requested via `--traitex-insecure` for local testing.

### Tasker eval wait window
`backend/tasker/eval/internal/runner` must wait long enough for all clusters to close before collecting generated tasks. The default `overall-timeout` is 20 minutes; `inactivityMinutes` (currently 10) determines when clusters close, so the timeout must exceed replay time + inactivity window. The runner logs snapshot replay and per-poll task count progress to stderr while waiting.

### Traitex processing snapshots
Traitex snapshot recording is time-window based. Creating a snapshot only inserts a `[start, finish]` row into `traitex.processing_snapshot`; Gmail/Telegram consumers keep running normally, and if `now()` falls inside any snapshot window they also persist each enriched event to `processing_result` before sending it to SQS. Replay does not use `snapshot_id` links: `SendProcessingSnapshot` reloads rows by `processed_at BETWEEN snapshot.from_ AND snapshot.to`, capped at 1000 events.

### Auth API JWT middleware
`backend/Personage.Auth/Personage.Auth.Api/Program.cs` must include both `app.UseAuthentication()` and `app.UseAuthorization()` before `app.MapControllers()`. Without them, `[Authorize]` REST endpoints like `/user` return `401` even after a successful access-token refresh.

### Traitex YMQ client config
`backend/traitex` must keep the YMQ service endpoint and default queue URL as separate settings. `YMQ.EndpointUrl` is the base API endpoint (`https://message-queue.api.cloud.yandex.net`), while `YMQ.QueueUrl` is the default queue; `traitex/messaging/YMQClient.py` normalizes old queue-in-endpoint configs, but custom replay targets in `SendProcessingSnapshot` rely on the client endpoint staying at the base service URL.
