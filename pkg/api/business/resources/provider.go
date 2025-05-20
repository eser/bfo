package resources

import "context"

const (
	// File purpose for batch operations
	FilePurposeBatch = "batch"

	// Batch completion window
	BatchCompletionWindow24h = "24h"
)

// File represents an OpenAI file object.
type File struct {
	Id            string `json:"id"`
	Object        string `json:"object"` // "file"
	Filename      string `json:"filename"`
	Purpose       string `json:"purpose"` // e.g., "batch"
	Status        string `json:"status"`  // e.g., "uploaded", "processed", "error"
	StatusDetails string `json:"status_details,omitempty"`
	Bytes         int    `json:"bytes"`
	CreatedAt     int64  `json:"created_at"`
}

// CreateBatchRequest defines the request body for creating a batch.
type CreateBatchRequest struct {
	Metadata         map[string]string `json:"metadata,omitempty"`
	InputFileId      string            `json:"input_file_id"`
	Endpoint         string            `json:"endpoint"`          // e.g., "/v1/chat/completions"
	CompletionWindow string            `json:"completion_window"` // Currently only "24h"
}

// BatchError provides details about errors in a batch request if any.
// This is a sub-structure within the Batch object's 'errors' field.
type BatchError struct {
	Line    *int   `json:"line,omitempty"` // Pointer to allow null
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Param   string `json:"param,omitempty"`
}

// BatchErrors holds error data for a batch.
// This is the structure for the 'errors' field in the Batch object.
type BatchErrors struct {
	Object string       `json:"object,omitempty"` // e.g. "list"
	Data   []BatchError `json:"data,omitempty"`
}

// BatchRequestCounts details the number of requests in a batch by status.
type BatchRequestCounts struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

// Batch represents an OpenAI batch object.
type Batch struct {
	Errors           *BatchErrors       `json:"errors,omitempty"`         // Pointer to allow null
	OutputFileId     *string            `json:"output_file_id,omitempty"` // Pointer to allow null
	ErrorFileId      *string            `json:"error_file_id,omitempty"`  // Pointer to allow null
	InProgressAt     *int64             `json:"in_progress_at,omitempty"` // Pointer to allow null
	ExpiresAt        *int64             `json:"expires_at,omitempty"`     // Pointer to allow null
	FinalizingAt     *int64             `json:"finalizing_at,omitempty"`  // Pointer to allow null
	CompletedAt      *int64             `json:"completed_at,omitempty"`   // Pointer to allow null
	FailedAt         *int64             `json:"failed_at,omitempty"`      // Pointer to allow null
	CancelledAt      *int64             `json:"cancelled_at,omitempty"`   // Pointer to allow null
	Metadata         map[string]string  `json:"metadata,omitempty"`
	Id               string             `json:"id"`
	Object           string             `json:"object"` // "batch"
	Endpoint         string             `json:"endpoint"`
	InputFileId      string             `json:"input_file_id"`
	CompletionWindow string             `json:"completion_window"`
	Status           string             `json:"status"` // e.g., "validating", "failed", "in_progress", "finalizing", "completed", "expired", "cancelling", "cancelled"
	RequestCounts    BatchRequestCounts `json:"request_counts"`
	CreatedAt        int64              `json:"created_at"`
}

// ListBatchesResponse defines the response for listing batches.
type ListBatchesResponse struct {
	FirstId *string `json:"first_id,omitempty"`
	LastId  *string `json:"last_id,omitempty"`
	Object  string  `json:"object"` // "list"
	Data    []Batch `json:"data"`
	HasMore bool    `json:"has_more"`
}

// ListBatchesParams defines query parameters for listing batches.
// Note: these are query params, not a request body.
type ListBatchesParams struct {
	After *string `url:"after,omitempty"`
	Limit *int    `url:"limit,omitempty"`
}

type Provider interface {
	CreateFile(ctx context.Context, filePath string, purpose string) (*File, error)
	CreateBatch(ctx context.Context, batchReq CreateBatchRequest) (*Batch, error)
	RetrieveBatch(ctx context.Context, batchId string) (*Batch, error)
	CancelBatch(ctx context.Context, batchId string) (*Batch, error)
	ListBatches(ctx context.Context, params *ListBatchesParams) (*ListBatchesResponse, error)
}
