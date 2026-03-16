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
	// OpenAI默认API地址
	OpenAIDefaultBaseURL = "https://api.openai.com/v1/chat/completions"
)

func init() {
	// 注册OpenAI厂商
	RegisterProvider("openai", NewOpenAIClient)
}

// OpenAIClient OpenAI兼容客户端实现
// 可用于OpenAI、Azure OpenAI及其他兼容接口
type OpenAIClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
	debug   bool
}

// NewOpenAIClient 创建OpenAI客户端
func NewOpenAIClient(config ProviderConfig) (Client, error) {
	if config.APIKey == "" {
		return nil, ErrInvalidConfig
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = OpenAIDefaultBaseURL
	}

	model := config.Model
	if model == "" {
		model = "gpt-4"
	}

	return &OpenAIClient{
		apiKey:  config.APIKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 180 * time.Second},
	}, nil
}

// Chat 发送聊天请求
func (c *OpenAIClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

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
func (c *OpenAIClient) GetName() string {
	return "openai"
}

// GetModel 获取当前模型
func (c *OpenAIClient) GetModel() string {
	return c.model
}

// SetModel 设置模型
func (c *OpenAIClient) SetModel(model string) {
	c.model = model
}

// SetDebug 设置调试模式
func (c *OpenAIClient) SetDebug(debug bool) {
	c.debug = debug
}
