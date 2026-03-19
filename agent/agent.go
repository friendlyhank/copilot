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

// ToolHandler 工具处理函数类型
type ToolHandler func(args map[string]interface{}) string

// Agent AI代理
// 核心洞察: "加工具不需要改循环"
// 只需在 TOOL_HANDLERS 中添加新的 handler，循环本身不变
type Agent struct {
	Client   llm.Client
	Messages []llm.Message
	Tools    []llm.Tool
	System   string
	Debug    bool
	Handler  OutputHandler

	// 工具调度表 - dispatch map
	toolHandlers map[string]ToolHandler
	cwd          string
}

// NewAgent 创建新Agent
func NewAgent(client llm.Client, systemPrompt string) *Agent {
	cwd, _ := os.Getwd()

	a := &Agent{
		Client: client,
		System: systemPrompt,
		cwd:    cwd,
	}

	// 初始化工具
	a.initTools()

	return a
}

// initTools 初始化工具 - 在这里添加新工具
func (a *Agent) initTools() {
	// 工具定义
	a.Tools = []llm.Tool{
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
		{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        "read_file",
				Description: "Read file contents. Returns the text content of a file.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "The path to the file to read",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Optional: maximum number of lines to read",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        "write_file",
				Description: "Write content to file. Creates the file if it doesn't exist, overwrites if it does.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "The path to the file to write",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "The content to write to the file",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        "edit_file",
				Description: "Replace exact text in file. Finds and replaces the first occurrence of old_text with new_text.",
				Parameters: map[string]interface{}{
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
				},
			},
		},
	}

	// 工具处理函数调度表 - dispatch map
	// 添加新工具 = 在这里添加 handler
	a.toolHandlers = map[string]ToolHandler{
		"bash":       a.runBash,
		"read_file":  a.runReadFile,
		"write_file": a.runWriteFile,
		"edit_file":  a.runEditFile,
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

// safePath 检查路径是否在工作目录内
func (a *Agent) safePath(path string) (string, error) {
	absPath := path
	if !strings.HasPrefix(path, "/") {
		absPath = a.cwd + "/" + path
	}

	// 清理路径
	absPath = strings.ReplaceAll(absPath, "//", "/")

	// 简单检查路径逃逸
	if strings.Contains(absPath, "..") {
		// 检查是否逃出工作目录
		cleanPath := absPath
		for strings.Contains(cleanPath, "..") {
			idx := strings.Index(cleanPath, "..")
			if idx >= 2 {
				// 找到前一个 /
				prevSlash := strings.LastIndex(cleanPath[:idx], "/")
				if prevSlash >= 0 {
					cleanPath = cleanPath[:prevSlash] + cleanPath[idx+2:]
				}
			}
		}
		if !strings.HasPrefix(cleanPath, a.cwd) {
			return "", fmt.Errorf("path escapes workspace: %s", path)
		}
		absPath = cleanPath
	}

	return absPath, nil
}

// runBash 执行bash命令
func (a *Agent) runBash(args map[string]interface{}) string {
	command, ok := args["command"].(string)
	if !ok {
		return "Error: missing command argument"
	}

	// 危险命令检查
	dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"}
	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return "Error: Dangerous command blocked"
		}
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = a.cwd

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	timer := time.AfterFunc(120*time.Second, func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	})
	defer timer.Stop()

	err := cmd.Run()
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

// runReadFile 读取文件
func (a *Agent) runReadFile(args map[string]interface{}) string {
	path, ok := args["path"].(string)
	if !ok {
		return "Error: missing path argument"
	}

	safePath, err := a.safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	text := string(content)

	// 处理行数限制
	if limit, ok := args["limit"].(float64); ok {
		lines := strings.Split(text, "\n")
		if int(limit) < len(lines) {
			lines = lines[:int(limit)]
			text = strings.Join(lines, "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-int(limit))
		}
	}

	if len(text) > 50000 {
		text = text[:50000]
	}

	return text
}

// runWriteFile 写入文件
func (a *Agent) runWriteFile(args map[string]interface{}) string {
	path, ok := args["path"].(string)
	if !ok {
		return "Error: missing path argument"
	}
	content, ok := args["content"].(string)
	if !ok {
		return "Error: missing content argument"
	}

	safePath, err := a.safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	// 创建父目录
	dir := safePath[:strings.LastIndex(safePath, "/")]
	if dir != "" {
		os.MkdirAll(dir, 0755)
	}

	if err := os.WriteFile(safePath, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return fmt.Sprintf("Wrote %d bytes to %s", len(content), path)
}

// runEditFile 编辑文件
func (a *Agent) runEditFile(args map[string]interface{}) string {
	path, ok := args["path"].(string)
	if !ok {
		return "Error: missing path argument"
	}
	oldText, ok := args["old_text"].(string)
	if !ok {
		return "Error: missing old_text argument"
	}
	newText, ok := args["new_text"].(string)
	if !ok {
		return "Error: missing new_text argument"
	}

	safePath, err := a.safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, oldText) {
		return fmt.Sprintf("Error: Text not found in %s", path)
	}

	// 只替换第一个匹配
	newContent := strings.Replace(text, oldText, newText, 1)

	if err := os.WriteFile(safePath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return fmt.Sprintf("Edited %s", path)
}

// Loop 执行Agent循环
// 核心洞察: 循环不变，工具通过 dispatch map 调度
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

		// 调试：打印 finish_reason 和 tool_calls 数量
		if a.Debug {
			fmt.Printf("\n[DEBUG] finish_reason: %s, tool_calls: %d\n", choice.FinishReason, len(choice.Message.ToolCalls))
		}

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

		// 处理工具调用 - 通过 dispatch map 调度
		for _, toolCall := range choice.Message.ToolCalls {
			// 解析参数
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				a.emit(OutputError, fmt.Sprintf("Error parsing tool arguments: %v", err))
				continue
			}

			// 显示命令
			displayArgs := toolCall.Function.Arguments
			if len(displayArgs) > 100 {
				displayArgs = displayArgs[:100] + "..."
			}
			a.emit(OutputCommand, fmt.Sprintf("%s(%s)", toolCall.Function.Name, displayArgs))

			// 通过 dispatch map 查找处理函数
			handler, exists := a.toolHandlers[toolCall.Function.Name]
			var output string
			if !exists {
				output = fmt.Sprintf("Unknown tool: %s", toolCall.Function.Name)
			} else {
				output = handler(args)
			}

			// 截断输出用于显示
			displayOutput := output
			if len(displayOutput) > 1000 {
				displayOutput = displayOutput[:1000] + "..."
			}
			a.emit(OutputResult, displayOutput)

			// 添加工具结果到消息历史
			a.Messages = append(a.Messages, llm.Message{
				Role:       "tool",
				Content:    output,
				ToolCallID: toolCall.ID,
			})
		}
	}
}