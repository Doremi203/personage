package moderationpostgres

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Doremi203/personage/backend/libs/go/postgres"
)

var tester postgres.Tester

func TestMain(m *testing.M) {
	postgres.SetupTests(m, &tester, "tasker",
		postgres.WithImage("pgvector/pgvector:pg18"),
		postgres.WithBeforeMigrate(func(ctx context.Context, db *sql.DB) error {
			_, err := db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
			return err
		}),
	)
}
