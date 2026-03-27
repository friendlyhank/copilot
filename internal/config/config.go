package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构
// 注意: LLM 配置从 .env 环境变量读取，不通过 yaml 文件配置
type Config struct {
	LLM    LLMConfig    // 从环境变量读取
	UI     UIConfig     `yaml:"ui"`
	Agent  AgentConfig  `yaml:"agent"`
	Logger LoggerConfig `yaml:"logger"`
}

// LLMConfig LLM 配置 (从 .env 读取)
type LLMConfig struct {
	Provider       string   // LLM 提供商: iflow, openai
	APIKey         string   // API Key
	BaseURL        string   // API Base URL (可选)
	Model          string   // 默认模型
	AvailableModels []string // 可选模型列表
	Timeout        int      // 请求超时时间(秒)
}

// UIConfig UI 配置
type UIConfig struct {
	Theme    string `yaml:"theme"`
	MaxLines int    `yaml:"max_lines"`
}

// AgentConfig Agent 配置
type AgentConfig struct {
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
	Thinking    bool    `yaml:"thinking"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level      string `yaml:"level"`       // debug, info, warn, error
	Output     string `yaml:"output"`      // stdout, stderr, 或文件路径
	Format     string `yaml:"format"`      // json, text
	MaxSize    int    `yaml:"max_size"`    // MB, 日志文件最大大小
	MaxBackups int    `yaml:"max_backups"` // 保留的旧文件数量
	MaxAge     int    `yaml:"max_age"`     // 保留天数
	Compress   bool   `yaml:"compress"`    // 是否压缩旧文件
}

// Load 加载配置（优先级：环境变量 > 配置文件 > 默认值）
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// 1. 从配置文件加载
	configPaths := []string{
		"config/config.yaml",
		"config.yaml",
		filepath.Join(os.Getenv("HOME"), ".ai_code", "config.yaml"),
	}

	for _, path := range configPaths {
		if err := loadFromFile(cfg, path); err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("load config from %s: %w", path, err)
			}
			continue
		}
		break
	}

	// 2. 环境变量覆盖
	loadFromEnv(cfg)

	// 3. 验证
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadFromFile 从文件加载配置
func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, cfg)
}

// loadFromEnv 从环境变量加载配置
func loadFromEnv(cfg *Config) {
	if v := os.Getenv("LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = v
	}
	if v := os.Getenv("LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	} else if v := os.Getenv("IFLOW_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("LLM_BASE_URL"); v != "" {
		cfg.LLM.BaseURL = v
	}
	if v := os.Getenv("LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("LLM_TIMEOUT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.LLM.Timeout)
	}
	// 读取可选模型列表
	if v := os.Getenv("LLM_AVAILABLE_MODELS"); v != "" {
		models := strings.Split(v, ",")
		for i, m := range models {
			models[i] = strings.TrimSpace(m)
		}
		cfg.LLM.AvailableModels = models
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.LLM.APIKey == "" {
		return fmt.Errorf("LLM API key is required (set LLM_API_KEY or IFLOW_API_KEY environment variable)")
	}
	if c.LLM.Provider == "" {
		return fmt.Errorf("LLM provider is required")
	}
	return nil
}
