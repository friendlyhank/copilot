package llm

import (
	"fmt"
	"sync"

	"ai_code/internal/port"
)

// 全局注册表
var (
	registry = make(map[string]port.LLMClientFactory)
	mu       sync.RWMutex
)

// ErrProviderNotFound 提供商未找到错误
var ErrProviderNotFound = fmt.Errorf("provider not found")

// ErrInvalidConfig 无效配置错误
var ErrInvalidConfig = fmt.Errorf("invalid config")

// Register 注册 LLM 提供商
func Register(name string, factory port.LLMClientFactory) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = factory
}

// Get 获取 LLM 客户端
func Get(name string, config port.ProviderConfig) (port.LLMClient, error) {
	mu.RLock()
	defer mu.RUnlock()

	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}
	return factory(config)
}

// List 列出所有已注册的提供商
func List() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
