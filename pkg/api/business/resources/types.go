package resources

import "time"

// ResourceInstanceState represents the state of a resource instance.
type ResourceInstanceState struct {
	LastActivityTime     time.Time `json:"last_activity_time" dynamodbav:"LastActivityTime"`
	ResourceInstanceId   string    `json:"resource_instance_id" dynamodbav:"ResourceInstanceId"`
	ProviderName         string    `json:"provider_name" dynamodbav:"ProviderName"`          // e.g., "openai-gpt4", "anthropic-claude2"
	State                []byte    `json:"state,omitempty" dynamodbav:"State,omitempty"`     // Generic state blob if needed
	CurrentTokenLoad     int64     `json:"current_token_load" dynamodbav:"CurrentTokenLoad"` // Tokens currently being processed or in queue for this instance
	ActiveBatches        int       `json:"active_batches" dynamodbav:"ActiveBatches"`
	MaxConcurrentBatches int       `json:"max_concurrent_batches" dynamodbav:"MaxConcurrentBatches"` // From config
	TokensPerMinute      int64     `json:"tokens_per_minute" dynamodbav:"TokensPerMinute"`           // From config (rate limit)
	MaxTokensPerBatch    int64     `json:"max_tokens_per_batch" dynamodbav:"MaxTokensPerBatch"`      // From config
	Version              int       `json:"version" dynamodbav:"Version"`                             // For optimistic locking
}

// Business layer errors (optional, but good practice)
// var ErrStateNotFound = errors.New("resource instance state not found")
// var ErrOptimisticLock = errors.New("optimistic lock failure: state has been modified")
