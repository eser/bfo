package dynamodb_store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/eser/ajan/logfx"
	"github.com/eser/bfo/pkg/api/business/resources"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

const (
	TaskStatusTableName    = "task_statuses"
	ResourceStateTableName = "resource_states"

	TableCreationTimeout = 2 * time.Minute
)

type Store struct {
	Config *Config
	logger *logfx.Logger
	client *dynamodb.Client
}

func New(cfg *Config, logger *logfx.Logger) *Store {
	return &Store{Config: cfg, logger: logger}
}

func (s *Store) Init(ctx context.Context) error {
	var cfgOptions []func(*config.LoadOptions) error
	var ddbClientOptions []func(*dynamodb.Options)

	if s.Config.ConnectionEndpoint != "" {
		customResolver := NewEndpointResolver(s.Config.ConnectionEndpoint)
		ddbClientOptions = append(ddbClientOptions, dynamodb.WithEndpointResolverV2(customResolver))
	}

	if s.Config.ConnectionProfile != "" {
		cfgOptions = append(cfgOptions, config.WithSharedConfigProfile(s.Config.ConnectionProfile))
	}

	if s.Config.ConnectionRegion != "" {
		cfgOptions = append(cfgOptions, config.WithRegion(s.Config.ConnectionRegion))
	}

	sdkConfig, err := config.LoadDefaultConfig(ctx, cfgOptions...)
	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] unable to load SDK config for DynamoDb", "error", err)

		return fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	s.client = dynamodb.NewFromConfig(sdkConfig, ddbClientOptions...)

	// Ensure Resource Instance State table exists
	if err := s.ensureTableExists(ctx, ResourceStateTableName, "ResourceInstanceId", TableCreationTimeout); err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to ensure DynamoDb table exists for resource instances", "tableName", ResourceStateTableName, "error", err)

		return fmt.Errorf("failed to ensure DynamoDb table %s exists: %w", ResourceStateTableName, err)
	}

	// Ensure Task Status table exists, if configured
	if err := s.ensureTableExists(ctx, TaskStatusTableName, "TaskId", TableCreationTimeout); err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to ensure DynamoDb table exists for task statuses", "tableName", TaskStatusTableName, "error", err)

		return fmt.Errorf("failed to ensure DynamoDb table %s exists: %w", TaskStatusTableName, err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] DynamoDb Store initialized successfully", "module", "dynamodb_store", "region", s.Config.ConnectionRegion, "endpoint", s.Config.ConnectionEndpoint)

	return nil
}

var _ resources.ResourceStateStore = (*Store)(nil)
var _ tasks.TaskStatusStore = (*Store)(nil)

// ensureTableExists checks if a table exists and creates it if it doesn't.
func (s *Store) ensureTableExists(ctx context.Context, tableName string, primaryKeyAttributeName string, timeout time.Duration) error {
	s.logger.DebugContext(ctx, "[DynamoDbStore] Checking if table exists", "module", "dynamodb_store", "tableName", tableName)

	_, err := s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		var notFoundEx *types.ResourceNotFoundException
		if errors.As(err, &notFoundEx) {
			s.logger.InfoContext(ctx, "[DynamoDbStore] Table not found, creating table", "module", "dynamodb_store", "tableName", tableName, "primaryKey", primaryKeyAttributeName)
			return s.createTable(ctx, tableName, primaryKeyAttributeName, timeout)
		}
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to describe table", "module", "dynamodb_store", "tableName", tableName, "error", err)
		return fmt.Errorf("failed to describe table %s: %w", tableName, err)
	}

	s.logger.DebugContext(ctx, "[DynamoDbStore] Table already exists", "module", "dynamodb_store", "tableName", tableName)
	return nil
}

// createTable creates a new DynamoDB table with the given name and primary key.
func (s *Store) createTable(ctx context.Context, tableName string, primaryKeyAttributeName string, timeout time.Duration) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String(primaryKeyAttributeName),
				AttributeType: types.ScalarAttributeTypeS, // Assuming primary key is always a string
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String(primaryKeyAttributeName),
				KeyType:       types.KeyTypeHash,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	}

	_, err := s.client.CreateTable(ctx, input)
	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to create table", "module", "dynamodb_store", "tableName", tableName, "primaryKey", primaryKeyAttributeName, "error", err)
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] Table creation initiated, waiting for table to become active", "module", "dynamodb_store", "tableName", tableName)

	waiter := dynamodb.NewTableExistsWaiter(s.client)
	if timeout == 0 {
		timeout = 2 * time.Minute // Default timeout
	}
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}, timeout)

	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Error waiting for table to exist", "module", "dynamodb_store", "tableName", tableName, "error", err)
		return fmt.Errorf("error waiting for table %s to exist: %w", tableName, err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] Table created and active", "module", "dynamodb_store", "tableName", tableName)
	return nil
}

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
