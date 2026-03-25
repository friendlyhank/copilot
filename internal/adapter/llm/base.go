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
	var responseChunks []map[string]any // 收集解析后的 chunk 用于日志

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
			break
		}

		// 解析 chunk
		var chunk port.StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			c.logger.Warn("failed to parse stream chunk", logger.F("data", data))
			continue
		}

		// 收集用于日志
		responseChunks = append(responseChunks, map[string]any{
			"id":        chunk.ID,
			"model":     chunk.Model,
			"choices":   chunk.Choices,
			"tool_calls": chunk.ToolCalls,
		})

		// 调用 handler 处理 chunk
		if err := handler(&chunk); err != nil {
			return fmt.Errorf("stream handler error: %w", err)
		}
	}

	// 记录流式响应摘要
	if len(responseChunks) > 0 {
		// 提取最终内容摘要
		var contentBuilder strings.Builder
		var toolCalls []port.StreamToolCall
		for _, ch := range responseChunks {
			if choices, ok := ch["choices"].([]port.StreamChoice); len(choices) > 0 && ok {
				if choices[0].Delta.Content != "" {
					contentBuilder.WriteString(choices[0].Delta.Content)
				}
				if len(choices[0].Delta.ToolCalls) > 0 {
					toolCalls = append(toolCalls, choices[0].Delta.ToolCalls...)
				}
			}
		}
		c.logger.Debug("Stream Response",
			logger.F("chunks", len(responseChunks)),
			logger.F("content_len", contentBuilder.Len()),
			logger.F("tool_calls", len(toolCalls)),
		)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}

	return nil
}