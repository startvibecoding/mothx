# Multi-Agent 架构演进计划

## 设计决策 (已确认)

| # | 决策点 | 选择 | 说明 |
|---|--------|------|------|
| 1 | Agent 接口粒度 | **A: 单一大接口** | 定义完整的 `Agent` 接口 (Run/Abort/GetMessages 等)，不拆分 |
| 2 | Registry 隔离程度 | **A: 独立 Registry 实例** | 每个 Agent 创建独立 Registry，各自持有完整 Tool 对象集 |
| 3 | 子 Agent 交互模式 | **B: 异步 handle** | 主 Agent 启动子 Agent 后立即返回 handle，后续通过工具查询状态/结果 |
| 4 | 实施节奏 | **B: 分批** | 先做 Phase 1-3 (接口+解耦+工厂)，合入验证稳定后再做 Phase 4-6 |
| 5 | 子 Agent 嵌套 | **禁止** | 子 Agent 不能再派生子 Agent，仅主 Agent (depth=0) 有权 spawn |
| 6 | 子 Agent 上下文 | **完全隔离** | 子 Agent 有独立的 messages/context/session，不污染主 Agent 上下文 |
| 7 | 子 Agent 缓存优化 | **继承主 Agent 策略** | 子 Agent 同样使用 frozen system prompt + dual-marker rolling buffer |
| 8 | 多 Agent 模式开关 | **默认关闭** | 需 Ctrl+P 快捷键或 `--multi-agent` 参数才启用，subagent 工具仅在开启时注册 |
| 9 | 定时任务模式 | **独立功能，依赖多 Agent** | `/cron` + 自然语言管理定时任务，触发时自动派生 subagent 执行 |
| 10 | Agent 接口可见性 | **公共包，外部可引用** | 接口定义放在 `agent/` (非 internal)，第三方 Go 开发者可 import 并自定义实现 |
| 11 | 公共 Agent 创建方式 | **Builder 模式** | `agent.NewBuilder().WithProvider(...).WithWorkDir(...).Build()` 返回 Agent 接口，不暴露 Registry 细节 |
| 12 | Provider 适配架构 | **三层结构** | 公共 Provider 接口 → 厂商适配层 (deepseek/xiaomi/claude/...) → 通用 fallback provider |
| 13 | Provider 厂商选择 | **三级 fallback** | 1. `vendor` 字段显式指定 → 2. `baseUrl` 自动识别 → 3. 通用 fallback |
| 14 | 厂商差异处理 | **compat 标志位** | 大多数厂商走通用 provider + compat 标志位，仅少数需要独立 provider 包 (参考 pi/packages/ai) |

---

## 现状问题

| # | 问题 | 位置 | 严重度 |
|---|------|------|--------|
| P1 | `Agent` 是具体 struct，无接口抽象 | `agent/agent.go:124` | 🔴 |
| P2 | Agent 创建散落 3 处，Config 组装重复 | main.go:564, app.go:1133, acp.go:584 | 🟡 |
| P3 | `tools.Registry` 持有全局 workDir + sandbox | `tools/tool.go:144-150` | 🔴 |
| P4 | `JobManager` 是全局的，进程级单例 | `tools/jobmanager.go:28` | 🔴 |
| P5 | Event 无 AgentID，无法路由到正确的 Agent | `agent/events.go:52` | 🔴 |
| P6 | Approval 机制无 Agent 路由 | `agent/agent.go:1134-1161` | 🟡 |
| P7 | `BashTool` 直接持有 Registry 引用 | `tools/bash.go` | 🟡 |
| P8 | Session 是 1:1 绑定 Agent 的 | `session/session.go:21` | 🟡 |
| P9 | 无 Agent 生命周期管理器 | 不存在 | 🔴 |
| P10 | Provider 接口定义在两个包中有重复 | `agent/provider.go` vs `provider/` | 🟡 |

---

## 第一批: Phase 1-3 (接口 + 解耦 + 工厂)

### Phase 1: 接口抽象 (Foundation) — 2-3天

#### Step 1.1: 定义 Agent 接口
- [x] 新建 `agent/types.go` (公共包，非 internal，外部可引用)
  - Go 的 `internal/` 包外部无法 import，所以接口放在顶层 `agent/` 包
  - import path: `github.com/startvibecoding/vibecoding/agent`
- [x] 定义 `AgentID` 类型 (`type AgentID string`)
- [x] 定义 `Agent` 接口，方法列表:
  - `ID() AgentID`
  - `ParentID() AgentID`
  - `Run(ctx context.Context, userMsg string) <-chan Event`
  - `RunWithMessages(ctx context.Context, messages []provider.Message) <-chan Event`
  - `Abort()`
  - `GetMessages() []provider.Message`
  - `SetMessages(msgs []provider.Message)`
  - `GetContext() *AgentContext`
  - `SetContext(ctx *AgentContext)`
  - `GetContextUsage() *ctxpkg.ContextUsage`
  - `LoadHistoryMessages(messages []provider.Message)`
  - `HandleApprovalResponse(approvalID string, approved bool)`
- [x] 定义 `AgentConfigView` 只读视图 (ID, ParentID, Mode, Model)
- [x] 定义公共类型: `Event`, `EventType`, `AgentContext`, `AgentID` (从 internal/agent/events.go 迁移到公共包)
- [x] 内部实现 `internal/agent/` import 公共包 `agent/`，保持内部逻辑不变
- [x] `make test` 通过

