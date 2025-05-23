package repository

import (
	"context"
	"fmt"

	"github.com/eser/bfo/pkg/api/business/tasks"
)

// GetTaskStatus retrieves the status of a task by its ID.
func (r *Repository) GetTaskStatus(ctx context.Context, taskId string) (*tasks.TaskStatus, error) {
	r.logger.DebugContext(ctx, "[DynamoDbStore] Getting task status", "module", "dynamodb_store", "taskId", taskId, "tableName", TaskStatusTableName)

	var status tasks.TaskStatus
	found, err := r.dynamoDbStore.GetItem(ctx, TaskStatusTableName, "TaskId", taskId, &status)
	if err != nil {
		// Specific error logging is handled by the generic GetItem method.
		// We can add more context here if needed or just return the wrapped error.
		return nil, fmt.Errorf("failed to get task status: %w", err)
	}

	if !found {
		r.logger.DebugContext(ctx, "[DynamoDbStore] Task status not found", "module", "dynamodb_store", "taskId", taskId, "tableName", TaskStatusTableName)
		return nil, tasks.ErrTaskStatusNotFound
	}

	return &status, nil
}

// PutTaskStatus stores the entire task status.
func (r *Repository) PutTaskStatus(ctx context.Context, status *tasks.TaskStatus) error {
	r.logger.DebugContext(ctx, "[DynamoDbStore] Putting task status", "module", "dynamodb_store", "taskId", status.TaskId, "tableName", TaskStatusTableName)

	err := r.dynamoDbStore.UpsertItem(ctx, TaskStatusTableName, status)
	if err != nil {
		// Specific error logging is handled by the generic PutItem method.
		return fmt.Errorf("failed to put task status: %w", err)
	}

	r.logger.InfoContext(ctx, "[DynamoDbStore] Successfully put task status", "module", "dynamodb_store", "taskId", status.TaskId, "tableName", TaskStatusTableName)
	return nil
}

// // UpdateTaskStatus partially updates a task's status.
func (r *Repository) UpdateTaskStatus(ctx context.Context, taskId string, updates map[string]any) error {
	r.logger.DebugContext(ctx, "[DynamoDbStore] Updating task status", "module", "dynamodb_store", "taskId", taskId, "tableName", TaskStatusTableName, "updates", updates)

	// 	key := map[string]types.AttributeValue{
	// 		"TaskId": &types.AttributeValueMemberS{Value: taskId},
	// 	}

	// 	var updateExpression strings.Builder
	// 	updateExpression.WriteString("SET ")
	// 	expressionAttributeValues := make(map[string]types.AttributeValue)
	// 	expressionAttributeNames := make(map[string]string) // For attribute names that are reserved keywords
	// 	first := true

	// 	// Ensure UpdatedAt is always part of the update
	// 	if _, ok := updates["UpdatedAt"]; !ok {
	// 		updates["UpdatedAt"] = time.Now()
	// 	}

	// 	i := 0
	// 	for k, v := range updates {
	// 		if !first {
	// 			updateExpression.WriteString(", ")
	// 		}
	// 		first = false

	// 		// Use placeholders for attribute names and values to avoid issues with reserved words and injection
	// 		attrNamePlaceholder := fmt.Sprintf("#attr%d", i)
	// 		attrValPlaceholder := fmt.Sprintf(":val%d", i)

	// 		expressionAttributeNames[attrNamePlaceholder] = k
	// 		updateExpression.WriteString(fmt.Sprintf("%s = %s", attrNamePlaceholder, attrValPlaceholder))

	// 		av, err := attributevalue.Marshal(v)
	// 		if err != nil {
	// 			r.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to marshal attribute value for update", "module", "dynamodb_store", "key", k, "value", v, "error", err)
	// 			return fmt.Errorf("failed to marshal attribute '%s': %w", k, err)
	// 		}
	// 		expressionAttributeValues[attrValPlaceholder] = av
	// 		i++
	// 	}

	// 	input := &dynamodb.UpdateItemInput{
	// 		TableName:                 aws.String(TaskStatusTableName),
	// 		Key:                       key,
	// 		UpdateExpression:          aws.String(updateExpression.String()),
	// 		ExpressionAttributeNames:  expressionAttributeNames,
	// 		ExpressionAttributeValues: expressionAttributeValues,
	// 		ReturnValues:              types.ReturnValueUpdatedNew, // Or other values as needed
	// 	}

	// 	// Note on optimistic locking for UpdateTaskStatus:
	// 	// If true optimistic locking is needed (check current version before updating),
	// 	// the calling business logic should fetch the current TaskStatus (which includes Version),
	// 	// prepare the updates map including the new Version (e.g., current Version + 1),
	// 	// and then this method would need to be extended to accept an `expectedVersion` parameter.
	// 	// This `expectedVersion` would be used to add a ConditionExpression to the UpdateItemInput:
	// 	//   input.ConditionExpression = aws.String("#v = :expectedV")
	// 	//   expressionAttributeNames["#v"] = "Version"
	// 	//   expressionAttributeValues[":expectedV"], _ = attributevalue.Marshal(expectedVersion)
	// 	// The current implementation directly sets the Version if it's in the `updates` map,
	// 	// without a conditional check on its previous value.

	// 	_, err := s.client.UpdateItem(ctx, input)
	// 	if err != nil {
	// 		var condCheckFailed *types.ConditionalCheckFailedException
	// 		if errors.As(err, &condCheckFailed) {
	// 			s.logger.WarnContext(ctx, "[DynamoDbStore] Conditional check failed during UpdateTaskStatus (optimistic lock failure?)", "module", "dynamodb_store", "taskId", taskId, "error", err)
	// 			// Consider returning a specific business error for optimistic lock failures, e.g., tasks.ErrOptimisticLock
	// 			return fmt.Errorf("optimistic lock failed for task %s: %w", taskId, err)
	// 		}
	// 		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to update item in DynamoDb for task status", "module", "dynamodb_store", "taskId", taskId, "tableName", TaskStatusTableName, "error", err)
	// 		return fmt.Errorf("dynamodb.UpdateItem failed for task status: %w", err)
	// 	}

	// 	s.logger.InfoContext(ctx, "[DynamoDbStore] Successfully updated task status", "module", "dynamodb_store", "taskId", taskId, "tableName", TaskStatusTableName)
	return nil
}
