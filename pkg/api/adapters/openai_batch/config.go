package openai_batch

import "time"

// Config holds the configuration for the OpenAI Batch API client.
type Config struct {
	APIKey  string        `conf:"API_KEY" default:""`
	BaseURL string        `conf:"BASE_URL" default:"https://api.openai.com/v1"`
	Timeout time.Duration `conf:"TIMEOUT" default:"30s"`
}
