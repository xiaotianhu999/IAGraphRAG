package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aiplusall/aiplusall-kb/internal/config"
	"github.com/aiplusall/aiplusall-kb/internal/logger"
	"github.com/aiplusall/aiplusall-kb/internal/types"
	"github.com/aiplusall/aiplusall-kb/internal/types/interfaces"
	"github.com/hibiken/asynq"
)

// GraphRebuildService handles graph rebuild tasks
type GraphRebuildService struct {
	config        *config.Config
	kbRepo        interfaces.KnowledgeBaseRepository
	knowledgeRepo interfaces.KnowledgeRepository
	chunkRepo     interfaces.ChunkRepository
	graphEngine   interfaces.RetrieveGraphRepository
	modelService  interfaces.ModelService
	task          *asynq.Client
}

// NewGraphRebuildService creates a new graph rebuild service
func NewGraphRebuildService(
	config *config.Config,
	kbRepo interfaces.KnowledgeBaseRepository,
	knowledgeRepo interfaces.KnowledgeRepository,
	chunkRepo interfaces.ChunkRepository,
	graphEngine interfaces.RetrieveGraphRepository,
	modelService interfaces.ModelService,
	task *asynq.Client,
) interfaces.GraphRebuildService {
	return &GraphRebuildService{
		config:        config,
		kbRepo:        kbRepo,
		knowledgeRepo: knowledgeRepo,
		chunkRepo:     chunkRepo,
		graphEngine:   graphEngine,
		modelService:  modelService,
		task:          task,
	}
}

// RebuildGraphAsync 异步触发知识图谱批量重建
func (s *GraphRebuildService) RebuildGraphAsync(ctx context.Context, kbID string, modelID string, batchSize int) error {
	logger.Infof(ctx, "Starting async graph rebuild for knowledge base: %s", kbID)

	// 验证知识库是否存在
	kb, err := s.kbRepo.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge base %s: %v", kbID, err)
		return fmt.Errorf("knowledge base not found: %w", err)
	}

	// 检查知识库是否启用了图谱提取
	if kb.ExtractConfig == nil || !kb.ExtractConfig.Enabled {
		return fmt.Errorf("knowledge base %s does not have graph extraction enabled", kbID)
	}

	// 如果未指定模型，使用知识库的默认模型
	if modelID == "" {
		if kb.EmbeddingModelID == "" {
			return fmt.Errorf("no model specified and knowledge base has no default model")
		}
		modelID = kb.EmbeddingModelID
	}

	// 获取租户信息
	tenantID := kb.TenantID

	// 创建异步任务
	payload := types.GraphRebuildPayload{
		TenantID:        tenantID,
		KnowledgeBaseID: kbID,
		ModelID:         modelID,
		BatchSize:       batchSize,
	}

	payloadBytes, err := payload.Marshal()
	if err != nil {
		logger.Errorf(ctx, "Failed to marshal graph rebuild payload: %v", err)
		return fmt.Errorf("failed to create task: %w", err)
	}

	task := asynq.NewTask(types.TypeGraphRebuild, payloadBytes, asynq.Queue("default"))
	info, err := s.task.Enqueue(task)
	if err != nil {
		logger.Errorf(ctx, "Failed to enqueue graph rebuild task: %v", err)
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	logger.Infof(ctx, "Graph rebuild task enqueued: task_id=%s, kb_id=%s", info.ID, kbID)
	return nil
}

