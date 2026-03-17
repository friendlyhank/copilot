package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"ai_code/internal/usecase"
)

// Version 版本号
const Version = "v1.0.0"

// View 渲染视图
func (m *Model) View() string {
	if m.quitting {
		return m.styles.WelcomeTitle.Render("\n👋 Thanks for using AI Code! See you next time.\n")
	}

	// 构建底部输入区域（固定高度）
	inputArea := m.renderInputSection()
	statusBar := m.renderStatusBar()
	bottomSection := inputArea + "\n" + statusBar + "\n"

	// 构建消息历史区域
	var messagesArea string
	if m.state == StateModelSelector {
		messagesArea = m.renderModelSelector()
	} else {
		messagesArea = m.renderMessagesArea()
	}

	// 使用 lipgloss 将消息区域放在上方，输入区域固定在底部
	content := lipgloss.JoinVertical(lipgloss.Top,
		messagesArea,
		bottomSection,
	)

	return content
}

// renderMessagesArea 渲染消息区域
func (m *Model) renderMessagesArea() string {
	if len(m.messages) == 0 {
		// 显示简单的欢迎信息
		return m.styles.WelcomeText.Render("💬 Welcome to AI Code! Type your message below.") + "\n"
	}

	var b strings.Builder
	for _, msg := range m.messages {
		b.WriteString(m.renderMessage(msg) + "\n")
	}
	return b.String()
}

// renderInputSection 渲染输入区域（包含分隔线和输入框）
func (m *Model) renderInputSection() string {
	var b strings.Builder

	// 分隔线
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", 60)) + "\n")

	b.WriteString(m.renderInputArea())

	return b.String()
}

// renderModelSelector 渲染模型选择器
func (m *Model) renderModelSelector() string {
	var b strings.Builder

	b.WriteString(m.styles.WelcomeTitle.Render("📦 Select Model") + "\n\n")
	for i, model := range m.availableModels {
		if i == m.modelIndex {
			b.WriteString("  " + m.styles.SelectedModel.Render("▶ "+model) + "\n")
		} else {
			b.WriteString("  " + m.styles.ModelSelector.Render("  "+model) + "\n")
		}
	}
	b.WriteString("\n" + m.styles.Help.Render("↑/↓ select • Enter confirm • Esc cancel") + "\n")
	return b.String()
}

// renderProcessingIndicator 渲染处理中指示器
func (m *Model) renderProcessingIndicator() string {
	timeStr := formatDuration(m.elapsedSeconds)
	return m.styles.Help.Render(" (esc to cancel) ") + m.styles.Input.Render(timeStr)
}

// formatDuration 格式化秒数为易读格式
func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	secs := seconds % 60
	if minutes < 60 {
		if secs > 0 {
			return fmt.Sprintf("%dm %ds", minutes, secs)
		}
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dh", hours)
}

// renderInputArea 渲染输入区域
func (m *Model) renderInputArea() string {
	var b strings.Builder

	// 输入提示（光标紧跟 > 后面）
	b.WriteString(m.styles.InputPrompt.Render(">"))

	// 输入内容
	b.WriteString(m.styles.Input.Render(m.renderTextInput()))

	// Processing 状态时显示计时
	if m.state == StateProcessing {
		b.WriteString(m.renderProcessingIndicator())
	}
	b.WriteString("\n")

	// 快捷键提示
	hints := []string{
		m.styles.ShortcutKey.Render(" Tab ") + m.styles.Help.Render(" thinking"),
		m.styles.ShortcutKey.Render(" /model ") + m.styles.Help.Render(" switch"),
		m.styles.ShortcutKey.Render(" /help ") + m.styles.Help.Render(" commands"),
	}
	b.WriteString("  " + strings.Join(hints, "  ") + "\n")

	return b.String()
}

// renderMessage 渲染消息
func (m *Model) renderMessage(msg UIMessage) string {
	switch msg.Type {
	case usecase.OutputText:
		if strings.HasPrefix(msg.Content, "You: ") {
			// 用户消息显示为 "> hello"
			return m.styles.User.Render("> " + strings.TrimPrefix(msg.Content, "You: "))
		}
		return m.styles.Assistant.Render("🤖 " + msg.Content)
	case usecase.OutputCommand:
		return m.styles.Command.Render("⚡ $ " + msg.Content)
	case usecase.OutputResult:
		return m.styles.Result.Render("✓ " + msg.Content)
	case usecase.OutputError:
		return m.styles.Error.Render("✗ " + msg.Content)
	default:
		return msg.Content
	}
}

// renderStatusBar 渲染状态栏
func (m *Model) renderStatusBar() string {
	// Thinking 状态
	thinkingIcon := "○"
	thinkingText := "Off"
	if m.thinking {
		thinkingIcon = "●"
		thinkingText = "On"
	}

	// 工作目录
	cwdDisplay := m.cwd
	if len(cwdDisplay) > 30 {
		home := os.Getenv("HOME")
		if strings.HasPrefix(cwdDisplay, home) {
			cwdDisplay = "~" + cwdDisplay[len(home):]
		}
		if len(cwdDisplay) > 30 {
			cwdDisplay = "..." + cwdDisplay[len(cwdDisplay)-27:]
		}
	}

	// 构建状态栏
	statusParts := []string{
		m.styles.StatusItem.Render("🤖 " + m.llmClient.GetModel()),
		m.styles.Status.Render(thinkingIcon + " Think: " + thinkingText),
		m.styles.Status.Render("📁 " + cwdDisplay),
	}

	return m.styles.StatusBar.Render(strings.Join(statusParts, " │ "))
}

// renderTextInput 渲染文本输入
func (m *Model) renderTextInput() string {
	if ti, ok := m.textInput.(*textinput.Model); ok {
		return ti.View()
	}
	return ""
}
