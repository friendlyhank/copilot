package mock

import (
	"context"

	"ai_code/internal/domain/entity"
	"ai_code/internal/port"
)

// MockLLMClient Mock LLM 客户端
type MockLLMClient struct {
	Responses map[string]*port.ChatResponse
	Calls     []port.ChatRequest
	Model     string
}

// NewMockLLMClient 创建 Mock LLM 客户端
func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		Responses: make(map[string]*port.ChatResponse),
		Calls:     []port.ChatRequest{},
		Model:     "mock-model",
	}
}

// Chat 发送聊天请求
func (m *MockLLMClient) Chat(ctx context.Context, req *port.ChatRequest) (*port.ChatResponse, error) {
	m.Calls = append(m.Calls, *req)
	
	// 查找匹配的响应
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		if resp, ok := m.Responses[lastMsg.Content]; ok {
			return resp, nil
		}
	}
	
	// 返回默认响应
	return &port.ChatResponse{
		ID: "mock-id",
		Choices: []port.Choice{
			{
				Message: port.ResponseMsg{
					Role:    "assistant",
					Content: "Mock response",
				},
			},
		},
	}, nil
}

// GetName 获取名称
func (m *MockLLMClient) GetName() string {
	return "mock"
}

// GetModel 获取模型
func (m *MockLLMClient) GetModel() string {
	return m.Model
}

// SetModel 设置模型
func (m *MockLLMClient) SetModel(model string) {
	m.Model = model
}

// SetDebug 设置调试模式
func (m *MockLLMClient) SetDebug(debug bool) {}

// SetResponse 设置响应
func (m *MockLLMClient) SetResponse(input, output string) {
	m.Responses[input] = &port.ChatResponse{
		ID: "mock-id",
		Choices: []port.Choice{
			{
				Message: port.ResponseMsg{
					Role:    "assistant",
					Content: output,
				},
			},
		},
	}
}

// SetToolCallResponse 设置工具调用响应
func (m *MockLLMClient) SetToolCallResponse(input string, toolCalls []entity.ToolCall) {
	m.Responses[input] = &port.ChatResponse{
		ID: "mock-id",
		Choices: []port.Choice{
			{
				Message: port.ResponseMsg{
					Role:      "assistant",
					ToolCalls: toolCalls,
				},
			},
		},
	}
}
