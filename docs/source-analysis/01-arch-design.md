# 架构与核心设计

> **项目**: ai_code (copilot)  
> **分析日期**: 2026-03-30

---

## 一、整体架构

### 1.1 六边形架构概览

项目采用**六边形架构（端口-适配器模式）**，将核心业务逻辑与外部依赖解耦。

```mermaid
graph TB
    subgraph "外部世界"
        USER[用户]
        LLM_API[LLM API]
        FS[文件系统]
    end
    
    subgraph "适配器层 (Adapter)"
        TUI[TUI 界面]
        IFlow[iFlow 适配器]
        OpenAI[OpenAI 适配器]
        Tools[工具适配器]
    end
    
    subgraph "端口层 (Port)"
        LLP[LLMClient 端口]
        TP[Tool 端口]
        TRP[ToolRegistry 端口]
        SAP[SubAgentRunner 端口]
    end
    
    subgraph "用例层 (UseCase)"
        Agent[Agent 编排器]
    end
    
    subgraph "领域层 (Domain)"
        Entity[实体]
        Errors[领域错误]
    end
    
    USER --> TUI
    LLM_API --> IFlow
    LLM_API --> OpenAI
    FS --> Tools
    
    TUI --> Agent
    Agent --> LLP
    Agent --> TRP
    Agent --> SAP
    
    LLP --> IFlow
    LLP --> OpenAI
    TRP --> Tools
    SAP --> Agent
    
    Agent --> Entity
    Agent --> Errors
```

### 1.2 分层职责

| 层级 | 目录 | 职责 | 依赖方向 |
|------|------|------|---------|
| **领域层** | `internal/domain/` | 定义实体和领域错误 | 无外部依赖 |
| **端口层** | `internal/port/` | 定义接口契约 | 依赖领域层 |
| **用例层** | `internal/usecase/` | 编排业务逻辑 | 依赖端口层 |
| **适配器层** | `internal/adapter/` | 实现具体技术 | 依赖端口层 |

### 1.3 目录结构映射

```
internal/
├── domain/                    # 领域层：核心业务概念
│   ├── entity/               # 实体定义
│   │   ├── session.go        # 会话实体
│   │   ├── message.go        # 消息实体
│   │   ├── tool.go           # 工具调用实体
│   │   └── id.go             # ID 生成
│   └── errors/               # 领域错误
│       └── errors.go
│
├── port/                      # 端口层：接口契约
│   ├── llm.go                # LLM 客户端接口
│   ├── tool.go               # 工具接口
│   └── subagent.go           # 子智能体接口
│
├── usecase/                   # 用例层：业务编排
│   └── agent.go              # Agent 核心逻辑
│
└── adapter/                   # 适配器层：技术实现
    ├── llm/                   # LLM 适配器
    │   ├── base.go           # 基础客户端
    │   ├── factory.go        # 工厂注册表
    │   ├── iflow.go          # iFlow 实现
    │   └── openai.go         # OpenAI 实现
    ├── tool/                  # 工具适配器
    │   ├── registry.go       # 工具注册表
    │   ├── bash.go           # Bash 工具
    │   ├── read_file.go      # 文件读取
    │   ├── write_file.go     # 文件写入
    │   ├── edit_file.go      # 文件编辑
    │   ├── todo.go           # Todo 工具
    │   └── task.go           # Task 工具
    └── ui/tui/                # UI 适配器
        ├── model.go          # 状态模型
        ├── update.go         # 事件处理
        └── view.go           # 视图渲染
```

---

## 二、核心实体关系

### 2.1 领域实体类图

```mermaid
classDiagram
    class Session {
        +ID string
        +Messages []Message
        +Model string
        +Provider string
        +CreatedAt time.Time
        +UpdatedAt time.Time
        +AddMessage(msg Message)
        +Clear()
        +LastMessage() *Message
        +SetModel(model string)
    }
    
    class Message {
        +ID string
        +Role Role
        +Content string
        +ToolCalls []ToolCall
        +ToolCallID string
        +Timestamp time.Time
        +WithToolCalls(calls) Message
        +WithToolCallID(id) Message
        +ToLLMMessage() map
    }
    
    class Role {
        <<enumeration>>
        system
        user
        assistant
        tool
    }
    
    class ToolCall {
        +ID string
        +Type string
        +Function FunctionCall
        +Result string
        +Status string
        +GetName() string
        +GetArguments() string
    }
    
    class FunctionCall {
        +Name string
        +Arguments string
    }
    
    class ToolResult {
        +ToolCallID string
        +Content string
        +IsError bool
        +Timestamp time.Time
    }
    
    Session "1" --> "*" Message : contains
    Message "1" --> "*" ToolCall : may have
    Message --> Role : has
    ToolCall --> FunctionCall : contains
    ToolResult --> ToolCall : references
```

### 2.2 实体职责说明

