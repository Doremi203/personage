---
paths:
  - "**/*.go"
---

# Go Style Guide

Project-specific Go conventions for `personage`. Complements `modern-go.md` (language features) and `go-unit-tests.md` (test structure). When this file conflicts with general Go advice you've seen elsewhere, this file wins.

---

## 1. Errors

Use the project's `libs/go/errors` package. **Do not** use `fmt.Errorf("%w", err)` or `errors.New` from stdlib for new code.

- Sentinel errors are package-level `var`s named `ErrXxx`, declared with `errors.Error("...")`.
- Wrap with `errors.WrapFail(err, "doing X")` or `errors.WrapFailf(err, "doing X for %s", errors.Token("user_id", id))`.
- Use `errors.Token("name", value)` to attach structured key/value context — never interpolate values into the message format.
- Match wrapped sentinels with `errors.Is(err, ErrXxx)`. Map to gRPC status codes in the grpc layer, never deeper.

Example — `tasker/internal/domain/repo.go`:
```go
var ErrTaskNotFound = errors.Error("task not found")
```

Example — `tasker/internal/usecase/scheduling/usecase.go`:
```go
userIDs, err := uc.taskRepo.GetUsersWithUnplannedTasks(ctx)
if err != nil {
    return errors.WrapFail(err, "get users with pending tasks")
}
```

## 2. Logging

Use `libs/go/log.Logger` exclusively. The interface is `Infof(format, args...)`, `Warn(error)`, `Error(error)`. Inject the logger as a constructor dep — never use a package-level logger.

