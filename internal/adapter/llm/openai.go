package llm

import (
	"ai_code/internal/port"
)

const (
	// OpenAIDefaultBaseURL OpenAI 默认 API 地址
	OpenAIDefaultBaseURL = "https://api.openai.com"
)

func init() {
	// 注册 OpenAI 提供商
	Register("openai", NewOpenAIClient)
}

// OpenAIClient OpenAI 兼容客户端实现
type OpenAIClient struct {
	*BaseClient
}

// NewOpenAIClient 创建 OpenAI 客户端
func NewOpenAIClient(config port.ProviderConfig) (port.LLMClient, error) {
	if config.APIKey == "" {
		return nil, port.ErrInvalidConfig
	}

	// 设置默认值
	if config.BaseURL == "" {
		config.BaseURL = OpenAIDefaultBaseURL
	}
	if config.Model == "" {
		config.Model = "gpt-4"
	}

	return &OpenAIClient{
		BaseClient: NewBaseClient("openai", config),
	}, nil
}
