package dynamodb_store

import "time"

type Config struct {
	ConnectionEndpoint   string        `conf:"CONNECTION_ENDPOINT" default:"http://localhost:4566"`
	ConnectionProfile    string        `conf:"CONNECTION_PROFILE" default:"default"`
	ConnectionRegion     string        `conf:"CONNECTION_REGION" default:"eu-west-1"`
	TableName            string        `conf:"TABLE_NAME" default:"worker_pool_states"`
	TableCreationTimeout time.Duration `conf:"TABLE_CREATION_TIMEOUT" default:"5m"`
}
