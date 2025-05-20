package dynamodb_store

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/eser/ajan/logfx"
	"github.com/eser/bfo/pkg/api/business/resources"
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

	if err := s.CreateTableIfNotExists(ctx, s.Config.TableName); err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to ensure DynamoDb table exists during init", "tableName", s.Config.TableName, "error", err)

		return fmt.Errorf("failed to ensure DynamoDb table %s exists: %w", s.Config.TableName, err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] DynamoDb Store initialized", "module", "dynamodb_store", "region", s.Config.ConnectionRegion, "endpoint", s.Config.ConnectionEndpoint, "tableName", s.Config.TableName)

	return nil
}

var _ resources.Store = (*Store)(nil)

func (s *Store) CreateTableIfNotExists(ctx context.Context, tableName string) error {
	s.logger.DebugContext(ctx, "[DynamoDbStore] Checking if table exists", "module", "dynamodb_store", "tableName", tableName)

	_, err := s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		var notFoundEx *types.ResourceNotFoundException

		if errors.As(err, &notFoundEx) {
			s.logger.InfoContext(ctx, "[DynamoDbStore] Table not found, creating table", "module", "dynamodb_store", "tableName", tableName)
			return s.CreateTable(ctx, tableName)
		}

		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to describe table", "module", "dynamodb_store", "tableName", tableName, "error", err)

		return fmt.Errorf("failed to describe table %s: %w", tableName, err)
	}

	s.logger.DebugContext(ctx, "[DynamoDbStore] Table already exists", "module", "dynamodb_store", "tableName", tableName)

	return nil
}

func (s *Store) CreateTable(ctx context.Context, tableName string) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("ResourceInstanceId"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("ResourceInstanceId"),
				KeyType:       types.KeyTypeHash,
			},
		},
		BillingMode: types.BillingModePayPerRequest, // Or types.BillingModeProvisioned with ProvisionedThroughput
	}

	_, err := s.client.CreateTable(ctx, input)
	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to create table", "module", "dynamodb_store", "tableName", tableName, "error", err)
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] Table creation initiated, waiting for table to become active", "module", "dynamodb_store", "tableName", tableName)

	waiter := dynamodb.NewTableExistsWaiter(s.client)
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}, s.Config.TableCreationTimeout)

	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Error waiting for table to exist", "module", "dynamodb_store", "tableName", tableName, "error", err)
		return fmt.Errorf("error waiting for table %s to exist: %w", tableName, err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] Table created and active", "module", "dynamodb_store", "tableName", tableName)

	return nil
}

func (s *Store) GetResourceInstanceState(ctx context.Context, resourceInstanceId string) (*resources.ResourceInstanceState, error) {
	s.logger.DebugContext(ctx, "[DynamoDbStore] Getting resource instance state", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", s.Config.TableName)

	key := map[string]types.AttributeValue{
		"ResourceInstanceId": &types.AttributeValueMemberS{Value: resourceInstanceId},
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(s.Config.TableName),
		Key:       key,
	}

	result, err := s.client.GetItem(ctx, input)
	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to get item from DynamoDb", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", s.Config.TableName, "error", err)
		return nil, fmt.Errorf("dynamodb.GetItem failed: %w", err)
	}

	if result.Item == nil {
		s.logger.DebugContext(ctx, "[DynamoDbStore] Worker pool state not found", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", s.Config.TableName)
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
	s.logger.DebugContext(ctx, "[DynamoDbStore] Putting resource instance state", "module", "dynamodb_store", "resourceInstanceId", state.ResourceInstanceId, "tableName", s.Config.TableName)

	item, err := attributevalue.MarshalMap(state)

	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to marshal state for PutItem", "module", "dynamodb_store", "resourceInstanceId", state.ResourceInstanceId, "error", err)
		return fmt.Errorf("attributevalue.MarshalMap failed: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(s.Config.TableName),
		Item:      item,
	}

	_, err = s.client.PutItem(ctx, input)

	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to put item to DynamoDb", "module", "dynamodb_store", "resourceInstanceId", state.ResourceInstanceId, "tableName", s.Config.TableName, "error", err)
		return fmt.Errorf("dynamodb.PutItem failed: %w", err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] Successfully put resource instance state", "module", "dynamodb_store", "resourceInstanceId", state.ResourceInstanceId, "tableName", s.Config.TableName)

	return nil
}

func (s *Store) DeleteResourceInstanceState(ctx context.Context, resourceInstanceId string) error {
	s.logger.DebugContext(ctx, "[DynamoDbStore] Deleting resource instance state", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", s.Config.TableName)

	key := map[string]types.AttributeValue{
		"ResourceInstanceId": &types.AttributeValueMemberS{Value: resourceInstanceId},
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(s.Config.TableName),
		Key:       key,
	}

	_, err := s.client.DeleteItem(ctx, input)

	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to delete item from DynamoDb", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", s.Config.TableName, "error", err)
		return fmt.Errorf("dynamodb.DeleteItem failed: %w", err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] Successfully deleted resource instance state", "module", "dynamodb_store", "resourceInstanceId", resourceInstanceId, "tableName", s.Config.TableName)

	return nil
}
