package tool

import (
	"context"
	"encoding/json"

	"ai_code/internal/domain/entity"
	"ai_code/internal/port"
	"ai_code/internal/usecase"
)

// TaskTool 任务工具
// 用于启动子智能体执行独立任务，实现上下文隔离
type TaskTool struct {
	llmClient    port.LLMClient
	toolReg      port.ToolRegistry
	cwd          string
	subAgentOpts []SubAgentOption
}

// TaskToolParams 任务工具参数
type TaskToolParams struct {
	Prompt      string `json:"prompt"`                // 必需：子任务描述
	Description string `json:"description,omitempty"` // 可选：任务简短描述
}

// SubAgentOption 子 Agent 选项
type SubAgentOption func(*SubAgentConfig)

// SubAgentConfig 子 Agent 配置
type SubAgentConfig struct {
	MaxIterations int
	MaxTokens     int
	SystemPrompt  string
}

// WithMaxIterations 设置最大迭代次数
func WithMaxIterations(n int) SubAgentOption {
	return func(c *SubAgentConfig) {
		c.MaxIterations = n
	}
}

// WithMaxTokens 设置最大 token 数
func WithMaxTokens(n int) SubAgentOption {
	return func(c *SubAgentConfig) {
		c.MaxTokens = n
	}
}

// WithSystemPrompt 设置系统提示
func WithSystemPrompt(prompt string) SubAgentOption {
	return func(c *SubAgentConfig) {
		c.SystemPrompt = prompt
	}
}

// NewTaskTool 创建任务工具
func NewTaskTool(llmClient port.LLMClient, toolReg port.ToolRegistry, cwd string, opts ...SubAgentOption) *TaskTool {
	return &TaskTool{
		llmClient:    llmClient,
		toolReg:      toolReg,
		cwd:          cwd,
		subAgentOpts: opts,
	}
}

// Name 工具名称
func (t *TaskTool) Name() string {
	return "task"
}

// Description 工具描述
func (t *TaskTool) Description() string {
	return "Spawn a subagent with fresh context. It shares the filesystem but not conversation history."
}

// Parameters 工具参数定义
func (t *TaskTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "Detailed instructions for the subagent to execute",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Short description of the task (for logging)",
			},
		},
		"required": []string{"prompt"},
	}
}

// Execute 执行工具
func (t *TaskTool) Execute(ctx context.Context, args string) (string, error) {
	var params TaskToolParams
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}

	if params.Prompt == "" {
		return "Error: prompt is required", nil
	}

	// 构建子 Agent 配置
	config := SubAgentConfig{
		MaxIterations: 30,
		MaxTokens:     8000,
	}
	for _, opt := range t.subAgentOpts {
		opt(&config)
	}

	subAgentSystem := config.SystemPrompt
	if subAgentSystem == "" {
		if t.cwd != "" {
			subAgentSystem = "You are a coding subagent at " + t.cwd + ". Complete the given task, then summarize your findings."
		} else {
			subAgentSystem = "You are a coding subagent. Complete the given task, then summarize your findings."
		}
	}

	// 创建子 Agent（使用独立的 Session）
	agentConfig := usecase.AgentConfig{
		MaxTokens: config.MaxTokens,
	}

	subAgentConfig := port.SubAgentConfig{
		MaxIterations: config.MaxIterations,
		MaxTokens:     config.MaxTokens,
		SystemPrompt:  subAgentSystem,
	}

	subAgent := usecase.NewSubAgent(t.llmClient, t.toolReg, agentConfig, subAgentConfig)

	// 创建独立 Session 用于子 Agent（虽然 NewSubAgent 内部已经创建，这里确保隔离）
	subSession := entity.NewSession(t.llmClient.GetModel(), t.llmClient.GetName())

	// 使用私有方法重新设置 session（需要修改 agent.go）
	// 这里简化处理：直接调用 Run 方法
	_ = subSession // 子 Agent 内部已有独立 session

	// 执行子 Agent
	summary, err := subAgent.Run(ctx, params.Prompt)
	if err != nil {
		return "Subagent error: " + err.Error(), nil
	}

	// 截断过长输出
	if len(summary) > 50000 {
		summary = summary[:50000] + "\n... (output truncated)"
	}

	if summary == "" {
		summary = "(no summary returned)"
	}

	return summary, nil
}
