package embedding

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/utils/ollama"
	ollamaapi "github.com/ollama/ollama/api"
)

// OllamaEmbedder implements text vectorization functionality using Ollama
type OllamaEmbedder struct {
	modelName            string
	truncatePromptTokens int
	ollamaService        *ollama.OllamaService
	dimensions           int
	modelID              string
	EmbedderPooler
}

// OllamaEmbedRequest represents an Ollama embedding request
type OllamaEmbedRequest struct {
	Model                string `json:"model"`
	Prompt               string `json:"prompt"`
	TruncatePromptTokens int    `json:"truncate_prompt_tokens"`
}

// OllamaEmbedResponse represents an Ollama embedding response
type OllamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// NewOllamaEmbedder creates a new Ollama embedder
func NewOllamaEmbedder(baseURL,
	modelName string,
	truncatePromptTokens int,
	dimensions int,
	modelID string,
	pooler EmbedderPooler,
	ollamaService *ollama.OllamaService,
) (*OllamaEmbedder, error) {
	if modelName == "" {
		modelName = "nomic-embed-text"
	}

	if truncatePromptTokens == 0 {
		truncatePromptTokens = 511
	}

	return &OllamaEmbedder{
		modelName:            modelName,
		truncatePromptTokens: truncatePromptTokens,
		ollamaService:        ollamaService,
		EmbedderPooler:       pooler,
		dimensions:           dimensions,
		modelID:              modelID,
	}, nil
}

// ensureModelAvailable ensures that the model is available
func (e *OllamaEmbedder) ensureModelAvailable(ctx context.Context) error {
	logger.GetLogger(ctx).Infof("Ensuring model %s is available", e.modelName)
	return e.ollamaService.EnsureModelAvailable(ctx, e.modelName)
}

// Embed converts text to vector
func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	embedding, err := e.BatchEmbed(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("failed to embed text: %w", err)
	}

	if len(embedding) == 0 {
		return nil, fmt.Errorf("failed to embed text: %w", err)
	}

	return embedding[0], nil
}

// BatchEmbed converts multiple texts to vectors in batch
func (e *OllamaEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	// Ensure model is available
	if err := e.ensureModelAvailable(ctx); err != nil {
		return nil, err
	}

	// Try embedding with progressive text truncation on failure
	return e.batchEmbedWithRetry(ctx, texts, 1.0)
}

// batchEmbedWithRetry attempts embedding with progressive truncation on NaN errors
func (e *OllamaEmbedder) batchEmbedWithRetry(ctx context.Context, texts []string, ratio float64) ([][]float32, error) {
	// Truncate texts if ratio < 1.0
	processedTexts := texts
	if ratio < 1.0 {
		processedTexts = make([]string, len(texts))
		for i, text := range texts {
			processedTexts[i] = TruncateTextWithRatio(text, ratio)
		}
		logger.GetLogger(ctx).Warnf("Retrying embedding with %.0f%% text length", ratio*100)
	}

	// Create request
	req := &ollamaapi.EmbedRequest{
		Model:   e.modelName,
		Input:   processedTexts,
		Options: make(map[string]interface{}),
	}

	// Set truncation parameters
	if e.truncatePromptTokens > 0 {
		req.Options["truncate"] = e.truncatePromptTokens
	}

	// Send request
	startTime := time.Now()
	resp, err := e.ollamaService.Embeddings(ctx, req)

	// Handle NaN-related errors with progressive truncation
	if err != nil {
		errMsg := err.Error()
		isNaNError := strings.Contains(errMsg, "NaN") ||
			strings.Contains(errMsg, "Inf") ||
			strings.Contains(errMsg, "invalid values")

		if isNaNError {
			// Try progressively shorter texts based on current ratio
			if ratio >= 1.0 {
				// First retry: 70%
				logger.GetLogger(ctx).Warnf("NaN error detected, retrying with 70%% text length")
				return e.batchEmbedWithRetry(ctx, texts, 0.7)
			} else if ratio > 0.6 {
				// Second retry: 50%
				logger.GetLogger(ctx).Warnf("NaN error persists, retrying with 50%% text length")
				return e.batchEmbedWithRetry(ctx, texts, 0.5)
			} else if ratio > 0.4 {
				// Third retry: 30%
				logger.GetLogger(ctx).Warnf("NaN error persists, retrying with 30%% text length")
				return e.batchEmbedWithRetry(ctx, texts, 0.3)
			} else if ratio > 0.2 {
				// Fourth retry: first 512 characters only
				logger.GetLogger(ctx).Warnf("NaN error persists, trying first 512 characters only")
				shortenedTexts := make([]string, len(texts))
				for i, text := range texts {
					if len(text) > 512 {
						shortenedTexts[i] = text[:512]
					} else {
						shortenedTexts[i] = text
					}
				}
				return e.batchEmbedWithRetry(ctx, shortenedTexts, 0.1) // Use 0.1 to mark final attempt
			}

			// Last resort: log warning and return zero vectors
			logger.GetLogger(ctx).Errorf("Failed to embed texts after all retries, using zero vectors as fallback")
			fallbackEmbeddings := make([][]float32, len(texts))
			for i := range fallbackEmbeddings {
				fallbackEmbeddings[i] = make([]float32, 1024) // bge-m3 dimension
			}
			return fallbackEmbeddings, nil
		}

		return nil, fmt.Errorf("failed to get embedding vectors: %w", err)
	}

	logger.GetLogger(ctx).Debugf("Embedding vector retrieval took: %v", time.Since(startTime))

	// Sanitize embeddings to remove NaN/Inf values
	sanitizedEmbeddings := make([][]float32, len(resp.Embeddings))
	for i, embedding := range resp.Embeddings {
		sanitized := make([]float32, len(embedding))
		hasInvalid := false
		for j, v := range embedding {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				sanitized[j] = 0.0
				hasInvalid = true
			} else {
				sanitized[j] = v
			}
		}
		if hasInvalid {
			logger.GetLogger(ctx).Warnf("OllamaEmbedder: Vector contains NaN/Inf values, replaced with 0.0")
		}
		sanitizedEmbeddings[i] = sanitized
	}

	return sanitizedEmbeddings, nil
}

// GetModelName returns the model name
func (e *OllamaEmbedder) GetModelName() string {
	return e.modelName
}

// GetDimensions returns the vector dimensions
func (e *OllamaEmbedder) GetDimensions() int {
	return e.dimensions
}

// GetModelID returns the model ID
func (e *OllamaEmbedder) GetModelID() string {
	return e.modelID
}
