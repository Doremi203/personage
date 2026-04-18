# Notification Rate Limiting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-user rate limiting to `schedule_change` notifications (max 2/hour, 10/day), delaying over-limit notifications with retries every 15 min and dropping after 2 hours; `upcoming_event` always sends immediately.

**Architecture:** Extend the `notifications` table with a `status` column (`sent`/`pending`/`dropped`) and scheduling columns so pending notifications live alongside sent history. A `RateLimiter` service counts sent notifications per user/type/window; a `Retrier` background goroutine polls every minute to retry or expire pending records.

**Tech Stack:** Go, PostgreSQL (pgx v5), gomock (go.uber.org/mock), goose migrations. Module: `github.com/Doremi203/personage/backend`.

---

## File Map

| Action | Path |
|--------|------|
| Create | `notificator/migrations/00004_add_rate_limiting_columns.sql` |
| Modify | `notificator/internal/domain/notification/notification.go` |
| Modify | `notificator/internal/domain/notification/repo.go` |
| Generate | `notificator/internal/domain/notification/mock/repo_mock.go` |
| Modify | `notificator/internal/repo/notification/postgres/entity.go` |
| Modify | `notificator/internal/repo/notification/postgres/repo.go` |
| Create | `notificator/internal/services/ratelimit/ratelimiter.go` |
| Create | `notificator/internal/services/ratelimit/ratelimiter_test.go` |
| Create | `notificator/internal/services/retrier/retrier.go` |
| Create | `notificator/internal/services/retrier/retrier_test.go` |
| Modify | `notificator/internal/worker/matcher.go` |
| Modify | `notificator/internal/grpc/notifications.go` |
| Modify | `notificator/cmd/app/main.go` |

---

### Task 1: DB Migration

**Files:**
- Create: `notificator/migrations/00004_add_rate_limiting_columns.sql`

- [ ] **Step 1: Create migration file**

```sql
-- +goose Up
-- +goose StatementBegin
alter table notifications
    alter column sent_at drop not null,
    add column status       text        not null default 'sent',
    add column retry_after  timestamptz,
    add column expires_at   timestamptz,
    add column push_payload text;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table notifications
    alter column sent_at set not null,
    drop column status,
    drop column retry_after,
    drop column expires_at,
    drop column push_payload;
-- +goose StatementEnd
```

- [ ] **Step 2: Run migration**

```bash
cd backend && make notificator/migrate/up
```

Expected: `OK    00004_add_rate_limiting_columns.sql`

- [ ] **Step 3: Verify rollback works**

```bash
cd backend && make notificator/migrate/down-one
cd backend && make notificator/migrate/up
```

Expected: both complete without error.

- [ ] **Step 4: Commit**

```bash
git add notificator/migrations/00004_add_rate_limiting_columns.sql
git commit -m "feat(notificator): add rate limiting columns to notifications table"
```

---

### Task 2: Domain Model Updates

**Files:**
- Modify: `notificator/internal/domain/notification/notification.go`
- Modify: `notificator/internal/domain/notification/repo.go`

- [ ] **Step 1: Update notification.go**

Replace the entire file with:

```go
package notification

import (
	"time"

	"github.com/google/uuid"
)

type NotificationStatus string

const (
	StatusSent    NotificationStatus = "sent"
	StatusPending NotificationStatus = "pending"
	StatusDropped NotificationStatus = "dropped"
)

type PushPayload struct {
	Body string `json:"body"`
	Icon string `json:"icon"`
	URL  string `json:"url"`
}

type Notification struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Title       string
	Type        string
	Text        string
	Status      NotificationStatus
	SentAt      *time.Time
	RetryAfter  *time.Time
	ExpiresAt   *time.Time
	PushPayload *PushPayload
}
```

- [ ] **Step 2: Update repo.go**

Replace the entire file with:

```go
package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
)

//go:generate mockgen -source=repo.go -destination=mock/repo_mock.go -typed

type Repo interface {
	Create(ctx context.Context, n Notification) error
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Notification, error)
	ToggleSetting(ctx context.Context, userID uuid.UUID, notificationType string) (Setting, error)
	GetSettings(ctx context.Context, userID uuid.UUID) ([]Setting, error)

	ListPending(ctx context.Context) ([]Notification, error)
	Drop(ctx context.Context, id uuid.UUID) error
	MarkSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error
	UpdateRetryAfter(ctx context.Context, id uuid.UUID, retryAfter time.Time) error
	CountSentSince(ctx context.Context, userID uuid.UUID, typ SettingType, since time.Time) (int, error)
}
```

- [ ] **Step 3: Commit**

```bash
git add notificator/internal/domain/notification/notification.go notificator/internal/domain/notification/repo.go
git commit -m "feat(notificator): extend notification domain with status and pending fields"
```

