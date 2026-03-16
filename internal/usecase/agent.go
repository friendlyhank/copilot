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

		// 调用 LLM
		resp, err := a.callLLM(ctx)
		if err != nil {
			a.emit(OutputError, fmt.Sprintf("API call failed: %v", err))
			return err
		}

		if len(resp.Choices) == 0 {
			a.emit(OutputError, "no response from LLM")
			return errors.New(errors.CodeAgentError, "no choices in response")
		}

		choice := resp.Choices[0]

		// 输出 assistant 的文本内容
		if choice.Message.Content != "" {
			a.emit(OutputText, choice.Message.Content)
		}

		// 如果没有工具调用，循环结束
		if len(choice.Message.ToolCalls) == 0 {
			// 添加 assistant 消息到会话
			assistantMsg := entity.NewMessage(entity.RoleAssistant, choice.Message.Content)
			a.session.AddMessage(assistantMsg)
			return nil
		}

		// 添加 assistant 消息到会话
		assistantMsg := entity.NewMessage(entity.RoleAssistant, choice.Message.Content).
			WithToolCalls(choice.Message.ToolCalls)
		a.session.AddMessage(assistantMsg)

		// 处理工具调用
		for _, toolCall := range choice.Message.ToolCalls {
			result, err := a.executeTool(ctx, toolCall)
			if err != nil {
				a.logger.Error("tool execution failed",
					logger.F("tool", toolCall.Name),
					logger.F("error", err),
				)
			}

			// 添加工具结果消息到会话
			toolMsg := entity.NewMessage(entity.RoleTool, result.Content)
			a.session.AddMessage(toolMsg)
		}
	}
}

// callLLM 调用 LLM
func (a *Agent) callLLM(ctx context.Context) (*port.ChatResponse, error) {
	// 构建消息
	messages := a.buildMessages()

	// 构建请求
	req := &port.ChatRequest{
		Model:       a.llmClient.GetModel(),
		Messages:    messages,
		Stream:      false,
		MaxTokens:   a.config.MaxTokens,
		Temperature: a.config.Temperature,
		Tools:       a.toolReg.ToLLMTools(),
	}

	return a.llmClient.Chat(ctx, req)
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
	a.emit(OutputCommand, call.Arguments)

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
