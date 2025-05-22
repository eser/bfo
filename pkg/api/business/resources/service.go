package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eser/ajan/logfx"
	"github.com/eser/bfo/pkg/api/business/tasks"
	"github.com/oklog/ulid/v2"
)

type ProviderFn = func(config *ConfigResource) Provider

type Repository interface {
	GetResourceInstanceState(ctx context.Context, resourceInstanceId string) (*ResourceInstanceState, error)
	PutResourceInstanceState(ctx context.Context, state *ResourceInstanceState) error
}

type Service struct {
	config     *Config
	logger     *logfx.Logger
	repository Repository

	providers       map[string]ProviderFn
	resources       map[string]Provider       // map of resource_key to Provider instance
	resourceConfigs map[string]ConfigResource // map of resource_key to its config
}

func NewService(config *Config, logger *logfx.Logger, repository Repository) *Service {
	return &Service{
		config:     config,
		logger:     logger,
		repository: repository,

		providers:       make(map[string]ProviderFn),
		resources:       make(map[string]Provider),
		resourceConfigs: make(map[string]ConfigResource),
	}
}

func (s *Service) AddProvider(key string, providerFn ProviderFn) {
	s.providers[key] = providerFn

	s.logger.Debug("[Resources] Provider added", "module", "resources", "key", key)
}

func (s *Service) AddResource(key string, config ConfigResource) error {
	if config.Disabled {
		s.logger.Debug("[Resources] Resource is disabled, skipping...", "module", "resources", "key", key, "config", config)

		return nil
	}

	providerFn, okProviderFn := s.providers[config.Provider]
	if !okProviderFn {
		return fmt.Errorf("provider '%s' not found for resource '%s'", config.Provider, key)
	}

	s.resources[key] = providerFn(&config)
	s.resourceConfigs[key] = config // Store the config for later use

	s.logger.Debug("[Resources] Resource added", "module", "resources", "key", key, "config", config)

	// Initialize resource state if not present
	_, err := s.repository.GetResourceInstanceState(context.Background(), key) // Using key as ResourceInstanceId for simplicity
	if err != nil {
		// Assuming error means not found, create a new state
		// In a real scenario, you'd check the specific error type (e.g., worker_pool_store.ErrStateNotFound)
		initialState := &ResourceInstanceState{
			ResourceInstanceId:   key,
			ProviderName:         config.Provider,
			LastActivityTime:     time.Now().Unix(),
			CurrentTokenLoad:     0,
			ActiveBatches:        0,
			MaxConcurrentBatches: 8,    // Corrected to use MaxConcurrency
			TokensPerMinute:      1000, // Assuming this field exists in ConfigResource
			MaxTokensPerBatch:    1000, // Assuming this field exists in ConfigResource
			Version:              1,
		}
		err = s.repository.PutResourceInstanceState(context.Background(), initialState)
		if err != nil {
			s.logger.Error("[Resources] Failed to initialize resource state", "key", key, "error", err)
			return fmt.Errorf("failed to initialize resource state for %s: %w", key, err)
		}
		s.logger.Info("[Resources] Initialized resource state", "key", key)
	}

	return nil
}

func (s *Service) Init() error {
	s.logger.Debug("[Resources] Loading resources from config", "module", "resources")

	for key, resourceConfig := range *s.config {
		err := s.AddResource(key, resourceConfig)

		if err != nil {
			return err
		}
	}

	s.logger.Debug("[Resources] Resources loaded from config", "module", "resources")

	return nil
}

// estimateTokens estimates the number of tokens for a given task.
// This is a placeholder and should be replaced with a more accurate tokenizer.
func (s *Service) estimateTokens(task *tasks.Task) int {
	// Simple estimation: sum of characters in messages / 4 (common rough estimate)
	// For a robust solution, use a proper tokenizer library specific to the LLM provider.
	// e.g., for OpenAI: https://github.com/tiktoken-go/tiktoken-go
	charCount := 0
	for _, msg := range task.Messages {
		charCount += len(msg.Content)
	}
	estimated := charCount / 4 // Very rough estimate
	s.logger.DebugContext(context.Background(), "[Resources] Estimated tokens for task", "task_id", task.Id, "estimated_tokens", estimated, "char_count", charCount)
	return estimated
}

