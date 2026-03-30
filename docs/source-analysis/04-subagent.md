# 子智能体架构设计

> **项目**: ai_code (copilot)  
> **分析日期**: 2026-03-30

---

## 一、概念概述

### 1.1 什么是子智能体

子智能体（SubAgent）是一个拥有**独立上下文**的任务执行单元，通过 `task` 工具启动。

```mermaid
graph TB
    subgraph "父 Agent"
        PA_Messages["messages = [<br/>  {user: '之前的问题'},<br/>  {assistant: '之前的回答'},<br/>  {user: '调查测试框架'},<br/>  {assistant: '...', tool_calls: [task]},<br/>  {tool: '摘要: 使用 Go testing'}<br/>]"]
    end
    
    subgraph "子 Agent"
        SA_Messages["messages = [<br/>  {user: '调查测试框架'}<br/>]"]
        SA_Tools["tool_calls: [<br/>  read_file('go.mod'),<br/>  read_file('main_test.go')<br/>]"]
    end
    
    PA_Messages -->|"task 工具"| SA_Messages
    SA_Messages --> SA_Tools
    SA_Tools -->|"返回摘要"| PA_Messages
    
    style PA_Messages fill:#e3f2fd
    style SA_Messages fill:#c8e6c9
```

### 1.2 为什么需要子智能体

**问题：上下文膨胀**

```mermaid
graph LR
    subgraph "无子 Agent"
        A1[用户: 调查测试框架]
        A2[AI: 读取 go.mod]
        A3[tool: go.mod 内容 500 行]
        A4[AI: 读取 main_test.go]
        A5[tool: 测试文件 300 行]
        A6[AI: 使用 Go testing]
        
        A1 --> A2 --> A3 --> A4 --> A5 --> A6
    end
    
    style A3 fill:#ffcdd2
    style A5 fill:#ffcdd2
```

**解决方案：上下文隔离**

```mermaid
graph LR
    subgraph "有子 Agent"
        B1[用户: 调查测试框架]
        B2[AI: task 调查...]
        B3[tool: 使用 Go testing]
        B4[AI: 使用 Go testing]
        
        B1 --> B2 --> B3 --> B4
    end
    
    subgraph "子 Agent 内部"
        C1[读取 go.mod 500 行]
        C2[读取 main_test.go 300 行]
        C1 --> C2
    end
    
    B2 -.-> C1
    C2 -.-> B3
    
    style B3 fill:#c8e6c9
    style C1 fill:#fff3e0
    style C2 fill:#fff3e0
```

### 1.3 核心优势

| 优势 | 说明 |
|------|------|
| **上下文隔离** | 子 Agent 的工具调用对父 Agent 不可见 |
| **Token 节约** | 中间步骤不污染父上下文 |
| **任务分解** | 复杂任务可拆分为独立子任务 |
| **防止递归** | 子 Agent 不能再创建子 Agent |

---

## 二、架构设计

### 2.1 组件关系

```mermaid
classDiagram
    class SubAgentRunner {
        <<interface>>
        +Run(ctx context.Context, prompt string) string, error
    }
    
    class SubAgentConfig {
        +MaxIterations int
        +MaxTokens int
        +SystemPrompt string
    }
    
    class TaskTool {
        -llmClient LLMClient
        -toolReg ToolRegistry
        -cwd string
        +Name() string
        +Description() string
        +Parameters() map
        +Execute(ctx, args) string, error
    }
    
    class Agent {
        -llmClient LLMClient
        -toolReg ToolRegistry
        -session Session
        -isSubAgent bool
        -excludeTools []string
        -subAgentConfig SubAgentConfig
        +NewSubAgent() Agent
        +Run(ctx, prompt) string, error
        +ProcessMessage(ctx, input) error
        -getTools() []ToolDefinition
        -executeToolSilent(ctx, call) ToolResult
    }
    
    class Session {
        +ID string
        +Messages []Message
        +AddMessage(msg)
    }
    
    SubAgentRunner <|.. Agent : implements
    TaskTool --> Agent : creates SubAgent
    Agent --> Session : has (独立)
    Agent --> SubAgentConfig : uses
```

### 2.2 接口定义