// Handle 处理图谱批量重建任务
func (s *GraphRebuildService) Handle(ctx context.Context, t *asynq.Task) error {
	var payload types.GraphRebuildPayload
	if err := payload.Unmarshal(t.Payload()); err != nil {
		logger.Errorf(ctx, "Failed to unmarshal graph rebuild payload: %v", err)
		return fmt.Errorf("invalid payload: %w", err)
	}

	logger.Infof(ctx, "Processing graph rebuild task: kb_id=%s, model_id=%s, batch_size=%d",
		payload.KnowledgeBaseID, payload.ModelID, payload.BatchSize)

	startTime := time.Now()

	// 获取知识库信息
	kb, err := s.kbRepo.GetKnowledgeBaseByID(ctx, payload.KnowledgeBaseID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge base: %v", err)
		return err
	}

	// 验证图谱配置
	if kb.ExtractConfig == nil || !kb.ExtractConfig.Enabled {
		logger.Warnf(ctx, "Graph extraction not enabled for kb %s, skipping", payload.KnowledgeBaseID)
		return nil
	}

	// 获取模型
	chatModel, err := s.modelService.GetChatModel(ctx, payload.ModelID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get chat model: %v", err)
		return err
	}

	// 获取知识库下所有knowledge
	knowledgeList, err := s.knowledgeRepo.ListKnowledgeByKnowledgeBaseID(ctx, payload.TenantID, payload.KnowledgeBaseID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge list for kb %s: %v", payload.KnowledgeBaseID, err)
		return err
	}

	if len(knowledgeList) == 0 {
		logger.Warnf(ctx, "No knowledge found for kb %s, skipping graph rebuild", payload.KnowledgeBaseID)
		return nil
	}

	// 收集所有chunks
	var allChunks []*types.Chunk
	for _, knowledge := range knowledgeList {
		chunks, err := s.chunkRepo.ListChunksByKnowledgeID(ctx, payload.TenantID, knowledge.ID)
		if err != nil {
			logger.Warnf(ctx, "Failed to get chunks for knowledge %s: %v", knowledge.ID, err)
			continue
		}
		allChunks = append(allChunks, chunks...)
	}

	if len(allChunks) == 0 {
		logger.Warnf(ctx, "No chunks found for kb %s, skipping graph rebuild", payload.KnowledgeBaseID)
		return nil
	}

	logger.Infof(ctx, "Found %d chunks from %d knowledge items to process for graph rebuild",
		len(allChunks), len(knowledgeList))

	// 删除旧图谱数据
	namespace := types.NameSpace{KnowledgeBase: payload.KnowledgeBaseID}
	if err := s.graphEngine.DelGraph(ctx, []types.NameSpace{namespace}); err != nil {
		logger.Warnf(ctx, "Failed to delete old graph data (may not exist): %v", err)
	} else {
		logger.Infof(ctx, "Deleted old graph data for kb %s", payload.KnowledgeBaseID)
	}

	// 使用GraphBuilder进行批量构建
	builder := NewGraphBuilder(s.config, chatModel)

	// 根据批次大小进行分批处理
	batchSize := payload.BatchSize
	if batchSize <= 0 || batchSize > len(allChunks) {
		batchSize = len(allChunks) // 全量处理
	}

	totalBatches := (len(allChunks) + batchSize - 1) / batchSize
	logger.Infof(ctx, "Processing %d chunks in %d batches (batch_size=%d)", len(allChunks), totalBatches, batchSize)

	for i := 0; i < len(allChunks); i += batchSize {
		end := i + batchSize
		if end > len(allChunks) {
			end = len(allChunks)
		}

		batch := allChunks[i:end]
		batchNum := i/batchSize + 1

		logger.Infof(ctx, "Processing batch %d/%d (%d chunks)", batchNum, totalBatches, len(batch))

		// 调用GraphBuilder构建图谱
		if err := builder.BuildGraph(ctx, batch); err != nil {
			logger.Errorf(ctx, "Failed to build graph for batch %d: %v", batchNum, err)
			// 继续处理下一批，不中断整个任务
			continue
		}

		logger.Infof(ctx, "Successfully processed batch %d/%d", batchNum, totalBatches)
	}

	elapsed := time.Since(startTime)
	logger.Infof(ctx, "Graph rebuild completed for kb %s: processed %d chunks in %v",
		payload.KnowledgeBaseID, len(allChunks), elapsed)

	return nil
}