- Pass errors as values to `Warn`/`Error`. Build the message via `errors.WrapFailf` + `errors.Token` so structured context is preserved.
- **Log an error exactly once, at the layer where it stops being recoverable.** If a function aborts because of `err`, log it where it aborts and return. If a function recovers and continues (e.g. a per-user loop where one user's failure is non-fatal), log at that layer and continue.
- No `fmt.Println`, `log.Printf`, or string interpolation in log messages.

Example — recover-and-continue pattern from `scheduling/usecase.go`:
```go
for _, userID := range userIDs {
    if err := uc.scheduleForUser(ctx, userID); err != nil {
        uc.logger.Error(errors.WrapFailf(
            err,
            "schedule tasks for user %s",
            errors.Token("user_id", userID.String()),
        ))
        continue
    }
}
```

## 3. Context

- `ctx context.Context` is **always the first parameter** on any function that does I/O, DB work, gRPC, SQS, or transitively could.
- Never store `ctx` on a struct. Never pass `context.Background()` outside `main` / worker entry points.
- Honor `ctx` cancellation in any wait — use `select` with `ctx.Done()`, prefer `context.WithDeadline`/`Timeout` over independent timers, and use `ctx.Deadline()` when both deadlines exist.
- In tests use `t.Context()` (see `modern-go.md`).

## 4. Constructors & DI

Plain `New(deps...) *T` with positional required dependencies. **No** functional options, **no** config struct for dependencies.

- Return a pointer (`*T`). The struct type itself is usually unexported when only behavior is consumed via interfaces; otherwise exported.
- Constructors do **no** I/O and never fail — if you need fallible setup, that's a separate `Init` / `Start` method.
- No godoc on constructors. They are obvious.

Example — `tasker/internal/usecase/scheduling/usecase.go`:
```go
func NewUseCase(
    logger log.Logger,
    taskRepo domain.TaskRepo,
    notifier domain.NotificationsService,
    windowDuration time.Duration,
) *UseCase {
    return &UseCase{
        taskRepo:       taskRepo,
        notifier:       notifier,
        windowDuration: windowDuration,
        logger:         logger,
    }
}
```

## 5. Interfaces

- **Domain interfaces (repos, domain services)**: declared in the `domain` package as **public** interfaces. Implementations live elsewhere (e.g. `repo/.../postgres`) and depend on `domain`.
- **Client / SDK-like interfaces (broad APIs)**: producer-side — declared in the package that owns the implementation, exported for consumers.
- **Do not** declare consumer-side mini-interfaces in usecases for domain dependencies — the canonical interface lives in `domain`.
- Single-method interfaces use the `-er` suffix (`Reader`, `TaskCreator`). Multi-method interfaces use a domain noun (`TaskRepo`, `NotificationsService`).
- Place a `//go:generate mockgen` directive next to the interface; mocks land in a sibling `mock/` package.

Example — `tasker/internal/domain/repo.go`:
```go
//go:generate mockgen -source=repo.go -destination=mock/repo_mock.go -typed

type TaskRepo interface {
    CreateTask(ctx context.Context, task Task) error
    GetTaskByID(ctx context.Context, taskID TaskID, userID UserID) (Task, error)
    // ...
}
```

## 6. Receivers

- Pointer receivers (`*T`) for services, usecases, repos, workers, handlers — anything stateful or shared.
- Value receivers for domain value types: IDs, enums, small immutable structs. E.g. `func (id TaskID) String() string`.
- Never mix pointer and value receivers on the same type.

## 7. Domain Types & Validation

- Construct domain entities through `NewXxx(...) (Xxx, error)` constructors that enforce invariants. Direct struct-literal construction is reserved for trivial DTOs (filter/update/pagination structs) and tests.
- Prefer typed primitives over raw `string` / `int` for IDs, statuses, categories: `type TaskID string`, `type TaskStatus string` with `const` enum values. Add a `String()` method when the value is logged or formatted.
- Use `*time.Time`, `*Status`, etc. only when `nil` carries meaning ("no value", "do not update"). Document the meaning on the field.

Example — `tasker/internal/domain/task.go`:
```go
type TaskID string

func (id TaskID) String() string { return string(id) }

type TaskStatus string

const (
    TaskStatusUnplanned TaskStatus = "unplanned"
    TaskStatusPlanned   TaskStatus = "planned"
    TaskStatusCompleted TaskStatus = "completed"
)

type TaskUpdate struct {
    Title       *string
    Description *string
    // nil pointers mean "do not update this field"
}
```

## 8. Database Access

- **Hand-written SQL only**, executed via `pgx`. No ORMs, no query builders.
- Use positional `$1, $2, ...` placeholders. Multi-line queries go in a backtick-quoted string with SQL-style indentation.
- A repo method does **one** SQL statement (or one transaction). Composing multiple operations is the usecase's job.
- Repos accept and return **domain types only**. Row structs / scanned types are private to the repo package; map at the boundary.
- Multi-statement work goes through `libs/go/tx.Provider.RunWithTx(ctx, isolation, op)`. **Never** open a `pgx.Tx` directly in usecase or grpc layers.
- Translate "not found" rows to the domain sentinel (`ErrTaskNotFound`, etc.) — do not leak `pgx.ErrNoRows` to the caller.

Example — `tasker/internal/repo/task/postgres/repo.go`:
```go
func (r *repo) CreateTask(ctx context.Context, task domain.Task) error {
    query := `
        INSERT INTO tasks (task_id, user_id, ...)
        VALUES ($1, $2, ...)
        ON CONFLICT (cluster_id) DO NOTHING
    `
    _, err := r.client.Exec(ctx, query, task.ID, task.UserID, ...)
    if err != nil {
        return errors.WrapFail(err, "exec insert task")
    }
    return nil
}
```

## 9. Concurrency

- **No premature concurrency.** Default to sequential code; introduce goroutines only with a measured reason (I/O fan-out, background worker, scheduled job).
- For parallel work, use `golang.org/x/sync/errgroup`. Never bare `go func() { ... }()` if the function returns an error.
- Every goroutine must have a clear owner: tied to `ctx` cancellation, an `errgroup`, or a `sync.WaitGroup` (use `wg.Go(fn)`, see `modern-go.md`). No fire-and-forget.

## 10. Naming

- Receiver names are 1–2 letters: `s *Service`, `r *repo`, `uc *UseCase`, `t Task`. Not the full type name.
- Sentinel errors: `ErrXxx` (`ErrTaskNotFound`).
- Getters that return a field have no `Get` prefix: `Name()` not `GetName()`. Reserve `Get` for lookups that perform I/O and can fail.
- Single-method interfaces: `-er` suffix. Multi-method: domain noun.
- Mock packages prefixed `mock` (e.g. `mockrepo`), generated into a sibling `mock/` directory next to the interface.

## 11. Comments & godoc

- **Godoc on exported identifiers only.** Comment must start with the identifier name (`// PriorityFromInt converts ...`).
- **No godoc on constructors** (`NewXxx`) or other obvious functions. The signature speaks for itself.
- Inline comments only when the WHY is non-obvious: a hidden invariant, a workaround, surprising behavior, the meaning of `nil` on a pointer field. If removing the comment wouldn't confuse a future reader, don't write it.
- Never describe WHAT the code does — well-named identifiers already do that.
- Never reference the current task / PR / caller in comments.

Good (from `domain/task.go`):
```go
ClusterID *ClusterID // nil for tasks not created from the AI pipeline
```

## 12. Time

- Store, compute, and serialize all `time.Time` in **UTC**. Convert to local timezone only at presentation boundaries (frontend, notification rendering).
- Use `time.Time` everywhere internally. Convert to unix seconds/millis only at proto/JSON serialization where the wire format requires it.
- Inject a clock dependency (a `Clock` interface or `func() time.Time`) on services that read "now". **Do not** call `time.Now()` directly inside a service — tests need to control time.
- Honor `ctx` cancellation in any wait. Prefer `context.WithDeadline`/`Timeout` over independent timers; in `time.After` loops, `select` on `ctx.Done()`.

## 13. Returning slices and maps

Return `nil` for "no results", not an allocated empty container. Callers must use `len()` and `range`, both of which work on `nil`.

## 14. Generics

Use generics only when they remove **real, repeated** duplication — typically in shared libs (`libs/go/slices`, set helpers, `must` wrappers). Do **not** introduce generics in usecase, service, repo, or grpc layers as a forward-looking abstraction. Concrete types in business code are easier to read and grep for.

## 15. Panics

Forbidden in any production code path that handles a request, message, or scheduled job. Allowed only:
- At package init / process startup for unrecoverable misconfiguration (`regexp.MustCompile`, `must.X` helpers).
- In `main` setup before the server starts serving.

For unrecoverable runtime errors during startup, use `log.Fatal`. Never `panic` to signal a request-level failure.

## 16. File organization

Group files by responsibility within a package: `service.go`, `repo.go`, `model.go`, `scenarios.go`, etc. Multiple types in a file is fine when they share a concern. Split a file out only when it has clearly become two responsibilities — not just because it grew long.

## 17. Imports

Let `goimports` / `gofumpt` decide. Do not hand-curate import grouping.

## 18. Generated code

- Protobuf / gRPC output: `gen/` at the service root. Generated by `make generate` (which runs `go generate ./...` → `buf generate`). Never edit generated files; never invoke `protoc` or `buf` directly.
- Mocks: in a `mock/` subpackage next to the interface, with a `//go:generate mockgen` directive on the source file. Package name prefixed `mock` (e.g. `mockrepo`).

---

## Quick checklist before submitting Go code

- [ ] Errors wrapped via `errors.WrapFail` / `errors.WrapFailf` with `errors.Token` for context.
- [ ] Each error logged at exactly one layer.
- [ ] `ctx` is the first parameter; not stored on structs.
- [ ] Constructor is a plain `New(deps...) *T` with no godoc.
- [ ] Domain interfaces live in `domain/`; mocks in sibling `mock/`.
- [ ] Pointer receivers on services; value receivers on domain value types.
- [ ] No raw `time.Now()` in a service — clock injected.
- [ ] No `pgx.Tx` outside `tx.Provider.RunWithTx`.
- [ ] No bare `go func()` returning errors — use `errgroup` or `wg.Go`.
- [ ] No comments describing WHAT the code does.
