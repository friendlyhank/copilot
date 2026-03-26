package llm

import (
	"ai_code/internal/port"
)

const (
	// IflowDefaultBaseURL iFlow 默认 API 地址
	IflowDefaultBaseURL = "https://apis.iflow.cn/v1/chat/completions"
)

func init() {
	// 注册 iFlow 提供商
	Register("iflow", NewIflowClient)
}

// IflowClient iFlow 客户端实现
type IflowClient struct {
	*BaseClient
}

// NewIflowClient 创建 iFlow 客户端
func NewIflowClient(config port.ProviderConfig) (port.LLMClient, error) {
	if config.APIKey == "" {
		return nil, port.ErrInvalidConfig
	}

	// 设置默认值
	if config.BaseURL == "" {
		config.BaseURL = IflowDefaultBaseURL
	}
	if config.Model == "" {
		config.Model = "qwen3-max"
	}

	return &IflowClient{
		BaseClient: NewBaseClient("iflow", config),
	}, nil
}
