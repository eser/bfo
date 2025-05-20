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

	"github.com/eser/bfo/pkg/api/adapters/dynamodb_store"
	"github.com/eser/bfo/pkg/api/adapters/providers"
	"github.com/eser/bfo/pkg/api/adapters/sqs_queue"
	"github.com/eser/bfo/pkg/api/business/resources"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

var ErrInitFailed = errors.New("failed to initialize app context")

type AppContext struct {
	Config  *AppConfig
	Logger  *logfx.Logger
	Metrics *metricsfx.MetricsProvider

	SqsQueue      *sqs_queue.Queue
	DynamoDbStore *dynamodb_store.Store

	Resources *resources.Service
	Tasks     *tasks.Service
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

	// sqs queue
	appContext.SqsQueue = sqs_queue.New(&appContext.Config.SqsQueue, appContext.Logger)

	// dynamodb store
	appContext.DynamoDbStore = dynamodb_store.New(&appContext.Config.DynamoDbStore, appContext.Logger)

	// resources
	appContext.Resources = resources.NewService(&appContext.Config.Resources, appContext.Logger, appContext.DynamoDbStore) // Use consolidated store
	appContext.Resources.AddProvider("openai", func(config *resources.ConfigResource) resources.Provider {
		return providers.NewOpenAiClient(config, appContext.Logger)
	})
	appContext.Resources.AddProvider("echo", func(config *resources.ConfigResource) resources.Provider {
		return providers.NewEchoClient(config, appContext.Logger)
	})

	// tasks
	appContext.Tasks = tasks.NewService(&appContext.Config.Tasks, appContext.Logger, appContext.SqsQueue, appContext.DynamoDbStore)

	return appContext, nil
}

func (a *AppContext) Init(ctx context.Context) error {
	a.Logger.InfoContext(
		ctx,
		"[AppContext] Starting application layer",
		slog.String("module", "appcontext"),
		slog.String("name", a.Config.AppName),
		slog.String("environment", a.Config.AppEnv),
		slog.Any("features", a.Config.Features),
	)

	// dynamodb store

	err := a.DynamoDbStore.Init(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// sqs queue

	taskQueueURL, err := a.SqsQueue.Init(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// resources

	err = a.Resources.Init()
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

func (a *AppContext) Tick(ctx context.Context) error {
	fn := func(innerCtx context.Context, task tasks.Task) (tasks.TaskResult, error) {
		taskResult, err := a.Resources.ProcessTask(innerCtx, task)

		if err != nil {
			return tasks.TaskResultSystemPermanentlyFailed, fmt.Errorf("failed to process task '%s': %w", task.Id, err)
		}

		return taskResult, nil
	}

	err := a.Tasks.ProcessNextTask(ctx, fn)

	return err
}
