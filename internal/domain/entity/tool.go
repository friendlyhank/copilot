package entity

import "time"

// ToolCall 工具调用
type ToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function FunctionCall    `json:"function"`
	Result   string          `json:"result,omitempty"`
	Status   string          `json:"status,omitempty"` // pending, running, success, error
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// NewToolCall 创建工具调用
func NewToolCall(name, arguments string) ToolCall {
	return ToolCall{
		ID:   generateID(),
		Type: "function",
		Function: FunctionCall{
			Name:      name,
			Arguments: arguments,
		},
		Status: "pending",
	}
}

// WithResult 设置结果
func (t ToolCall) WithResult(result string, status string) ToolCall {
	t.Result = result
	t.Status = status
	return t
}

// GetName 获取工具名称（便捷方法）
func (t ToolCall) GetName() string {
	return t.Function.Name
}

// GetArguments 获取参数（便捷方法）
func (t ToolCall) GetArguments() string {
	return t.Function.Arguments
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolResult 工具执行结果
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error"`
	Timestamp  time.Time `json:"timestamp"`
}

// NewToolResult 创建工具结果
func NewToolResult(toolCallID, content string, isError bool) ToolResult {
	return ToolResult{
		ToolCallID: toolCallID,
		Content:    content,
		IsError:    isError,
		Timestamp:  time.Now(),
	}
}