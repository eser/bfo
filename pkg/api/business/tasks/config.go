package tasks

type Config struct {
	// TODO(@eser) convert to int
	DefaultMaxTokens string `conf:"DEFAULT_MAX_TOKENS" default:"1000"`
}
