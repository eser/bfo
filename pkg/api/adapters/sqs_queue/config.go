package sqs_queue

type Config struct {
	ConnectionEndpoint string `conf:"CONNECTION_ENDPOINT" default:"http://localhost:4566"`
	ConnectionProfile  string `conf:"CONNECTION_PROFILE" default:"default"`
	ConnectionRegion   string `conf:"CONNECTION_REGION" default:"eu-west-1"`

	TaskQueueName string `conf:"TASK_QUEUE_NAME" default:"test-queue"`

	MaxNumberOfMessages int32 `conf:"MAX_NUMBER_OF_MESSAGES" default:"10"`
	WaitTimeSeconds     int32 `conf:"WAIT_TIME_SECONDS" default:"10"`
	VisibilityTimeout   int32 `conf:"VISIBILITY_TIMEOUT" default:"30"`
}
