package tasks

import "time"

type RetryPolicy struct {
	MaxAttempts   int           `conf:"MAX_ATTEMPTS" default:"2"`
	BackoffPeriod time.Duration `conf:"BACKOFF_PERIOD" default:"3s"`
}

type Config struct {
	DefaultMaxTokens int `conf:"DEFAULT_MAX_TOKENS" default:"1000"`

	RetryPolicy RetryPolicy `conf:"RETRY_POLICY"`
}
