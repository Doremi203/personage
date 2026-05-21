package taskpostgres

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

func newTask(t *testing.T, userID domain.UserID) domain.Task {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	return domain.Task{
		ID:          domain.TaskID(uuid.NewString()),
		UserID:      userID,
		Title:       "title",
		Description: "desc",
		Duration:    30 * time.Minute,
		Priority:    5,
		Status:      domain.TaskStatusUnplanned,
		Category:    domain.TaskCategoryPersonal,
		IsApproved:  true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func Test_repo_CreateTask_GetTaskByID(t *testing.T) {
	userA := domain.UserID(uuid.NewString())
	userB := domain.UserID(uuid.NewString())

	tester.Run(t, "create then get", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			task := newTask(t, userA)
			require.NoError(t, r.CreateTask(ctx, task))

			got, err := r.GetTaskByID(ctx, task.ID, userA)
			require.NoError(t, err)
			assert.Equal(t, task.ID, got.ID)
			assert.Equal(t, "title", got.Title)
			assert.Equal(t, domain.TaskStatusUnplanned, got.Status)
		},
	)

	tester.Run(t, "get unknown returns ErrTaskNotFound", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			_, err := r.GetTaskByID(ctx, domain.TaskID(uuid.NewString()), userA)
			require.ErrorIs(t, err, domain.ErrTaskNotFound)
		},
	)

	tester.Run(t, "get with wrong user returns ErrTaskNotFound", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			task := newTask(t, userA)
			require.NoError(t, r.CreateTask(ctx, task))

			_, err := r.GetTaskByID(ctx, task.ID, userB)
			require.ErrorIs(t, err, domain.ErrTaskNotFound)
		},
	)
}

func Test_repo_GetTasksByUserID(t *testing.T) {
	userA := domain.UserID(uuid.NewString())
	userB := domain.UserID(uuid.NewString())

	tester.Run(t, "returns only user's tasks", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			require.NoError(t, r.CreateTask(ctx, newTask(t, userA)))
			require.NoError(t, r.CreateTask(ctx, newTask(t, userA)))
			require.NoError(t, r.CreateTask(ctx, newTask(t, userB)))

			got, err := r.GetTasksByUserID(ctx, userA)
			require.NoError(t, err)
			assert.Len(t, got, 2)
		},
	)

	tester.Run(t, "no tasks returns empty", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			got, err := r.GetTasksByUserID(ctx, userA)
			require.NoError(t, err)
			assert.Empty(t, got)
		},
	)
}

func Test_repo_GetTasksByStatus(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "filters by status", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			t1 := newTask(t, userA)
			t1.Status = domain.TaskStatusPlanned
			t2 := newTask(t, userA)
			t2.Status = domain.TaskStatusUnplanned
			require.NoError(t, r.CreateTask(ctx, t1))
			require.NoError(t, r.CreateTask(ctx, t2))

			got, err := r.GetTasksByStatus(ctx, userA, domain.TaskStatusPlanned)
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, t1.ID, got[0].ID)
		},
	)
}

func Test_repo_GetUsersWithUnplannedTasks(t *testing.T) {
	userA := domain.UserID(uuid.NewString())
	userB := domain.UserID(uuid.NewString())
	userC := domain.UserID(uuid.NewString())

	tester.Run(t, "returns distinct users with unplanned", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			a1 := newTask(t, userA)
			a2 := newTask(t, userA)
			b1 := newTask(t, userB)
			b1.Status = domain.TaskStatusPlanned
			c1 := newTask(t, userC)
			require.NoError(t, r.CreateTask(ctx, a1))
			require.NoError(t, r.CreateTask(ctx, a2))
			require.NoError(t, r.CreateTask(ctx, b1))
			require.NoError(t, r.CreateTask(ctx, c1))

			got, err := r.GetUsersWithUnplannedTasks(ctx)
			require.NoError(t, err)
			assert.ElementsMatch(t, []domain.UserID{userA, userC}, got)
		},
	)
}

func Test_repo_UpdateTaskSchedule(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "updates planned schedule", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			task := newTask(t, userA)
			require.NoError(t, r.CreateTask(ctx, task))

			start := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
			end := time.Date(2026, 4, 26, 11, 0, 0, 0, time.UTC)
			require.NoError(t, r.UpdateTaskSchedule(ctx, task.ID, start, end, domain.TaskStatusPlanned))

			got, err := r.GetTaskByID(ctx, task.ID, userA)
			require.NoError(t, err)
			require.NotNil(t, got.StartTime)
			require.NotNil(t, got.EndTime)
			assert.True(t, got.StartTime.Equal(start))
			assert.True(t, got.EndTime.Equal(end))
			assert.Equal(t, domain.TaskStatusPlanned, got.Status)
		},
	)

	tester.Run(t, "unknown task returns error", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			err := r.UpdateTaskSchedule(ctx, domain.TaskID(uuid.NewString()),
				time.Now().UTC(), time.Now().UTC(), domain.TaskStatusPlanned)
			require.Error(t, err)
		},
	)
}

func Test_repo_UpdateTaskStatus(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "updates status", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			task := newTask(t, userA)
			require.NoError(t, r.CreateTask(ctx, task))
			require.NoError(t, r.UpdateTaskStatus(ctx, task.ID, domain.TaskStatusCompleted))

			got, err := r.GetTaskByID(ctx, task.ID, userA)
			require.NoError(t, err)
			assert.Equal(t, domain.TaskStatusCompleted, got.Status)
		},
	)

	tester.Run(t, "unknown task returns error", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			err := r.UpdateTaskStatus(ctx, domain.TaskID(uuid.NewString()), domain.TaskStatusCompleted)
			require.Error(t, err)
		},
	)
}

