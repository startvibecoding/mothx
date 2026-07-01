# /auth & /settings 重构优化方案

> 状态: 全部完成 (Phase 1-6)
> 日期: 2026-06-30
> 最后更新: 2026-07-01
> 目标版本: v0.2.0

## 1. 问题诊断

### 1.1 当前 /auth 的结构树深度问题

当前 `auth_dialog.go` 的 step 流是一条**线性序列**：

```
MainMenu → ExistingProvider/CustomID → APIKey → BaseURL → Models → Advanced → Default → Review
```

这个线性流程把不同层级的配置混在一个平面深度里：

| 层级 | 应属节点 | 当前处理方式 |
|------|----------|-------------|
| Provider 级 | apiKey, baseUrl, api, vendor, httpProxy, forceHTTP11, headers, thinkingFormat, cacheControl, responses | 混在步骤序列中，仅暴露 5 个字段 |
| Model 级 (每个模型独立) | id, name, contextWindow, maxTokens, reasoning, input, temperature, top_p, cost, compat | **全部共享一个 Advanced 页面，所有模型被统一覆盖** |
| Default 级 | defaultProvider, defaultModel | 最后一个步骤 |

**核心问题**：Advanced Parameters 页面会把用户填的 contextWindow/maxTokens 等值一次性应用到所有模型，抹杀了每个模型的独立参数。这与 `settings.json` 中 `models[]` 应有的 per-model 语义完全矛盾。

### 1.2 缺失 /settings 命令

当前 TUI 没有 `/settings` 命令。用户想查看/修改已有配置只能退出 TUI 手动编辑 `settings.json`，或重新走一遍 `/auth` 全量流程。需要补一个 `/settings` 别名作为快捷入口。

### 1.3 默认参数未透传

当用户在 `/auth` 中选择已有的厂商（如 `deepseek-openai`）时，不应使用通用模板（contextWindow=128000, maxTokens=8192），而应从 `internal/config/settings.go` 的 `defaultProviderConfigs` 中读取该厂商的**最佳默认配置**透传到 UI。这样用户注册的厂商一上来就是正确参数，无需手动校正。

### 1.4 配置参数覆盖不全

当前 /auth 仅暴露 ~10 个字段，而 `ProviderConfig` 有 11 个字段、`ModelConfig` 有 9 个字段（含 `compat` 的 14 个子字段）。重构后 /auth 和 /settings 必须覆盖 `ProviderConfig` + `ModelConfig` 的**全部字段**。

---

## 2. 设计方案

### 2.1 配置层级（正确的递进关系）

```
Settings (global)
├── defaultProvider / defaultModel / defaultMode / defaultThinkingLevel
├── providers
│   └── <providerId> (ProviderConfig)     ← Provider 层 (11 个字段全覆盖)
│       ├── apiKey, baseUrl, api, vendor
│       ├── httpProxy, forceHTTP11
│       ├── headers, thinkingFormat
│       ├── cacheControl, responses
│       └── models[]                      ← Model 层，每个独立 (9 个字段全覆盖)
│           └── <modelId> (ModelConfig)
│               ├── id, name
│               ├── contextWindow, maxTokens, reasoning, input
│               ├── temperature, top_p
│               ├── cost (input/output/cacheRead/cacheWrite)
│               └── compat (14 个布尔/枚举子字段)
├── compaction, sandbox, webSearch...     ← 全局功能层
```

### 2.2 默认透传机制

#### 2.2.1 三层配置合并

当用户选择已有厂商或输入 custom provider ID 时，按以下优先级构建初始状态：

```
优先级 1 (最高): settings.json 中用户已有的运行时配置
优先级 2        : config/defaultProviderConfigs 中的内置厂商默认
优先级 3 (最低): 通用保守默认值
```

```go
// resolveProviderDefaults 合并配置
func resolveProviderDefaults(providerID string, existing *config.ProviderConfig) *config.ProviderConfig {
    // 1. 以 defaultProviderConfigs[providerID] 为基础（如果是已知厂商）
    base := config.DefaultProviderConfig(providerID)  // 从 settings.go 导出
    
    // 2. 如果用户在 settings.json 已有配置，覆盖上去
    if existing != nil {
        base = mergeProviderConfig(base, existing)
    }
    
    // 3. 未知 custom provider 使用 safe defaults
    if base == nil {
        base = &config.ProviderConfig{
            API: "openai-chat",
            BaseURL: "",
            Models: []config.ModelConfig{},
        }
    }
    return base
}
```

#### 2.2.2 已知厂商 vs 自定义厂商的行为差异

| 场景 | Provider 层初始值 | Model 层初始值 |
|------|-------------------|----------------|
| `/auth` → Existing → `deepseek-openai` | 使用 `defaultProviderConfigs["deepseek-openai"]` 的 apiKey/api/vendor/httpProxy 等全部字段 | 使用该配置的 models[] 完整参数（含 cost、compat） |
| `/auth` → Custom → `local-llama` | apiKey 为空、api=openai-chat、baseUrl 为空、其余为 zero value | 空列表，用户 Add Model 时使用通用模板 |
| `/auth` → Custom → `longcat` (ID 巧合与内置同名) | **不**使用 longcat 内置默认，因为是 custom 模式 | 空列表 (custom 模式不假设用户想用内置 longcat) |

设计原则：只有 "**Existing Providers**" 路径才透传内置默认；"**Custom Provider**" 路径一律使用通用模板，避免混淆。

#### 2.2.3 config 包需要导出的 helper

```go
// internal/config/settings.go

// DefaultProviderConfigs returns a deep copy of the built-in provider presets.
func DefaultProviderConfigs() map[string]*ProviderConfig { ... }

// DefaultProviderConfig returns a deep copy of a single provider preset, or nil.
func DefaultProviderConfig(providerID string) *ProviderConfig { ... }

// DefaultModelConfig returns the preset for a specific model under a provider, or nil.
func DefaultModelConfig(providerID, modelID string) *ModelConfig { ... }

// ResolveProviderConfig returns the effective config for a provider,
// merging runtime settings.json over built-in defaults.
func ResolveProviderConfig(providerID string, runtime *Settings) *ProviderConfig { ... }
```

### 2.3 /settings 命令

`/settings` 直接弹出当前已配置的 Provider 列表（复用 /auth 的 Existing Providers 视图），选择后进入 **Settings Detail 编辑器**，按层级展示：

```
┌─ Settings · deepseek-openai ────────────────────────────────┐
│ › Provider Settings                                          │
│   apiKey, baseUrl, api, vendor, proxy...                     │
│                                                              │
│   Models                                                     │
│   ├ deepseek-v4-flash   1000000 · 384000 · reasoning · text │
│   └ deepseek-v4-pro     1000000 · 384000 · reasoning · text │
│                                                              │
│   + Add Model                                                │
│   ⚙ Set as Default                                           │
│                                                              │
│ Enter to edit, ↑↓ to navigate, Esc to close                  │
└──────────────────────────────────────────────────────────────┘
```

