package appcontext

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

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

	// ExternalTool *external_tool.ExternalTool

	Resources *resources.Service
	Tasks     *tasks.Service
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

	// // external tool
	// appContext.ExternalTool = external_tool.New(appContext.Config.Externals.ExternalTool)

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

	go func() {
		for {
			select {
			case <-ctx.Done():
				a.Logger.Info("Shutting down task processing")
				return
			default:
				err := a.Tasks.ProcessNextTask(ctx, func(ctx context.Context, task tasks.Task) error {
					a.Logger.InfoContext(ctx, "Processing task", "task", task)
					return nil
				})
				if err != nil {
					a.Logger.ErrorContext(ctx, "Failed to process task", "error", err)
				}
			}
		}
	}()

	return nil
}