func Test_repo_DeleteTask(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "deletes task", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			task := newTask(t, userA)
			require.NoError(t, r.CreateTask(ctx, task))
			require.NoError(t, r.DeleteTask(ctx, task.ID))

			_, err := r.GetTaskByID(ctx, task.ID, userA)
			require.ErrorIs(t, err, domain.ErrTaskNotFound)
		},
	)

	tester.Run(t, "unknown task returns ErrTaskNotFound", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			err := r.DeleteTask(ctx, domain.TaskID(uuid.NewString()))
			require.ErrorIs(t, err, domain.ErrTaskNotFound)
		},
	)
}

func Test_repo_UpdateTask(t *testing.T) {
	userA := domain.UserID(uuid.NewString())
	userB := domain.UserID(uuid.NewString())

	tester.Run(t, "updates fields and bumps updated_at", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			task := newTask(t, userA)
			require.NoError(t, r.CreateTask(ctx, task))

			start := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
			end := time.Date(2026, 4, 26, 11, 0, 0, 0, time.UTC)
			cat := domain.TaskCategoryWork
			updated, err := r.UpdateTask(ctx, task.ID, userA, domain.TaskUpdate{
				Title:       new("new"),
				Description: new("new desc"),
				StartTime:   &start,
				EndTime:     &end,
				Category:    &cat,
			})
			require.NoError(t, err)
			assert.Equal(t, "new", updated.Title)
			assert.Equal(t, "new desc", updated.Description)
			assert.Equal(t, domain.TaskCategoryWork, updated.Category)
			require.NotNil(t, updated.StartTime)
			assert.True(t, updated.StartTime.Equal(start))
		},
	)

	tester.Run(t, "no fields returns current task", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			task := newTask(t, userA)
			require.NoError(t, r.CreateTask(ctx, task))

			got, err := r.UpdateTask(ctx, task.ID, userA, domain.TaskUpdate{})
			require.NoError(t, err)
			assert.Equal(t, task.ID, got.ID)
		},
	)

	tester.Run(t, "wrong user returns ErrTaskNotFound", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			task := newTask(t, userA)
			require.NoError(t, r.CreateTask(ctx, task))

			_, err := r.UpdateTask(ctx, task.ID, userB, domain.TaskUpdate{Title: new("x")})
			require.ErrorIs(t, err, domain.ErrTaskNotFound)
		},
	)

	tester.Run(t, "unknown task returns ErrTaskNotFound", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			_, err := r.UpdateTask(ctx, domain.TaskID(uuid.NewString()), userA, domain.TaskUpdate{Title: new("x")})
			require.ErrorIs(t, err, domain.ErrTaskNotFound)
		},
	)
}

func Test_repo_ListTasks(t *testing.T) {
	userA := domain.UserID(uuid.NewString())
	userB := domain.UserID(uuid.NewString())

	tester.Run(t, "filters by user, status, category, text and time range", nil, 15*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			t1 := newTask(t, userA)
			t1.Title = "alpha report"
			t1.Status = domain.TaskStatusPlanned
			t1.Category = domain.TaskCategoryWork
			t1.CreatedAt = time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)

			t2 := newTask(t, userA)
			t2.Title = "beta study"
			t2.Status = domain.TaskStatusUnplanned
			t2.Category = domain.TaskCategoryStudy
			t2.CreatedAt = time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)

			t3 := newTask(t, userA)
			t3.Title = "alpha personal"
			t3.Status = domain.TaskStatusPlanned
			t3.Category = domain.TaskCategoryPersonal
			t3.CreatedAt = time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)

			t4 := newTask(t, userB)
			t4.Title = "alpha other user"
			t4.Status = domain.TaskStatusPlanned
			t4.Category = domain.TaskCategoryWork

			require.NoError(t, r.CreateTask(ctx, t1))
			require.NoError(t, r.CreateTask(ctx, t2))
			require.NoError(t, r.CreateTask(ctx, t3))
			require.NoError(t, r.CreateTask(ctx, t4))

			from := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
			till := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
			status := domain.TaskStatusPlanned
			category := domain.TaskCategoryPersonal
			tasks, total, err := r.ListTasks(ctx,
				domain.TaskFilter{
					UserID:   userA,
					Status:   &status,
					Category: &category,
					Text:     "alpha",
					From:     &from,
					Till:     &till,
				},
				domain.Pagination{Page: 1, PageSize: 10},
			)
			require.NoError(t, err)
			require.Len(t, tasks, 1)
			assert.Equal(t, t3.ID, tasks[0].ID)
			assert.Equal(t, 1, total)
		},
	)

	tester.Run(t, "paginates results", nil, 15*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db)

			for i := range 5 {
				task := newTask(t, userA)
				task.CreatedAt = time.Date(2026, 4, 20+i, 12, 0, 0, 0, time.UTC)
				require.NoError(t, r.CreateTask(ctx, task))
			}

			tasks, total, err := r.ListTasks(ctx,
				domain.TaskFilter{UserID: userA},
				domain.Pagination{Page: 2, PageSize: 2},
			)
			require.NoError(t, err)
			assert.Equal(t, 5, total)
			assert.Len(t, tasks, 2)
		},
	)
}