**行为**：
- `/settings` 无参数 → 打开 Provider 列表（同 /auth 的 Existing Providers）
- `/settings <providerId>` → 直接进入该 Provider 的 Settings Detail
- 选中 Provider 后展示分层菜单：Provider Settings / 每个 Model / Add Model / Default
- Provider Settings 内按 "组" 组织（见 §2.5）
- 编辑后走同一套 sparse save + provider factory 验证 + 热切换逻辑

### 2.4 /auth 重构

#### 2.4.1 新 flow 结构（树深度从 1 变为 2+）

```
/auth
  → Main Menu
      ├─ Existing Providers → [Provider 列表]
      │     → Provider Group List（按组分: 认证/协议/网络/高级）
      │         ├─ Group A: Credentials → API Key, Vendor
      │         ├─ Group B: Protocol  → Base URL, API Type, Responses
      │         ├─ Group C: Network   → HTTP Proxy, Force HTTP/1.1
      │         └─ Group D: Advanced  → Headers, Thinking Format, Cache Control
      │     → Models（进入 Model 子树）
      │             ├─ [Model 1] Group List（基础/能力/采样/Cost/Compat）
      │             ├─ [Model 2] Group List
      │             ├─ Add Model → 输入 ID → Name → 该 Model 的 Group List
      │             └─ Done → Default → Review
      │
      └─ Custom Provider
          ├─ Provider ID
          ├─ API Type
          ├─ Provider Group List（同 Above）
          └─ ...
```

#### 2.4.2 关键改动：per-model 全字段 params

**当前问题**：`authDialogState` 只有一组 `ContextWindow/MaxTokens/Reasoning/.../TopP` 字段，所有模型共用，且缺失 cost/compat/name。

**新方案**：引入 `modelEditState`，以 `modelId` 为 key：

```go
// modelEditState holds full ModelConfig fields for one model being edited.
// The Edited* flags track explicit user changes so we preserve untouched values.
type modelEditState struct {
    // Identity
    ID   string
    Name string
    
    // Capabilities
    ContextWindow int
    MaxTokens     int
    Reasoning     bool
    Input         []string  // e.g. ["text", "image"]
    
    // Sampling
    Temperature *float64
    TopP        *float64
    
    // Cost
    CostInput      float64
    CostOutput     float64
    CostCacheRead  float64
    CostCacheWrite float64
    CostEnabled    bool  // whether cost tracking is active
    
    // Compat (all 14 fields)
    ThinkingFormat                              string
    RequiresReasoningContentOnAssistant         bool
    RequiresReasoningContentOnAssistantMessages bool
    ForceAdaptiveThinking                       bool
    ParseReasoningInContent                     bool
    SupportsDeveloperRole                       *bool
    SupportsStore                               *bool
    SupportsReasoningEffort                     *bool
    SupportsStrictMode                          *bool
    MaxTokensField                              string
    SupportsCacheControlOnTools                 *bool
    SupportsLongCacheRetention                  *bool
    SupportsPromptCacheKey                      *bool
    SupportsReasoningSummary                    *bool
    SendSessionAffinityHeaders                  bool
    SupportsEagerToolInputStreaming             *bool
    
    // Edited flags (one per field)
    NameEdited     bool
    ContextWindowEdited bool
    MaxTokensEdited     bool
    ReasoningEdited     bool
    InputEdited         bool
    TemperatureEdited   bool
    TopPEdited          bool
    CostEdited          bool
    CompatEdited        bool
}

type providerEditState struct {
    APIKey         string
    BaseURL        string
    API            string
    Vendor         string
    HTTPProxy      string
    ForceHTTP11    bool
    Headers        map[string]string  // key-value pairs
    ThinkingFormat string
    CacheControl   *bool
    
    // Responses sub-config
    ResponsesReasoningSummary     string  // "auto" | "concise" | "detailed"
    ResponsesPromptCacheEnabled   *bool
    ResponsesPromptCacheKey       string
    ResponsesPromptCacheRetention string
    
    // Edited flags
    APIKeyEdited, BaseURLEdited, APIEdited, VendorEdited bool
    HTTPProxyEdited, ForceHTTP11Edited, HeadersEdited   bool
    ThinkingFormatEdited, CacheControlEdited            bool
    ResponsesEdited                                    bool
}

type authDialogState struct {
    // ... existing navigation fields ...
    
    Provider providerEditState
    Models   map[string]*modelEditState  // keyed by model ID
    ModelOrder []string  // insertion order for stable rendering
    
    // ...
}
```

#### 2.4.3 新增 authView 步骤

```go
const (
    authViewMain.authView = iota
    authViewExistingProvider
    authViewCustomID
    authViewAPI
    authViewBaseURLChoice
    authViewBaseURL
    // === 新增 ===
    authViewProviderGroupList    // Provider 设置组选择
    authViewProviderFieldEdit    // Provider 组内字段编辑
    authViewAuthEdit             // API Key + Vendor 编辑
    authViewProtocolEdit         // Base URL + API Type + Responses 编辑
    authViewNetworkEdit          // HTTP Proxy + Force HTTP/1.1 编辑
    authViewAdvancedProviderEdit // Headers + Thinking Format + Cache Control 编辑
    authViewHeadersEdit          // headers map 编辑器
    authViewResponsesEdit        // responses 子表单
    // === Model 层 ===
    authViewModelList            // Model 列表
    authViewModelGroupList       // 单个 Model 的组选择
    authViewModelFieldEdit       // Model 组内字段编辑
    authViewModelBasicsEdit      // ID, Name, ContextWindow, MaxTokens
    authViewModelCapabilityEdit  // Reasoning, Input modalities
    authViewModelSamplingEdit    // Temperature, TopP
    authViewModelCostEdit        // Cost input/output/cacheRead/cacheWrite
    authViewModelCompatEdit      // 14 个 compat 子字段
    authViewAddModel             // 输入新模型 ID
    // === 已有 ===
    authViewHTTPProxy
    authViewForceHTTP11
    authViewAPIKey
    authViewModels
    authViewAdvanced      // 过渡期保留，Phase 6 移除
    authViewParamInput
    authViewDefault
    authViewReview
    authViewEditMenu
    // === /settings ===
    authViewSettingsDetail  // /settings 专属入口视图
)
```

### 2.5 Provider 字段分组（Group List 模式）

将 ProviderConfig 的 11 个字段分为 4 个逻辑组，避免单一冗长列表：

| Group | 标题 | 字段 | 当前已暴露 |
|-------|------|------|-----------|
| A | Credentials (认证) | apiKey, vendor | apiKey ✅ vendor ❌ |
| B | Protocol (协议) | baseUrl, api, responses | baseUrl ✅ api ✅ responses ❌ |
| C | Network (网络) | httpProxy, forceHTTP11 | ✅ ✅ |
| D | Advanced (高级) | headers, thinkingFormat, cacheControl | ❌ ❌ ❌ |

`authViewProviderGroupList` 渲染：

