package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"

	"ai_code/internal/domain/entity"
	"ai_code/internal/domain/errors"
	"ai_code/pkg/logger"
)

// BashTool Bash 命令工具
type BashTool struct {
	timeout time.Duration
	cwd     string
	logger  logger.Logger
}

// BashToolOption Bash 工具选项
type BashToolOption func(*BashTool)

// NewBashTool 创建 Bash 工具
func NewBashTool(opts ...BashToolOption) *BashTool {
	cwd, _ := os.Getwd()
	
	t := &BashTool{
		timeout: 120 * time.Second,
		cwd:     cwd,
		logger:  logger.Default().WithPrefix("bash"),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// WithTimeout 设置超时
func WithTimeout(d time.Duration) BashToolOption {
	return func(t *BashTool) {
		t.timeout = d
	}
}

// WithCWD 设置工作目录
func WithCWD(cwd string) BashToolOption {
	return func(t *BashTool) {
		t.cwd = cwd
	}
}

// Name 工具名称
func (t *BashTool) Name() string {
	return "bash"
}

// Description 工具描述
func (t *BashTool) Description() string {
	return "Run a shell command. Execute a bash command and return the output."
}

// Parameters 参数定义
func (t *BashTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The bash command to execute",
			},
		},
		"required": []string{"command"},
	}
}

// Execute 执行命令
func (t *BashTool) Execute(ctx context.Context, args string) (string, error) {
	// 解析参数
	var params struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", errors.Wrap(errors.CodeInvalidInput, "failed to parse arguments", err)
	}

	// 危险命令检查
	if err := t.checkDangerous(params.Command); err != nil {
		return "", err
	}

	// 执行命令
	output, err := t.runCommand(ctx, params.Command)
	if err != nil {
		return output, err
	}

	return output, nil
}

// checkDangerous 检查危险命令
func (t *BashTool) checkDangerous(command string) error {
	dangerous := []string{
		"rm -rf /",
		"rm -rf /*",
		":(){:|:&};:",  // Fork bomb
		"mkfs",
		"dd if=/dev/zero",
	}

	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return errors.New(errors.CodeToolError, "dangerous command blocked")
		}
	}
	return nil
}

// runCommand 执行命令
func (t *BashTool) runCommand(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = t.cwd

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	output := strings.TrimSpace(stdout.String() + stderr.String())

	t.logger.Debug("command executed",
		logger.F("command", command),
		logger.F("duration", duration),
		logger.F("error", err),
	)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return output, errors.Timeout("command execution timeout")
		}
		return output, errors.Wrap(errors.CodeToolError, "command failed", err)
	}

	if output == "" {
		output = "(no output)"
	}

	// 截断输出
	if len(output) > 50000 {
		output = output[:50000] + "\n... (output truncated)"
	}

	return output, nil
}

// ToToolCall 转换为工具调用
func (t *BashTool) ToToolCall(command string) entity.ToolCall {
	return entity.NewToolCall("bash", command)
}