#### Step 1.1b: 定义 Builder (决策 11)
- [x] 新建 `agent/builder.go` (公共包)
- [x] 定义 `Builder` struct:
  ```go
  type Builder struct {
      provider           Provider       // 公共 Provider 接口
      modelID            string
      mode               string         // "plan", "agent", "yolo"
      workDir            string
      thinkingLevel      string
      maxTokens          int
      systemPromptExtra  string
      maxIterations      int
      toolExecutionMode  string         // "sequential", "parallel"
      tools              []string       // tool filter (空=全部)
      sandbox            bool           // 是否启用沙箱
      sessionDir         string         // session 持久化目录
      compactionEnabled  bool
      compactionReserve  int
      approvalHandler    func(toolCallID, toolName string, args map[string]any) bool
  }
  ```
- [x] 实现 Builder 方法链:
  - `NewBuilder() *Builder` — 创建 Builder，设置合理默认值
  - `WithProvider(p Provider) *Builder`
  - `WithModel(modelID string) *Builder`
  - `WithMode(mode string) *Builder`
  - `WithWorkDir(dir string) *Builder`
  - `WithThinkingLevel(level string) *Builder`
  - `WithMaxTokens(n int) *Builder`
  - `WithSystemPromptExtra(extra string) *Builder`
  - `WithMaxIterations(n int) *Builder`
  - `WithToolExecutionMode(mode string) *Builder`
  - `WithTools(tools []string) *Builder`
  - `WithSandbox(enabled bool) *Builder`
  - `WithSessionDir(dir string) *Builder`
  - `WithCompaction(enabled bool, reserveTokens int) *Builder`
  - `WithApprovalHandler(h func(...) bool) *Builder`
- [x] 实现 `Build() (Agent, error)`:
  - 内部创建 tools.Registry (用 workDir + sandbox)
  - 内部组装 internal agent.Config
  - 调用 internal agent.New() 创建实例
  - 返回 Agent 接口
- [x] 定义公共 `Provider` 接口 (agent 包内，避免开发者依赖 internal/provider):
  ```go
  type Provider interface {
      Chat(ctx context.Context, params ChatParams) <-chan StreamEvent
      Name() string
      Models() []ModelInfo
      GetModel(id string) *ModelInfo
  }
  ```
- [x] 定义公共 `ChatParams`, `StreamEvent`, `ModelInfo`, `ToolDefinition` 等类型
- [x] `make test` 通过

#### Step 1.1c: Provider 三层架构 (决策 12)

目标: 公共 Provider 接口 → 厂商适配层 → 通用 fallback

```
┌─────────────────────────────────────────────────────┐
│  agent.Provider (公共接口)                           │
│  - Chat / Name / Models / GetModel                  │
└──────────────────┬──────────────────────────────────┘
                   │
        ┌──────────┴──────────┐
        ▼                     ▼
┌───────────────┐    ┌────────────────────────────────┐
│ 适配层 (厂商)   │    │  通用 fallback                  │
│               │    │                                │
│ deepseek/     │    │  openai_compatible/             │
│ xiaomi/       │    │  - 任何 OpenAI 兼容 API          │
│ claude/       │    │  - 自动检测 API 格式              │
│ gemini/       │    │  - 按 URL + key 即可连接          │
│ qwen/         │    │                                │
│ ...           │    │                                │
└───────────────┘    └────────────────────────────────┘
```

**厂商适配层** (内部包 `internal/provider/`):

> 参考 `/home/free/src/pi/packages/ai/src/models.generated.ts` 的 compat 机制:
> 大多数厂商使用 OpenAI 或 Anthropic 兼容 API，差异通过 compat 标志位处理，而非独立 provider 实现。

- [x] `internal/provider/openai/` — OpenAI Chat Completions (已有，基础实现)
- [x] `internal/provider/anthropic/` — Anthropic Messages API (已有，基础实现)
- [x] `internal/provider/google/` — Google Gemini API
- [x] `internal/provider/mistral/` — Mistral Conversations API

**厂商差异通过 compat 标志位处理** (而非独立 provider 包):

大多数厂商 (deepseek/xiaomi/kimi/minimax/seed/qianfan/bailian/gitee) 实际上都走 OpenAI 或 Anthropic 兼容 API，
差异仅在于请求/响应的细微不同，通过 compat 标志位在通用 provider 中处理:

```go
// ModelCompat 定义模型级别的兼容性标志 (参考 pi/packages/ai)
type ModelCompat struct {
    // thinking/reasoning 相关
    ThinkingFormat                       string `json:"thinkingFormat,omitempty"`          // "deepseek" | "openai" | "anthropic" | "together" | "zai" | "qwen"
    RequiresReasoningContentOnAssistant  bool   `json:"requiresReasoningContentOnAssistant,omitempty"` // 从 assistant 消息提取 reasoning_content
    ForceAdaptiveThinking                bool   `json:"forceAdaptiveThinking,omitempty"`    // 强制自适应 thinking 模式

    // API 参数兼容
    SupportsDeveloperRole     bool   `json:"supportsDeveloperRole,omitempty"`     // 是否支持 system/developer role
    SupportsStore             bool   `json:"supportsStore,omitempty"`             // 是否支持 store 参数
    SupportsReasoningEffort   bool   `json:"supportsReasoningEffort,omitempty"`   // 是否支持 reasoning_effort
    SupportsStrictMode        bool   `json:"supportsStrictMode,omitempty"`        // 是否支持 strict JSON schema
    MaxTokensField            string `json:"maxTokensField,omitempty"`            // "max_tokens" | "max_completion_tokens"

    // 缓存相关
    SupportsCacheControlOnTools bool `json:"supportsCacheControlOnTools,omitempty"` // 工具定义上的缓存控制
    SupportsLongCacheRetention  bool `json:"supportsLongCacheRetention,omitempty"`  // 长缓存保留
    SendSessionAffinityHeaders  bool `json:"sendSessionAffinityHeaders,omitempty"`  // 会话亲和性头

    // 流式相关
    SupportsEagerToolInputStreaming bool `json:"supportsEagerToolInputStreaming,omitempty"` // 急切工具输入流
}
```

