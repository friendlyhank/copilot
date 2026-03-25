package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"

	"ai_code/internal/adapter/llm"
	"ai_code/internal/adapter/tool"
	tui "ai_code/internal/adapter/ui/tui"
	"ai_code/internal/config"
	"ai_code/internal/domain/entity"
	"ai_code/internal/port"
	"ai_code/pkg/logger"
)

func main() {
	// 加载 .env 文件
	godotenv.Load()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	log := logger.New(logger.Config{
		Level:      cfg.Logger.Level,
		Output:     cfg.Logger.Output,
		Format:     cfg.Logger.Format,
		MaxSize:    cfg.Logger.MaxSize,
		MaxBackups: cfg.Logger.MaxBackups,
		MaxAge:     cfg.Logger.MaxAge,
		Compress:   cfg.Logger.Compress,
	})
	logger.SetDefault(log)

	// 创建 LLM 客户端
	llmClient, err := llm.Get(cfg.LLM.Provider, port.ProviderConfig{
		Name:    cfg.LLM.Provider,
		APIKey:  cfg.LLM.APIKey,
		BaseURL: cfg.LLM.BaseURL,
		Model:   cfg.LLM.Model,
		Timeout: cfg.LLM.Timeout,
	})
	if err != nil {
		fmt.Printf("Error creating LLM client: %v\n", err)
		os.Exit(1)
	}

	// 获取工作目录
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// 创建工具注册表并注册所有工具
	// 核心洞察: 加工具 = 加 handler + 加 schema，循环不变
	toolReg := tool.NewRegistry()
	toolReg.Register(tool.NewBashTool(
		tool.WithTimeout(time.Duration(cfg.LLM.Timeout)*time.Second),
		tool.WithCWD(cwd),
	))
	toolReg.Register(tool.NewReadFileTool(
		tool.WithReadFileCWD(cwd),
	))
	toolReg.Register(tool.NewWriteFileTool(
		tool.WithWriteFileCWD(cwd),
	))
	toolReg.Register(tool.NewEditFileTool(
		tool.WithEditFileCWD(cwd),
	))
	toolReg.Register(tool.NewTodoWriteTool())

	// 创建会话
	session := entity.NewSession(cfg.LLM.Model, cfg.LLM.Provider)

	// 创建 TUI 模型
	model := tui.NewModel(llmClient, session, toolReg,
		tui.WithCWD(cwd),
		tui.WithThinking(cfg.Agent.Thinking),
		tui.WithAvailableModels(config.AvailableModels),
	)

	// 初始化文本输入
	ti := textinput.New()
	ti.Placeholder = "Type your message or /model"
	ti.Prompt = "" // 使用自定义样式的提示符
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60
	model.SetTextInput(&ti)

	// 启动 TUI
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	m, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if m, ok := m.(*tui.Model); ok && m.IsQuitting() {
		fmt.Println("Goodbye!")
	}
}