```
┌─ Provider · deepseek-openai · Settings ──────────────────────┐
│ › A. Credentials             apiKey=...Seek  vendor=         │
│   B. Protocol                api=openai-chat                  │
│   C. Network                 proxy=none  http/2               │
│   D. Advanced                headers=0  thinking=  cache=     │
│                                                              │
│   Models ──────────────────────────────────────────────────▶  │
│                                                              │
│ Enter select, ↑↓ navigate, Esc back                          │
└──────────────────────────────────────────────────────────────┘
```

点击每组进入该组的字段列表编辑：

```
┌─ Provider · Credentials ─────────────────────────────────────┐
│ › API Key ................... ${DEEPSEEK_API_KEY}             │
│   Vendor .................... (auto-detect)                  │
│                                                              │
│ ✓ Done                                                       │
│                                                              │
│ Edit field or Done to confirm group                          │
└──────────────────────────────────────────────────────────────┘
```

### 2.6 Model 字段分组（Group List 模式）

将 ModelConfig 的 9 个顶层字段分为 5 个逻辑组：

| Group | 标题 | 字段 | 当前已暴露 |
|-------|------|------|-----------|
| A | Basics (基础) | id, name, contextWindow, maxTokens | id ✅ name ❌ ctx ✅ max ✅ |
| B | Capabilities (能力) | reasoning, input | ✅ ✅ |
| C | Sampling (采样) | temperature, topP | ❌ ❌ |
| D | Cost (计费) | cost.input, cost.output, cost.cacheRead, cost.cacheWrite | ❌ |
| E | Compatibility (兼容) | compat.* (14 子字段) | ❌ |

`authViewModelGroupList` 渲染：

```
┌─ Model: deepseek-v4-flash · Parameters ──────────────────────┐
│ › A. Basics         ctx=1000000  max=384000                  │
│   B. Capabilities   reasoning  input=text                    │
│   C. Sampling       temp=none  top_p=none                    │
│   D. Cost           in=0.14  out=0.28  cr=0.0028  cw=none   │
│   E. Compatibility  format=  flags=0 active                 │
│                                                              │
│ ✓ Done     ✏ Edit Group                                      │
│                                                              │
│ ▶ Expand all for full JSON                                  │
└──────────────────────────────────────────────────────────────┘
```

点击 E (Compatibility) 进入 `authViewModelCompatEdit` — 14 个子字段的分组编辑：

```
┌─ Model: deepseek-v4-flash · Compatibility ───────────────────┐
│ › Thinking Format ........... (none)                         │
│   Req.ReasoningContent ..Asst disabled                       │
│   Req.ReasoningContent ..AsstMsgs disabled                   │
│   Force Adaptive Thinking ... disabled                       │
│   Parse ReasoningInContent ... disabled                      │
│   ─ API Params ───────────────────────────────────────────   │
│   Supports Developer Role ... (auto)                         │
│   Supports Store ......... (auto)                            │
│   Supports ReasoningEffort  (auto)                           │
│   Supports Strict Mode ... (auto)                            │
│   Max Tokens Field ........ (default)                        │
│   ─ Cache ────────────────────────────────────────────────   │
│   CacheControlOnTools .... (auto)                            │
│   LongCacheRetention ..... (auto)                            │
│   PromptCacheKey ......... (auto)                            │
│   ReasoningSummary ....... (auto)                            │
│   ─ Streaming ───────────────────────────────────────────   │
│   SessionAffinityHeaders . disabled                          │
│   EagerToolStreaming ..... (auto)                            │
│                                                              │
│ ✓ Done     ✏ Reset to defaults                               │
└──────────────────────────────────────────────────────────────┘
```

说明：
- `(auto)` 表示该 bool 为 nil (未设置，走 provider 自动推断)
- `disabled` / `enabled` 表示用户显式设置的 false / true
- Reset to defaults 可将 compat 全部清回 nil

### 2.7 Add Model 流程（使用内置默认透传）

```
+ Add Model
  → 输入 ID: deepseek-v4-pro
  → 系统检查 defaultProviderConfigs[providerID].Models 中是否有同名模型
     ├─ 存在 → 自动填充全部参数（ctx/max/reasoning/input/temp/top_p/cost/compat）
     └─ 不存在 → 使用通用模板 (ctx=128000, max=8192, input=text)
  → 进入 Model Group List（可直接 Done 确认或修改任意组）
  → 回到 Models 列表
```

---

## 3. 对比：当前 vs 新方案

| 维度 | 当前 | 新方案 |
|------|------|--------|
| 配置字段覆盖 | ~10 个（Provider 5 + Model 5） | **全部** Provider(11) + Model(9) + Compat(14) |
| 模型参数 | 所有模型共享一组高级参数 | 每个模型独立维护全字段参数 |
| 结构深度 | 线性单深度，所有步骤平铺 | 树形 3 层：Provider → Groups → Models → Groups |
| 已有厂商默认 | 通用保守值（128000/8192/text） | **透传内置最佳配置**（含 cost/compat） |
| 新增模型默认 | 通用保守值 | **从内置 models[] 同名匹配**，否则通用模板 |
| /settings | 不存在 | 快捷查看/编辑已有配置 + 全字段覆盖 |
| Headers | ❌ 不支持 | ✅ key-value 编辑器 |
| Thinking Format | ❌ /auth 中不暴露 | ✅ Provider 高级组 |
| Cache Control | ❌ 不支持 | ✅ Provider 高级组 |
| Responses | ❌ 不支持 | ✅ Protocol 组 |
| Cost | ❌ 不支持 | ✅ Model Cost 组 |
| Compat | ❌ 不支持 | ✅ Model Compatibility 组（14 子字段） |
| Display Name | ❌ 不可编辑 | ✅ Model Basics 组 |

---

## 4. 完整配置结构对应关系

### 4.1 Provider 层（ProviderConfig — 11 个字段全覆盖）

| 字段 | JSON Key | 类型 | 所属 Group | UI 控件 |
|------|----------|------|-----------|---------|
| APIKey | `apiKey` | string | Credentials | 文本输入（支持 `${ENV}`） |
| BaseURL | `baseUrl` | string | Protocol | 文本输入或选择预设 |
| API | `api` | string | Protocol | 枚举选择 |
| Vendor | `vendor` | string | Credentials | 文本输入（可为空=auto-detect） |
| HTTPProxy | `httpProxy` | string | Network | 文本输入 |
| ForceHTTP11 | `forceHTTP11` | bool | Network | 切换 |
| Headers | `headers` | map[string]string | Advanced | map 编辑器 |
| ThinkingFormat | `thinkingFormat` | string | Advanced | 枚举选择（`""`/`openai`/`anthropic`/`deepseek`/`xiaomi`/`zai`） |
| CacheControl | `cacheControl` | *bool | Advanced | 三态切换（nil/true/false） |
| Responses | `responses` | struct | Protocol | 子表单 |
| Models | `models` | []ModelConfig | — | Model 子树 |

### 4.2 Responses 子表单（ResponsesConfig — 4 字段）

