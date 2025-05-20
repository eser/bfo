package resources

type Model struct {
	Name string `json:"name"`

	NoBatching bool `json:"no_batching,omitempty"`

	EnqueuedTokenLimit int `json:"enqueued_token_limit,omitempty"`
}

type ResourceInstance struct {
	Region string `json:"region"`

	Models []Model `json:"models,omitempty"`

	MaxConcurrency     int `json:"max_concurrency,omitempty"`
	MaxTokenPerBatch   int `json:"max_token_per_batch,omitempty"`
	MaxRequestPerBatch int `json:"max_request_per_batch,omitempty"`
	EnqueuedTokenLimit int `json:"enqueued_token_limit,omitempty"`
}

type Resource struct {
	ID        string             `json:"id"`
	Provider  string             `json:"provider"`
	ApiKey    string             `json:"api_key"`
	Instances []ResourceInstance `json:"instances"`

	Models []Model `json:"models,omitempty"`

	MaxConcurrency     int  `json:"max_concurrency,omitempty"`
	MaxTokenPerBatch   int  `json:"max_token_per_batch,omitempty"`
	MaxRequestPerBatch int  `json:"max_request_per_batch,omitempty"`
	EnqueuedTokenLimit int  `json:"enqueued_token_limit,omitempty"`
	Disabled           bool `json:"disabled,omitempty"`
}
