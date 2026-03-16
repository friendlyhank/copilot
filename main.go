package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"ai_code/agent"
	"ai_code/llm"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

// 可用模型列表
var availableModels = []string{
	"qwen3-coder-plus",
	"qwen3-max",
	"kimi-k2-0905",
	"deepseek-v3.2",
}

// 样式定义
var (
	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6fc2ef")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a9a9a9")).
			Padding(0, 1)

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6fc2ef"))

	modelSelectorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")).
				Background(lipgloss.Color("#1e1e2e")).
				Padding(0, 1)

	selectedModelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#1e1e2e")).
				Background(lipgloss.Color("#6fc2ef")).
				Padding(0, 1)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6fc2ef"))

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#98c379"))

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e5c07b"))

	resultStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#98c379"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e06c75"))

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6fc2ef"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5c6370")).
			Italic(true)
)

// 模型状态
type modelState int

const (
	stateInput modelState = iota
	stateModelSelector
	stateProcessing
)

// 消息类型
type (
	agentOutputMsg struct {
		Type    agent.OutputType
		Content string
	}
	agentDoneMsg struct {
		err error
	}
	// tickMsg 计时消息
	tickMsg struct{}
)

// Message 消息结构
type Message struct {
	Type    agent.OutputType
	Content string
}

// Model 应用模型
type Model struct {
	state      modelState
	textInput  textinput.Model
	client     llm.Client
	messages   []Message
	cwd        string
	thinking   bool
	debug      bool
	modelIndex int
	quitting   bool

	// 处理中状态
	elapsedSeconds int

	// Agent 相关
	ag         *agent.Agent
	outputChan chan agent.Output
	cancelCtx  context.CancelFunc
	wg         sync.WaitGroup
}

// 初始模型
func initialModel(client llm.Client, cwd string, debug bool) Model {
	ti := textinput.New()
	ti.Placeholder = "Type your message or /model"
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60

	return Model{
		state:     stateInput,
		textInput: ti,
		client:    client,
		cwd:       cwd,
		thinking:  true,
		debug:     debug,
		messages:  []Message{},
	}
}