| 字段 | JSON Key | 类型 | UI 控件 |
|------|----------|------|---------|
| ReasoningSummary | `reasoningSummary` | string | 枚举（`auto`/`concise`/`detailed`） |
| PromptCacheEnabled | `promptCacheEnabled` | *bool | 三态 |
| PromptCacheKey | `promptCacheKey` | string | 文本输入 |
| PromptCacheRetention | `promptCacheRetention` | string | 文本输入 |

### 4.3 Model 层（ModelConfig — 9 个顶层字段全覆盖）

| 字段 | JSON Key | 类型 | 所属 Group | UI 控件 |
|------|----------|------|-----------|---------|
| ID | `id` | string | Basics | 文本输入（首次输入后只读） |
| Name | `name` | string | Basics | 文本输入（默认=id） |
| ContextWindow | `contextWindow` | int | Basics | 整数输入 |
| MaxTokens | `maxTokens` | int | Basics | 整数输入 |
| Reasoning | `reasoning` | bool | Capabilities | 切换 |
| Input | `input` | []string | Capabilities | 多选（text/image/audio/video/pdf） |
| Temperature | `temperature` | *float64 | Sampling | 浮点输入或空=auto |
| TopP | `top_p` | *float64 | Sampling | 浮点输入或空=auto |
| Cost | `cost` | struct | Cost | 子表单 |
| Compat | `compat` | struct | Compatibility | 子表单（14 子字段） |

### 4.4 Cost 子表单（CostConfig — 4 字段）

| 字段 | JSON Key | 类型 | UI 控件 |
|------|----------|------|---------|
| Input | `input` | float64 | 浮点（每百万 token 价格） |
| Output | `output` | float64 | 浮点 |
| CacheRead | `cacheRead` | float64 | 浮点 |
| CacheWrite | `cacheWrite` | float64 | 浮点 |

### 4.5 Compat 子表单（ModelCompat — 14 字段）

| 字段 | JSON Key | 类型 | 分类 |
|------|----------|------|------|
| ThinkingFormat | `thinkingFormat` | string | Thinking |
| RequiresReasoningContentOnAssistant | `requiresReasoningContentOnAssistant` | bool | Thinking |
| RequiresReasoningContentOnAssistantMessages | `requiresReasoningContentOnAssistantMessages` | bool | Thinking |
| ForceAdaptiveThinking | `forceAdaptiveThinking` | bool | Thinking |
| ParseReasoningInContent | `parseReasoningInContent` | bool | Thinking |
| SupportsDeveloperRole | `supportsDeveloperRole` | *bool | API Params |
| SupportsStore | `supportsStore` | *bool | API Params |
| SupportsReasoningEffort | `supportsReasoningEffort` | *bool | API Params |
| SupportsStrictMode | `supportsStrictMode` | *bool | API Params |
| MaxTokensField | `maxTokensField` | string | API Params |
| SupportsCacheControlOnTools | `supportsCacheControlOnTools` | *bool | Cache |
| SupportsLongCacheRetention | `supportsLongCacheRetention` | *bool | Cache |
| SupportsPromptCacheKey | `supportsPromptCacheKey` | *bool | Cache |
| SupportsReasoningSummary | `supportsReasoningSummary` | *bool | Cache |
| SendSessionAffinityHeaders | `sendSessionAffinityHeaders` | bool | Streaming |
| SupportsEagerToolInputStreaming | `supportsEagerToolInputStreaming` | *bool | Streaming |

---

## 5. 用户体验详情

### 5.1 /settings 流程

```
/settings
  → Provider 列表
  → 选择 Provider → Settings Detail (authViewSettingsDetail)
      ├─ Provider Settings → Provider Group List (A/B/C/D 四组)
      │     ├─ Group A Credentials → API Key, Vendor
      │     ├─ Group B Protocol → Base URL, API Type, Responses
      │     ├─ Group C Network → HTTP Proxy, Force HTTP/1.1
      │     └─ Group D Advanced → Headers, Thinking Format, Cache Control
      │
      ├─ Model: <model-id> → Model Group List (A/B/C/D/E 五组)
      │     ├─ Group A Basics → Name, ContextWindow, MaxTokens
      │     ├─ Group B Capabilities → Reasoning, Input
      │     ├─ Group C Sampling → Temperature, TopP
      │     ├─ Group D Cost → Input/Output/CacheRead/CacheWrite
      │     └─ Group E Compatibility → 14 子字段
      │
      ├─ + Add Model
      └─ ⚙ Set as Default
  → Review (全字段 JSON，折叠 cost/compat) → Save
```

### 5.2 /auth 完整流程（Existing Provider）

```
/auth → Existing → deepseek-openai
  → [系统透传 defaultProviderConfigs["deepseek-openai"] 全部字段]
  → Provider Group List:
      A. Credentials   → API Key: ${DEEPSEEK_API_KEY}  Vendor: (empty=auto)
      B. Protocol      → Base URL: https://api.deepseek.com  API: openai-chat  Responses: >
      C. Network       → HTTP Proxy: (none)  Force HTTP/1.1: No
      D. Advanced      → Headers: (empty)  Thinking Format: (empty)  Cache Control: (auto)
  → Models:
      ┌─ Models ──────────────────────────────────────────────────┐
      │ › deepseek-v4-flash                                        │
      │   1000000·384000·reasoning·text·in=0.14·out=0.28           │
      │                                                            │
      │   deepseek-v4-pro                                          │
      │   1000000·384000·reasoning·text·in=0.435·out=0.87          │
      │                                                            │
      │   + Add Model                                              │
      │   ✓ Done                                                   │
      └────────────────────────────────────────────────────────────┘
  → 选择 deepseek-v4-flash → Model Group List → (编辑任意组) → Done
  → Set Default → Review (全字段 JSON 预览) → Save
```

### 5.3 /auth 完整流程（Custom Provider）

```
/auth → Custom
  → Provider ID: local-llm
  → API Type: openai-chat
  → Base URL: http://localhost:8080/v1
  → HTTP Proxy: (none)
  → Force HTTP/1.1: No
  → API Key: (empty or test-key)
  → Headers: (none)
  → Thinking Format: (empty)
  → Cache Control: (auto)
  → Responses Reasoning Summary: auto
  → + Add Model
      → ID: qwen3-72b
      → [未找到同名内置模板，使用通用默认]
      → Model Group List:
          A. Basics        → Name: Qwen3-72B  ContextWindow: 128000  MaxTokens: 8192
          B. Capabilities  → Reasoning: off  Input: text
          C. Sampling      → Temp: (auto)  TopP: (auto)
          D. Cost          → in=0 out=0 cr=0 cw=0
          E. Compatibility → (none)
      → Done
  → ✓ All Models Confirmed
  → Set Default → Review → Save
```

---

## 6. 数据结构变更

### 6.1 authDialogState 扩展

