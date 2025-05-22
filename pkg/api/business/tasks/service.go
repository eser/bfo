package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eser/ajan/logfx"
	"github.com/oklog/ulid/v2"
)

var (
	ErrDispatchTaskBeforeInit   = errors.New("called dispatch task before init")
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

type Repository interface {
	GetTaskStatus(ctx context.Context, taskId string) (*TaskStatus, error)
	PutTaskStatus(ctx context.Context, status *TaskStatus) error
	UpdateTaskStatus(ctx context.Context, taskId string, updates map[string]any) error // For partial updates
	EnqueueTask(ctx context.Context, queueUrl string, task Task) error
	PickTaskFromQueue(ctx context.Context, queueUrl string) ([]TaskWithReceipt, error)
	DeleteTaskFromQueue(ctx context.Context, queueUrl string, receiptHandle string) error
}

type ServiceContext struct {
	taskQueueUrl *string
}

type Service struct {
	Config     *Config
	Context    *ServiceContext
	logger     *logfx.Logger
	repository Repository
}

func NewService(config *Config, logger *logfx.Logger, repository Repository) *Service {
	return &Service{Config: config, logger: logger, repository: repository}
}

func (s *Service) Init(taskQueueUrl string) error {
	s.Context = &ServiceContext{taskQueueUrl: &taskQueueUrl}

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
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	if task.MaxTokens == 0 {
		task.MaxTokens = s.Config.DefaultMaxTokens
	}

	s.logger.Info("[Tasks] Dispatching task", "module", "tasks", "task", task)

	// Create initial TaskStatus
	initialStatus := &TaskStatus{
		TaskId:    task.Id,
		Status:    "pending_queue", // Indicates it's about to be queued
		CreatedAt: task.CreatedAt.Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	err := s.repository.PutTaskStatus(ctx, initialStatus)
	if err != nil {
		s.logger.ErrorContext(ctx, "[Tasks] Failed to create initial task status", "task_id", task.Id, "error", err)
		return fmt.Errorf("failed to create initial task status for %s: %w", task.Id, err)
	}

	err = s.repository.EnqueueTask(ctx, *s.Context.taskQueueUrl, task)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToSendMessage, err)
	}

	return nil
}

