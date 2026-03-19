package tool

import (
	"path/filepath"

	"ai_code/internal/domain/errors"
)

// SafePath 检查路径是否在工作目录内，防止路径逃逸
func SafePath(workDir, path string) (string, error) {
	// 解析绝对路径
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(workDir, path)
	}

	// 清理路径（处理 .. 等）
	absPath = filepath.Clean(absPath)

	// 解析工作目录
	absWorkDir := filepath.Clean(workDir)

	// 检查是否在工作目录内
	relPath, err := filepath.Rel(absWorkDir, absPath)
	if err != nil {
		return "", errors.New(errors.CodeInvalidInput, "invalid path")
	}

	// 如果相对路径以 .. 开头，说明逃逸了工作目录
	if len(relPath) >= 2 && relPath[:2] == ".." {
		return "", errors.New(errors.CodeInvalidInput, "path escapes workspace: "+path)
	}

	return absPath, nil
}
