package vertex_batch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-querystring/query"
)

// Client is the Vertex AI Batch Prediction API client.
type Client struct {
	httpClient *http.Client
	config     Config
}

// NewClient creates a new Vertex AI Batch Prediction API client.
func NewClient(config Config) *Client {
	baseURL := config.BaseURL
	if baseURL == "" {
		if config.Region == "" {
			// Default region if not set, though ProjectID and Region are crucial.
			// This case should ideally be validated before client creation.
			config.Region = "us-central1" // Or handle error
		}
		baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1", config.Region)
	}
	config.BaseURL = baseURL // Store the resolved baseURL

	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: config.Timeout},
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	// Ensure path starts with a slash if it's not already part of a full URL replacement
	urlStr := c.config.BaseURL
	if !strings.HasSuffix(urlStr, "/") && !strings.HasPrefix(path, "/") {
		urlStr += "/"
	}
	// If path is a full resource name like projects/../locations/../batchPredictionJobs/..,
	// it might not need BaseURL prefixing in the same way or newRequest path would be just the resource name.
	// Assuming path is relative to BaseURL for now e.g. /projects/../batchPredictionJobs
	// If path starts with 'projects/', it's likely a full resource name for GET/CANCEL.
	// In that case, the path should be appended directly to the BaseURL's host part + /v1/
	// For simplicity here, ensuring the path parameter to newRequest is correctly formed by the caller.
	reqURL := urlStr + path

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if c.config.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIToken)
	}
	req.Header.Set("Content-Type", "application/json") // Default, can be overridden
	return req, nil
}

func (c *Client) do(req *http.Request, v any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		// Attempt to parse as GoogleRpcStatus for better error messages
		var googleErr GoogleRpcStatus
		if json.Unmarshal(bodyBytes, &googleErr) == nil && googleErr.Message != "" {
			return fmt.Errorf("api request failed with status %d: %s (code: %d)", resp.StatusCode, googleErr.Message, googleErr.Code)
		}
		return fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			// Handle cases where response might be empty (e.g. 204 No Content or successful POST with no body)
			if err == io.EOF && (resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK) {
				return nil // Expected empty body
			}
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

// parentPath returns the base path for project and location specific resources.
func (c *Client) parentPath() string {
	return fmt.Sprintf("/projects/%s/locations/%s", c.config.ProjectID, c.config.Region)
}

// CreateBatchPredictionJob creates a new batch prediction job.
// The BatchPredictionJob in jobReq must have DisplayName, Model, InputConfig, and OutputConfig.
func (c *Client) CreateBatchPredictionJob(ctx context.Context, jobReq CreateBatchPredictionJobRequest) (*BatchPredictionJob, error) {
	if c.config.ProjectID == "" || c.config.Region == "" {
		return nil, fmt.Errorf("ProjectID and Region must be set in config")
	}
	path := fmt.Sprintf("%s/batchPredictionJobs", c.parentPath())

	jsonBody, err := json.Marshal(jobReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create batch prediction job request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	var job BatchPredictionJob
	if err := c.do(req, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// GetBatchPredictionJob retrieves a specific batch prediction job.
// jobName is the full resource name, e.g., "projects/{project}/locations/{location}/batchPredictionJobs/{id}".
func (c *Client) GetBatchPredictionJob(ctx context.Context, jobName string) (*BatchPredictionJob, error) {
	// The jobName is already the full path component after /v1/
	// e.g., projects/my-project/locations/us-central1/batchPredictionJobs/123456789
	path := "/" + jobName
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var job BatchPredictionJob
	if err := c.do(req, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// ListBatchPredictionJobs lists batch prediction jobs for the configured project and region.
func (c *Client) ListBatchPredictionJobs(ctx context.Context, params *ListBatchPredictionJobsParams) (*ListBatchPredictionJobsResponse, error) {
	if c.config.ProjectID == "" || c.config.Region == "" {
		return nil, fmt.Errorf("ProjectID and Region must be set in config")
	}
	basePath := fmt.Sprintf("%s/batchPredictionJobs", c.parentPath())

	var fullPath string
	if params != nil {
		qValues, err := query.Values(params)
		if err != nil {
			return nil, fmt.Errorf("failed to encode query params: %w", err)
		}
		if qValues.Encode() != "" {
			fullPath = basePath + "?" + qValues.Encode()
		} else {
			fullPath = basePath
		}
	} else {
		fullPath = basePath
	}

	req, err := c.newRequest(ctx, http.MethodGet, fullPath, nil)
	if err != nil {
		return nil, err
	}

	var response ListBatchPredictionJobsResponse
	if err := c.do(req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// CancelBatchPredictionJob cancels an in-progress batch prediction job.
// jobName is the full resource name, e.g., "projects/{project}/locations/{location}/batchPredictionJobs/{id}".
// The response is an empty JSON object upon success (handled by c.do).
func (c *Client) CancelBatchPredictionJob(ctx context.Context, jobName string) error {
	path := "/" + jobName + ":cancel"

	// According to Google API standards, POST for cancel might need an empty JSON body {}
	emptyBody := bytes.NewBuffer([]byte{'{', '}'})

	req, err := c.newRequest(ctx, http.MethodPost, path, emptyBody)
	if err != nil {
		return err
	}

	// Expects an empty response on success. The `do` method handles this.
	return c.do(req, nil)
}
