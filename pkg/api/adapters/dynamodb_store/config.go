package dynamodb_store

type Config struct {
	ConnectionEndpoint string `conf:"connection_endpoint" default:"http://localhost:4566"`
	ConnectionProfile  string `conf:"connection_profile" default:"default"`
	ConnectionRegion   string `conf:"connection_region" default:"eu-west-1"`
}