**文件路径**: `internal/port/subagent.go`

```go
// SubAgentRunner 子智能体运行器接口
type SubAgentRunner interface {
    // Run 执行子智能体任务
    // prompt: 子任务描述
    // 返回: 任务执行摘要（不包含中间工具调用细节）
    Run(ctx context.Context, prompt string) (string, error)
}

// SubAgentConfig 子智能体配置
type SubAgentConfig struct {
    MaxIterations int      // 最大迭代次数（防止无限循环）
    MaxTokens     int      // 最大生成 token 数
    SystemPrompt  string   // 系统提示词
}
```

### 2.3 在系统中的位置

```mermaid
graph TB
    subgraph "外部世界"
        USER[用户]
    end
    
    subgraph "Adapter 层"
        TUI[TUI 界面]
        TASK[Task 工具]
    end
    
    subgraph "UseCase 层"
        PA[父 Agent]
        SA[子 Agent]
    end
    
    subgraph "Domain 层"
        PS[父 Session]
        SS[子 Session]
    end
    
    USER --> TUI
    TUI --> PA
    PA --> PS
    PA --> TASK
    TASK --> SA
    SA --> SS
    
    style PS fill:#e3f2fd
    style SS fill:#c8e6c9
```

---

## 三、Task 工具实现

### 3.1 工具定义

```json
{
  "type": "object",
  "properties": {
    "prompt": {
      "type": "string",
      "description": "Detailed instructions for the subagent to execute"
    },
    "description": {
      "type": "string",
      "description": "Short description of the task (for logging)"
    }
  },
  "required": ["prompt"]
}
```

### 3.2 执行流程

```mermaid
flowchart TD
    Start([Execute 调用]) --> Parse[解析 JSON 参数]
    Parse --> CheckPrompt{prompt 非空?}
    
    CheckPrompt -->|否| ErrorPrompt[返回错误: prompt required]
    CheckPrompt -->|是| BuildConfig[构建子 Agent 配置]
    
    BuildConfig --> CreateSubAgent[创建子 Agent]
    CreateSubAgent --> SetSystem[设置系统提示]
    
    SetSystem --> Run[调用 subAgent.Run]
    
    Run --> Loop["子 Agent 循环执行<br/>(最多 MaxIterations 次)"]
    
    Loop --> CheckResult{执行成功?}
    CheckResult -->|否| Error[返回错误]
    CheckResult -->|是| Truncate{摘要 > 50000 字符?}
    
    Truncate -->|是| Cut[截断输出]
    Truncate -->|否| Return
    Cut --> Return[返回摘要]
    
    style ErrorPrompt fill:#ffcdd2
    style Error fill:#ffcdd2
    style Return fill:#c8e6c9
```

### 3.3 核心代码解析

**文件路径**: `internal/adapter/tool/task.go`

```go
func (t *TaskTool) Execute(ctx context.Context, args string) (string, error) {
    var params TaskToolParams
    if err := json.Unmarshal([]byte(args), &params); err != nil {
        return "", err
    }

    if params.Prompt == "" {
        return "Error: prompt is required", nil
    }

    // 构建子 Agent 配置
    config := SubAgentConfig{
        MaxIterations: 30,
        MaxTokens:     8000,
    }
    for _, opt := range t.subAgentOpts {
        opt(&config)
    }

    // 构建系统提示
    subAgentSystem := config.SystemPrompt
    if subAgentSystem == "" {
        if t.cwd != "" {
            subAgentSystem = "You are a coding subagent at " + t.cwd + 
                ". Complete the given task, then summarize your findings."
        } else {
            subAgentSystem = "You are a coding subagent. Complete the given task, then summarize your findings."
        }
    }

    // 创建子 Agent 配置
    agentConfig := usecase.AgentConfig{MaxTokens: config.MaxTokens}
    subAgentConfig := port.SubAgentConfig{
        MaxIterations: config.MaxIterations,
        MaxTokens:     config.MaxTokens,
        SystemPrompt:  subAgentSystem,
    }

    // ★ 创建子 Agent（独立 Session）
    subAgent := usecase.NewSubAgent(t.llmClient, t.toolReg, agentConfig, subAgentConfig)

    // 执行子 Agent
    summary, err := subAgent.Run(ctx, params.Prompt)
    if err != nil {
        return "Subagent error: " + err.Error(), nil
    }

    // 截断过长输出
    if len(summary) > 50000 {
        summary = summary[:50000] + "\n... (output truncated)"
    }

    if summary == "" {
        summary = "(no summary returned)"
    }

    return summary, nil
}
```

