package appcontext

import (
	"github.com/eser/ajan"
	"github.com/eser/bfo/pkg/api/adapters/dynamodb_store"
	"github.com/eser/bfo/pkg/api/adapters/sqs_queue"
	"github.com/eser/bfo/pkg/api/business/resources"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

type FeatureFlags struct {
	Dummy bool `conf:"DUMMY" default:"false"` // dummy feature flag
}

type AppConfig struct {
	Resources resources.Config `conf:"RESOURCES"`

	DynamoDbStore dynamodb_store.Config `conf:"DYNAMODB_STORE"`

	SqsQueue sqs_queue.Config `conf:"SQS_QUEUE"`

	ajan.BaseConfig

	Tasks tasks.Config `conf:"TASKS"`

	Features FeatureFlags `conf:"FEATURES"`
}
