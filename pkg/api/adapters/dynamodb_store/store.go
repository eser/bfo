package dynamodb_store

import (
	"context"
	"errors" // Import errors package
	"fmt"

	// Import time package
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/eser/ajan/logfx"
	"github.com/eser/bfo/pkg/api/business/worker_pools"
)

type Store struct {
	Config *Config
	logger *logfx.Logger
	client *dynamodb.Client
}

func New(cfg *Config, logger *logfx.Logger) *Store {
	return &Store{Config: cfg, logger: logger}
}

func (s *Store) Init(ctx context.Context) error { // Added error return
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
		s.logger.ErrorContext(ctx, "unable to load SDK config for DynamoDB", "error", err)

		return fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	s.client = dynamodb.NewFromConfig(sdkConfig, ddbClientOptions...)

	if err := s.CreateTableIfNotExists(ctx, s.Config.TableName); err != nil {
		s.logger.ErrorContext(ctx, "Failed to ensure DynamoDB table exists during init", "tableName", s.Config.TableName, "error", err)

		return fmt.Errorf("failed to ensure DynamoDB table %s exists: %w", s.Config.TableName, err)
	}

	s.logger.InfoContext(ctx, "DynamoDB Store initialized", "region", s.Config.ConnectionRegion, "endpoint", s.Config.ConnectionEndpoint, "tableName", s.Config.TableName)

	return nil
}

var _ worker_pools.Store = (*Store)(nil)

func (s *Store) CreateTableIfNotExists(ctx context.Context, tableName string) error {
	s.logger.DebugContext(ctx, "Checking if table exists", "tableName", tableName)

	_, err := s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		var notFoundEx *types.ResourceNotFoundException
		if errors.As(err, &notFoundEx) {
			s.logger.InfoContext(ctx, "Table not found, creating table", "tableName", tableName)
			return s.CreateTable(ctx, tableName)
		}
		s.logger.ErrorContext(ctx, "Failed to describe table", "tableName", tableName, "error", err)
		return fmt.Errorf("failed to describe table %s: %w", tableName, err)
	}

	s.logger.DebugContext(ctx, "Table already exists", "tableName", tableName)
	return nil
}

func (s *Store) CreateTable(ctx context.Context, tableName string) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("PoolID"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("PoolID"),
				KeyType:       types.KeyTypeHash,
			},
		},
		BillingMode: types.BillingModePayPerRequest, // Or types.BillingModeProvisioned with ProvisionedThroughput
	}

	_, err := s.client.CreateTable(ctx, input)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to create table", "tableName", tableName, "error", err)
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	s.logger.InfoContext(ctx, "Table creation initiated, waiting for table to become active", "tableName", tableName)

	waiter := dynamodb.NewTableExistsWaiter(s.client)
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}, s.Config.GetTableCreationTimeout()) // Assuming you add GetTableCreationTimeout to your Config

	if err != nil {
		s.logger.ErrorContext(ctx, "Error waiting for table to exist", "tableName", tableName, "error", err)
		return fmt.Errorf("error waiting for table %s to exist: %w", tableName, err)
	}

	s.logger.InfoContext(ctx, "Table created and active", "tableName", tableName)
	return nil
}

func (s *Store) GetWorkerPoolState(ctx context.Context, poolID string) (*worker_pools.WorkerPoolState, error) {
	s.logger.DebugContext(ctx, "Getting worker pool state", "poolID", poolID, "tableName", s.Config.TableName)

	key := map[string]types.AttributeValue{
		"PoolID": &types.AttributeValueMemberS{Value: poolID},
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(s.Config.TableName),
		Key:       key,
	}

	result, err := s.client.GetItem(ctx, input)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get item from DynamoDB", "poolID", poolID, "tableName", s.Config.TableName, "error", err)
		return nil, fmt.Errorf("dynamodb.GetItem failed: %w", err)
	}

	if result.Item == nil {
		s.logger.DebugContext(ctx, "Worker pool state not found", "poolID", poolID, "tableName", s.Config.TableName)
		// Consider returning a business-layer specific error e.g., worker_pools.ErrStateNotFound
		return nil, nil // Or return an error indicating not found
	}

	var state worker_pools.WorkerPoolState
	err = attributevalue.UnmarshalMap(result.Item, &state)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to unmarshal item from DynamoDB", "poolID", poolID, "item", result.Item, "error", err)
		return nil, fmt.Errorf("attributevalue.UnmarshalMap failed: %w", err)
	}

	return &state, nil
}

func (s *Store) PutWorkerPoolState(ctx context.Context, state *worker_pools.WorkerPoolState) error {
	s.logger.DebugContext(ctx, "Putting worker pool state", "poolID", state.PoolID, "tableName", s.Config.TableName)

	item, err := attributevalue.MarshalMap(state)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to marshal state for PutItem", "poolID", state.PoolID, "error", err)
		return fmt.Errorf("attributevalue.MarshalMap failed: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(s.Config.TableName),
		Item:      item,
	}

	_, err = s.client.PutItem(ctx, input)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to put item to DynamoDB", "poolID", state.PoolID, "tableName", s.Config.TableName, "error", err)
		return fmt.Errorf("dynamodb.PutItem failed: %w", err)
	}

	s.logger.InfoContext(ctx, "Successfully put worker pool state", "poolID", state.PoolID, "tableName", s.Config.TableName)
	return nil
}

func (s *Store) DeleteWorkerPoolState(ctx context.Context, poolID string) error {
	s.logger.DebugContext(ctx, "Deleting worker pool state", "poolID", poolID, "tableName", s.Config.TableName)

	key := map[string]types.AttributeValue{
		"PoolID": &types.AttributeValueMemberS{Value: poolID},
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(s.Config.TableName),
		Key:       key,
	}

	_, err := s.client.DeleteItem(ctx, input)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to delete item from DynamoDB", "poolID", poolID, "tableName", s.Config.TableName, "error", err)
		return fmt.Errorf("dynamodb.DeleteItem failed: %w", err)
	}

	s.logger.InfoContext(ctx, "Successfully deleted worker pool state", "poolID", poolID, "tableName", s.Config.TableName)
	return nil
}
