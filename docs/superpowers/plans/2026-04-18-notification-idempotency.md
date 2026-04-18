# Notification Idempotency Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the wall-clock window dedup in tasker with a producer-generated idempotency key enforced by notificator via a unique DB index.

**Architecture:** Tasker computes a 5-minute-bucketed SHA-256 key from `(user_id, now, type, title)` at the SQS publish boundary and attaches it to `pushpb.Notification`. Notificator persists first with `INSERT ... ON CONFLICT DO NOTHING`; only new rows trigger the push. The `scenarios.go` narrow window is reverted to the original 2-minute window once the idempotency guard is in place.

**Tech Stack:** Go 1.x, PostgreSQL 18, goose migrations, protobuf/gRPC, testcontainers-go, testify, AWS SQS.

**Spec:** `docs/superpowers/specs/2026-04-18-notification-idempotency-design.md`

---

## File Map

**Notificator (consumer):**
- Create: `backend/notificator/migrations/00004_add_notifications_idempotency_key.sql` — adds `idempotency_key` column + partial unique index.
- Modify: `backend/notificator/api/push/push.proto` — adds `idempotency_key` field (number 8).
- Regenerate: `backend/notificator/gen/api/push/*.go` — via `make notificator/generate`.
- Modify: `backend/notificator/internal/domain/notification/notification.go` — adds `IdempotencyKey` field.
- Modify: `backend/notificator/internal/domain/notification/repo.go` — adds `CreateIfAbsent`.
- Modify: `backend/notificator/internal/repo/notification/postgres/entity.go` — entity + mappers.
- Modify: `backend/notificator/internal/repo/notification/postgres/repo.go` — implements `CreateIfAbsent`; keep `Create` as-is.
- Create: `backend/notificator/internal/repo/notification/postgres/setup_test.go` — testcontainers bootstrap.
- Create: `backend/notificator/internal/repo/notification/postgres/repo_test.go` — integration test for the new method.
- Modify: `backend/notificator/internal/worker/matcher.go` — persist first, push only on insert.

**Tasker (producer):**
- Create: `backend/tasker/internal/services/notifications/idempotency.go` — `IdempotencyKey` helper.
- Create: `backend/tasker/internal/services/notifications/idempotency_test.go` — unit tests.
- Modify: `backend/tasker/internal/services/notifications/service.go` — attaches key to `pushpb.Notification`.
- Modify: `backend/tasker/internal/services/notifications/scenarios.go` — revert window to 2 minutes.

---

## Task 1: Notificator migration — add `idempotency_key`

**Files:**
- Create: `backend/notificator/migrations/00004_add_notifications_idempotency_key.sql`

- [ ] **Step 1: Write the migration**

Create `backend/notificator/migrations/00004_add_notifications_idempotency_key.sql` with:

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE notifications ADD COLUMN idempotency_key text;

CREATE UNIQUE INDEX notifications_idempotency_key_uniq
    ON notifications (idempotency_key)
    WHERE idempotency_key IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS notifications_idempotency_key_uniq;
ALTER TABLE notifications DROP COLUMN IF EXISTS idempotency_key;
-- +goose StatementEnd
```

- [ ] **Step 2: Verify migration applies cleanly**

Start local Postgres via `make services/deploy` (from `backend/`) if not running, then from `backend/`:

```bash
make notificator/migrate/up
```

Expected: goose logs `OK 00004_add_notifications_idempotency_key.sql`.

Sanity-check the schema:

```bash
psql 'postgres://user:pass@localhost:5432/notificator?sslmode=disable' \
  -c "\d notifications"
