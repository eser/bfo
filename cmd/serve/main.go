package main

import (
	"context"

	"github.com/eser/ajan/processfx"
	"github.com/eser/bfo/pkg/api/adapters/appcontext"
	"github.com/eser/bfo/pkg/api/adapters/http"
)

func main() {
	appContext, err := appcontext.NewAppContext()
	if err != nil {
		panic(err)
	}

	baseCtx := context.Background()

	err = appContext.Init(baseCtx)
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
			appContext.Logger.ErrorContext(ctx, "[Main] HTTP server run failed", "module", "main", "error", err)
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
					appContext.Logger.ErrorContext(ctx, "[Main] Error during tick", "module", "main", "error", err)

					return err
				}
			}
		}
	})

	process.Wait()
	process.Shutdown()
}