**实际厂商差异对照** (来自参考实现):

| 厂商 | API 格式 | thinkingFormat | 特殊 compat |
|------|----------|---------------|-------------|
| deepseek | openai-completions | deepseek | requiresReasoningContent |
| xiaomi | openai-completions | deepseek | requiresReasoningContent |
| kimi (moonshotai) | openai-completions | — | supportsDeveloperRole=false |
| minimax | openai-completions | — | supportsStore=false |
| qwen | openai-completions | qwen | supportsReasoningEffort=false |
| anthropic | anthropic-messages | anthropic | 原生支持 |
| google | google-generative-ai | — | 原生 API |
| mistral | mistral-conversations | — | 原生 API |
| together | openai-completions | together | supportsDeveloperRole=false |
| zai | openai-completions | zai | zaiToolStream |

**实现方式**:
- `ModelConfig` struct 增加 `Compat *ModelCompat` 字段 (JSON: `"compat"`)
- 通用 openai_compatible provider 在发送请求前检查 compat 标志，调整请求格式
- 通用 anthropic_compatible provider 同理
- 大多数厂商只需配置正确的 compat 标志，无需独立 provider 包

**通用 fallback** (内部包):
- [x] `internal/provider/openai_compatible/` — OpenAI 兼容通用 provider
  - 接受任意 base URL + API key
  - 自动处理 OpenAI 兼容的流式 SSE 格式
  - 适用于: Azure OpenAI, vLLM, Ollama, LM Studio, DeepSeek, 任何 OpenAI 兼容 API
  - 作为未知厂商的默认选择
- [x] `internal/provider/anthropic_compatible/` — Anthropic Messages API 兼容通用 provider
  - 接受任意 base URL + API key
  - 自动处理 Anthropic Messages API 的流式 SSE 格式
  - 支持 thinking/extended thinking 等 Anthropic 特性
  - 适用于: 任何使用 Anthropic Messages API 格式的厂商代理/网关

**厂商适配 vs 通用 fallback 的选择逻辑**:
- 若用户配置 `provider: "deepseek"` → 走 `deepseek/` 适配 (处理 reasoning model 等特有逻辑)
- 若用户配置 `provider: "openai_compatible"` + `base_url` → 走通用 OpenAI 兼容
- 若用户配置 `provider: "anthropic_compatible"` + `base_url` → 走通用 Anthropic 兼容
- 厂商适配内部可以复用通用 fallback 的核心逻辑，只覆盖差异部分

**Provider 注册表**:
- [x] `internal/provider/registry.go` — Provider 注册表
  ```go
  type Registry struct {
      providers map[string]func(ProviderConfig) (agent.Provider, error)
  }
  ```
  - `Register(name string, factory func(ProviderConfig) (agent.Provider, error))`
  - `Create(name string, cfg ProviderConfig) (agent.Provider, error)`
  - `List() []string` — 返回已注册的 provider 名称
- [x] 各厂商 provider 在 init() 中自动注册
- [x] 用户在 settings.json 中配置 (保持现有格式，新增厂商自动可用):
  ```json
  {
    "providers": {
      "deepseek-openai": {
        "vendor": "deepseek",
        "baseUrl": "https://api.deepseek.com",
        "apiKey": "${DEEPSEEK_API_KEY}",
        "api": "openai-chat",
        "models": [
          {"id": "deepseek-v4-flash", "name": "DeepSeek-V4-Flash", "contextWindow": 1000000, "maxTokens": 384000}
        ]
      },
      "xiaomi-milm": {
        "vendor": "xiaomi",
        "baseUrl": "https://api.xiaomi.com/v1",
        "apiKey": "${XIAOMI_API_KEY}",
        "api": "openai-chat",
        "models": [
          {"id": "milm-v2", "name": "MiLM-V2", "contextWindow": 128000}
        ]
      },
      "deepseek-anthropic": {
        "vendor": "deepseek",
        "baseUrl": "https://api.deepseek.com/anthropic",
        "apiKey": "${DEEPSEEK_API_KEY}",
        "api": "anthropic-messages",
        "models": [...]
      },
      "local-ollama": {
        "baseUrl": "http://localhost:11434/v1",
        "api": "openai-chat",
        "models": [
          {"id": "llama3", "name": "Llama 3", "contextWindow": 8192}
        ]
      },
      "unknown-proxy": {
        "baseUrl": "https://some-proxy.example.com/v1",
        "api": "openai-chat",
        "models": [...]
      }
    },
    "defaultProvider": "deepseek-openai",
    "defaultModel": "deepseek-v4-flash"
  }
  ```
