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
	// Queue   *queuefx.Registry
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

	// // queue
	// appContext.Queue = queuefx.NewRegistry(appContext.Logger)

	// sqs queue
	appContext.SqsQueue = sqs_queue.New(&appContext.Config.SqsQueue, appContext.Logger)

	// dynamodb store
	appContext.DynamoDbStore = dynamodb_store.New(&appContext.Config.DynamoDbStore, appContext.Logger)

	// resources
	appContext.Resources = resources.NewService(appContext.Logger)
	appContext.Resources.AddProvider("mock", func(resourceDef *resources.ResourceDef) resources.Provider {
		return &providers.MockClient{}
	})
	appContext.Resources.AddProvider("openai", func(resourceDef *resources.ResourceDef) resources.Provider {
		return providers.NewOpenAiClient(resourceDef)
	})

	// tasks
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

	err = a.Resources.LoadResourcesFromFile("./etc/resources.json")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// tasks

	err = a.Tasks.Init(*taskQueueURL)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// err = a.DynamoDbStore.PutWorkerPoolState(ctx, &worker_pools.WorkerPoolState{
	// 	PoolId: "default",
	// 	State:  []byte("non-default"),
	// })
	// if err != nil {
	// 	return fmt.Errorf("%w: %w", ErrInitFailed, err)
	// }

	// state, err := a.DynamoDbStore.GetWorkerPoolState(ctx, "default")
	// a.Logger.InfoContext(ctx, "Worker pool state", "state", string(state.State))
	// if err != nil {
	// 	return fmt.Errorf("%w: %w", ErrInitFailed, err)
	// }

	return nil
}

func (a *AppContext) Tick(ctx context.Context) error {
	err := a.Tasks.ProcessNextTask(ctx, func(innerCtx context.Context, task tasks.Task) error {
		a.Logger.InfoContext(innerCtx, "Processing task", "task", task)
		return nil
	})

	return err
}
