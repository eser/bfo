package worker_pools

import "context"

// WorkerPoolState represents the state of a worker pool.
type WorkerPoolState struct {
	PoolID string `json:"pool_id" dynamodbav:"PoolID"` // Added dynamodbav tag for marshaling
	State  []byte `json:"state" dynamodbav:"State"`    // Using []byte for flexibility
	// Add other relevant fields like LastUpdated, Version, etc.
	// LastUpdated int64 `json:"last_updated" dynamodbav:"LastUpdated"`
	// Version     int   `json:"version" dynamodbav:"Version"`
}

// Store is the interface for storing and retrieving worker pool states.
type Store interface {
	GetWorkerPoolState(ctx context.Context, poolID string) (*WorkerPoolState, error)
	PutWorkerPoolState(ctx context.Context, state *WorkerPoolState) error
	DeleteWorkerPoolState(ctx context.Context, poolID string) error
	// ListWorkerPoolStates(ctx context.Context) ([]*WorkerPoolState, error) // Example of a potential list operation
}

// Business layer errors (optional, but good practice)
// var ErrStateNotFound = errors.New("worker pool state not found")
// var ErrOptimisticLock = errors.New("optimistic lock failure: state has been modified")
