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

var _ Provider = (*EchoClient)(nil)

type EchoClient struct {
	config *resources.ConfigResource
	logger *logfx.Logger
}

func NewEchoClient(config *resources.ConfigResource, logger *logfx.Logger) *EchoClient {
	return &EchoClient{
		config: config,
		logger: logger,
	}
}

func (c *EchoClient) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	reqUrl := c.config.BaseUrl + path
	req, err := http.NewRequestWithContext(ctx, method, reqUrl, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.ApiKey)
	return req, nil
}

func (c *EchoClient) do(req *http.Request, _v any) error {
	c.logger.Info("[EchoClient] Sending request", "module", "providers", "method", req.Method, "url", req.URL.String())

	return nil
}

// CreateFile uploads content to be used across OpenAI services.
func (c *EchoClient) CreateFile(ctx context.Context, fileName string, content []byte, purpose string) (*resources.File, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add purpose field
	if err := writer.WriteField("purpose", purpose); err != nil {
		return nil, fmt.Errorf("failed to write purpose to multipart form: %w", err)
	}

	// Add file field
	part, err := writer.CreateFormFile("file", fileName) // Use fileName passed as argument
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = io.Copy(part, bytes.NewReader(content)) // Use content passed as argument
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

// GetFileContent retrieves the content of a specific file.
// Note: EchoClient is a mock and might not have a real file store.
// This implementation will return an error indicating it's not implemented.
func (c *EchoClient) GetFileContent(ctx context.Context, fileId string) ([]byte, error) {
	c.logger.Warn("[EchoClient] GetFileContent called, but it's not implemented for EchoClient", "module", "providers", "file_id", fileId)
	return nil, fmt.Errorf("GetFileContent is not implemented for EchoClient")
}

// CreateBatch creates and executes a batch from an uploaded file.
func (c *EchoClient) CreateBatch(ctx context.Context, batchReq resources.CreateBatchRequest) (*resources.Batch, error) {
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
func (c *EchoClient) RetrieveBatch(ctx context.Context, batchId string) (*resources.Batch, error) {
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
func (c *EchoClient) CancelBatch(ctx context.Context, batchId string) (*resources.Batch, error) {
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

func (c *EchoClient) ListBatches(ctx context.Context, params *resources.ListBatchesParams) (*resources.ListBatchesResponse, error) {
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
