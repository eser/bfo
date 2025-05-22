package tasks

import (
	"errors"
	"time"
)

// ErrTaskStatusNotFound is returned when a task status is not found in the store.
var ErrTaskStatusNotFound = errors.New("task status not found")

type TaskBucketRetryPolicy struct {
	MaxAttempts int `json:"max_attempts"`
}

type TaskBucketHooks struct {
	Result  string `json:"result"`
	Refresh string `json:"refresh"`
}

type TaskBucket struct {
	Hooks TaskBucketHooks `json:"hooks"`
	Id    string          `json:"id,omitempty"`

	DefaultPriority int           `json:"default_priority,omitempty"`
	DefaultSla      time.Duration `json:"default_sla,omitempty"`

	RetryPolicy TaskBucketRetryPolicy `json:"retry_policy"`
}

type TaskMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Task struct {
	CreatedAt time.Time `json:"created_at"`
	Id        string    `json:"id,omitempty"`

	Messages []TaskMessage `json:"messages"`

	Priority int           `json:"priority,omitempty"`
	Sla      time.Duration `json:"sla,omitempty"`

	MaxTokens            int `json:"max_tokens,omitempty"`
	EstimatedInputTokens int `json:"estimated_input_tokens,omitempty"`
}

type TaskWithReceipt struct {
	Task          *Task  `json:"task"`
	ReceiptHandle string `json:"receipt_handle"`
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
