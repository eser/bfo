package tasks

import "time"

type RetryPolicy struct {
	MaxAttempts   int           `conf:"max_attempts" default:"2"`
	BackoffPeriod time.Duration `conf:"backoff_period" default:"3s"`
}

type Config struct {
	DefaultMaxTokens int `conf:"default_max_tokens" default:"1000"`

	RetryPolicy RetryPolicy `conf:"retry_policy"`
}