// Init 初始化
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update 更新状态
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			if m.state == stateProcessing && m.cancelCtx != nil {
				m.cancelCtx()
				m.wg.Wait()
				m.messages = append(m.messages, Message{
					Type:    agent.OutputError,
					Content: "Request cancelled",
				})
				m.state = stateInput
				return m, nil
			}
			if m.state == stateModelSelector {
				m.state = stateInput
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEsc:
			if m.state == stateModelSelector {
				m.state = stateInput
				return m, nil
			}
			// 处理中按 Esc 取消
			if m.state == stateProcessing && m.cancelCtx != nil {
				m.cancelCtx()
				m.wg.Wait()
				m.messages = append(m.messages, Message{
					Type:    agent.OutputError,
					Content: "Request cancelled",
				})
				m.state = stateInput
				return m, nil
			}

		case tea.KeyEnter:
			if m.state == stateModelSelector {
				m.client.SetModel(availableModels[m.modelIndex])
				m.messages = append(m.messages, Message{
					Type:    agent.OutputText,
					Content: fmt.Sprintf("Switched to model: %s", availableModels[m.modelIndex]),
				})
				m.state = stateInput
				return m, nil
			}

			if m.state == stateProcessing {
				return m, nil
			}

			input := strings.TrimSpace(m.textInput.Value())
			if input == "" {
				return m, nil
			}

			if strings.HasPrefix(input, "/") {
				return m.handleCommand(input)
			}

			m.messages = append(m.messages, Message{
				Type:    agent.OutputText,
				Content: "You: " + input,
			})
			m.textInput.SetValue("")
			return m.handleMessage(input)

		case tea.KeyTab:
			if m.state == stateInput {
				m.thinking = !m.thinking
				status := "off"
				if m.thinking {
					status = "on"
				}
				m.messages = append(m.messages, Message{
					Type:    agent.OutputText,
					Content: fmt.Sprintf("Thinking mode: %s", status),
				})
			}
			return m, nil

		case tea.KeyUp:
			if m.state == stateModelSelector && m.modelIndex > 0 {
				m.modelIndex--
				return m, nil
			}

		case tea.KeyDown:
			if m.state == stateModelSelector && m.modelIndex < len(availableModels)-1 {
				m.modelIndex++
				return m, nil
			}
		}

	case tickMsg:
		if m.state == stateProcessing {
			m.elapsedSeconds++
			return m, m.tick()
		}

	case agentOutputMsg:
		if msg.Type == agent.OutputDone {
			// 继续监听
			if m.state == stateProcessing && m.outputChan != nil {
				return m, m.listenOutput()
			}
			return m, nil
		}
		m.messages = append(m.messages, Message{
			Type:    msg.Type,
			Content: msg.Content,
		})
		// 继续监听下一个输出
		if m.state == stateProcessing {
			return m, m.listenOutput()
		}
		return m, nil

	case agentDoneMsg:
		m.state = stateInput
		if msg.err != nil {
			m.messages = append(m.messages, Message{
				Type:    agent.OutputError,
				Content: fmt.Sprintf("Error: %v", msg.err),
			})
		}
		return m, nil
	}

	if m.state != stateModelSelector {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleCommand 处理命令
func (m *Model) handleCommand(input string) (tea.Model, tea.Cmd) {
	cmd := strings.ToLower(strings.TrimSpace(input))

	switch {
	case cmd == "/model":
		m.state = stateModelSelector
		m.modelIndex = 0
		currentModel := m.client.GetModel()
		for i, model := range availableModels {
			if model == currentModel {
				m.modelIndex = i
				break
			}
		}
	case cmd == "/help":
		m.messages = append(m.messages, Message{
			Type: agent.OutputText,
			Content: "Commands:\n  /model - Switch model\n  /help - Show help\n  /clear - Clear messages\n  Tab - Toggle thinking\n  Ctrl+C - Quit",
		})
	case cmd == "/clear":
		m.messages = []Message{}
	case cmd == "/q", cmd == "/quit", cmd == "/exit":
		m.quitting = true
		return m, tea.Quit
	default:
		m.messages = append(m.messages, Message{
			Type:    agent.OutputError,
			Content: "Unknown command: " + cmd,
		})
	}

	m.textInput.SetValue("")
	return *m, nil
}

// handleMessage 处理消息
func (m *Model) handleMessage(input string) (tea.Model, tea.Cmd) {
	systemPrompt := fmt.Sprintf(
		"You are a coding agent at %s. Use bash tool to solve tasks. Be concise.",
		m.cwd,
	)
	if m.thinking {
		systemPrompt += " Think step by step."
	}

	m.ag = agent.NewAgent(m.client, systemPrompt)
	m.ag.SetDebug(m.debug)
	m.ag.Messages = []llm.Message{{Role: "user", Content: input}}

	m.outputChan = make(chan agent.Output, 100)
	m.ag.SetOutputHandler(func(output agent.Output) {
		m.outputChan <- output
	})

	m.state = stateProcessing
	m.elapsedSeconds = 0

	return *m, tea.Batch(m.tick(), m.runAgent())
}

// runAgent 运行 agent 并监听输出
func (m *Model) runAgent() tea.Cmd {
	return tea.Batch(
		m.startAgent(),
		m.listenOutput(),
	)
}

// startAgent 启动 agent
func (m *Model) startAgent() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelCtx = cancel

		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			defer close(m.outputChan)
			_ = m.ag.Loop(ctx)
		}()

		return nil
	}
}

// listenOutput 持续监听输出
func (m *Model) listenOutput() tea.Cmd {
	return func() tea.Msg {
		if m.outputChan == nil {
			return agentDoneMsg{}
		}

		output, ok := <-m.outputChan
		if !ok {
			return agentDoneMsg{}
		}
		if output.Type == agent.OutputDone {
			return agentDoneMsg{}
		}
		return agentOutputMsg{Type: output.Type, Content: output.Content}
	}
}