func (s *Service) PickUpNextAvailableTask(ctx context.Context, processFn func(ctx context.Context, task *Task) (TaskResult, error)) error {
	if s.Context == nil {
		return ErrDispatchTaskBeforeInit
	}

	records, err := s.repository.PickTaskFromQueue(ctx, *s.Context.taskQueueUrl)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToReceiveMessages, err)
	}

	for _, record := range records {
		s.logger.InfoContext(ctx, "[Tasks] Processing task from queue", "module", "tasks", "task_id", record.Task.Id)

		// Update task status to "in_progress"
		err = s.repository.UpdateTaskStatus(ctx, record.Task.Id, map[string]any{
			"Status":    "in_progress_worker",
			"UpdatedAt": time.Now().Unix(),
		})
		if err != nil {
			s.logger.ErrorContext(ctx, "[Tasks] Failed to update task status to in_progress_worker", "task_id", record.Task.Id, "error", err)
			// If status update fails, we might not want to proceed, or handle it gracefully.
			// For now, log and continue processing the task.
		}

		taskResult, processErr := processFn(ctx, record.Task)

		// Update task status based on the result from resources.Service.ProcessTask
		// Note: resources.Service.ProcessTask itself should update TaskStatus after batch creation.
		// The status update here is more about the SQS message handling outcome.

		finalStatusUpdate := map[string]any{
			"UpdatedAt": time.Now().Unix(),
		}

		if processErr != nil {
			s.logger.ErrorContext(ctx, "[Tasks] Error returned by task processing function", "module", "tasks", "task_id", record.Task.Id, "error", processErr)
			finalStatusUpdate["Status"] = "failed_worker_processing"
			finalStatusUpdate["LastError"] = processErr.Error()
			// Decide if message should be deleted or retried based on error type
		} else {
			switch taskResult {
			case TaskResultSuccess: // This means successfully submitted to batch by resources.Service
				s.logger.InfoContext(ctx, "[Tasks] Task successfully submitted to batch by worker", "module", "tasks", "task_id", record.Task.Id)
				// The actual completion status will be updated by the batch completion handler.
				// Status might have been set to "in_progress_batch" or similar by resources.Service.
				// We might not need to update status here if resources.Service handles it post-dispatch.
				// However, for clarity, let's assume resources.Service.ProcessTask updates the status to reflect batch submission.
				// If it doesn't, we'd set it here, e.g., finalStatusUpdate["Status"] = "submitted_to_batch_processor"

				// Delete message from SQS only if successfully handed off
				err = s.repository.DeleteTaskFromQueue(ctx, *s.Context.taskQueueUrl, record.ReceiptHandle)
				if err != nil {
					s.logger.ErrorContext(ctx, "[Tasks] Failed to delete SQS message after successful hand-off", "module", "tasks", "task_id", record.Task.Id, "receiptHandle", record.ReceiptHandle, "error", err)
					// This is problematic: task is processed but SQS message might reappear.
					// Idempotency in task processing is crucial.
				}
			case TaskResultSystemTemporarilyFailed:
				s.logger.WarnContext(ctx, "[Tasks] Task processing failed temporarily (system)", "module", "tasks", "task_id", record.Task.Id)
				finalStatusUpdate["Status"] = "retrying_worker_system_failure"
				// Do not delete SQS message, it will be retried after visibility timeout.
				// Increment retry count in TaskStatus.
				// TODO(@eser) Add retry count increment logic to UpdateTaskStatus or handle here.
			case TaskResultMessageTemporarilyFailed: // e.g. bad input that might be fixed
				s.logger.WarnContext(ctx, "[Tasks] Task processing failed temporarily (message)", "module", "tasks", "task_id", record.Task.Id)
				finalStatusUpdate["Status"] = "retrying_worker_message_failure"
				// Do not delete SQS message.
			case TaskResultMessagePermanentlyFailed:
				s.logger.ErrorContext(ctx, "[Tasks] Task processing failed permanently (message)", "module", "tasks", "task_id", record.Task.Id)
				finalStatusUpdate["Status"] = "failed_worker_message_permanent"
				// Delete SQS message as it cannot be processed.
				err = s.repository.DeleteTaskFromQueue(ctx, *s.Context.taskQueueUrl, record.ReceiptHandle)
				if err != nil {
					s.logger.ErrorContext(ctx, "[Tasks] Failed to delete SQS message for permanently failed task", "module", "tasks", "task_id", record.Task.Id, "error", err)
				}
			default:
				s.logger.ErrorContext(ctx, "[Tasks] Unknown task result from worker", "module", "tasks", "task_id", record.Task.Id, "result", taskResult)
				finalStatusUpdate["Status"] = "failed_worker_unknown_result"
				// Delete SQS message to prevent reprocessing an unknown state.
				err = s.repository.DeleteTaskFromQueue(ctx, *s.Context.taskQueueUrl, record.ReceiptHandle)
				if err != nil {
					s.logger.ErrorContext(ctx, "[Tasks] Failed to delete SQS message for task with unknown result", "module", "tasks", "task_id", record.Task.Id, "error", err)
				}
			}
		}

		// Update TaskStatus with the final outcome of this processing attempt.
		if len(finalStatusUpdate) > 1 { // More than just UpdatedAt
			err = s.repository.UpdateTaskStatus(ctx, record.Task.Id, finalStatusUpdate)
			if err != nil {
				s.logger.ErrorContext(ctx, "[Tasks] Failed to update final task status after processing attempt", "task_id", record.Task.Id, "error", err)
			}
		}
		// }() // End of removed go func
	}

	return nil
}
