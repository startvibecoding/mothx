# /auth Provider 登录配置方案设计

> 状态: 已实现 MVP，持续增强中
> 日期: 2026-06-16
> 目标版本: v0.1.43
> 最近更新: 2026-06-16 — 增加 sparse 保存、滚动列表、Review Edit、小参数调整、多 base_url 与 provider 排序

## 1. 概述

新增 TUI 内 `/auth` 指令，用于在交互界面中配置 Provider、Token、Base URL 和模型列表，降低首次使用和切换厂商的配置门槛。

核心体验参考 `/home/free/src/qwen-code-by-moark` 中 `/auth` 弹框：以 bordered dialog 展示多层菜单，支持键盘选择、逐步输入、回退、预览并保存。

同时优化首次安装启动体验：当全局 `settings.json` 不存在时自动创建默认配置文件，并在首次 TUI 启动时提示用户使用 `/auth` 添加登录厂商、token 和模型。

## 2. 设计目标

| 目标 | 说明 |
|------|------|
| TUI 内配置 | 用户无需手写 `settings.json`，可通过 `/auth` 完成常见配置 |
| 复用现有 schema | 不改变 `settings.json` schema，不引入新的凭证文件格式 |
| 支持已有厂商 | 可选择已有 provider，为其填写/更新 token、baseUrl、模型，并可保留已有模型参数 |
| 支持新增厂商 | 可按本项目 API 类型新增自定义 provider：`openai-chat` / `openai-responses` / `anthropic-messages` / `google-gemini` / `google-vertex` |
| 分步表单 | 使用弹框式 wizard：选择 → 输入 token → 选择/输入 base_url → 输入模型 → 小参数调整 → 预览保存 |
| 首次启动提示 | 首次自动创建配置文件后，在 TUI 显示 `/auth` 引导 |
| 安全保存 | `settings.json` 使用 `0600` 权限，预览中 token 默认脱敏 |

## 3. 非目标

- 第一阶段不实现 OAuth / 浏览器登录，只支持 API token 输入。
- 不改变 provider factory 的字段含义和现有 provider 创建逻辑。
- 不实现跨平台系统 keychain 存储；token 仍写入 `settings.json` 或以 `${ENV}` 引用形式保存。
- 不强制 Gateway / Hermes / ACP 暴露 `/auth`。第一阶段仅 TUI。
- 不自动联网拉取模型列表。推荐模型可以来自内置配置，用户也可手动输入。

## 4. 用户体验

### 4.1 首次启动

当 `~/.vibecoding/settings.json` 不存在时：

1. `config.LoadSettings()` 继续自动创建默认文件。
2. 记录“本次启动创建了配置文件”的信号。
3. TUI header / initial message 中追加提示：

```text
Created default config: ~/.vibecoding/settings.json
No provider token configured yet. Run /auth to add a provider token and model.
```

如果默认 provider 的 token 环境变量已存在（例如 `DEEPSEEK_API_KEY`），可以只提示配置文件已创建，不强制进入 `/auth`。

### 4.2 `/auth` 主菜单

输入 `/auth` 后打开弹框：

```text
┌─ Connect a Provider ─────────────────────────────────────────────┐
│ › Existing Providers                                             │
│   Add or update token/model under an existing provider            │
│                                                                  │
│   Custom Provider                                                │
│   Add provider by compatibility type, base URL, token and models  │
│                                                                  │
│   Env Token Helper                                               │
│   Save token as ${ENV_VAR} reference instead of plain text        │
│                                                                  │
│ Enter to select, ↑↓ to navigate, Esc to close                     │
└──────────────────────────────────────────────────────────────────┘
```

说明：样式参考 qwen-code 的 `AuthDialog`：单线/圆角边框、标题、单选列表、描述文本、底部快捷键提示。

### 4.3 在已有厂商下添加 token

流程：

