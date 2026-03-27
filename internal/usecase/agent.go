package usecase

import (
	"context"
	"fmt"
	"strings"

	"ai_code/internal/domain/entity"
	"ai_code/internal/domain/errors"
	"ai_code/internal/port"
	"ai_code/pkg/logger"
)

// OutputType 输出类型
type OutputType int

const (
	OutputText      OutputType = iota
	OutputTextChunk            // 流式文本片段（不加前缀，追加显示）
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
	llmClient    port.LLMClient
	toolReg      port.ToolRegistry
	session      *entity.Session
	config       AgentConfig
	logger       logger.Logger
	handler      OutputHandler
	system       string
	todoTool     string
	todoRounds   int
	todoNagAfter int

	// SubAgent 相关
	isSubAgent     bool                // 是否为子 Agent
	excludeTools   []string            // 子 Agent 排除的工具列表
	subAgentConfig port.SubAgentConfig // 子 Agent 配置
}

// NewAgent 创建 Agent
func NewAgent(llmClient port.LLMClient, toolReg port.ToolRegistry, session *entity.Session, config AgentConfig) *Agent {
	// 默认开启流式
	if config.MaxTokens == 0 {
		config.MaxTokens = 8000
	}

	return &Agent{
		llmClient:    llmClient,
		toolReg:      toolReg,
		session:      session,
		config:       config,
		logger:       logger.Default().WithPrefix("agent"),
		todoTool:     "todo",
		todoNagAfter: 3,
		isSubAgent:   false,
	}
}

// NewSubAgent 创建子 Agent
// 子 Agent 使用独立的 Session，排除指定工具（防止递归调用 task 工具）
func NewSubAgent(llmClient port.LLMClient, toolReg port.ToolRegistry, config AgentConfig, subConfig port.SubAgentConfig) *Agent {
	// 创建独立的 Session
	session := entity.NewSession(llmClient.GetModel(), llmClient.GetName())

	agent := NewAgent(llmClient, toolReg, session, config)
	agent.isSubAgent = true
	agent.excludeTools = []string{"task"} // 子 Agent 不能使用 task 工具
	agent.subAgentConfig = subConfig

	// 设置子 Agent 的系统提示
	if subConfig.SystemPrompt != "" {
		agent.system = subConfig.SystemPrompt
	}

	return agent
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
	a.todoRounds = 0

	// 添加用户消息到会话
	userMsg := entity.NewMessage(entity.RoleUser, input)
	a.session.AddMessage(userMsg)

	return a.Loop(ctx)
}

// Run 实现 SubAgentRunner 接口
// 子 Agent 在独立上下文中执行任务，只返回最终摘要
func (a *Agent) Run(ctx context.Context, prompt string) (string, error) {
	// 重置 todo 计数
	a.todoRounds = 0

	// 创建独立的用户消息
	userMsg := entity.NewMessage(entity.RoleUser, prompt)
	a.session.AddMessage(userMsg)

	// 执行 Agent 循环
	var finalContent strings.Builder
	maxIterations := a.subAgentConfig.MaxIterations
	if maxIterations == 0 {
		maxIterations = 30 // 默认最大迭代次数
	}

	iterationCount := 0
	for iterationCount < maxIterations {
		iterationCount++
		select {
		case <-ctx.Done():
			return finalContent.String(), errors.New(errors.CodeContextCanceled, "context canceled")
		default:
		}

		// 调用 LLM
		content, toolCalls, err := a.callLLMStream(ctx)
		if err != nil {
			return finalContent.String(), fmt.Errorf("API call failed: %v", err)
		}

		// 如果没有工具调用，返回最终内容
		if len(toolCalls) == 0 {
			finalContent.WriteString(content)
			return finalContent.String(), nil
		}

		// 添加 assistant 消息
		assistantMsg := entity.NewMessage(entity.RoleAssistant, content).
			WithToolCalls(toolCalls)
		a.session.AddMessage(assistantMsg)

		// 执行工具调用
		results := make([]entity.ToolResult, 0, len(toolCalls))
		for _, toolCall := range toolCalls {
			result, err := a.executeToolSilent(ctx, toolCall)
			if err != nil {
				a.logger.Error("tool execution failed",
					logger.F("tool", toolCall.GetName()),
					logger.F("error", err),
				)
			}
			results = append(results, result)
		}

		// 添加工具结果到会话
		for _, result := range results {
			toolMsg := entity.NewMessage(entity.RoleTool, result.Content).
				WithToolCallID(result.ToolCallID)
			a.session.AddMessage(toolMsg)
		}
	}

	return finalContent.String(), fmt.Errorf("max iterations (%d) reached", maxIterations)
}

