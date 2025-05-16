package main

import (
	"context"
	"time"

	"github.com/eser/bfo/pkg/api/adapters/appcontext"
	"github.com/eser/bfo/pkg/api/adapters/http"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	appContext, err := appcontext.NewAppContext(ctx)
	if err != nil {
		panic(err)
	}

	err = appContext.Run(ctx)
	if err != nil {
		panic(err)
	}

	httpRunErr := http.Run(
		ctx,
		cancel,
		appContext,
	)
	if httpRunErr != nil {
		appContext.Logger.ErrorContext(ctx, "HTTP server run failed", "error", httpRunErr)
	}

	shutdownGracePeriod := 30 * time.Second
	finalShutdownCtx, finalShutdownCancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer finalShutdownCancel()

	appContext.WaitForShutdown(finalShutdownCtx)
}
