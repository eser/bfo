package vertex_batch

import "time"

// Config holds the configuration for the Vertex AI Batch Prediction API client.
type Config struct {
	ProjectID string        `conf:"PROJECT_ID" default:""`
	Region    string        `conf:"REGION" default:"us-central1"` // e.g., "us-central1"
	APIToken  string        `conf:"API_TOKEN" default:""`         // OAuth2 Bearer token
	BaseURL   string        `conf:"BASE_URL" default:""`          // If empty, constructed from Region. E.g. https://us-central1-aiplatform.googleapis.com/v1
	Timeout   time.Duration `conf:"TIMEOUT" default:"60s"`
}