```text
/auth
  → Existing Providers
  → 选择 provider，例如 deepseek-openai / openai / anthropic
  → API Key 输入
  → Base URL 选择或输入/确认
  → 选择或输入模型
  → Advanced Parameters（可选）
  → 是否设为默认 provider/model
  → Review JSON
  → Save 或 Edit
```

保存行为：

- 更新全局 `settings.json` 中对应 `providers.<name>.apiKey`。
- 如用户修改 base URL，则更新 `providers.<name>.baseUrl`。
- 如用户输入模型，则更新 `providers.<name>.models`；已有模型 ID 会保留原参数，再应用用户设置的小参数覆盖。
- 可选择设为默认 provider/model，更新 `defaultProvider` / `defaultModel`。
- 保存只 patch 全局 `settings.json` 中真实存在的配置，不把默认 providers、项目级配置或环境变量覆盖写回全局文件。

### 4.4 新增自定义厂商

流程：

```text
/auth
  → Custom Provider
  → Provider ID 输入，例如 openrouter / local-llm
  → Compatibility Type:
            openai-chat
            openai-responses
            anthropic-messages
            google-gemini
            google-vertex
  → Base URL 输入
  → API Key 输入
  → Model IDs 输入或选择推荐项
  → Advanced Parameters（可选）
  → 是否设为默认 provider/model
  → Review JSON
  → Save 或 Edit
```

兼容类型映射（以本项目 `internal/config.ProviderConfig.API` 命名为准）：

| `/auth` 展示选项 | 本项目写入 `ProviderConfig.API` | qwen-code 参考类型 | 默认 Base URL 示例 |
|------|---------------------------|--------------------|--------------------|
| OpenAI-compatible | `openai-chat` | `AuthType.USE_OPENAI` | `https://api.openai.com/v1` |
| OpenAI Responses | `openai-responses` | 无直接等价，按本项目能力 | `https://api.openai.com/v1` |
| Anthropic-compatible | `anthropic-messages` | `AuthType.USE_ANTHROPIC` | `https://api.anthropic.com` |
| Gemini-compatible | `google-gemini` | `AuthType.USE_GEMINI` | `https://generativelanguage.googleapis.com/v1beta/models` |
| Google Vertex | `google-vertex` | 无直接等价，按本项目能力 | `https://aiplatform.googleapis.com/v1/publishers/google/models` |

第一阶段建议 OpenAI-compatible 默认写入 `openai-chat`，避免 Responses API 对兼容服务支持不一致。后续可在高级选项中加入 `openai-responses`。

### 4.5 参考 qwen-code 扩展更多内置厂商

除自定义厂商外，`/auth` 的 provider 选择列表应参考 qwen-code 的 provider registry，补齐当前项目中缺失或模型信息较旧的常用厂商。

qwen-code 参考位置：

- `/home/free/src/qwen-code-by-moark/packages/core/src/providers/all-providers.ts`
- `/home/free/src/qwen-code-by-moark/packages/core/src/providers/presets/*.ts`
- `/home/free/src/qwen-code-by-moark/packages/core/src/providers/provider-config.ts`
- `/home/free/src/qwen-code-by-moark/packages/core/src/providers/types.ts`

注意：qwen-code 仅作为厂商、模型、base URL 与交互样式参考；开发时所有配置字段、API 类型、vendor 名称、compat flag 名称都必须以本项目 Go 类型为准。如果 qwen-code 的类型名/枚举名与本项目不一致，不做沿用或翻译式引入，统一映射到本项目已有命名。

可参考引入或核对的厂商：

