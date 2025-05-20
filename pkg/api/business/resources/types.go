package resources

import "context"

// ResourceInstanceState represents the state of a resource instance.
type ResourceInstanceState struct {
	ResourceInstanceId   string `json:"resource_instance_id" dynamodbav:"ResourceInstanceId"`
	ProviderName         string `json:"provider_name" dynamodbav:"ProviderName"`      // e.g., "openai-gpt4", "anthropic-claude2"
	State                []byte `json:"state,omitempty" dynamodbav:"State,omitempty"` // Generic state blob if needed
	LastActivityTime     int64  `json:"last_activity_time" dynamodbav:"LastActivityTime"`
	CurrentTokenLoad     int64  `json:"current_token_load" dynamodbav:"CurrentTokenLoad"` // Tokens currently being processed or in queue for this instance
	ActiveBatches        int    `json:"active_batches" dynamodbav:"ActiveBatches"`
	MaxConcurrentBatches int    `json:"max_concurrent_batches" dynamodbav:"MaxConcurrentBatches"` // From config
	TokensPerMinute      int64  `json:"tokens_per_minute" dynamodbav:"TokensPerMinute"`           // From config (rate limit)
	MaxTokensPerBatch    int64  `json:"max_tokens_per_batch" dynamodbav:"MaxTokensPerBatch"`      // From config
	Version              int    `json:"version" dynamodbav:"Version"`                             // For optimistic locking
}

// ResourceStateStore is the interface for storing and retrieving resource instance states.
type ResourceStateStore interface {
	GetResourceInstanceState(ctx context.Context, resourceInstanceId string) (*ResourceInstanceState, error)
	PutResourceInstanceState(ctx context.Context, state *ResourceInstanceState) error
	DeleteResourceInstanceState(ctx context.Context, resourceInstanceId string) error
	// ListResourceInstanceStates(ctx context.Context) ([]*ResourceInstanceState, error) // Example of a potential list operation
}

// Business layer errors (optional, but good practice)
// var ErrStateNotFound = errors.New("resource instance state not found")
// var ErrOptimisticLock = errors.New("optimistic lock failure: state has been modified")
