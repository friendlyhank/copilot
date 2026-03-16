package entity

import "time"

// ToolCall 工具调用
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Arguments string `json:"arguments"`
	Result    string `json:"result,omitempty"`
	Status    string `json:"status,omitempty"` // pending, running, success, error
}

// NewToolCall 创建工具调用
func NewToolCall(name, arguments string) ToolCall {
	return ToolCall{
		ID:        generateID(),
		Name:      name,
		Type:      "function",
		Arguments: arguments,
		Status:    "pending",
	}
}

// WithResult 设置结果
func (t ToolCall) WithResult(result string, status string) ToolCall {
	t.Result = result
	t.Status = status
	return t
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