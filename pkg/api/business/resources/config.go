package resources

import "time"

type ConfigModel struct {
	Name string `conf:"name"`

	Batching bool `conf:"batching" default:"true"`

	MaxInputToken           int64 `conf:"max_input_token"`
	MaxInputTokensPerMinute int64 `conf:"max_input_tokens_per_minute"`
	MaxSizePerFile          int64 `conf:"max_size_per_file"`
	MaxRequestsPerFile      int64 `conf:"max_requests_per_file"`
	MaxFilesPerResource     int64 `conf:"max_files_per_resource"`
}

type ConfigResourceInstance struct {
	Models map[string]ConfigModel `conf:"models"`

	Properties map[string]string `conf:"properties"`

	Region   string        `conf:"region"`
	Provider string        `conf:"provider"`
	Period   time.Duration `conf:"period" default:"1m"`

	MaxInputToken           int64 `conf:"max_input_token"`
	MaxInputTokensPerMinute int64 `conf:"max_input_tokens_per_minute"`
	MaxSizePerFile          int64 `conf:"max_size_per_file"`
	MaxRequestsPerFile      int64 `conf:"max_requests_per_file"`
	MaxFilesPerResource     int64 `conf:"max_files_per_resource"`
	Disabled                bool  `conf:"disabled" default:"false"`
}

type ConfigResource struct {
	Instances map[string]ConfigResourceInstance `conf:"instances"`

	Models map[string]ConfigModel `conf:"MODELS"`

	Properties map[string]string `conf:"properties"`

	Provider string        `conf:"provider"`
	Period   time.Duration `conf:"period" default:"1m"`
	Disabled bool          `conf:"disabled" default:"false"`
}

type Config map[string]ConfigResource
