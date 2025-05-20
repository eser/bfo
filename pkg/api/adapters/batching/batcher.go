package batching

import "context"

//go:generate go tool mockery --name=Batcher --inpackage --inpackage-suffix --case=underscore --structname=MockClient --filename=mock_client.go
type Batcher interface {
	CreateFile(ctx context.Context, filePath string, purpose string) (*File, error)
	CreateBatch(ctx context.Context, batchReq CreateBatchRequest) (*Batch, error)
	RetrieveBatch(ctx context.Context, batchID string) (*Batch, error)
	CancelBatch(ctx context.Context, batchID string) (*Batch, error)
	ListBatches(ctx context.Context, params *ListBatchesParams) (*ListBatchesResponse, error)
}
