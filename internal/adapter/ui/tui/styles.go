package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// 主题颜色
type Theme struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color
	Text      lipgloss.Color
	Error     lipgloss.Color
	Success   lipgloss.Color
	Muted     lipgloss.Color
	Background lipgloss.Color
}

// DarkTheme 深色主题
var DarkTheme = Theme{
	Primary:    lipgloss.Color("#6fc2ef"),
	Secondary:  lipgloss.Color("#98c379"),
	Accent:     lipgloss.Color("#e5c07b"),
	Text:       lipgloss.Color("#ffffff"),
	Error:      lipgloss.Color("#e06c75"),
	Success:    lipgloss.Color("#98c379"),
	Muted:      lipgloss.Color("#5c6370"),
	Background: lipgloss.Color("#1e1e2e"),
}

// Styles 样式集合
type Styles struct {
	Input         lipgloss.Style
	Status        lipgloss.Style
	Divider       lipgloss.Style
	ModelSelector lipgloss.Style
	SelectedModel lipgloss.Style
	Assistant     lipgloss.Style
	User          lipgloss.Style
	Command       lipgloss.Style
	Result        lipgloss.Style
	Error         lipgloss.Style
	Spinner       lipgloss.Style
	Help          lipgloss.Style
}

// NewStyles 创建样式
func NewStyles(theme Theme) Styles {
	return Styles{
		Input: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Padding(0, 1),

		Status: lipgloss.NewStyle().
			Foreground(theme.Muted).
			Padding(0, 1),

		Divider: lipgloss.NewStyle().
			Foreground(theme.Primary),

		ModelSelector: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Background).
			Padding(0, 1),

		SelectedModel: lipgloss.NewStyle().
			Foreground(theme.Background).
			Background(theme.Primary).
			Padding(0, 1),

		Assistant: lipgloss.NewStyle().
			Foreground(theme.Primary),

		User: lipgloss.NewStyle().
			Foreground(theme.Secondary),

		Command: lipgloss.NewStyle().
			Foreground(theme.Accent),

		Result: lipgloss.NewStyle().
			Foreground(theme.Success),

		Error: lipgloss.NewStyle().
			Foreground(theme.Error),

		Spinner: lipgloss.NewStyle().
			Foreground(theme.Primary),

		Help: lipgloss.NewStyle().
			Foreground(theme.Muted).
			Italic(true),
	}
}

// DefaultStyles 默认样式
var DefaultStyles = NewStyles(DarkTheme)
