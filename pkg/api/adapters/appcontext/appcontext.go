package appcontext

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/eser/ajan/configfx"
	"github.com/eser/ajan/logfx"
	"github.com/eser/ajan/metricsfx"

	"github.com/eser/bfo/pkg/api/adapters/sqs_queue"
	"github.com/eser/bfo/pkg/api/business/resources"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

var ErrInitFailed = errors.New("failed to initialize app context")

type AppContext struct {
	Config  *AppConfig
	Logger  *logfx.Logger
	Metrics *metricsfx.MetricsProvider
	// Queue   *queuefx.Registry
	SqsQueue *sqs_queue.Queue

	Resources *resources.Service
	Tasks     *tasks.Service

	WaitGroup sync.WaitGroup
}

func NewAppContext() (*AppContext, error) {
	appContext := &AppContext{} //nolint:exhaustruct

	// config
	cl := configfx.NewConfigManager()

	appContext.Config = &AppConfig{} //nolint:exhaustruct

	err := cl.LoadDefaults(appContext.Config)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// logger
	appContext.Logger, err = logfx.NewLoggerAsDefault(os.Stdout, &appContext.Config.Log)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// metrics
	appContext.Metrics = metricsfx.NewMetricsProvider()

	err = appContext.Metrics.RegisterNativeCollectors()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// // queue
	// appContext.Queue = queuefx.NewRegistry(appContext.Logger)

	// sqs queue
	appContext.SqsQueue = sqs_queue.New(&appContext.Config.SqsQueue, appContext.Logger)

	// services
	appContext.Resources = resources.NewService(appContext.Logger)
	appContext.Tasks = tasks.NewService(&appContext.Config.Tasks, appContext.Logger, appContext.SqsQueue)

	return appContext, nil
}

func (a *AppContext) Init(ctx context.Context) error {
	a.Logger.InfoContext(
		ctx,
		"Starting application layer",
		slog.String("name", a.Config.AppName),
		slog.String("environment", a.Config.AppEnv),
		slog.Any("features", a.Config.Features),
	)

	// queue
	// err = appContext.Queue.LoadFromConfig(ctx, &appContext.Config.Queue)
	// if err != nil {
	// 	return nil, fmt.Errorf("%w: %w", ErrInitFailed, err)
	// }

	// sqs queue

	a.SqsQueue.Init(ctx)

	taskQueueURL, err := a.SqsQueue.CreateQueueIfNotExists(ctx, a.Config.SqsQueue.TaskQueueName)
	if err != nil {
		panic(err)
	}

	a.Logger.Info("Task Queue URL", "taskQueueURL", *taskQueueURL)

	// resources

	err = a.Resources.LoadResourcesFromFile("./etc/resources.json")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// tasks

	err = a.Tasks.Init(*taskQueueURL)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	return nil
}

func (a *AppContext) WaitForShutdown(shutdownCtx context.Context) {
	a.Logger.InfoContext(shutdownCtx, "AppContext: Waiting for services to shut down...")

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		a.WaitGroup.Wait()
	}()

	select {
	case <-doneCh:
		a.Logger.InfoContext(shutdownCtx, "AppContext: All services shut down gracefully.")
	case <-shutdownCtx.Done():
		a.Logger.ErrorContext(shutdownCtx, "AppContext: Shutdown timed out waiting for services.", "error", shutdownCtx.Err())
	}
}
