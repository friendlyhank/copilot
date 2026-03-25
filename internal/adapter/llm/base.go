package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	// 记录请求
	c.logDebug("Request", body)

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

	// 记录响应
	c.logDebug("Response", respBody)

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

// logDebug 使用 logger 输出调试信息（日志等级控制是否输出）
func (c *BaseClient) logDebug(title string, body []byte) {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, body, "", "  "); err != nil {
		c.logger.Debug(title, logger.F("raw", string(body)))
		return
	}
	c.logger.Debug(title, logger.F("body", pretty.String()))
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

	// 记录请求
	c.logDebug("Stream Request", body)

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
	collector := newStreamResponseCollector()

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行
		if line == "" {
			continue
		}

		// 记录原始行内容
		c.logger.Debug("SSE line", logger.F("content", line))

		// SSE 格式: "data: {...}" 或 "data:{...}" (兼容有无空格)
		var data string
		if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		} else if strings.HasPrefix(line, "data:") {
			data = strings.TrimPrefix(line, "data:")
		} else {
			continue
		}

		// 流结束标记
		if data == "[DONE]" {
			break
		}

		// 解析 chunk
		var chunk port.StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			c.logger.Warn("failed to parse stream chunk", logger.F("data", data))
			continue
		}

		// 收集响应
		collector.Collect(&chunk)

		// 调用 handler 处理 chunk
		if err := handler(&chunk); err != nil {
			return fmt.Errorf("stream handler error: %w", err)
		}
	}

	// 记录完整响应日志
	if collector.HasContent() {
		if respBody, err := json.Marshal(collector.Build()); err == nil {
			c.logDebug("Stream Response", respBody)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}

	return nil
}

// streamResponseCollector 流式响应收集器
// 用于收集流式响应的各个 chunk，拼接成完整响应
type streamResponseCollector struct {
	id            string
	model         string
	content       strings.Builder
	toolCalls     []map[string]any
	finishReason  string
	chunkCount    int
}

// newStreamResponseCollector 创建收集器
func newStreamResponseCollector() *streamResponseCollector {
	return &streamResponseCollector{}
}

// Collect 收集单个 chunk
func (c *streamResponseCollector) Collect(chunk *port.StreamChunk) {
	c.chunkCount++

	// 收集基本信息
	if chunk.ID != "" {
		c.id = chunk.ID
	}
	if chunk.Model != "" {
		c.model = chunk.Model
	}

	// 处理 choices
	if len(chunk.Choices) == 0 {
		return
	}

	choice := chunk.Choices[0]

	// 拼接内容
	if choice.Delta.Content != "" {
		c.content.WriteString(choice.Delta.Content)
	}

	// 收集工具调用
	for _, tc := range choice.Delta.ToolCalls {
		c.collectToolCall(tc)
	}

	// 记录结束原因
	if choice.FinishReason != "" {
		c.finishReason = choice.FinishReason
	}
}

// collectToolCall 收集工具调用
func (c *streamResponseCollector) collectToolCall(tc port.StreamToolCall) {
	idx := tc.Index

	// 扩展切片
	for len(c.toolCalls) <= idx {
		c.toolCalls = append(c.toolCalls, map[string]any{
			"id":       "",
			"type":     "function",
			"function": map[string]any{"name": "", "arguments": ""},
		})
	}

	if tc.ID != "" {
		c.toolCalls[idx]["id"] = tc.ID
	}
	if tc.Type != "" {
		c.toolCalls[idx]["type"] = tc.Type
	}

	fnMap := c.toolCalls[idx]["function"].(map[string]any)
	if tc.Function.Name != "" {
		fnMap["name"] = tc.Function.Name
	}
	if tc.Function.Arguments != "" {
		args := fnMap["arguments"].(string)
		fnMap["arguments"] = args + tc.Function.Arguments
	}
}

// Build 构建完整响应
func (c *streamResponseCollector) Build() map[string]any {
	message := map[string]any{
		"role":    "assistant",
		"content": c.content.String(),
	}

	if len(c.toolCalls) > 0 {
		message["tool_calls"] = c.toolCalls
	}

	return map[string]any{
		"id":    c.id,
		"model": c.model,
		"choices": []map[string]any{
			{
				"index":          0,
				"message":        message,
				"finish_reason":  c.finishReason,
			},
		},
	}
}

// HasContent 是否有内容
func (c *streamResponseCollector) HasContent() bool {
	return c.chunkCount > 0
}