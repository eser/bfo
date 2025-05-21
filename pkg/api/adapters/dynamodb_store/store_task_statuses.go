package dynamodb_store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

var _ tasks.TaskStatusStore = (*Store)(nil)

// GetTaskStatus retrieves the status of a task by its ID.
func (s *Store) GetTaskStatus(ctx context.Context, taskId string) (*tasks.TaskStatus, error) {
	s.logger.DebugContext(ctx, "[DynamoDbStore] Getting task status", "module", "dynamodb_store", "taskId", taskId, "tableName", TaskStatusTableName)

	key := map[string]types.AttributeValue{
		"TaskId": &types.AttributeValueMemberS{Value: taskId},
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(TaskStatusTableName),
		Key:       key,
	}

	result, err := s.client.GetItem(ctx, input)
	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to get item from DynamoDb for task status", "module", "dynamodb_store", "taskId", taskId, "tableName", TaskStatusTableName, "error", err)
		return nil, fmt.Errorf("dynamodb.GetItem failed for task status: %w", err)
	}

	if result.Item == nil {
		s.logger.DebugContext(ctx, "[DynamoDbStore] Task status not found", "module", "dynamodb_store", "taskId", taskId, "tableName", TaskStatusTableName)
		return nil, tasks.ErrTaskStatusNotFound
	}

	var status tasks.TaskStatus
	err = attributevalue.UnmarshalMap(result.Item, &status)
	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to unmarshal item from DynamoDb for task status", "module", "dynamodb_store", "taskId", taskId, "item", result.Item, "error", err)
		return nil, fmt.Errorf("attributevalue.UnmarshalMap failed for task status: %w", err)
	}

	return &status, nil
}

// PutTaskStatus stores the entire task status.
func (s *Store) PutTaskStatus(ctx context.Context, status *tasks.TaskStatus) error {
	s.logger.DebugContext(ctx, "[DynamoDbStore] Putting task status", "module", "dynamodb_store", "taskId", status.TaskId, "tableName", TaskStatusTableName)

	item, err := attributevalue.MarshalMap(status)
	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to marshal task status for PutItem", "module", "dynamodb_store", "taskId", status.TaskId, "error", err)
		return fmt.Errorf("attributevalue.MarshalMap failed for task status: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(TaskStatusTableName),
		Item:      item,
	}

	_, err = s.client.PutItem(ctx, input)
	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to put item to DynamoDb for task status", "module", "dynamodb_store", "taskId", status.TaskId, "tableName", TaskStatusTableName, "error", err)
		return fmt.Errorf("dynamodb.PutItem failed for task status: %w", err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] Successfully put task status", "module", "dynamodb_store", "taskId", status.TaskId, "tableName", TaskStatusTableName)
	return nil
}

// UpdateTaskStatus partially updates a task's status.
func (s *Store) UpdateTaskStatus(ctx context.Context, taskId string, updates map[string]any) error {
	s.logger.DebugContext(ctx, "[DynamoDbStore] Updating task status", "module", "dynamodb_store", "taskId", taskId, "tableName", TaskStatusTableName, "updates", updates)

	key := map[string]types.AttributeValue{
		"TaskId": &types.AttributeValueMemberS{Value: taskId},
	}

	var updateExpression strings.Builder
	updateExpression.WriteString("SET ")
	expressionAttributeValues := make(map[string]types.AttributeValue)
	expressionAttributeNames := make(map[string]string) // For attribute names that are reserved keywords
	first := true

	// Ensure UpdatedAt is always part of the update
	if _, ok := updates["UpdatedAt"]; !ok {
		updates["UpdatedAt"] = time.Now().Unix()
	}

	i := 0
	for k, v := range updates {
		if !first {
			updateExpression.WriteString(", ")
		}
		first = false

		// Use placeholders for attribute names and values to avoid issues with reserved words and injection
		attrNamePlaceholder := fmt.Sprintf("#attr%d", i)
		attrValPlaceholder := fmt.Sprintf(":val%d", i)

		expressionAttributeNames[attrNamePlaceholder] = k
		updateExpression.WriteString(fmt.Sprintf("%s = %s", attrNamePlaceholder, attrValPlaceholder))

		av, err := attributevalue.Marshal(v)
		if err != nil {
			s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to marshal attribute value for update", "module", "dynamodb_store", "key", k, "value", v, "error", err)
			return fmt.Errorf("failed to marshal attribute '%s': %w", k, err)
		}
		expressionAttributeValues[attrValPlaceholder] = av
		i++
	}

	input := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(TaskStatusTableName),
		Key:                       key,
		UpdateExpression:          aws.String(updateExpression.String()),
		ExpressionAttributeNames:  expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
		ReturnValues:              types.ReturnValueUpdatedNew, // Or other values as needed
	}

	// Note on optimistic locking for UpdateTaskStatus:
	// If true optimistic locking is needed (check current version before updating),
	// the calling business logic should fetch the current TaskStatus (which includes Version),
	// prepare the updates map including the new Version (e.g., current Version + 1),
	// and then this method would need to be extended to accept an `expectedVersion` parameter.
	// This `expectedVersion` would be used to add a ConditionExpression to the UpdateItemInput:
	//   input.ConditionExpression = aws.String("#v = :expectedV")
	//   expressionAttributeNames["#v"] = "Version"
	//   expressionAttributeValues[":expectedV"], _ = attributevalue.Marshal(expectedVersion)
	// The current implementation directly sets the Version if it's in the `updates` map,
	// without a conditional check on its previous value.

	_, err := s.client.UpdateItem(ctx, input)
	if err != nil {
		var condCheckFailed *types.ConditionalCheckFailedException
		if errors.As(err, &condCheckFailed) {
			s.logger.WarnContext(ctx, "[DynamoDbStore] Conditional check failed during UpdateTaskStatus (optimistic lock failure?)", "module", "dynamodb_store", "taskId", taskId, "error", err)
			// Consider returning a specific business error for optimistic lock failures, e.g., tasks.ErrOptimisticLock
			return fmt.Errorf("optimistic lock failed for task %s: %w", taskId, err)
		}
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to update item in DynamoDb for task status", "module", "dynamodb_store", "taskId", taskId, "tableName", TaskStatusTableName, "error", err)
		return fmt.Errorf("dynamodb.UpdateItem failed for task status: %w", err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] Successfully updated task status", "module", "dynamodb_store", "taskId", taskId, "tableName", TaskStatusTableName)
	return nil
}
