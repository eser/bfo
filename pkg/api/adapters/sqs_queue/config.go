package sqs_queue

type Config struct {
	ConnectionEndpoint string `conf:"connection_endpoint" default:"http://localhost:4566"`
	ConnectionProfile  string `conf:"connection_profile" default:"default"`
	ConnectionRegion   string `conf:"connection_region" default:"eu-west-1"`

	TaskQueueName string `conf:"task_queue_name" default:"test-queue"`

	MaxNumberOfMessages int32 `conf:"max_number_of_messages" default:"10"`
	WaitTimeSeconds     int32 `conf:"wait_time_seconds" default:"10"`
	VisibilityTimeout   int32 `conf:"visibility_timeout" default:"30"`
}