| qwen-code provider | 建议 vibecoding provider id | 本项目 API 类型 | 说明 |
|--------------------|-----------------------------|----------------|------|
| `deepseek` | `deepseek-openai` / `deepseek` | `openai-chat` 或 `anthropic-messages` | 当前已有 `deepseek-openai` 与 `deepseek-anthropic`，需核对模型和参数 |
| `openrouter` | `openrouter` | `openai-chat` | 已新增，常用 OpenRouter 配置 |
| `minimax` | `minimax` | `openai-chat` | 已新增，支持国际/中国 base URL 可选 |
| `zai` | `zai` | `openai-chat` | 已新增，GLM / Z.AI 系列，支持 Standard/Coding Plan endpoint |
| `modelscope` | `modelscope` | `openai-chat` | 已新增，魔搭推理 API |
| `alibabaStandard` | `alibaba-standard` | `openai-chat` | 已新增，百炼标准 API Key，多区域 endpoint |
| `coding-plan` | `alibaba-coding-plan` | `openai-chat` | 已新增，支持中国/国际 endpoint |
| `token-plan` | `alibaba-token-plan` | `openai-chat` | 已新增，专属 token-plan endpoint |
| `moark` | `moark` | `openai-chat` | 已新增，Moark Token Factory 入口 |
| `idealab` | `idealab` | `openai-chat` | 内部/实验性质，默认不建议面向普通用户展示 |

落地原则：

- 新增 vendor 默认配置仍放在 `internal/config/settings.go` 的 `DefaultSettings().Providers`，或后续拆成 provider preset registry，但不改变 `settings.json` schema。
- 所有实现必须使用本项目已有类型命名：`config.ProviderConfig`、`config.ModelConfig`、`config.ModelCompat`、`ProviderConfig.API`、`ProviderConfig.Vendor`、`ProviderConfig.ThinkingFormat` 等。
- qwen-code 的 `AuthType.USE_OPENAI`、`contextWindowSize`、`enableThinking`、`modalities` 等仅作为输入资料，写入 Go 配置时分别映射为本项目的 `api`、`contextWindow`、`reasoning`、`input` 等字段。
- 需要特殊行为的厂商放入 `internal/provider/vendor_<name>.go`；仅 OpenAI 兼容且无特殊行为的厂商复用通用 provider。
- `/auth` 列表展示内置厂商时，应读取当前 `settings.Providers` 与新增 preset metadata，避免 UI hardcode 一份和配置不一致的列表。
- Existing Providers 列表最多展示 5 个项目，支持上下滚动和虚拟滚动提示，避免 provider 较多时撑高弹框。
- Existing Providers 排序优先级：`moark` → `deepseek` → `xiaomi` → 火山引擎（`doubao` / `volc` / `ark` / `seed`）→ `openai` → `anthropic` → `google` → 其它字母序。
- 对于 qwen-code 中存在多个 base URL 的 provider，`/auth` 应提供 base URL 选择步骤，而不是只写死一个 endpoint。当前已整理：
  - `minimax`: International / China
  - `zai`: Standard API Key / Coding Plan
  - `alibaba-standard`: China Beijing / Singapore / US Virginia / China Hong Kong
  - `alibaba-coding-plan`: China Beijing / Singapore International

### 4.6 模型输入

模型步骤参考 qwen-code 的 `ProviderSetupSteps`：

- 支持手动输入多个模型 ID，用逗号分隔。
- 如果已有 provider 配置了 models，则显示推荐模型列表，可用 Space 勾选。
- 自定义 provider 没有推荐模型时，仅展示输入框。
- 每个模型默认生成：

```jsonc
{
  "id": "model-id",
  "name": "model-id",
  "contextWindow": 128000,
  "maxTokens": 8192,
  "input": ["text"]
}
```

### 4.7 Advanced Parameters 小参数调整

模型输入后进入 `Advanced Parameters`，用于对本次配置中的模型应用小参数覆盖。空值表示“不覆盖/保留已有值”。

支持字段：

| UI 项 | 写入字段 | 校验/语义 |
|------|----------|----------|
| Context window | `models[].contextWindow` | 正整数；空值表示保留已有或使用默认 |
| Max output tokens | `models[].maxTokens` | 正整数；空值表示保留已有或使用默认 |
| Reasoning | `models[].reasoning` | Enter 切换 true/false |
| Input modalities | `models[].input` | 逗号分隔，如 `text,image,audio,video,pdf` |
| Temperature | `models[].temperature` | `0..2`；使用指针字段，支持显式 `0` |
| Top P | `models[].top_p` | `0..1`；使用指针字段，支持显式 `0` |

