package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"ai_code/internal/domain/errors"
	"ai_code/internal/port"
	"ai_code/pkg/logger"
)

// BaseClient 基础客户端，提取共享实现
type BaseClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
	debug   bool
	logger  logger.Logger
	name    string
}

// BaseClientOption 基础客户端选项
type BaseClientOption func(*BaseClient)

// NewBaseClient 创建基础客户端
func NewBaseClient(name string, config port.ProviderConfig, opts ...BaseClientOption) *BaseClient {
	c := &BaseClient{
		name:   name,
		apiKey: config.APIKey,
		model:  config.Model,
		logger: logger.Default().WithPrefix(name),
	}

	// 设置 baseURL
	c.baseURL = config.BaseURL

	// 设置超时
	timeout := 180 * time.Second
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}
	c.client = &http.Client{Timeout: timeout}

	// 应用选项
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithDebug 设置调试模式
func WithDebug(debug bool) BaseClientOption {
	return func(c *BaseClient) {
		c.debug = debug
	}
}

// WithLogger 设置日志
func WithLogger(l logger.Logger) BaseClientOption {
	return func(c *BaseClient) {
		c.logger = l
	}
}

// Chat 发送聊天请求 - 共享实现
func (c *BaseClient) Chat(ctx context.Context, req *port.ChatRequest) (*port.ChatResponse, error) {
	// 确保模型名称设置
	if req.Model == "" {
		req.Model = c.model
	}

	// 转换消息格式
	llmReq := c.buildLLMRequest(req)

	body, err := json.Marshal(llmReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 调试模式打印请求
	if c.debug {
		c.printDebug("Request", body, "\033[34m")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 调试模式打印响应
	if c.debug {
		c.printDebug("Response", respBody, "\033[32m")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.APIError(
			"API request failed",
			resp.StatusCode,
			fmt.Errorf("%s", string(respBody)),
		)
	}

	return c.parseResponse(respBody)
}

// buildLLMRequest 构建 LLM 请求格式（可被子类覆盖）
func (c *BaseClient) buildLLMRequest(req *port.ChatRequest) interface{} {
	// 转换消息格式
	messages := make([]map[string]interface{}, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]interface{}{
			"role":    string(msg.Role),
			"content": msg.Content,
		}
		if len(msg.ToolCalls) > 0 {
			messages[i]["tool_calls"] = msg.ToolCalls
		}
		if msg.ToolCallID != "" {
			messages[i]["tool_call_id"] = msg.ToolCallID
		}
	}

	// 构建请求
	result := map[string]interface{}{
		"model":       req.Model,
		"messages":    messages,
		"stream":      req.Stream,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}

	// 添加工具定义
	if len(req.Tools) > 0 {
		result["tools"] = req.Tools
	}

	return result
}

// setHeaders 设置请求头（可被子类覆盖）
func (c *BaseClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
}

// parseResponse 解析响应（可被子类覆盖）
func (c *BaseClient) parseResponse(body []byte) (*port.ChatResponse, error) {
	var resp port.ChatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// iFlow 兼容：将根级别的 ToolCalls 合并到 message 中
	if len(resp.ToolCalls) > 0 && len(resp.Choices) > 0 {
		resp.Choices[0].Message.ToolCalls = resp.ToolCalls
	}

	return &resp, nil
}

// printDebug 打印调试信息到文件（TUI 模式下终端输出会被覆盖）
func (c *BaseClient) printDebug(title string, body []byte, color string) {
	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	// 创建 log 目录
	logDir := cwd + "/log"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return
	}

	// 写入调试日志文件
	f, err := os.OpenFile(logDir+"/copilot.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	var pretty bytes.Buffer
	json.Indent(&pretty, body, "", "  ")
	f.WriteString(fmt.Sprintf("\n========== %s ==========\n", title))
	f.WriteString(pretty.String() + "\n")
}

// GetName 获取提供商名称
func (c *BaseClient) GetName() string {
	return c.name
}

// GetModel 获取当前模型
func (c *BaseClient) GetModel() string {
	return c.model
}

// SetModel 设置模型
func (c *BaseClient) SetModel(model string) {
	c.model = model
}

// SetDebug 设置调试模式
func (c *BaseClient) SetDebug(debug bool) {
	c.debug = debug
}

// ChatStream 发送流式聊天请求
func (c *BaseClient) ChatStream(ctx context.Context, req *port.ChatRequest, handler port.StreamHandler) error {
	// 确保模型名称设置
	if req.Model == "" {
		req.Model = c.model
	}

	// 强制设置为流式
	req.Stream = true

	// 转换消息格式
	llmReq := c.buildLLMRequest(req)

	body, err := json.Marshal(llmReq)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// 调试模式打印请求
	if c.debug {
		c.printDebug("Stream Request", body, "\033[34m")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)
	// 流式请求需要接受 text/event-stream
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return errors.APIError(
			"API request failed",
			resp.StatusCode,
			fmt.Errorf("%s", string(respBody)),
		)
	}

	// 解析 SSE 流
	return c.parseStreamResponse(resp.Body, handler)
}

// parseStreamResponse 解析 SSE 流式响应
func (c *BaseClient) parseStreamResponse(reader io.Reader, handler port.StreamHandler) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行
		if line == "" {
			continue
		}

		// SSE 格式: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 流结束标记
		if data == "[DONE]" {
			return nil
		}

		// 解析 chunk
		var chunk port.StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			c.logger.Warn("failed to parse stream chunk", logger.F("data", data))
			continue
		}

		// 调用 handler 处理 chunk
		if err := handler(&chunk); err != nil {
			return fmt.Errorf("stream handler error: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}

	return nil
}
