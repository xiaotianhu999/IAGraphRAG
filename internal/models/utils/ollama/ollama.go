package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aiplusall/aiplusall-kb/internal/logger"
	"github.com/ollama/ollama/api"
)

// OllamaService manages Ollama service
type OllamaService struct {
	client      *api.Client
	baseURL     string
	mu          sync.Mutex
	isAvailable bool
	isOptional  bool // Added: marks if Ollama service is optional
}

// GetOllamaService gets Ollama service instance (singleton pattern)
func GetOllamaService() (*OllamaService, error) {
	// Get Ollama base URL from environment variable, if not set use provided baseURL or default value
	logger.GetLogger(context.Background()).Infof("Ollama base URL: %s", os.Getenv("OLLAMA_BASE_URL"))
	baseURL := "http://localhost:11434"
	envURL := os.Getenv("OLLAMA_BASE_URL")
	if envURL != "" {
		baseURL = envURL
	}

	// Create URL object
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama service URL: %w", err)
	}

	// Create official client
	client := api.NewClient(parsedURL, http.DefaultClient)

	// Check if Ollama is set as optional
	isOptional := false
	if os.Getenv("OLLAMA_OPTIONAL") == "true" {
		isOptional = true
		logger.GetLogger(context.Background()).Info("Ollama service set to optional mode")
	}

	service := &OllamaService{
		client:     client,
		baseURL:    baseURL,
		isOptional: isOptional,
	}

	return service, nil
}

// StartService checks if Ollama service is available
func (s *OllamaService) StartService(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if service is available
	err := s.client.Heartbeat(ctx)
	if err != nil {
		logger.GetLogger(ctx).Warnf("ollama service unavailable: %v", err)
		s.isAvailable = false

		// If configured as optional, don't return an error
		if s.isOptional {
			logger.GetLogger(ctx).Info("ollama service set as optional, will continue running the application")
			return nil
		}

		return fmt.Errorf("ollama service unavailable: %w", err)
	}

	s.isAvailable = true
	return nil
}

// IsAvailable returns whether the service is available
func (s *OllamaService) IsAvailable() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isAvailable
}

// IsModelAvailable checks if a model is available
func (s *OllamaService) IsModelAvailable(ctx context.Context, modelName string) (bool, error) {
	// First check if the service is available
	if err := s.StartService(ctx); err != nil {
		return false, err
	}

	// If service is not available but set as optional, return false but no error
	if !s.isAvailable && s.isOptional {
		return false, nil
	}

	// Get model list
	listResp, err := s.client.List(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get model list: %w", err)
	}

	// If no version is specified for the model, add ":latest" by default
	checkModelName := modelName
	if !strings.Contains(modelName, ":") {
		checkModelName = modelName + ":latest"
	}
	// Check if model is in the list
	for _, model := range listResp.Models {
		if model.Name == checkModelName {
			return true, nil
		}
	}

	return false, nil
}

