package entity

import "time"

// Role 消息角色
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message 消息实体
type Message struct {
	ID         string     `json:"id"`
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // 工具结果消息需要此字段
	Timestamp  time.Time  `json:"timestamp"`
}

// NewMessage 创建新消息
func NewMessage(role Role, content string) Message {
	return Message{
		ID:        generateID(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// WithToolCalls 添加工具调用
func (m Message) WithToolCalls(calls []ToolCall) Message {
	m.ToolCalls = calls
	return m
}

// WithToolCallID 设置工具调用ID
func (m Message) WithToolCallID(id string) Message {
	m.ToolCallID = id
	return m
}

// ToLLMMessage 转换为 LLM 格式消息
func (m Message) ToLLMMessage() map[string]interface{} {
	msg := map[string]interface{}{
		"role":    string(m.Role),
		"content": m.Content,
	}
	if len(m.ToolCalls) > 0 {
		msg["tool_calls"] = m.ToolCalls
	}
	return msg
}
