package vertex_batch

import "time"

// JobState represents the state of a BatchPredictionJob or other long-running operations.
type JobState string

const (
	JobStateUnspecified JobState = "JOB_STATE_UNSPECIFIED"
	JobStateQueued      JobState = "JOB_STATE_QUEUED"
	JobStatePending     JobState = "JOB_STATE_PENDING"
	JobStateRunning     JobState = "JOB_STATE_RUNNING"
	JobStateSucceeded   JobState = "JOB_STATE_SUCCEEDED"
	JobStateFailed      JobState = "JOB_STATE_FAILED"
	JobStateCancelling  JobState = "JOB_STATE_CANCELLING"
	JobStateCancelled   JobState = "JOB_STATE_CANCELLED"
	JobStatePaused      JobState = "JOB_STATE_PAUSED"
	JobStateExpired     JobState = "JOB_STATE_EXPIRED"
	JobStateUpdating    JobState = "JOB_STATE_UPDATING"
)

// GcsSource specifies a GCS location for input data.
type GcsSource struct {
	Uris []string `json:"uris,omitempty"` // A list of GCS URIs.
}

// BigQuerySource specifies a BigQuery table for input data.
type BigQuerySource struct {
	InputURI string `json:"inputUri,omitempty"` // Format: bq://project.dataset.table
}

// BatchPredictionJobInputConfig defines the input configuration for a batch prediction job.
type BatchPredictionJobInputConfig struct {
	GcsSource       *GcsSource      `json:"gcsSource,omitempty"`
	BigQuerySource  *BigQuerySource `json:"bigquerySource,omitempty"`
	InstancesFormat string          `json:"instancesFormat,omitempty"` // e.g., "jsonl", "csv", "bigquery", "file-list"
}

// GcsDestination specifies a GCS location for output data.
type GcsDestination struct {
	OutputUriPrefix string `json:"outputUriPrefix,omitempty"` // GCS URI prefix for output files.
}

// BigQueryDestination specifies a BigQuery table for output data.
type BigQueryDestination struct {
	OutputURI string `json:"outputUri,omitempty"` // Format: bq://project.dataset.table
}

// BatchPredictionJobOutputConfig defines the output configuration for a batch prediction job.
type BatchPredictionJobOutputConfig struct {
	GcsDestination      *GcsDestination      `json:"gcsDestination,omitempty"`
	BigQueryDestination *BigQueryDestination `json:"bigqueryDestination,omitempty"`
	PredictionsFormat   string               `json:"predictionsFormat,omitempty"` // e.g., "jsonl", "csv", "bigquery"
}

// MachineSpec defines the machine type and accelerator for dedicated resources.
type MachineSpec struct {
	MachineType      string `json:"machineType,omitempty"`     // e.g., "n1-standard-2"
	AcceleratorType  string `json:"acceleratorType,omitempty"` // e.g., "NVIDIA_TESLA_K80"
	AcceleratorCount int    `json:"acceleratorCount,omitempty"`
}

// BatchDedicatedResources defines dedicated resources for a batch prediction job.
type BatchDedicatedResources struct {
	MachineSpec          *MachineSpec `json:"machineSpec,omitempty"`
	StartingReplicaCount int          `json:"startingReplicaCount,omitempty"`
	MaxReplicaCount      int          `json:"maxReplicaCount,omitempty"`
}

// GoogleRpcStatus defines the structure for error details from Google APIs.
type GoogleRpcStatus struct {
	Message string           `json:"message,omitempty"`
	Details []map[string]any `json:"details,omitempty"`
	Code    int              `json:"code,omitempty"`
}

// BatchPredictionJobOutputInfo contains information about the output of a completed job.
type BatchPredictionJobOutputInfo struct {
	GcsOutputDirectory    string `json:"gcsOutputDirectory,omitempty"`    // Output only. GCS output directory.
	BigqueryOutputDataset string `json:"bigqueryOutputDataset,omitempty"` // Output only. BigQuery output dataset.
	BigqueryOutputTable   string `json:"bigqueryOutputTable,omitempty"`   // Output only. BigQuery output table.
}

// BatchPredictionJob represents a Vertex AI batch prediction job resource.
type BatchPredictionJob struct {
	CreateTime          time.Time                       `json:"createTime"`                // Output only.
	StartTime           time.Time                       `json:"startTime"`                 // Output only.
	EndTime             time.Time                       `json:"endTime"`                   // Output only.
	UpdateTime          time.Time                       `json:"updateTime"`                // Output only.
	ModelParameters     map[string]any                  `json:"modelParameters,omitempty"` // google.protobuf.Value
	InputConfig         *BatchPredictionJobInputConfig  `json:"inputConfig"`               // Required.
	OutputConfig        *BatchPredictionJobOutputConfig `json:"outputConfig"`              // Required.
	DedicatedResources  *BatchDedicatedResources        `json:"dedicatedResources,omitempty"`
	Labels              map[string]string               `json:"labels,omitempty"`
	Error               *GoogleRpcStatus                `json:"error,omitempty"`      // Output only.
	OutputInfo          *BatchPredictionJobOutputInfo   `json:"outputInfo,omitempty"` // Output only.
	Name                string                          `json:"name,omitempty"`       // Output only. Resource name.
	DisplayName         string                          `json:"displayName"`          // Required. User-defined display name.
	Model               string                          `json:"model"`                // Required. Model resource name.
	ServiceAccount      string                          `json:"serviceAccount,omitempty"`
	State               JobState                        `json:"state,omitempty"`           // Output only.
	PartialFailures     []*GoogleRpcStatus              `json:"partialFailures,omitempty"` // Output only
	GenerateExplanation bool                            `json:"generateExplanation,omitempty"`
}

// CreateBatchPredictionJobRequest is used as the request body when creating a job.
// The `parent` path parameter (projects/*/locations/*) is part of the URL.
type CreateBatchPredictionJobRequest struct {
	BatchPredictionJob *BatchPredictionJob `json:"batchPredictionJob"` // The BatchPredictionJob to create.
}

// ListBatchPredictionJobsResponse is the response for listing batch prediction jobs.
type ListBatchPredictionJobsResponse struct {
	NextPageToken       string                `json:"nextPageToken,omitempty"`
	BatchPredictionJobs []*BatchPredictionJob `json:"batchPredictionJobs,omitempty"`
}

// ListBatchPredictionJobsParams defines query parameters for listing jobs.
type ListBatchPredictionJobsParams struct {
	Filter    *string `url:"filter,omitempty"`
	PageSize  *int    `url:"pageSize,omitempty"`
	PageToken *string `url:"pageToken,omitempty"`
	ReadMask  *string `url:"readMask,omitempty"` // Fields to include in the response
}
