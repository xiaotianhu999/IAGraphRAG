package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	chatpipline "github.com/aiplusall/aiplusall-kb/internal/application/service/chat_pipline"
	"github.com/aiplusall/aiplusall-kb/internal/config"
	"github.com/aiplusall/aiplusall-kb/internal/logger"
	"github.com/aiplusall/aiplusall-kb/internal/types"
	"github.com/aiplusall/aiplusall-kb/internal/types/interfaces"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// NewChunkExtractTask creates a new chunk extract task
func NewChunkExtractTask(
	ctx context.Context,
	client *asynq.Client,
	tenantID uint64,
	chunkID string,
	modelID string,
) error {
	if strings.ToLower(os.Getenv("NEO4J_ENABLE")) != "true" {
		logger.Warn(ctx, "NEO4J is not enabled, skip chunk extract task")
		return nil
	}
	payload, err := json.Marshal(types.ExtractChunkPayload{
		TenantID: tenantID,
		ChunkID:  chunkID,
		ModelID:  modelID,
	})
	if err != nil {
		return err
	}
	task := asynq.NewTask(types.TypeChunkExtract, payload, asynq.MaxRetry(3))
	info, err := client.Enqueue(task)
	if err != nil {
		logger.Errorf(ctx, "failed to enqueue task: %v", err)
		return fmt.Errorf("failed to enqueue task: %v", err)
	}
	logger.Infof(ctx, "enqueued task: id=%s queue=%s chunk=%s", info.ID, info.Queue, chunkID)
	return nil
}

// ChunkExtractService is a service for extracting chunks
type ChunkExtractService struct {
	template          *types.PromptTemplateStructured
	modelService      interfaces.ModelService
	knowledgeBaseRepo interfaces.KnowledgeBaseRepository
	chunkRepo         interfaces.ChunkRepository
	graphEngine       interfaces.RetrieveGraphRepository
}

// NewChunkExtractService creates a new chunk extract service
func NewChunkExtractService(
	config *config.Config,
	modelService interfaces.ModelService,
	knowledgeBaseRepo interfaces.KnowledgeBaseRepository,
	chunkRepo interfaces.ChunkRepository,
	graphEngine interfaces.RetrieveGraphRepository,
) interfaces.Extracter {
	// generator := chatpipline.NewQAPromptGenerator(chatpipline.NewFormater(), config.ExtractManager.ExtractGraph)
	// ctx := context.Background()
	// logger.Debugf(ctx, "chunk extract system prompt: %s", generator.System(ctx))
	// logger.Debugf(ctx, "chunk extract user prompt: %s", generator.User(ctx, "demo"))
	return &ChunkExtractService{
		template:          config.ExtractManager.ExtractGraph,
		modelService:      modelService,
		knowledgeBaseRepo: knowledgeBaseRepo,
		chunkRepo:         chunkRepo,
		graphEngine:       graphEngine,
	}
}

// Extract extracts a chunk
func (s *ChunkExtractService) Extract(ctx context.Context, t *asynq.Task) error {
	var p types.ExtractChunkPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		logger.Errorf(ctx, "failed to unmarshal task payload: %v", err)
		return err
	}
	ctx = logger.WithRequestID(ctx, uuid.New().String())
	ctx = logger.WithField(ctx, "extract", p.ChunkID)
	ctx = context.WithValue(ctx, types.TenantIDContextKey, p.TenantID)

	chunk, err := s.chunkRepo.GetChunkByID(ctx, p.TenantID, p.ChunkID)
	if err != nil {
		logger.Errorf(ctx, "failed to get chunk: %v", err)
		return err
	}
	kb, err := s.knowledgeBaseRepo.GetKnowledgeBaseByID(ctx, chunk.KnowledgeBaseID)
	if err != nil {
		logger.Errorf(ctx, "failed to get knowledge base: %v", err)
		return err
	}
	if kb.ExtractConfig == nil {
		logger.Warnf(ctx, "knowledge base has no extract config")
		return err
	}
	if !kb.ExtractConfig.Enabled {
		logger.Warnf(ctx, "knowledge base extract config is disabled")
		return nil
	}

	chatModel, err := s.modelService.GetChatModel(ctx, p.ModelID)
	if err != nil {
		logger.Errorf(ctx, "failed to get chat model: %v", err)
		return err
	}

	// Merge knowledge base config with default config from config.yaml
	// Priority: KB config > default config
	template := &types.PromptTemplateStructured{
		Description: s.template.Description, // Always use description from config.yaml
		Tags:        kb.ExtractConfig.Tags,
		Examples:    make([]types.GraphData, 0),
	}

	// Use default tags if KB config has no tags
	if len(template.Tags) == 0 {
		template.Tags = s.template.Tags
		logger.Debugf(ctx, "Using default tags from config.yaml: %d tags", len(template.Tags))
	}

	// Build example from KB config or use default examples
	if kb.ExtractConfig.Text != "" || len(kb.ExtractConfig.Nodes) > 0 || len(kb.ExtractConfig.Relations) > 0 {
		// KB has custom example configuration
		template.Examples = []types.GraphData{
			{
				Text:     kb.ExtractConfig.Text,
				Node:     kb.ExtractConfig.Nodes,
				Relation: kb.ExtractConfig.Relations,
			},
		}
		logger.Debugf(ctx, "Using custom example from knowledge base config")
	} else if len(s.template.Examples) > 0 {
		// Use default examples from config.yaml
		template.Examples = s.template.Examples
		logger.Debugf(ctx, "Using default examples from config.yaml: %d examples", len(template.Examples))
	}

	extractor := chatpipline.NewExtractor(chatModel, template)
	graph, err := extractor.Extract(ctx, chunk.Content)
	if err != nil {
		return err
	}

	chunk, err = s.chunkRepo.GetChunkByID(ctx, p.TenantID, p.ChunkID)
	if err != nil {
		logger.Warnf(ctx, "graph ignore chunk %s: %v", p.ChunkID, err)
		return nil
	}

	for _, node := range graph.Node {
		node.Chunks = []string{chunk.ID}
	}
	if err = s.graphEngine.AddGraph(ctx,
		types.NameSpace{KnowledgeBase: chunk.KnowledgeBaseID, Knowledge: chunk.KnowledgeID},
		[]*types.GraphData{graph},
	); err != nil {
		logger.Errorf(ctx, "failed to add graph: %v", err)
		return err
	}
	return nil
}