保存模型时：

- 如果模型 ID 已存在，先保留已有完整模型参数，再应用本页设置的覆盖。
- 如果模型 ID 是新增的，先使用保守默认值：`contextWindow=128000`、`maxTokens=8192`、`input=["text"]`，再应用本页设置的覆盖。
- `Skip/Clear` 表示清空本次高级参数，使用已有模型参数或默认值。

### 4.8 Review 与保存

保存前展示脱敏预览：

```text
The following JSON will be saved to settings.json:
{
  "defaultProvider": "openrouter",
  "defaultModel": "anthropic/claude-sonnet-4",
  "providers": {
    "openrouter": {
      "apiKey": "sk-****abcd",
      "baseUrl": "https://openrouter.ai/api/v1",
      "api": "openai-chat",
      "models": [...]
    }
  }
}

Enter to save, Esc to go back
```

Review 页面包含两个操作：

- `Save`：写入全局 `settings.json` 并热切换当前 TUI provider/model。
- `Edit`：进入编辑菜单，可返回修改 API Key、Base URL、Model IDs、Advanced Parameters、Default setting；自定义 provider 额外可修改 Provider ID 和 API Type。

Review JSON 预览最多展示固定行数，超出时显示 `… N more lines hidden`，避免大 provider 配置撑高弹框。

保存成功后关闭弹框并显示：

```text
✅ Provider saved: openrouter / anthropic/claude-sonnet-4
Next message will use the new provider/model.
```

若当前 agent 已存在，需要重建 provider/model 并重置 agent history binding，避免下一轮继续使用旧 provider。

## 5. TUI 状态与交互设计

### 5.1 App 状态

在 `internal/tui.App` 增加 auth dialog 状态：

```go
type authView string

type authDialogState struct {
    Open bool
    View authView
    Step string
    ProviderID string
    API string
    BaseURL string
    APIKey string
    ModelIDs string
    ContextWindow string
    MaxTokens string
    Reasoning bool
    InputTypes string
    Temperature string
    TopP string
    ParamField string
    SetDefault bool
    Error string
}
```

可按实际实现拆分到 `internal/tui/auth_*` 文件，避免 `app.go` 膨胀。

### 5.2 键盘行为

| 按键 | 行为 |
|------|------|
| `/auth` | 打开弹框 |
| ↑/↓ | 菜单/推荐项移动 |
| Enter | 选择或提交当前输入 |
| Space | 勾选推荐模型/开关项；部分菜单中等同 Enter |
| Esc | 当前步骤回退；主菜单关闭 |
| Ctrl+C | 关闭弹框或退出程序，遵循现有 TUI 行为 |

### 5.3 渲染方式

优先复用现有 Bubble Tea + Lipgloss：

- 使用 `lipgloss.NewStyle().Border(...).Padding(1)` 渲染 dialog。
- 使用现有 `renderutil` 或新增小型 list/input 组件。
- 如果项目已有 editor input 组件可复用，则复用 `internal/tui/components/editor` 的输入能力。

## 6. 配置写入设计

### 6.1 新增 config 写入 API

在 `internal/config` 增加小范围写入函数：

```go
func SaveGlobalSettings(s *Settings) error
func LoadGlobalSettingsOrDefault() (*Settings, error) // defaults + global，用于运行时完整配置
func LoadGlobalSettingsSparse() (*Settings, error)    // 仅全局文件显式字段，用于 patch 写回
```

原则：

- 写入全局 `settings.json`，不自动改项目 `.vibe/settings.json`。
- `/auth` 保存必须基于 `LoadGlobalSettingsSparse()` patch 原始全局配置，不能把默认 providers、项目级 settings 或环境变量覆盖写回全局文件。
- 运行时热切换可以基于当前 runtime settings patch 后调用 provider factory，但持久化写入必须使用 sparse 全局配置。
- 配置读取合并要尊重显式零值/false。项目级 settings 通过直接 unmarshal 到已加载配置上实现 presence-aware 覆盖，避免 `!= 0` / `!= ""` 造成显式 `0`、`false` 失效。
- 写入前创建目录，权限 `0700`；文件权限 `0600`。
- 使用临时文件 + rename，降低写坏配置风险。

