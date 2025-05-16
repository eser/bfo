package resources

import (
	"encoding/json"
	"os"

	"github.com/eser/ajan/logfx"
)

type Service struct {
	logger *logfx.Logger

	resources *[]Resource
}

func NewService(logger *logfx.Logger) *Service {
	return &Service{logger: logger, resources: &[]Resource{}}
}

func (s *Service) GetResources() *[]Resource {
	return s.resources
}

func (s *Service) SetResources(resources *[]Resource) {
	s.resources = resources
}

func (s *Service) LoadResourcesFromFile(path string) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var resources []Resource
	if err := json.Unmarshal(file, &resources); err != nil {
		return err
	}

	s.SetResources(&resources)

	return nil
}
