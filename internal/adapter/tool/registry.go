package tool

import (
	"context"
	"sync"

	"ai_code/internal/domain/entity"
	"ai_code/internal/domain/errors"
	"ai_code/internal/port"
)

// Registry 工具注册表
type Registry struct {
	mu    sync.RWMutex
	tools map[string]port.Tool
}

// NewRegistry 创建工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]port.Tool),
	}
}

// Register 注册工具
func (r *Registry) Register(tool port.Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// Get 获取工具
func (r *Registry) Get(name string) (port.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// List 列出所有工具
func (r *Registry) List() []port.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]port.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ToLLMTools 转换为 LLM 工具格式
func (r *Registry) ToLLMTools() []port.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	definitions := make([]port.ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		definitions = append(definitions, port.ToolDefinition{
			Type: "function",
			Function: port.ToolFunction{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Parameters(),
			},
		})
	}
	return definitions
}

// ExecuteTool 执行工具调用
func (r *Registry) ExecuteTool(ctx context.Context, call entity.ToolCall) (entity.ToolResult, error) {
	tool, ok := r.Get(call.Name)
	if !ok {
		return entity.ToolResult{}, errors.New(errors.CodeToolNotFound, "tool not found: "+call.Name)
	}

	output, err := tool.Execute(ctx, call.Arguments)
	if err != nil {
		return entity.ToolResult{
			ToolCallID: call.ID,
			Content:    err.Error(),
			IsError:    true,
		}, err
	}

	return entity.ToolResult{
		ToolCallID: call.ID,
		Content:    output,
		IsError:    false,
	}, nil
}

// DefaultRegistry 默认注册表
var defaultRegistry = NewRegistry()

// Register 注册工具到默认注册表
func Register(tool port.Tool) {
	defaultRegistry.Register(tool)
}

// Get 从默认注册表获取工具
func Get(name string) (port.Tool, bool) {
	return defaultRegistry.Get(name)
}

// List 列出默认注册表的所有工具
func List() []port.Tool {
	return defaultRegistry.List()
}
