package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/eser/ajan/logfx"
	"github.com/eser/bfo/pkg/api/adapters/sqs_queue"
)

var ErrDispatchTaskBeforeInit = errors.New("called dispatch task before init")

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
		maxTokensConfigValue, err := strconv.Atoi(s.Config.DefaultMaxTokens)
		if err != nil {
			return fmt.Errorf("failed to convert default max tokens to int: %w", err)
		}

		task.MaxTokens = maxTokensConfigValue
	}

	s.logger.Info("Dispatching task", "task", task)

	// marshal task to json
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	err = s.sqsQueue.SendMessage(ctx, *s.Context.taskQueueURL, string(taskJSON))
	if err != nil {
		return fmt.Errorf("failed to send message to task queue: %w", err)
	}

	return nil
}
