package embedding

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"sync"

	"github.com/aiplusall/aiplusall-kb/internal/models/utils"
	"github.com/panjf2000/ants/v2"
)

type batchEmbedder struct {
	pool *ants.Pool
}

func NewBatchEmbedder(pool *ants.Pool) EmbedderPooler {
	return &batchEmbedder{pool: pool}
}

type textEmbedding struct {
	text    string
	results []float32
}

// sanitizeVector removes NaN and Inf values from embedding vectors
// Replaces invalid values with 0.0 to prevent JSON serialization errors
func sanitizeVector(vec []float32) ([]float32, error) {
	hasInvalid := false
	for i, v := range vec {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			vec[i] = 0.0
			hasInvalid = true
		}
	}
	if hasInvalid {
		return vec, fmt.Errorf("vector contained NaN or Inf values, replaced with 0.0")
	}
	return vec, nil
}

func (e *batchEmbedder) BatchEmbedWithPool(ctx context.Context, model Embedder, texts []string) ([][]float32, error) {
	// Create goroutine pool for concurrent processing of document chunks
	var wg sync.WaitGroup
	var mu sync.Mutex  // For synchronizing access to error
	var firstErr error // Record the first error that occurs
	batchSizeStr := os.Getenv("BATCH_EMBED_SIZE")
	if batchSizeStr == "" {
		batchSizeStr = "5"
	}
	batchSize, err := strconv.Atoi(batchSizeStr)
	if err != nil {
		return nil, err
	}
	textEmbeddings := utils.MapSlice(texts, func(text string) *textEmbedding {
		return &textEmbedding{text: text}
	})

	// Function to process each document chunk
	processChunk := func(texts []*textEmbedding) func() {
		return func() {
			defer wg.Done()
			// If an error has already occurred, don't continue processing
			if firstErr != nil {
				return
			}
			// Embed text
			embedding, err := model.BatchEmbed(ctx, utils.MapSlice(texts, func(text *textEmbedding) string {
				return text.text
			}))
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			// Check if embedding result is valid
			if len(embedding) == 0 {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("received empty embedding result")
				}
				mu.Unlock()
				return
			}

			// Check if embedding length matches input length
			if len(embedding) != len(texts) {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("embedding count mismatch: expected %d, got %d", len(texts), len(embedding))
				}
				mu.Unlock()
				return
			}

			mu.Lock()
			for i, text := range texts {
				if text == nil {
					continue
				}
				// Sanitize vector to remove NaN/Inf values
				sanitized, sanitizeErr := sanitizeVector(embedding[i])
				if sanitizeErr != nil {
					// Log warning but continue with sanitized vector
					text.results = sanitized
				} else {
					text.results = embedding[i]
				}
			}
			mu.Unlock()
		}
	}

	// Submit all tasks to the goroutine pool
	for _, texts := range utils.ChunkSlice(textEmbeddings, batchSize) {
		wg.Add(1)
		err := e.pool.Submit(processChunk(texts))
		if err != nil {
			return nil, err
		}
	}

	// Wait for all tasks to complete
	wg.Wait()

	// Check if any errors occurred
	if firstErr != nil {
		return nil, firstErr
	}

	// Sanitize all results and return
	results := make([][]float32, 0, len(textEmbeddings))
	for _, text := range textEmbeddings {
		if text.results != nil {
			sanitized, _ := sanitizeVector(text.results)
			results = append(results, sanitized)
		} else {
			results = append(results, nil)
		}
	}
	return results, nil
}