---

## 四、子 Agent 创建与运行

### 4.1 创建流程

```mermaid
sequenceDiagram
    participant TT as TaskTool
    participant A as Agent
    participant S as Session
    
    TT->>A: NewSubAgent(llmClient, toolReg, config, subConfig)
    
    Note over A: 1. 创建独立 Session
    A->>S: NewSession(model, provider)
    S-->>A: session (空 messages)
    
    Note over A: 2. 创建 Agent 实例
    A->>A: NewAgent(llmClient, toolReg, session, config)
    
    Note over A: 3. 设置子 Agent 属性
    A->>A: isSubAgent = true
    A->>A: excludeTools = ["task"]
    A->>A: subAgentConfig = subConfig
    
    A-->>TT: subAgent
```

### 4.2 核心代码解析

**文件路径**: `internal/usecase/agent.go`

```go
// NewSubAgent 创建子 Agent
func NewSubAgent(llmClient port.LLMClient, toolReg port.ToolRegistry, 
                 config AgentConfig, subConfig port.SubAgentConfig) *Agent {
    // ★ 创建独立的 Session
    session := entity.NewSession(llmClient.GetModel(), llmClient.GetName())

    agent := NewAgent(llmClient, toolReg, session, config)
    
    // ★ 设置子 Agent 标识
    agent.isSubAgent = true
    agent.excludeTools = []string{"task"}  // 防止递归
    agent.subAgentConfig = subConfig

    // 设置子 Agent 的系统提示
    if subConfig.SystemPrompt != "" {
        agent.system = subConfig.SystemPrompt
    }

    return agent
}
```

### 4.3 运行流程

```mermaid
flowchart TD
    Start([Run 调用]) --> Reset[重置 todoRounds = 0]
    Reset --> AddMsg[添加用户消息到独立 Session]
    
    AddMsg --> Loop["循环 (最多 MaxIterations 次)"]
    
    Loop --> CallLLM[callLLMStream]
    CallLLM --> CheckTools{有工具调用?}
    
    CheckTools -->|否| ReturnContent[返回最终内容]
    
    CheckTools -->|是| AddAssistant[添加 Assistant 消息]
    AddAssistant --> ExecTools[执行工具调用（静默）]
    
    ExecTools --> AddToolResults[添加工具结果到 Session]
    AddToolResults --> Loop
    
    ReturnContent --> End([返回摘要])
    
    style ReturnContent fill:#c8e6c9
```

### 4.4 Run 方法实现

```go
// Run 实现 SubAgentRunner 接口
func (a *Agent) Run(ctx context.Context, prompt string) (string, error) {
    // 重置 todo 计数
    a.todoRounds = 0

    // ★ 创建独立的用户消息
    userMsg := entity.NewMessage(entity.RoleUser, prompt)
    a.session.AddMessage(userMsg)

    var finalContent strings.Builder
    maxIterations := a.subAgentConfig.MaxIterations
    if maxIterations == 0 {
        maxIterations = 30
    }

    for iterationCount := 0; iterationCount < maxIterations; iterationCount++ {
        select {
        case <-ctx.Done():
            return finalContent.String(), errors.New(errors.CodeContextCanceled, "context canceled")
        default:
        }

        // 调用 LLM
        content, toolCalls, err := a.callLLMStream(ctx)
        if err != nil {
            return finalContent.String(), fmt.Errorf("API call failed: %v", err)
        }

        // ★ 如果没有工具调用，返回最终内容
        if len(toolCalls) == 0 {
            finalContent.WriteString(content)
            return finalContent.String(), nil
        }

        // 添加 assistant 消息
        assistantMsg := entity.NewMessage(entity.RoleAssistant, content).
            WithToolCalls(toolCalls)
        a.session.AddMessage(assistantMsg)

        // 执行工具调用（静默执行，不输出到 UI）
        for _, toolCall := range toolCalls {
            result, err := a.executeToolSilent(ctx, toolCall)
            if err != nil {
                a.logger.Error("tool execution failed",
                    logger.F("tool", toolCall.GetName()),
                    logger.F("error", err),
                )
            }
            
            // 添加工具结果到子 Agent 的 Session
            toolMsg := entity.NewMessage(entity.RoleTool, result.Content).
                WithToolCallID(result.ToolCallID)
            a.session.AddMessage(toolMsg)
        }
    }

    return finalContent.String(), fmt.Errorf("max iterations (%d) reached", maxIterations)
}
```

