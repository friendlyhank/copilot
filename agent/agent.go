package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"ai_code/llm"
)

// OutputType 输出类型
type OutputType int

const (
	OutputText OutputType = iota
	OutputCommand
	OutputResult
	OutputError
	OutputDone
)

// Output 输出消息
type Output struct {
	Type    OutputType
	Content string
}

// OutputHandler 输出处理函数
type OutputHandler func(Output)

// Agent AI代理
type Agent struct {
	Client   llm.Client
	Messages []llm.Message
	Tools    []llm.Tool
	System   string
	Debug    bool
	Handler  OutputHandler
}

// NewAgent 创建新Agent
func NewAgent(client llm.Client, systemPrompt string) *Agent {
	tools := []llm.Tool{
		{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        "bash",
				Description: "Run a shell command. Execute a bash command and return the output.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "The bash command to execute",
						},
					},
					"required": []string{"command"},
				},
			},
		},
	}

	return &Agent{
		Client: client,
		Tools:  tools,
		System: systemPrompt,
	}
}

// SetDebug 设置调试模式
func (a *Agent) SetDebug(debug bool) {
	a.Debug = debug
	// 如果客户端支持设置调试模式
	if setter, ok := a.Client.(interface{ SetDebug(bool) }); ok {
		setter.SetDebug(debug)
	}
}

// SetOutputHandler 设置输出处理函数
func (a *Agent) SetOutputHandler(handler OutputHandler) {
	a.Handler = handler
}

// emit 发送输出
func (a *Agent) emit(outputType OutputType, content string) {
	if a.Handler != nil {
		a.Handler(Output{Type: outputType, Content: content})
	}
}

// buildMessages 构建完整的消息列表
func (a *Agent) buildMessages() []llm.Message {
	messages := make([]llm.Message, 0)

	if a.System != "" {
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: a.System,
		})
	}

	messages = append(messages, a.Messages...)

	return messages
}

// callAPI 调用LLM API
func (a *Agent) callAPI(ctx context.Context) (*llm.ChatResponse, error) {
	req := &llm.ChatRequest{
		Model:       a.Client.GetModel(),
		Messages:    a.buildMessages(),
		Stream:      false,
		MaxTokens:   8000,
		Temperature: 0.7,
		Tools:       a.Tools,
	}

	return a.Client.Chat(ctx, req)
}

// runBash 执行bash命令
func runBash(command string) string {
	// 危险命令检查
	dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"}
	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return "Error: Dangerous command blocked"
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = cwd

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	timer := time.AfterFunc(120*time.Second, func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	})
	defer timer.Stop()

	err = cmd.Run()
	output := stdout.String() + stderr.String()
	output = strings.TrimSpace(output)

	if err != nil {
		if strings.Contains(err.Error(), "signal: killed") {
			return "Error: Timeout (120s)"
		}
	}

	if output == "" {
		return "(no output)"
	}

	if len(output) > 50000 {
		output = output[:50000]
	}

	return output
}

// Loop 执行Agent循环
func (a *Agent) Loop(ctx context.Context) error {
	defer a.emit(OutputDone, "")

	for {
		resp, err := a.callAPI(ctx)
		if err != nil {
			a.emit(OutputError, fmt.Sprintf("API call failed: %v", err))
			return fmt.Errorf("API call failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			a.emit(OutputError, "no choices in response")
			return fmt.Errorf("no choices in response")
		}

		choice := resp.Choices[0]

		// 打印assistant的文本内容
		if choice.Message.Content != "" {
			a.emit(OutputText, choice.Message.Content)
		}

		// 如果没有工具调用，循环结束
		if len(choice.Message.ToolCalls) == 0 {
			return nil
		}

		// 添加assistant消息到历史
		assistantMsg := llm.Message{
			Role:    "assistant",
			Content: choice.Message.Content,
		}
		a.Messages = append(a.Messages, assistantMsg)

		// 处理工具调用
		for _, toolCall := range choice.Message.ToolCalls {
			if toolCall.Function.Name == "bash" {
				var args struct {
					Command string `json:"command"`
				}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					a.emit(OutputError, fmt.Sprintf("Error parsing tool arguments: %v", err))
					continue
				}

				a.emit(OutputCommand, args.Command)

				output := runBash(args.Command)

				// 截断输出用于显示
				displayOutput := output
				if len(displayOutput) > 1000 {
					displayOutput = displayOutput[:1000] + "..."
				}
				a.emit(OutputResult, displayOutput)

				a.Messages = append(a.Messages, llm.Message{
					Role:       "tool",
					Content:    output,
					ToolCallID: toolCall.ID,
				})
			}
		}
	}
}