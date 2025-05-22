package storage

import (
	"context"
	"fmt"

	"github.com/eser/bfo/pkg/api/business/resources"
)

func (r *Repository) GetResourceInstanceState(ctx context.Context, resourceInstanceId string) (*resources.ResourceInstanceState, error) {
	r.logger.DebugContext(ctx, "[DynamoDbStore] Getting resource instance state", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", ResourceStateTableName)

	var state resources.ResourceInstanceState
	found, err := r.dynamoDbStore.GetItem(ctx, ResourceStateTableName, "ResourceInstanceId", resourceInstanceId, &state)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource instance state: %w", err)
	}

	if !found {
		r.logger.DebugContext(ctx, "[DynamoDbStore] Resource instance state not found", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", ResourceStateTableName)
		// Consider returning a business-layer specific error e.g., resources.ErrStateNotFound
		return nil, nil // Or return an error indicating not found
	}

	return &state, nil
}

func (r *Repository) PutResourceInstanceState(ctx context.Context, state *resources.ResourceInstanceState) error {
	r.logger.DebugContext(ctx, "[DynamoDbStore] Putting resource instance state", "module", "dynamodb_store", "resourceInstanceId", state.ResourceInstanceId, "tableName", ResourceStateTableName)

	err := r.dynamoDbStore.PutItem(ctx, ResourceStateTableName, state)
	if err != nil {
		return fmt.Errorf("failed to put resource instance state: %w", err)
	}

	r.logger.InfoContext(ctx, "[DynamoDbStore] Successfully put resource instance state", "module", "dynamodb_store", "resourceInstanceId", state.ResourceInstanceId, "tableName", ResourceStateTableName)

	return nil
}

func (r *Repository) DeleteResourceInstanceState(ctx context.Context, resourceInstanceId string) error {
	r.logger.DebugContext(ctx, "[DynamoDbStore] Deleting resource instance state", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", ResourceStateTableName)

	err := r.dynamoDbStore.DeleteItem(ctx, ResourceStateTableName, "ResourceInstanceId", resourceInstanceId)
	if err != nil {
		return fmt.Errorf("failed to delete resource instance state: %w", err)
	}

	r.logger.InfoContext(ctx, "[DynamoDbStore] Successfully deleted resource instance state", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", ResourceStateTableName)

	return nil
}