func (s *Service) FindBestAvailableResource(ctx context.Context, task *tasks.Task) (Provider, string, error) {
	s.logger.DebugContext(ctx, "[Resources] Finding best available resource", "module", "resources", "task_id", task.Id)

	estimatedTokensForTask := s.estimateTokens(task) // Use the new estimateTokens method
	if task.EstimatedInputTokens == 0 {              // If not pre-calculated, use our estimation
		task.EstimatedInputTokens = estimatedTokensForTask
	}

	var bestResource Provider
	var bestResourceKey string
	// TODO(@eser) Implement more sophisticated scoring (e.g., least loaded, cheapest, specific capabilities)
	// For now, pick the first one that can handle the task based on token limits and concurrency.

	for key, resource := range s.resources {
		s.logger.DebugContext(ctx, "[Resources] Checking resource", "module", "resources", "resource_key", key)

		resourceState, err := s.repository.GetResourceInstanceState(ctx, key)
		if err != nil {
			s.logger.ErrorContext(ctx, "[Resources] Failed to get resource state", "resource_key", key, "error", err)
			continue // Skip this resource if we can't get its state
		}

		// Ensure resourceState is not nil before proceeding
		if resourceState == nil {
			s.logger.ErrorContext(ctx, "[Resources] Resource state is nil, but no error was returned", "resource_key", key)
			continue // Skip this resource if its state is unexpectedly nil
		}

		// Check if resource can handle the estimated tokens for this task
		if int64(task.EstimatedInputTokens+task.MaxTokens) > resourceState.MaxTokensPerBatch {
			s.logger.DebugContext(ctx, "[Resources] Resource cannot handle token size", "resource_key", key, "task_tokens", task.EstimatedInputTokens+task.MaxTokens, "max_tokens_per_batch", resourceState.MaxTokensPerBatch)
			continue
		}

		// Check concurrency limits
		if resourceState.ActiveBatches >= resourceState.MaxConcurrentBatches {
			s.logger.DebugContext(ctx, "[Resources] Resource at max concurrent batches", "resource_key", key, "active_batches", resourceState.ActiveBatches, "max_concurrent_batches", resourceState.MaxConcurrentBatches)
			continue
		}

		// TODO(@eser) Check token rate limits (TokensPerMinute) - this is more complex as it involves time windows.
		// For now, we assume if it passes concurrency and batch size, it's a candidate.

		// This resource is a candidate. For now, take the first one.
		// A more advanced strategy would score them and pick the best.
		bestResource = resource
		bestResourceKey = key
		s.logger.DebugContext(ctx, "[Resources] Found candidate resource", "module", "resources", "resource_key", key)
		break // Found a suitable resource
	}

	if bestResource == nil {
		s.logger.WarnContext(ctx, "[Resources] No suitable resource found for task", "task_id", task.Id, "estimated_tokens", task.EstimatedInputTokens)
		return nil, "", fmt.Errorf("no suitable resource found for task Id '%s'", task.Id) // Return an error
	}

	return bestResource, bestResourceKey, nil
}

