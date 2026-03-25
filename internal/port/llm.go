package port

import (
	"context"
	"fmt"

	"ai_code/internal/domain/entity"
)

// StreamHandler 流式响应处理器
type StreamHandler func(chunk *StreamChunk) error

// LLMClient LLM 客户端端口接口
type LLMClient interface {
	// Chat 发送聊天请求（非流式）
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStream 发送流式聊天请求
	ChatStream(ctx context.Context, req *ChatRequest, handler StreamHandler) error

	// GetName 获取提供商名称
	GetName() string

	// GetModel 获取当前使用的模型
	GetModel() string

	// SetModel 设置使用的模型
	SetModel(model string)

	// SetDebug 设置调试模式
	SetDebug(debug bool)
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model       string           `json:"model"`
	Messages    []entity.Message `json:"messages"`
	Stream      bool             `json:"stream"`
	MaxTokens   int              `json:"max_tokens"`
	Temperature float64          `json:"temperature"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID        string          `json:"id"`
	Object    string          `json:"object"`
	Created   int64           `json:"created"`
	Model     string          `json:"model"`
	Choices   []Choice        `json:"choices"`
	Usage     Usage           `json:"usage"`
	ToolCalls []entity.ToolCall `json:"tool_calls,omitempty"` // iFlow 返回在根级别
}

// Choice 选择项
type Choice struct {
	Index        int         `json:"index"`
	Message      ResponseMsg `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// ResponseMsg 响应消息
type ResponseMsg struct {
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	ToolCalls []entity.ToolCall `json:"tool_calls,omitempty"`
}

// Usage Token 使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk 流式响应的 chunk
type StreamChunk struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []StreamChoice  `json:"choices"`
	Usage   *Usage          `json:"usage,omitempty"`
	// iFlow 兼容：根级别的 tool_calls
	ToolCalls []entity.ToolCall `json:"tool_calls,omitempty"`
}

// StreamChoice 流式响应的选择项
type StreamChoice struct {
	Index        int          `json:"index"`
	Delta        StreamDelta  `json:"delta"`
	FinishReason string       `json:"finish_reason"`
}

// StreamDelta 流式响应的增量内容
type StreamDelta struct {
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []StreamToolCall `json:"tool_calls,omitempty"`
}

// StreamToolCall 流式响应中的工具调用（包含 index）
type StreamToolCall struct {
	Index    int                    `json:"index"`
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function StreamFunctionCall     `json:"function"`
}

// StreamFunctionCall 流式响应中的函数调用
type StreamFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction 工具函数定义
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// LLMClientFactory 客户端工厂函数类型
type LLMClientFactory func(config ProviderConfig) (LLMClient, error)

// ProviderConfig 提供商配置
type ProviderConfig struct {
	Name    string
	APIKey  string
	BaseURL string
	Model   string
	Timeout int
}

// 错误定义
var (
	ErrInvalidConfig    = fmt.Errorf("invalid config")
	ErrProviderNotFound = fmt.Errorf("provider not found")
)

// LLMRegistry LLM 客户端注册表
type LLMRegistry interface {
	Register(name string, factory LLMClientFactory)
	Get(name string, config ProviderConfig) (LLMClient, error)
	List() []string
}