| 实体 | 职责 | 生命周期 |
|------|------|---------|
| **Session** | 管理对话历史和模型配置 | 整个会话期间 |
| **Message** | 封装单条消息（用户/助手/工具） | 创建后不可变 |
| **ToolCall** | 表示 LLM 发起的工具调用请求 | 随消息存在 |
| **ToolResult** | 表示工具执行结果 | 随工具调用产生 |

---

## 三、端口接口设计

### 3.1 端口接口关系图

```mermaid
classDiagram
    class LLMClient {
        <<interface>>
        +Chat(ctx, req) ChatResponse, error
        +ChatStream(ctx, req, handler) error
        +GetName() string
        +GetModel() string
        +SetModel(model string)
    }
    
    class Tool {
        <<interface>>
        +Name() string
        +Description() string
        +Parameters() map
        +Execute(ctx, args) string, error
    }
    
    class ToolRegistry {
        <<interface>>
        +Register(tool Tool)
        +Get(name) Tool, bool
        +List() []Tool
        +ToLLMTools() []ToolDefinition
        +ExecuteTool(ctx, call) ToolResult, error
    }
    
    class SubAgentRunner {
        <<interface>>
        +Run(ctx, prompt) string, error
    }
    
    class Agent {
        -llmClient LLMClient
        -toolReg ToolRegistry
        -session Session
        -isSubAgent bool
        +ProcessMessage(ctx, input) error
        +Run(ctx, prompt) string, error
    }
    
    Agent ..> LLMClient : uses
    Agent ..> ToolRegistry : uses
    Agent ..|> SubAgentRunner : implements
    ToolRegistry --> Tool : manages
```

### 3.2 接口契约说明

| 接口 | 方法 | 说明 |
|------|------|------|
| **LLMClient** | `Chat()` | 非流式对话 |
| | `ChatStream()` | 流式对话 |
| | `GetModel()/SetModel()` | 模型切换 |
| **Tool** | `Name()` | 工具标识 |
| | `Parameters()` | JSON Schema 参数定义 |
| | `Execute()` | 执行工具逻辑 |
| **ToolRegistry** | `Register()` | 注册工具 |
| | `ToLLMTools()` | 生成 LLM 可识别的工具定义 |
| | `ExecuteTool()` | 根据调用执行工具 |
| **SubAgentRunner** | `Run()` | 执行子任务并返回摘要 |

---

## 四、核心流程

### 4.1 Agent 循环流程图

Agent 循环是整个系统的核心执行模型。

```mermaid
flowchart TD
    Start([用户输入]) --> AddUser[添加用户消息到 Session]
    AddUser --> CallLLM[调用 LLM ChatStream]
    
    CallLLM --> ReceiveStream[接收流式响应]
    ReceiveStream --> Accumulate[累积文本内容]
    Accumulate --> AccumulateTools[累积工具调用]
    
    AccumulateTools --> HasTools{有工具调用?}
    
    HasTools -->|否| AddAssistant[添加 Assistant 消息]
    AddAssistant --> End([循环结束])
    
    HasTools -->|是| AddAssistantWithTools[添加带 ToolCalls 的 Assistant 消息]
    AddAssistantWithTools --> ExecuteTools[执行工具调用]
    
    ExecuteTools --> CheckTodo{使用了 todo 工具?}
    CheckTodo -->|否| IncrementCounter[todoRounds++]
    IncrementCounter --> CheckThreshold{todoRounds >= 3?}
    CheckThreshold -->|是| InjectReminder[注入提醒标签]
    CheckThreshold -->|否| AddToolResults
    CheckTodo -->|是| ResetCounter[重置 todoRounds = 0]
    ResetCounter --> AddToolResults
    
    InjectReminder --> AddToolResults[添加工具结果到 Session]
    AddToolResults --> CallLLM
    
    style Start fill:#e3f2fd
    style End fill:#c8e6c9
    style InjectReminder fill:#fff3e0
```

### 4.2 Agent 循环序列图

```mermaid
sequenceDiagram
    participant U as 用户
    participant A as Agent
    participant S as Session
    participant L as LLMClient
    participant T as ToolRegistry
    
    U->>A: ProcessMessage(input)
    A->>S: AddMessage(userMsg)
    
    loop Agent 循环
        A->>S: buildMessages()
        S-->>A: messages
        A->>T: ToLLMTools()
        T-->>A: tools
        A->>L: ChatStream(messages, tools)
        
        loop 流式接收
            L-->>A: chunk (delta)
            A->>A: 累积 content/toolCalls
        end
        
        alt 有工具调用
            A->>S: AddMessage(assistantMsg with toolCalls)
            
            loop 每个工具调用
                A->>T: ExecuteTool(call)
                T-->>A: ToolResult
            end
            
            A->>A: injectTodoReminder()
            
            loop 每个工具结果
                A->>S: AddMessage(toolMsg)
            end
        else 无工具调用
            A->>S: AddMessage(assistantMsg)
            A-->>U: 完成
        end
    end
```

