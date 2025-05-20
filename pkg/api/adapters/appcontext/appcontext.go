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

	wg sync.WaitGroup
}

func NewAppContext(ctx context.Context) (*AppContext, error) {
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

	// err = appContext.Queue.LoadFromConfig(ctx, &appContext.Config.Queue)
	// if err != nil {
	// 	return nil, fmt.Errorf("%w: %w", ErrInitFailed, err)
	// }

	// sqs queue
	appContext.SqsQueue = sqs_queue.New(&appContext.Config.SqsQueue, appContext.Logger)

	// services
	appContext.Resources = resources.NewService(appContext.Logger)
	appContext.Tasks = tasks.NewService(&appContext.Config.Tasks, appContext.Logger, appContext.SqsQueue)

	return appContext, nil
}

func (a *AppContext) Run(ctx context.Context) error {
	a.Logger.InfoContext(
		ctx,
		"Starting application layer",
		slog.String("name", a.Config.AppName),
		slog.String("environment", a.Config.AppEnv),
		slog.Any("features", a.Config.Features),
	)

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

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.Logger.InfoContext(ctx, "Task processing goroutine started.")

		for {
			select {
			case <-ctx.Done():
				a.Logger.InfoContext(ctx, "Task processing goroutine shutting down due to context cancellation.")
				return
			default:
				processErr := a.Tasks.ProcessNextTask(ctx, func(innerCtx context.Context, task tasks.Task) error {
					a.Logger.InfoContext(innerCtx, "Processing task", "task", task)
					return nil
				})
				if processErr != nil {
					// don't log an error if it's just the context being cancelled during shutdown.
					if !errors.Is(processErr, context.Canceled) && !errors.Is(processErr, context.DeadlineExceeded) {
						a.Logger.ErrorContext(ctx, "Failed to process task", "error", processErr)
					} else {
						// if it's a cancellation error, it might be normal during shutdown.
						a.Logger.InfoContext(ctx, "Task processing cycle interrupted by context", "error", processErr)
					}
				}
			}
		}
	}()

	return nil
}

func (a *AppContext) WaitForShutdown(shutdownCtx context.Context) {
	a.Logger.InfoContext(shutdownCtx, "AppContext: Waiting for services to shut down...")

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		a.wg.Wait()
	}()

	select {
	case <-doneCh:
		a.Logger.InfoContext(shutdownCtx, "AppContext: All services shut down gracefully.")
	case <-shutdownCtx.Done():
		a.Logger.ErrorContext(shutdownCtx, "AppContext: Shutdown timed out waiting for services.", "error", shutdownCtx.Err())
	}
}