func (s *Service) DispatchTask(ctx context.Context, resourceKey string, provider Provider, task *tasks.Task) (*Batch, error) {
	// 1. Prepare the batch input file (this is a simplified representation)
	// In a real scenario, you'd create a JSONL file with multiple task requests if batching multiple LLM calls.
	// For a single task dispatch, the input file might contain just that one task's data formatted for the provider.
	// This part is highly dependent on the specific LLM provider's batch API requirements.

	// For OpenAI, a batch input file looks like:
	// {"custom_id": "<request-1>", "method": "POST", "url": "/v1/chat/completions", "body": {"model": "gpt-3.5-turbo", "messages": [{"role": "system", "content": "..."}, {"role": "user", "content": "..."}]}}
	// {"custom_id": "<request-2>", ...}

	// This example assumes we are creating a batch for a single task.
	// You'll need a mechanism to create this input file and upload it.
	// Let's assume `provider.CreateFile` handles creating and uploading the content.
	// The content itself needs to be constructed based on `task`.

	// Placeholder for input file creation logic:
	// This would involve serializing the task into the provider-specific format for a batch request line.
	inputFileContent := fmt.Sprintf(
		`{"custom_id": "%s", "method": "POST", "url": "%s", "body": {"model": "%s", "messages": %s, "max_tokens": %d}}`,
		task.Id,                              // custom_id
		"/v1/chat/completions",               // url (e.g., /v1/chat/completions)
		"gpt-3.5-turbo",                      // model (e.g., gpt-3.5-turbo)
		convertMessagesToJSON(task.Messages), // messages
		task.MaxTokens,
	)

	// Create a temporary file with this content or pass content directly if API supports
	// For simplicity, let's assume CreateFile takes content as a string for now.
	// In reality, it would take a file path or io.Reader.
	// This is a conceptual step. The actual `CreateFile` in `provider.go` needs to be implemented
	// to upload this content to the provider (e.g., OpenAI files API).

	// For now, let's skip the actual file creation and assume we have an input_file_id
	// This is a MAJOR simplification and needs proper implementation.
	// You would call something like: inputFile, err := provider.CreateFile(ctx, inputFileContentBytes, FilePurposeBatch)
	// if err != nil { ... }
	// inputFileId := inputFile.Id
	inputFileId := "fake-input-file-id-" + task.Id // Placeholder!
	s.logger.InfoContext(ctx, "[Resources] Conceptual input file created for batch", "task_id", task.Id, "input_file_id", inputFileId, "content_snippet", inputFileContent[:min(100, len(inputFileContent))])

	batchReq := CreateBatchRequest{
		InputFileId:      inputFileId,            // This Id must come from a successful file upload to the provider
		Endpoint:         "/v1/chat/completions", // e.g., "/v1/chat/completions"
		CompletionWindow: BatchCompletionWindow24h,
		Metadata: map[string]string{
			"task_id":      task.Id,
			"resource_key": resourceKey,
		},
	}

	// 2. Optimistically update resource state (before making the call)
	currentState, err := s.repository.GetResourceInstanceState(ctx, resourceKey)
	if err != nil {
		s.logger.ErrorContext(ctx, "[Resources] Failed to get resource state before dispatch", "resource_key", resourceKey, "error", err)
		return nil, fmt.Errorf("failed to get resource state for %s: %w", resourceKey, err)
	}

	newVersion := currentState.Version + 1
	currentState.ActiveBatches++
	currentState.CurrentTokenLoad += int64(task.EstimatedInputTokens + task.MaxTokens) // Add estimated total tokens for the batch
	currentState.LastActivityTime = time.Now().Unix()
	currentState.Version = newVersion

	err = s.repository.PutResourceInstanceState(ctx, currentState) // This should ideally use the version for optimistic lock
	if err != nil {
		s.logger.ErrorContext(ctx, "[Resources] Failed to update resource state before dispatch (optimistic lock might fail here)", "resource_key", resourceKey, "error", err)
		// TODO(@eser) Implement retry logic for optimistic locking failures if PutResourceInstanceState supports it.
		return nil, fmt.Errorf("failed to update resource state for %s: %w", resourceKey, err)
	}
	s.logger.InfoContext(ctx, "[Resources] Updated resource state before dispatch", "resource_key", resourceKey, "active_batches", currentState.ActiveBatches, "token_load", currentState.CurrentTokenLoad)

	// 3. Create the batch with the provider
	batch, err := provider.CreateBatch(ctx, batchReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "[Resources] Failed to create batch with provider", "task_id", task.Id, "resource_key", resourceKey, "error", err)
		// Attempt to roll back state update if batch creation fails
		currentState.ActiveBatches--
		currentState.CurrentTokenLoad -= int64(task.EstimatedInputTokens + task.MaxTokens)
		// No need to change LastActivityTime, an activity did occur.
		// Version increment for this rollback attempt.
		currentState.Version++
		rollbackErr := s.repository.PutResourceInstanceState(ctx, currentState)
		if rollbackErr != nil {
			s.logger.ErrorContext(ctx, "[Resources] CRITICAL: Failed to roll back resource state after batch creation failure", "resource_key", resourceKey, "error", rollbackErr)
			// This is a critical situation. The state in DB might be inconsistent.
		}
		return nil, fmt.Errorf("failed to create batch for task %s: %w", task.Id, err)
	}

	s.logger.InfoContext(ctx, "[Resources] Batch created successfully with provider", "task_id", task.Id, "batch_id", batch.Id, "resource_key", resourceKey)

	// The TODO for token calculation and state update is now handled optimistically before the call,
	// and will be finalized/adjusted when the batch completes or fails (see ProcessTask and a potential new method for handling batch webhooks/polling).

	return batch, nil
}

// Helper function to convert messages to JSON string for the batch body
// This is a simplified version. Error handling should be more robust.
func convertMessagesToJSON(messages []tasks.TaskMessage) string {
	// In a real implementation, ensure this matches the exact structure expected by the LLM's batch API.
	// This might involve creating a temporary struct that mirrors the LLM API's message format.
	type BatchMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	batchMessages := make([]BatchMessage, len(messages))
	for i, m := range messages {
		batchMessages[i] = BatchMessage{Role: m.Role, Content: m.Content}
	}
	mjson, err := json.Marshal(batchMessages)
	if err != nil {
		// Log the error, but for this simplified version, return an empty array string
		// In production, this should probably return an error up the call stack.
		fmt.Printf("Error marshalling messages for batch: %v\n", err) // Basic logging
		return "[]"
	}
	return string(mjson)
}

