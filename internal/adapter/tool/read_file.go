package tool

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"ai_code/internal/domain/entity"
	"ai_code/internal/domain/errors"
	"ai_code/pkg/logger"
)

// ReadFileTool 文件读取工具
type ReadFileTool struct {
	cwd    string
	logger logger.Logger
}

// ReadFileToolOption 读取工具选项
type ReadFileToolOption func(*ReadFileTool)

// NewReadFileTool 创建文件读取工具
func NewReadFileTool(opts ...ReadFileToolOption) *ReadFileTool {
	cwd, _ := os.Getwd()

	t := &ReadFileTool{
		cwd:    cwd,
		logger: logger.Default().WithPrefix("read_file"),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// WithReadFileCWD 设置工作目录
func WithReadFileCWD(cwd string) ReadFileToolOption {
	return func(t *ReadFileTool) {
		t.cwd = cwd
	}
}

// Name 工具名称
func (t *ReadFileTool) Name() string {
	return "read_file"
}

// Description 工具描述
func (t *ReadFileTool) Description() string {
	return "Read file contents. Returns the text content of a file."
}

// Parameters 参数定义
func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The path to the file to read",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Optional: maximum number of lines to read",
			},
		},
		"required": []string{"path"},
	}
}

// Execute 执行读取
func (t *ReadFileTool) Execute(ctx context.Context, args string) (string, error) {
	// 解析参数
	var params struct {
		Path  string `json:"path"`
		Limit *int   `json:"limit"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", errors.Wrap(errors.CodeInvalidInput, "failed to parse arguments", err)
	}

	// 安全路径检查
	safePath, err := SafePath(t.cwd, params.Path)
	if err != nil {
		return "", err
	}

	// 读取文件
	content, err := os.ReadFile(safePath)
	if err != nil {
		return "", errors.Wrap(errors.CodeToolError, "failed to read file", err)
	}

	text := string(content)

	// 处理行数限制
	if params.Limit != nil {
		lines := strings.Split(text, "\n")
		if *params.Limit < len(lines) {
			lines = lines[:*params.Limit]
			text = strings.Join(lines, "\n") + "\n... (" + string(rune(len(lines)-*params.Limit)) + " more lines)"
		}
	}

	// 截断输出
	if len(text) > 50000 {
		text = text[:50000] + "\n... (output truncated)"
	}

	return text, nil
}

// ToToolCall 转换为工具调用
func (t *ReadFileTool) ToToolCall(path string, limit *int) entity.ToolCall {
	args := map[string]interface{}{"path": path}
	if limit != nil {
		args["limit"] = *limit
	}
	argsJSON, _ := json.Marshal(args)
	return entity.NewToolCall("read_file", string(argsJSON))
}