---

## 五、工具过滤机制

### 5.1 设计目的

防止子 Agent 递归创建子 Agent，导致资源耗尽。

```mermaid
graph TB
    subgraph "无过滤：无限递归"
        A1[父 Agent]
        A2[子 Agent]
        A3[孙 Agent]
        A4[...]
        
        A1 --> A2 --> A3 --> A4
    end
    
    subgraph "有过滤：单层隔离"
        B1[父 Agent<br/>tools: all]
        B2[子 Agent<br/>tools: all - task]
        
        B1 --> B2
        B2 -.->|task 不可用| X[❌]
    end
    
    style A4 fill:#ffcdd2
    style X fill:#ffcdd2
    style B2 fill:#c8e6c9
```

### 5.2 过滤实现

```go
// getTools 获取工具列表
// 子 Agent 需要排除特定工具（如 task）
func (a *Agent) getTools() []port.ToolDefinition {
    allTools := a.toolReg.ToLLMTools()

    // 如果不是子 Agent 或者没有排除工具，直接返回
    if !a.isSubAgent || len(a.excludeTools) == 0 {
        return allTools
    }

    // ★ 过滤排除的工具
    excludeSet := make(map[string]bool)
    for _, name := range a.excludeTools {
        excludeSet[name] = true
    }

    filtered := make([]port.ToolDefinition, 0, len(allTools))
    for _, tool := range allTools {
        if !excludeSet[tool.Function.Name] {
            filtered = append(filtered, tool)
        }
    }

    return filtered
}
```

### 5.3 工具对比

| Agent 类型 | excludeTools | 可用工具 |
|-----------|--------------|---------|
| 父 Agent | `[]` | bash, read_file, write_file, edit_file, todo, **task** |
| 子 Agent | `["task"]` | bash, read_file, write_file, edit_file, todo |

---

## 六、父子 Agent 交互

### 6.1 完整交互序列

```mermaid
sequenceDiagram
    participant U as 用户
    participant PA as 父 Agent
    participant PS as 父 Session
    participant TT as Task 工具
    participant SA as 子 Agent
    participant SS as 子 Session
    participant L as LLM
    participant T as 工具
    
    U->>PA: "调查测试框架"
    PA->>PS: AddMessage(userMsg)
    
    PA->>L: ChatStream(tools=[task, read_file, ...])
    L-->>PA: tool_call: task(prompt="调查...")
    
    PA->>PS: AddMessage(assistantMsg with toolCalls)
    PA->>TT: Execute(task)
    
    Note over TT,SA: 创建子 Agent
    TT->>SA: NewSubAgent()
    SA->>SS: NewSession()
    
    Note over SA,SS: 子 Agent 独立上下文
    
    TT->>SA: Run("调查...")
    SA->>SS: AddMessage(userMsg)
    
    loop 子 Agent 循环
        SA->>L: ChatStream(tools=[read_file, ...], 无 task)
        L-->>SA: tool_call: read_file("go.mod")
        SA->>T: Execute(read_file)
        T-->>SA: 文件内容
        SA->>SS: AddMessage(toolResult)
        
        SA->>L: ChatStream
        L-->>SA: tool_call: read_file("main_test.go")
        SA->>T: Execute(read_file)
        T-->>SA: 测试文件内容
        SA->>SS: AddMessage(toolResult)
        
        SA->>L: ChatStream
        L-->>SA: "使用 Go testing 框架"
    end
    
    Note over SS: 子 Session 丢弃
    
    SA-->>TT: "使用 Go testing 框架"
    TT-->>PA: tool_result
    
    PA->>PS: AddMessage(toolResult)
    
    Note over PS: 父 Session 干净<br/>只有摘要
    
    PA-->>U: "这个项目使用 Go testing 框架"
```

