package appcontext

import (
	"github.com/eser/ajan"
	"github.com/eser/bfo/pkg/api/adapters/dynamodb_store"
	"github.com/eser/bfo/pkg/api/adapters/sqs_queue"
	"github.com/eser/bfo/pkg/api/business/resources"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

type FeatureFlags struct {
	Dummy bool `conf:"dummy" default:"false"` // dummy feature flag
}

type AppConfig struct {
	Resources resources.Config `conf:"resources"`

	DynamoDbStore dynamodb_store.Config `conf:"dynamodb_store"`

	SqsQueue sqs_queue.Config `conf:"sqs_queue"`

	ajan.BaseConfig

	Tasks tasks.Config `conf:"tasks"`

	Features FeatureFlags `conf:"features"`
}
