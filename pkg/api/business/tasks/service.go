package tasks

import (
	"fmt"
	"strconv"

	"github.com/eser/ajan/logfx"
)

type Service struct {
	config *Config
	logger *logfx.Logger
}

func NewService(config *Config, logger *logfx.Logger) *Service {
	return &Service{config: config, logger: logger}
}

func (s *Service) DispatchTask(task Task) error {
	if task.MaxTokens == 0 {
		maxTokensConfigValue, err := strconv.Atoi(s.config.DefaultMaxTokens)
		if err != nil {
			return fmt.Errorf("failed to convert default max tokens to int: %w", err)
		}

		task.MaxTokens = maxTokensConfigValue
	}

	fmt.Println("Dispatching task", task)
	return nil
}
