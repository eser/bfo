package resources

import "time"

type ConfigModel struct {
	Name string `conf:"NAME"`

	Batching bool `conf:"BATCHING" default:"true"`

	EnqueuedTokenLimit int `conf:"ENQUEUED_TOKEN_LIMIT" default:"1000"`
}

type ConfigResourceInstance struct {
	Models map[string]ConfigModel `conf:"MODELS"`

	Region string `conf:"REGION"`

	MaxConcurrency     int `conf:"MAX_CONCURRENCY"`
	MaxTokenPerBatch   int `conf:"MAX_TOKEN_PER_BATCH"`
	MaxRequestPerBatch int `conf:"MAX_REQUEST_PER_BATCH"`
	EnqueuedTokenLimit int `conf:"ENQUEUED_TOKEN_LIMIT"`
}

type ConfigResource struct {
	Instances map[string]ConfigResourceInstance `conf:"INSTANCES"`

	Models map[string]ConfigModel `conf:"MODELS"`

	Provider string `conf:"PROVIDER"`
	ApiKey   string `conf:"API_KEY"`
	BaseUrl  string `conf:"BASE_URL" default:"https://api.openai.com/v1"`

	MaxConcurrency     int           `conf:"MAX_CONCURRENCY"`
	MaxTokenPerBatch   int           `conf:"MAX_TOKEN_PER_BATCH"`
	MaxRequestPerBatch int           `conf:"MAX_REQUEST_PER_BATCH"`
	RequestTimeout     time.Duration `conf:"REQUEST_TIMEOUT" default:"30s"`
	EnqueuedTokenLimit int           `conf:"ENQUEUED_TOKEN_LIMIT"`
	Disabled           bool          `conf:"DISABLED" default:"false"`
}

type Config map[string]ConfigResource