---

### Task 3: Generate Mock

**Files:**
- Generate: `notificator/internal/domain/notification/mock/repo_mock.go`

- [ ] **Step 1: Run go generate**

```bash
cd backend && go generate ./notificator/internal/domain/notification/...
```

Expected: creates `notificator/internal/domain/notification/mock/repo_mock.go` with no errors.

- [ ] **Step 2: Commit**

```bash
git add notificator/internal/domain/notification/mock/
git commit -m "feat(notificator): generate mock for notification.Repo"
```

---

### Task 4: Postgres Repo — entity.go

**Files:**
- Modify: `notificator/internal/repo/notification/postgres/entity.go`

- [ ] **Step 1: Replace entity.go**

```go
package notificationpostgres

import (
	"encoding/json"
	"time"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
)

type notificationEntity struct {
	ID          uuid.UUID  `db:"id"`
	RecipientID uuid.UUID  `db:"recipient_id"`
	Title       string     `db:"title"`
	Type        string     `db:"type"`
	Text        string     `db:"text"`
	Status      string     `db:"status"`
	SentAt      *time.Time `db:"sent_at"`
	RetryAfter  *time.Time `db:"retry_after"`
	ExpiresAt   *time.Time `db:"expires_at"`
	PushPayload *string    `db:"push_payload"`
}

func entityToDomain(e notificationEntity) notification.Notification {
	n := notification.Notification{
		ID:         e.ID,
		UserID:     e.RecipientID,
		Title:      e.Title,
		Type:       e.Type,
		Text:       e.Text,
		Status:     notification.NotificationStatus(e.Status),
		SentAt:     e.SentAt,
		RetryAfter: e.RetryAfter,
		ExpiresAt:  e.ExpiresAt,
	}
	if e.PushPayload != nil {
		var p notification.PushPayload
		if err := json.Unmarshal([]byte(*e.PushPayload), &p); err == nil {
			n.PushPayload = &p
		}
	}
	return n
}

func domainToEntity(n notification.Notification) notificationEntity {
	e := notificationEntity{
		RecipientID: n.UserID,
		Title:       n.Title,
		Type:        n.Type,
		Text:        n.Text,
		Status:      string(n.Status),
		SentAt:      n.SentAt,
		RetryAfter:  n.RetryAfter,
		ExpiresAt:   n.ExpiresAt,
	}
	if n.PushPayload != nil {
		b, err := json.Marshal(n.PushPayload)
		if err == nil {
			s := string(b)
			e.PushPayload = &s
		}
	}
	return e
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

- [ ] **Step 2: Verify it compiles**

```bash
cd backend && go build ./notificator/internal/repo/notification/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add notificator/internal/repo/notification/postgres/entity.go
git commit -m "feat(notificator): update notification entity for rate limiting columns"
```

---

### Task 5: Postgres Repo — repo.go

**Files:**
- Modify: `notificator/internal/repo/notification/postgres/repo.go`

- [ ] **Step 1: Replace repo.go**

```go
package notificationpostgres

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/libs/go/slices"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func NewRepo(db postgres.Client) *repo {
	return &repo{db: db}
}

type repo struct {
	db postgres.Client
}

func (r *repo) Create(ctx context.Context, n notification.Notification) error {
	const query = `
		INSERT INTO notifications (recipient_id, title, type, text, status, sent_at, retry_after, expires_at, push_payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	e := domainToEntity(n)
	_, err := r.db.Exec(ctx, query,
		e.RecipientID, e.Title, e.Type, e.Text,
		e.Status, e.SentAt, e.RetryAfter, e.ExpiresAt, e.PushPayload,
	)
	if err != nil {
		return errors.WrapFail(err, "exec insert notification query")
	}
	return nil
}

func (r *repo) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]notification.Notification, error) {
	const query = `
		SELECT id, recipient_id, title, type, text, status, sent_at, retry_after, expires_at, push_payload
		FROM notifications
		WHERE recipient_id = $1 AND status = 'sent'
		ORDER BY sent_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, errors.WrapFail(err, "exec select notifications query")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[notificationEntity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect notification rows")
	}

	return slices.Map(entities, entityToDomain), nil
}

