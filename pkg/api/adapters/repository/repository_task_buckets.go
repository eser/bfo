package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/eser/bfo/pkg/api/business/tasks"
)

const (
	TaskBucketTableName = "task_buckets"
	TaskBucketTablePK   = "id"
)

var (
	ErrFailedToGetTaskBucket = fmt.Errorf("failed to get task bucket")
)

func (r *Repository) ListTaskBuckets(ctx context.Context) ([]*tasks.TaskBucket, error) {
	r.logger.DebugContext(
		ctx,
		"[Repository] Listing task buckets",
		slog.String("module", "repository"),
	)

	var items []*tasks.TaskBucket
	err := r.dynamoDbStore.ListItems(ctx, TaskBucketTableName, &items)
	if err != nil {
		r.logger.ErrorContext(
			ctx,
			"[Repository] Failed to list task buckets",
			slog.String("module", "repository"),
			slog.Any("error", err),
		)

		return nil, fmt.Errorf("failed to list task buckets: %w", err)
	}

	return items, nil
}

func (r *Repository) GetTaskBucket(ctx context.Context, id string) (*tasks.TaskBucket, error) {
	r.logger.DebugContext(
		ctx,
		"[Repository] Getting task bucket",
		slog.String("module", "repository"),
		slog.String("id", id),
	)

	var item tasks.TaskBucket
	found, err := r.dynamoDbStore.GetItem(
		ctx,
		TaskBucketTableName,
		TaskBucketTablePK,
		id,
		&item,
	)
	if err != nil {
		r.logger.ErrorContext(
			ctx,
			"[Repository] Failed to get task bucket",
			slog.String("module", "repository"),
			slog.String("id", id),
			slog.Any("error", err),
		)

		return nil, fmt.Errorf("%w: %w", ErrFailedToGetTaskBucket, err)
	}

	if !found {
		return nil, nil
	}

	return &item, nil
}

func (r *Repository) UpsertTaskBucket(ctx context.Context, item *tasks.TaskBucket) error {
	r.logger.DebugContext(
		ctx,
		"[Repository] Upserting task bucket",
		slog.String("module", "repository"),
		slog.String("id", item.Id),
	)

	err := r.dynamoDbStore.UpsertItem(ctx, TaskBucketTableName, item)
	if err != nil {
		r.logger.ErrorContext(
			ctx,
			"[Repository] Failed to upsert task bucket",
			slog.String("module", "repository"),
			slog.String("id", item.Id),
			slog.Any("error", err),
		)

		return fmt.Errorf("failed to upsert task bucket: %w", err)
	}

	return nil
}

func (r *Repository) DeleteTaskBucket(ctx context.Context, id string) error {
	r.logger.DebugContext(
		ctx,
		"[Repository] Deleting task bucket",
		slog.String("module", "repository"),
		slog.String("id", id),
	)

	err := r.dynamoDbStore.DeleteItem(
		ctx,
		TaskBucketTableName,
		TaskBucketTablePK,
		id,
	)
	if err != nil {
		r.logger.ErrorContext(
			ctx,
			"[Repository] Failed to delete task bucket",
			slog.String("module", "repository"),
			slog.String("id", id),
			slog.Any("error", err),
		)

		return fmt.Errorf("failed to delete task bucket: %w", err)
	}

	return nil
}
