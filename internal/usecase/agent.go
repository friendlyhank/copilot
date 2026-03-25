package usecase

import (
	"context"
	"fmt"

	"ai_code/internal/domain/entity"
	"ai_code/internal/domain/errors"
	"ai_code/internal/port"
	"ai_code/pkg/logger"
)

// OutputType 输出类型
type OutputType int

const (
	OutputText OutputType = iota
	OutputCommand
	OutputResult
	OutputError
	OutputDone
)

// Output 输出消息
type Output struct {
	Type    OutputType
	Content string
}

// OutputHandler 输出处理函数
type OutputHandler func(Output)

// AgentConfig Agent 配置
type AgentConfig struct {
	MaxTokens   int
	Temperature float64
	Thinking    bool
	UseStream   bool // 是否使用流式模式
}

// Agent Agent 用例
type Agent struct {
	llmClient port.LLMClient
	toolReg   port.ToolRegistry
	session   *entity.Session
	config    AgentConfig
	logger    logger.Logger
	handler   OutputHandler
	system    string
}

// NewAgent 创建 Agent
func NewAgent(llmClient port.LLMClient, toolReg port.ToolRegistry, session *entity.Session, config AgentConfig) *Agent {
	// 默认开启流式
	if config.MaxTokens == 0 {
		config.MaxTokens = 8000
	}
	
	return &Agent{
		llmClient: llmClient,
		toolReg:   toolReg,
		session:   session,
		config:    config,
		logger:    logger.Default().WithPrefix("agent"),
	}
}

// SetSystem 设置系统提示
func (a *Agent) SetSystem(system string) {
	a.system = system
}

// SetOutputHandler 设置输出处理器
func (a *Agent) SetOutputHandler(handler OutputHandler) {
	a.handler = handler
}

// emit 发送输出
func (a *Agent) emit(outputType OutputType, content string) {
	if a.handler != nil {
		a.handler(Output{Type: outputType, Content: content})
	}
}

// ProcessMessage 处理用户消息
func (a *Agent) ProcessMessage(ctx context.Context, input string) error {
	// 添加用户消息到会话
	userMsg := entity.NewMessage(entity.RoleUser, input)
	a.session.AddMessage(userMsg)

	return a.Loop(ctx)
}

// Loop 执行 Agent 循环
func (a *Agent) Loop(ctx context.Context) error {
	defer a.emit(OutputDone, "")

	for {
		select {
		case <-ctx.Done():
			return errors.New(errors.CodeContextCanceled, "context canceled")
		default:
		}

		// 使用流式调用
		content, toolCalls, err := a.callLLMStream(ctx)
		if err != nil {
			a.emit(OutputError, fmt.Sprintf("API call failed: %v", err))
			return err
		}

		// 如果没有工具调用，循环结束
		if len(toolCalls) == 0 {
			// 添加 assistant 消息到会话
			assistantMsg := entity.NewMessage(entity.RoleAssistant, content)
			a.session.AddMessage(assistantMsg)
			return nil
		}

		// 添加 assistant 消息到会话
		assistantMsg := entity.NewMessage(entity.RoleAssistant, content).
			WithToolCalls(toolCalls)
		a.session.AddMessage(assistantMsg)

		// 处理工具调用
		for _, toolCall := range toolCalls {
			result, err := a.executeTool(ctx, toolCall)
			if err != nil {
				a.logger.Error("tool execution failed",
					logger.F("tool", toolCall.GetName()),
					logger.F("error", err),
				)
			}

			// 添加工具结果消息到会话
			toolMsg := entity.NewMessage(entity.RoleTool, result.Content).
				WithToolCallID(result.ToolCallID)
			a.session.AddMessage(toolMsg)
		}
	}
}

// callLLMStream 流式调用 LLM
func (a *Agent) callLLMStream(ctx context.Context) (string, []entity.ToolCall, error) {
	// 构建消息
	messages := a.buildMessages()

	// 构建请求
	req := &port.ChatRequest{
		Model:       a.llmClient.GetModel(),
		Messages:    messages,
		Stream:      true,
		MaxTokens:   a.config.MaxTokens,
		Temperature: a.config.Temperature,
		Tools:       a.toolReg.ToLLMTools(),
	}

	var contentBuilder string
	var toolCallsMap = make(map[int]*entity.ToolCall)

	err := a.llmClient.ChatStream(ctx, req, func(chunk *port.StreamChunk) error {
		if len(chunk.Choices) == 0 {
			return nil
		}

		choice := chunk.Choices[0]
		delta := choice.Delta

		// 累积文本内容并实时输出
		if delta.Content != "" {
			contentBuilder += delta.Content
			a.emit(OutputText, delta.Content)
		}

		// 累积工具调用
		for _, tc := range delta.ToolCalls {
			idx := tc.Index

			if tc.ID != "" {
				// 新的工具调用
				toolCallsMap[idx] = &entity.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: entity.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			} else if existing, ok := toolCallsMap[idx]; ok {
				// 累积到现有的工具调用
				existing.Function.Arguments += tc.Function.Arguments
			}
		}

		return nil
	})

	if err != nil {
		return "", nil, err
	}

	// 按 index 排序转换为切片
	toolCalls := make([]entity.ToolCall, len(toolCallsMap))
	for idx, tc := range toolCallsMap {
		toolCalls[idx] = *tc
	}

	return contentBuilder, toolCalls, nil
}

// buildMessages 构建消息列表
func (a *Agent) buildMessages() []entity.Message {
	messages := make([]entity.Message, 0)

	// 添加系统消息
	if a.system != "" {
		messages = append(messages, entity.NewMessage(entity.RoleSystem, a.system))
	}

	// 添加会话消息
	messages = append(messages, a.session.Messages...)

	return messages
}

// executeTool 执行工具
func (a *Agent) executeTool(ctx context.Context, call entity.ToolCall) (entity.ToolResult, error) {
	// 输出命令
	a.emit(OutputCommand, call.GetArguments())

	// 执行工具
	result, err := a.toolReg.ExecuteTool(ctx, call)
	if err != nil {
		a.emit(OutputError, err.Error())
		return result, err
	}

	// 输出结果（截断显示）
	displayResult := result.Content
	if len(displayResult) > 1000 {
		displayResult = displayResult[:1000] + "..."
	}
	a.emit(OutputResult, displayResult)

	return result, nil
}

// SwitchModel 切换模型
func (a *Agent) SwitchModel(model string) {
	a.llmClient.SetModel(model)
	a.session.SetModel(model)
}