### 6.2 上下文对比

```mermaid
graph TB
    subgraph "父 Session"
        PS1["messages = [<br/>  {user: '调查测试框架'},<br/>  {assistant: '...', tool_calls: [task]},<br/>  {tool: '使用 Go testing 框架'}<br/>]"]
    end
    
    subgraph "子 Session（执行后丢弃）"
        SS1["messages = [<br/>  {user: '调查测试框架'},<br/>  {assistant: '...', tool_calls: [read_file]},<br/>  {tool: 'go.mod 内容 500 行'},<br/>  {assistant: '...', tool_calls: [read_file]},<br/>  {tool: 'main_test.go 内容 300 行'},<br/>  {assistant: '使用 Go testing 框架'}<br/>]"]
    end
    
    style PS1 fill:#c8e6c9
    style SS1 fill:#ffcdd2
```

---

## 七、静默执行模式

### 7.1 设计目的

子 Agent 执行工具时不发送输出到父 Agent UI，保持父 Agent 输出干净。

### 7.2 实现对比

```mermaid
graph LR
    subgraph "父 Agent 巧行"
        PA1[executeTool] --> PA2[emit OutputCommand]
        PA2 --> PA3[执行工具]
        PA3 --> PA4[emit OutputResult]
    end
    
    subgraph "子 Agent 静默执行"
        SA1[executeToolSilent] --> SA2[检查工具排除]
        SA2 --> SA3[执行工具]
        SA3 --> SA4[返回结果]
    end
    
    style PA2 fill:#e3f2fd
    style PA4 fill:#e3f2fd
```

### 7.3 核心代码

```go
// executeToolSilent 静默执行工具（不发送输出）
func (a *Agent) executeToolSilent(ctx context.Context, call entity.ToolCall) (entity.ToolResult, error) {
    // 检查工具是否被排除
    for _, excluded := range a.excludeTools {
        if call.GetName() == excluded {
            return entity.ToolResult{
                ToolCallID: call.ID,
                Content:    "Error: This tool is not available in subagent mode",
                IsError:    true,
            }, nil
        }
    }

    // 执行工具（无 emit 输出）
    return a.toolReg.ExecuteTool(ctx, call)
}
```

---

## 八、设计总结

### 8.1 隔离机制

```mermaid
graph TB
    subgraph "隔离层次"
        L1[Session 隔离<br/>独立消息历史]
        L2[工具隔离<br/>excludeTools 过滤]
        L3[输出隔离<br/>静默执行模式]
    end
    
    L1 --> L2 --> L3
    
    style L1 fill:#e3f2fd
    style L2 fill:#c8e6c9
    style L3 fill:#fff3e0
```

### 8.2 设计亮点

| 亮点 | 说明 |
|------|------|
| **工具驱动** | 子 Agent 通过 task 工具启动，LLM 自主决策 |
| **安全隔离** | excludeTools 机制防止递归 |
| **成本节约** | 中间步骤不污染父上下文 |
| **灵活配置** | 支持自定义迭代次数、系统提示 |
| **资源共享** | LLM Client 和 Tool Registry 共享 |

### 8.3 设计局限

| 局限 | 原因 | 可能改进 |
|------|------|---------|
| 无进度反馈 | 子 Agent 静默执行 | 添加回调机制 |
| 无并发限制 | 简单实现 | 添加并发池 |
| 无超时控制 | 依赖 ctx | 添加独立超时 |
| 单层嵌套 | 只排除 task | 支持多层嵌套 |

### 8.4 适用场景

| 场景 | 是否适用 | 原因 |
|------|---------|------|
| 调查代码库 | ✅ 适用 | 需要多轮探索，但只需要最终答案 |
| 搜索分析 | ✅ 适用 | 中间步骤多，结果简洁 |
| 修改文件 | ⚠️ 视情况 | 如果需要用户确认，不适合 |
| 长时间任务 | ❌ 不适用 | 无进度反馈 |