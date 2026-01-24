package main

import (
	"context"
	"time"

	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino-ext/components/model/openrouter"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
	"gitlab.com/amoguscorp/personage/backend/libs/go/errors"
	"gitlab.com/amoguscorp/personage/backend/libs/go/postgres"
	"gitlab.com/amoguscorp/personage/backend/libs/go/sqs"
	"gitlab.com/amoguscorp/personage/backend/libs/go/webapp"
	eventsPb "gitlab.com/amoguscorp/personage/backend/tasker/gen/api/events"
	"gitlab.com/amoguscorp/personage/backend/tasker/internal/handlers/sqs/event"
	clusterpostgres "gitlab.com/amoguscorp/personage/backend/tasker/internal/repo/cluster/postgres"
	eventpostgres "gitlab.com/amoguscorp/personage/backend/tasker/internal/repo/event/postgres"
	taskpostgres "gitlab.com/amoguscorp/personage/backend/tasker/internal/repo/task/postgres"
	"gitlab.com/amoguscorp/personage/backend/tasker/internal/services/embedding"
	"gitlab.com/amoguscorp/personage/backend/tasker/internal/services/llm"
	"gitlab.com/amoguscorp/personage/backend/tasker/internal/usecase/clusterization"
	"gitlab.com/amoguscorp/personage/backend/tasker/internal/usecase/taskgeneration"
	"gitlab.com/amoguscorp/personage/backend/tasker/internal/workers/clusterclosure"
)

func main() {
	webapp.Run(func(ctx context.Context, app *webapp.App) error {
		sqsConfig := sqs.Config{}
		err := app.Config.ReadSection(ctx, "sqs", &sqsConfig)
		if err != nil {
			return err
		}

		dbConfig := postgres.Config{}
		err = app.Config.ReadSection(ctx, "database", &dbConfig)
		if err != nil {
			return err
		}

		poolConfig, err := pgxpool.ParseConfig(dbConfig.ConnectionString())
		if err != nil {
			return errors.WrapFail(err, "create pool config")
		}
		poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			return pgxvec.RegisterTypes(ctx, conn)
		}

		dbClient, err := postgres.NewClient(ctx, poolConfig)
		if err != nil {
			return errors.WrapFail(err, "create postgres client")
		}
		app.AddCloser(dbClient.Close)

		postgresTxProvider := postgres.NewTxProvider(dbClient.Pool)

		postgresEventRepo := eventpostgres.NewRepo(dbClient)
		postgresClusterRepo := clusterpostgres.NewRepo(dbClient)
		postgresTaskRepo := taskpostgres.NewRepo(dbClient)

		type LLMConfig struct {
			ApiKey string
			Model  string
		}

		llmConfig := LLMConfig{}
		err = app.Config.ReadSection(ctx, "llm", &llmConfig)
		if err != nil {
			return err
		}

		llmModel, err := openrouter.NewChatModel(ctx, &openrouter.Config{
			APIKey: llmConfig.ApiKey,
			Model:  llmConfig.Model,
		})
		if err != nil {
			return errors.WrapFail(err, "init openrouter llm model")
		}

		type EmbeddingConfig struct {
			Model  string
			ApiKey string
		}
		embeddingConfig := EmbeddingConfig{}
		err = app.Config.ReadSection(ctx, "embedding", &embeddingConfig)
		if err != nil {
			return err
		}

		embedder, err := openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
			APIKey:  embeddingConfig.ApiKey,
			Model:   embeddingConfig.Model,
			Timeout: 30 * time.Second,
			BaseURL: "https://openrouter.ai/api/v1",
		})
		if err != nil {
			return errors.WrapFail(err, "init openai embedder")
		}
		embeddingService := embedding.NewEinoService(embedder)

		clusterizationUseCase := clusterization.NewUseCase(
			app.Log,
			postgresTxProvider,
			embeddingService,
			postgresEventRepo,
			postgresClusterRepo,
			0.6,
			5,
		)

		connectorEventsProcessor, err := sqs.NewMessageProcessor(
			ctx,
			app.Log,
			sqsConfig,
			func() *eventsPb.Event { return &eventsPb.Event{} },
			event.NewHandler(clusterizationUseCase),
			10*time.Second,
			5,
		)
		if err != nil {
			return errors.WrapFail(err, "create connector events processor")
		}

		app.AddBackgroundJob(
			webapp.NewBackgroundJob(
				"sqs-worker",
				connectorEventsProcessor.ProcessMessages,
			).WithInterval(time.Second),
		)

		taskGenerationService := llm.NewTaskGenerationService(llmModel)

		type ClusterClosureConfig struct {
			MaxEventCount     int
			InactivityMinutes int
			Interval          time.Duration
			BatchSize         int
		}

		clusterClosureConfig := ClusterClosureConfig{
			MaxEventCount:     5,
			InactivityMinutes: 5,
			Interval:          time.Second * 5,
			BatchSize:         10,
		}
		err = app.Config.ReadSection(ctx, "cluster-closure", &clusterClosureConfig)
		if err != nil {
			app.Log.Infof("cluster-closure config not found, using defaults: %+v", clusterClosureConfig)
		}

		taskGenerationUseCase := taskgeneration.NewUseCase(
			postgresClusterRepo,
			postgresEventRepo,
			postgresTaskRepo,
			taskGenerationService,
			clusterClosureConfig.MaxEventCount,
			time.Duration(clusterClosureConfig.InactivityMinutes)*time.Minute,
		)

		clusterClosureWorker := clusterclosure.NewWorker(
			taskGenerationUseCase,
			clusterClosureConfig.BatchSize,
			app.Log,
		)

		app.AddBackgroundJob(
			webapp.NewBackgroundJob(
				"cluster-closure-worker",
				clusterClosureWorker.Process,
			).WithInterval(clusterClosureConfig.Interval),
		)

		return nil
	})
}
