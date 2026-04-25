package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	clusterpostgres "github.com/Doremi203/personage/backend/tasker/internal/repo/cluster/postgres"
	taskpostgres "github.com/Doremi203/personage/backend/tasker/internal/repo/task/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
)

type testUser struct {
	UserID domain.UserID
	Tasks  []testTask
}

type testTask struct {
	Title           string
	Description     string
	DurationMinutes int
	Priority        int
	Deadline        *time.Time
}

func main() {
	ctx := context.Background()

	dbConfig := postgres.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		Database: "tasker",
		Options:  "sslmode=disable",
	}

	poolConfig, err := pgxpool.ParseConfig(dbConfig.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to parse pool config: %v", err)
	}

	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	dbClient, err := postgres.NewClient(ctx, poolConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		err = dbClient.Close()
		if err != nil {
			log.Fatalf("Failed to close database client: %v", err)
		}
	}()

	fmt.Println("Connected to database successfully")

	clusterRepo := clusterpostgres.NewRepo(dbClient, time.Now)
	taskRepo := taskpostgres.NewRepo(dbClient)

	testUsers := generateTestUsers()

	for _, u := range testUsers {
		if err := insertUserData(ctx, clusterRepo, taskRepo, u); err != nil {
			log.Printf("Failed to insert data for user %s: %v", u.UserID, err)
			continue
		}
		fmt.Printf("Inserted %d pending tasks for user %s\n", len(u.Tasks), u.UserID)
	}

	fmt.Println("\nDone. You can now run the scheduling worker to verify it picks up these tasks.")
}

func insertUserData(
	ctx context.Context,
	clusterRepo domain.ClusterRepo,
	taskRepo domain.TaskRepo,
	u testUser,
) error {
	now := time.Now()

	for _, t := range u.Tasks {
		clusterID := domain.ClusterID(uuid.New().String())

		cluster := domain.Cluster{
			ID:         clusterID,
			UserID:     u.UserID,
			Centroid:   make([]float32, 1536),
			EventCount: 1,
			Status:     domain.ClusterStatusClosed,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := clusterRepo.UpsertCluster(ctx, cluster); err != nil {
			return errors.WrapFailf(err, "create cluster for task %v", errors.Token("title", t.Title))
		}

		task := domain.Task{
			ID:          domain.TaskID(uuid.New().String()),
			UserID:      u.UserID,
			ClusterID:   &clusterID,
			Title:       t.Title,
			Description: t.Description,
			Duration:    time.Duration(t.DurationMinutes) * time.Minute,
			Priority:    t.Priority,
			Deadline:    t.Deadline,
			StartTime:   nil,
			Status:      domain.TaskStatusUnplanned,
			Category:    domain.TaskCategoryPersonal,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := taskRepo.CreateTask(ctx, task); err != nil {
			return errors.WrapFailf(err, "create task %v", errors.Token("title", t.Title))
		}
	}

	return nil
}

func generateTestUsers() []testUser {
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)
	inTwoDays := now.Add(48 * time.Hour)
	inThreeDays := now.Add(72 * time.Hour)

	return []testUser{
		{
			UserID: domain.UserID("11111111-1111-1111-1111-111111111111"),
			Tasks: []testTask{
				{
					Title:           "Review project proposal",
					Description:     "Review and provide feedback on the Q1 project proposal",
					DurationMinutes: 60,
					Priority:        3,
					Deadline:        &tomorrow,
				},
				{
					Title:           "Team standup",
					Description:     "Daily team standup meeting",
					DurationMinutes: 15,
					Priority:        5,
					Deadline:        &tomorrow,
				},
				{
					Title:           "Code review",
					Description:     "Review pull request #123",
					DurationMinutes: 45,
					Priority:        2,
					Deadline:        &inTwoDays,
				},
			},
		},
		{
			UserID: domain.UserID("22222222-2222-2222-2222-222222222222"),
			Tasks: []testTask{
				{
					Title:           "Write documentation",
					Description:     "Update API documentation for new endpoints",
					DurationMinutes: 90,
					Priority:        2,
					Deadline:        &inTwoDays,
				},
				{
					Title:           "Fix bug #456",
					Description:     "Investigate and fix the authentication bug",
					DurationMinutes: 120,
					Priority:        4,
					Deadline:        &tomorrow,
				},
				{
					Title:           "Client meeting",
					Description:     "Quarterly review meeting with client",
					DurationMinutes: 60,
					Priority:        5,
					Deadline:        &tomorrow,
				},
				{
					Title:           "Update dependencies",
					Description:     "Update all Go dependencies to latest versions",
					DurationMinutes: 30,
					Priority:        1,
					Deadline:        &inThreeDays,
				},
			},
		},
		{
			UserID: domain.UserID("33333333-3333-3333-3333-333333333333"),
			Tasks: []testTask{
				{
					Title:           "Design review",
					Description:     "Review new UI designs for dashboard",
					DurationMinutes: 45,
					Priority:        3,
					Deadline:        &tomorrow,
				},
				{
					Title:           "Performance testing",
					Description:     "Run performance tests on the new API",
					DurationMinutes: 180,
					Priority:        4,
					Deadline:        &inTwoDays,
				},
				{
					Title:           "Sprint planning",
					Description:     "Prepare for next sprint planning meeting",
					DurationMinutes: 60,
					Priority:        3,
					Deadline:        &tomorrow,
				},
				{
					Title:           "Security audit",
					Description:     "Review security findings and create remediation plan",
					DurationMinutes: 90,
					Priority:        5,
					Deadline:        &inTwoDays,
				},
				{
					Title:           "Database backup verification",
					Description:     "Verify database backup procedures",
					DurationMinutes: 30,
					Priority:        2,
					Deadline:        &inThreeDays,
				},
			},
		},
	}
}
