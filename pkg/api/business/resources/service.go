package resources

import (
	"fmt"

	"github.com/eser/ajan/logfx"
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
