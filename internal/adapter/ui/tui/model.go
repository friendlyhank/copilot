package tui

import (
	"context"
	"sync"

	"ai_code/internal/domain/entity"
	"ai_code/internal/port"
	"ai_code/internal/usecase"
)

// ModelState 模型状态
type ModelState int

const (
	StateWelcome ModelState = iota
	StateInput
	StateModelSelector
	StateProcessing
)

// UIMessage UI 消息
type UIMessage struct {
	Type    usecase.OutputType
	Content string
}

// Model TUI 模型
type Model struct {
	// 状态
	state    ModelState
	quitting bool
	thinking bool

	// 当前用户输入
	currentInput string

	// 输入组件
	textInput interface{} // *textinput.Model

	// LLM 客户端
	llmClient port.LLMClient

	// 会话
	session *entity.Session

	// 工具注册表
	toolReg port.ToolRegistry

	// UI 消息历史
	messages []UIMessage

	// 可用模型列表
	availableModels []string
	modelIndex      int

	// 工作目录
	cwd string

	// Agent 相关
	agent      *usecase.Agent
	outputChan chan usecase.Output
	cancelCtx  context.CancelFunc
	wg         sync.WaitGroup

	// 处理中状态
	elapsedSeconds int

	// 样式
	styles Styles
}

// ModelOption 模型选项
type ModelOption func(*Model)

// NewModel 创建模型
func NewModel(llmClient port.LLMClient, session *entity.Session, toolReg port.ToolRegistry, opts ...ModelOption) *Model {
	m := &Model{
		state:           StateInput,
		llmClient:       llmClient,
		session:         session,
		toolReg:         toolReg,
		messages:        []UIMessage{},
		availableModels: []string{},
		styles:          DefaultStyles,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// WithCWD 设置工作目录
func WithCWD(cwd string) ModelOption {
	return func(m *Model) {
		m.cwd = cwd
	}
}

// WithThinking 设置思考模式
func WithThinking(thinking bool) ModelOption {
	return func(m *Model) {
		m.thinking = thinking
	}
}

// WithAvailableModels 设置可用模型列表
func WithAvailableModels(models []string) ModelOption {
	return func(m *Model) {
		m.availableModels = models
	}
}

// SetTextInput 设置文本输入组件
func (m *Model) SetTextInput(ti interface{}) {
	m.textInput = ti
}

// IsQuitting 检查是否正在退出
func (m *Model) IsQuitting() bool {
	return m.quitting
}

// GetState 获取当前状态
func (m *Model) GetState() ModelState {
	return m.state
}
