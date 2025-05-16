package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/eser/ajan/logfx"
	"github.com/eser/bfo/pkg/api/adapters/sqs_queue"
)

var (
	ErrDispatchTaskBeforeInit   = errors.New("called dispatch task before init")
	ErrFailedToMarshalTask      = errors.New("failed to marshal task")
	ErrFailedToUnmarshalTask    = errors.New("failed to unmarshal task")
	ErrFailedToSendMessage      = errors.New("failed to send message to task queue")
	ErrFailedToReceiveMessages  = errors.New("failed to receive messages from task queue")
	ErrFailedToDeleteMessage    = errors.New("failed to delete message from task queue")
	ErrFailedToExecuteProcessFn = errors.New("failed to execute process function")
)

type ServiceContext struct {
	taskQueueURL *string
}

type Service struct {
	Config   *Config
	Context  *ServiceContext
	logger   *logfx.Logger
	sqsQueue *sqs_queue.Queue
}

func NewService(config *Config, logger *logfx.Logger, sqsQueue *sqs_queue.Queue) *Service {
	return &Service{Config: config, logger: logger, sqsQueue: sqsQueue}
}

func (s *Service) Init(taskQueueURL string) error {
	s.Context = &ServiceContext{taskQueueURL: &taskQueueURL}

	return nil
}

func (s *Service) DispatchTask(ctx context.Context, task Task) error {
	if s.Context == nil {
		return ErrDispatchTaskBeforeInit
	}

	if task.MaxTokens == 0 {
		task.MaxTokens = s.Config.DefaultMaxTokens
	}

	s.logger.Info("Dispatching task", "task", task)

	// marshal task to json
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToMarshalTask, err)
	}

	err = s.sqsQueue.SendMessage(ctx, *s.Context.taskQueueURL, string(taskJSON))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToSendMessage, err)
	}

	return nil
}

func (s *Service) ProcessNextTask(ctx context.Context, fn func(ctx context.Context, task Task) error) error {
	if s.Context == nil {
		return ErrDispatchTaskBeforeInit
	}

	messages, err := s.sqsQueue.ReceiveMessages(ctx, *s.Context.taskQueueURL)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToReceiveMessages, err)
	}

	for _, message := range messages {
		var task Task
		err = json.Unmarshal([]byte(message.Body), &task)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToUnmarshalTask, err)
		}

		err = fn(ctx, task)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToExecuteProcessFn, err)
		}

		err = s.sqsQueue.DeleteMessage(ctx, *s.Context.taskQueueURL, message.ReceiptHandle)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToDeleteMessage, err)
		}
	}

	return nil
}
