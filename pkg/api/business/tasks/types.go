package tasks

import (
	"errors"
	"time"
)

// ErrTaskStatusNotFound is returned when a task status is not found in the store.
var ErrTaskStatusNotFound = errors.New("task status not found")

// ----------------------------------------------------
// TaskBucket
// ----------------------------------------------------
type TaskBucketRetryPolicy struct {
	MaxAttempts int `json:"max_attempts" dynamodbav:"max_attempts"`
}

type TaskBucketHooks struct {
	Result  string `json:"result" dynamodbav:"result"`
	Refresh string `json:"refresh" dynamodbav:"refresh"`
}

type TaskBucket struct {
	Hooks TaskBucketHooks `json:"hooks" dynamodbav:"hooks"`
	Id    string          `json:"id,omitempty" dynamodbav:"id"`

	DefaultPriority int           `json:"default_priority,omitempty" dynamodbav:"default_priority"`
	DefaultSla      time.Duration `json:"default_sla,omitempty" dynamodbav:"default_sla"`

	RetryPolicy TaskBucketRetryPolicy `json:"retry_policy" dynamodbav:"retry_policy"`
}

// ----------------------------------------------------
// Task
// ----------------------------------------------------
type TaskMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Task struct {
	CreatedAt time.Time `json:"created_at"`
	BucketId  string    `json:"bucket_id"`
	Id        string    `json:"id,omitempty"`

	Model                string        `json:"model,omitempty"`
	Messages             []TaskMessage `json:"messages"`
	MaxTokens            int           `json:"max_tokens,omitempty"`
	EstimatedInputTokens int           `json:"estimated_input_tokens,omitempty"`
	Temperature          float32       `json:"temperature,omitempty"`

	Priority int           `json:"priority,omitempty"`
	Sla      time.Duration `json:"sla,omitempty"`
}

type TaskWithReceipt struct {
	Task          *Task  `json:"task"`
	ReceiptHandle string `json:"receipt_handle"`
}

// TaskStatus represents the status of a task in its lifecycle.
// This would typically be stored in your DynamoDB table for tasks.
type TaskStatus struct {
	CreatedAt          time.Time `json:"created_at" dynamodbav:"CreatedAt"`
	UpdatedAt          time.Time `json:"updated_at" dynamodbav:"UpdatedAt"`
	TaskId             string    `json:"task_id" dynamodbav:"TaskId"`
	Status             string    `json:"status" dynamodbav:"Status"` // e.g., "pending", "in_progress", "completed", "failed", "retrying"
	ResourceInstanceId string    `json:"resource_instance_id,omitempty" dynamodbav:"ResourceInstanceId,omitempty"`
	BatchId            string    `json:"batch_id,omitempty" dynamodbav:"BatchId,omitempty"`
	LastError          string    `json:"last_error,omitempty" dynamodbav:"LastError,omitempty"`
	Output             string    `json:"output,omitempty" dynamodbav:"Output,omitempty"` // or a more structured output
	Retries            int       `json:"retries" dynamodbav:"Retries"`
	Version            int       `json:"version" dynamodbav:"Version"` // For optimistic locking
}
