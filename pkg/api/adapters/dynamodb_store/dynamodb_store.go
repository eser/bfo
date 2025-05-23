package dynamodb_store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/eser/ajan/logfx"
)

const (
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
		s.logger.ErrorContext(
			ctx,
			"[DynamoDbStore] unable to load SDK config for DynamoDb",
			slog.String("module", "dynamodb_store"),
			slog.Any("error", err),
		)

		return fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	s.client = dynamodb.NewFromConfig(sdkConfig, ddbClientOptions...)

	s.logger.InfoContext(
		ctx,
		"[DynamoDbStore] DynamoDb Store initialized successfully",
		slog.String("module", "dynamodb_store"),
		slog.String("region", s.Config.ConnectionRegion),
		slog.String("endpoint", s.Config.ConnectionEndpoint),
	)

	return nil
}

// EnsureTableExists checks if a table exists and creates it if it doesn't.
func (s *Store) EnsureTableExists(ctx context.Context, tableName string, primaryKeyAttributeName string) error {
	s.logger.DebugContext(
		ctx,
		"[DynamoDbStore] Checking if table exists",
		slog.String("module", "dynamodb_store"),
		slog.String("tableName", tableName),
	)

	_, err := s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		var notFoundEx *types.ResourceNotFoundException

		if errors.As(err, &notFoundEx) {
			s.logger.InfoContext(
				ctx,
				"[DynamoDbStore] Table not found, creating table",
				slog.String("module", "dynamodb_store"),
				slog.String("tableName", tableName),
				slog.String("primaryKey", primaryKeyAttributeName),
			)

			return s.createTable(ctx, tableName, primaryKeyAttributeName)
		}

		s.logger.ErrorContext(
			ctx,
			"[DynamoDbStore] Failed to describe table",
			slog.String("module", "dynamodb_store"),
			slog.String("tableName", tableName),
			slog.Any("error", err),
		)

		return fmt.Errorf("failed to describe table %s: %w", tableName, err)
	}

	s.logger.DebugContext(
		ctx,
		"[DynamoDbStore] Table already exists",
		slog.String("module", "dynamodb_store"),
		slog.String("tableName", tableName),
	)

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
		s.logger.ErrorContext(
			ctx,
			"[DynamoDbStore] Failed to create table",
			slog.String("module", "dynamodb_store"),
			slog.String("tableName", tableName),
			slog.String("primaryKey", primaryKeyAttributeName),
			slog.Any("error", err),
		)

		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	s.logger.InfoContext(
		ctx,
		"[DynamoDbStore] Table creation initiated, waiting for table to become active",
		slog.String("module", "dynamodb_store"),
		slog.String("tableName", tableName),
	)

	waiter := dynamodb.NewTableExistsWaiter(s.client)
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}, TableCreationTimeout)

	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"[DynamoDbStore] Error waiting for table to exist",
			slog.String("module", "dynamodb_store"),
			slog.String("tableName", tableName),
			slog.Any("error", err),
		)

		return fmt.Errorf("error waiting for table %s to exist: %w", tableName, err)
	}

	s.logger.InfoContext(
		ctx,
		"[DynamoDbStore] Table created and active",
		slog.String("module", "dynamodb_store"),
		slog.String("tableName", tableName),
	)

	return nil
}

func (s *Store) ListItems(ctx context.Context, tableName string, out any) error {
	s.logger.DebugContext(
		ctx,
		"[DynamoDbStore] Listing items",
		slog.String("module", "dynamodb_store"),
		slog.String("tableName", tableName),
	)

	input := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}

	result, err := s.client.Scan(ctx, input)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"[DynamoDbStore] Failed to list items from DynamoDb",
			slog.String("module", "dynamodb_store"),
			slog.String("tableName", tableName),
			slog.Any("error", err),
		)

		return fmt.Errorf("dynamodb.Scan failed for table %s: %w", tableName, err)
	}

	err = attributevalue.UnmarshalListOfMaps(result.Items, &out)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"[DynamoDbStore] Failed to unmarshal items from DynamoDb",
			slog.String("module", "dynamodb_store"),
			slog.String("tableName", tableName),
			slog.Any("error", err),
		)

		return fmt.Errorf("attributevalue.UnmarshalListOfMaps failed for table %s: %w", tableName, err)
	}

	return nil
}