func (r *repo) GetSettings(ctx context.Context, userID uuid.UUID) ([]notification.Setting, error) {
	const query = `
		SELECT recipient_id, type, enabled
		FROM notification_settings
		WHERE recipient_id = $1
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, errors.WrapFail(err, "exec select notification settings query")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[settingEntity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect notification settings rows")
	}

	stored := make(map[notification.SettingType]notification.Setting, len(entities))
	for _, e := range entities {
		s := settingEntityToDomain(e)
		stored[s.Type] = s
	}

	result := make([]notification.Setting, 0, len(notification.AvailableSettingTypes))
	for _, typ := range notification.AvailableSettingTypes {
		if s, ok := stored[typ]; ok {
			result = append(result, s)
		} else {
			result = append(result, notification.Setting{
				UserID:  userID,
				Type:    typ,
				Enabled: true,
			})
		}
	}

	return result, nil
}

func (r *repo) ToggleSetting(ctx context.Context, userID uuid.UUID, notificationType string) (notification.Setting, error) {
	const query = `
		INSERT INTO notification_settings (recipient_id, type, enabled)
		VALUES ($1, $2, false)
		ON CONFLICT (recipient_id, type) DO UPDATE
		SET enabled = NOT notification_settings.enabled
		RETURNING recipient_id, type, enabled
	`
	rows, err := r.db.Query(ctx, query, userID, notificationType)
	if err != nil {
		return notification.Setting{}, errors.WrapFail(err, "exec toggle notification setting query")
	}
	defer rows.Close()

	entity, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[settingEntity])
	if err != nil {
		return notification.Setting{}, errors.WrapFail(err, "collect toggle setting row")
	}

	return settingEntityToDomain(entity), nil
}

func (r *repo) ListPending(ctx context.Context) ([]notification.Notification, error) {
	const query = `
		SELECT id, recipient_id, title, type, text, status, sent_at, retry_after, expires_at, push_payload
		FROM notifications
		WHERE status = 'pending'
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, errors.WrapFail(err, "exec select pending notifications query")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[notificationEntity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect pending notification rows")
	}

	return slices.Map(entities, entityToDomain), nil
}

func (r *repo) Drop(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE notifications SET status = 'dropped' WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return errors.WrapFail(err, "exec drop notification query")
	}
	return nil
}

func (r *repo) MarkSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error {
	const query = `UPDATE notifications SET status = 'sent', sent_at = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, sentAt, id)
	if err != nil {
		return errors.WrapFail(err, "exec mark notification sent query")
	}
	return nil
}

func (r *repo) UpdateRetryAfter(ctx context.Context, id uuid.UUID, retryAfter time.Time) error {
	const query = `UPDATE notifications SET retry_after = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, retryAfter, id)
	if err != nil {
		return errors.WrapFail(err, "exec update notification retry_after query")
	}
	return nil
}

func (r *repo) CountSentSince(ctx context.Context, userID uuid.UUID, typ notification.SettingType, since time.Time) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM notifications
		WHERE recipient_id = $1
		  AND type = $2
		  AND status = 'sent'
		  AND sent_at > $3
	`
	rows, err := r.db.Query(ctx, query, userID, string(typ), since)
	if err != nil {
		return 0, errors.WrapFail(err, "exec count sent notifications query")
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		if err = rows.Scan(&count); err != nil {
			return 0, errors.WrapFail(err, "scan count")
		}
	}
	return count, nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd backend && go build ./notificator/internal/repo/notification/...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add notificator/internal/repo/notification/postgres/repo.go
git commit -m "feat(notificator): implement new repo methods for rate limiting"
```

---

### Task 6: RateLimiter Service

**Files:**
- Create: `notificator/internal/services/ratelimit/ratelimiter.go`
- Create: `notificator/internal/services/ratelimit/ratelimiter_test.go`

- [ ] **Step 1: Write failing tests**

Create `notificator/internal/services/ratelimit/ratelimiter_test.go`:

