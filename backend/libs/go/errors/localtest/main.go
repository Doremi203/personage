package main

import (
	"log/slog"
	"os"

	"gitlab.com/amoguscorp/personage/backend/libs/go/errors"
)

func main() {
	slogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	logger := errors.Logger(slogger)

	myErr := errors.Wrapf(
		errors.Wrapf(
			errors.Errorf(
				"errorIs custom error %v and %v",
				errors.Token("token1", "ival1"),
				errors.Token("token2", "ival2"),
			),
			"internal %v",
			errors.Token("token1", "val1"),
		),
		"do something with %v",
		errors.Token("token1", "val1"),
	)
	if myErr != nil {
		slogger.Error(
			myErr.Error(),
			"token1##", "val1",
			"token1#", "val1",
			"token1", "ival1",
			"token2", "ival2",
		)
		logger.Error(myErr)
		logger.Infof("check test %v", errors.Token("check", "value"))
	}
}
