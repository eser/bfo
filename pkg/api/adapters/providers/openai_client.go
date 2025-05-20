package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/eser/ajan/logfx"
	"github.com/eser/bfo/pkg/api/business/resources"
	"github.com/google/go-querystring/query"
)

var _ Provider = (*OpenAiClient)(nil)

type OpenAiClient struct {
	config     *resources.ConfigResource
	logger     *logfx.Logger
	httpClient *http.Client
}

func NewOpenAiClient(config *resources.ConfigResource, logger *logfx.Logger) *OpenAiClient {
	return &OpenAiClient{
		config: config,
		logger: logger,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout,
		},
	}
}

func (c *OpenAiClient) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	reqURL := c.config.BaseUrl + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.ApiKey)
	return req, nil
}

func (c *OpenAiClient) do(req *http.Request, v any) error {
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
// It now accepts content as bytes and a filename.
func (c *OpenAiClient) CreateFile(ctx context.Context, fileName string, content []byte, purpose string) (*resources.File, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add purpose field
	if err := writer.WriteField("purpose", purpose); err != nil {
		return nil, fmt.Errorf("failed to write purpose to multipart form: %w", err)
	}

	// Add file field
	part, err := writer.CreateFormFile("file", fileName) // Use the provided fileName
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = io.Copy(part, bytes.NewReader(content)) // Use bytes.NewReader for content
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

	var resultFile resources.File
	if err := c.do(req, &resultFile); err != nil {
		return nil, err
	}
	return &resultFile, nil
}

// GetFileContent retrieves the content of a specific file from OpenAI.
func (c *OpenAiClient) GetFileContent(ctx context.Context, fileId string) ([]byte, error) {
	path := fmt.Sprintf("/files/%s/content", fileId)

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for getting file content: %w", err)
	}

	// The OpenAI API for file content might return it directly, not as JSON.
	// So, we handle the response differently from the standard `do` method.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request for getting file content: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api request for file content failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content from response: %w", err)
	}

	return content, nil
}

// CreateBatch creates and executes a batch from an uploaded file.
func (c *OpenAiClient) CreateBatch(ctx context.Context, batchReq resources.CreateBatchRequest) (*resources.Batch, error) {
	jsonBody, err := json.Marshal(batchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create batch request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/batches", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var batch resources.Batch
	if err := c.do(req, &batch); err != nil {
		return nil, err
	}
	return &batch, nil
}

// RetrieveBatch retrieves a batch.
func (c *OpenAiClient) RetrieveBatch(ctx context.Context, batchId string) (*resources.Batch, error) {
	path := fmt.Sprintf("/batches/%s", batchId)

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var batch resources.Batch
	if err := c.do(req, &batch); err != nil {
		return nil, err
	}

	return &batch, nil
}

// CancelBatch cancels an in-progress batch.
func (c *OpenAiClient) CancelBatch(ctx context.Context, batchId string) (*resources.Batch, error) {
	path := fmt.Sprintf("/batches/%s/cancel", batchId)

	req, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}

	var batch resources.Batch
	if err := c.do(req, &batch); err != nil {
		return nil, err
	}

	return &batch, nil
}

func (c *OpenAiClient) ListBatches(ctx context.Context, params *resources.ListBatchesParams) (*resources.ListBatchesResponse, error) {
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

	var response resources.ListBatchesResponse
	if err := c.do(req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