### 6.2 token 保存策略

提供两种模式：

1. Plain token：直接写入 `apiKey: "sk-..."`。
2. Env reference：写入 `apiKey: "${OPENROUTER_API_KEY}"`，并提示用户自行设置环境变量。

### 6.3 厂商模型与参数核对

在实现 `/auth` 前应先做一次 provider inventory，对当前项目默认厂商与 qwen-code preset 做对齐，避免 UI 推荐用户选择过期模型或错误参数。

核对范围：

| 字段 | vibecoding 字段 | qwen-code 参考字段 | 核对要求 |
|------|-----------------|--------------------|----------|
| Provider ID | `providers.<id>` | `ProviderConfig.id` | id 命名稳定，避免破坏用户现有配置；必要时新增 alias 而不是重命名 |
| Base URL | `baseUrl` | `baseUrl` / `BaseUrlOption[]` | 核对官方 endpoint，多个区域 endpoint 在 `/auth` 中可选 |
| API 类型 | `api` | `protocol` / auth type | 映射到本项目已有字符串：`openai-chat`、`openai-responses`、`anthropic-messages`、`google-gemini`、`google-vertex` 等；不直接使用 qwen-code `AuthType.*` 名称 |
| API Key | `apiKey` | env key / credential metadata | 建议 env var 命名清晰，如 `${OPENROUTER_API_KEY}` |
| 模型 ID | `models[].id` | `models[].id` | 移除明显下线模型，补充最新推荐模型 |
| 展示名称 | `models[].name` | `models[].id` / description | 默认可等于 id，重要模型可设置友好名称 |
| Reasoning | `models[].reasoning` | `enableThinking` | qwen-code `enableThinking=true` 映射为 `reasoning=true` |
| Context Window | `models[].contextWindow` | `contextWindowSize` | 按官方/参考值核对，未知时保守默认 |
| Max Tokens | `models[].maxTokens` | 无直接等价或 provider 默认 | 按官方输出上限核对，未知时保守默认 8192/16384 |
| Input Modality | `models[].input` | `modalities` | text/image/audio/video/pdf 能力尽量准确 |
| Compat | `models[].compat` / `vendor` | qwen-code provider adapter/test | 对有特殊协议行为的模型补充本项目 `ModelCompat` flags 或 `ProviderConfig.Vendor`；不引入 qwen-code adapter 类型名 |
| Thinking Format | `thinkingFormat` / `compat.thinkingFormat` | provider adapter | DeepSeek/Xiaomi 等特殊 reasoning 字段需核对 |

优先核对当前已有厂商：

- `deepseek-openai`：核对 `deepseek-v4-flash`、`deepseek-v4-pro` 的 context window、maxTokens、reasoning、thinking format、价格信息。
- `deepseek-anthropic`：确认 DeepSeek Anthropic endpoint 是否仍推荐，模型参数是否与 OpenAI endpoint 一致。
- `openai`：核对是否应补充 GPT-5 / o-series 最新模型，以及 `openai-responses` 下各模型的 compat flags。
- `anthropic`：核对 Claude 4/4.5/Haiku/Opus 模型名、上下文、maxTokens、cache 价格与 `cacheControl`。
- `google-gemini` / `google-vertex`：核对 Gemini 2.5/3 系列模型、base URL、max output token、multimodal 能力。
- `xiaomi`：核对 MiMo 模型列表、`thinkingFormat: xiaomi`、多模态能力。

建议新增厂商的首批核对顺序（当前实现状态）：

