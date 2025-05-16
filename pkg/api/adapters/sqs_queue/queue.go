package sqs_queue

import (
	"context"
	"errors"
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

func (q *Queue) Init(ctx context.Context) {
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
		q.logger.Error("unable to load SDK config", "error", err)
	}

	q.client = sqs.NewFromConfig(cfg, sqsClientOptions...)
}

func (q *Queue) GetQueueURL(ctx context.Context, queueName string) (*string, error) {
	q.logger.DebugContext(ctx, "GetQueueURL is trying to get queue url", "queueName", queueName)
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
	q.logger.DebugContext(ctx, "ListQueues is trying to list queues")
	listOut, err := q.client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		return nil, err
	}

	return listOut.QueueUrls, nil
}

func (q *Queue) CreateQueue(ctx context.Context, queueName string) (*string, error) {
	q.logger.DebugContext(ctx, "CreateQueue is trying to create queue", "queueName", queueName)
	createOut, err := q.client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return nil, err
	}

	q.logger.Info("Queue created", "queueUrl", *createOut.QueueUrl)
	return createOut.QueueUrl, nil
}

func (q *Queue) CreateQueueIfNotExists(ctx context.Context, queueName string) (*string, error) {
	q.logger.DebugContext(ctx, "CreateQueueIfNotExists is trying to get queue url to find out if queue exists", "queueName", queueName)
	queueURL, err := q.GetQueueURL(ctx, queueName)
	if err != nil {
		return nil, err
	}

	if queueURL == nil {
		q.logger.DebugContext(ctx, "CreateQueueIfNotExists couldn't find queue, creating", "queueName", queueName)
		return q.CreateQueue(ctx, queueName)
	}

	return queueURL, nil
}

func (q *Queue) SendMessage(ctx context.Context, queueURL string, message string) error {
	q.logger.DebugContext(ctx, "SendMessage is trying to send message", "queueURL", queueURL)
	sendMessageInput := &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(message),
	}

	sendOut, err := q.client.SendMessage(ctx, sendMessageInput)
	if err != nil {
		return err
	}

	q.logger.Info("Message sent", "messageId", *sendOut.MessageId)
	return nil
}

func (q *Queue) ReceiveMessages(ctx context.Context, queueURL string) ([]Message, error) {
	q.logger.DebugContext(ctx, "ReceiveMessages is trying to receive messages", "queueURL", queueURL)
	receiveOut, err := q.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: q.Config.MaxNumberOfMessages,
		WaitTimeSeconds:     q.Config.WaitTimeSeconds,
		VisibilityTimeout:   q.Config.VisibilityTimeout,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			q.logger.InfoContext(ctx, "ReceiveMessages context was canceled")
			return nil, nil
		}

		q.logger.ErrorContext(ctx, "ReceiveMessages failed", "error", err)
		return nil, err
	}

	messages := make([]Message, len(receiveOut.Messages))
	if len(messages) == 0 {
		q.logger.DebugContext(ctx, "ReceiveMessages received no messages")
		return messages, nil
	}

	for i, message := range receiveOut.Messages {
		messages[i] = Message{
			Body:          *message.Body,
			ReceiptHandle: *message.ReceiptHandle,
		}
	}

	q.logger.DebugContext(ctx, "ReceiveMessages received messages", "messages", messages)
	return messages, nil
}

func (q *Queue) DeleteMessage(ctx context.Context, queueURL string, receiptHandle string) error {
	q.logger.DebugContext(ctx, "DeleteMessage is trying to delete message", "queueURL", queueURL, "receiptHandle", receiptHandle)
	_, err := q.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		q.logger.ErrorContext(ctx, "DeleteMessage failed", "error", err)
		return err
	}

	q.logger.InfoContext(ctx, "Message deleted", "receiptHandle", receiptHandle)
	return nil
}
