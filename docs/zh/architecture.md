# 系统架构

## 项目结构

```
vibecoding/
├── agent/                       # 公共 Agent/Provider 接口与 Builder
├── cmd/vibecoding/              # CLI 入口点
│   └── main.go                  # 主程序
├── internal/
│   ├── agent/                   # 核心 Agent 循环
│   │   ├── agent.go             # Agent 主逻辑
│   │   ├── factory.go           # AgentFactory，统一每个 Agent 的创建
│   │   ├── manager.go           # AgentManager 生命周期管理
│   │   ├── router.go            # EventRouter
│   │   ├── subagent.go          # subagent_* 工具
│   │   ├── events.go            # 事件类型定义
│   │   ├── provider.go          # Provider 接口适配
│   │   └── system_prompt.go     # 系统提示词生成
│   ├── config/                  # 配置管理
│   ├── context/                 # 上下文管理和 token 估算
│   ├── contextfiles/            # 上下文文件加载
│   ├── platform/                # 跨平台兼容工具
│   ├── provider/                # LLM Provider 抽象
│   │   ├── anthropic/           # Anthropic Messages API
│   │   ├── factory/             # 共享 provider/model 创建逻辑
│   │   ├── vendor*.go           # 厂商适配注册和默认值
│   │   └── openai/              # OpenAI Chat Completions API
│   ├── cron/                    # 定时任务存储和调度器
│   ├── sandbox/                 # 沙箱抽象 (bwrap, none)
│   ├── session/                 # 会话管理 (JSONL)
│   ├── skills/                  # 技能系统
│   ├── tools/                   # 工具实现
│   │   ├── bash.go              # Bash 命令执行
│   │   ├── read.go              # 文件读取
│   │   ├── write.go             # 文件写入
│   │   ├── edit.go              # 文件编辑
│   │   ├── grep.go              # 内容搜索
│   │   ├── find.go              # 文件查找
│   │   └── ls.go                # 目录列表
│   ├── tui/                     # 终端 UI (BubbleTea)
│   └── ua/                      # User-Agent 字符串生成
```

## 核心组件

### 1. Provider 系统

Provider 是与 LLM API 交互的抽象层。CLI 与 ACP 的 provider 创建都经过
`internal/provider/factory`，先应用厂商适配默认值，再构造通用 OpenAI
兼容或 Anthropic 兼容协议 provider。

```
┌─────────────────────────────────────────────────────────────┐
│                      Provider Interface                      │
├─────────────────────────────────────────────────────────────┤
│  Chat(ctx, params) <-chan StreamEvent                       │
│  Models() []*Model                                          │
│  GetModel(id string) *Model                                 │
│  Name() string                                              │
└─────────────────────────────────────────────────────────────┘
                              │
                              │
            ┌─────────────────┴─────────────────┐
            ▼                                   ▼
    ┌───────────────────┐             ┌───────────────────┐
    │ 厂商适配器         │             │ 通用 fallback      │
    │ vendor_*.go       │             │ openai/anthropic  │
    └───────────────────┘             └───────────────────┘
```

厂商选择顺序：

1. provider 配置中的显式 `vendor`
2. 根据 Base URL 自动识别
3. 根据 `api` 回退到通用协议 provider

#### StreamEvent 类型

```go
type StreamEvent struct {
    Type        EventType
    TextDelta   string      // 文本增量
    ThinkingDelta string    // 思考增量
    ToolCall    *ToolCall   // 工具调用
    Usage       *Usage      // Token 使用量
    Error       error       // 错误
}
```

### 2. Agent 循环

Agent 是核心逻辑，协调 Provider、Tools 和 Session。

```
┌─────────────────────────────────────────────────────────────┐
│                       Agent Loop                             │
├─────────────────────────────────────────────────────────────┤
│  1. 构建系统提示词 (模式 + 工具 + 上下文文件 + 技能)         │
│  2. 发送消息到 Provider                                      │
│  3. 处理流式事件 (文本、思考、工具调用)                       │
│  4. 执行工具并收集结果                                        │
│  5. 将工具结果添加到消息                                      │
│  6. 重复直到完成                                              │
└─────────────────────────────────────────────────────────────┘
```

#### 执行流程

```
User Input
    │
    ▼
┌───────────────┐
│ Build Context │ ← System Prompt + Tools + Context Files + Skills
└───────┬───────┘
        │
        ▼
┌───────────────┐
│  Provider     │ ← LLM API (OpenAI/Anthropic)
│  Chat()       │
└───────┬───────┘
        │
        ▼
┌───────────────┐     ┌───────────────┐
│ Stream Events │────▶│ Tool Calls?   │
└───────┬───────┘     └───────┬───────┘
        │                     │
        │                     ▼
        │              ┌───────────────┐
        │              │ Execute Tools │
        │              └───────┬───────┘
        │                     │
        │                     ▼
        │              ┌───────────────┐
        └──────────────│ Append Results│
                       └───────────────┘
```

### 3. 多 Agent 运行时

多 Agent 模式通过 `--multi-agent` 显式启用。启用后，主 Agent 会获得
`subagent_spawn`、`subagent_status`、`subagent_send`、`subagent_destroy`
工具。子 Agent 拥有独立的 messages、context、session、registry 和 job
manager 状态。

