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

func Test_repo_MarkAsRead(t *testing.T) {
	owner := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	other := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tester.Run(t, "marks as read sets read_at", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			id, err := r.CreateAndReturnID(ctx, notification.Notification{
				UserID: owner,
				Title:  "n",
				Type:   "upcoming_event",
				Text:   "body",
			})
			require.NoError(t, err)

			readAt := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			require.NoError(t, r.MarkAsRead(ctx, id, owner, readAt))

			list, err := r.ListByUserID(ctx, owner, 10, 0)
			require.NoError(t, err)
			require.Len(t, list, 1)
			require.NotNil(t, list[0].ReadAt)
			assert.True(t, list[0].ReadAt.Equal(readAt))
		},
	)

	tester.Run(t, "second call is idempotent and does not overwrite", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			id, err := r.CreateAndReturnID(ctx, notification.Notification{
				UserID: owner,
				Title:  "n",
				Type:   "upcoming_event",
				Text:   "body",
			})
			require.NoError(t, err)

			first := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			second := time.Date(2026, 4, 26, 13, 0, 0, 0, time.UTC)

			require.NoError(t, r.MarkAsRead(ctx, id, owner, first))
			require.NoError(t, r.MarkAsRead(ctx, id, owner, second))

			list, err := r.ListByUserID(ctx, owner, 10, 0)
			require.NoError(t, err)
			require.Len(t, list, 1)
			require.NotNil(t, list[0].ReadAt)
			assert.True(t, list[0].ReadAt.Equal(first))
		},
	)

	tester.Run(t, "cross-user blocked", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			id, err := r.CreateAndReturnID(ctx, notification.Notification{
				UserID: owner,
				Title:  "n",
				Type:   "upcoming_event",
				Text:   "body",
			})
			require.NoError(t, err)

			err = r.MarkAsRead(ctx, id, other, time.Now().UTC())
			require.ErrorIs(t, err, notification.ErrNotificationNotFound)

			list, err := r.ListByUserID(ctx, owner, 10, 0)
			require.NoError(t, err)
			require.Len(t, list, 1)
			assert.Nil(t, list[0].ReadAt)
		},
	)

	tester.Run(t, "unknown id returns ErrNotificationNotFound", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			err := r.MarkAsRead(ctx, uuid.New(), owner, time.Now().UTC())
			require.ErrorIs(t, err, notification.ErrNotificationNotFound)
		},
	)
}

func Test_repo_MarkAllAsRead(t *testing.T) {
	owner := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	other := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	tester.Run(t, "marks every unread for user", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			for range 3 {
				_, err := r.CreateAndReturnID(ctx, notification.Notification{
					UserID: owner,
					Title:  "n",
					Type:   "upcoming_event",
					Text:   "body",
				})
				require.NoError(t, err)
			}

			readAt := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			require.NoError(t, r.MarkAllAsRead(ctx, owner, readAt))

			list, err := r.ListByUserID(ctx, owner, 10, 0)
			require.NoError(t, err)
			require.Len(t, list, 3)
			for i, n := range list {
				require.NotNil(t, n.ReadAt, "row %d", i)
				assert.True(t, n.ReadAt.Equal(readAt), "row %d", i)
			}
		},
	)

	tester.Run(t, "leaves already-read rows untouched", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			id1, err := r.CreateAndReturnID(ctx, notification.Notification{
				UserID: owner, Title: "n1", Type: "upcoming_event", Text: "body",
			})
			require.NoError(t, err)
			id2, err := r.CreateAndReturnID(ctx, notification.Notification{
				UserID: owner, Title: "n2", Type: "upcoming_event", Text: "body",
			})
			require.NoError(t, err)

			early := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
			late := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			require.NoError(t, r.MarkAsRead(ctx, id1, owner, early))
			require.NoError(t, r.MarkAllAsRead(ctx, owner, late))

			list, err := r.ListByUserID(ctx, owner, 10, 0)
			require.NoError(t, err)
			require.Len(t, list, 2)

			byID := map[uuid.UUID]*time.Time{}
			for _, n := range list {
				byID[n.ID] = n.ReadAt
			}
			require.NotNil(t, byID[id1])
			assert.True(t, byID[id1].Equal(early))
			require.NotNil(t, byID[id2])
			assert.True(t, byID[id2].Equal(late))
		},
	)

	tester.Run(t, "leaves other users untouched", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			_, err := r.CreateAndReturnID(ctx, notification.Notification{
				UserID: owner, Title: "n", Type: "upcoming_event", Text: "body",
			})
			require.NoError(t, err)
			_, err = r.CreateAndReturnID(ctx, notification.Notification{
				UserID: other, Title: "n", Type: "upcoming_event", Text: "body",
			})
			require.NoError(t, err)

			require.NoError(t, r.MarkAllAsRead(ctx, owner, time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)))

			otherList, err := r.ListByUserID(ctx, other, 10, 0)
			require.NoError(t, err)
			require.Len(t, otherList, 1)
			assert.Nil(t, otherList[0].ReadAt)
		},
	)
}
