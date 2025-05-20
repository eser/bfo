package resources

import (
	"context"
	"fmt"

	"github.com/eser/ajan/logfx"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

type ProviderFn = func(config *ConfigResource) Provider

type Service struct {
	config *Config
	logger *logfx.Logger

	providers map[string]ProviderFn
	resources map[string]Provider
}

func NewService(config *Config, logger *logfx.Logger) *Service {
	return &Service{
		config: config,
		logger: logger,

		providers: make(map[string]ProviderFn),
		resources: make(map[string]Provider),
	}
}

func (s *Service) AddProvider(key string, providerFn ProviderFn) {
	s.providers[key] = providerFn

	s.logger.Debug("[Resources] Provider added", "module", "resources", "key", key)
}

func (s *Service) AddResource(key string, config ConfigResource) error {
	providerFn, okProviderFn := s.providers[config.Provider]
	if !okProviderFn {
		return fmt.Errorf("provider '%s' not found for resource '%s'", config.Provider, key)
	}

	s.resources[key] = providerFn(&config)

	s.logger.Debug("[Resources] Resource added", "module", "resources", "key", key, "config", config)

	return nil
}

func (s *Service) Init() error {
	s.logger.Debug("[Resources] Loading resources from config", "module", "resources")

	for key, resourceConfig := range *s.config {
		err := s.AddResource(key, resourceConfig)

		if err != nil {
			return err
		}
	}

	s.logger.Debug("[Resources] Resources loaded from config", "module", "resources")

	return nil
}

func (s *Service) FindBestAvailableResource(ctx context.Context, task tasks.Task) (Provider, error) {
	s.logger.DebugContext(ctx, "[Resources] Finding best available resource", "module", "resources", "task", task)

	for key, resource := range s.resources {
		s.logger.DebugContext(ctx, "[Resources] Checking resource", "module", "resources", "resource", key)

		// TODO(@eser): try to estimate tokens needed for task, then check best resource available
		if true { // resource.IsAvailable()
			s.logger.DebugContext(ctx, "[Resources] Found available resource", "module", "resources", "resource", resource)

			return resource, nil
		}
	}

	return nil, nil
}

func (s *Service) DispatchTask(ctx context.Context, resource Provider, task tasks.Task) (*Batch, error) {
	batch, err := resource.CreateBatch(ctx, CreateBatchRequest{})

	if err != nil {
		s.logger.ErrorContext(ctx, "[Resources] Failed to create batch", "module", "resources", "task", task, "error", err)

		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	// TODO(@eser) calculate tokens spent, then update resource instance state

	return batch, nil
}

func (s *Service) ProcessTask(ctx context.Context, task tasks.Task) (tasks.TaskResult, error) {
	s.logger.DebugContext(ctx, "[Resources] Processing task", "module", "resources", "task", task)

	provider, err := s.FindBestAvailableResource(ctx, task)
	if err != nil {
		return tasks.TaskResultSystemTemporarilyFailed, fmt.Errorf("failed to find available resource: %w", err)
	}

	if provider == nil {
		// return fmt.Errorf("no available resources found for task id '%s'", task.Id)
		s.logger.InfoContext(ctx, "[Resources] No available resources found", "module", "resources", "task", task)
		return tasks.TaskResultSystemTemporarilyFailed, nil
	}

	batch, err := s.DispatchTask(ctx, provider, task)
	if err != nil {
		s.logger.ErrorContext(ctx, "[Resources] Failed to create batch", "module", "resources", "task", task, "error", err)
		// return fmt.Errorf("failed to create batch: %w", err)
		return tasks.TaskResultSystemTemporarilyFailed, nil
	}

	s.logger.DebugContext(ctx, "[Resources] Created batch", "module", "resources", "batch", batch)

	return tasks.TaskResultSuccess, nil
}
