package llm

import "context"

// Client LLM客户端接口
// 不同厂商实现此接口即可接入
type Client interface {
	// Chat 发送聊天请求
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// GetName 获取厂商名称
	GetName() string

	// GetModel 获取当前使用的模型
	GetModel() string

	// SetModel 设置使用的模型
	SetModel(model string)
}

// ClientFactory 客户端工厂函数类型
type ClientFactory func(config ProviderConfig) (Client, error)

// 注册的厂商工厂
var providers = make(map[string]ClientFactory)

// RegisterProvider 注册厂商
func RegisterProvider(name string, factory ClientFactory) {
	providers[name] = factory
}

// GetProvider 获取厂商客户端
func GetProvider(name string, config ProviderConfig) (Client, error) {
	factory, ok := providers[name]
	if !ok {
		return nil, ErrProviderNotFound
	}
	return factory(config)
}

// ListProviders 列出所有支持的厂商
func ListProviders() []string {
	result := make([]string, 0, len(providers))
	for name := range providers {
		result = append(result, name)
	}
	return result
}
