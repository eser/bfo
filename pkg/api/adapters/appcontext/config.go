package appcontext

import (
	"github.com/eser/ajan"
	"github.com/eser/bfo/pkg/api/adapters/external_tool"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

type FeatureFlags struct {
	Dummy bool `conf:"DUMMY" default:"false"` // dummy feature flag
}

type Externals struct {
	ExternalTool external_tool.Config `conf:"EXTERNAL_TOOL"`
}

type AppConfig struct {
	Externals Externals `conf:"EXTERNALS"`

	Tasks tasks.Config `conf:"TASKS"`
	ajan.BaseConfig

	Features FeatureFlags `conf:"FEATURES"`
}
