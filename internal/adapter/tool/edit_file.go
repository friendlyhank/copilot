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

// EditFileTool 文件编辑工具
type EditFileTool struct {
	cwd    string
	logger logger.Logger
}

// EditFileToolOption 编辑工具选项
type EditFileToolOption func(*EditFileTool)

// NewEditFileTool 创建文件编辑工具
func NewEditFileTool(opts ...EditFileToolOption) *EditFileTool {
	cwd, _ := os.Getwd()

	t := &EditFileTool{
		cwd:    cwd,
		logger: logger.Default().WithPrefix("edit_file"),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// WithEditFileCWD 设置工作目录
func WithEditFileCWD(cwd string) EditFileToolOption {
	return func(t *EditFileTool) {
		t.cwd = cwd
	}
}

// Name 工具名称
func (t *EditFileTool) Name() string {
	return "edit_file"
}

// Description 工具描述
func (t *EditFileTool) Description() string {
	return "Replace exact text in file. Finds and replaces the first occurrence of old_text with new_text."
}

// Parameters 参数定义
func (t *EditFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The path to the file to edit",
			},
			"old_text": map[string]interface{}{
				"type":        "string",
				"description": "The exact text to find and replace",
			},
			"new_text": map[string]interface{}{
				"type":        "string",
				"description": "The text to replace old_text with",
			},
		},
		"required": []string{"path", "old_text", "new_text"},
	}
}

// Execute 执行编辑
func (t *EditFileTool) Execute(ctx context.Context, args string) (string, error) {
	// 解析参数
	var params struct {
		Path     string `json:"path"`
		OldText  string `json:"old_text"`
		NewText  string `json:"new_text"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", errors.Wrap(errors.CodeInvalidInput, "failed to parse arguments", err)
	}

	// 安全路径检查
	safePath, err := SafePath(t.cwd, params.Path)
	if err != nil {
		return "", err
	}

	// 读取文件内容
	content, err := os.ReadFile(safePath)
	if err != nil {
		return "", errors.Wrap(errors.CodeToolError, "failed to read file", err)
	}

	text := string(content)

	// 检查旧文本是否存在
	if !strings.Contains(text, params.OldText) {
		return "", errors.New(errors.CodeToolError, "text not found in "+params.Path)
	}

	// 替换文本（只替换第一个匹配）
	newText := strings.Replace(text, params.OldText, params.NewText, 1)

	// 写入文件
	if err := os.WriteFile(safePath, []byte(newText), 0644); err != nil {
		return "", errors.Wrap(errors.CodeToolError, "failed to write file", err)
	}

	t.logger.Debug("file edited", logger.F("path", safePath))

	return "Edited " + params.Path, nil
}

// ToToolCall 转换为工具调用
func (t *EditFileTool) ToToolCall(path, oldText, newText string) entity.ToolCall {
	args := map[string]interface{}{
		"path":     path,
		"old_text": oldText,
		"new_text": newText,
	}
	argsJSON, _ := json.Marshal(args)
	return entity.NewToolCall("edit_file", string(argsJSON))
}