```

Expected: `idempotency_key | text` column and `"notifications_idempotency_key_uniq" UNIQUE, btree (idempotency_key) WHERE idempotency_key IS NOT NULL`.

- [ ] **Step 3: Roll back once to confirm Down works, then re-apply**

```bash
make notificator/migrate/down-one
make notificator/migrate/up
```

Expected: both commands exit 0.

- [ ] **Step 4: Commit**

```bash
git add backend/notificator/migrations/00004_add_notifications_idempotency_key.sql
git commit -m "feat(notificator): add idempotency_key to notifications"
```

---

## Task 2: Add `idempotency_key` to the proto contract

**Files:**
- Modify: `backend/notificator/api/push/push.proto`
- Regenerate: `backend/notificator/gen/api/push/*` and `backend/tasker` imports of `pushpb.Notification`

- [ ] **Step 1: Edit the proto**

Open `backend/notificator/api/push/push.proto` and add field `8`:

```proto
syntax = "proto3";

package push;

option go_package = "github.com/Doremi203/personage/backend/notificator/api/push";

message Notification {
  string recipient_id = 1;
  string title = 2;
  string body = 3;
  string icon = 4;
  string url = 5;
  // type is the notification category (e.g. "upcoming_event", "schedule_change").
  string type = 6;
  // detailed_text is the verbose notification text that gets persisted
  // to the database for the frontend notifications API.
  string detailed_text = 7;
  // idempotency_key dedupes delivery when the producer retries or fires
  // twice. Optional; when empty, notificator treats each message as unique.
  string idempotency_key = 8;
}
```

- [ ] **Step 2: Regenerate stubs**

From `backend/`:

```bash
make notificator/generate
```

Expected: no errors; `backend/notificator/gen/api/push/push.pb.go` now contains `IdempotencyKey string` and `GetIdempotencyKey() string`.

Verify:

```bash
grep -n "IdempotencyKey" backend/notificator/gen/api/push/push.pb.go
```

Expected: two lines — the struct field and the getter.

- [ ] **Step 3: Verify both services still compile**

From `backend/`:

```bash
go build ./notificator/... ./tasker/...
```

Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add backend/notificator/api/push/push.proto backend/notificator/gen/api/push/
git commit -m "feat(push): add idempotency_key to Notification proto"
```

---

## Task 3: Extend notificator domain model

**Files:**
- Modify: `backend/notificator/internal/domain/notification/notification.go`
- Modify: `backend/notificator/internal/domain/notification/repo.go`

- [ ] **Step 1: Add `IdempotencyKey` to the domain type**

Replace the contents of `backend/notificator/internal/domain/notification/notification.go` with:

```go
package notification

import (
	"time"

	"github.com/google/uuid"
)

// Notification represents a sent notification stored in the database.
type Notification struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Title  string
	Type   string
	Text   string
	SentAt time.Time
	// IdempotencyKey dedupes writes. Empty means "no dedup, always insert".
	IdempotencyKey string
}
```

- [ ] **Step 2: Extend the Repo interface**

Replace the contents of `backend/notificator/internal/domain/notification/repo.go` with:

```go
package notification

import (
	"context"

	"github.com/google/uuid"
)

// Repo defines the persistence operations for notifications and notification settings.
type Repo interface {
	// Create inserts a new notification into the database.
	// Prefer CreateIfAbsent when the caller has an idempotency key.
	Create(ctx context.Context, n Notification) error

	// CreateIfAbsent inserts a new notification. When n.IdempotencyKey is
	// non-empty and a row with the same key already exists, returns
	// inserted=false and a nil error. An empty IdempotencyKey behaves like
	// Create (always inserts, inserted=true on success).
	CreateIfAbsent(ctx context.Context, n Notification) (inserted bool, err error)

	// ListByUserID returns a paginated list of notifications for the given user,
	// ordered by sent_at descending. offset is zero-based.
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Notification, error)

	// ToggleSetting flips the enabled flag for the given user and notification type.
	// If no row exists yet, it inserts one with enabled=false (toggled from the default true).
	// Returns the new state after toggling.
	ToggleSetting(ctx context.Context, userID uuid.UUID, notificationType string) (Setting, error)

	// GetSettings returns all notification settings rows for the given user.
	GetSettings(ctx context.Context, userID uuid.UUID) ([]Setting, error)
}
```

- [ ] **Step 3: Verify the package still compiles**

From `backend/`:

```bash
go build ./notificator/internal/domain/notification/...
```

Expected: exit 0. The build of the whole service will fail until Task 4 lands the repo implementation — that's expected and will be fixed in Task 4.

- [ ] **Step 4: Commit**

```bash
git add backend/notificator/internal/domain/notification/
git commit -m "feat(notificator): add IdempotencyKey to notification domain"
```

---

## Task 4: Implement postgres `CreateIfAbsent` — failing integration test first

**Files:**
- Create: `backend/notificator/internal/repo/notification/postgres/setup_test.go`
- Create: `backend/notificator/internal/repo/notification/postgres/repo_test.go`

- [ ] **Step 1: Add testcontainers bootstrap**

Create `backend/notificator/internal/repo/notification/postgres/setup_test.go`:

```go
package notificationpostgres

import (
	"testing"

	"github.com/Doremi203/personage/backend/libs/go/postgres"
)

var tester postgres.Tester

func TestMain(m *testing.M) {
	postgres.SetupTests(m, &tester, "notificator")
}
```

- [ ] **Step 2: Write the failing test for `CreateIfAbsent`**

Create `backend/notificator/internal/repo/notification/postgres/repo_test.go`:

```go
package notificationpostgres

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_repo_CreateIfAbsent(t *testing.T) {
	userA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userB := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tester.Run(t, "first insert with key succeeds", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			inserted, err := r.CreateIfAbsent(ctx, notification.Notification{
				UserID:         userA,
				Title:          "hello",
				Type:           "upcoming_event",
				Text:           "body",
				IdempotencyKey: "key-1",
			})
			require.NoError(t, err)
			assert.True(t, inserted)

			got, err := r.ListByUserID(ctx, userA, 10, 0)
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "hello", got[0].Title)
		},
	)

	tester.Run(t, "second insert with same key is skipped", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			_, err := r.CreateIfAbsent(ctx, notification.Notification{
				UserID:         userA,
				Title:          "first",
				Type:           "upcoming_event",
				Text:           "first-body",
				IdempotencyKey: "dup-key",
			})
			require.NoError(t, err)

			inserted, err := r.CreateIfAbsent(ctx, notification.Notification{
				UserID:         userA,
				Title:          "second",
				Type:           "upcoming_event",
				Text:           "second-body",
				IdempotencyKey: "dup-key",
			})
			require.NoError(t, err)
			assert.False(t, inserted)

			got, err := r.ListByUserID(ctx, userA, 10, 0)
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "first", got[0].Title)
		},
	)

	tester.Run(t, "empty key always inserts", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			for range 2 {
				inserted, err := r.CreateIfAbsent(ctx, notification.Notification{
					UserID:         userB,
					Title:          "no-key",
					Type:           "upcoming_event",
					Text:           "body",
					IdempotencyKey: "",
				})
				require.NoError(t, err)
				assert.True(t, inserted)
			}

			got, err := r.ListByUserID(ctx, userB, 10, 0)
			require.NoError(t, err)
			assert.Len(t, got, 2)
		},
	)

	tester.Run(t, "different keys coexist", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			for i, k := range []string{"a", "b"} {
				inserted, err := r.CreateIfAbsent(ctx, notification.Notification{
					UserID:         userA,
					Title:          "t" + k,
					Type:           "upcoming_event",
					Text:           "body",
					IdempotencyKey: k,
				})
				require.NoError(t, err, "iteration %d", i)
				assert.True(t, inserted, "iteration %d", i)
			}

			got, err := r.ListByUserID(ctx, userA, 10, 0)
			require.NoError(t, err)
			assert.Len(t, got, 2)
		},
	)
}
```

- [ ] **Step 3: Run the test and confirm it fails**

From `backend/`:

```bash
go test ./notificator/internal/repo/notification/postgres/... -run Test_repo_CreateIfAbsent -race -count=1
```

Expected: compile error — `r.CreateIfAbsent undefined (type *repo has no field or method CreateIfAbsent)`. That failure proves the test actually exercises the method we're about to add.

- [ ] **Step 4: Commit the failing test**

```bash
git add backend/notificator/internal/repo/notification/postgres/setup_test.go \
        backend/notificator/internal/repo/notification/postgres/repo_test.go
git commit -m "test(notificator): add failing CreateIfAbsent integration tests"
```

---

## Task 5: Implement postgres `CreateIfAbsent` — make the tests pass

**Files:**
- Modify: `backend/notificator/internal/repo/notification/postgres/entity.go`
- Modify: `backend/notificator/internal/repo/notification/postgres/repo.go`

- [ ] **Step 1: Extend the entity**

Replace the contents of `backend/notificator/internal/repo/notification/postgres/entity.go` with:

```go
package notificationpostgres

import (
	"database/sql"
	"time"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
)

type notificationEntity struct {
	ID             uuid.UUID      `db:"id"`
	RecipientID    uuid.UUID      `db:"recipient_id"`
	Title          string         `db:"title"`
	Type           string         `db:"type"`
	Text           string         `db:"text"`
	SentAt         time.Time      `db:"sent_at"`
	IdempotencyKey sql.NullString `db:"idempotency_key"`
}

func entityToDomain(e notificationEntity) notification.Notification {
	return notification.Notification{
		ID:             e.ID,
		UserID:         e.RecipientID,
		Title:          e.Title,
		Type:           e.Type,
		Text:           e.Text,
		SentAt:         e.SentAt,
		IdempotencyKey: e.IdempotencyKey.String,
	}
}

func domainToEntity(n notification.Notification) notificationEntity {
	return notificationEntity{
		RecipientID:    n.UserID,
		Title:          n.Title,
		Type:           n.Type,
		Text:           n.Text,
		IdempotencyKey: sql.NullString{String: n.IdempotencyKey, Valid: n.IdempotencyKey != ""},
	}
}

type settingEntity struct {
	RecipientID uuid.UUID `db:"recipient_id"`
	Type        string    `db:"type"`
	Enabled     bool      `db:"enabled"`
}

func settingEntityToDomain(e settingEntity) notification.Setting {
	return notification.Setting{
		UserID:  e.RecipientID,
		Type:    notification.SettingType(e.Type),
		Enabled: e.Enabled,
	}
}
```

- [ ] **Step 2: Add `CreateIfAbsent` and update `ListByUserID` select list**

Edit `backend/notificator/internal/repo/notification/postgres/repo.go`. Replace the `Create` method and add a new `CreateIfAbsent` method, and update the `ListByUserID` query to include the new column. Keep the rest of the file unchanged.

Replace the `Create` method with:

```go
func (r *repo) Create(ctx context.Context, n notification.Notification) error {
	const query = `
		INSERT INTO notifications (recipient_id, title, type, text, idempotency_key)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
	`

	e := domainToEntity(n)

	_, err := r.db.Exec(ctx, query, e.RecipientID, e.Title, e.Type, e.Text, n.IdempotencyKey)
	if err != nil {
		return errors.WrapFail(err, "exec insert notification query")
	}

	return nil
}

func (r *repo) CreateIfAbsent(ctx context.Context, n notification.Notification) (bool, error) {
	if n.IdempotencyKey == "" {
		if err := r.Create(ctx, n); err != nil {
			return false, err
		}
		return true, nil
	}

	const query = `
		INSERT INTO notifications (recipient_id, title, type, text, idempotency_key)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id
	`

	e := domainToEntity(n)

	rows, err := r.db.Query(ctx, query, e.RecipientID, e.Title, e.Type, e.Text, n.IdempotencyKey)
	if err != nil {
		return false, errors.WrapFail(err, "exec insert-if-absent notification query")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return false, errors.WrapFail(err, "iterate insert-if-absent rows")
		}
		return false, nil
	}

	return true, nil
}
```

Update `ListByUserID` so the row scan still works with the struct tag on the new `IdempotencyKey` field. Change the `SELECT` list in the existing query from

```sql
SELECT id, recipient_id, title, type, text, sent_at
```

to

```sql
SELECT id, recipient_id, title, type, text, sent_at, idempotency_key
```

- [ ] **Step 3: Run the integration tests**

From `backend/`:

```bash
go test ./notificator/internal/repo/notification/postgres/... -run Test_repo_CreateIfAbsent -race -count=1
```

Expected: `PASS` on all four sub-tests.

- [ ] **Step 4: Run the full notificator test suite to catch regressions**

```bash
go test ./notificator/... -race -count=1
```

Expected: `ok` everywhere.

- [ ] **Step 5: Commit**

```bash
git add backend/notificator/internal/repo/notification/postgres/entity.go \
        backend/notificator/internal/repo/notification/postgres/repo.go
git commit -m "feat(notificator): implement CreateIfAbsent for notifications"
```

---

## Task 6: Wire the matcher to persist-then-push

**Files:**
- Modify: `backend/notificator/internal/worker/matcher.go`

- [ ] **Step 1: Reorder the handler**

Replace the body of `Process` in `backend/notificator/internal/worker/matcher.go` (keep imports and struct definitions intact). The new method:

```go
func (p *notificationHandler) Process(
	ctx context.Context,
	data *pushpb.Notification,
) error {
	recipientUUID, err := uuid.Parse(data.GetRecipientId())
	if err != nil {
		return errors.WrapFailf(
			err,
			"parse recipient id %v",
			errors.Token("recipient_id", data.GetRecipientId()),
		)
	}

	pushRecipientID := push.RecipientID(recipientUUID)

	pushRecipient, err := p.subscriptionUseCase.GetRecipient(ctx, pushRecipientID)
	if err != nil {
		return errors.WrapFailf(
			err,
			"get push recipient with id %v",
			errors.Token("id", pushRecipientID),
		)
	}
	if len(pushRecipient.Subscriptions) == 0 {
		p.logger.Infof(
			"no subscriptions for recipient %v, skipping push",
			errors.Token("id", pushRecipientID),
		)
		return nil
	}

	inserted, err := p.notificationRepo.CreateIfAbsent(ctx, notification.Notification{
		UserID:         recipientUUID,
		Title:          data.GetTitle(),
		Type:           data.GetType(),
		Text:           data.GetDetailedText(),
		IdempotencyKey: data.GetIdempotencyKey(),
	})
	if err != nil {
		return errors.WrapFailf(
			err,
			"persist notification for recipient %v",
			errors.Token("id", pushRecipientID),
		)
	}
	if !inserted {
		p.logger.Infof(
			"duplicate notification skipped for recipient %v key %v",
			errors.Token("id", pushRecipientID),
			errors.Token("idempotency_key", data.GetIdempotencyKey()),
		)
		return nil
	}

	err = p.senderUseCase.Send(ctx, pushRecipient, push.Push{
		Title: data.GetTitle(),
		Body:  data.GetBody(),
		Icon:  data.GetIcon(),
		Url:   data.GetUrl(),
	})
	if err != nil {
		return errors.WrapFailf(
			err,
			"send push to recipient with id %v",
			errors.Token("id", pushRecipientID),
		)
	}

	return nil
}
```

- [ ] **Step 2: Verify the notificator builds**

From `backend/`:

```bash
go build ./notificator/...
```

Expected: exit 0.

- [ ] **Step 3: Run the full notificator test suite**

```bash
go test ./notificator/... -race -count=1
```

Expected: `ok` everywhere.

- [ ] **Step 4: Commit**

```bash
git add backend/notificator/internal/worker/matcher.go
git commit -m "feat(notificator): dedupe notifications via idempotency key"
```

---

## Task 7: Tasker `IdempotencyKey` helper — failing unit test first

**Files:**
- Create: `backend/tasker/internal/services/notifications/idempotency_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/tasker/internal/services/notifications/idempotency_test.go`:

```go
package notifications

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIdempotencyKey(t *testing.T) {
	userA := "11111111-1111-1111-1111-111111111111"
	userB := "22222222-2222-2222-2222-222222222222"

	base := time.Date(2026, 4, 18, 10, 2, 30, 0, time.UTC) // bucket 10:00
	sameBucket := base.Add(2 * time.Minute)                // 10:04:30 — same 5m bucket
	nextBucket := base.Add(5 * time.Minute)                // 10:07:30 — next bucket

	t.Run("deterministic for identical inputs", func(t *testing.T) {
		a := IdempotencyKey(userA, base, "upcoming_event", "hello")
		b := IdempotencyKey(userA, base, "upcoming_event", "hello")
		assert.Equal(t, a, b)
	})

	t.Run("stable within the same 5-minute bucket", func(t *testing.T) {
		a := IdempotencyKey(userA, base, "upcoming_event", "hello")
		b := IdempotencyKey(userA, sameBucket, "upcoming_event", "hello")
		assert.Equal(t, a, b)
	})

	t.Run("changes across buckets", func(t *testing.T) {
		a := IdempotencyKey(userA, base, "upcoming_event", "hello")
		b := IdempotencyKey(userA, nextBucket, "upcoming_event", "hello")
		assert.NotEqual(t, a, b)
	})

	t.Run("different users differ", func(t *testing.T) {
		a := IdempotencyKey(userA, base, "upcoming_event", "hello")
		b := IdempotencyKey(userB, base, "upcoming_event", "hello")
		assert.NotEqual(t, a, b)
	})

	t.Run("different types differ", func(t *testing.T) {
		a := IdempotencyKey(userA, base, "upcoming_event", "hello")
		b := IdempotencyKey(userA, base, "schedule_change", "hello")
		assert.NotEqual(t, a, b)
	})

	t.Run("different titles differ", func(t *testing.T) {
		a := IdempotencyKey(userA, base, "upcoming_event", "hello")
		b := IdempotencyKey(userA, base, "upcoming_event", "world")
		assert.NotEqual(t, a, b)
	})

	t.Run("timezone-independent (UTC bucketing)", func(t *testing.T) {
		moscow := time.FixedZone("MSK", 3*3600)
		utc := IdempotencyKey(userA, base, "upcoming_event", "hello")
		local := IdempotencyKey(userA, base.In(moscow), "upcoming_event", "hello")
		assert.Equal(t, utc, local)
	})
}
```

- [ ] **Step 2: Run the test and confirm it fails**

From `backend/`:

```bash
go test ./tasker/internal/services/notifications/... -run TestIdempotencyKey -race -count=1
```

Expected: compile error — `undefined: IdempotencyKey`.

- [ ] **Step 3: Commit the failing test**

```bash
git add backend/tasker/internal/services/notifications/idempotency_test.go
git commit -m "test(tasker): add failing IdempotencyKey tests"
```

---

## Task 8: Tasker `IdempotencyKey` helper — implementation

**Files:**
- Create: `backend/tasker/internal/services/notifications/idempotency.go`

- [ ] **Step 1: Implement the helper**

Create `backend/tasker/internal/services/notifications/idempotency.go`:

```go
package notifications

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// idempotencyBucket is the time granularity used when hashing. All calls
// landing in the same UTC-aligned 5-minute bucket produce identical keys for
// the same (user, type, title) triple.
const idempotencyBucket = 5 * time.Minute

// IdempotencyKey returns a deterministic idempotency key for a notification
// destined to userID, emitted around now, carrying the given type and title.
// Duplicate notifications fired within the same 5-minute UTC bucket produce
// the same key and are deduplicated by the notificator.
func IdempotencyKey(userID string, now time.Time, typ, title string) string {
	bucket := now.UTC().Truncate(idempotencyBucket).Unix()
	h := sha256.New()
	fmt.Fprintf(h, "%s|%d|%s|%s", userID, bucket, typ, title)
	return hex.EncodeToString(h.Sum(nil))
}
```

- [ ] **Step 2: Run the test**

From `backend/`:

```bash
go test ./tasker/internal/services/notifications/... -run TestIdempotencyKey -race -count=1
```

Expected: `PASS` on every sub-test.

- [ ] **Step 3: Commit**

```bash
git add backend/tasker/internal/services/notifications/idempotency.go
git commit -m "feat(tasker): add IdempotencyKey helper for notifications"
```

---

## Task 9: Attach the idempotency key at the SQS boundary

**Files:**
- Modify: `backend/tasker/internal/services/notifications/service.go`

- [ ] **Step 1: Wire the key into the proto message**

`domain.UserID` is a `type UserID string` with a `String()` method (see `backend/tasker/internal/domain/event.go`), so we pass it to the helper as a string.

Replace the contents of `backend/tasker/internal/services/notifications/service.go` with:

```go
package notifications

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/sqs"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

// NewNotificatorPushService creates a new push notification service that implements domain.NotificationsService.
func NewNotificatorPushService(
	client sqs.ClientWriter[*pushpb.Notification],
) domain.NotificationsService {
	return &notificatorPushService{
		client: client,
		now:    time.Now,
	}
}

type notificatorPushService struct {
	client sqs.ClientWriter[*pushpb.Notification]
	now    func() time.Time
}

func (s *notificatorPushService) Send(
	ctx context.Context,
	notification domain.Notification,
) error {
	userID := notification.UserID.String()
	return s.client.SendMessage(ctx, &pushpb.Notification{
		RecipientId:    userID,
		Title:          notification.Title,
		Body:           notification.Body,
		Icon:           "/icon-72x72.png",
		Url:            "/",
		Type:           notification.Type,
		IdempotencyKey: IdempotencyKey(userID, s.now(), notification.Type, notification.Title),
	}, sqs.WithGroupID("tasker"))
}
```

- [ ] **Step 2: Build tasker**

From `backend/`:

```bash
go build ./tasker/...
```

Expected: exit 0.

- [ ] **Step 3: Run tasker tests**

```bash
go test ./tasker/... -race -count=1
```

Expected: `ok` everywhere.

- [ ] **Step 4: Commit**

```bash
git add backend/tasker/internal/services/notifications/service.go
git commit -m "feat(tasker): attach idempotency key to outbound notifications"
```

---

## Task 10: Revert the `scenarios.go` window to 2 minutes

**Files:**
- Modify: `backend/tasker/internal/services/notifications/scenarios.go`

- [ ] **Step 1: Restore the original window**

In `backend/tasker/internal/services/notifications/scenarios.go`, locate the block inside `NotifyUpcomingEvents`:

```go
sinceNotificationTime := now.Sub(notificationTime)
if sinceNotificationTime >= 0 && sinceNotificationTime < time.Minute {
```

Replace with the original symmetric window:

```go
if now.After(notificationTime.Add(-time.Minute)) && now.Before(notificationTime.Add(time.Minute)) {
```

This reintroduces the 2-minute window. Duplicate firings are absorbed by the idempotency key.

- [ ] **Step 2: Run the scenarios tests**

From `backend/`:

```bash
go test ./tasker/internal/services/notifications/... -race -count=1
```

Expected: `ok`.

- [ ] **Step 3: Run the whole backend test suite as a final check**

```bash
go test ./... -race -count=1
```

Expected: `ok` everywhere. On a fresh machine testcontainers pulls `postgres:18-alpine` on first run; allow a few minutes.

- [ ] **Step 4: Lint**

```bash
make lint
```

Expected: no findings.

- [ ] **Step 5: Commit**

```bash
git add backend/tasker/internal/services/notifications/scenarios.go
git commit -m "feat(tasker): restore 2-minute upcoming-event window"
```

---

## Done criteria

- `make lint` clean.
- `go test ./... -race -count=1` passes under `backend/`.
- `backend/notificator/migrations/00004_add_notifications_idempotency_key.sql` applied in all environments before deploying the updated tasker binary (see spec §Rollout).
- Producing two identical `pushpb.Notification` messages within a 5-minute window results in exactly one row in `notifications` and one `pushSender.Send` invocation.
