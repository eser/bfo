package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/eser/ajan/logfx"
	"github.com/eser/bfo/pkg/api/adapters/sqs_queue"
	"github.com/oklog/ulid/v2" // For generating task Ids
)

var (
	ErrDispatchTaskBeforeInit   = errors.New("called dispatch task before init")
	ErrFailedToMarshalTask      = errors.New("failed to marshal task")
	ErrFailedToUnmarshalTask    = errors.New("failed to unmarshal task")
	ErrFailedToSendMessage      = errors.New("failed to send message to task queue")
	ErrFailedToReceiveMessages  = errors.New("failed to receive messages from task queue")
	ErrFailedToDeleteMessage    = errors.New("failed to delete message from task queue")
	ErrFailedToExecuteProcessFn = errors.New("failed to execute process function")
)

type TaskResult int

const (
	TaskResultSuccess TaskResult = iota
	TaskResultMessageTemporarilyFailed
	TaskResultMessagePermanentlyFailed
	TaskResultSystemTemporarilyFailed
	TaskResultSystemPermanentlyFailed
)

type ServiceContext struct {
	taskQueueURL *string
}

type Service struct {
	Config          *Config
	Context         *ServiceContext
	logger          *logfx.Logger
	sqsQueue        *sqs_queue.Queue
	taskStatusStore TaskStatusStore
}

func NewService(config *Config, logger *logfx.Logger, sqsQueue *sqs_queue.Queue, taskStatusStore TaskStatusStore) *Service {
	return &Service{Config: config, logger: logger, sqsQueue: sqsQueue, taskStatusStore: taskStatusStore}
}

func (s *Service) Init(taskQueueURL string) error {
	s.Context = &ServiceContext{taskQueueURL: &taskQueueURL}

	return nil
}

func (s *Service) DispatchTask(ctx context.Context, task Task) error {
	if s.Context == nil {
		return ErrDispatchTaskBeforeInit
	}

	// Generate Id and Timestamp if not present
	if task.Id == "" {
		task.Id = ulid.Make().String()
	}
	if task.CreatedAt == 0 {
		task.CreatedAt = time.Now().Unix()
	}

	if task.MaxTokens == 0 {
		task.MaxTokens = s.Config.DefaultMaxTokens
	}

	s.logger.Info("[Tasks] Dispatching task", "module", "tasks", "task", task)

	// marshal task to json
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToMarshalTask, err)
	}

	// Create initial TaskStatus
	initialStatus := &TaskStatus{
		TaskId:    task.Id,
		Status:    "pending_queue", // Indicates it's about to be queued
		CreatedAt: task.CreatedAt,
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}
	err = s.taskStatusStore.PutTaskStatus(ctx, initialStatus)
	if err != nil {
		s.logger.ErrorContext(ctx, "[Tasks] Failed to create initial task status", "task_id", task.Id, "error", err)
		return fmt.Errorf("failed to create initial task status for %s: %w", task.Id, err)
	}

	err = s.sqsQueue.SendMessage(ctx, *s.Context.taskQueueURL, string(taskJSON))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToSendMessage, err)
	}

	return nil
}

