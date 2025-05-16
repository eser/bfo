package sqs_queue

import (
	"context"
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

func (q *Queue) ListQueues(ctx context.Context) ([]string, error) {
	listOut, err := q.client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		return nil, err
	}

	return listOut.QueueUrls, nil
}

func (q *Queue) GetQueueURLByName(ctx context.Context, queueName string) (*string, error) {
	queueURLs, err := q.ListQueues(ctx)
	if err != nil {
		return nil, err
	}

	for _, queueURL := range queueURLs {
		// check last segment of queueURL is equal to queueName
		if strings.Split(queueURL, "/")[len(strings.Split(queueURL, "/"))-1] == queueName {
			return &queueURL, nil
		}
	}

	return nil, nil
}

func (q *Queue) CreateQueue(ctx context.Context, queueName string) (*string, error) {
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
	queueURL, err := q.GetQueueURLByName(ctx, queueName)
	if err != nil {
		return nil, err
	}

	if queueURL == nil {
		return q.CreateQueue(ctx, queueName)
	}

	return queueURL, nil
}

func (q *Queue) SendMessage(ctx context.Context, queueURL string, message string) error {
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

func (q *Queue) ReceiveMessage(ctx context.Context, queueURL string) (string, error) {
	receiveOut, err := q.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		return "", err
	}

	if len(receiveOut.Messages) == 0 {
		return "", nil
	}

	return *receiveOut.Messages[0].Body, nil
}