1. `openrouter`：已新增，常用且用户配置频率高，OpenAI-compatible，base URL 稳定。
2. `minimax`：已新增，国际/中国 endpoint 与 M 系列模型参数来自 qwen-code preset。
3. `zai`：已新增，GLM/Z.AI 系列，支持 Standard API Key / Coding Plan endpoint。
4. `modelscope`：已新增，魔搭 API，模型 ID 带 namespace，输入和显示应保留 `/`。
5. `alibaba-standard`：已新增，百炼 DashScope/OpenAI-compatible endpoint，支持四个区域 endpoint。
6. `alibaba-coding-plan`：已新增，支持中国/国际 endpoint。
7. `alibaba-token-plan`：已新增，专属 token-plan endpoint。
8. `moark`：已新增，Moark API Key provider。

产出物建议：

- 新增 `internal/provider/presets` 或 `internal/config/provider_presets.go` 记录 provider metadata（若不想扩大范围，可先直接补充 `DefaultSettings`）。
- 新增/更新 `internal/provider/vendor_<name>.go`，只处理确有特殊兼容逻辑的厂商。
- 增加 `docs/proposal` 或代码注释中的“模型参数来源”记录：官方文档链接、qwen-code preset、最后核对日期。
- 增加测试确保 `/auth` 推荐模型与 `DefaultSettings().Providers` 中模型一致。



保存后 TUI 需要立即生效：

1. 调用 `providerfactory.Create(settings, providerID, modelID)` 重建 provider/model。
2. 更新 `a.provider`、`a.model`、`a.settings`。
3. 将 `a.agent = nil`、`a.agentHistoryLoaded = false`，下一次消息重新创建 agent。
4. 更新 header 或追加状态消息，展示当前 provider/model。

若保存失败或 provider 创建失败，不覆盖当前运行中的 provider/model，只显示错误。

## 8. 首次启动提示设计

当前 `ensureConfigExists()` 已会创建默认 `settings.json` 并打印 `Created default config`。建议改为返回创建状态：

```go
type LoadResult struct {
    Settings *Settings
    CreatedGlobalConfig bool
}
```

为了减少破坏面，可先新增辅助函数而不改现有 `LoadSettings()` 签名：

```go
func LoadSettingsWithMeta() (*Settings, LoadMeta, error)

type LoadMeta struct {
    CreatedGlobalConfig bool
    GlobalSettingsPath string
}
```

CLI TUI 路径使用 `LoadSettingsWithMeta()`；其他调用继续用 `LoadSettings()`。

提示条件：

- `CreatedGlobalConfig == true`
- 且默认 provider token 未解析成功：`settings.ResolveKey(settings.DefaultProvider)` 为空或仍为 `${...}` 引用
- 且当前不是 `--print` 非交互模式

## 9. 文件与模块计划

建议新增/修改：

| 文件 | 变更 |
|------|------|
| `internal/tui/commands.go` | 增加 `/auth` 分支 |
| `internal/tui/auth_dialog.go` | Auth dialog 状态机、渲染入口 |
| `internal/tui/auth_flow.go` | Provider setup flow：步骤、校验、preview、save |
| `internal/tui/auth_render.go` | 菜单、输入、review 等渲染函数 |
| `internal/config/settings.go` | 增加 `LoadSettingsWithMeta` / 保存全局 settings API |
| `cmd/vibecoding/main.go` | 首次创建配置后向 TUI initial message 注入 `/auth` 提示 |
| `internal/provider/factory` | 不改或仅复用 `Create` 进行保存后验证 |
| `internal/config/provider_presets.go`（可选） | 从 qwen-code preset 梳理内置厂商元数据、base URL 选项、推荐模型 |
| `internal/provider/vendor_<name>.go`（按需） | 为 OpenRouter/MiniMax/Z.AI/ModelScope/Alibaba 等补充特殊兼容逻辑 |
| `internal/tui/command_render.go` | `/help` 增加 `/auth` 说明 |

## 10. 实施步骤

### Phase 0：参考 qwen-code 核对厂商与模型