- [x] ProviderConfig 新增 `vendor` 字段 (可选):
  ```go
  type ProviderConfig struct {
      Vendor       string        `json:"vendor,omitempty"`       // 显式指定厂商适配器 (决策 12)
      APIKey       string        `json:"apiKey,omitempty"`
      BaseURL      string        `json:"baseUrl,omitempty"`
      API          string        `json:"api,omitempty"`
      // ... 其余字段不变
  }
  ```
- [x] Provider 选择优先级 (三级 fallback):
  1. `vendor` 字段显式指定 → 走对应厂商适配层
  2. 未指定 `vendor` → 通过 `baseUrl` 自动识别厂商 (如 `api.deepseek.com` → deepseek)
  3. 无法识别 → 走通用 fallback (`openai-chat` → openai_compatible, `anthropic-messages` → anthropic_compatible)
- [x] `api` 字段决定 API 格式层: `"openai-chat"` / `"anthropic-messages"`
- [x] `vendor` 字段决定厂商适配层: `"deepseek"` / `"xiaomi"` / `"claude"` / ...
- [x] 两层独立正交: 同一 vendor 可用不同 api 格式 (如 deepseek 同时支持 openai-chat 和 anthropic-messages)
- [x] 现有配置完全兼容 (vendor 字段可选，不配则自动推断)

**公共 Builder 集成**:
- [x] Builder 新增 `WithProviderByName(name string, settings *config.Settings) *Builder` 便捷方法
  - 从 settings.Providers[name] 读取 ProviderConfig
  - 三级 fallback 选择 provider:
    1. 若 config.Vendor 非空 → 查找对应厂商适配器
    2. 若 config.Vendor 为空 → 通过 config.BaseURL 自动识别 (如 `api.deepseek.com` → deepseek)
    3. 无法识别 → 根据 config.API 选择通用 fallback (openai-chat → openai_compatible, anthropic-messages → anthropic_compatible)
  - 厂商适配器可组合 API 格式层: 如 deepseek + openai-chat = DeepSeek 适配器用 OpenAI 协议但处理 reasoning model 差异
  - 开发者也可以直接 `WithProvider(myImpl)` 传入自定义实现
- [x] 新增 `baseUrlToVendor(baseURL string) string` 自动识别函数:
  - `api.deepseek.com` → `"deepseek"`
  - `api.moonshot.cn` → `"kimi"`
  - `api.minimax.chat` → `"minimax"`
  - `ark.cn-beijing.volces.com` → `"seed"`
  - `aip.baidubce.com` → `"qianfan"`
  - `dashscope.aliyuncs.com` → `"bailian"`
  - `ai.gitee.com` → `"gitee"`
  - `api.xiaomi.com` → `"xiaomi"`
  - 无法匹配 → `""` (走通用 fallback)
- [x] `make test` 通过

#### Step 1.2: Agent struct 实现接口 + ID 字段
- [x] `Config` struct 增加 `ID AgentID` 和 `ParentID AgentID` 字段
- [x] `Agent` struct 增加 `id AgentID` 和 `parentID AgentID` 字段
- [x] `New()` 和 `NewWithLoopConfig()` 自动分配 ID (若未指定)
- [x] 实现 `ID()`, `ParentID()` 方法
- [x] `make test` 通过

#### Step 1.3: Event 增加 AgentID
- [x] `Event` struct 增加 `AgentID AgentID` 字段
- [x] 新增 `emit(ch chan<- Event, event Event)` helper 方法，自动注入 AgentID
- [x] 将 `Agent.loop()` 中所有 `ch <- Event{...}` 替换为 `a.emit(ch, Event{...})`
- [x] 将 `executeSingleToolCall` 中的 `ch <- Event{...}` 同样替换
- [x] 将 `Compact` 中的 `ch <- Event{...}` 同样替换
- [x] `make test` 通过

---

### Phase 2: Registry 解耦 (Isolation) — 2-3天

#### Step 2.1: Registry 工厂化
- [x] 新增 `RegistryConfig` 结构体:
  ```go
  type RegistryConfig struct {
      WorkDir    string
      Sandbox    sandbox.Sandbox
      ToolFilter []string // optional: only register these tools
  }
  ```
- [x] 新增 `NewRegistryWithConfig(cfg RegistryConfig) *Registry`
- [x] 保留 `NewRegistry(workDir, sb)` 作为向后兼容包装 (内部调用 NewRegistryWithConfig)
- [x] 新增 `RegisterFiltered(toolNames []string)` 方法

#### Step 2.2: JobManager per-Registry
- [x] `Registry` struct 增加 `jobManager *JobManager` 字段
- [x] `Registry` 增加 `JobManager() *JobManager` getter
- [x] `RegisterDefaults()` 中创建 per-Registry JobManager 并注入到工具:
  - `BashTool` 构造函数改为 `NewBashTool(r *Registry, jm *JobManager)`
  - `JobsTool` 构造函数改为 `NewJobsTool(r *Registry, bashTool *BashTool, jm *JobManager)`
  - `KillTool` 构造函数改为 `NewKillTool(r *Registry, bashTool *BashTool, jm *JobManager)`
- [x] `make test` 通过

#### Step 2.3: Agent 创建注入 per-agent Registry
- [x] 新增 `NewWithRegistry(cfg Config, registry *tools.Registry) *Agent` 工厂方法
- [x] 内部逻辑与 `New()` 一致，区别在于接收独立 registry
- [x] `make test` 通过