```go
type authDialogState struct {
    Open bool
    View authView
    Cursor int
    Stack []authView
    Mode string // existing/custom

    ProviderID string
    Search     string
    SetDefault bool

    // === 新增：结构化编辑状态 ===
    Provider   providerEditState
    Models     map[string]*modelEditState  // keyed by model ID
    ModelOrder []string                    // stable iteration order

    // === 当前编辑上下文（指向 Models map 中的当前项）===
    CurrentModelID  string
    CurrentGroup    string  // "A"/"B"/"C"/"D"/"E" 视图上下文
    CurrentField    string  // 当前正在编辑的字段名

    // === 老的扁平字段（过渡期保留，最终移除）===
    API                  
    BaseURL              
    HTTPProxy            
    ForceHTTP11          
    APIKey               
    ModelIDs             
    ContextWindow        
    MaxTokens            
    Reasoning            
    InputTypes           
    Temperature          
    TopP                 
    ParamField           
    // ... edited flags ...

    Error   string
    Preview string
}
```

### 6.2 providerEditState（新增）

```go
type providerEditState struct {
    APIKey         string
    BaseURL        string
    API            string
    Vendor         string
    HTTPProxy      string
    ForceHTTP11    bool
    Headers        map[string]string
    ThinkingFormat string
    CacheControl   *bool

    Responses struct {
        ReasoningSummary     string
        PromptCacheEnabled   *bool
        PromptCacheKey       string
        PromptCacheRetention string
    }
}
```

### 6.3 modelEditState（新增）

```go
type modelEditState struct {
    // === Identity ===
    ID   string
    Name string

    // === Capabilities ===
    ContextWindow int
    MaxTokens     int
    Reasoning     bool
    Input         []string

    // === Sampling ===
    Temperature *float64
    TopP        *float64

    // === Cost ===
    Cost struct {
        Input      float64
        Output     float64
        CacheRead  float64
        CacheWrite float64
    }
    CostEnabled bool  // whether to write cost at all

    // === Compatibility (全 14 字段) ===
    Compat struct {
        ThinkingFormat                              string
        RequiresReasoningContentOnAssistant         bool
        RequiresReasoningContentOnAssistantMessages bool
        ForceAdaptiveThinking                       bool
        ParseReasoningInContent                     bool
        SupportsDeveloperRole                       *bool
        SupportsStore                               *bool
        SupportsReasoningEffort                     *bool
        SupportsStrictMode                          *bool
        MaxTokensField                              string
        SupportsCacheControlOnTools                 *bool
        SupportsLongCacheRetention                  *bool
        SupportsPromptCacheKey                      *bool
        SupportsReasoningSummary                    *bool
        SendSessionAffinityHeaders                  bool
        SupportsEagerToolInputStreaming             *bool
    }
}
```

### 6.4 默认透传辅助函数

```go
// internal/config/settings.go

// DefaultProviderConfigs 返回内置厂商默认配置的深拷贝。
func DefaultProviderConfigs() map[string]*ProviderConfig {
    return cloneProviderConfigs(defaultProviderConfigs)
}

// DefaultProviderConfig 返回单个内置厂商默认配置的深拷贝。未知名商返回 nil。
func DefaultProviderConfig(id string) *ProviderConfig {
    src := defaultProviderConfigs[id]
    if src == nil { return nil }
    return cloneProviderConfig(src)
}

// DefaultModelConfig 返回某厂商下某模型的内置默认配置。
func DefaultModelConfig(providerID, modelID string) *ModelConfig {
    pc := defaultProviderConfigs[providerID]
    if pc == nil { return nil }
    for _, m := range pc.Models {
        if m.ID == modelID {
            cm := cloneModelConfig(m)
            return &cm
        }
    }
    return nil
}

// ResolveProviderConfig 合并运行时配置和内置默认。
func ResolveProviderConfig(id string, runtime *Settings) *ProviderConfig {
    base := DefaultProviderConfig(id)
    if base == nil {
        base = &ProviderConfig{API: "openai-chat"}
    }
    if runtime != nil {
        if existing, ok := runtime.Providers[id]; ok && existing != nil {
            base = mergeProviderConfig(base, existing)
        }
    }
    return base
}

// mergeProviderConfig 将 overlay 的非零字段合并到 base。
// nil *bool 视为未设置（不覆盖），非 nil 视为显式值。
func mergeProviderConfig(base, overlay *ProviderConfig) *ProviderConfig { ... }
```

---

## 7. 渲染新增

### 7.1 Provider Group List (authViewProviderGroupList)

```
┌─ Provider · deepseek-openai · Settings ──────────────────────┐
│ › A. Credentials .............. API Key, Vendor              │
│   B. Protocol ................. api, Base URL, Responses      │
│   C. Network .................. Proxy, HTTP/1.1              │
│   D. Advanced ................. Headers, ThinkingFmt, Cache   │
│                                                              │
│   ─────────────────────────────────────────────────────────   │
│   Models ▶                                                   │
│                                                              │
│ Enter select, ↑↓ navigate, Esc back                          │
└──────────────────────────────────────────────────────────────┘
```

### 7.2 Provider Sub-Form（如 Group B Protocol）

```
┌─ Provider · Protocol ────────────────────────────────────────┐
│ › API Type .................... openai-chat                  │
│   Base URL .................... https://api.deepseek.com     │
│   Responses ▶ ...................                            │
│     ReasoningSummary .......... auto                         │
│     PromptCacheEnabled ........ (auto)                       │
│                                                              │
│ ✓ Done                                                       │
│                                                              │
│ Edit or Done to confirm                                      │
└──────────────────────────────────────────────────────────────┘
```

### 7.3 Model Group List (authViewModelGroupList)

```
┌─ Model: deepseek-v4-flash · Parameters ──────────────────────┐
│ › A. Basics .... ctx=1000000  max=384000  name=Flash        │
│   B. Capabilities  reasoning  input=text                     │
│   C. Sampling ...... temp=none  top_p=none                   │
│   D. Cost .......... in=0.14  out=0.28  cr=0.0028  cw=none   │
│   E. Compatibility . (none active)                           │
│                                                              │
│ ✓ Done     ✏ Expand all                                      │
│                                                              │
│ ▶ Full JSON                                                  │
└──────────────────────────────────────────────────────────────┘
```

### 7.4 Headers 编辑器 (authViewHeadersEdit)

```
┌─ Provider · Headers ─────────────────────────────────────────┐
│ › + Add header                                               │
│                                                              │
│   (empty — no custom headers)                                │
│                                                              │
│ ✓ Done                                                       │
│                                                              │
│ Add header or Done to confirm                                │
└──────────────────────────────────────────────────────────────┘
```

点击 Add header:

```
┌─ Add Header ─────────────────────────────────────────────────┐
│ Key: [.............................]                          │
│ Value: [...........................]                         │
│                                                              │
│ Enter to add another, Esc to finish                          │
└──────────────────────────────────────────────────────────────┘
```

### 7.5 Review 全字段预览

