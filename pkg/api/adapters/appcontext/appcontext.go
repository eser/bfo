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
	"github.com/eser/bfo/pkg/api/adapters/repository"
	"github.com/eser/bfo/pkg/api/adapters/sqs_queue"
	"github.com/eser/bfo/pkg/api/business/resources"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

var ErrInitFailed = errors.New("failed to initialize app context")

// AppContext is our composition root for the application.
type AppContext struct {
	Config  *AppConfig
	Logger  *logfx.Logger
	Metrics *metricsfx.MetricsProvider

	DynamoDbStore *dynamodb_store.Store
	SqsQueue      *sqs_queue.Queue

	Repository *repository.Repository
	Resources  *resources.Service
	Tasks      *tasks.Service
}

func NewAppContext() *AppContext {
	return &AppContext{} //nolint:exhaustruct
}

func (a *AppContext) Init(ctx context.Context) error { //nolint:lll
	// ----------------------------------------------------
	// Config
	// ----------------------------------------------------
	cl := configfx.NewConfigManager()

	a.Config = &AppConfig{} //nolint:exhaustruct

	err := cl.LoadDefaults(a.Config)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// ----------------------------------------------------
	// Logger
	// ----------------------------------------------------
	a.Logger, err = logfx.NewLoggerAsDefault(os.Stdout, &a.Config.Log)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	a.Logger.InfoContext(
		ctx,
		"[AppContext] Initialization in progress",
		slog.String("module", "appcontext"),
		slog.String("name", a.Config.AppName),
		slog.String("environment", a.Config.AppEnv),
		slog.Any("features", a.Config.Features),
	)

	// ----------------------------------------------------
	// Metrics
	// ----------------------------------------------------
	a.Logger.DebugContext(
		ctx,
		"[AppContext] Initializing metrics",
		slog.String("module", "appcontext"),
	)

	a.Metrics = metricsfx.NewMetricsProvider()

	err = a.Metrics.RegisterNativeCollectors()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// ----------------------------------------------------
	// DynamoDbStore
	// ----------------------------------------------------
	a.Logger.DebugContext(
		ctx,
		"[AppContext] Initializing dynamodb store",
		slog.String("module", "appcontext"),
	)

	a.DynamoDbStore = dynamodb_store.New(
		&a.Config.DynamoDbStore,
		a.Logger,
	)

	err = a.DynamoDbStore.Init(ctx)
	if err != nil {
		a.Logger.ErrorContext(
			ctx,
			"[AppContext] Failed to initialize dynamodb store",
			slog.String("module", "appcontext"),
			slog.Any("error", err),
		)

		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// ----------------------------------------------------
	// SqsQueue
	// ----------------------------------------------------
	a.Logger.DebugContext(
		ctx,
		"[AppContext] Initializing sqs queue",
		slog.String("module", "appcontext"),
	)

	a.SqsQueue = sqs_queue.New(
		&a.Config.SqsQueue,
		a.Logger,
	)

	taskQueueUrl, err := a.SqsQueue.Init(ctx)
	if err != nil {
		a.Logger.ErrorContext(
			ctx,
			"[AppContext] Failed to initialize sqs queue",
			slog.String("module", "appcontext"),
			slog.Any("error", err),
		)

		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// ----------------------------------------------------
	// Repository
	// ----------------------------------------------------
	a.Logger.DebugContext(
		ctx,
		"[AppContext] Initializing repository",
		slog.String("module", "appcontext"),
	)

	a.Repository = repository.New(
		a.Logger,
		a.DynamoDbStore,
		a.SqsQueue,
	)

	err = a.Repository.Init(ctx)
	if err != nil {
		a.Logger.ErrorContext(
			ctx,
			"[AppContext] Failed to initialize repository",
			slog.String("module", "appcontext"),
			slog.Any("error", err),
		)

		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// // ----------------------------------------------------
	// // Resources
	// // ----------------------------------------------------
	// a.Logger.DebugContext(
	// 	ctx,
	// 	"[AppContext] Initializing resources",
	// 	slog.String("module", "appcontext"),
	// )

	// a.Resources = resources.NewService(
	// 	&a.Config.Resources,
	// 	a.Logger,
	// 	a.Repository,
	// )

	// a.Resources.AddProvider(
	// 	ctx,
	// 	"openai",
	// 	func(config *resources.ConfigResource) resources.Provider {
	// 		return providers.NewOpenAiClient(config, a.Logger)
	// 	},
	// )

	// a.Resources.AddProvider(
	// 	ctx,
	// 	"echo",
	// 	func(config *resources.ConfigResource) resources.Provider {
	// 		return providers.NewEchoClient(config, a.Logger)
	// 	},
	// )

	// err = a.Resources.Init(ctx)
	// if err != nil {
	// 	a.Logger.ErrorContext(
	// 		ctx,
	// 		"[AppContext] Failed to initialize resources",
	// 		slog.String("module", "appcontext"),
	// 		slog.Any("error", err),
	// 	)

	// 	return fmt.Errorf("%w: %w", ErrInitFailed, err)
	// }

	// ----------------------------------------------------
	// Tasks
	// ----------------------------------------------------
	a.Logger.DebugContext(
		ctx,
		"[AppContext] Initializing tasks",
		slog.String("module", "appcontext"),
	)

	a.Tasks = tasks.NewService(
		&a.Config.Tasks,
		a.Logger,
		a.Repository,
	)

	err = a.Tasks.Init(*taskQueueUrl)
	if err != nil {
		a.Logger.ErrorContext(
			ctx,
			"[AppContext] Failed to initialize tasks",
			slog.String("module", "appcontext"),
			slog.Any("error", err),
		)

		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	return nil
}

func (a *AppContext) Tick(ctx context.Context) error {
	processFn := func(innerCtx context.Context, task *tasks.Task) (tasks.TaskResult, error) {
		taskResult, err := a.Resources.TryProcessTask(innerCtx, task)

		if err != nil {
			return tasks.TaskResultSystemPermanentlyFailed, fmt.Errorf("failed to process task '%s': %w", task.Id, err)
		}

		return taskResult, nil
	}

	err := a.Tasks.PickUpNextAvailableTask(ctx, processFn)

	return err
}