---

### Phase 3: Agent 工厂 (Factory) — 1-2天

#### Step 3.1: 提取 AgentFactory
- [x] 新建 `internal/agent/factory.go`
- [x] 定义 `AgentFactory` struct:
  ```go
  type AgentFactory struct {
      provider           provider.Provider
      model              *provider.Model
      settings           *config.Settings
      sandboxMgr         *sandbox.Manager
      extraContext        string
      compactionSettings ctxpkg.CompactionSettings
      approvalHandler    func(toolCallID, toolName string, args map[string]any) bool
  }
  ```
- [x] 定义 `AgentOptions` struct:
  ```go
  type AgentOptions struct {
      ID                AgentID
      ParentID          AgentID
      Mode              string
      Model             *provider.Model
      WorkDir           string
      Tools             []string        // optional: tool filter
      SystemPromptExtra string          // extra context for this agent
      MaxIterations     int
      ToolExecutionMode string
      Session           *session.Manager
  }
  ```
- [x] 实现 `NewAgentFactory(...)` 构造函数
- [x] 实现 `Create(opts AgentOptions) Agent`:
  - 用 opts.WorkDir + factory.sandboxMgr 创建独立 Registry
  - 组装 Config
  - 调用 `NewWithRegistry()` 返回 Agent

#### Step 3.2: 迁移调用点
- [x] `cmd/vibecoding/main.go:564` — 用 factory.Create() 替换 agent.New()
- [x] `internal/tui/app.go:1133` — App 持有 factory，用 Create() 替换
- [x] `internal/acp/acp.go:584` — sessionRuntime 用 factory.Create() 替换
- [x] `make test` 通过

---

## 第二批: Phase 4-6 (管理器 + 子Agent + UI) — 第一批稳定后实施

### Phase 4: Agent 生命周期管理 (Lifecycle) — 2-3天

#### Step 4.1: AgentManager
- [x] 新建 `internal/agent/manager.go`
- [x] 实现 `AgentManager` struct:
  ```go
  type AgentManager struct {
      mu       sync.RWMutex
      agents   map[AgentID]Agent
      parentOf map[AgentID]AgentID
      children map[AgentID][]AgentID
      factory  *AgentFactory
      counter  int64
  }
  ```
- [x] 实现方法:
  - `Create(opts AgentOptions) (Agent, error)` — 创建 + 注册 + 父子关系
  - `Get(id AgentID) (Agent, bool)` — 按 ID 查询
  - `Destroy(id AgentID) error` — 停止 + 递归销毁子 Agent
  - `List() []AgentID` — 列出所有 Agent ID
  - `Children(id AgentID) []AgentID` — 查询子 Agent
  - `Parent(id AgentID) (AgentID, bool)` — 查询父 Agent

#### Step 4.2: EventRouter
- [x] 新建 `internal/agent/router.go`
- [x] 实现 `EventRouter` struct:
  ```go
  type EventRouter struct {
      mu       sync.RWMutex
      handlers map[AgentID][]EventHandler
      global   []EventHandler
  }
  ```
- [x] 实现方法:
  - `RegisterAgent(id AgentID, handler EventHandler)`
  - `UnregisterAgent(id AgentID)`
  - `RegisterGlobal(handler EventHandler)`
  - `Dispatch(event Event)` — 按 AgentID 路由到对应 handler + global handlers

---

### Phase 5: Sub-Agent 支持 (SubAgent) — 3-5天

#### Step 5.1: SubAgent 工具 (异步模式，仅主 Agent 可用)
- [x] 新建 `internal/tools/subagent.go`
- [x] 实现 4 个工具:
  - `subagent_spawn` — 主 Agent 创建并启动子 Agent，返回 handle ID
    ```json
    {
      "task": "string (required)",
      "mode": "plan|agent|yolo (default: agent)",
      "work_dir": "string (optional)",
      "tools": ["string"] (optional, tool filter),
      "max_iterations": 50,
      "system_prompt_extra": "string (optional, extra context for sub-agent)"
    }
    ```
    返回: `{ "handle": "sub-1", "status": "running" }`
  - `subagent_status` — 查询子 Agent 状态和结果
    ```json
    { "handle": "sub-1" }
    ```
    返回: `{ "status": "running|done|failed", "messages": [...], "error": "..." }`
  - `subagent_send` — 向运行中的子 Agent 发送后续指令
    ```json
    { "handle": "sub-1", "message": "now focus on..." }
    ```
  - `subagent_destroy` — 销毁子 Agent 并释放资源
    ```json
    { "handle": "sub-1" }
    ```
- [x] 子 Agent 的 Registry 中**不注册** subagent_* 工具 (禁止嵌套派生)
- [x] 子 Agent 使用独立 messages/context/session (决策 6: 完全隔离)
- [x] 子 Agent 继承 frozen prompt + dual-marker 缓存策略 (决策 7)

#### Step 5.2: 安全约束
- [x] 定义 `SubAgentPolicy`:
  ```go
  type SubAgentPolicy struct {
      MaxChildren     int           // 最大子 Agent 数 (默认 5)
      AllowedModes    []string      // 子 Agent 可用模式 (默认 ["agent"])
      InheritSandbox  bool          // 是否继承父级沙箱 (默认 true)
      TimeoutPerAgent time.Duration // 单个子 Agent 超时 (默认 10min)
      TotalTimeout    time.Duration // 所有子 Agent 总超时 (默认 30min)
  }
  ```
  注意: MaxDepth 固定为 1 (决策 5: 子 Agent 不可嵌套)，不作为可配置项
