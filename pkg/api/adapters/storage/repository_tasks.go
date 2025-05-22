package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/eser/bfo/pkg/api/business/tasks"
)

var (
	ErrFailedToMarshalTask     = errors.New("failed to marshal task")
	ErrFailedToUnmarshalTask   = errors.New("failed to unmarshal task")
	ErrFailedToEnqueueTask     = errors.New("failed to send message to task queue")
	ErrFailedToReceiveMessages = errors.New("failed to receive messages from task queue")
	ErrFailedToDeleteMessage   = errors.New("failed to delete message from task queue")
)

func (r *Repository) EnqueueTask(ctx context.Context, queueUrl string, task tasks.Task) error {
	// marshal task to json
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToMarshalTask, err)
	}

	err = r.sqsQueue.SendMessage(ctx, queueUrl, string(taskJSON))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToEnqueueTask, err)
	}

	return nil
}

func (r *Repository) PickTaskFromQueue(ctx context.Context, queueUrl string) ([]tasks.TaskWithReceipt, error) {
	// receive message from queue
	messages, err := r.sqsQueue.ReceiveMessages(ctx, queueUrl)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToReceiveMessages, err)
	}

	if len(messages) == 0 {
		return nil, nil // No message received
	}

	records := make([]tasks.TaskWithReceipt, len(messages))

	for i, message := range messages {
		record := tasks.TaskWithReceipt{
			Task:          &tasks.Task{},
			ReceiptHandle: message.ReceiptHandle,
		}

		// unmarshal message body to task
		err = json.Unmarshal([]byte(message.Body), &record.Task)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToUnmarshalTask, err)
		}

		records[i] = record
	}

	return records, nil
}

func (r *Repository) DeleteTaskFromQueue(ctx context.Context, queueUrl string, receiptHandle string) error {
	err := r.sqsQueue.DeleteMessage(ctx, queueUrl, receiptHandle)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToDeleteMessage, err)
	}

	return nil
}