func (s *Service) TryProcessTask(ctx context.Context, task *tasks.Task) (tasks.TaskResult, error) {
	s.logger.DebugContext(ctx, "[Resources] Processing task", "module", "resources", "task_id", task.Id)

	// Generate a unique Id for the task if it doesn't have one
	if task.Id == "" {
		task.Id = ulid.Make().String()
		s.logger.InfoContext(ctx, "[Resources] Generated new Id for task", "task_id", task.Id)
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	// 1. Find the best available resource
	provider, resourceKey, err := s.FindBestAvailableResource(ctx, task)
	if err != nil {
		s.logger.ErrorContext(ctx, "[Resources] Failed to find available resource for task", "task_id", task.Id, "error", err)
		// TODO(@eser) Update TaskStatus to reflect failure or need for retry (e.g., no resource available)
		return tasks.TaskResultSystemTemporarilyFailed, fmt.Errorf("failed to find available resource for task %s: %w", task.Id, err)
	}
	s.logger.InfoContext(ctx, "[Resources] Found best resource for task", "task_id", task.Id, "resource_key", resourceKey)

	// 2. Dispatch the task to the chosen resource (creates a batch)
	batch, err := s.DispatchTask(ctx, resourceKey, provider, task)
	if err != nil {
		s.logger.ErrorContext(ctx, "[Resources] Failed to dispatch task / create batch", "task_id", task.Id, "resource_key", resourceKey, "error", err)
		// TODO(@eser) Update TaskStatus to reflect failure (e.g., batch creation failed)
		// The resource state rollback is attempted within DispatchTask.
		return tasks.TaskResultSystemTemporarilyFailed, fmt.Errorf("failed to dispatch task %s: %w", task.Id, err)
	}

	s.logger.InfoContext(ctx, "[Resources] Task dispatched, batch created", "module", "resources", "task_id", task.Id, "batch_id", batch.Id, "resource_key", resourceKey)

	// TODO(@eser) Persist/Update Task Status
	// Now that a batch is created, update the TaskStatus in the TaskStatusStore.
	// Example:
	// taskStatus := &tasks.TaskStatus{
	// 	TaskId:             task.Id,
	// 	Status:             "in_progress", // Or "submitted_to_batch"
	// 	ResourceInstanceId: resourceKey,
	// 	BatchId:            batch.Id,
	// 	CreatedAt:          task.CreatedAt, // Or time.Now().Unix() if not set
	// 	UpdatedAt:          time.Now().Unix(),
	// 	Version:            1,
	// }
	// err = s.taskStatusStore.PutTaskStatus(ctx, taskStatus) // Assuming s.taskStatusStore is initialized
	// if err != nil { ... handle error ... }

	// The actual result of the LLM processing will come when the batch completes.
	// This ProcessTask method now primarily handles dispatching.
	// The original `TaskResultSuccess` here might mean "successfully submitted to batch processor".
	// A separate mechanism (webhook handler or poller) will be needed to get the final LLM output.

	return tasks.TaskResultSuccess, nil // Meaning: task successfully submitted for batch processing
}

// TODO(@eser) Add a new method like HandleBatchCompletion(ctx context.Context, batch Batch) error
// This method would be called by a webhook from the LLM provider or by a polling mechanism.
// It would:
// 1. Retrieve the batch output file(s) from provider.GetFileContent(batch.OutputFileId) and provider.GetFileContent(batch.ErrorFileId).
// 2. Parse the output and error files.
// 3. For each result in the output file:
//    a. Find the original Task (e.g., using custom_id from batch output line, which we set to task.Id).
//    b. Update the TaskStatus in TaskStatusStore with "completed" and the LLM output.
//    c. Potentially enqueue a follow-up action (e.g., notify user, store result elsewhere).
// 4. For each error in the error file:
//    a. Find the original Task.
//    b. Update TaskStatus with "failed" and the error details.
//    c. Implement retry logic if applicable (e.g., requeue task, increment retry count).
// 5. Update the ResourceInstanceState:
//    a. Decrement ActiveBatches.
//    b. Adjust CurrentTokenLoad (subtract tokens from this completed/failed batch).
//    c. Update LastActivityTime.
//    d. Use optimistic locking (Version).

// TODO(@eser) Add a polling mechanism if webhooks are not available/reliable for all providers.
// This poller would periodically call provider.ListBatches() and provider.RetrieveBatch()
// to check for status changes and then trigger HandleBatchCompletion.

// TODO(@eser) Implement token counting more accurately using a tokenizer library.
// The current estimateTokens is very basic.

// TODO(@eser) Implement more sophisticated resource selection in FindBestAvailableResource.
// This could involve a scoring system based on current load, cost, provider-specific limits,
// task priority, etc.

// TODO(@eser) Ensure robust error handling and retries for operations involving external providers
// and database interactions. This includes handling optimistic locking failures.

// TODO(@eser) Implement the actual file creation/upload logic in the provider adapters for CreateFile.
// The current DispatchTask has a placeholder for this.

// TODO(@eser) Add configuration for MaxConcurrentBatches, TokensPerMinute, MaxTokensPerBatch to ConfigResource
// and ensure they are used during resource initialization and selection.

// Helper for min for slicing strings safely
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
