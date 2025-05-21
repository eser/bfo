package sqs_queue

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/eser/ajan/logfx"
)

type Queue struct {
	Config *Config

	logger *logfx.Logger
	client *sqs.Client
}

type Message struct {
	Body          string
	ReceiptHandle string
}

func New(config *Config, logger *logfx.Logger) *Queue {
	return &Queue{Config: config, logger: logger}
}

func (q *Queue) Init(ctx context.Context) (*string, error) {
	var cfgOptions []func(*config.LoadOptions) error
	var sqsClientOptions []func(*sqs.Options)

	if q.Config.ConnectionEndpoint != "" {
		customResolver := NewEndpointResolver(q.Config.ConnectionEndpoint)
		sqsClientOptions = append(sqsClientOptions, sqs.WithEndpointResolverV2(customResolver))
	}

	if q.Config.ConnectionProfile != "" {
		cfgOptions = append(cfgOptions, config.WithSharedConfigProfile(q.Config.ConnectionProfile))
	}

	if q.Config.ConnectionRegion != "" {
		cfgOptions = append(cfgOptions, config.WithRegion(q.Config.ConnectionRegion))
	}

	cfg, err := config.LoadDefaultConfig(ctx, cfgOptions...)
	if err != nil {
		q.logger.ErrorContext(ctx, "[SqsQueue] unable to load SDK config", "module", "sqs_queue", "error", err)

		return nil, fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	q.client = sqs.NewFromConfig(cfg, sqsClientOptions...)

	taskQueueURL, err := q.CreateQueueIfNotExists(ctx, q.Config.TaskQueueName)
	if err != nil {
		q.logger.ErrorContext(ctx, "[SqsQueue] Failed to ensure SQS queue exists during init", "module", "sqs_queue", "queueName", q.Config.TaskQueueName, "error", err)

		return nil, fmt.Errorf("failed to ensure SQS queue %s exists: %w", q.Config.TaskQueueName, err)
	}

	q.logger.InfoContext(ctx, "[SqsQueue] SQS Queue initialized", "module", "sqs_queue", "region", q.Config.ConnectionRegion, "endpoint", q.Config.ConnectionEndpoint, "taskQueueURL", *taskQueueURL)

	return taskQueueURL, nil
}

func (q *Queue) GetQueueURL(ctx context.Context, queueName string) (*string, error) {
	q.logger.DebugContext(ctx, "[SqsQueue] GetQueueURL is trying to get queue url", "module", "sqs_queue", "queueName", queueName)
	queueURLOut, err := q.client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})

	if err != nil {
		if strings.HasSuffix(err.Error(), "AWS.SimpleQueueService.NonExistentQueue: The specified queue does not exist.") {
			return nil, nil
		}

		return nil, err
	}

	return queueURLOut.QueueUrl, nil
}

func (q *Queue) ListQueues(ctx context.Context) ([]string, error) {
	q.logger.DebugContext(ctx, "[SqsQueue] ListQueues is trying to list queues", "module", "sqs_queue")
	listOut, err := q.client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		return nil, err
	}

	return listOut.QueueUrls, nil
}

func (q *Queue) CreateQueue(ctx context.Context, queueName string) (*string, error) {
	q.logger.DebugContext(ctx, "[SqsQueue] CreateQueue is trying to create queue", "module", "sqs_queue", "queueName", queueName)

	createOut, err := q.client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return nil, err
	}

	q.logger.InfoContext(ctx, "[SqsQueue] Queue created", "module", "sqs_queue", "queueUrl", *createOut.QueueUrl)
	return createOut.QueueUrl, nil
}

func (q *Queue) CreateQueueIfNotExists(ctx context.Context, queueName string) (*string, error) {
	q.logger.DebugContext(ctx, "[SqsQueue] CreateQueueIfNotExists is trying to get queue url to find out if queue exists", "module", "sqs_queue", "queueName", queueName)
	queueURL, err := q.GetQueueURL(ctx, queueName)
	if err != nil {
		return nil, err
	}

	if queueURL == nil {
		q.logger.DebugContext(ctx, "[SqsQueue] CreateQueueIfNotExists couldn't find queue, creating", "module", "sqs_queue", "queueName", queueName)
		return q.CreateQueue(ctx, queueName)
	}

	return queueURL, nil
}

func (q *Queue) SendMessage(ctx context.Context, queueURL string, message string) error {
	q.logger.DebugContext(ctx, "[SqsQueue] SendMessage is trying to send message", "module", "sqs_queue", "queueURL", queueURL)
	sendMessageInput := &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(message),
	}

	sendOut, err := q.client.SendMessage(ctx, sendMessageInput)
	if err != nil {
		return err
	}

	q.logger.InfoContext(ctx, "[SqsQueue] Message sent", "module", "sqs_queue", "messageId", *sendOut.MessageId)
	return nil
}

func (q *Queue) ReceiveMessages(ctx context.Context, queueURL string) ([]Message, error) {
	q.logger.DebugContext(ctx, "[SqsQueue] ReceiveMessages is trying to receive messages", "module", "sqs_queue", "queueURL", queueURL)
	receiveOut, err := q.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: q.Config.MaxNumberOfMessages,
		WaitTimeSeconds:     q.Config.WaitTimeSeconds,
		VisibilityTimeout:   q.Config.VisibilityTimeout,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, nil
		}

		q.logger.ErrorContext(ctx, "[SqsQueue] ReceiveMessages failed", "module", "sqs_queue", "error", err)
		return nil, err
	}

	messages := make([]Message, len(receiveOut.Messages))
	if len(messages) == 0 {
		q.logger.DebugContext(ctx, "[SqsQueue] ReceiveMessages received no messages", "module", "sqs_queue")
		return messages, nil
	}

	for i, message := range receiveOut.Messages {
		messages[i] = Message{
			Body:          *message.Body,
			ReceiptHandle: *message.ReceiptHandle,
		}
	}

	q.logger.DebugContext(ctx, "[SqsQueue] ReceiveMessages received messages", "module", "sqs_queue", "messages", messages)
	return messages, nil
}

func (q *Queue) DeleteMessage(ctx context.Context, queueURL string, receiptHandle string) error {
	q.logger.DebugContext(ctx, "[SqsQueue] DeleteMessage is trying to delete message", "module", "sqs_queue", "queueURL", queueURL, "receiptHandle", receiptHandle)
	_, err := q.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		q.logger.ErrorContext(ctx, "[SqsQueue] DeleteMessage failed", "module", "sqs_queue", "error", err)
		return err
	}

	q.logger.InfoContext(ctx, "[SqsQueue] Message deleted", "module", "sqs_queue", "receiptHandle", receiptHandle)
	return nil
}
