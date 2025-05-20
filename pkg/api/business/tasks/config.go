package tasks

type RetryPolicy struct {
	MaxAttempts    int `conf:"MAX_ATTEMPTS" default:"2"`
	BackoffSeconds int `conf:"BACKOFF_SECONDS" default:"3"`
}

type Config struct {
	DefaultMaxTokens int `conf:"DEFAULT_MAX_TOKENS" default:"1000"`

	RetryPolicy RetryPolicy `conf:"RETRY_POLICY"`
}
