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
