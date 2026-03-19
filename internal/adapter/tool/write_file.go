package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"ai_code/internal/domain/entity"
	"ai_code/internal/domain/errors"
	"ai_code/pkg/logger"
)

// WriteFileTool 文件写入工具
type WriteFileTool struct {
	cwd    string
	logger logger.Logger
}

// WriteFileToolOption 写入工具选项
type WriteFileToolOption func(*WriteFileTool)

// NewWriteFileTool 创建文件写入工具
func NewWriteFileTool(opts ...WriteFileToolOption) *WriteFileTool {
	cwd, _ := os.Getwd()

	t := &WriteFileTool{
		cwd:    cwd,
		logger: logger.Default().WithPrefix("write_file"),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// WithWriteFileCWD 设置工作目录
func WithWriteFileCWD(cwd string) WriteFileToolOption {
	return func(t *WriteFileTool) {
		t.cwd = cwd
	}
}

// Name 工具名称
func (t *WriteFileTool) Name() string {
	return "write_file"
}

// Description 工具描述
func (t *WriteFileTool) Description() string {
	return "Write content to file. Creates the file if it doesn't exist, overwrites if it does."
}

// Parameters 参数定义
func (t *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The path to the file to write",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

// Execute 执行写入
func (t *WriteFileTool) Execute(ctx context.Context, args string) (string, error) {
	// 解析参数
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", errors.Wrap(errors.CodeInvalidInput, "failed to parse arguments", err)
	}

	// 安全路径检查
	safePath, err := SafePath(t.cwd, params.Path)
	if err != nil {
		return "", err
	}

	// 创建父目录（如果不存在）
	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", errors.Wrap(errors.CodeToolError, "failed to create parent directory", err)
	}

	// 写入文件
	if err := os.WriteFile(safePath, []byte(params.Content), 0644); err != nil {
		return "", errors.Wrap(errors.CodeToolError, "failed to write file", err)
	}

	t.logger.Debug("file written", logger.F("path", safePath), logger.F("size", len(params.Content)))

	return fmt.Sprintf("Wrote %d bytes to %s", len(params.Content), params.Path), nil
}

// ToToolCall 转换为工具调用
func (t *WriteFileTool) ToToolCall(path, content string) entity.ToolCall {
	args := map[string]interface{}{
		"path":    path,
		"content": content,
	}
	argsJSON, _ := json.Marshal(args)
	return entity.NewToolCall("write_file", string(argsJSON))
}