```go
package ratelimit_test

import (
	"context"
	"testing"
	"time"

	mock_notification "github.com/Doremi203/personage/backend/notificator/internal/domain/notification/mock"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/Doremi203/personage/backend/notificator/internal/services/ratelimit"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	userID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	now    = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
)

func newLimiter(ctrl *gomock.Controller, repo *mock_notification.MockRepo) *ratelimit.RateLimiter {
	return ratelimit.New(repo, map[notification.SettingType]ratelimit.Limits{
		notification.SettingTypeScheduleChange: {Hourly: 2, Daily: 10},
	})
}

func TestRateLimiter_Allow_typeNotInLimits(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)
	// repo should NOT be called for unconstrained types
	limiter := newLimiter(ctrl, repo)

	allowed, err := limiter.Allow(context.Background(), userID, notification.SettingTypeUpcomingEvent)

	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestRateLimiter_Allow_withinBothLimits(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)
	limiter := newLimiter(ctrl, repo)

	repo.EXPECT().
		CountSentSince(gomock.Any(), userID, notification.SettingTypeScheduleChange, gomock.Any()).
		Return(1, nil). // hourly: 1 < 2
		Times(1)
	repo.EXPECT().
		CountSentSince(gomock.Any(), userID, notification.SettingTypeScheduleChange, gomock.Any()).
		Return(5, nil). // daily: 5 < 10
		Times(1)

	allowed, err := limiter.Allow(context.Background(), userID, notification.SettingTypeScheduleChange)

	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestRateLimiter_Allow_exceedsHourlyLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)
	limiter := newLimiter(ctrl, repo)

	repo.EXPECT().
		CountSentSince(gomock.Any(), userID, notification.SettingTypeScheduleChange, gomock.Any()).
		Return(2, nil). // hourly: 2 >= 2 → denied
		Times(1)

	allowed, err := limiter.Allow(context.Background(), userID, notification.SettingTypeScheduleChange)

	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestRateLimiter_Allow_exceedsDailyLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)
	limiter := newLimiter(ctrl, repo)

	repo.EXPECT().
		CountSentSince(gomock.Any(), userID, notification.SettingTypeScheduleChange, gomock.Any()).
		Return(1, nil). // hourly: 1 < 2
		Times(1)
	repo.EXPECT().
		CountSentSince(gomock.Any(), userID, notification.SettingTypeScheduleChange, gomock.Any()).
		Return(10, nil). // daily: 10 >= 10 → denied
		Times(1)

	allowed, err := limiter.Allow(context.Background(), userID, notification.SettingTypeScheduleChange)

	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestRateLimiter_Allow_dbError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)
	limiter := newLimiter(ctrl, repo)

	repo.EXPECT().
		CountSentSince(gomock.Any(), userID, notification.SettingTypeScheduleChange, gomock.Any()).
		Return(0, assert.AnError)

	allowed, err := limiter.Allow(context.Background(), userID, notification.SettingTypeScheduleChange)

	assert.NoError(t, err) // no error returned to caller
	assert.False(t, allowed) // fail-safe: deny on DB error
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd backend && go test ./notificator/internal/services/ratelimit/... -race
```

Expected: compile error — package does not exist yet.

- [ ] **Step 3: Implement ratelimiter.go**

Create `notificator/internal/services/ratelimit/ratelimiter.go`:

```go
package ratelimit

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
)

type Limits struct {
	Hourly int
	Daily  int
}

type RateLimiter struct {
	repo   notification.Repo
	limits map[notification.SettingType]Limits
}

func New(repo notification.Repo, limits map[notification.SettingType]Limits) *RateLimiter {
	return &RateLimiter{repo: repo, limits: limits}
}

func (r *RateLimiter) Allow(ctx context.Context, userID uuid.UUID, typ notification.SettingType) (bool, error) {
	limits, ok := r.limits[typ]
	if !ok {
		return true, nil
	}

	now := time.Now()

	hourlyCount, err := r.repo.CountSentSince(ctx, userID, typ, now.Add(-time.Hour))
	if err != nil {
		return false, nil // fail-safe: deny on DB error
	}
	if hourlyCount >= limits.Hourly {
		return false, nil
	}

	dailyCount, err := r.repo.CountSentSince(ctx, userID, typ, now.Add(-24*time.Hour))
	if err != nil {
		return false, nil
	}
	if dailyCount >= limits.Daily {
		return false, nil
	}

	return true, nil
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd backend && go test ./notificator/internal/services/ratelimit/... -race
```

Expected: `ok  github.com/Doremi203/personage/backend/notificator/internal/services/ratelimit`

- [ ] **Step 5: Commit**

```bash
git add notificator/internal/services/ratelimit/
git commit -m "feat(notificator): add RateLimiter service"
```

---

### Task 7: Retrier Service

**Files:**
- Create: `notificator/internal/services/retrier/retrier.go`
- Create: `notificator/internal/services/retrier/retrier_test.go`

- [ ] **Step 1: Write failing tests**

Create `notificator/internal/services/retrier/retrier_test.go`:

