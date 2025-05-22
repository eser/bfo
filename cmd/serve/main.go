package main

import (
	"context"
	"log/slog"

	"github.com/eser/ajan/processfx"
	"github.com/eser/bfo/pkg/api/adapters/appcontext"
	"github.com/eser/bfo/pkg/api/adapters/http"
)

func main() {
	appContext := appcontext.NewAppContext()
	baseCtx := context.Background()

	err := appContext.Init(baseCtx)
	if err != nil {
		panic(err)
	}

	process := processfx.New(baseCtx, appContext.Logger)

	process.StartGoroutine("http-server", func(ctx context.Context) error {
		cleanup, err := http.Run(
			process.Ctx,
			appContext,
		)

		if err != nil {
			appContext.Logger.ErrorContext(
				ctx,
				"[Main] HTTP server run failed",
				slog.String("module", "main"),
				slog.Any("error", err))
		}

		defer cleanup()

		<-ctx.Done()

		return nil
	})

	process.StartGoroutine("event-loop", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil

			default:
				err := appContext.Tick(ctx)

				if err != nil {
					appContext.Logger.ErrorContext(
						ctx,
						"[Main] Error during tick",
						slog.String("module", "main"),
						slog.Any("error", err))

					return err
				}
			}
		}
	})

	process.Wait()
	process.Shutdown()
}
