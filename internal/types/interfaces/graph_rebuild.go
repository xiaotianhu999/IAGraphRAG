package interfaces

import (
	"context"

	"github.com/hibiken/asynq"
)

// GraphRebuildService defines the interface for graph rebuild operations
type GraphRebuildService interface {
	// RebuildGraphAsync asynchronously triggers graph rebuild for a knowledge base
	RebuildGraphAsync(ctx context.Context, kbID string, modelID string, batchSize int) error

	// Handle processes the graph rebuild task
	Handle(ctx context.Context, t *asynq.Task) error
}
