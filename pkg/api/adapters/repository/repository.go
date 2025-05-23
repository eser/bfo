package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/eser/ajan/logfx"
	"github.com/eser/bfo/pkg/api/adapters/dynamodb_store"
	"github.com/eser/bfo/pkg/api/adapters/sqs_queue"
)

const (
	TaskStatusTableName = "task_statuses"
	TaskStatusTablePK   = "TaskId"

	ResourceStateTableName = "resource_states"
	ResourceStateTablePK   = "ResourceInstanceId"
)

type Repository struct {
	logger        *logfx.Logger
	dynamoDbStore *dynamodb_store.Store
	sqsQueue      *sqs_queue.Queue
}

func New(logger *logfx.Logger, dynamoDbStore *dynamodb_store.Store, sqsQueue *sqs_queue.Queue) *Repository {
	return &Repository{
		logger:        logger,
		dynamoDbStore: dynamoDbStore,
		sqsQueue:      sqsQueue,
	}
}

func (r *Repository) ensureDynamoDbExists(ctx context.Context, tableName string, tablePK string) error {
	if err := r.dynamoDbStore.EnsureTableExists(ctx, tableName, tablePK); err != nil {
		r.logger.ErrorContext(
			ctx,
			"[Repository] Failed to ensure DynamoDb table exists for resource instances",
			slog.String("module", "dynamodb_store"),
			slog.String("tableName", tableName),
			slog.Any("error", err),
		)

		return fmt.Errorf("failed to ensure DynamoDb table %s exists: %w", tableName, err)
	}

	return nil
}

func (r *Repository) Init(ctx context.Context) error {
	// if err := r.ensureDynamoDbExists(ctx, TaskStatusTableName, TaskStatusTablePK); err != nil {
	// 	return err
	// }

	// if err := r.ensureDynamoDbExists(ctx, ResourceStateTableName, ResourceStateTablePK); err != nil {
	// 	return err
	// }

	if err := r.ensureDynamoDbExists(ctx, TaskBucketTableName, TaskBucketTablePK); err != nil {
		return err
	}

	return nil
}