- [x] AgentManager.Create() 中集成策略检查
  - 若调用者自身是子 Agent (ParentID != "")，拒绝创建
  - 检查 MaxChildren 上限
  - 检查 AllowedModes

#### Step 5.3: 多 Agent 模式开关 (决策 8)
- [x] 新增 `--multi-agent` CLI flag (cmd/vibecoding/main.go)
- [x] TUI 中新增 `Ctrl+P` 快捷键切换多 Agent 模式
- [x] 多 Agent 模式关闭时:
  - subagent_* 工具不注册到 Registry
  - AgentManager 不创建 (或创建但限制为单 agent)
  - TUI 不显示 agent 相关命令
- [x] 多 Agent 模式开启时:
  - subagent_* 工具注册到 Registry
  - AgentManager 可用
  - TUI 显示 `/agent list|switch|destroy` 命令

#### Step 5.4: System Prompt 更新
- [x] 主 Agent system prompt 增加 Sub-Agent 使用说明段落 (仅多 Agent 模式下注入)
- [x] `make test` 通过

---

### Phase 6: TUI 多 Agent 视图 (UI) — 3-5天

#### Step 6.1: App 持有 AgentManager
- [x] `App` struct 中 `agent *agent.Agent` 改为 `agentMgr *agent.AgentManager`
- [x] 增加 `activeAgent agent.AgentID` 跟踪当前活跃 Agent
- [x] 初始创建 main agent 作为活跃 Agent

#### Step 6.2: 多 Agent 事件合并
- [x] 实现 `mergedEventChan()` — fan-in 合并所有 Agent 事件到单一 channel
- [x] 事件按 AgentID 标识来源
- [x] 非活跃 Agent 的事件缓存，切换时回放

#### Step 6.3: UI 命令
- [x] `/agent list` — 列出所有 Agent (ID, 状态, 父子关系)
- [x] `/agent switch <id>` — 切换活跃 Agent
- [x] `/agent destroy <id>` — 销毁子 Agent
- [x] 底部状态栏显示当前 Agent ID 和子 Agent 数量

---

### Phase 7: 定时任务模式 (Cron) — 2-3天

> 决策 9: `/cron` + 自然语言管理定时任务，触发时派生 subagent 执行。依赖多 Agent 模式开启。

#### Step 7.1: Cron 数据模型
- [x] 新建 `internal/cron/` 包
- [x] 定义 `CronJob` struct:
  ```go
  type CronJob struct {
      ID          string    `json:"id"`
      Name        string    `json:"name"`          // 自然语言描述的简短名称
      Prompt      string    `json:"prompt"`         // 触发时发给 subagent 的任务指令
      Schedule    string    `json:"schedule"`       // cron 表达式 (标准 5 字段)
      Mode        string    `json:"mode"`           // subagent 模式: agent/yolo
      WorkDir     string    `json:"work_dir"`       // subagent 工作目录
      Enabled     bool      `json:"enabled"`
      CreatedAt   time.Time `json:"created_at"`
      LastRun     time.Time `json:"last_run,omitempty"`
      NextRun     time.Time `json:"next_run,omitempty"`
      RunCount    int       `json:"run_count"`
      LastStatus  string    `json:"last_status,omitempty"` // success/failed/running
      LastError   string    `json:"last_error,omitempty"`
  }
  ```
- [x] 定义 `CronStore` 接口:
  ```go
  type CronStore interface {
      List() ([]CronJob, error)
      Get(id string) (*CronJob, error)
      Create(job CronJob) (*CronJob, error)
      Update(job CronJob) error
      Delete(id string) error
  }
  ```
- [x] 实现 `FileCronStore` — 持久化到 `~/.vibecoding/cron.json`

#### Step 7.2: Cron 调度器
- [x] 新建 `internal/cron/scheduler.go`
- [x] 实现 `Scheduler` struct:
  ```go
  type Scheduler struct {
      store    CronStore
      agentMgr *agent.AgentManager
      ticker   *time.Ticker
      quit     chan struct{}
  }
  ```
- [x] 实现方法:
  - `Start()` — 启动定时检查循环 (每 30 秒扫描一次)
  - `Stop()` — 停止调度器
  - `CheckAndRun()` — 检查到期任务，派生 subagent 执行
  - `ExecuteJob(job CronJob)` — 通过 AgentManager.Create() 创建 subagent，将 job.Prompt 作为任务发送
- [x] 执行完成后更新 job.LastRun / LastStatus / RunCount

#### Step 7.3: /cron TUI 命令
- [x] TUI 中新增 `/cron` 命令族 (仅多 Agent 模式下可用):
  - `/cron add <自然语言描述>` — 解析自然语言为 cron 任务
    示例: `/cron add 每天早上 9 点检查 git status 并汇报`
    内部: 调用 LLM 将自然语言转为 cron 表达式 + prompt
  - `/cron list` — 列出所有定时任务 (ID, 名称, 调度, 状态)
  - `/cron enable <id>` — 启用任务
  - `/cron disable <id>` — 禁用任务
  - `/cron remove <id>` — 删除任务
  - `/cron run <id>` — 立即手动触发一次
  - `/cron logs <id>` — 查看最近执行记录