```go
package retrier_test

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	mock_notification "github.com/Doremi203/personage/backend/notificator/internal/domain/notification/mock"
	"github.com/Doremi203/personage/backend/notificator/internal/services/retrier"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	userID    = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	notifID   = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	now       = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	retryInt  = 15 * time.Minute
)

// mockRateLimiter is a simple test double — no gomock needed.
type mockRateLimiter struct{ allow bool }

func (m *mockRateLimiter) Allow(_ context.Context, _ uuid.UUID, _ notification.SettingType) (bool, error) {
	return m.allow, nil
}

// mockSender records the last Send call.
type mockSender struct{ called bool }

func (m *mockSender) Send(_ context.Context, _ push.Recipient, _ push.Push) error {
	m.called = true
	return nil
}

// mockSubscriptions always returns a single-subscription recipient.
type mockSubscriptions struct{}

func (m *mockSubscriptions) GetRecipient(_ context.Context, id push.RecipientID) (push.Recipient, error) {
	return push.Recipient{
		ID: id,
		Subscriptions: []push.Subscription{{
			RecipientID: id,
			Endpoint:    "https://example.com/push",
		}},
	}, nil
}

func expiredNotif() notification.Notification {
	expired := now.Add(-3 * time.Hour)
	retryT := now.Add(-time.Minute)
	return notification.Notification{
		ID:     notifID,
		UserID: userID,
		Type:   string(notification.SettingTypeScheduleChange),
		Status: notification.StatusPending,
		PushPayload: &notification.PushPayload{Body: "b", Icon: "i", URL: "u"},
		RetryAfter: &retryT,
		ExpiresAt:  &expired,
	}
}

func readyNotif() notification.Notification {
	expires := now.Add(time.Hour)
	retryT := now.Add(-time.Minute)
	return notification.Notification{
		ID:     notifID,
		UserID: userID,
		Type:   string(notification.SettingTypeScheduleChange),
		Status: notification.StatusPending,
		PushPayload: &notification.PushPayload{Body: "b", Icon: "i", URL: "u"},
		RetryAfter: &retryT,
		ExpiresAt:  &expires,
	}
}

func notReadyNotif() notification.Notification {
	expires := now.Add(time.Hour)
	retryT := now.Add(time.Minute) // retry in the future
	return notification.Notification{
		ID:         notifID,
		UserID:     userID,
		Type:       string(notification.SettingTypeScheduleChange),
		Status:     notification.StatusPending,
		PushPayload: &notification.PushPayload{Body: "b", Icon: "i", URL: "u"},
		RetryAfter: &retryT,
		ExpiresAt:  &expires,
	}
}

func TestRetrier_ProcessOnce_ExpiredNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)

	repo.EXPECT().ListPending(gomock.Any()).Return([]notification.Notification{expiredNotif()}, nil)
	repo.EXPECT().Drop(gomock.Any(), notifID).Return(nil)

	r := retrier.New(repo, &mockRateLimiter{allow: true}, &mockSender{}, &mockSubscriptions{}, retryInt)
	r.ProcessOnce(context.Background(), now)
}

func TestRetrier_ProcessOnce_RateLimited(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)

	repo.EXPECT().ListPending(gomock.Any()).Return([]notification.Notification{readyNotif()}, nil)
	repo.EXPECT().UpdateRetryAfter(gomock.Any(), notifID, now.Add(retryInt)).Return(nil)

	r := retrier.New(repo, &mockRateLimiter{allow: false}, &mockSender{}, &mockSubscriptions{}, retryInt)
	r.ProcessOnce(context.Background(), now)
}

func TestRetrier_ProcessOnce_SendsWhenAllowed(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)
	sender := &mockSender{}

	repo.EXPECT().ListPending(gomock.Any()).Return([]notification.Notification{readyNotif()}, nil)
	repo.EXPECT().MarkSent(gomock.Any(), notifID, now).Return(nil)

	r := retrier.New(repo, &mockRateLimiter{allow: true}, sender, &mockSubscriptions{}, retryInt)
	r.ProcessOnce(context.Background(), now)

	assert.True(t, sender.called)
}

func TestRetrier_ProcessOnce_NotYetReady(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)

	repo.EXPECT().ListPending(gomock.Any()).Return([]notification.Notification{notReadyNotif()}, nil)
	// No Drop, MarkSent, or UpdateRetryAfter — notification is not due yet.

	r := retrier.New(repo, &mockRateLimiter{allow: true}, &mockSender{}, &mockSubscriptions{}, retryInt)
	r.ProcessOnce(context.Background(), now)
}

func TestRetrier_ProcessOnce_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)

	repo.EXPECT().ListPending(gomock.Any()).Return(nil, nil)

	r := retrier.New(repo, &mockRateLimiter{allow: true}, &mockSender{}, &mockSubscriptions{}, retryInt)
	r.ProcessOnce(context.Background(), now)
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd backend && go test ./notificator/internal/services/retrier/... -race
```

Expected: compile error — package does not exist yet.

- [ ] **Step 3: Implement retrier.go**

Create `notificator/internal/services/retrier/retrier.go`:

