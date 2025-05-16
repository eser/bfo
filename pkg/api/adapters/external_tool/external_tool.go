package external_tool

import (
	"context"
)

type ExternalTool struct {
	Config Config
}

func New(config Config) *ExternalTool {
	return &ExternalTool{
		Config: config,
	}
}

func (externalTool *ExternalTool) Do(ctx context.Context, dummy string) string {
	return "OK"
}
