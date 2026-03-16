package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	}

	return map[string]interface{}{
		"model":       req.Model,
		"messages":    messages,
		"stream":      req.Stream,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}
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
	return &resp, nil
}

// printDebug 打印调试信息
func (c *BaseClient) printDebug(title string, body []byte, color string) {
	fmt.Printf("\n%s========== %s ==========%s\n", color, title, "\033[0m")
	var pretty bytes.Buffer
	json.Indent(&pretty, body, "", "  ")
	fmt.Printf("%s%s%s\n", color, pretty.String(), "\033[0m")
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
