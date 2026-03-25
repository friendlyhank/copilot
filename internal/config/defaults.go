package config

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			Provider: "iflow",
			Model:    "qwen3-coder-plus",
			Timeout:  180,
		},
		UI: UIConfig{
			Theme:    "dark",
			MaxLines: 15,
		},
		Agent: AgentConfig{
			MaxTokens:   8000,
			Temperature: 0.7,
			Thinking:    true,
		},
		Logger: LoggerConfig{
			Level:      "info",
			Output:     "log/copilot.log",
			Format:     "text",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     30,
			Compress:   true,
		},
	}
}

// AvailableModels 可用模型列表
var AvailableModels = []string{
	"qwen3-coder-plus",
	"qwen3-max",
	"kimi-k2-0905",
	"deepseek-v3.2",
}
