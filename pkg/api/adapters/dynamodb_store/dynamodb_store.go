package dynamodb_store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/eser/ajan/logfx"
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
	if err := s.ensureTableExists(ctx, ResourceStateTableName, "ResourceInstanceId"); err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to ensure DynamoDb table exists for resource instances", "tableName", ResourceStateTableName, "error", err)

		return fmt.Errorf("failed to ensure DynamoDb table %s exists: %w", ResourceStateTableName, err)
	}

	// Ensure Task Status table exists, if configured
	if err := s.ensureTableExists(ctx, TaskStatusTableName, "TaskId"); err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to ensure DynamoDb table exists for task statuses", "tableName", TaskStatusTableName, "error", err)

		return fmt.Errorf("failed to ensure DynamoDb table %s exists: %w", TaskStatusTableName, err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] DynamoDb Store initialized successfully", "module", "dynamodb_store", "region", s.Config.ConnectionRegion, "endpoint", s.Config.ConnectionEndpoint)

	return nil
}

// ensureTableExists checks if a table exists and creates it if it doesn't.
func (s *Store) ensureTableExists(ctx context.Context, tableName string, primaryKeyAttributeName string) error {
	s.logger.DebugContext(ctx, "[DynamoDbStore] Checking if table exists", "module", "dynamodb_store", "tableName", tableName)

	_, err := s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		var notFoundEx *types.ResourceNotFoundException
		if errors.As(err, &notFoundEx) {
			s.logger.InfoContext(ctx, "[DynamoDbStore] Table not found, creating table", "module", "dynamodb_store", "tableName", tableName, "primaryKey", primaryKeyAttributeName)
			return s.createTable(ctx, tableName, primaryKeyAttributeName)
		}

		s.logger.ErrorContext(ctx, "[DynamoDbStore] Failed to describe table", "module", "dynamodb_store", "tableName", tableName, "error", err)
		return fmt.Errorf("failed to describe table %s: %w", tableName, err)
	}

	s.logger.DebugContext(ctx, "[DynamoDbStore] Table already exists", "module", "dynamodb_store", "tableName", tableName)
	return nil
}

// createTable creates a new DynamoDB table with the given name and primary key.
func (s *Store) createTable(ctx context.Context, tableName string, primaryKeyAttributeName string) error {
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
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}, TableCreationTimeout)

	if err != nil {
		s.logger.ErrorContext(ctx, "[DynamoDbStore] Error waiting for table to exist", "module", "dynamodb_store", "tableName", tableName, "error", err)
		return fmt.Errorf("error waiting for table %s to exist: %w", tableName, err)
	}

	s.logger.InfoContext(ctx, "[DynamoDbStore] Table created and active", "module", "dynamodb_store", "tableName", tableName)
	return nil
}
