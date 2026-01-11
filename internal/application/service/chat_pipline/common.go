package chatpipline

import (
	"context"

	"github.com/aiplusall/aiplusall-kb/internal/common"
	"github.com/aiplusall/aiplusall-kb/internal/logger"
	"github.com/aiplusall/aiplusall-kb/internal/models/chat"
	"github.com/aiplusall/aiplusall-kb/internal/types"
	"github.com/aiplusall/aiplusall-kb/internal/types/interfaces"
)

// pipelineInfo logs pipeline info level entries.
func pipelineInfo(ctx context.Context, stage, action string, fields map[string]interface{}) {
	common.PipelineInfo(ctx, stage, action, fields)
}

// pipelineWarn logs pipeline warning level entries.
func pipelineWarn(ctx context.Context, stage, action string, fields map[string]interface{}) {
	common.PipelineWarn(ctx, stage, action, fields)
}

// pipelineError logs pipeline error level entries.
func pipelineError(ctx context.Context, stage, action string, fields map[string]interface{}) {
	common.PipelineError(ctx, stage, action, fields)
}

// prepareChatModel shared logic to prepare chat model and options
// it gets the chat model and sets up the chat options based on the chat manage.
func prepareChatModel(ctx context.Context, modelService interfaces.ModelService,
	chatManage *types.ChatManage,
) (chat.Chat, *chat.ChatOptions, error) {
	chatModel, err := modelService.GetChatModel(ctx, chatManage.ChatModelID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get chat model: %v", err)
		return nil, nil, err
	}

	opt := &chat.ChatOptions{
		Temperature:         chatManage.SummaryConfig.Temperature,
		TopP:                chatManage.SummaryConfig.TopP,
		Seed:                chatManage.SummaryConfig.Seed,
		MaxTokens:           chatManage.SummaryConfig.MaxTokens,
		MaxCompletionTokens: chatManage.SummaryConfig.MaxCompletionTokens,
		FrequencyPenalty:    chatManage.SummaryConfig.FrequencyPenalty,
		PresencePenalty:     chatManage.SummaryConfig.PresencePenalty,
	}

	return chatModel, opt, nil
}

// prepareMessagesWithHistory prepare complete messages including history
func prepareMessagesWithHistory(chatManage *types.ChatManage) []chat.Message {
	chatMessages := []chat.Message{
		{Role: "system", Content: chatManage.SummaryConfig.Prompt},
	}

	chatHistory := chatManage.History
	if len(chatHistory) > 2 {
		chatHistory = chatHistory[len(chatHistory)-2:]
	}

	// Add conversation history
	for _, history := range chatHistory {
		chatMessages = append(chatMessages, chat.Message{Role: "user", Content: history.Query})
		chatMessages = append(chatMessages, chat.Message{Role: "assistant", Content: history.Answer})
	}

	// Add current user message
	chatMessages = append(chatMessages, chat.Message{Role: "user", Content: chatManage.UserContent})

	return chatMessages
}
