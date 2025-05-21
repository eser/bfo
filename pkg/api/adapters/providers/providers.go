package providers

import "github.com/eser/bfo/pkg/api/business/resources"

//go:generate go tool mockery --name=Provider --inpackage --inpackage-suffix --case=underscore --structname=MockClient --filename=mock_client.go
type Provider resources.Provider