```go
package retrier

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	"github.com/google/uuid"
)

type rateLimiter interface {
	Allow(ctx context.Context, userID uuid.UUID, typ notification.SettingType) (bool, error)
}

type pushSender interface {
	Send(ctx context.Context, r push.Recipient, p push.Push) error
}

type subscriptionGetter interface {
	GetRecipient(ctx context.Context, id push.RecipientID) (push.Recipient, error)
}

type Retrier struct {
	repo          notification.Repo
	rateLimiter   rateLimiter
	sender        pushSender
	subscriptions subscriptionGetter
	retryInterval time.Duration
	logger        log.Logger
}

func New(
	repo notification.Repo,
	rateLimiter rateLimiter,
	sender pushSender,
	subscriptions subscriptionGetter,
	retryInterval time.Duration,
) *Retrier {
	return &Retrier{
		repo:          repo,
		rateLimiter:   rateLimiter,
		sender:        sender,
		subscriptions: subscriptions,
		retryInterval: retryInterval,
	}
}

func NewWithLogger(
	repo notification.Repo,
	rateLimiter rateLimiter,
	sender pushSender,
	subscriptions subscriptionGetter,
	retryInterval time.Duration,
	logger log.Logger,
) *Retrier {
	r := New(repo, rateLimiter, sender, subscriptions, retryInterval)
	r.logger = logger
	return r
}

func (r *Retrier) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:
			r.ProcessOnce(ctx, t)
		}
	}
}

func (r *Retrier) ProcessOnce(ctx context.Context, now time.Time) {
	pending, err := r.repo.ListPending(ctx)
	if err != nil {
		r.logError(err, "list pending notifications")
		return
	}

	for _, n := range pending {
		r.process(ctx, n, now)
	}
}

func (r *Retrier) process(ctx context.Context, n notification.Notification, now time.Time) {
	if n.ExpiresAt != nil && now.After(*n.ExpiresAt) {
		if err := r.repo.Drop(ctx, n.ID); err != nil {
			r.logError(err, "drop expired notification")
		}
		return
	}

	if n.RetryAfter != nil && now.Before(*n.RetryAfter) {
		return
	}

	allowed, err := r.rateLimiter.Allow(ctx, n.UserID, notification.SettingType(n.Type))
	if err != nil {
		r.logError(err, "check rate limit for pending notification")
		return
	}

	if !allowed {
		if err := r.repo.UpdateRetryAfter(ctx, n.ID, now.Add(r.retryInterval)); err != nil {
			r.logError(err, "update retry_after for pending notification")
		}
		return
	}

	recipient, err := r.subscriptions.GetRecipient(ctx, push.RecipientID(n.UserID))
	if err != nil {
		r.logError(err, "get recipient for pending notification")
		return
	}

	if len(recipient.Subscriptions) == 0 {
		if err := r.repo.Drop(ctx, n.ID); err != nil {
			r.logError(err, "drop notification with no subscriptions")
		}
		return
	}

	p := push.Push{Title: n.Title}
	if n.PushPayload != nil {
		p.Body = n.PushPayload.Body
		p.Icon = n.PushPayload.Icon
		p.Url = n.PushPayload.URL
	}

	if err := r.sender.Send(ctx, recipient, p); err != nil {
		if updateErr := r.repo.UpdateRetryAfter(ctx, n.ID, now.Add(r.retryInterval)); updateErr != nil {
			r.logError(updateErr, "update retry_after after send failure")
		}
		r.logError(err, "send push for pending notification")
		return
	}

	if err := r.repo.MarkSent(ctx, n.ID, now); err != nil {
		r.logError(err, "mark notification sent")
	}
}

func (r *Retrier) logError(err error, msg string) {
	if r.logger != nil {
		r.logger.Error(err)
	}
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd backend && go test ./notificator/internal/services/retrier/... -race
```

Expected: `ok  github.com/Doremi203/personage/backend/notificator/internal/services/retrier`

- [ ] **Step 5: Commit**

```bash
git add notificator/internal/services/retrier/
git commit -m "feat(notificator): add Retrier service for pending notifications"
```

---

### Task 8: Worker Update

**Files:**
- Modify: `notificator/internal/worker/matcher.go`

- [ ] **Step 1: Replace matcher.go**

