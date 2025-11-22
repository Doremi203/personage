package main

import (
	"context"

	"gitlab.com/amoguscorp/personage/backend/libs/go/webapp"
)

func main() {
	webapp.Run(func(ctx context.Context, app *webapp.App) error {
		return nil
	})
}
