package main

import (
	"context"
	"time"
	_ "time/tzdata"

	authpb "github.com/Doremi203/personage/backend/libs/go/auth/gen/api/auth"
	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/libs/go/sqs"
	"github.com/Doremi203/personage/backend/libs/go/token"
	"github.com/Doremi203/personage/backend/libs/go/webapp"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	eventsPb "github.com/Doremi203/personage/backend/tasker/gen/api/events"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	taskergrpc "github.com/Doremi203/personage/backend/tasker/internal/grpc"
	"github.com/Doremi203/personage/backend/tasker/internal/handlers/sqs/event"
	clusterpostgres "github.com/Doremi203/personage/backend/tasker/internal/repo/cluster/postgres"
	eventpostgres "github.com/Doremi203/personage/backend/tasker/internal/repo/event/postgres"
	moderationpostgres "github.com/Doremi203/personage/backend/tasker/internal/repo/moderation/postgres"
	pausepostgres "github.com/Doremi203/personage/backend/tasker/internal/repo/pause/postgres"
	promptpostgres "github.com/Doremi203/personage/backend/tasker/internal/repo/prompt/postgres"
	taskpostgres "github.com/Doremi203/personage/backend/tasker/internal/repo/task/postgres"
	"github.com/Doremi203/personage/backend/tasker/internal/services/embedding"
	"github.com/Doremi203/personage/backend/tasker/internal/services/llm"
	"github.com/Doremi203/personage/backend/tasker/internal/services/notifications"
	"github.com/Doremi203/personage/backend/tasker/internal/services/prompts"
	"github.com/Doremi203/personage/backend/tasker/internal/services/userprofile"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/admin"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/clusterization"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/scheduling"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/taskgeneration"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/tasklist"
	"github.com/Doremi203/personage/backend/tasker/internal/workers/clusterclosure"
	schedulingworker "github.com/Doremi203/personage/backend/tasker/internal/workers/scheduling"
	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino-ext/components/model/openrouter"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

		type TimeConfig struct {
			DefaultTimezone string
		}
		timeConfig := TimeConfig{DefaultTimezone: "Europe/Moscow"}
		if err = app.Config.ReadSection(ctx, "time", &timeConfig); err != nil {
			app.Log.Infof("time config not found, using defaults: %+v", timeConfig)
		}
		defaultLocation, err := time.LoadLocation(timeConfig.DefaultTimezone)
		if err != nil {
			return errors.WrapFailf(
				err,
				"load default timezone %s",
				errors.Token("timezone", timeConfig.DefaultTimezone),
			)
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

		postgresTxProvider := postgres.NewTxProvider(dbClient.Pool, app.Log)

		postgresEventRepo := eventpostgres.NewRepo(dbClient, time.Now)
		postgresClusterRepo := clusterpostgres.NewRepo(dbClient, time.Now)
		postgresTaskRepo := taskpostgres.NewRepo(dbClient)
		postgresPauseRepo := pausepostgres.NewRepo(dbClient, time.Now)
		postgresModerationRepo := moderationpostgres.NewRepo(dbClient)
		postgresPromptRepo := promptpostgres.NewRepo(dbClient, time.Now)

		promptsService := prompts.NewService(postgresPromptRepo, 30*time.Second, time.Now)

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
			postgresPauseRepo,
			0.65,
			0.90,
			5,
			time.Now,
		)

		connectorEventsProcessor, err := sqs.NewMessageProcessor(
			ctx,
			app.Log,
			sqsConfig,
			func() *eventsPb.Event { return &eventsPb.Event{} },
			event.NewHandler(clusterizationUseCase, defaultLocation),
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

		actionabilityService := llm.NewClusterActionabilityService(llmModel, app.Log, promptsService)
		taskGenerationService := llm.NewTaskGenerationService(llmModel, app.Log, promptsService, defaultLocation)

		isTestEnv := app.Env == webapp.TestsEnvironment || app.Env == webapp.EvalEnvironment

		var authConn *grpc.ClientConn
		var userProfileSvc domain.UserProfileService
		if isTestEnv {
			userProfileSvc = userprofile.NewStub()
		} else {
			type AuthConfig struct {
				Address string
			}
			authConfig := AuthConfig{}
			if err = app.Config.ReadSection(ctx, "auth", &authConfig); err != nil {
				return err
			}

			authConn, err = grpc.NewClient(
				authConfig.Address,
				grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
			)
			if err != nil {
				return errors.WrapFail(err, "create auth grpc client")
			}
			app.AddCloser(authConn.Close)

			userProfileSvc = userprofile.NewCachedService(
				userprofile.NewGRPCService(authpb.NewAuthServiceClient(authConn)),
				30*time.Minute,
				5*time.Minute,
				time.Now,
			)
		}

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
			postgresModerationRepo,
			actionabilityService,
			taskGenerationService,
			userProfileSvc,
			postgresTxProvider,
			app.Log,
			clusterClosureConfig.MaxEventCount,
			time.Duration(clusterClosureConfig.InactivityMinutes)*time.Minute,
			time.Now,
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

		sqsNotificatorConfig := sqs.Config{}
		err = app.Config.ReadSection(ctx, "sqs-notificator", &sqsNotificatorConfig)
		if err != nil {
			return err
		}

		notificatorSQSClient, err := sqs.New(
			ctx,
			sqsNotificatorConfig,
			func() *pushpb.Notification { return &pushpb.Notification{} },
		)
		if err != nil {
			return errors.WrapFail(err, "create notificator sqs client")
		}

		notificationConfig := domain.NotificationConfig{}
		err = app.Config.ReadSection(ctx, "notifications", &notificationConfig)
		if err != nil {
			return err
		}

		var ruPrinter = message.NewPrinter(language.Russian)

		notificationSender := notifications.NewNotificatorPushService(notificatorSQSClient, time.Now)

		upcomingEventNotifier, err := notifications.NewUpcomingEventNotifier(
			app.Log,
			notificationSender,
			notificationConfig,
			ruPrinter,
		)
		if err != nil {
			return err
		}

		type NotificationWorkerConfig struct {
			Interval time.Duration
		}
		notificationWorkerConfig := NotificationWorkerConfig{}
		err = app.Config.ReadSection(ctx, "notifications-worker", &notificationWorkerConfig)
		if err != nil {
			return err
		}

		notificationWorker := notifications.NewWorker(
			app.Log,
			postgresTaskRepo,
			upcomingEventNotifier,
		)
		app.AddBackgroundJob(
			webapp.NewBackgroundJob(
				"notification-worker",
				notificationWorker.Process,
			).WithInterval(notificationWorkerConfig.Interval),
		)

		type SchedulingConfig struct {
			WindowHours int
			Interval    time.Duration
			Disabled    bool
		}

		schedulingConfig := SchedulingConfig{
			WindowHours: 24,
			Interval:    30 * time.Second,
		}
		err = app.Config.ReadSection(ctx, "scheduling", &schedulingConfig)
		if err != nil {
			app.Log.Infof("scheduling config not found, using defaults: %+v", schedulingConfig)
		}

		if !schedulingConfig.Disabled {
			schedulingUseCase := scheduling.NewUseCase(
				app.Log,
				postgresTaskRepo,
				notificationSender,
				time.Duration(schedulingConfig.WindowHours)*time.Hour,
				time.Now,
			)

			schedulingWorker := schedulingworker.NewWorker(schedulingUseCase, app.Log)

			app.AddBackgroundJob(
				webapp.NewBackgroundJob(
					"scheduling-worker",
					schedulingWorker.Process,
				).WithInterval(schedulingConfig.Interval),
			)
		}

		taskListUseCase := tasklist.NewUseCase(
			postgresTaskRepo,
			postgresEventRepo,
			postgresClusterRepo,
			postgresTxProvider,
		)
		tasksService := taskergrpc.NewTasksService(taskListUseCase, app.Log)

		app.RegisterGRPCServices(tasksService)
		app.AddGatewayHandlers(tasksService)

		if app.Env == webapp.DevEnvironment || app.Env == webapp.TestsEnvironment || app.Env == webapp.EvalEnvironment {
			app.AddHTTPHandler("/v1/test/tasks", taskergrpc.NewTestCreateTaskHandler(postgresTaskRepo, time.Now))
			app.AddHTTPHandler("/v1/test/tasks/list", taskergrpc.NewTestListTasksHandler(postgresTaskRepo))
			app.AddHTTPHandler("/v1/test/clusters/list", taskergrpc.NewTestListClusterGenerationDiagnosticsHandler(postgresClusterRepo))
		}

		type AdminConfig struct {
			ApiKey string
		}
		adminConfig := AdminConfig{}
		err = app.Config.ReadSection(ctx, "admin", &adminConfig)
		if err != nil {
			return err
		}

		adminUseCase := admin.NewUseCase(
			postgresTaskRepo,
			postgresModerationRepo,
			postgresClusterRepo,
			postgresEventRepo,
			postgresPromptRepo,
			promptsService,
		)

		app.AddHTTPHandler("GET /admin/users/{userId}/tasks", taskergrpc.NewAdminListTasksHandler(adminUseCase, adminConfig.ApiKey))
		app.AddHTTPHandler("GET /admin/users/{userId}/tasks/{taskId}", taskergrpc.NewAdminGetTaskHandler(adminUseCase, adminConfig.ApiKey))
		app.AddHTTPHandler("PATCH /admin/users/{userId}/tasks/{taskId}", taskergrpc.NewAdminUpdateTaskHandler(adminUseCase, adminConfig.ApiKey))
		app.AddHTTPHandler("POST /admin/users/{userId}/tasks/{taskId}/approve", taskergrpc.NewAdminApproveTaskHandler(adminUseCase, adminConfig.ApiKey))
		app.AddHTTPHandler("GET /admin/users/{userId}/clusters", taskergrpc.NewAdminListClustersHandler(adminUseCase, adminConfig.ApiKey))
		app.AddHTTPHandler("GET /admin/users/{userId}/clusters/{clusterId}/events", taskergrpc.NewAdminListClusterEventsHandler(adminUseCase, adminConfig.ApiKey))
		app.AddHTTPHandler("GET /admin/moderation", taskergrpc.NewAdminListModerationHandler(adminUseCase, adminConfig.ApiKey))
		app.AddHTTPHandler("PUT /admin/moderation/{userId}", taskergrpc.NewAdminSetModerationHandler(adminUseCase, adminConfig.ApiKey))
		app.AddHTTPHandler("GET /admin/prompts", taskergrpc.NewAdminListPromptsHandler(adminUseCase, adminConfig.ApiKey))
		app.AddHTTPHandler("GET /admin/prompts/{promptId}", taskergrpc.NewAdminGetPromptHandler(adminUseCase, adminConfig.ApiKey))
		app.AddHTTPHandler("PUT /admin/prompts/{promptId}", taskergrpc.NewAdminUpdatePromptHandler(adminUseCase, adminConfig.ApiKey))

		if isTestEnv {
			app.AddGRPCUnaryInterceptor(
				token.NewUnaryTokenInterceptor(
					token.NewVerifierStub(),
					app.Log,
					token.InterceptAllMethodsOption,
				),
			)
		} else {
			app.AddGRPCUnaryInterceptor(
				token.NewUnaryTokenInterceptor(
					token.NewGRPCVerifier(authConn),
					app.Log,
					token.InterceptAllMethodsOption,
				),
			)
		}

		return nil
	})
}