#### Step 7.4: 自然语言解析
- [x] 利用当前 LLM Provider 将自然语言转为 cron 表达式:
  - 输入: `每天早上 9 点检查 git status`
  - LLM 输出: `{"schedule": "0 9 * * *", "prompt": "检查 git status 并汇报", "name": "每日 git 检查"}`
- [x] 若 LLM 解析失败，回退为手动输入 cron 表达式

#### Step 7.5: 集成与测试
- [x] AgentManager 启动时自动加载并启动 Scheduler
- [x] AgentManager 销毁时停止 Scheduler
- [x] 新增测试: CronStore 持久化、Scheduler 调度准确性、/cron 命令解析
- [x] `make test` 通过

---

## 文件变更总览

### 第一批新增
- `agent/types.go` — **公共包**，Agent 接口 + AgentID + Event + EventType + AgentContext + AgentConfigView
- `agent/builder.go` — **公共包**，Builder 模式创建 Agent (决策 11)
- `agent/provider.go` — **公共包**，公共 Provider 接口 + ChatParams + StreamEvent + ModelInfo + ToolDefinition + ModelCompat
- `internal/agent/factory.go` — 内部工厂 (Builder.Build() 内部调用)
- `internal/tools/registry_config.go` (或直接在 tool.go 中扩展)
- `internal/provider/registry.go` — Provider 注册表
- `internal/provider/openai_compatible/` — 通用 OpenAI 兼容 provider (处理 compat 标志位)
- `internal/provider/anthropic_compatible/` — 通用 Anthropic Messages API 兼容 provider (处理 compat 标志位)

注意: 大多数厂商 (deepseek/xiaomi/kimi/minimax/seed/qianfan/bailian/gitee) 不需要独立 provider 包，
通过 ModelCompat 标志位在通用 provider 中处理差异。仅 Google Gemini 和 Mistral 需要独立 provider 包 (API 格式不同)。

### 厂商适配参考

开发具体厂商适配时，参考以下资源:

**参考实现**: https://github.com/earendil-works/pi
- 源码目录: `/home/free/src/pi/packages/ai/src/`
- 厂商 provider: `/home/free/src/pi/packages/ai/src/providers/` — 各厂商流式实现
- 模型定义: `/home/free/src/pi/packages/ai/src/models.generated.ts` — 所有厂商的模型配置和 compat 标志
- API 注册表: `/home/free/src/pi/packages/ai/src/api-registry.ts` — provider 注册模式
- 类型定义: `/home/free/src/pi/packages/ai/src/types.ts` — Api/Provider/Model 类型

**关键文件对照**:

| 我们的包 | 参考文件 | 说明 |
|----------|----------|------|
| `internal/provider/openai_compatible/` | `providers/openai-completions.ts` | OpenAI Chat Completions 流式实现 |
| `internal/provider/anthropic_compatible/` | `providers/anthropic.ts` | Anthropic Messages 流式实现 |
| `internal/provider/google/` | `providers/google.ts` | Google Gemini 流式实现 |
| `internal/provider/mistral/` | `providers/mistral.ts` | Mistral Conversations 流式实现 |
| `config/settings.go` ModelCompat | `models.generated.ts` 的 compat 字段 | 兼容性标志定义 |
| `internal/provider/registry.go` | `api-registry.ts` | Provider 注册表模式 |

**厂商 baseUrl 自动识别参考** (来自 models.generated.ts):

| 厂商 | baseUrl | vendor 值 |
|------|---------|----------|
| DeepSeek | `api.deepseek.com` | deepseek |
| 小米 MiMo | `api.xiaomimimo.com` | xiaomi |
| Kimi | `api.moonshot.cn` | kimi |
| MiniMax | `api.minimax.chat` | minimax |
| 火山引擎 | `ark.cn-beijing.volces.com` | seed |
| 百度千帆 | `aip.baidubce.com` | qianfan |
| 阿里百炼 | `dashscope.aliyuncs.com` | bailian |
| Gitee AI | `ai.gitee.com` | gitee |
| OpenRouter | `openrouter.ai/api` | openrouter |
| Together | `api.together.xyz` | together |
| Groq | `api.groq.com` | groq |
| Fireworks | `api.fireworks.ai` | fireworks |

### 第一批修改
- `internal/agent/agent.go` — import `agent` 公共包，实现接口 + ID + emit helper
- `internal/agent/events.go` — 删除已迁移到 `agent/types.go` 的类型定义，改为 import 公共包
- `internal/tools/tool.go` — RegistryConfig + NewRegistryWithConfig + JobManager per-Registry
- `internal/tools/bash.go` — 接收 JobManager 参数
- `internal/tools/jobstool.go` — 接收 JobManager 参数
- `internal/tools/killtool.go` — 接收 JobManager 参数
- `internal/tui/app.go` — 使用 AgentFactory，import 公共 `agent` 包
- `internal/acp/acp.go` — 使用 AgentFactory，import 公共 `agent` 包
- `cmd/vibecoding/main.go` — 使用 AgentFactory，import 公共 `agent` 包

### 第二批新增
- `internal/agent/manager.go`
- `internal/agent/router.go`
- `internal/tools/subagent.go`

### 第二批修改
- `internal/tui/app.go` — AgentManager + EventRouter + UI 命令
- `internal/agent/system_prompt.go` — 增加 Sub-Agent 段落