func (s *Service) ProcessNextTask(ctx context.Context, fn func(ctx context.Context, task Task) (TaskResult, error)) error {
	if s.Context == nil {
		return ErrDispatchTaskBeforeInit
	}

	messages, err := s.sqsQueue.ReceiveMessages(ctx, *s.Context.taskQueueURL)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToReceiveMessages, err)
	}

	for _, message := range messages {
		// Process messages one by one for now, can be made concurrent with a worker pool later
		// go func() { // Removing go func for sequential processing first, easier to debug
		var task Task
		err = json.Unmarshal([]byte(message.Body), &task)
		if err != nil {
			s.logger.ErrorContext(ctx, "[Tasks] Failed to unmarshal task from SQS message", "module", "tasks", "message_body", message.Body, "error", err)
			// TODO: Decide if this message should be deleted or moved to a DLQ
			// For now, continue to the next message
			continue
		}

		s.logger.InfoContext(ctx, "[Tasks] Processing task from queue", "module", "tasks", "task_id", task.Id)

		// Update task status to "in_progress"
		err = s.taskStatusStore.UpdateTaskStatus(ctx, task.Id, map[string]any{
			"Status":    "in_progress_worker",
			"UpdatedAt": time.Now().Unix(),
		})
		if err != nil {
			s.logger.ErrorContext(ctx, "[Tasks] Failed to update task status to in_progress_worker", "task_id", task.Id, "error", err)
			// If status update fails, we might not want to proceed, or handle it gracefully.
			// For now, log and continue processing the task.
		}

		taskResult, processErr := fn(ctx, task) // This fn is `resources.Service.ProcessTask`

		// Update task status based on the result from resources.Service.ProcessTask
		// Note: resources.Service.ProcessTask itself should update TaskStatus after batch creation.
		// The status update here is more about the SQS message handling outcome.

		finalStatusUpdate := map[string]any{
			"UpdatedAt": time.Now().Unix(),
		}

		if processErr != nil {
			s.logger.ErrorContext(ctx, "[Tasks] Error returned by task processing function", "module", "tasks", "task_id", task.Id, "error", processErr)
			finalStatusUpdate["Status"] = "failed_worker_processing"
			finalStatusUpdate["LastError"] = processErr.Error()
			// Decide if message should be deleted or retried based on error type
		} else {
			switch taskResult {
			case TaskResultSuccess: // This means successfully submitted to batch by resources.Service
				s.logger.InfoContext(ctx, "[Tasks] Task successfully submitted to batch by worker", "module", "tasks", "task_id", task.Id)
				// The actual completion status will be updated by the batch completion handler.
				// Status might have been set to "in_progress_batch" or similar by resources.Service.
				// We might not need to update status here if resources.Service handles it post-dispatch.
				// However, for clarity, let's assume resources.Service.ProcessTask updates the status to reflect batch submission.
				// If it doesn't, we'd set it here, e.g., finalStatusUpdate["Status"] = "submitted_to_batch_processor"

				// Delete message from SQS only if successfully handed off
				err = s.sqsQueue.DeleteMessage(ctx, *s.Context.taskQueueURL, message.ReceiptHandle)
				if err != nil {
					s.logger.ErrorContext(ctx, "[Tasks] Failed to delete SQS message after successful hand-off", "module", "tasks", "task_id", task.Id, "receiptHandle", message.ReceiptHandle, "error", err)
					// This is problematic: task is processed but SQS message might reappear.
					// Idempotency in task processing is crucial.
				}
			case TaskResultSystemTemporarilyFailed:
				s.logger.WarnContext(ctx, "[Tasks] Task processing failed temporarily (system)", "module", "tasks", "task_id", task.Id)
				finalStatusUpdate["Status"] = "retrying_worker_system_failure"
				// Do not delete SQS message, it will be retried after visibility timeout.
				// Increment retry count in TaskStatus.
				// TODO: Add retry count increment logic to UpdateTaskStatus or handle here.
			case TaskResultMessageTemporarilyFailed: // e.g. bad input that might be fixed
				s.logger.WarnContext(ctx, "[Tasks] Task processing failed temporarily (message)", "module", "tasks", "task_id", task.Id)
				finalStatusUpdate["Status"] = "retrying_worker_message_failure"
				// Do not delete SQS message.
			case TaskResultMessagePermanentlyFailed:
				s.logger.ErrorContext(ctx, "[Tasks] Task processing failed permanently (message)", "module", "tasks", "task_id", task.Id)
				finalStatusUpdate["Status"] = "failed_worker_message_permanent"
				// Delete SQS message as it cannot be processed.
				err = s.sqsQueue.DeleteMessage(ctx, *s.Context.taskQueueURL, message.ReceiptHandle)
				if err != nil {
					s.logger.ErrorContext(ctx, "[Tasks] Failed to delete SQS message for permanently failed task", "module", "tasks", "task_id", task.Id, "error", err)
				}
			default:
				s.logger.ErrorContext(ctx, "[Tasks] Unknown task result from worker", "module", "tasks", "task_id", task.Id, "result", taskResult)
				finalStatusUpdate["Status"] = "failed_worker_unknown_result"
				// Delete SQS message to prevent reprocessing an unknown state.
				err = s.sqsQueue.DeleteMessage(ctx, *s.Context.taskQueueURL, message.ReceiptHandle)
				if err != nil {
					s.logger.ErrorContext(ctx, "[Tasks] Failed to delete SQS message for task with unknown result", "module", "tasks", "task_id", task.Id, "error", err)
				}
			}
		}

		// Update TaskStatus with the final outcome of this processing attempt.
		if len(finalStatusUpdate) > 1 { // More than just UpdatedAt
			err = s.taskStatusStore.UpdateTaskStatus(ctx, task.Id, finalStatusUpdate)
			if err != nil {
				s.logger.ErrorContext(ctx, "[Tasks] Failed to update final task status after processing attempt", "task_id", task.Id, "error", err)
			}
		}
		// }() // End of removed go func
	}

	return nil
}
