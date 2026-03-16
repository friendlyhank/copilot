package mock

import (
	"context"

	"ai_code/internal/port"
)

// MockTool Mock 工具
type MockTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	output      string
	err         error
}

// NewMockTool 创建 Mock 工具
func NewMockTool(name string) *MockTool {
	return &MockTool{
		name:        name,
		description: "Mock tool for testing",
		parameters: map[string]interface{}{
			"type": "object",
		},
	}
}

// Name 工具名称
func (t *MockTool) Name() string {
	return t.name
}

// Description 工具描述
func (t *MockTool) Description() string {
	return t.description
}

// Parameters 参数定义
func (t *MockTool) Parameters() map[string]interface{} {
	return t.parameters
}

// Execute 执行工具
func (t *MockTool) Execute(ctx context.Context, args string) (string, error) {
	return t.output, t.err
}

// SetOutput 设置输出
func (t *MockTool) SetOutput(output string) {
	t.output = output
}

// SetError 设置错误
func (t *MockTool) SetError(err error) {
	t.err = err
}

// MockToolRegistry Mock 工具注册表
type MockToolRegistry struct {
	tools map[string]port.Tool
}

// NewMockToolRegistry 创建 Mock 工具注册表
func NewMockToolRegistry() *MockToolRegistry {
	return &MockToolRegistry{
		tools: make(map[string]port.Tool),
	}
}

// Register 注册工具
func (r *MockToolRegistry) Register(tool port.Tool) {
	r.tools[tool.Name()] = tool
}

// Get 获取工具
func (r *MockToolRegistry) Get(name string) (port.Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List 列出所有工具
func (r *MockToolRegistry) List() []port.Tool {
	tools := make([]port.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ToLLMTools 转换为 LLM 工具格式
func (r *MockToolRegistry) ToLLMTools() []port.ToolDefinition {
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

// ExecuteTool 执行工具
func (r *MockToolRegistry) ExecuteTool(ctx context.Context, call interface{}) (interface{}, error) {
	// 简化实现
	return "mock result", nil
}
