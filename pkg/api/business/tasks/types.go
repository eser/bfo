package tasks

import (
	"context"
	"errors" // Added import for errors package
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Task struct {
	Id                   string    `json:"id,omitempty"`
	Messages             []Message `json:"messages"`
	MaxTokens            int       `json:"max_tokens,omitempty"`             // Max tokens for the output
	EstimatedInputTokens int       `json:"estimated_input_tokens,omitempty"` // Estimated tokens for the input messages
	Priority             int       `json:"priority,omitempty"`               // Task priority (e.g., 0 = highest)
	CreatedAt            int64     `json:"created_at,omitempty"`             // Unix timestamp
	// Add other fields like UserId, RequestId, etc. if needed for tracking/billing
}

// TaskStatus represents the status of a task in its lifecycle.
// This would typically be stored in your DynamoDB table for tasks.
type TaskStatus struct {
	TaskId             string `json:"task_id" dynamodbav:"TaskId"`
	Status             string `json:"status" dynamodbav:"Status"` // e.g., "pending", "in_progress", "completed", "failed", "retrying"
	ResourceInstanceId string `json:"resource_instance_id,omitempty" dynamodbav:"ResourceInstanceId,omitempty"`
	BatchId            string `json:"batch_id,omitempty" dynamodbav:"BatchId,omitempty"`
	LastError          string `json:"last_error,omitempty" dynamodbav:"LastError,omitempty"`
	Output             string `json:"output,omitempty" dynamodbav:"Output,omitempty"` // or a more structured output
	Retries            int    `json:"retries" dynamodbav:"Retries"`
	CreatedAt          int64  `json:"created_at" dynamodbav:"CreatedAt"`
	UpdatedAt          int64  `json:"updated_at" dynamodbav:"UpdatedAt"`
	Version            int    `json:"version" dynamodbav:"Version"` // For optimistic locking
}

// TaskStatusStore is the interface for storing and retrieving task statuses.
// This would be a new store, similar to ResourceInstanceStateStore.
type TaskStatusStore interface {
	GetTaskStatus(ctx context.Context, taskId string) (*TaskStatus, error)
	PutTaskStatus(ctx context.Context, status *TaskStatus) error
	UpdateTaskStatus(ctx context.Context, taskId string, updates map[string]any) error // For partial updates
	// Potentially methods for querying tasks by status, resource, etc.
}

// ErrTaskStatusNotFound is returned when a task status is not found in the store.
var ErrTaskStatusNotFound = errors.New("task status not found") // Added error definition