### 4.3 消息状态流转

```mermaid
stateDiagram-v2
    [*] --> UserInput: 用户发送消息
    
    UserInput --> Pending: 添加到 Session
    
    state Pending {
        [*] --> WaitingLLM
        WaitingLLM --> Streaming: LLM 开始响应
    }
    
    state Streaming {
        [*] --> Accumulating
        Accumulating --> HasToolCalls: 检测到工具调用
        Accumulating --> NoToolCalls: 无工具调用
    }
    
    state ToolExecution {
        [*] --> Executing
        Executing --> Completed: 所有工具执行完毕
    }
    
    HasToolCalls --> ToolExecution
    ToolExecution --> Pending: 添加工具结果
    
    NoToolCalls --> [*]: 返回最终响应
    
    note right of ToolExecution
        工具执行可能触发
        多轮 Agent 循环
    end note
```

---

## 五、启动流程

### 5.1 应用启动序列图

```mermaid
sequenceDiagram
    participant M as main()
    participant C as Config
    participant L as LLM Factory
    participant T as Tool Registry
    participant S as Session
    participant U as TUI Model
    participant P as tea.Program
    
    M->>C: Load()
    C-->>M: Config
    
    M->>L: Get(provider, config)
    L-->>M: LLMClient
    
    M->>T: NewRegistry()
    M->>T: Register(BashTool)
    M->>T: Register(ReadFileTool)
    M->>T: Register(WriteFileTool)
    M->>T: Register(EditFileTool)
    M->>T: Register(TodoTool)
    M->>T: Register(TaskTool)
    
    M->>S: NewSession(model, provider)
    
    M->>U: NewModel(llmClient, session, toolReg)
    M->>U: SetTextInput()
    
    M->>P: NewProgram(model)
    M->>P: Run()
    P-->>M: 退出
```

### 5.2 组件初始化顺序

```mermaid
flowchart LR
    subgraph "配置阶段"
        A1[加载 .env]
        A2[加载 config.yaml]
        A3[环境变量覆盖]
    end
    
    subgraph "核心组件"
        B1[创建 LLM Client]
        B2[创建 Tool Registry]
        B3[创建 Session]
    end
    
    subgraph "工具注册"
        C1[Bash]
        C2[ReadFile]
        C3[WriteFile]
        C4[EditFile]
        C5[Todo]
        C6[Task]
    end
    
    subgraph "UI 层"
        D1[TUI Model]
        D2[TextInput]
        D3[tea.Program]
    end
    
    A1 --> A2 --> A3 --> B1
    B1 --> B2 --> B3
    B2 --> C1 & C2 & C3 & C4 & C5 & C6
    B1 & B2 & B3 --> D1
    D1 --> D2 --> D3
```

---

## 六、TUI 架构

### 6.1 TUI 状态机

TUI 基于 Bubble Tea 框架实现，采用 Elm 架构。

```mermaid
stateDiagram-v2
    [*] --> StateInput: 初始化
    
    StateInput --> StateProcessing: Enter 提交消息
    StateInput --> StateModelSelector: /model 命令
    
    StateModelSelector --> StateInput: Enter 选择模型
    StateModelSelector --> StateInput: Esc 取消
    
    StateProcessing --> StateInput: 任务完成
    StateProcessing --> StateInput: Ctrl+C 取消
    
    note right of StateProcessing
        处理中状态：
        - 流式输出显示
        - 工具执行进度
        - 计时器更新
    end note
```

### 6.2 TUI 事件流

```mermaid
sequenceDiagram
    participant U as 用户
    participant P as tea.Program
    participant M as Model
    participant A as Agent
    participant O as Output Channel
    
    U->>P: 按键事件
    P->>M: Update(msg)
    
    alt Enter 键
        M->>A: ProcessMessage(input)
        A->>O: emit(OutputTextChunk)
        loop 监听输出
            O-->>M: Output
            M->>M: append message
            M-->>P: listenOutput cmd
        end
    else Ctrl+C
        M->>A: cancel context
        M->>M: state = StateInput
    end
    
    M-->>P: 返回 cmd
    P->>M: View()
    M-->>P: 渲染字符串
    P-->>U: 显示界面
```

---

## 七、设计亮点总结

| 设计点 | 实现方式 | 优势 |
|--------|---------|------|
| **六边形架构** | Port + Adapter 分层 | 核心逻辑与技术解耦，易于测试 |
| **工厂模式** | LLM 注册表 | 支持多提供商，扩展性强 |
| **工具注册表** | 统一 Tool 接口 | 工具可插拔，LLM 可发现 |
| **流式响应** | SSE + StreamHandler | 实时输出，用户体验好 |
| **上下文隔离** | 独立 Session + Task 工具 | 子 Agent 上下文干净 |
| **Elm 架构** | Bubble Tea TUI | 状态管理清晰，事件驱动 |
