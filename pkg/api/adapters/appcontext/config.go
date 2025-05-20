package appcontext

import (
	"github.com/eser/ajan"
	"github.com/eser/bfo/pkg/api/adapters/dynamodb_store"
	"github.com/eser/bfo/pkg/api/adapters/sqs_queue"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

type FeatureFlags struct {
	Dummy bool `conf:"DUMMY" default:"false"` // dummy feature flag
}

type AppConfig struct {
	SqsQueue sqs_queue.Config `conf:"SQS_QUEUE"`

	DynamoDBStore dynamodb_store.Config `conf:"DYNAMODB_STORE"`

	ajan.BaseConfig

	Tasks tasks.Config `conf:"TASKS"`

	Features FeatureFlags `conf:"FEATURES"`
}