### 第三批新增 (Cron)
- `internal/cron/cron.go` — CronJob, CronStore, FileCronStore
- `internal/cron/scheduler.go` — Scheduler 调度器

### 第三批修改 (Cron)
- `internal/tui/app.go` — /cron 命令处理
- `internal/agent/manager.go` — 启动/停止 Scheduler
- `cmd/vibecoding/main.go` — --multi-agent 时启动 Scheduler

---

## 验收标准

### 第一批完成后
- [x] `Agent` 接口定义完成，现有 `*Agent` 完全实现且通过编译
- [x] 公共 `Provider` 接口定义完成，内部 provider 可适配
- [x] Builder 模式可用: `agent.NewBuilder().WithProvider(...).Build()` 返回 Agent 接口
- [x] Builder 合理默认值: mode="agent", maxIterations=200, toolExecutionMode="parallel"
- [x] Provider 注册表可用，各厂商 provider 在 init() 中自动注册
- [x] DeepSeek 适配完成 (OpenAI 兼容但处理 reasoning model 差异)
- [x] 通用 openai_compatible fallback 可连接任意 OpenAI 兼容 API
- [x] `WithProviderByName("deepseek", cfg)` 便捷方法可用
- [x] Event 携带 AgentID，现有消费者忽略该字段，无行为变化
- [x] 每个 Agent 拥有独立 Registry + JobManager
- [x] AgentFactory 统一 3 处创建逻辑，行为与之前一致
- [x] 所有现有测试通过 (`make test`)
- [x] 新增测试: Agent 接口方法、Builder.Build()、AgentFactory.Create()、Registry 独立性、ProviderRegistry、各厂商适配

### 第二批完成后
- [x] AgentManager 支持创建/销毁/查询/父子关系
- [x] EventRouter 按 AgentID 正确路由事件
- [x] subagent_spawn/status/send/destroy 四个工具可正常工作
- [x] 子 Agent 有独立 workDir、sandbox、工具集、messages、context (决策 6: 完全隔离)
- [x] 子 Agent 继承 frozen prompt + dual-marker 缓存策略 (决策 7)
- [x] 子 Agent 的 Registry 中不包含 subagent_* 工具 (决策 5: 禁止嵌套)
- [x] 子 Agent 尝试调用 subagent_spawn 时返回错误
- [x] 多 Agent 模式默认关闭，`--multi-agent` 或 Ctrl+P 可开启 (决策 8)
- [x] 多 Agent 模式关闭时 subagent_* 工具不注册，TUI 不显示 agent 命令
- [x] TUI 支持 `/agent list|switch|destroy` 命令
- [x] 所有测试通过 + 新增 Manager/Router/SubAgent 测试

### 第三批完成后 (Cron)
- [x] `/cron add <自然语言>` 可创建定时任务
- [x] `/cron list|enable|disable|remove|run|logs` 各命令正常工作
- [x] 定时任务到期时自动派生 subagent 执行
- [x] 任务执行结果持久化到 cron.json
- [x] 任务执行完成后更新 LastRun / LastStatus / RunCount
- [x] 多 Agent 模式关闭时 /cron 命令不可用
- [x] 所有测试通过 + 新增 CronStore / Scheduler 测试

---

## 第三方开发者使用示例

公共包 `agent/` 允许外部 Go 开发者通过两种方式使用 Agent:

### 方式一: 使用内置 Agent (Builder 模式)

```go
package main

import (
    "context"
    "github.com/startvibecoding/vibecoding/agent"
)

func main() {
    // 1. 实现自己的 Provider (对接任意 LLM API)
    myProvider := NewMyProvider("my-llm", "gpt-4")

    // 2. 通过 Builder 创建内置 Agent
    a, err := agent.NewBuilder().
        WithProvider(myProvider).
        WithModel("gpt-4").
        WithMode("yolo").
        WithWorkDir("/home/user/project").
        WithMaxIterations(100).
        WithCompaction(true, 16384).
        Build()
    if err != nil {
        log.Fatal(err)
    }

    // 3. 使用 Agent 接口
    ch := a.Run(context.Background(), "列出当前目录的文件")
    for event := range ch {
        switch event.Type {
        case agent.EventTextDelta:
            fmt.Print(event.TextDelta)
        case agent.EventDone:
            fmt.Println("\n完成:", event.StopReason)
        case agent.EventError:
            fmt.Println("错误:", event.Error)
        }
    }
}
```

### 方式二: 自定义 Agent 实现

```go
package main

import (
    "context"
    "github.com/startvibecoding/vibecoding/agent"
    "github.com/startvibecoding/vibecoding/internal/provider"
)

// MyCustomAgent 自定义 Agent 实现
type MyCustomAgent struct {
    id       agent.AgentID
    messages []provider.Message
}

func (a *MyCustomAgent) ID() agent.AgentID { return a.id }
func (a *MyCustomAgent) ParentID() agent.AgentID { return "" }

func (a *MyCustomAgent) Run(ctx context.Context, userMsg string) <-chan agent.Event {
    ch := make(chan agent.Event, 100)
    go func() {
        defer close(ch)
        ch <- agent.Event{Type: agent.EventAgentStart, AgentID: a.id}
        // 自定义逻辑...
        ch <- agent.Event{Type: agent.EventDone, AgentID: a.id}
        ch <- agent.Event{Type: agent.EventAgentEnd, AgentID: a.id}
    }()
    return ch
}

// ... 实现其余接口方法
```
