package dynamodb_store

type Config struct {
	ConnectionEndpoint string `conf:"CONNECTION_ENDPOINT" default:"http://localhost:4566"`
	ConnectionProfile  string `conf:"CONNECTION_PROFILE" default:"default"`
	ConnectionRegion   string `conf:"CONNECTION_REGION" default:"eu-west-1"`
}
