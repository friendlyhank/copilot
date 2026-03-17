package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// 主题颜色
type Theme struct {
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	Text       lipgloss.Color
	Error      lipgloss.Color
	Success    lipgloss.Color
	Muted      lipgloss.Color
	Background lipgloss.Color
	Border     lipgloss.Color
	Gradient1  lipgloss.Color
	Gradient2  lipgloss.Color
}

// DarkTheme 深色主题 - 现代科技感配色
var DarkTheme = Theme{
	Primary:    lipgloss.Color("#7dd3fc"), // Sky blue - 主色调
	Secondary:  lipgloss.Color("#a5f3fc"), // Cyan - 次要色
	Accent:     lipgloss.Color("#fbbf24"), // Amber - 强调色
	Text:       lipgloss.Color("#f1f5f9"), // Slate 100 - 主文本
	Error:      lipgloss.Color("#f87171"), // Red 400 - 错误
	Success:    lipgloss.Color("#4ade80"), // Green 400 - 成功
	Muted:      lipgloss.Color("#64748b"), // Slate 500 - 弱化文本
	Background: lipgloss.Color("#0f172a"), // Slate 900 - 背景
	Border:     lipgloss.Color("#334155"), // Slate 700 - 边框
	Gradient1:  lipgloss.Color("#38bdf8"), // Sky 400
	Gradient2:  lipgloss.Color("#818cf8"), // Indigo 400
}

// Styles 样式集合
type Styles struct {
	// 输入相关
	Input       lipgloss.Style
	InputPrompt lipgloss.Style
	InputBox    lipgloss.Style

	// 状态栏
	Status     lipgloss.Style
	StatusBar  lipgloss.Style
	StatusItem lipgloss.Style

	// 分隔线
	Divider lipgloss.Style

	// 模型选择器
	ModelSelector lipgloss.Style
	SelectedModel lipgloss.Style

	// 消息样式
	Assistant lipgloss.Style
	User      lipgloss.Style
	Command   lipgloss.Style
	Result    lipgloss.Style
	Error     lipgloss.Style

	// 动画
	Spinner lipgloss.Style

	// 帮助
	Help    lipgloss.Style
	KeyHint lipgloss.Style

	// 欢迎页面
	Logo         lipgloss.Style
	LogoAccent   lipgloss.Style
	WelcomeTitle lipgloss.Style
	WelcomeText  lipgloss.Style
	Version      lipgloss.Style
	ShortcutKey  lipgloss.Style
	ShortcutDesc lipgloss.Style
	Box          lipgloss.Style
}

// NewStyles 创建样式
func NewStyles(theme Theme) Styles {
	return Styles{
		// 输入样式 - 更现代化的输入框
		Input: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(lipgloss.Color("#1e293b")).
			Padding(0, 2).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(theme.Primary),

		InputPrompt: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Padding(0, 1),

		InputBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Background(lipgloss.Color("#1e293b")).
			Padding(1, 2).
			Margin(1, 0),

		// 状态栏 - 底部状态栏样式
		Status: lipgloss.NewStyle().
			Foreground(theme.Muted).
			Padding(0, 1),

		StatusBar: lipgloss.NewStyle().
			Background(lipgloss.Color("#1e293b")).
			Foreground(theme.Text).
			Padding(0, 2).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(theme.Border),

		StatusItem: lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Padding(0, 1),

		// 分隔线
		Divider: lipgloss.NewStyle().
			Foreground(theme.Border),

		// 模型选择器
		ModelSelector: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(lipgloss.Color("#1e293b")).
			Padding(0, 2).
			Border(lipgloss.NormalBorder(), false, true, false, true).
			BorderForeground(theme.Border),

		SelectedModel: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0f172a")).
			Background(theme.Primary).
			Padding(0, 2).
			Bold(true).
			Border(lipgloss.NormalBorder(), false, true, false, true).
			BorderForeground(theme.Primary),

		// 消息样式
		Assistant: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Padding(0, 1),

		User: lipgloss.NewStyle().
			Foreground(theme.Success).
			Bold(true).
			Padding(0, 1),

		Command: lipgloss.NewStyle().
			Foreground(theme.Accent).
			Padding(0, 1),

		Result: lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Padding(0, 1),

		Error: lipgloss.NewStyle().
			Foreground(theme.Error).
			Padding(0, 1),

		// 动画
		Spinner: lipgloss.NewStyle().
			Foreground(theme.Primary),

		// 帮助
		Help: lipgloss.NewStyle().
			Foreground(theme.Muted).
			Italic(true).
			Padding(0, 1),

		KeyHint: lipgloss.NewStyle().
			Foreground(theme.Accent).
			Bold(true).
			Padding(0, 1),

		// 欢迎页面样式
		Logo: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true),

		LogoAccent: lipgloss.NewStyle().
			Foreground(theme.Accent).
			Bold(true),

		WelcomeTitle: lipgloss.NewStyle().
			Foreground(theme.Text).
			Bold(true).
			Padding(1, 0),

		WelcomeText: lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Padding(0, 1),

		Version: lipgloss.NewStyle().
			Foreground(theme.Muted).
			Padding(0, 1),

		ShortcutKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0f172a")).
			Background(theme.Primary).
			Padding(0, 1).
			Bold(true),

		ShortcutDesc: lipgloss.NewStyle().
			Foreground(theme.Text).
			Padding(0, 1),

		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Primary).
			Padding(1, 3).
			Margin(1, 2),
	}
}

// DefaultStyles 默认样式
var DefaultStyles = NewStyles(DarkTheme)

// Logo ASCII Art
const Logo = `
██╗ ██████╗██╗  ██╗███████╗████████╗███████╗██████╗ ███╗   ███╗
██║██╔════╝██║  ██║██╔════╝╚══██╔══╝██╔════╝██╔══██╗████╗ ████║
██║██║     ███████║█████╗     ██║   █████╗   ██████╔╝██╔████╔██║
██║██║     ██╔══██║██╔══╝     ██║   ██╔══╝   ██╔══██╗██║╚██╔╝██║
██║╚██████╗██║  ██║███████╗   ██║   ███████╗██║  ██║██║ ╚═╝ ██║
╚═╝ ╚═════╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝`
