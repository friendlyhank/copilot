package llm

import "errors"

var (
	// ErrProviderNotFound 厂商未找到
	ErrProviderNotFound = errors.New("provider not found")
	// ErrInvalidConfig 无效配置
	ErrInvalidConfig = errors.New("invalid config")
	// ErrAPIError API错误
	ErrAPIError = errors.New("API error")
)