// executeToolSilent 静默执行工具（不发送输出）
func (a *Agent) executeToolSilent(ctx context.Context, call entity.ToolCall) (entity.ToolResult, error) {
	// 检查工具是否被排除
	for _, excluded := range a.excludeTools {
		if call.GetName() == excluded {
			return entity.ToolResult{
				ToolCallID: call.ID,
				Content:    "Error: This tool is not available in subagent mode",
				IsError:    true,
			}, nil
		}
	}

	// 执行工具
	return a.toolReg.ExecuteTool(ctx, call)
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

		usedTodo := false
		results := make([]entity.ToolResult, 0, len(toolCalls))
		for _, toolCall := range toolCalls {
			if toolCall.GetName() == a.todoTool {
				usedTodo = true
			}

			result, err := a.executeTool(ctx, toolCall)
			if err != nil {
				a.logger.Error("tool execution failed",
					logger.F("tool", toolCall.GetName()),
					logger.F("error", err),
				)
			}

			results = append(results, result)
		}

		results = a.injectTodoReminder(results, usedTodo)
		for _, result := range results {
			toolMsg := entity.NewMessage(entity.RoleTool, result.Content).
				WithToolCallID(result.ToolCallID)
			a.session.AddMessage(toolMsg)
			a.emitToolResult(result)
		}
	}
}

// callLLMStream 流式调用 LLM
func (a *Agent) callLLMStream(ctx context.Context) (string, []entity.ToolCall, error) {
	// 构建消息
	messages := a.buildMessages()

	// 获取工具列表（子 Agent 需要过滤）
	tools := a.getTools()

	// 构建请求
	req := &port.ChatRequest{
		Model:       a.llmClient.GetModel(),
		Messages:    messages,
		Stream:      true,
		MaxTokens:   a.config.MaxTokens,
		Temperature: a.config.Temperature,
		Tools:       tools,
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
			// 子 Agent 不输出流式内容到父 Agent
			if !a.isSubAgent {
				a.emit(OutputTextChunk, delta.Content)
			}
		}

		// 累积工具调用
		for _, tc := range delta.ToolCalls {
			idx := tc.Index

			// 确保索引位置存在
			if toolCallsMap[idx] == nil {
				toolCallsMap[idx] = &entity.ToolCall{
					Type:     "function",
					Function: entity.FunctionCall{},
				}
			}

			existing := toolCallsMap[idx]

			// 更新非空字段
			if tc.ID != "" {
				existing.ID = tc.ID
			}
			if tc.Type != "" {
				existing.Type = tc.Type
			}
			if tc.Function.Name != "" {
				existing.Function.Name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
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

// getTools 获取工具列表
// 子 Agent 需要排除特定工具（如 task）
func (a *Agent) getTools() []port.ToolDefinition {
	allTools := a.toolReg.ToLLMTools()

	// 如果不是子 Agent 或者没有排除工具，直接返回
	if !a.isSubAgent || len(a.excludeTools) == 0 {
		return allTools
	}

	// 过滤排除的工具
	excludeSet := make(map[string]bool)
	for _, name := range a.excludeTools {
		excludeSet[name] = true
	}

	filtered := make([]port.ToolDefinition, 0, len(allTools))
	for _, tool := range allTools {
		if !excludeSet[tool.Function.Name] {
			filtered = append(filtered, tool)
		}
	}

	return filtered
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

	return result, nil
}

// injectTodoReminder 在工具执行结果中注入 Todo 提醒
// 该函数用于追踪 LLM 在多轮对话中是否使用了 todo 工具，若连续多轮未使用则自动注入提醒
// 参数:
//   - results: 工具执行结果列表
//   - usedTodo: 本轮是否使用了 todo 工具
//
// 返回:
//   - 可能被修改的工具执行结果列表
func (a *Agent) injectTodoReminder(results []entity.ToolResult, usedTodo bool) []entity.ToolResult {
	// 如果本轮使用了 todo 工具，重置计数器并直接返回原结果
	if usedTodo {
		a.todoRounds = 0
		return results
	}

	// 未使用 todo 工具，计数器递增
	a.todoRounds++

	// 若未达到提醒阈值，直接返回原结果
	if a.todoRounds < a.todoNagAfter {
		return results
	}

	// 如果没有结果，无需注入提醒
	if len(results) == 0 {
		return results
	}

	// 在第一个工具结果的 Content 前添加提醒信息
	results[0].Content = "<reminder>Update your todos.</reminder>\n" + results[0].Content
	return results
}

func (a *Agent) emitToolResult(result entity.ToolResult) {
	displayResult := result.Content
	if len(displayResult) > 1000 {
		displayResult = displayResult[:1000] + "..."
	}
	a.emit(OutputResult, displayResult)
}

// SwitchModel 切换模型
func (a *Agent) SwitchModel(model string) {
	a.llmClient.SetModel(model)
	a.session.SetModel(model)
}

// IsSubAgent 返回是否为子 Agent
func (a *Agent) IsSubAgent() bool {
	return a.isSubAgent
}