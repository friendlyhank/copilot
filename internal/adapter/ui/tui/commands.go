package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"ai_code/internal/usecase"
)

// handleCommand 处理命令
func (m *Model) handleCommand(input string) (tea.Model, tea.Cmd) {
	cmd := strings.ToLower(strings.TrimSpace(input))

	switch {
	case cmd == "/model":
		m.state = StateModelSelector
		m.modelIndex = 0
		currentModel := m.llmClient.GetModel()
		for i, model := range m.availableModels {
			if model == currentModel {
				m.modelIndex = i
				break
			}
		}
	case cmd == "/help":
		m.messages = append(m.messages, UIMessage{
			Type: usecase.OutputText,
			Content: `📚 AI Code Help

Commands:
  /model  - Switch AI model
  /help   - Show this help message
  /clear  - Clear conversation history
  /quit   - Exit application

Keyboard Shortcuts:
  Tab     - Toggle thinking mode
  Enter   - Send message / Confirm selection
  Esc     - Cancel / Go back
  Ctrl+C  - Quit application

Tips:
  • Enable thinking mode for complex tasks
  • Use /model to switch between different AI models
  • Press Esc to cancel ongoing operations`,
		})
	case cmd == "/clear":
		m.messages = []UIMessage{}
	case cmd == "/q", cmd == "/quit", cmd == "/exit":
		m.quitting = true
		return m, tea.Quit
	default:
		m.messages = append(m.messages, UIMessage{
			Type:    usecase.OutputError,
			Content: "Unknown command: " + cmd,
		})
	}

	if ti, ok := m.textInput.(*textinput.Model); ok {
		ti.SetValue("")
		m.textInput = ti
	}
	return m, nil
}

// handleMessage 处理消息
func (m *Model) handleMessage(input string) (tea.Model, tea.Cmd) {
	// 保存当前输入
	m.currentInput = input

	systemPrompt := fmt.Sprintf("You are a coding agent at %s. Use tools to solve tasks. Act, don't explain.", m.cwd)
	if m.thinking {

	}

	// 创建 Agent
	m.agent = usecase.NewAgent(m.llmClient, m.toolReg, m.session, usecase.AgentConfig{
		MaxTokens:   8000,
		Temperature: 0.7,
		Thinking:    m.thinking,
	})
	m.agent.SetSystem(systemPrompt)

	// 设置输出处理器
	m.outputChan = make(chan usecase.Output, 100)
	m.agent.SetOutputHandler(func(output usecase.Output) {
		m.outputChan <- output
	})

	m.state = StateProcessing
	m.elapsedSeconds = 0

	return m, tea.Batch(m.tick(), m.startAgent(), m.listenOutput())
}

// startAgent 启动 Agent
func (m *Model) startAgent() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelCtx = cancel

		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			defer close(m.outputChan)
			_ = m.agent.ProcessMessage(ctx, m.currentInput)
		}()

		return nil
	}
}
