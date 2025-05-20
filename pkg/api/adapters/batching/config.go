package batching

import "time"

// Config holds the configuration for a Batcher client.
type Config struct {
	APIKey  string        `conf:"API_KEY" default:""`
	BaseURL string        `conf:"BASE_URL" default:"https://api.openai.com/v1"`
	Timeout time.Duration `conf:"TIMEOUT" default:"30s"`
}
