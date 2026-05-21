package moderationpostgres

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_repo_AddRemoveListUsers(t *testing.T) {
	userA := domain.UserID(uuid.NewString())
	userB := domain.UserID(uuid.NewString())

	tester.Run(t, "add then list then remove round-trip", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			required, err := r.RequiresModeration(ctx, userA)
			require.NoError(t, err)
			assert.False(t, required)

			require.NoError(t, r.AddUser(ctx, userA))
			require.NoError(t, r.AddUser(ctx, userB))

			require.NoError(t, r.AddUser(ctx, userA)) // idempotent

			required, err = r.RequiresModeration(ctx, userA)
			require.NoError(t, err)
			assert.True(t, required)

			users, err := r.ListUsers(ctx)
			require.NoError(t, err)
			assert.ElementsMatch(t, []domain.UserID{userA, userB}, users)

			require.NoError(t, r.RemoveUser(ctx, userA))

			required, err = r.RequiresModeration(ctx, userA)
			require.NoError(t, err)
			assert.False(t, required)

			users, err = r.ListUsers(ctx)
			require.NoError(t, err)
			assert.ElementsMatch(t, []domain.UserID{userB}, users)
		},
	)

	tester.Run(t, "remove unknown is no-op", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)
			require.NoError(t, r.RemoveUser(ctx, domain.UserID(uuid.NewString())))
		},
	)
}
