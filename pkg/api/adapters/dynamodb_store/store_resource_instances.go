package dynamodb_store

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/eser/bfo/pkg/api/business/resources"
)

var _ resources.ResourceStateStore = (*Store)(nil)

func (s *Store) GetResourceInstanceState(ctx context.Context, resourceInstanceId string) (*resources.ResourceInstanceState, error) {
	s.logger.DebugContext(ctx, "[DynamoDbStore] Getting resource instance state", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", ResourceStateTableName)

	key := map[string]types.AttributeValue{
		"ResourceInstanceId": &types.AttributeValueMemberS{Value: resourceInstanceId},
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(ResourceStateTableName),
		Key:       key,
	}

	result, err := s.client.GetItem(ctx, input)
	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to get item from DynamoDb", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", ResourceStateTableName, "error", err)
		return nil, fmt.Errorf("dynamodb.GetItem failed: %w", err)
	}

	if result.Item == nil {
		s.logger.DebugContext(ctx, "[DynamoDbStore] Worker pool state not found", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", ResourceStateTableName)
		// Consider returning a business-layer specific error e.g., resources.ErrStateNotFound
		return nil, nil // Or return an error indicating not found
	}

	var state resources.ResourceInstanceState

	err = attributevalue.UnmarshalMap(result.Item, &state)

	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to unmarshal item from DynamoDb", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "item", result.Item, "error", err)
		return nil, fmt.Errorf("attributevalue.UnmarshalMap failed: %w", err)
	}

	return &state, nil
}

func (s *Store) PutResourceInstanceState(ctx context.Context, state *resources.ResourceInstanceState) error {
	s.logger.DebugContext(ctx, "[DynamoDbStore] Putting resource instance state", "module", "dynamodb_store", "resourceInstanceId", state.ResourceInstanceId, "tableName", ResourceStateTableName)

	item, err := attributevalue.MarshalMap(state)

	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to marshal state for PutItem", "module", "dynamodb_store", "resourceInstanceId", state.ResourceInstanceId, "error", err)
		return fmt.Errorf("attributevalue.MarshalMap failed: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(ResourceStateTableName),
		Item:      item,
	}

	_, err = s.client.PutItem(ctx, input)

	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to put item to DynamoDb", "module", "dynamodb_store", "resourceInstanceId", state.ResourceInstanceId, "tableName", ResourceStateTableName, "error", err)
		return fmt.Errorf("dynamodb.PutItem failed: %w", err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] Successfully put resource instance state", "module", "dynamodb_store", "resourceInstanceId", state.ResourceInstanceId, "tableName", ResourceStateTableName)

	return nil
}

func (s *Store) DeleteResourceInstanceState(ctx context.Context, resourceInstanceId string) error {
	s.logger.DebugContext(ctx, "[DynamoDbStore] Deleting resource instance state", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", ResourceStateTableName)

	key := map[string]types.AttributeValue{
		"ResourceInstanceId": &types.AttributeValueMemberS{Value: resourceInstanceId},
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(ResourceStateTableName),
		Key:       key,
	}

	_, err := s.client.DeleteItem(ctx, input)

	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to delete item from DynamoDb", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", ResourceStateTableName, "error", err)
		return fmt.Errorf("dynamodb.DeleteItem failed: %w", err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] Successfully deleted resource instance state", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", ResourceStateTableName)

	return nil
}