- 对照 qwen-code `packages/core/src/providers/presets/*.ts`，整理当前项目已有/缺失厂商清单。
- 建立 qwen-code → 本项目配置字段映射表，开发和写入时一律使用本项目 Go 类型/字段/字符串命名。
- 核对已有厂商模型 ID、contextWindow、maxTokens、reasoning、input modality、thinkingFormat、compat flags。
- 确定首批新增 provider：建议 OpenRouter、MiniMax、Z.AI、ModelScope、Alibaba Standard。
- 为多 endpoint 厂商整理 base URL 选项，供 `/auth` 选择。
- 明确每个新增厂商是否需要 `vendor_<name>.go` 特殊适配，或可复用通用 OpenAI-compatible provider。

### Phase 1：配置元信息与首次提示

- 增加 `LoadSettingsWithMeta()`，保留 `LoadSettings()` 兼容。
- `ensureConfigExists` 返回是否创建。
- TUI 首次启动时提示 `/auth`。
- 添加配置加载/创建相关单元测试。

### Phase 2：基础 `/auth` 弹框

- 新增 `/auth` 命令。
- 实现主菜单、已有 provider 列表、自定义 provider 兼容类型选择。
- 实现 Esc/Enter/↑↓ 基础交互。
- `/help` 展示 `/auth`。

### Phase 3：Provider setup flow

- 实现 baseUrl、apiKey、modelIds 输入步骤。
- Existing Providers 列表最多展示 5 个，支持虚拟滚动提示；排序按 moark/deepseek/xiaomi/火山/openai/anthropic/google/其它。
- 实现多 base URL provider 的 endpoint 选择：MiniMax、Z.AI、Alibaba Standard、Alibaba Coding Plan。
- 实现 Advanced Parameters：contextWindow、maxTokens、reasoning、input modalities、temperature、top_p。
- 实现 Review JSON 脱敏预览、固定高度截断和 Edit 菜单。
- 实现校验：provider ID 非空且合法、base URL URL 格式、API key 非空、至少一个模型、小参数范围合法。

### Phase 4：保存与热切换

- 保存到全局 `settings.json`，但只 patch sparse global settings，不展开默认 providers，不写入项目级/环境变量覆盖。
- 保存后调用 provider factory 验证并热切换当前 provider/model。
- 失败时回滚 UI 状态，不破坏当前 provider。
- 配置读取路径尊重显式 `0` / `false`，避免按 Go 零值错误跳过用户意图。

### Phase 5：测试与文档

- 单元测试：settings 保存、token 脱敏、provider update merge。
- TUI 状态机测试：主菜单导航、步骤回退、保存错误。
- 手动测试：无 settings 首启、已有 provider 更新、自定义 OpenAI-compatible provider。
- 如用户可见变化较大，更新 README/README_zh 的配置章节。

## 11. 风险与处理

| 风险 | 处理 |
|------|------|
| 明文 token 写入文件 | 文件权限 `0600`，Review 脱敏，并提供 `${ENV}` 保存方式 |
| 覆盖用户手写配置格式 | 使用最小范围 merge；后续可考虑保留 JSON raw 字段/格式 |
| 保存后 provider 创建失败 | 先验证新配置，成功后再切换运行态；失败只展示错误 |
| 默认模型参数不准确 | 提供保守默认值，并允许 Advanced Config 修改 |
| TUI 状态复杂 | 独立 `auth_*` 文件和小型状态机，避免污染主 App |

## 12. 验收标准

- 输入 `/auth` 能打开弹框，并可 Esc 关闭。
- 可选择已有 provider，填写 token 和模型后保存到 `settings.json`。
- 可新增自定义 OpenAI-compatible provider，填写 base URL/token/model 后保存并设为默认。
- 保存后下一条消息使用新 provider/model。
- 首次无 `settings.json` 启动会自动创建文件，并在 TUI 提示使用 `/auth`。
- `/help` 中包含 `/auth` 说明。
- token 在预览和错误信息中不会明文泄露。