```
┌─ Review ─────────────────────────────────────────────────────┐
│ Will save to settings.json:                                  │
│ {                                                            │
│   "defaultProvider": "deepseek-openai",                       │
│   "defaultModel": "deepseek-v4-flash",                       │
│   "providers": {                                             │
│     "deepseek-openai": {                                     │
│       "apiKey": "****Seek",                                  │
│       "baseUrl": "https://api.deepseek.com",                 │
│       "api": "openai-chat",                                  │
│       "models": [                                            │
│         {                                                    │
│           "id": "deepseek-v4-flash",                         │
│           "name": "DeepSeek V4 Flash",                       │
│           "reasoning": true,                                 │
│           "contextWindow": 1000000,                          │
│           "maxTokens": 384000,                               │
│           "input": ["text"],                                 │
│           ▶ "cost": { ... },                                 │
│           ▶ "compat": { ... }                                │
│         },                                                   │
│         ...                                                  │
│       ]                                                      │
│     }                                                        │
│   }                                                          │
│ }                                                            │
│                                                              │
│ Enter save, Esc back, ▶ expand cost/compat                   │
└──────────────────────────────────────────────────────────────┘
```

---

## 8. 键盘映射

| 上下文 | 按键 | 行为 |
|--------|------|------|
| Provider Group List | Enter | 进入所选 Group 的子表单 |
| Provider Group List | Enter on Models ▶ | 进入 Model 列表 |
| Provider Sub-Form | Enter | 编辑当前字段 |
| Provider Sub-Form | Enter on Done | 确认，返回 Group List |
| Model Group List | Enter | 进入所选 Group 的子表单 |
| Model Group List | Enter on Done | 确认所有模型，进入 Default |
| Model Sub-Form | Enter | 编辑当前字段值 |
| Model Sub-Form | Space | 切换 bool（三态：nil→true→false→nil） |
| Model Sub-Form | Enter on Done | 保存本模型，返回 Group List |
| Add Model input | Enter | 创建模型实例，进入其 Group List |
| Headers Edit | Enter on Add | 进入 Key/Value 输入 |
| Headers Edit | Backspace on row | 删除该 header |
| Review | ▶ on collapsed | 展开 cost/compat 全文 |
| Settings Detail | Enter on Provider | 进入 Provider Group List |
| Settings Detail | Enter on Model | 进入该 Model Group List |

---

## 9. 保存逻辑

### 9.1 buildAuthSettings 全字段写入

```go
func (a *App) buildAuthSettingsFrom(base *config.Settings) (*config.Settings, string) {
    next := cloneSettings(base)
    
    pc := next.Providers[a.auth.ProviderID]
    if pc == nil {
        pc = &config.ProviderConfig{}
        next.Providers[a.auth.ProviderID] = pc
    }
    
    // Provider 层全字段写入
    pc.APIKey = a.auth.Provider.APIKey
    pc.BaseURL = a.auth.Provider.BaseURL
    pc.API = a.auth.Provider.API
    pc.Vendor = a.auth.Provider.Vendor
    pc.HTTPProxy = a.auth.Provider.HTTPProxy
    pc.ForceHTTP11 = a.auth.Provider.ForceHTTP11
    pc.Headers = cloneStringMap(a.auth.Provider.Headers)
    pc.ThinkingFormat = a.auth.Provider.ThinkingFormat
    pc.CacheControl = cloneBoolPtr(a.auth.Provider.CacheControl)
    pc.Responses = cloneResponsesConfig(a.auth.Provider.Responses)
    
    // Models 全字段写入
    pc.Models = make([]config.ModelConfig, 0, len(a.auth.ModelOrder))
    for _, id := range a.auth.ModelOrder {
        me := a.auth.Models[id]
        if me == nil { continue }
        mc := config.ModelConfig{
            ID:            me.ID,
            Name:          me.Name,
            ContextWindow: me.ContextWindow,
            MaxTokens:     me.MaxTokens,
            Reasoning:     me.Reasoning,
            Input:         cloneStringSlice(me.Input),
            Temperature:   cloneFloat64Ptr(me.Temperature),
            TopP:          cloneFloat64Ptr(me.TopP),
        }
        if me.CostEnabled {
            mc.Cost = &config.CostConfig{
                Input:      me.Cost.Input,
                Output:     me.Cost.Output,
                CacheRead:  me.Cost.CacheRead,
                CacheWrite: me.Cost.CacheWrite,
            }
        }
        mc.Compat = buildCompatFromEditState(&me.Compat)
        pc.Models = append(pc.Models, mc)
    }
    
    modelID := ""
    if len(a.auth.ModelOrder) > 0 {
        modelID = a.auth.ModelOrder[0]
    }
    if a.auth.SetDefault {
        next.DefaultProvider = a.auth.ProviderID
        next.DefaultModel = modelID
    }
    return next, modelID
}
```

### 9.2 /settings 保存（复用）

1. `LoadGlobalSettingsSparse()` → 加载
2. 编辑 → 字段级修改（map 引用直接改）
3. `providerfactory.Create()` → 验证
4. `SaveGlobalSettings()` → 写入
5. 热切换 runtime provider/model

### 9.3 默认透传到 UI 的初始化

```go
func (a *App) initAuthForProvider(providerID string) {
    // 合并三层：默认 → 运行时已有 → 通用保底
    resolved := config.ResolveProviderConfig(providerID, a.settings)
    
    a.auth.ProviderID = providerID
    p := &a.auth.Provider
    p.APIKey = resolved.APIKey
    p.BaseURL = resolved.BaseURL
    p.API = resolved.API
    p.Vendor = resolved.Vendor
    p.HTTPProxy = resolved.HTTPProxy
    p.ForceHTTP11 = resolved.ForceHTTP11
    p.Headers = cloneStringMap(resolved.Headers)
    p.ThinkingFormat = resolved.ThinkingFormat
    p.CacheControl = cloneBoolPtr(resolved.CacheControl)
    p.Responses = cloneResponsesConfig(resolved.Responses)
    
    // 初始化 Models map（含完整参数）
    a.auth.Models = map[string]*modelEditState{}
    a.auth.ModelOrder = nil
    for _, m := range resolved.Models {
        me := modelEditStateFromConfig(&m)
        a.auth.Models[m.ID] = me
        a.auth.ModelOrder = append(a.auth.ModelOrder, m.ID)
    }
}
```

---

## 10. /settings 与 /auth 入口收敛

| 命令 | 入口视图 | 说明 |
|------|----------|------|
| `/auth` | authViewMain | 显示 Existing + Custom 菜单 |
| `/settings` | authViewExistingProvider | 直接进入 Provider 列表 |
| `/auth existing` | authViewExistingProvider | 直接进 Existing 列表 |
| `/auth custom` | authViewCustomID | 直接进 Custom ID 输入 |
| `/settings <providerId>` | authViewSettingsDetail | 直接进入该 Provider 的分层编辑 |
| `/auth <providerId>` | authViewProviderGroupList | 等同 /settings <id> |

---

## 11. 实施计划

### Phase 1：默认透传 + config 包导出

1. 在 `config/settings.go` 导出 `DefaultProviderConfigs/DefaultProviderConfig/DefaultModelConfig/ResolveProviderConfig`
2. 新增 `mergeProviderConfig` 三层合并逻辑
3. `commands.go` 增加 `/settings` case

### Phase 2：结构化编辑状态层

