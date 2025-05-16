package appcontext

import (
	"github.com/eser/ajan"
	"github.com/eser/bfo/pkg/api/adapters/sqs_queue"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

type FeatureFlags struct {
	Dummy bool `conf:"DUMMY" default:"false"` // dummy feature flag
}

// type Externals struct {
// 	ExternalTool external_tool.Config `conf:"EXTERNAL_TOOL"`
// }

type AppConfig struct {
	SqsQueue sqs_queue.Config `conf:"SQS_QUEUE"`

	ajan.BaseConfig

	Tasks tasks.Config `conf:"TASKS"`

	// Externals Externals `conf:"EXTERNALS"`
	Features FeatureFlags `conf:"FEATURES"`
}
