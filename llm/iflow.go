package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// iFlow默认API地址
	IflowDefaultBaseURL = "https://apis.iflow.cn/v1/chat/completions"
)

func init() {
	// 注册iFlow厂商
	RegisterProvider("iflow", NewIflowClient)
}

// IflowClient iFlow客户端实现
type IflowClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
	debug   bool
}

// IflowOption iFlow客户端选项
type IflowOption func(*IflowClient)

// WithDebug 设置调试模式
func WithDebug(debug bool) IflowOption {
	return func(c *IflowClient) {
		c.debug = debug
	}
}

// WithHTTPClient 设置HTTP客户端
func WithHTTPClient(httpClient *http.Client) IflowOption {
	return func(c *IflowClient) {
		c.client = httpClient
	}
}

// NewIflowClient 创建iFlow客户端
func NewIflowClient(config ProviderConfig) (Client, error) {
	if config.APIKey == "" {
		return nil, ErrInvalidConfig
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = IflowDefaultBaseURL
	}

	model := config.Model
	if model == "" {
		model = "kimi-k2-0905"
	}

	return &IflowClient{
		apiKey:  config.APIKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 180 * time.Second},
	}, nil
}

// Chat 发送聊天请求
func (c *IflowClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 确保模型名称设置
	if req.Model == "" {
		req.Model = c.model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 调试模式打印请求
	if c.debug {
		fmt.Printf("\n\033[34m========== Request ==========\033[0m\n")
		var prettyReq bytes.Buffer
		json.Indent(&prettyReq, body, "", "  ")
		fmt.Printf("\033[34m%s\033[0m\n", prettyReq.String())
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		fmt.Printf("\n\033[32m========== Response ==========\033[0m\n")
		var prettyResp bytes.Buffer
		json.Indent(&prettyResp, respBody, "", "  ")
		fmt.Printf("\033[32m%s\033[0m\n", prettyResp.String())
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w (status %d): %s", ErrAPIError, resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &chatResp, nil
}

// GetName 获取厂商名称
func (c *IflowClient) GetName() string {
	return "iflow"
}

// GetModel 获取当前模型
func (c *IflowClient) GetModel() string {
	return c.model
}

// SetModel 设置模型
func (c *IflowClient) SetModel(model string) {
	c.model = model
}

// SetDebug 设置调试模式
func (c *IflowClient) SetDebug(debug bool) {
	c.debug = debug
}
