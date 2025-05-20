package resources

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/eser/ajan/logfx"
)

type Service struct {
	logger *logfx.Logger

	providers map[string]ProviderFn
	resources []Resource
}

func NewService(logger *logfx.Logger) *Service {
	return &Service{
		logger:    logger,
		providers: make(map[string]ProviderFn),
		resources: make([]Resource, 0),
	}
}

func (s *Service) AddProvider(key string, providerFn ProviderFn) {
	s.providers[key] = providerFn

	s.logger.Debug("[Resources] Provider added", "module", "resources", "key", key)
}

func (s *Service) AddResource(resourceDef ResourceDef) error {
	providerFn, okProviderFn := s.providers[resourceDef.Provider]
	if !okProviderFn {
		return fmt.Errorf("provider '%s' not found", resourceDef.Provider)
	}

	providerInstance := providerFn(&resourceDef)

	resource := Resource{
		ResourceDef:      resourceDef,
		providerInstance: providerInstance,
	}

	s.resources = append(s.resources, resource)

	s.logger.Debug("[Resources] Resource added", "module", "resources", "resourceDef", resourceDef)

	return nil
}

func (s *Service) LoadResourcesFromFile(path string) error {
	s.logger.Debug("[Resources] Loading resources from file", "module", "resources", "path", path)

	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var resourceDefs []ResourceDef
	if err := json.Unmarshal(file, &resourceDefs); err != nil {
		return err
	}

	for _, resourceDef := range resourceDefs {
		err := s.AddResource(resourceDef)

		if err != nil {
			return err
		}
	}

	s.logger.Debug("[Resources] Resources loaded from file", "module", "resources", "path", path)

	return nil
}
