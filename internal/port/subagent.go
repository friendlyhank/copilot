package port

import (
	"context"
)

// SubAgentRunner 子智能体运行器接口
// 用于在独立上下文中执行子任务，返回摘要结果
type SubAgentRunner interface {
	// Run 执行子智能体任务
	// prompt: 子任务描述
	// 返回: 任务执行摘要（不包含中间工具调用细节）
	Run(ctx context.Context, prompt string) (string, error)
}

// SubAgentConfig 子智能体配置
type SubAgentConfig struct {
	// MaxIterations 最大迭代次数（防止无限循环）
	MaxIterations int
	// MaxTokens 最大生成 token 数
	MaxTokens int
	// SystemPrompt 系统提示词
	SystemPrompt string
}