1. 新增 `providerEditState` + `modelEditState` struct（含全字段）
2. `authDialogState` 增加结构化字段替代老的扁平字段（双向兼容）
3. `initAuthForProvider()` — 默认透传初始化
4. `initAuthForCustom()` — 通用模板初始化
5. `initModelFromDefault()` — Add Model 时匹配内置模板

### Phase 3：Provider Group List + 子表单

1. `authViewProviderGroupList` / `authViewProviderFieldEdit` 视图
2. 4 个 Provider Group（Credentials/Protocol/Network/Advanced）
3. Responses 子表单 (`authViewResponsesEdit`)
4. Headers map 编辑器 (`authViewHeadersEdit`)
5. CacheControl 三态切换

### Phase 4：Model Group List + 子表单 + per-model

1. `authViewModelList` 列表渲染
2. `authViewModelGroupList` 5 个 Model Group
3. Basics/Capabilities/Sampling 字段编辑
4. Cost 子表单 (`authViewModelCostEdit`)
5. Compat 子表单 (`authViewModelCompatEdit`, 14 子字段)
6. Add Model 全流程

### Phase 5：Review 全字段 + 保存

1. Review JSON 展示（折叠 cost/compat, ▶ 展开）
2. `buildAuthSettingsFrom` 全字段写入
3. Edit 菜单跳到任意 Provider Group / Model Group
4. Save 路径：sparse → factory 验证 → SaveGlobalSettings → 热切换

### Phase 6：测试与清理

1. 单元测试：默认透传正确性、全字段 round-trip、compat 三态
2. TUI 状态机：Provider Group → Sub-Form → Models → Model Group → Compat → Done
3. 移除老的扁平字段和 authViewAdvanced（或标记 deprecated）
4. 兼容旧 settings.json（无 Name 回退 id、无 Cost 跳过、无 Compat 为 zero struct）

---

## 12. 文件修改计划

| 文件 | 变更 |
|------|------|
| `internal/config/settings.go` | 导出 DefaultProviderConfigs 等 helper |
| `internal/tui/commands.go` | 新增 `/settings` case，收敛入口 |
| `internal/tui/auth_state.go` | 新增文件：providerEditState/modelEditState 定义 + 初始化函数 |
| `internal/tui/auth_provider.go` | 新增文件：Provider Group List + 子表单 + headers/responses 编辑 |
| `internal/tui/auth_model.go` | 新增文件：Model Group List + 子表单 + cost/compat 编辑 |
| `internal/tui/auth_dialog.go` | 瘦身：保留状态定义 + 主菜单 + AddModel + Default + Review + EditMenu |
| `internal/tui/auth_settings.go` | 新增文件：/settings 入口 + Settings Detail 视图 |
| `internal/tui/auth_render.go` | 新增文件：所有 auth 渲染函数从 auth_dialog.go 拆出 |
| `internal/provider/factory` | 无变更 |

---

## 13. 风险与处理

| 风险 | 处理 |
|------|------|
| auth_dialog.go 已 1200+ 行 | 拆 5 个文件，单文件 < 500 行 |
| 内置默认透传导致用户配置"意外覆盖" | 三层合并：已有配置优先级 > 内置默认；仅对**未配置字段**透传 |
| Custom provider 误匹配同名内置默认 | Custom 模式不查 defaultProviderConfigs，一律通用模板 |
| Review JSON 过大（全字段含 cost/compat） | 默认折叠 cost/compat，仅展示字段计数；▶ 按键展开 |
| compat 14 字段认知负担高 | Compatibility 组默认折叠，标注"none active"即正确；高级用户再展开 |
| 三态 bool（nil/true/false）交互不清 | UI 显示 `(auto)` / `enabled` / `disabled`；Space 循环切换 |
| 旧 settings.json 无新字段 | 新字段缺失时回退默认（Name=id, Cost=nil, Compat=zero struct） |
| Add Model 时内置模板匹配多个 provider | 只查当前 providerID 的 models[]，不跨 provider |

---

## 14. 验收标准

- `/settings` 展示所有 provider 及其全字段（含 cost/compat 摘要）
- `/settings <id>` 直接进入该 provider 编辑
- `/auth` → Existing 厂商时，所有字段从 `defaultProviderConfigs` 透传最佳默认值
- `/auth` → Custom 厂商时，使用通用保守默认（openai-chat、空 base URL）
- Add Model 时若内置存在同名模型参数，自动填充（含 cost/compat）
- Provider 层 11 字段全覆盖（apiKey/baseUrl/api/vendor/httpProxy/forceHTTP11/headers/thinkingFormat/cacheControl/responses/models）
- Model 层 9 顶层字段全覆盖（id/name/contextWindow/maxTokens/reasoning/input/temperature/topP/cost/compat）
- Compat 子表单覆盖全部 14 子字段
- Responses 子表单覆盖全部 4 字段
- Cost 子表单覆盖全部 4 字段
- Display Name 可自定义，默认 = model ID
- Headers 支持增删改（key-value 对）
- CacheControl 三态切换（auto/on/off）
- Save 使用 sparse global settings，不覆盖项目级和环境变量
- token 在 Review 中脱敏展示
- 热切换成功后下一条消息使用新 provider/model
- 旧 settings.json 完全兼容（新字段缺失不崩溃）

---

## 15. 实施进度追踪

> 最后核对: 2026-07-01

### Phase 1：默认透传 + config 包导出 ✅ 已完成

| 项目 | 状态 | 说明 |
|------|------|------|
| `DefaultProviderConfigs()` | ✅ | `config/settings.go:1110` |
| `DefaultProviderConfig(id)` | ✅ | `config/settings.go:1116` |
| `DefaultModelConfig(providerID, modelID)` | ✅ | `config/settings.go:1126` |
| `ResolveProviderConfig(id, runtime)` | ✅ | `config/settings.go:1142` |
| `mergeProviderConfig` 三层合并 | ✅ | 内嵌于 `ResolveProviderConfig` |
| `/settings` 命令入口 | ✅ | `commands.go` + `command_suggest.go` |

### Phase 2：结构化编辑状态层 ✅ 已完成

| 项目 | 状态 | 说明 |
|------|------|------|
| `providerEditState` struct | ✅ | `auth_state.go:13` — 含全字段 |
| `modelEditState` struct | ✅ | `auth_state.go:58` — 含全字段 |
| `compatEditState` struct | ✅ | `auth_state.go:36` — 含 14 子字段 |
| `responsesEditState` struct | ✅ | `auth_state.go:28` — 含 4 子字段 |
| `authDialogState` 结构化字段 | ✅ | `auth_dialog.go:53` — Provider/Models/ModelOrder |
| `initAuthForProvider()` | ✅ | `auth_settings.go:32` — 三层合并透传 |
| `initAuthForCustom()` | ✅ | `auth_settings.go:52` — 通用模板 |
| `initModelFromDefault()` | ✅ | `auth_settings.go:63` — 内置匹配+runtime+通用三层回退 |
| `providerEditStateFrom()` | ✅ | `auth_state.go:82` |
| `modelEditStateFromMC()` | ✅ | `auth_state.go:142` |
| `toConfig()` write-back | ✅ | `auth_state.go:177/202/226` — provider/responses/compat/model |
| Clone helpers | ✅ | `auth_state.go:256-290` |