// PullModel pulls a model
func (s *OllamaService) PullModel(ctx context.Context, modelName string) error {
	// First check if the service is available
	if err := s.StartService(ctx); err != nil {
		return err
	}

	// If service is not available but set as optional, return nil without further operations
	if !s.isAvailable && s.isOptional {
		logger.GetLogger(ctx).Warnf("Ollama service unavailable, unable to pull model %s", modelName)
		return nil
	}

	// Check if model already exists
	available, err := s.IsModelAvailable(ctx, modelName)
	if err != nil {
		return err
	}
	if available {
		logger.GetLogger(ctx).Infof("Model %s already exists", modelName)
		return nil
	}

	// Use official client to pull model
	pullReq := &api.PullRequest{
		Name: modelName,
	}

	err = s.client.Pull(ctx, pullReq, func(progress api.ProgressResponse) error {
		if progress.Status != "" {
			if progress.Total > 0 && progress.Completed > 0 {
				percentage := float64(progress.Completed) / float64(progress.Total) * 100
				logger.GetLogger(ctx).Infof("Pull progress: %s (%.2f%%)",
					progress.Status, percentage)
			} else {
				logger.GetLogger(ctx).Infof("Pull status: %s", progress.Status)
			}
		}

		if progress.Total > 0 && progress.Completed == progress.Total {
			logger.GetLogger(ctx).Infof("Model %s pull completed", modelName)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}

	return nil
}

// EnsureModelAvailable ensures the model is available, pulls it if not available
func (s *OllamaService) EnsureModelAvailable(ctx context.Context, modelName string) error {
	// If service is not available but set as optional, return nil directly
	if !s.IsAvailable() && s.isOptional {
		logger.GetLogger(ctx).Warnf("Ollama service unavailable, skipping ensuring model %s availability", modelName)
		return nil
	}

	available, err := s.IsModelAvailable(ctx, modelName)
	if err != nil {
		if s.isOptional {
			logger.GetLogger(ctx).
				Warnf("Failed to check model %s availability, but Ollama is set as optional", modelName)
			return nil
		}
		return err
	}

	if !available {
		return s.PullModel(ctx, modelName)
	}

	return nil
}

// GetVersion gets Ollama version
func (s *OllamaService) GetVersion(ctx context.Context) (string, error) {
	// If service is not available but set as optional, return empty version info
	if !s.IsAvailable() && s.isOptional {
		return "unavailable", nil
	}

	version, err := s.client.Version(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get Ollama version: %w", err)
	}
	return version, nil
}

// CreateModel creates a custom model
func (s *OllamaService) CreateModel(ctx context.Context, name, modelfile string) error {
	req := &api.CreateRequest{
		Model:    name,
		Template: modelfile, // Use Template field instead of Modelfile
	}

	err := s.client.Create(ctx, req, func(progress api.ProgressResponse) error {
		if progress.Status != "" {
			logger.GetLogger(ctx).Infof("Model creation status: %s", progress.Status)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create model: %w", err)
	}

	return nil
}

// GetModelInfo gets model information
func (s *OllamaService) GetModelInfo(ctx context.Context, modelName string) (*api.ShowResponse, error) {
	req := &api.ShowRequest{
		Name: modelName,
	}

	resp, err := s.client.Show(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get model information: %w", err)
	}

	return resp, nil
}

// OllamaModelInfo represents detailed information about an Ollama model
type OllamaModelInfo struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	ModifiedAt time.Time `json:"modified_at"`
}

// ListModels lists all available models with basic info (names only)
func (s *OllamaService) ListModels(ctx context.Context) ([]string, error) {
	listResp, err := s.client.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get model list: %w", err)
	}

	modelNames := make([]string, len(listResp.Models))
	for i, model := range listResp.Models {
		modelNames[i] = model.Name
	}

	return modelNames, nil
}

// ListModelsDetailed lists all available models with detailed information
func (s *OllamaService) ListModelsDetailed(ctx context.Context) ([]OllamaModelInfo, error) {
	listResp, err := s.client.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get model list: %w", err)
	}
	jsonData, err := json.Marshal(listResp.Models)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal model list: %w", err)
	}
	logger.GetLogger(ctx).Infof("List models detailed: %s", string(jsonData))

	models := make([]OllamaModelInfo, len(listResp.Models))
	for i, model := range listResp.Models {
		models[i] = OllamaModelInfo{
			Name:       model.Name,
			Size:       model.Size,
			Digest:     model.Digest,
			ModifiedAt: model.ModifiedAt,
		}
	}

	return models, nil
}