```
Main Agent
    │
    ├── AgentManager 创建子 Agent
    ├── EventRouter 按 AgentID 路由事件
    └── subagent_* 工具管理异步子任务
```

子 Agent 的 registry 会过滤 `subagent_*` 工具，因此不能继续创建嵌套子 Agent。

### 4. Cron 调度器

`internal/cron` 包提供文件持久化的 cron store 和 scheduler，可通过子 Agent
执行任务。TUI 在多 Agent 模式下暴露 `/cron` 命令入口；自然语言解析和持久化
TUI 管理仍属于后续接线工作。

### 5. 工具系统

工具是 Agent 与外部世界交互的方式。

```
┌─────────────────────────────────────────────────────────────┐
│                    Tool Interface                            │
├─────────────────────────────────────────────────────────────┤
│  Name() string                                              │
│  Description() string                                       │
│  Parameters() json.RawMessage                               │
│  Execute(ctx, params) (*ToolResult, error)                  │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│  File Tools   │   │  Search Tools │   │  System Tools │
│  - read       │   │  - grep       │   │  - bash       │
│  - write      │   │  - find       │   │  - ls         │
│  - edit       │   │               │   │               │
└───────────────┘   └───────────────┘   └───────────────┘
```

### 6. 会话管理

会话使用 JSONL 格式存储，支持树状结构和分支。

```
┌─────────────────────────────────────────────────────────────┐
│                    Session Structure                         │
├─────────────────────────────────────────────────────────────┤
│  {                                                          │
│    "id": "session-abc123",                                  │
│    "type": "session",                                       │
│    "timestamp": "2024-01-01T00:00:00Z",                     │
│    "cwd": "/home/user/project",                             │
│    "provider": "anthropic",                                 │
│    "model": "claude-sonnet-4-20250514"                      │
│  }                                                          │
│  {                                                          │
│    "id": "msg-001",                                         │
│    "parentId": "session-abc123",                            │
│    "type": "message",                                       │
│    "role": "user",                                          │
│    "content": "..."                                         │
│  }                                                          │
│  {                                                          │
│    "id": "msg-002",                                         │
│    "parentId": "msg-001",                                   │
│    "type": "message",                                       │
│    "role": "assistant",                                     │
│    "content": "..."                                         │
│  }                                                          │
└─────────────────────────────────────────────────────────────┘
```

#### 会话类型

| type | 描述 |
|------|------|
| `session` | 会话元数据 |
| `message` | 用户/助手消息 |
| `model_change` | 模型切换记录 |
| `compaction` | 上下文压缩记录 |
| `label` | 会话标签 |

### 7. 沙箱系统

沙箱通过 bubblewrap (bwrap) 实现进程隔离。

```
┌─────────────────────────────────────────────────────────────┐
│                     Sandbox Manager                          │
├─────────────────────────────────────────────────────────────┤
│  SetLevel(level)                                            │
│  GetActive() *Sandbox                                       │
│  Execute(cmd) (stdout, stderr, error)                       │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│  LevelNone    │   │ LevelStandard │   │  LevelStrict  │
│  (无限制)     │   │ (项目读写)    │   │  (项目只读)   │
└───────────────┘   └───────────────┘   └───────────────┘
```

### 8. TUI 系统

基于 BubbleTea 的终端用户界面。

```
┌─────────────────────────────────────────────────────────────┐
│                        TUI App                              │
├─────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   Header Bar                          │  │
│  │  Provider: anthropic │ Model: claude-sonnet-4 │ Mode  │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   Message Area                        │  │
│  │  User: ...                                            │  │
│  │  Assistant: ...                                        │  │
│  │  [tool: bash] running...                              │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   Input Area                          │  │
│  │  > _                                                  │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   Status Bar                          │  │
│  │  Thinking: medium │ Tokens: 1234 in / 567 out │ Cost  │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## 数据流

### 完整请求流程

```
1. 用户输入
   │
   ▼
2. TUI 捕获输入
   │
   ▼
3. Agent.Run(ctx, input)
   │
   ▼
4. 构建系统提示词
   ├── 模式提示 (plan/agent/yolo)
   ├── 工具定义 (JSON Schema)
   ├── 上下文文件 (AGENTS.md, CLAUDE.md)
   └── 技能上下文
   │
   ▼
5. 构建消息历史
   ├── 历史消息 (from Session)
   └── 新用户消息
   │
   ▼
6. Provider.Chat(ctx, params)
   │
   ▼
7. SSE 流式响应
   ├── TextDelta → 显示文本
   ├── ThinkingDelta → 显示思考
   └── ToolCall → 执行工具
   │
   ▼
8. 工具执行 (通过 Sandbox)
   │
   ▼
9. 收集工具结果
   │
   ▼
10. 继续对话 (回到步骤 5)
   │
   ▼
11. 完成，保存会话
```

## 关键设计决策

### 1. 接口抽象

使用接口抽象 Provider 和 Tool，便于扩展和测试。

### 2. 流式处理

使用 Channel 实现流式响应，提供实时反馈。

### 3. 会话树

使用树状结构存储会话，支持分支和恢复。

### 4. 分层配置

支持全局和项目配置，项目配置覆盖全局。

### 5. 沙箱隔离

通过 bubblewrap 实现进程级隔离，保护系统安全。
