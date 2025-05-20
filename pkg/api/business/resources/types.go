package resources

import "context"

// ResourceInstanceState represents the state of a resource instance.
type ResourceInstanceState struct {
	ResourceInstanceId string `json:"resource_instance_id" dynamodbav:"ResourceInstanceId"` // Added dynamodbav tag for marshaling
	State              []byte `json:"state" dynamodbav:"State"`                             // Using []byte for flexibility
	// Add other relevant fields like LastUpdated, Version, etc.
	// LastUpdated int64 `json:"last_updated" dynamodbav:"LastUpdated"`
	// Version     int   `json:"version" dynamodbav:"Version"`
}

// Store is the interface for storing and retrieving resource instance states.
type Store interface {
	GetResourceInstanceState(ctx context.Context, resourceInstanceId string) (*ResourceInstanceState, error)
	PutResourceInstanceState(ctx context.Context, state *ResourceInstanceState) error
	DeleteResourceInstanceState(ctx context.Context, resourceInstanceId string) error
	// ListResourceInstanceStates(ctx context.Context) ([]*ResourceInstanceState, error) // Example of a potential list operation
}

// Business layer errors (optional, but good practice)
// var ErrStateNotFound = errors.New("resource instance state not found")
// var ErrOptimisticLock = errors.New("optimistic lock failure: state has been modified")
