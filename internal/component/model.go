package component

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
)

func NewChatModel(ctx context.Context, apiKey, baseURL, modelName string) (*openai.ChatModel, error) {
	modelConfig := &openai.ChatModelConfig{
		Model:   modelName,
		APIKey:  apiKey,
		BaseURL: baseURL,
	}

	chatModel, err := openai.NewChatModel(ctx, modelConfig)
	if err != nil {
		return nil, err
	}

	return chatModel, nil
}