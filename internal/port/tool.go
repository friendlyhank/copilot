package port

import (
	"context"

	"ai_code/internal/domain/entity"
)

// Tool 工具端口接口
type Tool interface {
	// Name 工具名称
	Name() string

	// Description 工具描述
	Description() string

	// Parameters 工具参数定义（JSON Schema）
	Parameters() map[string]interface{}

	// Execute 执行工具
	Execute(ctx context.Context, args string) (string, error)
}

// ToolRegistry 工具注册表接口
type ToolRegistry interface {
	// Register 注册工具
	Register(tool Tool)

	// Get 获取工具
	Get(name string) (Tool, bool)

	// List 列出所有工具
	List() []Tool

	// ToLLMTools 转换为 LLM 工具格式
	ToLLMTools() []ToolDefinition

	// ExecuteTool 执行工具调用
	ExecuteTool(ctx context.Context, call entity.ToolCall) (entity.ToolResult, error)
}

// ToolExecutor 工具执行器接口
type ToolExecutor interface {
	// ExecuteTool 执行工具调用
	ExecuteTool(ctx context.Context, call entity.ToolCall) (entity.ToolResult, error)
}