// DeleteModel deletes a model
func (s *OllamaService) DeleteModel(ctx context.Context, modelName string) error {
	req := &api.DeleteRequest{
		Name: modelName,
	}

	err := s.client.Delete(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	return nil
}

// IsValidModelName checks if model name is valid
func IsValidModelName(name string) bool {
	// Simple check for model name format
	return name != "" && !strings.Contains(name, " ")
}

// Chat uses Ollama chat
func (s *OllamaService) Chat(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error {
	// First check if service is available
	if err := s.StartService(ctx); err != nil {
		return err
	}

	// Use official client Chat method
	return s.client.Chat(ctx, req, fn)
}

// Embeddings gets text embedding vectors
func (s *OllamaService) Embeddings(ctx context.Context, req *api.EmbedRequest) (*api.EmbedResponse, error) {
	// First check if service is available
	if err := s.StartService(ctx); err != nil {
		return nil, err
	}

	// Use custom embedding call to handle NaN values
	resp, err := s.embedWithNaNHandling(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// embedWithNaNHandling calls the Ollama API with custom NaN handling
func (s *OllamaService) embedWithNaNHandling(ctx context.Context, req *api.EmbedRequest) (*api.EmbedResponse, error) {
	// Build request URL
	embedURL := fmt.Sprintf("%s/api/embed", s.baseURL)

	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embed request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", embedURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send embed request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if response contains an error from Ollama server
	var errorResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error != "" {
		logger.GetLogger(ctx).Errorf("Ollama API returned error: %s", errorResp.Error)

		// If error mentions NaN or JSON encoding, it means Ollama itself failed to serialize the response
		// This typically happens when the model generates NaN values
		if strings.Contains(errorResp.Error, "json: unsupported value") ||
			strings.Contains(errorResp.Error, "NaN") ||
			strings.Contains(errorResp.Error, "Inf") {
			logger.GetLogger(ctx).Warn("Ollama model generated invalid values (NaN/Inf). Consider restarting Ollama service or using a different model.")
			return nil, fmt.Errorf("ollama model generated invalid values: %s", errorResp.Error)
		}

		return nil, fmt.Errorf("ollama API error: %s", errorResp.Error)
	}

	// Replace NaN and Inf values in JSON response
	cleanedBody := cleanJSONNumbers(respBody)

	// Unmarshal cleaned response
	var embedResp api.EmbedResponse
	if err := json.Unmarshal(cleanedBody, &embedResp); err != nil {
		logger.GetLogger(ctx).Errorf("Failed to unmarshal response (even after cleaning): %v, body: %s", err, string(cleanedBody[:min(500, len(cleanedBody))]))
		return nil, fmt.Errorf("failed to unmarshal embed response: %w", err)
	}

	// Validate response
	if len(embedResp.Embeddings) == 0 {
		// Get input count safely
		inputCount := 0
		if inputs, ok := req.Input.([]string); ok {
			inputCount = len(inputs)
		}
		logger.GetLogger(ctx).Errorf("Received empty embeddings from Ollama API, request had %d inputs, response body: %s",
			inputCount, string(cleanedBody[:min(200, len(cleanedBody))]))
		return nil, fmt.Errorf("received empty embeddings from Ollama API")
	}

	// Check if embeddings count matches input count (if we can determine it)
	if inputs, ok := req.Input.([]string); ok {
		if len(embedResp.Embeddings) != len(inputs) {
			logger.GetLogger(ctx).Warnf("Embeddings count mismatch: expected %d, got %d", len(inputs), len(embedResp.Embeddings))
		}
	}

	// Additional sanitization: clean embeddings in-place
	for i := range embedResp.Embeddings {
		embedResp.Embeddings[i] = sanitizeEmbedding(embedResp.Embeddings[i])
	}

	return &embedResp, nil
}

// cleanJSONNumbers replaces NaN and Infinity values in JSON with 0
func cleanJSONNumbers(data []byte) []byte {
	// Replace NaN with 0.0
	nanRegex := regexp.MustCompile(`:\s*NaN\b`)
	data = nanRegex.ReplaceAll(data, []byte(": 0.0"))

	// Replace Infinity with a large number
	infRegex := regexp.MustCompile(`:\s*Infinity\b`)
	data = infRegex.ReplaceAll(data, []byte(": 1e308"))

	// Replace -Infinity with a large negative number
	negInfRegex := regexp.MustCompile(`:\s*-Infinity\b`)
	data = negInfRegex.ReplaceAll(data, []byte(": -1e308"))

	return data
}

// sanitizeEmbedding removes NaN and Inf values from an embedding vector
func sanitizeEmbedding(embedding []float32) []float32 {
	sanitized := make([]float32, len(embedding))
	hasInvalid := false

	for i, v := range embedding {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			sanitized[i] = 0.0
			hasInvalid = true
		} else {
			sanitized[i] = v
		}
	}

	if hasInvalid {
		logger.GetLogger(context.Background()).Warn("Embedding vector contained NaN/Inf values, replaced with 0.0")
	}

	return sanitized
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Generate generates text (used for Rerank)
func (s *OllamaService) Generate(ctx context.Context, req *api.GenerateRequest, fn api.GenerateResponseFunc) error {
	// First check if service is available
	if err := s.StartService(ctx); err != nil {
		return err
	}

	// Use official client Generate method
	return s.client.Generate(ctx, req, fn)
}

// GetClient returns the underlying ollama client for advanced operations
func (s *OllamaService) GetClient() *api.Client {
	return s.client
}