// tick 计时器
func (m *Model) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// View 渲染视图
func (m Model) View() string {
	if m.quitting {
		return "\nGoodbye!\n"
	}

	var b strings.Builder

	// 显示消息历史
	maxLines := 15
	start := 0
	if len(m.messages) > maxLines {
		start = len(m.messages) - maxLines
	}
	for i := start; i < len(m.messages); i++ {
		b.WriteString(m.renderMessage(m.messages[i]) + "\n")
	}

	// 模型选择器
	if m.state == stateModelSelector {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Select Model:") + "\n\n")
		for i, model := range availableModels {
			if i == m.modelIndex {
				b.WriteString("  " + selectedModelStyle.Render("▶ "+model) + "\n")
			} else {
				b.WriteString("  " + modelSelectorStyle.Render("  "+model) + "\n")
			}
		}
		b.WriteString("\n" + helpStyle.Render("↑/↓ select • Enter confirm • Esc cancel") + "\n")
		return b.String()
	}

	// 分隔线
	b.WriteString(dividerStyle.Render(strings.Repeat("─", 50)) + "\n")

	// 输入区域
	b.WriteString("\n")
	if m.state == stateProcessing {
		b.WriteString(inputStyle.Render(fmt.Sprintf("Generating %ds ...", m.elapsedSeconds)) + "\n")
		b.WriteString(helpStyle.Render("(esc to cancel)") + "\n")
	} else {
		b.WriteString(inputStyle.Render("> "+m.textInput.View()) + "\n")
		b.WriteString(helpStyle.Render("Tab: thinking • /model: switch • /help: commands") + "\n")
	}

	// 状态栏
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar() + "\n")

	return b.String()
}

// renderMessage 渲染消息
func (m Model) renderMessage(msg Message) string {
	switch msg.Type {
	case agent.OutputText:
		if strings.HasPrefix(msg.Content, "You: ") {
			return userStyle.Render(msg.Content)
		}
		return assistantStyle.Render(msg.Content)
	case agent.OutputCommand:
		return commandStyle.Render("$ " + msg.Content)
	case agent.OutputResult:
		return resultStyle.Render(msg.Content)
	case agent.OutputError:
		return errorStyle.Render(msg.Content)
	default:
		return msg.Content
	}
}

// renderStatusBar 渲染状态栏
func (m Model) renderStatusBar() string {
	thinkingStatus := "Off"
	if m.thinking {
		thinkingStatus = "On"
	}

	cwdDisplay := m.cwd
	if len(cwdDisplay) > 35 {
		home := os.Getenv("HOME")
		if strings.HasPrefix(cwdDisplay, home) {
			cwdDisplay = "~" + cwdDisplay[len(home):]
		}
		if len(cwdDisplay) > 35 {
			cwdDisplay = "..." + cwdDisplay[len(cwdDisplay)-32:]
		}
	}

	return statusStyle.Render(fmt.Sprintf(
		"%s | Thinking: %s | cwd: %s",
		m.client.GetModel(),
		thinkingStatus,
		cwdDisplay,
	))
}

func main() {
	godotenv.Load()

	provider := getEnvOrDefault("LLM_PROVIDER", "iflow")
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("IFLOW_API_KEY")
	}
	model := getEnvOrDefault("LLM_MODEL", "qwen3-coder-plus")
	baseURL := os.Getenv("LLM_BASE_URL")
	debug := os.Getenv("LLM_DEBUG") == "true"

	if apiKey == "" {
		fmt.Println("Error: LLM_API_KEY environment variable is required")
		os.Exit(1)
	}

	config := llm.ProviderConfig{
		Name:    provider,
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	}

	client, err := llm.GetProvider(provider, config)
	if err != nil {
		fmt.Printf("Error creating LLM client: %v\n", err)
		os.Exit(1)
	}

	if setter, ok := client.(interface{ SetDebug(bool) }); ok && debug {
		setter.SetDebug(true)
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	p := tea.NewProgram(
		initialModel(client, cwd, debug),
		tea.WithAltScreen(),
	)

	m, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if m, ok := m.(Model); ok && m.quitting {
		fmt.Println("Goodbye!")
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}