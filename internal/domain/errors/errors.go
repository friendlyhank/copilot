package errors

import (
	"fmt"
)

// Code 错误码
type Code string

const (
	// 配置相关
	CodeInvalidConfig  Code = "INVALID_CONFIG"
	CodeConfigNotFound Code = "CONFIG_NOT_FOUND"

	// LLM 相关
	CodeProviderNotFound Code = "PROVIDER_NOT_FOUND"
	CodeAPIError         Code = "API_ERROR"
	CodeTimeout          Code = "TIMEOUT"
	CodeRateLimited      Code = "RATE_LIMITED"
	CodeInvalidResponse  Code = "INVALID_RESPONSE"

	// 工具相关
	CodeToolNotFound  Code = "TOOL_NOT_FOUND"
	CodeToolError     Code = "TOOL_ERROR"
	CodeInvalidInput  Code = "INVALID_INPUT"

	// Agent 相关
	CodeAgentError     Code = "AGENT_ERROR"
	CodeContextCanceled Code = "CONTEXT_CANCELED"

	// 通用
	CodeUnknown Code = "UNKNOWN"
)

// Error 领域错误
type Error struct {
	Code    Code
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error 实现 error 接口
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现错误解包
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithContext 添加上下文
func (e *Error) WithContext(key string, value interface{}) *Error {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// New 创建新错误
func New(code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误
func Wrap(code Code, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// 便捷构造函数

func InvalidConfig(message string) *Error {
	return New(CodeInvalidConfig, message)
}

func ProviderNotFound(provider string) *Error {
	return New(CodeProviderNotFound, fmt.Sprintf("provider '%s' not found", provider))
}

func APIError(message string, statusCode int, cause error) *Error {
	return &Error{
		Code:    CodeAPIError,
		Message: message,
		Cause:   cause,
		Context: map[string]interface{}{"status_code": statusCode},
	}
}

func Timeout(message string) *Error {
	return New(CodeTimeout, message)
}

func ToolError(toolName, message string, cause error) *Error {
	return &Error{
		Code:    CodeToolError,
		Message: fmt.Sprintf("tool '%s' error: %s", toolName, message),
		Cause:   cause,
	}
}

func IsCode(err error, code Code) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == code
	}
	return false
}