```go
package sqspush

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase"
	"github.com/google/uuid"
)

type rateLimiter interface {
	Allow(ctx context.Context, userID uuid.UUID, typ notification.SettingType) (bool, error)
}

func NewNotificationHandler(
	logger log.Logger,
	senderUseCase usecase.PushSender,
	subscriptionUseCase usecase.PushSubscription,
	notificationRepo notification.Repo,
	rateLimiter rateLimiter,
	retryInterval time.Duration,
	maxAge time.Duration,
) *notificationHandler {
	return &notificationHandler{
		logger:              logger,
		senderUseCase:       senderUseCase,
		subscriptionUseCase: subscriptionUseCase,
		notificationRepo:    notificationRepo,
		rateLimiter:         rateLimiter,
		retryInterval:       retryInterval,
		maxAge:              maxAge,
	}
}

type notificationHandler struct {
	senderUseCase       usecase.PushSender
	subscriptionUseCase usecase.PushSubscription
	notificationRepo    notification.Repo
	rateLimiter         rateLimiter
	retryInterval       time.Duration
	maxAge              time.Duration
	logger              log.Logger
}

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

	typ := notification.SettingType(data.GetType())
	allowed, err := p.rateLimiter.Allow(ctx, recipientUUID, typ)
	if err != nil {
		p.logger.Error(errors.WrapFailf(err, "check rate limit for recipient %v", errors.Token("id", recipientUUID)))
	}

	if !allowed {
		now := time.Now()
		retryAfter := now.Add(p.retryInterval)
		expiresAt := now.Add(p.maxAge)
		return p.notificationRepo.Create(ctx, notification.Notification{
			UserID:     recipientUUID,
			Title:      data.GetTitle(),
			Type:       data.GetType(),
			Text:       data.GetDetailedText(),
			Status:     notification.StatusPending,
			RetryAfter: &retryAfter,
			ExpiresAt:  &expiresAt,
			PushPayload: &notification.PushPayload{
				Body: data.GetBody(),
				Icon: data.GetIcon(),
				URL:  data.GetUrl(),
			},
		})
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

	now := time.Now()
	return p.notificationRepo.Create(ctx, notification.Notification{
		UserID: recipientUUID,
		Title:  data.GetTitle(),
		Type:   data.GetType(),
		Text:   data.GetDetailedText(),
		Status: notification.StatusSent,
		SentAt: &now,
	})
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd backend && go build ./notificator/internal/worker/...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add notificator/internal/worker/matcher.go
git commit -m "feat(notificator): add rate limit check in SQS notification handler"
```

---

### Task 9: gRPC Handler Update

**Files:**
- Modify: `notificator/internal/grpc/notifications.go`

- [ ] **Step 1: Fix SentAt dereference in ListNotificationsV1**

In `notificator/internal/grpc/notifications.go`, find the loop that builds `protoNotifications` and change the `SentAt` line. The old code was:

```go
SentAt: timestamppb.New(n.SentAt),
```