func (s *Store) GetItem(ctx context.Context, tableName string, fieldName string, fieldValue string, out any) (bool, error) {
	s.logger.DebugContext(
		ctx,
		"[DynamoDbStore] Getting item",
		slog.String("module", "dynamodb_store"),
		slog.String("tableName", tableName),
		slog.String("fieldName", fieldName),
		slog.String("fieldValue", fieldValue),
	)

	key := map[string]types.AttributeValue{
		fieldName: &types.AttributeValueMemberS{Value: fieldValue},
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key:       key,
	}

	result, err := s.client.GetItem(ctx, input)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"[DynamoDbStore] Failed to get item",
			slog.String("module", "dynamodb_store"),
			slog.String("tableName", tableName),
			slog.String("fieldName", fieldName),
			slog.String("fieldValue", fieldValue),
			slog.Any("error", err),
		)

		return false, fmt.Errorf("dynamodb.GetItem failed for item: %w", err)
	}

	if result.Item == nil {
		return false, nil
	}

	err = attributevalue.UnmarshalMap(result.Item, &out)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"[DynamoDbStore] Failed to unmarshal item for getting item",
			"module", "dynamodb_store",
			slog.String("tableName", tableName),
			slog.String("fieldName", fieldName),
			slog.String("fieldValue", fieldValue),
			slog.Any("item", result.Item),
			slog.Any("error", err),
		)

		return false, fmt.Errorf("attributevalue.UnmarshalMap failed for getting item: %w", err)
	}

	return true, nil
}

func (s *Store) UpsertItem(ctx context.Context, tableName string, item any) error {
	s.logger.DebugContext(
		ctx,
		"[DynamoDbStore] Upserting item",
		slog.String("module", "dynamodb_store"),
		slog.String("tableName", tableName),
		slog.Any("item", item),
	)

	itemMap, err := attributevalue.MarshalMapWithOptions(item)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"[DynamoDbStore] Failed to marshal item for upsert",
			slog.String("module", "dynamodb_store"),
			slog.String("tableName", tableName),
			slog.Any("item", item),
			slog.Any("error", err),
		)

		return fmt.Errorf("attributevalue.MarshalMap failed for item: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      itemMap,
		// No ConditionExpression means it will upsert
	}

	_, err = s.client.PutItem(ctx, input)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"[DynamoDbStore] Failed to upsert item in DynamoDb",
			slog.String("module", "dynamodb_store"),
			slog.String("tableName", tableName),
			slog.Any("item", item),
			slog.Any("error", err),
		)

		return fmt.Errorf("dynamodb.PutItem failed for upsert: %w", err)
	}

	s.logger.DebugContext(
		ctx,
		"[DynamoDbStore] Successfully upserted item",
		slog.String("module", "dynamodb_store"),
		slog.String("tableName", tableName),
		slog.Any("item", item),
	)

	return nil
}

func (s *Store) DeleteItem(ctx context.Context, tableName string, fieldName string, fieldValue string) error {
	s.logger.DebugContext(
		ctx,
		"[DynamoDbStore] Deleting item",
		slog.String("module", "dynamodb_store"),
		slog.String("tableName", tableName),
		slog.String("fieldName", fieldName),
		slog.String("fieldValue", fieldValue),
	)

	key := map[string]types.AttributeValue{
		fieldName: &types.AttributeValueMemberS{Value: fieldValue},
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key:       key,
	}

	_, err := s.client.DeleteItem(ctx, input)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"[DynamoDbStore] Failed to delete item from DynamoDb",
			slog.String("module", "dynamodb_store"),
			slog.String("tableName", tableName),
			slog.String("fieldName", fieldName),
			slog.String("fieldValue", fieldValue),
			slog.Any("error", err),
		)

		return fmt.Errorf("dynamodb.DeleteItem failed for item: %w", err)
	}

	return nil
}
