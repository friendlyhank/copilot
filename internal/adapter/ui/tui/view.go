package tui

import (
	"fmt"
	"os"
	"strings"

	"ai_code/internal/usecase"
)

// View 渲染视图
func (m *Model) View() string {
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
	if m.state == StateModelSelector {
		b.WriteString("\n")
		b.WriteString(m.styles.Help.Render("Select Model:") + "\n\n")
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

	// 分隔线
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", 50)) + "\n")

	// 输入区域
	b.WriteString("\n")
	if m.state == StateProcessing {
		b.WriteString(m.styles.Input.Render(fmt.Sprintf("Generating %ds ...", m.elapsedSeconds)) + "\n")
		b.WriteString(m.styles.Help.Render("(esc to cancel)") + "\n")
	} else {
		b.WriteString(m.styles.Input.Render("> "+m.renderTextInput()) + "\n")
		b.WriteString(m.styles.Help.Render("Tab: thinking • /model: switch • /help: commands") + "\n")
	}

	// 状态栏
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar() + "\n")

	return b.String()
}

// renderMessage 渲染消息
func (m *Model) renderMessage(msg UIMessage) string {
	switch msg.Type {
	case usecase.OutputText:
		if strings.HasPrefix(msg.Content, "You: ") {
			return m.styles.User.Render(msg.Content)
		}
		return m.styles.Assistant.Render(msg.Content)
	case usecase.OutputCommand:
		return m.styles.Command.Render("$ " + msg.Content)
	case usecase.OutputResult:
		return m.styles.Result.Render(msg.Content)
	case usecase.OutputError:
		return m.styles.Error.Render(msg.Content)
	default:
		return msg.Content
	}
}

// renderStatusBar 渲染状态栏
func (m *Model) renderStatusBar() string {
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

	return m.styles.Status.Render(fmt.Sprintf(
		"%s | Thinking: %s | cwd: %s",
		m.llmClient.GetModel(),
		thinkingStatus,
		cwdDisplay,
	))
}

// renderTextInput 渲染文本输入（需要在外部实现）
func (m *Model) renderTextInput() string {
	// 这个方法会在具体的 Bubble Tea 实现中覆盖
	return ""
}