Change it to (SentAt is now `*time.Time`; ListByUserID only returns `status='sent'` records so it's never nil):

```go
SentAt: timestamppb.New(*n.SentAt),
```

Also change `Type: n.Type` — the field is still `string` so no change needed.

- [ ] **Step 2: Verify it compiles**

```bash
cd backend && go build ./notificator/internal/grpc/...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add notificator/internal/grpc/notifications.go
git commit -m "fix(notificator): dereference nullable SentAt in notifications gRPC response"
```

---

### Task 10: App Wiring

**Files:**
- Modify: `notificator/cmd/app/main.go`

- [ ] **Step 1: Replace main.go**

```go
package main

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/libs/go/sqs"
	"github.com/Doremi203/personage/backend/libs/go/token"
	"github.com/Doremi203/personage/backend/libs/go/webapp"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/Doremi203/personage/backend/notificator/internal/grpc"
	notificationpostgres "github.com/Doremi203/personage/backend/notificator/internal/repo/notification/postgres"
	pushpostgres "github.com/Doremi203/personage/backend/notificator/internal/repo/push/postgres"
	"github.com/Doremi203/personage/backend/notificator/internal/services/ratelimit"
	"github.com/Doremi203/personage/backend/notificator/internal/services/retrier"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase"
	sqspush "github.com/Doremi203/personage/backend/notificator/internal/worker"
	"github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5/pgxpool"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	webapp.Run(func(ctx context.Context, app *webapp.App) error {
		dbConfig := postgres.Config{}
		err := app.Config.ReadSection(ctx, "database", &dbConfig)
		if err != nil {
			return err
		}

		webPushConfig := struct {
			VapidPublicKey  string
			VapidPrivateKey string
			Subscriber      string
		}{}
		err = app.Config.ReadSection(ctx, "web-push", &webPushConfig)
		if err != nil {
			return err
		}

		sqsConfig := sqs.Config{}
		err = app.Config.ReadSection(ctx, "sqs", &sqsConfig)
		if err != nil {
			return err
		}

		rateLimitConfig := struct {
			ScheduleChangeHourlyLimit int
			ScheduleChangeDailyLimit  int
			RetryInterval             time.Duration
			MaxAge                    time.Duration
		}{
			ScheduleChangeHourlyLimit: 2,
			ScheduleChangeDailyLimit:  10,
			RetryInterval:             15 * time.Minute,
			MaxAge:                    2 * time.Hour,
		}
		_ = app.Config.ReadSection(ctx, "rate-limit", &rateLimitConfig)

		poolConfig, err := pgxpool.ParseConfig(dbConfig.ConnectionString())
		if err != nil {
			return errors.WrapFail(err, "create pool config")
		}

		dbClient, err := postgres.NewClient(ctx, poolConfig)
		if err != nil {
			return errors.WrapFail(err, "create postgres client")
		}
		app.AddCloser(dbClient.Close)

		pushRepo := pushpostgres.NewRepo(dbClient)
		notificationRepo := notificationpostgres.NewRepo(dbClient)

		rateLimiter := ratelimit.New(notificationRepo, map[notification.SettingType]ratelimit.Limits{
			notification.SettingTypeScheduleChange: {
				Hourly: rateLimitConfig.ScheduleChangeHourlyLimit,
				Daily:  rateLimitConfig.ScheduleChangeDailyLimit,
			},
		})

		pushSubscriptionUseCase := usecase.NewPushSubscription(pushRepo)
		pushSubscriptionService := grpc.NewPushSubscriptionService(pushSubscriptionUseCase, app.Log)

		pushSenderUseCase := usecase.NewPushSender(
			&webpush.Options{
				Subscriber:      webPushConfig.Subscriber,
				TTL:             60,
				VAPIDPublicKey:  webPushConfig.VapidPublicKey,
				VAPIDPrivateKey: webPushConfig.VapidPrivateKey,
			},
			pushRepo,
			app.Log,
		)

		pushAdminService := grpc.NewAdminService(pushRepo, pushSenderUseCase, app.Log)

		notificationMessagesProcessor, err := sqs.NewMessageProcessor(
			ctx,
			app.Log,
			sqsConfig,
			func() *pushpb.Notification { return &pushpb.Notification{} },
			sqspush.NewNotificationHandler(
				app.Log,
				pushSenderUseCase,
				pushSubscriptionUseCase,
				notificationRepo,
				rateLimiter,
				rateLimitConfig.RetryInterval,
				rateLimitConfig.MaxAge,
			),
			5*time.Second,
			5,
		)
		if err != nil {
			return errors.WrapFail(err, "create notification messages processor")
		}
		app.AddBackgroundJob(webapp.NewBackgroundJob(
			"sqs-notifications-worker",
			notificationMessagesProcessor.ProcessMessages,
		))

		notificationRetrier := retrier.NewWithLogger(
			notificationRepo,
			rateLimiter,
			pushSenderUseCase,
			pushSubscriptionUseCase,
			rateLimitConfig.RetryInterval,
			app.Log,
		)
		app.AddBackgroundJob(webapp.NewBackgroundJob(
			"notification-retrier",
			notificationRetrier.Run,
		))

		notificationsUseCase := usecase.NewNotifications(notificationRepo)
		notificationsService := grpc.NewNotificationsService(notificationsUseCase, app.Log)

		app.AddAPIKeyProtectedEndpoints(pushpb.Admin_SendPushV1_FullMethodName)
		if app.Env == webapp.TestsEnvironment {
			app.AddGRPCUnaryInterceptor(
				token.NewUnaryTokenInterceptor(
					token.NewVerifierStub(),
					app.Log,
					pushpb.Subscription_SubscribeV1_FullMethodName,
					pushpb.Subscription_UnsubscribeV1_FullMethodName,
					pushpb.Notifications_ListNotificationsV1_FullMethodName,
					pushpb.Notifications_ToggleNotificationV1_FullMethodName,
					pushpb.Notifications_GetNotificationSettingsV1_FullMethodName,
				),
			)
		} else {
			type AuthConfig struct {
				Address string
			}
			authConfig := AuthConfig{}
			err = app.Config.ReadSection(ctx, "auth", &authConfig)
			if err != nil {
				return err
			}

			authConn, err := googlegrpc.NewClient(
				authConfig.Address,
				googlegrpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
			)
			if err != nil {
				return errors.WrapFail(err, "create auth grpc client")
			}
			app.AddCloser(authConn.Close)

			app.AddGRPCUnaryInterceptor(
				token.NewUnaryTokenInterceptor(
					token.NewGRPCVerifier(authConn),
					app.Log,
					pushpb.Subscription_SubscribeV1_FullMethodName,
					pushpb.Subscription_UnsubscribeV1_FullMethodName,
					pushpb.Notifications_ListNotificationsV1_FullMethodName,
					pushpb.Notifications_ToggleNotificationV1_FullMethodName,
					pushpb.Notifications_GetNotificationSettingsV1_FullMethodName,
				),
			)
		}

		app.RegisterGRPCServices(pushSubscriptionService, pushAdminService, notificationsService)
		app.AddGatewayHandlers(pushSubscriptionService, pushAdminService, notificationsService)

		return nil
	})
}
```

- [ ] **Step 2: Verify full build**

```bash
cd backend && go build ./notificator/...
```

Expected: no output.

- [ ] **Step 3: Run all tests**

```bash
cd backend && go test ./notificator/... -race
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add notificator/cmd/app/main.go
git commit -m "feat(notificator): wire RateLimiter and Retrier into app"
```
