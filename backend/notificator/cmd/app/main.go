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
	"github.com/Doremi203/personage/backend/notificator/internal/grpc"
	notificationpostgres "github.com/Doremi203/personage/backend/notificator/internal/repo/notification/postgres"
	pushpostgres "github.com/Doremi203/personage/backend/notificator/internal/repo/push/postgres"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase"
	sqspush "github.com/Doremi203/personage/backend/notificator/internal/worker"
	"github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5/pgxpool"
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

		pushSubscriptionUseCase := usecase.NewPushSubscription(
			pushRepo,
		)
		pushSubscriptionService := grpc.NewPushSubscriptionService(
			pushSubscriptionUseCase,
			app.Log,
		)

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

		pushAdminService := grpc.NewAdminService(
			pushRepo,
			pushSenderUseCase,
			app.Log,
		)

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
			),
			5*time.Second,
			5,
		)
		if err != nil {
			return errors.WrapFail(err, "create notification messages processor")
		}
		app.AddBackgroundJob(
			webapp.NewBackgroundJob(
				"sqs-notifications-worker",
				notificationMessagesProcessor.ProcessMessages,
			),
		)

		notificationsUseCase := usecase.NewNotifications(notificationRepo)
		notificationsService := grpc.NewNotificationsService(
			notificationsUseCase,
			app.Log,
		)

		app.AddAPIKeyProtectedEndpoints(pushpb.Admin_SendPushV1_FullMethodName)
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
		app.RegisterGRPCServices(
			pushSubscriptionService,
			pushAdminService,
			notificationsService,
		)
		app.AddGatewayHandlers(
			pushSubscriptionService,
			pushAdminService,
			notificationsService,
		)

		return nil
	})
}
