package openai_batch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-querystring/query"
)

// Client is the OpenAI Batch API client.
type Client struct {
	httpClient *http.Client
	config     Config
}

// NewClient creates a new OpenAI Batch API client.
func NewClient(config Config) *Client {
	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: config.Timeout},
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	reqURL := c.config.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
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
		return fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

// CreateFile uploads a file that can be used across OpenAI services.
// The file path provided should be an absolute path or relative to the execution directory.
func (c *Client) CreateFile(ctx context.Context, filePath string, purpose string) (*File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close() //nolint:errcheck

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add purpose field
	if err := writer.WriteField("purpose", purpose); err != nil {
		return nil, fmt.Errorf("failed to write purpose to multipart form: %w", err)
	}

	// Add file field
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/files", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	var resultFile File
	if err := c.do(req, &resultFile); err != nil {
		return nil, err
	}
	return &resultFile, nil
}

// CreateBatch creates and executes a batch from an uploaded file.
func (c *Client) CreateBatch(ctx context.Context, batchReq CreateBatchRequest) (*Batch, error) {
	jsonBody, err := json.Marshal(batchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create batch request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/batches", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var batch Batch
	if err := c.do(req, &batch); err != nil {
		return nil, err
	}
	return &batch, nil
}

// RetrieveBatch retrieves a batch.
func (c *Client) RetrieveBatch(ctx context.Context, batchID string) (*Batch, error) {
	path := fmt.Sprintf("/batches/%s", batchID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var batch Batch
	if err := c.do(req, &batch); err != nil {
		return nil, err
	}
	return &batch, nil
}

// CancelBatch cancels an in-progress batch.
func (c *Client) CancelBatch(ctx context.Context, batchID string) (*Batch, error) {
	path := fmt.Sprintf("/batches/%s/cancel", batchID)
	req, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}

	var batch Batch
	if err := c.do(req, &batch); err != nil {
		return nil, err
	}
	return &batch, nil
}

// ListBatches lists your organization's batches.
func (c *Client) ListBatches(ctx context.Context, params *ListBatchesParams) (*ListBatchesResponse, error) {
	path := "/batches"
	if params != nil {
		q, err := query.Values(params)
		if err != nil {
			return nil, fmt.Errorf("failed to encode query params: %w", err)
		}
		if q.Encode() != "" {
			path = path + "?" + q.Encode()
		}
	}

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response ListBatchesResponse
	if err := c.do(req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
