package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"ai_code/internal/usecase"
)

// 消息类型
type (
	agentOutputMsg struct {
		Type    usecase.OutputType
		Content string
	}
	agentDoneMsg struct {
		err error
	}
	tickMsg struct{}
)

// Init 初始化
func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update 更新状态
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m.handleCtrlC()

		case tea.KeyEsc:
			return m.handleEsc()

		case tea.KeyEnter:
			return m.handleEnter()

		case tea.KeyTab:
			return m.handleTab()

		case tea.KeyUp:
			if m.state == StateModelSelector && m.modelIndex > 0 {
				m.modelIndex--
			}
			return m, nil

		case tea.KeyDown:
			if m.state == StateModelSelector && m.modelIndex < len(m.availableModels)-1 {
				m.modelIndex++
			}
			return m, nil
		}

	case tickMsg:
		if m.state == StateProcessing {
			m.elapsedSeconds++
			return m, m.tick()
		}

	case agentOutputMsg:
		return m.handleAgentOutput(msg)

	case agentDoneMsg:
		return m.handleAgentDone(msg)
	}

	// 处理文本输入（不在模型选择器状态时）
	if m.state != StateModelSelector {
		if ti, ok := m.textInput.(*textinput.Model); ok {
			var cmd tea.Cmd
			*ti, cmd = ti.Update(msg)
			m.textInput = ti
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// handleCtrlC 处理 Ctrl+C
func (m *Model) handleCtrlC() (tea.Model, tea.Cmd) {
	if m.state == StateProcessing && m.cancelCtx != nil {
		m.cancelCtx()
		m.wg.Wait()
		m.messages = append(m.messages, UIMessage{
			Type:    usecase.OutputError,
			Content: "Request cancelled",
		})
		m.state = StateInput
		return m, nil
	}
	if m.state == StateModelSelector {
		m.state = StateInput
		return m, nil
	}
	m.quitting = true
	return m, tea.Quit
}

// handleEsc 处理 Esc
func (m *Model) handleEsc() (tea.Model, tea.Cmd) {
	if m.state == StateModelSelector {
		m.state = StateInput
		return m, nil
	}
	if m.state == StateProcessing && m.cancelCtx != nil {
		m.cancelCtx()
		m.wg.Wait()
		m.messages = append(m.messages, UIMessage{
			Type:    usecase.OutputError,
			Content: "Request cancelled",
		})
		m.state = StateInput
		return m, nil
	}
	return m, nil
}

// handleEnter 处理 Enter
func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.state == StateModelSelector {
		m.llmClient.SetModel(m.availableModels[m.modelIndex])
		m.messages = append(m.messages, UIMessage{
			Type:    usecase.OutputText,
			Content: fmt.Sprintf("Switched to model: %s", m.availableModels[m.modelIndex]),
		})
		m.state = StateInput
		return m, nil
	}

	if m.state == StateProcessing {
		return m, nil
	}

	var input string
	if ti, ok := m.textInput.(*textinput.Model); ok {
		input = strings.TrimSpace(ti.Value())
	}
	if input == "" {
		return m, nil
	}

	if strings.HasPrefix(input, "/") {
		return m.handleCommand(input)
	}

	m.messages = append(m.messages, UIMessage{
		Type:    usecase.OutputText,
		Content: "You: " + input,
	})
	if ti, ok := m.textInput.(*textinput.Model); ok {
		ti.SetValue("")
		m.textInput = ti
	}
	return m.handleMessage(input)
}

// handleTab 处理 Tab
func (m *Model) handleTab() (tea.Model, tea.Cmd) {
	if m.state == StateInput {
		m.thinking = !m.thinking
		status := "off"
		if m.thinking {
			status = "on"
		}
		m.messages = append(m.messages, UIMessage{
			Type:    usecase.OutputText,
			Content: fmt.Sprintf("Thinking mode: %s", status),
		})
	}
	return m, nil
}

// handleAgentOutput 处理 Agent 输出
func (m *Model) handleAgentOutput(msg agentOutputMsg) (tea.Model, tea.Cmd) {
	if msg.Type == usecase.OutputDone {
		if m.state == StateProcessing && m.outputChan != nil {
			return m, m.listenOutput()
		}
		return m, nil
	}
	m.messages = append(m.messages, UIMessage{
		Type:    msg.Type,
		Content: msg.Content,
	})
	if m.state == StateProcessing {
		return m, m.listenOutput()
	}
	return m, nil
}

// handleAgentDone 处理 Agent 完成
func (m *Model) handleAgentDone(msg agentDoneMsg) (tea.Model, tea.Cmd) {
	m.state = StateInput
	if msg.err != nil {
		m.messages = append(m.messages, UIMessage{
			Type:    usecase.OutputError,
			Content: fmt.Sprintf("Error: %v", msg.err),
		})
	}
	return m, nil
}

// tick 计时器
func (m *Model) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// listenOutput 监听输出
func (m *Model) listenOutput() tea.Cmd {
	return func() tea.Msg {
		if m.outputChan == nil {
			return agentDoneMsg{}
		}
		output, ok := <-m.outputChan
		if !ok {
			return agentDoneMsg{}
		}
		if output.Type == usecase.OutputDone {
			return agentDoneMsg{}
		}
		return agentOutputMsg{Type: output.Type, Content: output.Content}
	}
}
