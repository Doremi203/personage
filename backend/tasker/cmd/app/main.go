package main

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/amoguscorp/personage/backend/libs/go/errors"
	"gitlab.com/amoguscorp/personage/backend/libs/go/sqs"
	"gitlab.com/amoguscorp/personage/backend/libs/go/webapp"
	connectoreventsPb "gitlab.com/amoguscorp/personage/backend/tasker/gen/api/sqs/connector-events"
)

type ConnectorEventsProcessor struct {
}

func (c ConnectorEventsProcessor) Process(ctx context.Context, data *connectoreventsPb.Event) error {
	fmt.Printf("Processing connector event: %v\n", data)
	return nil
}

func main() {
	webapp.Run(func(ctx context.Context, app *webapp.App) error {
		sqsConfig := sqs.Config{}
		err := app.Config.ReadSection(ctx, "sqs", &sqsConfig)
		if err != nil {
			return err
		}

		connectorEventsProcessor, err := sqs.NewMessageProcessor[*connectoreventsPb.Event](
			ctx,
			app.Log,
			sqsConfig,
			ConnectorEventsProcessor{},
			2*time.Second,
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
		return nil
	})
}