### Phase 3：Provider Group List + 子表单 ✅ 已完成

| 项目 | 状态 | 说明 |
|------|------|------|
| `authViewProviderGroupList` | ✅ | `auth_provider.go` |
| Group A: Credentials (APIKey, Vendor) | ✅ | `authViewProviderCredentials` |
| Group B: Protocol (BaseURL, API, Responses) | ✅ | `authViewProviderProtocol` |
| Group C: Network (HTTPProxy, ForceHTTP11) | ✅ | `authViewProviderNetwork` |
| Group D: Advanced (Headers, ThinkingFormat, CacheControl) | ✅ | `authViewProviderAdvanced` |
| Responses 子表单 | ✅ | `authViewResponsesEdit` — 4 字段全覆盖 |
| Headers map 编辑器 | ✅ | `authViewHeadersEdit` — 增删改 |
| CacheControl 三态切换 | ✅ | `cycleTriState()` |
| API Type 枚举选择 | ✅ | `authViewAPIChoice` |

### Phase 4：Model Group List + 子表单 + per-model ✅ 已完成

| 项目 | 状态 | 说明 |
|------|------|------|
| `authViewModelList` | ✅ | `authModelListOptions()` |
| `authViewModelGroupList` (5 组) | ✅ | `authModelGroupOptions()` |
| Group A: Basics | ✅ | `authViewBasics` — Name/ContextWindow/MaxTokens |
| Group B: Capabilities | ✅ | `authViewCapabilities` — Reasoning/Input |
| Group C: Sampling | ✅ | `authViewSampling` — Temperature/TopP |
| Group D: Cost | ✅ | `authViewCost` — 4 字段 + CostEnabled 开关 |
| Group E: Compatibility | ✅ | `authViewCompat` — 14 子字段全覆盖 |
| Add Model 流程 | ✅ | `authViewAddModelID` → `authViewAddModelName` |
| Per-model 独立编辑 | ✅ | `map[string]*modelEditState` |

### Phase 5：Review 全字段 + 保存 ✅ 已完成

| 项目 | 状态 | 说明 |
|------|------|------|
| `authViewReview` | ✅ | `auth_dialog.go` |
| Review JSON 展示 | ✅ | `previewBuildFoldedJSON()` |
| cost/compat 折叠/展开 | ✅ | `previewExpansion` + `togglePreviewFold()` |
| `buildAuthSettingsFrom` 全字段写入 | ✅ | `auth_dialog.go:633` — Provider 11 字段 + Model 全字段 |
| Edit 菜单跳转 | ✅ | `authViewEditMenu` |
| Save: sparse → factory → SaveGlobal → 热切换 | ✅ | `saveAuthProvider()` |
| Settings Detail 视图 | ✅ | `authViewSettingsDetail` + `authSettingsDetailOptions()` |

### Phase 6：测试与清理 ✅ 已完成

| 项目 | 状态 | 说明 |
|------|------|------|
| 单元测试（默认透传） | ✅ | `TestResolveProviderConfigMergesDefaults`, `TestDefaultModelConfigLookup` |
| 单元测试（全字段 round-trip） | ✅ | `TestProviderEditStateRoundTrip`, `TestModelEditStateRoundTrip` |
| 单元测试（compat 三态） | ✅ | `TestCycleTriState`, `TestToggleModelTriState`, `TestCompatEditStateActiveCount`, `TestCompatResetToAuto` |
| 单元测试（headers/cost/fold） | ✅ | `TestHeadersEditFlow`, `TestCostEnabledToggle`, `TestPreviewBuildFoldedJSONMultipleModels`, `TestPreviewFoldMaskedKey` |
| 移除老扁平字段 | ✅ | `authDialogState` 已不再有老扁平字段 |
| 移除死代码 | ✅ | 6 个函数已删除 |
| 统一 clone helpers | ✅ | config 导出，tui 重复已删除 |
| 修复 fold markers | ✅ | ReplaceAll + 名称解析修复 |
| 旧 settings.json 兼容 | ✅ | Name 回退 id、Cost=nil 跳过、Compat=nil 回退 zero |

---

## 16. 新增内容（方案之外的实现）

以下内容在实施过程中新增，原方案未涉及：

| 新增项 | 文件 | 说明 |
|--------|------|------|
| `authViewAddModelName` 步骤 | `auth_dialog.go` | 方案仅设计 `authViewAddModel` 一个输入，实现拆分为 ID → Name 两步 |
| `← Back` 返回按钮 | `auth_dialog.go` | Existing Providers 列表末尾新增返回主菜单选项 |
| `✓ Done` 保存按钮 | `auth_provider.go` | Provider Group List 末尾新增保存退出选项 |
| `previewExpansion` 折叠机制 | `auth_render.go` | Review JSON 的 cost/compat 折叠/展开状态追踪 |
| `authViewAPIChoice` 视图 | `auth_provider.go` | API Type 独立枚举选择器（方案中放在 Protocol 组内） |
| `previewBuildFoldedJSON()` | `auth_render.go` | 带折叠的 JSON 预览渲染器 |
| `compatEditState.Active` 标记 | `auth_state.go` | 追踪 compat 是否有显式编辑，避免无改动时写入空 compat |
| Provider Group 的 Done 直接保存 | `auth_dialog.go` | Provider Group List 的 Done 调用 `saveAuthProvider()` 保存并退出 |
| `filterAuthProviderIDs` 搜索过滤 | `auth_render.go` | Existing Providers 支持类型搜索过滤 |
| `sortedAuthProviderIDs` 排序 | `auth_render.go` | Provider 列表按字母排序 |

---

## 17. 与方案的偏差

| 偏差点 | 方案设计 | 实际实现 | 影响 |
|--------|----------|----------|------|
| `/settings` 命令 | `commands.go` 增加 `/settings` case | 已注册 ✅ |
| Add Model 拆为两步 | 方案 §2.7 设计为一个输入 → Group List | 实现为 ID 输入 → Name 输入 → Group List | **轻微偏差** — 体验更清晰，但多一步 |
| modelEditState.Cost 平铺 | 方案 §6.3 设计为嵌套 `Cost struct` | 实现为 `CostInput/CostOutput/CacheRead/CacheWrite` 平铺字段 | **无功能影响** — 字段覆盖完整 |
| authView 枚举顺序 | 方案 §2.4.3 列出预期顺序 | 实际顺序不同（按文件分组） | **无影响** — iota 值仅内部使用 |
| Provider Group List 的 Done 行为 | 方案未明确 Done 退出到哪里 | 实现为直接 `saveAuthProvider()` 保存并退出 /auth | **符合预期** — 用户要求的快速退出 |
| Provider 子表单的 Done 行为 | 方案设计返回 Group List | 实现为 `popAuthView()` 返回上一级 | **符合预期** |
| Model Group List 的 Done 行为 | 方案设计 Done → Default → Review | 实现为 `popAuthView()` 返回 Model List | **轻微偏差** — 不强制走 Default/Review 流程 |

