package http

import (
	"context"

	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/httpfx/middlewares"
	"github.com/eser/ajan/httpfx/modules/healthcheck"
	"github.com/eser/ajan/httpfx/modules/openapi"
	"github.com/eser/ajan/httpfx/modules/profiling"
	"github.com/eser/bfo/pkg/api/adapters/appcontext"
)

func Run(
	ctx context.Context,
	appContext *appcontext.AppContext,
) (func(), error) {
	routes := httpfx.NewRouter("/")
	httpService := httpfx.NewHttpService(
		&appContext.Config.Http,
		routes,
		appContext.Metrics,
		appContext.Logger,
	)

	// http middlewares
	routes.Use(middlewares.ErrorHandlerMiddleware())
	routes.Use(middlewares.ResolveAddressMiddleware())
	routes.Use(middlewares.ResponseTimeMiddleware())
	routes.Use(middlewares.CorrelationIdMiddleware())
	routes.Use(middlewares.CorsMiddleware())
	routes.Use(middlewares.MetricsMiddleware(httpService.InnerMetrics))

	// http modules
	healthcheck.RegisterHttpRoutes(routes, &appContext.Config.Http)
	openapi.RegisterHttpRoutes(routes, &appContext.Config.Http)
	profiling.RegisterHttpRoutes(routes, &appContext.Config.Http)

	// http routes
	RegisterHttpRoutesForTaskBuckets(routes, appContext) //nolint:contextcheck
	RegisterHttpRoutesForTasks(routes, appContext)       //nolint:contextcheck

	// run
	return httpService.Start(ctx)
}
