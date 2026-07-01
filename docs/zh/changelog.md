# 更新日志


## v1.1.57

### 🔧 改进

- **统计面板：协议与厂商分离**
  - 统计面板中的「Provider」列已按语义拆分为**厂商**（公司名称）和**协议**（API 协议类型，如 `openai-chat`、`anthropic-messages`、`google-gemini`）。
  - 新增 `Provider.API()` 接口方法，在 `request_stats` 中同时记录协议类型与厂商名称。
  - 新增厂商与协议筛选下拉框；饼图与表格现在同时展示两个维度。
  - 数据库迁移 006 为 `request_stats` 表添加 `protocol` 列（已有数据回填为空字符串）。

## v1.1.56

### ✨ 新功能

- **LongCat 厂商支持**
  - 新增 `longcat` 厂商适配器，支持 OpenAI 兼容协议（`https://api.longcat.chat/openai`）与 Anthropic 兼容协议（`https://api.longcat.chat/anthropic`）两种接入方式。
  - 默认设置中注册了两个内置 provider：`longcat`（OpenAI 协议，`LONGCAT_API_KEY`）与 `longcat-anthropic`（Anthropic 协议，`LONGCAT_ANTHROPIC_API_KEY`）。
  - 默认模型 `LongCat-2.0`：上下文长度 1M，最大输出长度 128K Tokens。
  - TUI 授权对话框中，在 `longcat` 厂商下提供 OpenAI / Anthropic 两种 BaseURL 的选择。


## v1.1.55

### ✨ 新功能

- **TUI 会话选择器与延迟创建会话**
  - `/sessions` 现在会打开交互式选择框，支持方向键上下选择、回车切换、`n` 新建会话、`d` 删除会话。原有 `/sessions ls`、`/sessions set <id>`、`/sessions clear`、`/sessions del <id>` 命令仍然保留。
  - TUI 启动时不再立即创建空会话；只有在用户第一次发送消息时才初始化新会话。`--continue`、`--resume`、`--session` 和 `/sessions set` 仍会绑定已有会话。
  - 在 TUI 中继续或切换会话时，现在会把加载到的会话历史打印到正常终端 scrollback 中，与启动时展示历史的行为保持一致。

- **OpenAI 兼容模型的内联 `<think>` 推理**
  - 为 OpenAI 兼容供应商新增 `parseReasoningInContent` 模型兼容标志。启用后，正文流中以 `<think>...</think>` 包裹的推理内容会被提取并作为思考增量输出，而不再作为普通文本。
  - 流式解析器能正确处理跨多个 SSE 分块的标签，并在流结束时将残留的不完整标签按字面文本处理。


## v1.1.54

### ✨ 新功能

- **Gateway 多工作区会话隔离**
  - HTTP 网关的默认会话（x_session_id 为空时）改为按工作目录（`workDir`）进行隔离，不再共享全局唯一的默认会话，从而防止不同工作区的客户端混用会话上下文。
  - 新增 `OpenByIDExact` 接口，支持忽略当前工作目录限制、直接通过精确会话 UUID 加载并重建会话元数据。
  - 网关内增加了并发会话创建序列化锁，防止客户端并发高频调用时创建出重复的会话。
  - 优化 `/sessions del` 斜杠命令，支持对会话 ID 的前缀模糊匹配，并防止误删当前正在使用的活跃会话。
  - `/clear` 斜杠命令改为清空会话历史消息，但保持会话卡槽（Session Slot）不变，无需重建。

- **PyPI 安装包**
  - 新增 `vibecoding-installer` 的 PyPI 包装器，提供 `vibecoding` 命令入口，并通过内嵌原生二进制的平台 wheel 分发。
  - 新增 `make pypi-*` 发布目标，以及版本同步和 wheel 构建脚本，在流程上对齐 npm 发布方式，同时使用 pip 原生的平台 wheel 选择机制。
  - 更新安装与发布文档，补充 `pipx install vibecoding-installer`。

### 💅 优化

- **更可靠的后备工具调用 ID 生成机制**
  - 后备工具调用 ID（Tool Call ID）生成机制改为“进程级原子计数器 + 高精度时间戳”组合，彻底杜绝高并发多工具调用场景下因 ID 重复而触发的 Anthropic/OpenAI Schema 校验报错。
  - 更新了部分默认模型及配置，并解决了 Gemini 特定的工具调用 ID 唯一性要求。
  - TUI 授权对话框（Auth Dialog）在保存时，现在会完整保留用户自定义的模型参数，而不是重置为厂商默认值。

### 🐛 修复

- **思维深度（Thinking Level）归一化**
  - 引入了 `thinkingLevel` 归一化步骤。当配置值为空或非法时，自动优雅回退至 `medium` 而不是静默禁用思维，从而保证推理模型默认行为符合预期。


## v1.1.53

### ✨ 新功能

- **可嵌入 agent：宿主提供的外部工具**
  - 新增公开的 `agent.ExternalTool` 接口，嵌入方应用可将自身受控能力暴露给 agent，与内置编码工具并存（或完全替代）。
  - 新增 `ExternalToolResult`（文本/错误 + 可选的富 `Contents` 内容块）以及可选的 `ExternalToolPromptInfo` 接口，用于贡献系统提示词信息（`PromptSnippet`、`PromptGuidelines`）。
  - 新增 `Builder.WithExternalTools(...)` 用于注册自定义工具，`Builder.WithoutBuiltinTools()` 用于禁用全部内置工具，从而构建只能使用宿主工具的 agent。
  - 外部工具通过内部 factory 的 `externalToolAdapter` 接入，内部包现在通过 `CreateFromPublicOptions` 从公开 `Builder` 配置构建 agent。
  - 新增 `bootstrap` 包：外部模块只需空白导入 `github.com/startvibecoding/vibecoding/bootstrap` 一次即可注册内部 builder 与 provider 解析 hook（因为内部包无法被直接导入）。

### 💅 优化

- **端到端遵循已配置的 provider 模型**
  - 在公开与内部的 `ChatParams` 中贯穿 `ModelID` 字段，使所选模型能一路传递到 provider 请求。
  - OpenAI 与 Anthropic 兼容 provider 现在会在存在配置时从 provider 配置解析模型列表与 `compat` 标志，否则回退到内置默认值。
  - provider 工厂改为通过 `init` hook 注册，使 `ResolveProvider` 能通过全局 registry 按名称构建 provider，并简化了回退链，对不支持的 `api` 直接报错。

- **Provider 指南文档**
  - 新增 provider 指南（`docs/en/provider-guide.md`、`docs/zh/provider-guide.md`），介绍 provider/vendor 配置。

- **bash 执行体验优化**
  - 将同步执行的 `bash` 默认超时收紧到 45 秒，保留 `async=true` 作为后台任务模式，并将 `timeout=0` 明确为“不设置工具层 deadline”。
  - 更新了 `bash` 的提示语，强调长驻服务应使用 `async=true`，网络探测和其他容易挂住的命令应显式设置超时。
  - TUI 现在会把工具执行拆成“即将运行”和“运行结果”两条独立消息，长命令执行中也能直接看见状态。

- **内部模块拆分**
  - 将 agent、TUI 与命令文件拆分为更聚焦的模块（agent 审批/上下文、TUI 粘贴/渲染、会话/状态行命令），便于维护，行为不变。

### 🐛 修复

- **自定义 provider 认证流程**
  - 修复自定义 provider 认证流程，使其能正确从 API key 步骤推进到模型选择步骤。

## v1.1.52

### 💅 优化

- **Provider HTTP/1.1 fallback 配置**
  - 新增 `providers.<name>.forceHTTP11`，可为单个 provider HTTP client 禁用 HTTP/2。
  - 当代理或 API 网关偶发将 HTTP/2 SSE 流重置并报出 `stream ID ... INTERNAL_ERROR` 时，可用该配置提升稳定性。

- **早期 provider SSE 读流失败遵守 retry 配置**
  - OpenAI 兼容、Anthropic 与 Google 流在尚未输出任何可见内容前遇到暂时性读流错误时，会按已配置的 `retry` 规则自动重试。
  - HTTP/2 `INTERNAL_ERROR` stream reset 现在会被归类为可重试网络错误。
  - 一旦文本、思考、工具调用或 usage 已输出，读流错误仍会立即失败，以避免重复输出。

- **移除内嵌 rg/fd 二进制，切换为纯 Go SDK**
  - 将内嵌的 `rg` 二进制替换为 [`go-ripgrep`](https://github.com/startvibecoding/go-ripgrep) 包。`grep` 工具现在以纯 Go 方式在进程内执行 ripgrep 兼容搜索，不再回退到系统 `grep`。
  - 将内嵌的 `fd` 二进制替换为 [`go-fd`](https://github.com/startvibecoding/go-fd) SDK（`gofd.Find()`）。`find` 工具现在以纯 Go 方式在进程内执行 fd 兼容的文件发现，不再回退到系统 `find`。
  - 删除整个 `internal/vendored/` 包（embed 文件、二进制提取逻辑、`RgPath`/`FdPath`/`Ensure` 辅助函数）以及全部 12 个平台的 `rg`/`fd` 二进制文件（约 42 MB）。
  - 移除 `scripts/prepare-vendored.sh`、`scripts/extract-vendored-tool.sh`、`scripts/download-ripgrep.sh`、`scripts/download-fd.sh` 以及 `pkgs/` 目录（缓存的压缩包）。
  - 移除 Makefile 中的 `prepare-vendored` 和 `test-vendored` 目标；`build`、`build-all`、`test` 不再依赖二进制提取。
  - `bash` 工具不再将 `~/.vibecoding/bin` 注入 `PATH`，因为已无提取的二进制需要暴露。
  - `grep` 和 `find` 仍保持按行输出；无效根路径和搜索初始化错误会直接作为工具错误返回。

- **FreeBSD 编译与打包**
  - 在构建矩阵中新增 FreeBSD `amd64` 和 `arm64`（`make build-freebsd`）、tarball 分发（`make dist-freebsd`），并接入完整的 `make dist` / `make build-all` 流程。
  - 新增 FreeBSD 平台 npm 包（`vibecoding-installer-freebsd-x64`、`vibecoding-installer-freebsd-arm64`）作为可选依赖，并在 npm wrapper 和 `install.sh` 中加入平台识别。
  - FreeBSD 使用纯 Go 的 `grep`/`find` 实现，并回退到 no-op 沙箱，因为 bwrap/seatbelt 仅支持 Linux/macOS。

- **Windows 内嵌 BusyBox 支持**
  - 为 Windows 平台内嵌 `busybox32u.exe` 和 `busybox64u.exe` 资产，运行时解压后作为 `bash` 工具的默认 shell。
  - BusyBox 不可用时回退到 PowerShell。
  - bash 工具输出现在包含运行时标签，指示当前使用的是 BusyBox 还是系统 shell。

- **交互式 Model 选择器**
  - `/model` 不带参数时现在会打开交互式选择对话框，而非以纯文本列出模型。
  - 支持搜索过滤、方向键导航、当前模型指示，以及回车切换。

- **ccstatusline 原生支持**
  - 新增 `statusLine` 配置（`type`、`command`、`padding`、`refreshInterval`、`timeoutMs`、`fallback`），用于外部状态行渲染器。
  - 以 Claude 兼容的 JSON stdin payload 执行状态行命令；支持多行输出、ANSI 颜色和 OSC 8 超链接。
  - 新增 `/statusline` 斜杠命令（`on`/`off`/`status`/`test`/`refresh`），可在运行时控制状态行。

## v1.1.51

### ✨ 新功能

- **新增 Provider: 火山引擎 (Volcengine)**
  - 新增火山引擎 Provider，通过方舟 API 平台接入豆包 Seed 系列模型。
  - 支持模型：豆包 Seed 2.1 Turbo（`doubao-seed-2-1-turbo-260628`，256K 上下文，纯文本）、豆包 Seed Evolving（`doubao-seed-evolving`，256K 上下文，文本+图片）、豆包 Seed 2.1 Pro（`doubao-seed-2-1-pro-260628`，256K 上下文，文本+图片）。
  - 使用 OpenAI 兼容 API 端点 `https://ark.cn-beijing.volces.com/api/v3`。
  - 通过 `ark.cn-beijing.volces.com` 域名自动识别供应商。

- **SQLite 会话存储**
  - 新增和恢复会话统一使用 SQLite（`modernc.org/sqlite`），提升查询性能和元数据管理能力。
  - 对于 CLI 和 Gateway，所有会话的元数据和条目日志均存储在单个统一的 `sessions.db` 数据库文件中，列表/切换/删除时使用虚拟的 `.db` 路径句柄；只有 Hermes 会在用户目录下写入物理的会话句柄文件（如 `active.db` 与归档的 `*_corrupt.db`）。
  - `OpenByID` 和 `OpenByPathOrID` 新增快速精确/前缀匹配，支持歧义检测，并可直接基于统一 SQLite 数据库还原会话结构。
  - ACP 历史重放现在会在加载存储对话历史时流式传输工具执行事件（`toolCall`/`toolResult`）。
  - `DeleteSession` 清理 SQLite 中的会话与条目记录，并在物理句柄文件存在时（如 Hermes）将其删除，同时拒绝将共享的 `sessions.db` 作为会话句柄删除。
  - Hermes 现在使用 `active.db` 会话物理句柄，损坏会话归档为 `*_corrupt.db`，并移除旧版 `active.jsonl` fallback。
  - 移除旧版 JSONL 加载/写入路径，新增和恢复会话仅使用 SQLite。

### 🐛 Bug 修复

- **ACP Systeminit Plan Mode 写权限**
  - 修复 ACP systeminit 在 plan 模式下允许文件写入，使 TUI/ACP 可在使用 `/systeminit` 生成 `AGENTS.md` 时不受模式限制错误影响。

### ✨ 新功能

- **`/systeminit` 与 `/reload` 指令**
  - 新增 `/systeminit`：生成或刷新项目级 `AGENTS.md`。在 TUI、ACP 以及 `vibecoding systeminit` CLI 子命令中均可用。TUI 与 ACP 下会启发式地使用 `question` 工具先向用户提问几个关键问题，再生成更优质的 `AGENTS.md`；CLI 为非交互式直接生成。支持传入附加说明，例如 `/systeminit 用中文提问我，用英文写 AGENTS.md`。
  - `question` 工具现在在 `agent` 模式下也可用（以前仅 plan），并为 ACP 服务器注册，ACP 通过 `session/request_permission` 通道呈现问题。
  - 新增 `/reload`（TUI）：以全新进程重启并开启新 session，重新加载配置、上下文文件、skills 与 MCP，等同于重新启动程序。

- **Mode 边界增强：`/btw` 旁路问答 + 可编辑路径白名单 + 全自动编辑**
  - 新增 `/btw <问题>`：在不中断主任务的前提下，继承主任务对话历史（只读）快速启动一个一次性 sub-agent 回答临时问题。答案显示在临时浮动层，不写回主 session，不增加主任务上下文窗口占用与 token 统计；sub-agent 仅拥有只读工具（read/grep/find/ls/skill_ref）。主历史过长时会自动裁剪注入快照以控制旁路开销。
  - 新增 `/alloweditpath [add <glob>|remove <glob>|clear]`：维护可编辑路径白名单（支持 `**`/`*` 通配符），agent 模式下命中白名单的 `write`/`edit` 无需逐次申请、自动放行。
  - 新增 `/allowautoedit [on|off] [global]`：打开 agent 模式下的全自动编辑（相当于只有 bash 需要申请权限）。
  - 白名单与全自动开关落盘到独立的 `allow.json`：`/alloweditpath` 与默认的 `/allowautoedit` 写项目级 `.vibe/allow.json`；`/allowautoedit on global` 写全局 `allow.json`。加载顺序为全局→项目覆盖（`editPaths` 仅项目级）。新会话启动时自动载入。
  - 仅放宽审批层，不改变 sandbox / allowedWorkDirs 物理边界，也不改变 plan / yolo 语义。

- **基于 npm 接口的版本更新检测**
  - VibeCoding 现在会通过 npm registry（`vibecoding-installer`）检测是否有新版本，并在启动时给出非阻塞的更新提醒。
  - 网络检测在后台进行（最多每 24 小时一次），仅刷新本地缓存（`update-check.json`），前台不会因网络请求而阻塞。
  - 提醒会显示在 TUI 启动信息中，`--print` 模式下输出到 stderr，并提示执行 `npm install -g vibecoding-installer@latest`。
  - 可在配置文件 `settings.json` 中设置 `"updateCheck": false` 关闭，也可通过 `VIBECODING_NO_UPDATE_CHECK=1` 关闭；通过 `VIBECODING_NPM_REGISTRY` 覆盖 registry 地址。

### 📚 文档

- 更新会话文档、CLI 示例、FAQ 清理建议、架构图、Hermes 文档和 README 功能摘要，说明 SQLite 存储、`.db` 句柄文件和 Hermes `active.db` 会话。
- 新增内置火山引擎/豆包 provider 配置文档，并刷新 provider 适配器列表，加入火山引擎、Mistral、GitHub Copilot、Cloudflare 和 Amazon Bedrock。

### 💅 优化

- **TUI 头部与底部美化**
  - 放大 ASCII logo 并在头部区域垂直居中显示。
  - 弱化底部分隔线，并统一模式/模型/路径的配色，界面更清爽。

## v1.1.50

### ✨ 新功能

- **流式 Delta Builder 优化**
  - 用 `strings.Builder` 替代字符串拼接来累积助手和思考文本 delta，避免长回复时 O(n²) 的内存增长。
  - Builder 在轮次结束、审批和错误事件时先 finalize 再打印，确保输出一致性。

- **新增 Provider: Mistral**
  - 新增 Mistral AI Provider，支持模型包括：Mistral Large、Mistral Medium 3.5、Mistral Small、Codestral、Devstral、Magistral Medium/Small 和 Pixtral Large。
  - 使用 OpenAI 兼容 API 端点 `https://api.mistral.ai/v1`。

- **新增 Provider: GitHub Copilot**
  - 新增 GitHub Copilot Provider，支持 Claude Sonnet 4.6/4.5、Claude Opus 4.8、Claude Haiku 4.5、Claude Fable 5、GPT-5.5/5.4/5.2、Gemini 2.5 Pro 和 Gemini 3.5 Flash 模型。
  - 使用 OpenAI 兼容 API 端点 `https://api.individual.githubcopilot.com`。

- **新增 Provider: Cloudflare AI Gateway**
  - 新增 Cloudflare AI Gateway Provider，支持 Claude、GPT、Gemini 和 Llama 4 Scout 模型。
  - 支持通过 Cloudflare AI Gateway 路由来自 Anthropic、OpenAI、Google 和 Meta 的模型。

- **新增 Provider: Cloudflare Workers AI**
  - 新增 Cloudflare Workers AI Provider，支持 Llama 4 Scout 17B、Llama 3.3 70B、Gemma 4 26B、Mistral Small 3.1 24B、GPT OSS 120B/20B、Kimi K2.7 Code 和 GLM 5.2 模型。
  - 使用 Cloudflare Workers AI 推理端点。

- **新增 Provider: Amazon Bedrock**
  - 新增 Amazon Bedrock Provider，支持 Claude Sonnet 4.6/4.5、Claude Opus 4.8、Claude Haiku 4.5、Claude Fable 5、Amazon Nova Pro/Micro/Lite 以及 DeepSeek V3.2/R1 模型。
  - 使用 OpenAI 兼容跨区域推理端点。

- **紧凑 TUI 底栏与输入分隔线**
  - 将 mode、model 和 path 合并为单行底栏（原来 3 行）。
  - 在 transcript 和输入区域之间新增半块分隔线，增强视觉区分。
  - 编辑器光标和 placeholder 样式新增背景色。
  - npm 新增 postinstall 脚本，安装后显示快速开始信息。

### 🐛 Bug 修复

- **TUI 输入框宽度对齐**
  - 修复输入框宽度与上方分隔线不对齐的问题，布局更一致。
  - 编辑器宽度设为完整终端宽度以匹配分隔线。
  - 修复编辑器 Width 计算中双重 padding 扣减的问题，改用 `m.width` 作为最终渲染宽度。

- **TUI `compactBashOutput` 尾部空白**
  - 修复 `compactBashOutput` 在空行去重后写入原始未 trim 行而非 trim 后行的问题，避免保留尾部空白字符。

- **TUI Program 模式下转录内容重复**
  - 当 Bubble Tea program 活跃时清空受管 liveContent，避免通过 `Program.Println` 打印到原生 scrollback 的已完成转录块在 live 视图中重复显示。

- **Sandbox 状态标签**
  - 移除"无沙箱"状态显示中多余的 "YOLO mode" 文本。

---

## v0.1.47

### ✨ 新功能

- **扩展模型目录**
  - Anthropic 新增模型：Claude Opus 4.8、Claude Opus 4.1、Claude Opus 4、Claude Sonnet 4.0、Claude Haiku 4.5、Claude Fable 5，以及 Claude 3 系列遗留模型。
  - OpenAI 新增模型：GPT-5.5、GPT-5.5 Pro、GPT-5.4 系列、GPT-5.3 Codex/Spark、GPT-5.2 Pro/Codex、GPT-5.1 Codex 系列、GPT-4.1 系列、o4-mini、o3/o3-pro/o3-deep-research、o1-pro，以及 GPT-4 系列遗留模型。
  - OpenRouter 新增模型：Claude Sonnet 4.6/4.5、Claude Opus 4.8、Claude Haiku 4.5、GPT-5.5/5.5 Pro/5.4、Gemini 3.5 Flash/2.5 Pro、DeepSeek V4 Flash/Pro、Qwen 3.7 Plus、Kimi K2.7 Code、MiniMax M3、Llama 4 Scout、GLM 5/5.2、Grok 4.3、GPT-OSS-120B（免费）。
  - Vercel AI Gateway 新增模型：Claude Sonnet 4.6/4.5、Claude Opus 4.8、Claude Haiku 4.5、GPT-5.5/5.4、Gemini 3.5 Flash、DeepSeek V4 Flash/Pro、Qwen3.6 Plus、MiniMax M3、Kimi K2.7 Code、Grok 4.3、GLM 5.2。
  - Anthropic 和 OpenAI 模型列表重新排序，最新模型排在前面。

### 🐛 Bug 修复

- **TUI 审批详情在实时视图中可见性**
  - 修复排队审批请求在等待用户输入时不在实时 transcript 中显示详情的问题。
  - 现在跟踪当前审批消息索引，确保审批提示期间保持可见。
  - 审批完成后正确清除索引，并在状态重置/清除路径中重置。

- **TUI 工具弹窗性能与显示**
  - 工具弹窗渲染输出新增行级缓存，避免每次渲染时重新解析完整 transcript。
  - 新增按条目缓存展开的工具结果，避免重复格式化。
  - 所有 transcript 状态变更点现在都会调用 `invalidateToolModalCache()` 保持缓存一致。
  - 修复展开视图中 edit 工具结果重复显示 diff 片段的问题，提取了专用的 edit header 格式化函数。
  - 工具弹窗现在默认在顶部（offset 0）打开，而不是滚动到底部。

### 🧪 测试

- 新增回归测试，验证展开的 edit 输出不会重复 diff 片段。

---

## v0.1.46

### ✨ 新功能

- **Workflow Agent 实例 Key**
  - 新增重复逻辑 workflow agent 的 `:key`，有界 `while` 循环可以保持 agent 名称为字面量，同时将每轮结果保存为 `phase.agent[key]`。
  - 新增 `result-key`、`result-latest`，并支持 `(result "phase.agent" :key "r0")`，用于显式读取 keyed 结果或最新实例结果。
  - Keyed workflow worker 使用带实例的运行时 ID，例如 `agent-worker[r0]`，避免循环中的重复 worker 碰撞，同时保留稳定的逻辑 agent 名称。

- **Workflow Lint 工具**
  - 新增 `workflow_lint`，可在不运行 worker agents 的情况下验证 workflow Elisp DSL。
  - Lint 会检查 Elisp 语法、workflow/phase/agent 表单、关键字参数、必需 prompt，以及 result 引用。
  - 将 lint 工具与 workflow run/status/cancel 工具一起注册，并更新 workflow prompt 指引：非平凡的生成或修改后 workflow 应先 lint 再执行。

- **可配置上下文压缩**
  - 新增 `tokenizer`、`tokenizerModel` 和 `template` 压缩配置，并贯通 CLI、print 模式、ACP、Gateway、Hermes、TUI 模式切换和 delegate agent factory。
  - 新增内置压缩摘要模板：`default`、`code` 和 `conversation`，长会话可按任务类型保留更合适的 checkpoint。
  - 引入 token 估算器抽象，同时保持 `auto` 和 `generic` 使用现有 chars/4 通用估算器。
  - Compaction entry 现在记录 summary version、previous compaction ID 和 last summarized entry ID，便于 session replay 与调试。

### 🐛 Bug 修复

- **Context Compaction Replay**
  - Print 模式现在会在运行 agent 前恢复 session replay 历史，保留此前对话上下文。
  - 手动和强制 compaction 现在会检查是否真的存在可压缩的旧历史，避免只剩近期上下文时仍触发压缩。
  - Replay 已压缩消息时会移除保留消息中的旧 usage metadata，避免过期 token 统计泄漏到后续运行。

- **并发文件写入**
  - 新增进程级内存文件锁管理器，默认 tool registry 共享同一个管理器。
  - `write` 和 `edit` 在读取和修改文件前会获取按文件粒度的锁，避免多个 agent 并发写同一目标文件时互相交错覆盖。
  - 等待锁时支持 context 取消和 deadline；等待被中断时会报告当前锁持有者。

### 🔧 重构

- **预发布包发布**
  - `npm-publish-pre` 现在会先同步并使用 `-pre` 版本后缀构建 npm packages，再发布预发布包。
  - 更新 npm package metadata 和各平台 optional dependency 版本为预发布版本。

- **命名 Workflow Worker Agents**
  - Workflow worker agent 现在使用由 DSL agent 名称派生的确定性 ID（`agent-<name>`），改善事件归属和后台 agent 可见性。
  - Workflow skill 指引已记录该 ID 映射，并建议在同一个 workflow 内保持 agent 名称唯一。

### 📚 文档

- 更新 Workflow 模式文档、工具参考和 `workflow-elisp` skill，记录 `:key`、keyed 结果读取，以及有界 while 循环写法。
- 记录 context compaction 的 `tokenizer`、`tokenizerModel` 和 `template` 配置，包括内置模板选项，以及 idle compaction 设置当前为预留/弃用字段的状态。
- 澄清 Ctrl+O 详情弹窗中的按键提示，包括切换目标、翻页、滚动和关闭。
- 记录 TUI scrollback 的取舍：已完成 transcript block 会打印到原生终端 scrollback，以保证选择和历史滚动稳定；用户输入仍应按 block 打印，而不是无缓存流式输出，避免干扰 Bubble Tea 的 live view 重绘。

### 🧪 测试

- 新增 workflow runner、lint、集成和 skill 覆盖，验证 keyed 重复 agent 和 keyed result 查询。
- 新增 context compaction 测试，覆盖自定义 token estimator、模板解析、配置化摘要 prompt、compaction metadata、可压缩性检查，以及 session replay usage 清理。
- 新增 Gateway 和 Hermes 测试覆盖 `/compact` 在只剩近期上下文可保留时的行为。
- 新增 workflow lint 测试，覆盖有效 source 收集和缺失 result 引用错误。
- 新增 workflow 集成测试，验证 DSL agent 名称会反映到运行时 worker agent ID。
- 新增文件锁测试，覆盖等待/取消行为、默认管理器共享，以及 `write`/`edit` 的 context 处理。

---

## v0.1.45

### ✨ 新功能

- **Workflow Skill 渐进式参考文档**
  - 将 workflow Elisp/DSL 文档从 system prompt 中提取为独立的 `workflow-elisp` skill，减少 system prompt 体积。
  - 引入渐进式参考结构：skill 索引页列出 9 个参考文件，按需加载，核心规则默认加载。
  - 8 个模式指南：研究与调研、串行与并行组合、决策路由、有界 While 循环、水平多 Agent 协作、主从小团队、评估优化器评审轮次、治理与人审检查点。
  - 每个参考文件包含可直接复制的 Elisp 骨架示例和模式选择指引。
  - `EnsureProjectSkill` 自动在项目 `.skills/workflow-elisp/` 下创建 skill 和所有参考文件，不覆盖用户已有的自定义内容。
- **Workflow 超时控制**
  - `workflow_run` 新增可选 `timeoutSeconds` 参数，有明确上限的长 workflow 可设置合适的超时时间；需要持续运行的 workflow 可设置为 `0`，避免触发默认 agent 级 deadline。

- **vibeEmacsLispVm v0.0.2 升级**
  - `vibeEmacsLispVm` 依赖从 v0.0.1 升级到 v0.0.2，扩展了 Elisp 支持范围。
  - 新增 backquote/comma、`let*`/`while`/`cond`/`catch`/`throw`/`lambda`/`defun`/`defmacro`/`with-current-buffer`/`save-current-buffer` 等特殊形式支持。
  - 新增内置函数：`cons`/`car`/`cdr`/`nth`/`append`/`reverse`/`member`/`assoc`/`funcall`/`apply`/`macroexpand`、算术与谓词函数、以及内存缓冲区 + marker 内置函数。
  - 新增 v0.0.2 Elisp 特性的全面测试覆盖。

### 🔧 重构

- **Gateway Session 级 Skills 支持**
  - Gateway session 现在支持独立的 `SkillsMgr` 和 `ExtraContext`，使 delegate 子 Agent 继承 session 级状态。
  - `/skill` 和 `/skills` 命令改为操作 session 级 skills，而非全局 server 级。

- **System Prompt 精简**
  - Workflow Elisp VM 语法和 DSL 表单的详细说明从 system prompt 移除，改为引用 `workflow-elisp` skill。
  - system prompt 中仅保留关键约束和调用说明，显著减少 token 占用。

- **Workflow Skill 参考文件职责澄清**
  - 重命名参考文件标题以更清晰："连续循环与迭代任务" → "有界 While 循环"，"评估优化器与评论家循环" → "评估优化器评审轮次"。
  - 拆分模式选择指引：有界 while 循环用于带停止条件的运行时重复；评估优化器用于单轮草稿/评审/修订流水线。
  - 新增约束：不要用编号 phase 模拟循环。
  - 渐进式参考状态标签统一为英文（"loaded" / "load on demand"），保持与 skill 其余内容一致。

### 📚 文档

- 新增 Workflow 模式使用指南和最佳实践文档（中英文），覆盖快速入门、核心概念、常见模式和避坑指南。
- 同步各文档页面的 workflow 引用：在功能概览中新增动态 Workflow 章节，在使用场景中新增 workflow 编排场景，并从工具参考文档添加交叉链接。
- 在 `workflow-elisp` skill 和文档中澄清 workflow 隐式默认值与限制：worker `:max-iterations` 默认值和失败行为、`workflow_run timeoutSeconds`、`concurrency` 默认值、继承的 `:mode`、默认 `:tools`、当前工作目录行为、禁用嵌套编排，以及不支持的 per-worker 选项。

### 🧪 测试

- 新增 workflow skill 测试，验证 skill 文件和 8 个参考文件的创建、不覆盖已有文件、缺失引用自动补全。
- 扩展 workflow runner 和 lisp 测试覆盖。
- 新增参考内容清晰度测试，验证循环与评估优化器模式不重叠。

---

## v0.1.44

### ✨ 新功能

- **Dynamic Workflows**
  - 新增独立 `--workflows` 模式，支持 CLI、ACP 和 Gateway，不依赖 `--multi-agent`。
  - 新增 Elisp workflow 工具：`workflow_run`、`workflow_status` 和 `workflow_cancel`。
  - Workflow runtime 支持 phase、series/parallel 执行、并发限制、worker agent 任务、结果汇总和运行日志。
  - 新增 workflow run 状态持久化，并在 TUI 与 Gateway 中提供 `/workflows` 状态命令。
  - 新增进程内 active run 取消能力，`workflow_cancel` 和 `/workflows cancel <id>` 可中断运行中的 workflow。

- **Z.AI 供应商适配器**
  - 新增 `vendor_zai.go`，注册 `zai` 供应商适配器，域名 `api.z.ai` 和 `open.bigmodel.cn`，设置 `thinkingFormat: zai`。
  - 更新 `zai` 和 `zai-coding-cn` 供应商配置：设置 `Vendor: "zai"`、`ThinkingFormat: "zai"`，更新 base URL 为 coding 端点，新增 `glm-5v-turbo` 视觉模型。

- **Kimi 供应商更新**
  - `kimi` 供应商适配器新增 `api.kimi.com` 域名，支持自动供应商检测。
  - `kimi-coding` 供应商配置新增 `User-Agent: KimiCLI/1.5` 请求头。
  - `moonshotai`、`moonshotai-cn`、`fireworks`、`opencode-go` 供应商新增 Kimi K2.7 Code 和 K2.7 Code HighSpeed 模型。

- **新模型**
  - `opencode-go` 供应商新增 `GLM-5.2` 模型（1M 上下文窗口，262K 最大输出）。
  - `fireworks` 供应商新增 Kimi K2.7 Code Fast 模型。

### 🐛 Bug 修复

- **TUI Agent 事件处理**
  - 修复了在流式响应中途发生错误事件时，部分响应文本未提交到终端 scrollback 的问题，确保部分内容不会丢失。
  - 新增回归测试，验证错误发生时流索引和打印队列行为正确。

- **版本号字符串**
  - 修复 `Makefile` 中 `git describe` 使用 `--abbrev=0`，确保生成干净的标签版本号，不附带 commit 数量和 hash 后缀。
  - 修复 `sync-npm-version.sh`，去除版本号中的 commit 数量和 hash 后缀。
  - 更新 `npm/bin/vibecoding`，使用 GitHub raw URL 作为安装脚本的 fallback。

### 🔧 重构

- **Agent Manager 确定性排序**
  - `AgentManager.List` 现在按启动时间和 ID 排序，确保列表顺序稳定可预测。
  - 提取 `resetAgent`/`abortAndResetAgent` 辅助函数，减少 TUI commands 中的代码重复。
  - TUI 创建 Agent 时现在会在 config 中设置 Agent ID。

### 📦 依赖

- 新增 `github.com/startvibecoding/vibeEmacsLispVm v0.0.1`，作为 workflow DSL 执行使用的内嵌 Elisp 子集解释器。

### 📚 文档

- 在 `docs/proposal/` 下新增 dynamic workflows Elisp 方案文档。
- 更新中英文工具文档，补充 workflow 工具用法、仅支持 Elisp DSL 的约束和取消范围说明。

### 🧪 测试

- 新增 workflow runner/store/tool 测试，覆盖 Elisp 执行、并行 worker、结果汇总、持久化、工具注册隔离和 active run 取消。
- 新增 prompt 与 CLI flag 测试，确认 workflow 模式不会污染 multi-agent、delegate 或 worker agent prompt。
- 新增 `VendorFromBaseURL` 测试用例：`api.kimi.com`、`api.z.ai`、`open.bigmodel.cn`。
- 新增 agent manager 测试，验证列表排序的确定性。

---

## v0.1.43

### 🐛 Bug 修复

- **TUI 输入 Flush**
  - 修复 TUI 中 `flushInputQueue` 未将其返回值作为 `tea.Cmd` 返回的问题，确保排队的按键在处理 `Enter`、`Tab`、`Up`、`Down` 等事件前正确刷新。

### 🔧 重构

- **移除未使用的 `mergeSettings`**
  - 移除未使用的 `mergeSettings()` 函数及相关测试；项目 settings 合现已由 `LoadSettings` 直接处理。
  - 重写 `settings_zero_test`，改为通过实际文件 I/O 调用 `LoadSettings()` 进行测试，而非直接 JSON 反序列化。

### 📦 依赖

- **GoStreamingMarkdown 更新**
  - 将 `github.com/startvibecoding/GoStreamingMarkdown` 从 `v0.0.2` 更新到 `v0.0.3`。

### 🧪 测试

- 新增测试验证 `Enter` 键在应用命令建议前先刷新排队输入。

---

## v0.1.42

### ✨ 新功能

- **TUI 多行输入**
  - 将 prompt 输入框替换为可复用的 TUI editor 组件，支持真正的多行 prompt 编写。
  - `Alt+Enter` 和 `Ctrl+J` 现在用于插入换行；`Enter` 仍用于提交 prompt。
  - 小型多行粘贴现在会保留换行，不再压平成空格；大型粘贴仍使用 paste marker。
  - `Up` / `Down` 会优先在多行输入内移动光标，只在输入边界处浏览 prompt 历史。

### 🐛 Bug 修复

- **TUI 输入编辑**
  - `Home` / `End` 编辑键现在能正确传递到输入 editor，不再被顶层 TUI 处理吞掉。
  - 修复在排队按键尚未 flush 时浏览 prompt 历史会丢失当前草稿的问题。
  - `/clear` 在输出清空确认后会重置 printed-message 记录，避免清空对话后复用陈旧的 transcript 打印状态。

### 📚 文档

- 更新 TUI 键盘快捷键文档，补充多行输入、插入换行、历史导航和工具 Modal 行为说明。

### 🧪 测试

- 新增多行 prompt 输入、`Alt+Enter` / `Ctrl+J`、小型多行粘贴保留、prompt 历史边界导航、Home/End 输入编辑和 `/clear` transcript 状态重置相关测试。

---

## v0.1.41

### ✨ 新功能

- **TUI 界面重设计**
  - 新增启动 Header，展示 Vibe Logo、版本、Provider/Model 和当前工作目录。
  - 重设计 Footer，展示模式、模型、cwd、当前/上次请求耗时、沙箱、上下文窗口用量、缓存指标和快捷键提示。
  - Agent 运行时新增内联 Loading 指示器，包含 spinner、耗时和取消提示。
  - `plan` 工具新增 sticky todo list，只展示未完成步骤，长任务执行时仍能持续看到当前计划。
  - 多 Agent 场景新增 Agent Tab Bar，当存在多个 Agent 时展示活跃 Agent 和状态。
  - 新增紧凑工具显示模式，可用 `Ctrl+G` 切换；工具输出折叠为单行摘要，详细内容仍可通过 `Ctrl+O` 查看。

- **终端原生 Scrollback Transcript**
  - 已完成的 transcript block 现在通过 Bubble Tea `Program.Println` 打印到终端原生 scrollback，只把实时流式内容保留在受管理的 TUI 视图中。
  - 改善长对话中的鼠标滚动、终端选择/复制和历史查看体验。

- **TUI 组件基础设施**
  - 在 `internal/tui/components/` 下新增可复用的 editor、suggestion list 和 vertical scroll 组件，包含 CJK 感知的 buffer 与渲染行为。

- **回复格式指南**
  - System prompt 新增格式约束，减少不必要的粗体、标题和列表，除非用户明确要求或内容确实需要结构化表达。

### 🐛 Bug 修复

- **TUI 工具结果打印**
  - 工具结果更新现在通过一次性 transcript 打印路径输出，而不是只刷新内存中的 live content，避免已完成工具输出从终端 scrollback 中消失。

- **Viewport 清理**
  - 在 TUI 历史迁移到终端原生 scrollback 后，移除过时的 viewport 状态重置逻辑。

### 📦 打包

- 更新 npm installer 包元数据及各平台 optional package 版本到 `0.1.40`。

### 🧪 测试

- 新增 TUI editor、suggestion list 和 vertical scroll 模型的组件测试。
- 更新 TUI cache/render 测试，覆盖 Header、Footer、终端原生 scrollback transcript 打印、紧凑显示模式和简化后的 viewport 行为。

---

## v0.1.40

### ✨ 新功能

- **GoStreamingMarkdown 渲染器**
  - 将 print 模式、本地 TUI 和 Hermes 远程 TUI 的 Markdown 渲染从 Glamour 替换为 `github.com/startvibecoding/GoStreamingMarkdown`（`gsm`）。
  - 移除本地模块替换，改为直接依赖远程 `github.com/startvibecoding/GoStreamingMarkdown` 模块。

- **Delegate 模式（阻塞式单子 Agent 委托）**
  - 新增 `delegate_subagent` 工具：同步运行一个子 Agent，等待其完成并返回摘要结果，同时限制同一时间只能运行一个 delegate。
  - Root、ACP、Gateway 命令新增 `--delegate` CLI 参数。
  - TUI 与 Gateway 新增 `/delegate [on|off|status]` 斜杠命令。
  - Gateway 配置（`gateway.json`）新增 `enableDelegate` 选项。
  - System prompt 新增专门的 **Delegation Mode** 章节，包含上下文成本启发式、正反示例和结果解读指南。
  - 新增 `AgentFactoryOptions.DelegateEnabled`，便于程序化启用。
  - 子 Agent system prompt 现在使用结构化汇报格式（`Result`、`Evidence`、`Changes`、`Risks`），并补充负向搜索结果、测试执行和简洁性的说明。
  - `delegate_subagent` 结果中新增 `tool_calls` 计数和按工具名统计的 `tool_breakdown`。

- **子 Agent 执行模式继承**
  - `subagent_spawn` 与 `delegate_subagent` 现在会继承父 Agent 的执行模式（`plan`/`agent`/`yolo`），不再硬编码为 `agent`。
  - `executeSingleToolCall` 通过 context 注入父 Agent 模式。
  - 子 Agent policy 的 `AllowedModes` 从 `["agent"]` 扩展为 `["plan", "agent", "yolo"]`。

- **多 Agent 审批处理改进**
  - 移除了容易造成死锁的 `newApprovalForwarder`（基于 Mutex pending map 的同步审批 handler）。
  - 子 Agent 审批请求现在通过事件通道（`sendParentEvent`）转发，不阻塞工具执行。
  - TUI 在 `pendingApproval` 中跟踪 `agentID`，并通过 `handleApprovalResponse` 将审批响应分发给正确的子 Agent。

### 🐛 Bug 修复

- **TUI 中止原因显示**
  - 当 TUI Agent 会话被中止（用户按 Esc 或切换模式）时，错误信息现在会包含中止原因，例如：`"Error: aborted (reason: user pressed Esc)"`。
  - TUI `App` 结构中新增 `pendingAbortReason`，并在 `EventError` 后清理。

### 🧪 测试

- 更新 TUI Markdown 渲染断言以匹配 `gsm` 行为，同时保留内容完整性和视口宽度限制的覆盖。
- 新增 `TestDelegateSubAgentTool` 和 `TestDelegateSubAgentToolMissingTask`，验证阻塞式子 Agent 委托。
- 更新 `TestSubAgentPolicyDefault`，期望扩展后的允许模式 `["plan", "agent", "yolo"]`。
- 更新 `TestAgentManagerEnforcesSubAgentPolicy`，默认允许 `yolo` 模式。
- 新增 `TestAgentErrorIncludesAbortReason`，覆盖 TUI 中止原因渲染。

---

## v0.1.39

### ✨ 新功能

- **子 Agent 继承 Skills 与 Plan 工具设置**
  - `AgentFactory` 新增 `skillsMgr` 参数，子 Agent 现在自动继承父会话的 `skill_ref` 工具。
  - `RegistryConfig` 新增 `EnablePlanTool` 字段，`settings.json` 中的 `enablePlanTool` 设置现在会正确传播到子 Agent 注册表，确保父子 Agent 之间工具可用性一致。

- **会话压缩重放状态持久化**
  - 会话现在会持久化压缩重放状态（`ReplayState`），压缩后的会话可以在重启后正确恢复。
  - `LoadHistoryState` / `GetHistoryState` 跟踪每条消息的 Session Entry ID，使压缩边界（`firstKeptEntryID`）在会话重载后不会丢失。
  - Gateway 和 Hermes 现在使用 `LoadHistoryState` 替代 `LoadHistoryMessages`，确保压缩后的会话重放准确。
  - TUI 在手动压缩期间阻塞用户输入，将 `agentStartMsg`/`compactionStartMsg` 合并为统一的 `agentStreamStartMsg`。


- **TUI 视口重写与 CJK 支持**
  - 使用 `bubbles/viewport` 和 `charmbracelet/x/ansi` 替换自定义 ANSI 解析和换行逻辑，修复快速流式输出时 CJK/ASCII 混合文本的字符交错问题。
  - Think 消息与 Assistant 消息现在使用独立的渲染路径。
  - 长路径和 URL 现在能正确按词换行，不会在 token 中间截断。
  - Markdown 渲染不再溢出视口宽度。
  - 新增 `renderutil` 工具包，包含混合 CJK/ASCII 换行、ANSI 序列完整性和完整文件渲染的全面测试覆盖。

- **Grep 工具输出限制**
  - `grep` 工具在流式输出时现在会限制输出大小，防止大型结果集消耗过多上下文。

### 🐛 Bug 修复

- **Google 工具结果分组**
  - 修复 Google/Gemini Provider 的工具结果分组逻辑，正确配对工具调用与其结果。

- **TUI Compact 命令**
  - `/compact` 命令现在立即触发压缩，不再等待下一个 Agent 轮次。

- **Agent 退出路径一致性**
  - 抽取 `agentEndEvent()` 辅助函数，消除 8 个退出路径中的重复事件发射代码。
  - 修复 Session 保存错误路径上缺失的 `EventAgentEnd` 事件。
  - 在 `ShouldStopAfterTurn` 路径中补充缺失的 `usage`/`contextUsage` 元数据。

### 🧪 测试

- 新增 `TestAgentFactorySubAgentsRespectPlanToolSetting` 测试，验证 `enablePlanTool` 设置正确传播到子 Agent 注册表。
- 新增 `TestAgentFactorySubAgentsRegisterSkillRef` 测试，验证配置了 Skills Manager 时子 Agent 继承 `skill_ref` 工具。
- 新增 `agentEndEvent` 一致性和 `ShouldStopAfterTurn` 元数据的测试。
- 新增 TUI 固定高度渲染、CJK 换行和 ANSI 完整性的全面测试。

---

## v0.1.38

### ✨ 新功能

- **自定义 Provider 模型自动回退（Fallback）**
  - 当显式指定了自定义 Provider（通过 CLI、Gateway 或 Hermes）但没有指定 Model ID 时，Factory 现在会自动回退并使用该 Provider 下的**首个可用模型**，而不是错误地采用 `settings.DefaultModel`（默认模型通常属于默认 Provider）。
  - 避免了在使用非默认 Provider 且未指定具体模型时，因加载了不匹配的全局默认模型而导致“未找到模型”的错误。

- **Hermes 默认配置解析优化**
  - 优化了 Hermes 的 `GetDefaultModel` 逻辑：当 `hermes.json` 中配置了 `DefaultProvider` 但 `DefaultModel` 留空时，系统现在能正确返回空字符串，从而触发上述自定义 Provider 首选模型的回退逻辑，不再强行透传 `settings.json` 中的全局默认模型。

- **完善公开的 Agent SDK 包与示例代码**
  - 完整补全了公开 `agent` 包与底层实现之间的流事件字段桥接映射（支持 `Messages`、`TurnMessage`、`TurnToolResults`、`Message`、`ToolCall`、`ToolDiff`、`ToolError`、`PartialResult`、`Plan`、`Usage` 和 `ContextUsage`）。
  - 修复了因内部流事件特有类型（如 `StreamThinkSignature`）导致的公开 `StreamEventType` 枚举下标错位问题，实现了健壮的显式双向转换。
  - 实现了 `PublicProviderAdapter` 适配器，将内部 Provider 无缝桥接到公开 `agent.Provider` 接口，并在初始化时自动关联，完美规避了 Go 包循环导入问题。
  - 在顶层新增了 `example/` 目录，设计并编写了两个极具代表性的高阶演示程序（`simple_agent` 和 `custom_provider`）以及详尽的中英文双语 `README` 文档，直观演示了如何自定义 LLM 后端、挂载内置工具框架并启动流式 Agent。

### 🧪 测试

- 在 `internal/provider/factory_test.go` 中新增 `TestCreateFallbackToFirstModel` 测试，覆盖当模型 ID 为空时，自定义 Provider 和内置 Provider 自动回退到其首选模型的行为。
- 在 `internal/hermes/config_test.go` 中新增针对 `GetDefaultModel` 方法的测试用例，覆盖 Hermes 配置中仅指定 `DefaultProvider` 时的行为。

---

## v0.1.37

### ✨ 新功能

- **Vertex AI API Key 认证支持**
  - 新增对 Google Vertex AI 使用 API key（通过 `x-goog-api-key` 请求头）直接认证的支持，无需强制要求使用 gcloud OAuth 凭证（`ya29.`）。
  - 当配置了 API key 时，默认 baseUrl 会自动路由到 `https://aiplatform.googleapis.com/v1/publishers/google/models`（无需传入 project/location 参数）。
  - 完美保持对现有 OAuth bearer token 的向下兼容。

### 🐛 Bug 修复

- **Google 工具思维签名（Thought Signatures）**
  - 正确提取并在工具调用中传递 Gemini 的思维/推理签名，避免签名丢失或不匹配的问题。

- **TUI 原始 Bash 输出保留**
  - 在 Bubble Tea TUI 中渲染 `bash` 工具执行结果时，完美保留 ANSI 转义着色和原始的空格/换行符，避免被 TUI 框架误附加斜体等全局样式。

---

## v0.1.36

### ✨ 新功能

- **Doctor 子命令** (`vibecoding doctor`)
  - 新增诊断命令，检查环境、配置、Provider、沙箱、MCP 服务器、Session、技能和上下文文件
  - 报告 OS/架构、Go 版本、Shell、Home/工作目录
  - 校验 settings、gateway 和 MCP 配置文件，带解析检查
  - 列出已配置的 Provider（API key 脱敏显示）、模型及其上下文窗口/最大 token/推理标志
  - 检查 bwrap 沙箱可用性和版本
  - 展示 MCP 服务器、Session 数量、技能目录和已发现的上下文文件
  - 未配置的 Provider（无 API key）静默跳过

### 🐛 Bug 修复

- **TUI 会话状态**
  - 修复 `/clear` 未清理 transcript 渲染状态、工具结果、助手 Markdown 缓存、活动流索引、Plan 面板和工具详情 Modal 状态的问题，与会话切换保持一致
  - 抽取共享的 transcript/input 重置辅助函数，减少不同清理路径的行为差异

- **TUI 模式切换**
  - 使用 Tab 循环切换模式时，现在会先中止正在运行的请求，与 `/mode` 行为一致，避免审批/提问响应发送到旧 agent

- **TUI Question 工具**
  - 提问工具的数字选项现在会解析为对应的选项文本，不再把原始数字发回给模型
  - 清理提问状态时也会清空当前问题元数据

- **TUI 警告与详情 Modal**
  - TUI 现在会显示 Context 压力和预算压力事件
  - Ctrl+O 在没有可展示的会话详情时会提示用户，而不是打开空 Modal

- **Context Pressure 阈值比较**
  - 修复 Context Pressure 阈值比较时单位不一致的 bug：`Percent`（0-100）与 `threshold`（0-1）直接比较，导致仅 ~0.5% 使用率就触发警告而非预期的 55%
  - 修复 `InitHermesConfig` 项目模板，显式写入 `context_pressure_threshold` 和 `budget_pressure_threshold` 默认值，避免序列化为 0 导致禁用

- **实时消息渲染**
  - 实时助手消息会将 fenced code block 按 Markdown 渲染，同时普通文本保留 plain-text wrapping 路径，避免中英文被异常拆词换行

- **模型校验与 Compaction 修复**
  - 在 `ChatParams` 中传递 `ModelID`，让 Provider 知晓当前活跃模型
  - 将模型信息透传到 compaction/summary 生成，避免静默回退到默认模型
  - 模型未找到时返回错误并列出可用模型列表，不再静默回退
  - 统一 factory、gateway 和 TUI 的模型错误提示

- **Google/OpenAI 工具结果文本提取**
  - 修复 Google 和 OpenAI Provider 在工具结果使用富 `Contents` 块而非纯 `Content` 字符串时发送空内容的问题
  - Google Provider 新增 `googleToolResultText()` 从 `Contents` 块提取文本
  - 修复 OpenAI 富工具结果分支中 `responseToolOutput()` 的使用

### 🧪 测试

- 新增 `/clear` transcript 清理、提问状态跟踪、空详情 Modal、压力警告、实时代码块渲染和普通文本换行的回归测试
- 新增 agent 级集成测试（`TestToolResultIsIncludedInNextProviderTurn`），验证含富 `Contents` 块的工具结果能正确传递到下一轮 provider 调用
- 新增 Google 和 OpenAI Provider 从 `Contents` 块提取工具结果文本的单元测试

## v0.1.35

### 🐛 Bug 修复

- **TUI 打印顺序修复**
  - 将分散的 `go program.Println(...)` 协程替换为单一 drain 协程（`printCh`），防止在 Bubble Tea 无缓冲 channel 上发生消息交错
  - `flushPendingPrints` 改用 `tea.Sequence` 替代 `tea.Batch`，保持打印顺序

- **显示宽度准确性**
  - `truncate()` 改用 `lipgloss.Width` 替代字节长度计算，CJK 字符（2 格）和 ANSI 转义序列（0 格）在 TUI 网格中正确对齐
  - 工具详情 Modal 标题分隔线使用 `lipgloss.Width` 计算正确行宽

- **工具输出改进**
  - `ls` 工具结果在折叠视图中显示 compact 摘要（去除空行），与 `bash` 输出行为一致
  - 工具结果渲染对多行摘要使用换行分隔符，不再强制压缩到单行

### 🧪 测试

- 新增 `formatters_test.go`，覆盖 ASCII、CJK 及混合内容的显示宽度截断测试

## v0.1.34

### ✨ 新功能

- **Hermes 远程 TUI 客户端**
  - 将纯文本 WebSocket 客户端替换为完整的 Bubble Tea TUI（`hermes client`）
  - 通过 Glamour 实现 Markdown 渲染与语法高亮
  - 可滚动的工具详情 Modal（Ctrl+O）、审批提示（Enter/Esc）和 question 工具支持
  - Plan 更新展示、Context 压力/预算警告和请求计时器
  - 已完成消息输出到终端原生 scrollback
  - 支持斜杠命令（`/clear`、`/mode`、`/model`、`/compact`、`/help` 等）
  - 新增 `internal/hermes/remotetui` 包：`app.go`、`render.go`、`input.go`、`remote.go`、`agent_events.go`、`approval.go`、`commands.go`、`formatters.go`、`tool_modal.go`、`events.go`

- **WebSocket 协议增强**
  - 新增 `question_request` / `question_response` 事件，支持 Plan 模式下通过 WebSocket 使用 question 工具
  - 新增 `plan_update` 事件，携带结构化 plan 步骤数据
  - 新增 `compaction_start` / `compaction_end` 事件，展示上下文压缩进度
  - `connected` 事件新增 `model` 和 `work_dir` 元数据
  - `approval_request` 事件新增 `approval_tool` 和 `approval_args`，丰富客户端展示

- **Dispatcher 重构**
  - 将 `buildAgent()` 从 `runAgent()` 中提取，改善消息平台与 WebSocket 路径间的 agent 创建复用

- **Provider 自定义 Header**
  - 新增 `providers.<name>.headers`，支持为 provider 请求携带自定义 HTTP header
  - Header 值支持与 `apiKey` 相同的 `${ENV}` 和需显式开启的 `!cmd` 解析

### 🧪 测试

- 新增 WebSocket Gateway 服务器测试，覆盖连接、认证、聊天、审批、提问和命令流程

## v0.1.33

### ✨ 新功能

- **多项目技能目录支持**
  - Skills 管理器现支持从项目根目录下的 `.skills/` 和 `skills/` 两个目录加载技能
  - 优先级顺序：`.skills/` > `skills/` > 全局技能目录
  - 新增 `NewManagerWithProjectDirs` 构造函数，接受按优先级排列的项目目录列表
  - 新增 `ProjectSkillDirs` 辅助函数，返回标准项目技能目录列表
  - 更新所有调用点：CLI、ACP、Gateway、Hermes
  - 新增多目录优先级和普通 `skills/` 目录加载的测试

## v0.1.32

### ✨ 新功能

- **工具系统完整性**
  - 补充所有已注册工具的完整文档：`jobs`、`kill`、`question`、`memory`、`cron` 及 MCP 动态工具
  - `jobs` 工具：列出并查看通过 `bash async=true` 启动的后台任务，支持清理已完成任务
  - `kill` 工具：通过 Job ID 终止正在运行的后台任务
  - `question` 工具：Plan 模式下 AI 可向用户提出多选问题以澄清需求
  - `memory` 工具（Hermes）：通过 `memory.md` 实现跨会话持久记忆，支持 read/add/update/delete 操作
  - `cron` 工具（Hermes/多 Agent）：通过子 Agent 执行定时后台任务，支持 `@daily`、`@weekly`、`@every N` 调度及单次执行
  - MCP 动态工具：来自 MCP 服务器的 tools/resources/prompts 在会话中自动发现和注册

- **Plan 模式提问工具**
  - 新增 `question` 工具，仅在 TUI + plan 模式下注册
  - AI 可向用户提出多选问题，用户选择预设选项或输入自定义答案
  - 用于在制定方案前澄清需求，形成更优质的计划
  - 通过 `QuestionHandler` 可选接口暴露（类型断言），不污染公共 `Agent` 接口

### 🐛 Bug 修复

- **Bash 工具输出安全**
  - 同步 bash 模式新增 1GB 输出限制，使用 `limitedBuffer` 防止无界 `bytes.Buffer` 导致 OOM

- **Hermes `/compact` 命令**
  - 实现 Hermes 消息模式下的 `/compact` 斜杠命令（之前是 TODO 桩）
  - 在 session 上设置 `ForceCompact` 标志，下次 agent 运行时消费以触发上下文压缩

- **Session 持久性**
  - `writeEntry` 写入后调用 `f.Sync()`，保证崩溃或断电后数据不丢失
  - 损坏的 session 行现在记录为 warning 并跳过，不再阻止 session 加载

- **Hermes 审批竞态修复**
  - `ResolveApproval` 使用 `select` 发送，避免超时与审批竞态时写入已消费的 channel

- **子代理 Panic 日志**
  - `sendParentEvent` 在 recover 前记录 panic 值，便于诊断关闭 channel 的竞态

- **原子文件写入清理**
  - `writeFileAtomic` 移除 `defer os.Remove(tmpPath)`，改为各错误路径显式清理，避免成功后尝试删除已重命名的文件

- **Agent 循环检测可配置化**
  - `MaxConsecutiveNoText`（卡住检测阈值）可通过 `AgentLoopConfig` 配置（默认 95）
  - 修复错误消息中错误地将前后警告计数器相加的问题

- **Job Manager 自动清理**
  - `AddJob` 时自动 GC 30 分钟前完成的 job（每 5 分钟检查一次）

- **Cron 调度器错误日志**
  - `checkAndRun` 现在记录 store 错误，不再静默吞掉

- **TUI Bash 输出显示**
  - 压缩 bash 工具输出摘要，去除空行，避免 TUI 折叠视图中占用过高垂直空间

- **内嵌搜索工具**
  - 当当前架构没有内嵌 `rg` / `fd` 时，退回使用系统 `grep` / `find`

### 📦 分发

- 新增 Linux LoongArch64 (`loong64`) 构建与打包目标，包括 tarball、Debian 和 npm 包元数据

### ✅ 测试

- 新增 `limitedBuffer` 截断、`JobManager` GC、`writeFileAtomic` 清理、`sendParentEvent` panic 恢复、`MaxConsecutiveNoText` 可配置性、session fsync 持久性、损坏行容忍、`QuestionTool` 元数据/模式过滤/执行/错误处理的单元测试


## v0.1.31

### 🐛 Bug 修复

- **终端输入**
  - 输入框支持 Home/End 光标移动
  - 修复在权限审批提示中按 Esc 取消后，第一次回车提交的输入被吞掉的问题
  - 输入框支持 Up/Down 历史记录导航，并可反复上下选择历史输入

- **A2A 安全与可靠性**
  - A2A 默认监听地址从 `0.0.0.0` 改为 `127.0.0.1`
  - 为 `/a2a`、REST A2A 路由和 SSE 事件添加 Bearer token 认证，同时保持 Agent Card 公开
  - 将基于时间戳的 A2A task ID 替换为抗碰撞的随机 ID
  - A2A task store 读写改为使用 task 快照，避免外部意外修改共享状态

- **路径与 Session 安全**
  - 路径包含校验改为使用路径边界，而不是字符串前缀匹配
  - 禁止 context `extraFiles` 逃逸工作目录
  - 对 Hermes session 路径组件进行安全编码，并在创建 session 时强制校验 `allowed_work_dirs`
  - 限制 session 删除只能删除配置 session 目录下的 `.db` 文件

- **认证、审批与资源限制**
  - Hermes HTTP/WebSocket token 校验改为常量时间比较
  - Hermes WebSocket 客户端改为通过 `Authorization: Bearer ...` 发送认证信息，不再放入 query string
  - ACP 权限请求超时后清理 pending 状态，并向调用方传播写入错误
  - 为 ACP、read 工具图片文件、微信响应和 cron A2A 响应增加大小限制
  - 为 cron A2A HTTP 请求增加超时

- **Memory、Context 与并发**
  - 为 memory store 操作增加锁
  - 修复 `memory.WriteAll()` 路径处理，并将 memory update/delete 限制在指定 section 内
  - Gateway 在请求级 `temperature`/`top_p` 覆盖前克隆模型配置
  - Agent callback 使用 context/message 快照，避免共享引用
  - Cron job 状态变更通过 job store 串行化

- **配置与 Gateway 加固**
  - `!command` API key 解析现在必须显式设置 `VIBECODING_ALLOW_SHELL_CONFIG=1`
  - 修复 Gateway CORS，使其只回显被允许的请求 origin
  - Gateway 在非 loopback 监听、`yolo` 模式且未开启认证时输出启动警告
  - 加固 platform home/shell fallback 行为

### 🧪 测试

- 增加 A2A 认证、task ID 唯一性、task 快照隔离和 working task message 持久化回归测试
- 增加路径逃逸、危险 session ID、memory section 操作、ACP 清理、CORS、UTF-8 截断和 shell-config opt-in 测试
- 已运行聚焦包测试，以及 A2A、agent、gateway、cron 的 race 测试

### 📝 文档

- 更新 A2A、Hermes、Gateway、配置和安全文档，说明新的认证和加固行为

## v0.1.30

### ✨ 新功能

- **Provider 级 HTTP 代理**
  - 新增 `providers.<name>.httpProxy`，支持为不同 provider 配置不同 HTTP 代理
  - 未配置 `httpProxy` 时继续保留默认环境变量代理行为

- **Google Gemini 和 Vertex 厂商适配器**
  - 新增原生 `google-gemini` 和 `google-vertex` provider，使用 Google `streamGenerateContent`
  - 支持 Gemini API 和 Vertex AI 原生 Gemini 端点的 baseUrl 自动识别
  - 新增 Gemini API key 和 Vertex bearer token 的默认 Google provider 模板
  - 更新 provider 文档与识别测试覆盖

- **Hosted Web Search 工具**
  - 为 CLI 和 ACP 运行新增 `--web-search`
  - 新增顶层 `webSearch` 配置，包含 `enabled`、`provider`、`providerType` 和 `model`
  - 仅在启用时注册 hosted `web_search`，并与本地 function tools 保持隔离
  - 新增 OpenAI Responses API 映射到 `web_search`
  - 将 Responses web search 映射改为 provider-neutral 的 `web_search`，兼容 provider 不必命名为 `openai`
  - 新增 Anthropic Messages API 映射到 `web_search_20250305`
  - 将 `webSearch.model` 保留为 provider-neutral metadata，用于后续路由和成本展示扩展

- **默认 Provider 模板**
  - 新增 OpenAI、Anthropic 和 Xiaomi MiMo 默认 provider 配置
  - 保留 DeepSeek providers，并继续使用 `deepseek-openai` 作为默认 provider/model
  - 首次生成的 `settings.json` 现在包含默认关闭的 web search 配置，以及 OpenAI/Anthropic/Xiaomi provider 模板

### 🧪 测试

- 增加 OpenAI Responses 和 Anthropic Messages hosted web search 序列化测试
- 增加 web search 配置默认值、CLI flag 解析和 hosted tool metadata 传递测试
- 增加 macOS 默认配置目录解析测试

### 🐛 Bug 修复

- **macOS 配置目录**
  - 将 macOS 默认全局配置目录与 Linux 统一为 `~/.vibecoding`

- **发布版本号**
  - npm 和发行包版本检测默认不再附加 `dirty` 后缀
  - 将 npm package metadata 规范化为 `0.1.30`

## v0.1.29

### 🐛 Bug 修复

- **NPM 包装修复**
  - 修复 `npm/bin/vibecoding` 入口脚本，确保安装包正确附带可执行包装器
  - 调整 `build-npm.sh` 和 `build-npm-packages.sh` 保证包装器一致性

## v0.1.28

### ✨ 新功能

- **Per-Model 温度/Top-P 配置**
  - 为 `ModelConfig` 和 `Model` 新增 `temperature` 和 `top_p` 字段，支持逐模型参数调优
  - 在 OpenAI 和 Anthropic 提供商中打通，使用 `omitempty` — `nil` 表示使用 API 默认值
  - 在 provider factory、agent loop、ACP 模式中打通
  - Gateway 模式支持请求级 `temperature`/`top_p` 覆盖（通过 `ChatParams`）
  - 未配置时完全省略参数（不会向 API 发送零值）

- **OpenAI Responses API 支持**
  - 新增独立的 OpenAI Responses provider 路径，通过 `api: "openai-responses"` 启用
  - 支持 Responses 流式输出、工具调用、reasoning summary 和 prompt cache 参数
  - 在 provider `responses` 配置中暴露 Responses 专用设置，默认启用 prompt cache
  - 新增模型兼容标志 `supportsPromptCacheKey` 和 `supportsReasoningSummary`

### 🧪 测试

- 提升 OpenAI Responses API 和 Anthropic 请求解析相关测试覆盖
- 将 Anthropic 测试改为内存 HTTP mock，避免依赖本地端口监听

### 📝 文档

- 更新 `AGENTS.md` 版本至 v0.1.28

## v0.1.27

### ✨ 新功能

- **Hermes 模式** (`vibecoding hermes`)
  - 新增消息平台网关模式，支持微信、飞书和 WebSocket
  - 持久化 per-user session，`/new` 时自动归档
  - 默认 `yolo` 模式，适合无人值守场景
  - 智能审批分级策略（low/medium/high 风险等级）
  - 用户白名单访问控制
  - WebSocket 流式推送：text_delta/think_delta/tool_call/tool_result/tool_diff/usage/done

- **A2A 协议** (`vibecoding a2a`)
  - 新增 Agent-to-Agent 协议服务器（JSON-RPC 2.0 over HTTP + SSE 流式）
  - 独立模式：`vibecoding a2a start`（端口 8093）
  - 集成模式：`hermes.json` 中 `a2a.enabled: true`，共享 hermes HTTP 端口
  - Agent Card：`/.well-known/agent.json`
  - Task 生命周期：submitted → working → completed/failed/canceled
  - REST 端点：`/a2a/send`、`/a2a/task`、`/a2a/task/cancel`、`/a2a/events`
  - **A2A Client**：`vibecoding a2a send <message>` 向其他 A2A Server 发送任务
  - **A2A 发现**：`vibecoding a2a discover <url>` 获取远程 Agent Card
  - **A2A 调度**：Cron 任务支持 `--a2a-target` 参数，定时向 A2A Server 发送任务

- **A2A Master 模式** (`--enable-a2a-master`)
  - 通过 `a2a-list.json` 配置多个远程 A2A Agent
  - 注册 `a2a_dispatch` tool，LLM 可自动向远程 agent 分发任务
  - 支持全局（`~/.vibecoding/a2a-list.json`）和项目级（`.vibe/a2a-list.json`）配置
  - `--init-a2a-master-config` 生成示例配置文件
  - 默认关闭，需显式启用

- **A2A 配置初始化**
  - `vibecoding a2a --init-a2a-config` 生成 `a2a.json` 配置模板
  - `vibecoding --init-gateway` 生成 `gateway.json` 配置模板（已有）
  - `vibecoding --init-a2a-master-config` 生成 `a2a-list.json` 配置模板
  - 所有 `--init-*` 支持 `--force` 覆盖已存在的文件

- **场景演示文档**
  - 新增 `docs/scenarios.md`（中英文），覆盖 9 种实际使用场景
  - 涵盖：日常编码、CI 集成、多 Agent、VS Code ACP、A2A 服务器、
    A2A Master 跨机器调度、Gateway HTTP 网关、Hermes 消息平台、组合模式

- **文档全面更新**
  - `architecture.md`：补全全部模块（a2a/acp/gateway/hermes/mcp/memory/messaging/vendored）
  - `tools.md`：新增 `a2a_dispatch` 和 `skill_ref` 工具文档
  - `cli-reference.md`：新增 `--enable-a2a-master`、`--init-a2a-master-config`、
    `--init-gateway`、`--force`、`a2a` 子命令文档
  - `README.md`：架构图补全、新增运行模式总览

- **压力系统**
  - Context Pressure：55% context 使用率时触发 `EventContextPressure`（可通过 `context_pressure_threshold` 配置）
  - Budget Pressure：剩余 20% 迭代时触发 `EventBudgetPressure`（可通过 `budget_pressure_threshold` 配置）
  - 一次性触发：每个阈值越界只触发一次，非每轮触发
  - 消息平台通过进度回调接收压力警告

- **智能审批（分级策略）**
  - low 风险：自动批准
  - medium 风险：自动批准 + 通知用户
  - high 风险（WebSocket）：发送 `approval_request`，等待用户 `approval_response`（5 分钟超时）
  - high 风险（消息平台）：自动拒绝 + 通知用户
  - 命令风险分类：基于 bash 命令模式的 low/medium/high 分级

- **Provider/Model 配置**
  - `hermes.json` 新增 `default_provider` / `default_model`（覆盖 `settings.json`）
  - `hermes start` 新增 `-p`/`--provider` 和 `-m`/`--model` CLI 标志
  - 优先级：CLI 标志 > `hermes.json` > `settings.json`

- **多 Agent 模式** (`--multi-agent`)
  - 启用子 Agent 工具（spawn/status/send/destroy）
  - 通过 `hermes.json` 的 `multi_agent` 字段或 `--multi-agent` CLI 标志配置

- **Sandbox 模式** (`--sandbox`)
  - 可选 bwrap 沙箱隔离（默认关闭）
  - 通过 `hermes.json` 的 `sandbox` 字段或 `--sandbox` CLI 标志配置

- **MCP 工具继承**
  - Hermes 自动加载全局/项目 `mcp.json` 中的 MCP 服务器
  - MCP 工具按 session 注册，session 移除时自动关闭连接

- **消息平台进度事件推送**
  - agent 执行过程中实时向微信/飞书推送工具执行进度
  - 格式：`[tool]: args ✅/❌`（工具）、`💭 ...`（思考过程）
  - agent 完成后发送完整总结

- **memory 工具**
  - `memory` 工具支持 read/add/update/delete 操作
  - section 级操作（User Profile、Working Memory、Lessons Learned）
  - 默认写入 `.vibe/memory.md`（项目目录）
  - 查找优先级：`memory.path` 配置 → `.vibe/memory.md` → `<GLOBAL_DIR>/memory.md`
  - `/api/memory` HTTP 端点（GET/PUT）用于 memory 访问

- **Hermes CLI 命令**
  - `hermes start` — 启动守护进程（支持所有 CLI 标志）
  - `hermes stop` — 通过 PID 文件 + SIGTERM 停止守护进程
  - `hermes status` — 通过 PID + HTTP health 检查守护进程状态
  - `hermes client` — WebSocket 客户端（流式输出 + 斜杠命令）
  - `hermes config init/show` — 配置管理
  - `hermes wechat login/status` — 微信 iLink 管理
  - `hermes feishu setup/status` — 飞书配置
  - `hermes webhook list` — webhook 路由查看
  - `hermes memory show/clear` — memory 管理
  - `hermes sessions list` — 活跃 session 列表（查询运行实例）
  - `hermes cron list/add/remove/enable/disable` — 定时任务管理
  - `a2a start/stop/status/card` — A2A 服务器管理

### 📝 变更

- 微信 iLink 协议实现，零外部依赖（5 个文件：types/protocol/auth/crypto/wechat）
- 飞书 Bot 使用官方 SDK + WebSocket 长连接
- Shell Hooks 支持 pre/post tool call 外部脚本（JSON stdin/stdout）
- Webhook 入站路由，支持 HMAC-SHA256 签名验证
- WebSocket 使用 `golang.org/x/net/websocket`（标准库兼容）
- 基于 PID 文件的守护进程管理（hermes stop/status）

### 🐛 问题修复

- **NPM 安装包修复**
  - 修复发布流水线，确保 `vibecoding-installer` 始终包含可执行入口 `bin/vibecoding`。
  - 新增 `scripts/npm-installer-wrapper.js` 作为统一的 wrapper 逻辑源，并被 `scripts/build-npm.sh`
    与 `scripts/build-npm-packages.sh` 复用，避免实现分叉。
  - 调整 `npm/.npmignore` 与 `npm/bin` 的处理方式，避免误打包非发布文件，并通过 `files` 字段显式声明要发布内容。

- **Hermes Webhook 投递与过滤**
  - 当 webhook 路由无法识别事件类型时，除非显式允许 `*`，否则按不匹配处理。
  - 为 webhook 路由新增 `delivery_target`，让微信/飞书投递拥有明确接收者。
  - 路由列表和配置模板会在存在投递目标时一并展示。

- **OpenAI Responses thinking 映射**
  - 将 `--thinking xhigh` 在 OpenAI Responses API 中映射为 `reasoning.effort: "high"`。

### 🧪 测试

- 将 webhook router 测试改为等待 handler 完成，去掉 `time.Sleep` 带来的竞态和不稳定。
- 增加无法推断事件类型时的 webhook 拒收测试。
- 增加 webhook delivery target 相关测试覆盖。

## v0.1.26

### ✨ 新功能

- **Gateway 模式** (`vibecoding gateway`)
  - 新增 HTTP 服务，对外暴露标准 OpenAI Chat Completions API (`/v1/chat/completions`、`/v1/models`、`/health`)
  - 任何兼容 OpenAI SDK 的客户端（Cursor、Continue、Open WebUI、Python SDK 等）可直接接入
  - 完整支持 Streaming (SSE) 和 Non-streaming 响应
  - 后端由 VibeCoding agent 循环驱动，tool 执行对调用方透明

- **多 Session 支持**
  - 内置 `SessionPool` 支持并发 session，每个 session 拥有独立的 agent、工具和消息历史
  - 通过请求体中的 `x_session_id` 关联 session，未指定时自动创建
  - 可配置空闲超时 (`session.idleTimeoutSeconds`) 和最大 session 数 (`session.maxSessions`)

- **Gateway Sub-Agent 支持**
  - 可选 `enableSubAgents` 配置，在 gateway 模式下启用多 Agent 编排
  - 复用现有 `AgentFactory` / `AgentManager` / 子Agent 工具，无需改动核心 agent 逻辑

- **Bearer Token 认证**
  - 通过 `gateway.json` 的 `auth.enabled` 和 `auth.tokens` 列表配置
  - 默认关闭；`/health` 端点始终不需认证

- **API 指令系统 (Slash Commands)**
  - `/clear`、`/mode`、`/model`、`/models`、`/sessions`、`/compact`、`/status`、`/skill`、`/skills`、`/help`
  - 当最后一条用户消息以 `/` 开头时触发，在 gateway 层直接处理，不调用 LLM
  - 响应使用标准 OpenAI 格式，附加 `x_command` 扩展字段

- **Tool 可见性配置** (`toolVisibility.mode`)
  - `"content"` (默认): streaming 时通过 `content` 字段发送 tool 状态文本
  - `"sse_event"`: 通过扩展 SSE event 发送，适合自定义客户端
  - `"none"`: 完全透明，客户端只见最终文本

- **System Prompt 处理策略** (`systemPromptMode`)
  - `"append"` (默认): 客户端 system message 追加到内置 system prompt 末尾
  - `"ignore"`: 完全忽略客户端 system message

- **安全: allowedWorkDirs 白名单**
  - 请求级 `x_working_dir` 的目录白名单，支持路径分隔符感知的前缀匹配
  - 三层安全模型: L1 认证 + L2 目录管控 + L3 沙箱 (bwrap)

- **Gateway Sandbox 支持**
  - 通过 `gateway.json` 的 `sandbox.enabled` / `sandbox.level` 或 `--sandbox` flag 配置
  - 细节配置（allowedRead、deniedPaths 等）继承 `settings.json`

- **Gateway 配置文件** (`gateway.json`)
  - 独立配置文件，位于 `~/.vibecoding/gateway.json`
  - 覆盖: 监听地址、认证、模式、沙箱、工作目录、目录白名单、session 管理、CORS、tool 可见性、system prompt 策略、请求超时、并发限制、日志
  - `vibecoding --init-gateway` 生成配置模板；`--force` 强制覆盖

- **请求超时与并发控制**
  - `requestTimeoutSeconds` (默认 1800s)；streaming 有数据流动不超时
  - `maxConcurrentRequests` (默认 0 = 不限制)

### 📝 文档

- 新增 `docs/gateway-proposal.md`，包含完整架构、API 设计、安全模型和实现计划
- 更新 `AGENTS.md` 版本标注

## v0.1.25

### ✨ 新功能

- **多 Agent 模式**
  - 在 CLI、TUI、ACP 模式中新增可选的 `--multi-agent` 支持
  - 新增 `AgentManager`、`EventRouter` 和每个 Agent 独立的 registry，隔离工具、job manager、session、messages 与 context
  - 新增 `subagent_spawn`、`subagent_status`、`subagent_send`、`subagent_destroy` 工具，用于派生后台子任务
  - 新增多 Agent system prompt 指引，并限制子 Agent 继续派生子 Agent

- **Cron 定时任务**
  - 新增 `internal/cron`，支持 cron store 持久化与调度器测试覆盖
  - 在多 Agent TUI 工作流中新增 `/cron` 命令入口

- **Provider 厂商适配层**
  - 新增 `internal/provider/vendor*.go` 厂商适配注册机制
  - 将 provider/model 创建逻辑统一到 `internal/provider/factory`
  - 新增 DeepSeek、Xiaomi、Kimi、MiniMax、Seed、Qianfan、Bailian、Gitee、OpenRouter、Together、Groq、Fireworks、OpenAI、Anthropic 等厂商识别
  - 保持现有 provider 配置格式不变，同时支持厂商默认值和通用 OpenAI/Anthropic 兼容 fallback
  - 新增模型 `compat` 处理，覆盖 thinking 格式、reasoning effort、max token 字段、自适应 Anthropic thinking，以及 DeepSeek/Xiaomi assistant `reasoning_content`

### 🐛 问题修复

- session 首次 append 时自动初始化，避免子 Agent 写入 session 前必须显式初始化
- 修复子 Agent 测试中的后台运行清理顺序，确保临时目录删除前已等待并销毁派生 Agent
- 在 provider 创建逻辑迁移到共享 factory 后，保留 ACP Anthropic cache-control 行为

### 📝 文档

- 更新 `AGENTS.md`，补充 provider factory 与 vendor adapter 工作约定
- 将多 Agent 实施 checklist 更新为已落地架构/状态说明
- 删除已过时的根目录 `todo.md`

### 🧪 测试

- 新增 provider vendor 解析、provider factory 创建、OpenAI/Anthropic compat、多 Agent manager/router/sub-agent 流程、cron 存储/调度、session 自动初始化等测试覆盖
- 已通过 `make test`（`go test -v -race ./...`）

---

## v0.1.24

### ✨ 新功能

- **API 重试与指数退避**
  - 对暂时性错误（5xx、网络故障、速率限制）在初始 HTTP 连接阶段自动重试
  - 指数退避策略：`baseDelay × 2^attempt`，上限 30 秒
  - 不会重试：用户中止（`context.Canceled`）、4xx 客户端错误、流传输中途失败
  - 通过 `retry` 配置项（`maxRetries`、`baseDelay`、`maxDelay`）灵活调整
  - Agent 将重试事件作为状态更新透出到 TUI 和 print 模式
  - ACP 模式同样接收重试配置

### 🐛 问题修复

- **Anthropic `cache_control` 改为显式启用**
  - 默认关闭 `cache_control`（此前会根据官方 API base URL 自动启用）
  - 需在 provider 配置中显式设置 `cacheControl: true` 才能启用 prompt 缓存
  - ACP provider 创建时显式为 Anthropic 启用 `cache_control`

- **Anthropic Tool Result 分组**
  - 修复连续 `toolResult` 消息未合并为单条 `user` 消息的问题
  - Anthropic API 要求前一轮 `tool_use` 对应的所有 `tool_result` 块在后续内容之前集中出现
  - 工具结果中的图片块现在会在同一消息中追加到所有结果块之后
  
- **Agent 纯工具循环告警顺序**
  - 将无文本输出的工具循环告警改为在 tool result 追加之后再注入
  - 保持 assistant -> toolResult -> warning 的消息顺序，确保 provider 与 session transcript 都合法
  - 告警消息现在也会持久化写入 session 存储

### 📝 文档

- **配置文档全面重写**
  - 补充缺失配置项：`cacheControl`、空闲压缩、完整沙箱字段（`bwrapPath`、`allowedRead`、`allowedWrite`、`deniedPaths`、`passEnv`、`tmpSize`）、`shellPath`、`shellCommandPrefix`、`sessionDir`、`skillsDir`、`theme`、`retry`
  - 记录 shell 命令格式的 `apiKey`（`!cmd`），支持密码管理器集成
  - 修正密钥解析顺序：优先使用配置中的 `apiKey`，其次使用推导的环境变量
  - 更新 macOS 配置路径文档
  - 新增顶层字段参考表及所有默认值
  - 新增各平台沙箱路径与环境变量默认值
  - 改进示例：Claude provider `cacheControl`、空闲压缩、项目级覆盖、自定义沙箱路径

### 🧪 测试

- 新增重试测试，覆盖 `IsRetryable`、`RetryDelay` 和 `FormatRetryMessage`
- 新增 Anthropic provider 测试，覆盖连续 tool result 分组
- 新增回归测试，覆盖 tool result 之后的纯工具循环告警插入位置

---

## v0.1.23

### 🛠 改进

- **DeepSeek Thinking 格式**
  - 新增 `thinkingFormat: "deepseek"`，用于 DeepSeek 推理请求
  - OpenAI 兼容请求现在会发送 `thinking: {type: "enabled"}` 和 `reasoning_effort`
  - Anthropic 兼容请求现在会发送 `thinking: {type: "enabled"}` 和 `output_config.effort`
  - 保留 `thinkingFormat: "xiaomi"` 作为旧的 thinking-only 格式

### 🧪 测试

- 新增 provider 测试，覆盖 OpenAI 与 Anthropic 兼容请求下的 `deepseek` thinking 格式

### 📝 文档

- 更新 `anthropic-api` skill 与配置文档中关于 `thinkingFormat` 选项的说明

---

## v0.1.22

### ✨ 新功能

- **CLI/TUI MCP 自动加载**
  - CLI/TUI 启动时现在会加载全局与项目 `mcp.json`，连接已配置的 MCP 服务器，并在 agent 工具列表冻结前注册 MCP 工具

### 🐛 问题修复

- **Markdown 渲染样式**
  - 将 CLI print 模式和 TUI 的 Markdown 渲染从 Glamour 自动样式检测改为固定 `dark` 样式，提升不同终端中的显示一致性

### 🧪 测试

- 新增 MCP 配置加载测试，覆盖模板占位服务器过滤

### 🛠 改进

- **共享 MCP 运行时**
  - 将 MCP 连接与工具注册从 ACP 私有实现提取为共享运行时，ACP 与普通 CLI/TUI 会话复用同一套逻辑
  - 自动启动加载时会忽略 starter 模板中的占位 MCP 服务器

---

## v0.1.21

### ✨ 新功能

- **Plan/Apply 工作流**
  - 新增内置 `plan` 工具，用结构化任务计划表达 `pending`、`running`、`done` 和 `failed` 步骤状态
  - TUI 现在会展示当前任务计划，并把计划更新记录到对话历史中
  - Print 模式和 ACP 现在也会透出计划更新，支持非交互和编辑器客户端流程

- **Apply 确认**
  - 新增 `approval.confirmBeforeWrite`，用于在 Agent 模式下要求 `write` 和 `edit` 执行前审批
  - 新生成的默认配置会启用写入/编辑确认
  - TUI 审批提示会用字节数摘要写入内容，避免直接展示完整文件内容

- **MCP 配置命令**
  - 新增 `/init_mcp`，支持创建项目/全局 `mcp.json`，并提供 `basic`/`full` 模板及 `--force` 覆盖
  - 新增 `/mcps`，用于列出全局与项目 `mcp.json` 中的 MCP 服务器
  - MCP 配置改为独立 `mcp.json`（不与 `settings.json` 混用）

### 🧪 测试

- 新增 `plan` 工具和 write/edit 审批门控测试覆盖
- 新增基于 HTTP 的 MCP 集成测试，覆盖 tool/resource/prompt 注册与回调链路
- 新增基于 SSE 的 MCP 集成测试，覆盖流通知回调与 message endpoint 请求/响应链路

### 🛠 改进

- **ACP MCP 健壮性增强**
  - 新增 `http` 和 `sse` MCP 传输支持（保留现有 `stdio`）
  - 为 MCP 初始化与工具发现增加超时控制，避免 ACP 会话长时间挂起
  - 为 `tools/list` 增加分页拉取与页数上限保护
  - 新增 MCP `resources/*` 与 `prompts/*` 发现和工具注册
  - 增加 MCP 服务器重名检测与 MCP 工具名去重注册
  - 增加 MCP 入站请求/通知处理（`ping`、progress/logging/cancel 通知）
  - 新增入站 `sampling/createMessage` 到当前 ACP provider/model 的桥接
  - 收紧关闭/错误传播行为

---

## v0.1.20

### ✨ 新功能

- **结构化文件变更报告**
  - `write` 和 `edit` 现在会在工具结果中附带结构化文件 diff 元数据
  - TUI 工具详情中展示完整 unified diff，折叠工具行保留简洁的 `+N -N` 摘要
  - Print 模式现在会为非交互运行输出清晰的文件变更摘要
  - ACP 工具更新会在 raw output 中包含 diff 元数据，方便兼容客户端使用

### 🧪 测试

- 新增 `write` 和 `edit` 结构化 diff 元数据测试覆盖

---

## v0.1.19

### ✨ 新功能

- **TUI 工具详情 Modal**
  - 将 `Ctrl+O` 切换展开替换为可滚动的全屏 modal overlay，展示所有工具调用及结果
  - 支持 PgUp/PgDn、Up/Down、Home/End 导航；Esc/Ctrl+O/q 关闭
  - 工具标题现在显示文件路径；移除了工具参数中的内容截断
  - Write 工具结果在摘要行显示 diff 信息
  - Modal 打开时屏蔽键盘输入，防止误操作

- **Write 工具 Diff 摘要**
  - `write` 工具现在在覆盖文件时基于 LCS 算法计算行级 diff
  - 在工具结果中返回结构化 diff 信息（`+N -N` 及行范围）
  - 对超大文件（>20 万行对）跳过 diff 计算，避免内存压力

### 🛠 改进

- **沙箱后端统一 Shell 参数**
  - 所有沙箱后端（`none`、`mac`、`windows`）现在统一使用 `platform.ShellArgs()` 构造 cmd.exe/PowerShell 参数
  - 修复沙箱模式下 Windows cmd.exe 和 PowerShell 命令执行问题
  - `ShellArgs` 现在在匹配前将 shell 名称转为小写

### 🧪 测试

- 新增 `TestNoneSandboxWrapCommandUsesPlatformShellArgs`，覆盖 cmd.exe 和 PowerShell 参数生成

---

## v0.1.18

### 🐛 问题修复

- **TUI Nil 指针 panic**
  - 修复 `printMessageOnce` 在 `printedMessageIdx` map 未初始化时导致的 nil 指针 panic
  - 添加 nil 检查，确保在消息打印逻辑中安全访问 map

- **工具执行前提交流**
  - 添加 `commitActiveStream()` 方法，用于在工具执行前将流式内容（thinking 和 assistant 消息）刷新到输出
  - 现在在 `EventToolCall` 和 `EventToolApprovalRequest` 处理前正确提交活跃的流
  - 确保在工具运行或请求审批时能看到 thinking 和部分 assistant 响应

### 🧪 测试

- 新增 `TestHandleAgentEventCommitsStreamBeforeApproval` 回归测试，覆盖流提交顺序

---

## v0.1.17

### 🛠 改进

- **TUI 原生滚动历史**
  - 重构 TUI 历史渲染：已完成消息会输出到终端原生 scrollback，而不是固定高度 viewport
  - 移除虚拟滚动条与鼠标捕获方案，鼠标滚轮现在使用终端自身的历史滚动行为
  - 保留实时流式内容、输入框、footer、上下文/缓存状态以及工具输出控制

- **TUI 请求计时器**
  - 响应运行期间显示本次请求耗时
  - 请求完成后在 footer 保留上一次请求耗时

- **事件循环解耦**
  - 新增共享的 agent event 消费辅助逻辑
  - 将 TUI 的 agent event bridge 从主 app 文件拆出，并让 CLI print 模式复用同一套事件消费逻辑

- **Windows 控制台兼容性**
  - 在可用时启用 Windows Virtual Terminal 控制台模式，改善 Windows 10 PowerShell 下的显示兼容性

### 🐛 问题修复

- 修复 TUI 启动时在 Bubble Tea 开始消费消息前打印初始/会话历史导致的卡死问题
- 修复 `go test -race` 发现的 agent 消息历史数据竞争
- 修复 mock provider 在 context 已取消时未稳定返回取消错误的问题

### 🧪 测试

- 全量 `make test` 已通过 race detection
- 新增 TUI 启动历史打印不阻塞的回归测试
- 增强受限环境下依赖本地 HTTP listener 或默认 home 目录会话路径的测试稳定性

---

## v0.1.16

### 🛠 改进

- **通过 ID 或路径打开会话**
  - 新增 `OpenByPathOrID` 函数，支持通过文件路径或会话 ID 打开会话
  - `OpenByID` 现在支持前缀匹配，并具备歧义检测
  - `ContinueRecent` 在创建新会话时立即初始化，确保可直接写入消息

- **会话保存错误处理**
  - `AppendMessage` 和 `AppendCompaction` 现在会向调用方返回错误
  - Agent 循环将会话保存失败作为 `EventError` 上报，不再静默丢弃

- **内嵌工具测试守卫**
  - Makefile `test` 目标现在依赖 `prepare-vendored` 和新增的 `test-vendored` 检查
  - 若当前平台缺少 `rg`/`fd` 二进制文件，测试会提前失败并给出明确提示

### 🧪 测试

- 新增 CLI flag 解析测试，覆盖 root 和 ACP 子命令
- 新增配置合并测试，覆盖项目级覆盖和环境变量
- 新增会话测试，覆盖 `OpenByPathOrID`、前缀歧义、损坏行和父链追踪

---

## v0.1.15

### 🐛 问题修复

- **内嵌搜索工具可用性**
  - 修复 `grep` 和 `find`：当内嵌的 `rg` / `fd` 尚未释放到本地时，会按需准备二进制文件，而不是直接失败
  - 为已释放的内嵌二进制补齐可执行权限，避免复用时出现 `permission denied` 错误

- **Bash 工具结果处理**
  - 修复 bash 工具返回内容，稳定输出 stdout、stderr、工作目录和退出码等结构化信息
  - 将命令非零退出保留为正常工具结果，并通过明确的 `exit_code` 字段表达，而不是混入传输级错误
  - 统一将空 stdout/stderr 渲染为 `(no output)`，便于下游稳定处理

---

## v0.1.14

### 🐛 问题修复

- **继续会话上下文注入（`-c`）**
  - 修复 TUI 状态耦合问题：继续会话时可能只显示历史记录，但后续提问未将历史真正注入模型上下文
  - 将会话历史状态拆分为“UI 展示标记”和“Agent 注入标记”，确保恢复会话后可持续携带上下文
  - 在 agent 重建场景（中止/模式切换/模型切换/技能切换/会话切换）统一重置历史注入状态
  - 补充 `EventStatus` 与 `EventMessageStart` 的 TUI 事件处理，确保状态/警告消息稳定渲染

### 🧪 测试

- 新增回归测试覆盖：
  - UI 历史已加载时的历史注入
  - 继续会话真实启动时序（`Init()` 先加载历史，再处理后续输入）

---

## v0.1.13

### 🐛 问题修复

- **流式事件与工具调用健壮性**
  - 保留 TUI 事件监听器中的 agent 事件，避免流式过程中丢失 done/error/status 处理
  - 为 Anthropic 增加 thinking signature 的流式接收与多轮回放支持，并将 SSE `error` 事件正确上报为流错误
  - 当 OpenAI 兼容 provider 在流式工具调用中省略 ID 时，自动生成回退 ID，并在 agent 循环中增加额外防御性回退

- **沙箱环境继承**
  - 修复 `none` 沙箱执行未继承父进程环境的问题，包括 `$HOME` 等环境变量
  - 明确 bubblewrap 环境变量覆盖逻辑，使实现与实际运行行为一致

### 🛠 改进

- **内嵌工具构建流程**
  - 围绕 `prepare-vendored` 统一构建与发包流程
  - 移除旧的 `vendored-tools` 发布步骤，并废弃过时的提取辅助脚本

- **文档站点布局**
  - 扩大文档首页内容区宽度，提升大屏阅读体验

- **包元数据**
  - 更新 npm 安装器相关包版本

### 📖 文档

- 更新 README 与文档首页，突出更安全的审批处理、统一缓存指标和一致的 provider 调试行为
- 精简仓库内 agent 使用说明 `AGENTS.md`

### 🧪 测试

- 为 bash 工具补充仅 stdout、仅 stderr、无输出、非零退出码等输出场景覆盖
- 为 TUI 增加状态/警告渲染与 done/error 事件透传的回归测试
- 为缺失 ID 的 OpenAI 流式工具调用增加回归测试

---

## v0.1.12

### 🐛 问题修复

- **统一缓存命中率语义**
  - 将缓存命中率计算恢复为基于完整 prompt 输入足迹（`CacheRead / TotalInputTokens()`）
  - 让 CLI print 模式的 token 显示与 TUI 的缓存感知总量保持一致
  - 更新 Anthropic 缓存测试与通用 provider usage 测试，使其与统一定义对齐

- **非交互与 YOLO 流程中的审批安全性**
  - 让 `bashBlacklist` 在审批检查中真正生效，且优先级高于 `bashWhitelist`
  - 在 `yolo` 模式下，命中黑名单的 bash 命令仍然要求审批
  - `--print` 模式遇到本应需要用户确认的命令时，改为直接报错退出，而不是自动批准

### 🛠 改进

- **调试输出一致性**
  - `--debug` 现在会同时启用 provider 级请求/响应调试输出
  - ACP 模式下也采用相同行为

- **跨平台路径处理**
  - 将 `.skills` 路径构造从字符串拼接改为 `filepath.Join(...)`

### 📖 文档

- 更新 CLI 参考文档，说明更严格的 `--print` 行为与 debug 输出行为
- 更新配置文档，说明审批优先级与 `VIBECODING_DEBUG`
- 更新根 README 与文档首页，突出更安全的审批处理、统一缓存指标和 provider 调试行为

### 🧪 测试

- 新增白名单/黑名单及 `yolo` 模式下的审批行为测试
- 新增 print 模式中需审批工具调用的回归测试
- 扩展 cache 相关 provider 测试，覆盖统一后的缓存命中率定义

---

## v0.1.11

### 🛠 改进

- **命令结构重构**
  - 将根命令创建提取为独立函数，提升可测试性
  - 新增命令初始化和配置的单元测试
  - 提高代码模块化和可维护性

### 📖 文档

- **许可证与文档更新**
  - 新增 MIT 许可证文件
  - 新增中文 README（README_zh.md），提升中文用户体验
  - 更新 npm 包版本

---

## v0.1.10

### ✨ 新功能

- **ACP 支持文档**
  - 在 README 中添加 ACP（Agent Client Protocol）支持文档
  - VibeCoding 可作为 ACP stdio Agent 运行，用于编辑器集成
  - 兼容 VS Code、Zed 和 JetBrains IDE（IntelliJ IDEA/WebStorm），通过 ACP 兼容插件接入

### 📖 文档

- 更新主 README.md 添加 ACP 支持特性
- 更新英文 README 添加功能特性部分
- 更新中文 README 添加功能特性部分

---

## v0.1.9

### 🐛 问题修复

- **TUI 延迟渲染协程安全**
  - 修复 `scheduleRender` 从后台协程直接调用 `updateViewportContent` 而未归队到 Bubble Tea UI 协程的问题
  - 新增 `renderRequestMsg` 类型和 `program.Send()` 方法，确保 UI 更新正确归队
  - 新增 `program *tea.Program` 字段和 `SetProgram()` 方法支持延迟 UI 调度

### 🛠 改进

- **TUI 中止时清空输入队列**
  - 手动中止和模式切换时清空输入队列并重置输入状态
  - 防止缓冲按键在中止后继续执行

- **助手消息槽位预留**
  - 新增 `EventTurnStart` 处理，在文本增量到达前预留显示槽位
  - 防止工具输出在流式传输过程中改变助手消息索引
  - 在 `updateViewportContent` 中增加空原始 markdown 检查

- **工具提示片段优化**
  - 为 `read`、`ls`、`grep`、`find` 工具描述添加 "(preferred for ...)" 提示
  - 调整工具注册顺序：只读工具优先注册在 write/edit/bash 之前

### 🧪 测试

- 新增 `TestHandleAgentEventReservesAssistantSlotBeforeTextDelta` 测试
- 新增 `TestAbortClearsQueuedInput` 测试

---

## v0.1.8

### 🐛 问题修复

- **缓存感知的 Token 计算修复**
  - 修复 Anthropic `TotalTokens` 计算未包含 `CacheRead` 和 `CacheWrite` 的问题
  - 为 `Usage` 结构体添加 `PromptTokens()` 和 `TotalInputTokens()` 辅助方法
  - 更新 `CacheInfo()` 使用 `TotalInputTokens()` 作为分母，确保缓存命中率准确
  - 更新 TUI 显示正确的 token 计数（包含缓存 token）

### 🧪 测试

- 添加 `PromptTokens()` 和 `TotalInputTokens()` 辅助方法的综合测试
- 更新 Anthropic provider 测试以验证 `TotalTokens`

---

## v0.1.7

### 🐛 问题修复

- **Anthropic Provider Tool Use 序列化**
  - 修复 `tool_use` 内容块在 tool 无参数时缺少 `input` 字段的问题
  - 将 `Input` 字段从 `map[string]interface{}` 改为 `*map[string]interface{}`，使 `omitempty` 仅检查指针是否为 nil，而非空 map
  - 修复使用小米 MiMo 等 Anthropic 兼容端点时的 API 错误

---

## v0.1.6

### ✨ 新功能

- **会话管理命令**
  - 新增 `/sessions` 命令，用于浏览和管理项目会话
  - 支持列出、切换、清除和删除会话
  - 显示会话详情，包括文件路径和消息数量

### 🐛 问题修复

- **沙箱初始化**
  - 修复沙箱初始化验证和 bwrap 多架构兼容性问题
  - 改进沙箱设置的错误处理

### 📖 文档

- 更新 AGENTS.md 中的当前版本信息
- 格式化 Go 代码以保持一致性

---

## v0.1.5

### ✨ 新功能

- **DeepSeek V4 默认模型**
  - 更新默认模型规格为 DeepSeek V4（Flash 和 Pro）
  - 100 万上下文窗口，最高 38.4 万最大输出 token
- **安装脚本改进**
  - 安装完成后显示配置目录路径

### 🐛 问题修复

- **Windows IME 支持**
  - 修复 Windows 终端的 IME（中日韩输入法）支持
  - 修复 Windows 上的 shell 命令解析
  - 新增配置加载诊断信息，便于排查问题
- **Musl Deb 包**
  - 修复 musl deb 包使用无效 dpkg 架构名的问题

### 🛠 改进

- **配置简化**
  - 移除 `auth.json` 支持 — 所有凭据统一使用 `settings.json`
  - 更简洁的配置路径，单一数据源

### 📖 文档

- 明确说明 OpenAI/Anthropic 兼容 API 服务也受支持
- 从文档和安装脚本中移除所有 `auth.json` 引用
- 新增 Windows `%APPDATA%` 路径的详细示例
- 清晰区分 Windows 与 Linux/macOS 的配置路径

---

## v0.1.4

### ✨ 新功能

- **Linux musl 构建支持**
  - 新增 `make build-linux-musl` 目标，静态链接 musl 二进制文件（amd64 + aarch64）
  - 通过 `dist-tarball` 和 `dist` 目标生成 musl tarball 包
  - 通过 `dist-deb` 目标生成 musl Debian 包（amd64-musl / arm64-musl）
  - npm 包：`vibecoding-installer-linux-musl-x64` 和 `vibecoding-installer-linux-musl-arm64`
  - npm 使用 `libc` 字段实现 musl/glibc 正确解析（npm >=9.4）
  - postinstall.js 自动检测 Linux 上的 musl 与 glibc

---

## v0.1.3

### ✨ 新功能

- **版本规则**
  - 新增版本号管理规则：版本号采用十进制进位（如 v0.1.9 -> v0.2.0）
  - 明确 changelog 编写规则：只在 docs/en/changelog.md 和 docs/zh/changelog.md 中编写
  - 不创建单独的 release notes 文件

---

## v0.1.2

### ✨ 新功能

- **Prompt Cache 优化**
  - 实现了基于 LLM_Agent_Cache.md 策略的提示缓存优化
  - 跨多轮对话缓存系统提示和静态上下文
  - 通过重用缓存 token 减少 API 成本

- **TUI Markdown 语法高亮**
  - TUI 中的助手消息现在支持 markdown 语法高亮
  - 代码块、标题和格式化内容有视觉区分
  - 提升 LLM 响应的可读性

### 🐛 问题修复

- **安全与正确性**
  - 解决了关键的安全、竞态条件和正确性问题
  - 修复了代码库中的高、中严重性正确性问题
  - 移除了死代码，提高了整体代码正确性

- **TUI 稳定性**
  - 修复了在不支持的 stdin 上 `clearStdin` 阻塞导致的 TUI 启动挂起
  - 修复了 ANSI 转义码在前缀检查中导致的 TUI 助手消息渲染损坏

### 🛠 改进

- **代码质量**
  - 修复了代码库中剩余的中等严重性问题
  - 更新了 npm 包版本

---

## v0.1.1

### ✨ 新功能

- **缓存命中率显示**
  - 状态栏现在显示所有轮次的累计缓存命中百分比
  - 缓存命中率 ≥ 50% 时高亮显示，便于快速识别
  - 每轮 token 使用行新增缓存读写数量显示

- **代理兼容性**
  - 支持在 `message_delta` 而非 `message_start` 中发送 usage 字段的代理
  - 支持将 usage 拆分到多个 SSE chunk 的 OpenAI 代理（每个字段取首次出现的值）
  - 修复 print 模式 token 汇总行 `$` 前缺少空格的问题

### 🛠 改进

- **代码质量**
  - 提取 `Usage.CacheInfo()` 消除 3 处重复的缓存显示逻辑
  - npm 包版本号改为 `v` 前缀格式（如 `v0.1.1`）
  - 统一所有 npm package.json 的 JSON 格式

### 🧪 测试

- 新增 37 个单元测试覆盖 `CacheInfo()`、`formatCachePercent()` 和 `renderFooter()` 缓存部分
- 新增 12 个 httptest 集成测试覆盖 Anthropic 和 OpenAI SSE 缓存 token 解析

---

## v0.1.0

### ✨ 新功能

- **小米 MiMo thinking 格式支持**
  - 新增 `thinkingFormat` 配置选项，支持小米 MiMo API 格式
  - OpenAI provider: 小米端点使用 `thinking: {type: "enabled"}` 格式
  - Anthropic provider: 小米端点省略 `budget_tokens`
  - URL 自动检测：未设置 `thinkingFormat` 时自动检测 `xiaomimimo` 端点
  - 调试日志：通过 `VIBECODING_DEBUG` 环境变量启用

### 🛠 改进

- **配置灵活性**
  - `thinkingFormat` 从配置传递到 provider，不再仅依赖 URL 检测
  - Anthropic `budget_tokens` 从必需改为可选（指针类型 + `omitempty`）

---

## v0.0.9

### ✨ 新功能

- **工具图像支持**
  - `read` 工具现在支持读取图像文件（PNG、JPEG、GIF、WebP）
  - 图像以 base64 编码数据和 MIME 类型信息返回
  - LLM 现在可以分析和理解图像内容
  - 支持格式：`.png`、`.jpg`、`.jpeg`、`.gif`、`.webp`

- **富内容工具结果**
  - 新的 `ToolResult` 结构体支持纯文本和富内容块
  - 工具现在可以在单个结果中返回文本 + 图像
  - 新增工厂函数：`NewTextToolResult()` 和 `NewImageToolResult()`

- **模型切换**
  - `/model <id>` 命令允许在交互模式下切换模型
  - `/model` 不带参数显示当前模型和可用选项
  - 切换模型时自动重置 Agent

- **增强的帮助系统**
  - `/help` 命令现在显示详细的命令说明
  - 新增键盘快捷键参考（Tab、Esc、Ctrl+O、PgUp/PgDn）

### 🛠 改进

- **上下文 Token 估算**
  - 修复了同时存在 `Content` 和 `Contents` 时的重复计算问题
  - 图像 token 估算为每张图约 1200 token

- **提供商消息转换**
  - OpenAI：工具结果中的图像作为补充用户消息发送
  - Anthropic：图像作为单独的用户消息与 tool_result 一起发送

### 🧪 测试

- 新增 `TestReadToolImage` 测试用例验证图像读取功能
- 所有工具测试已更新为新的 `ToolResult` 返回类型

---

## v0.0.8

### ✨ 新功能

- **NPM 多架构分包优化**
  - 将 npm 包从单包全平台（~60MB）拆分为 6 个平台独立包（每个 ~10MB）
  - 用户安装时只下载当前平台的二进制文件，体积减少 83%
  - 利用 npm `optionalDependencies` + `os`/`cpu` 字段自动匹配平台
  - 主包 `vibecoding-installer` 仅 ~2KB，通过 `postinstall` 链接正确的平台包

### 🛠 改进

- **构建系统**
  - 新增 `scripts/build-npm-packages.sh` 生成平台独立 npm 包
  - 新增 `make npm-packages`、`make npm-pack`、`make npm-publish-all` 目标
  - `sync-npm-version.sh` 同步更新所有平台包版本

---

## v0.0.7

### ✨ 新功能

- **跨平台沙箱支持**
  - 沙箱现在除 Linux 外还支持 macOS 和 Windows
  - macOS 使用 `sandbox-exec` 进行进程隔离
  - Windows 使用受限进程创建，禁止网络访问
  - 自动选择平台特定的沙箱实现

- **仓库重命名**
  - 模块路径更名为 `github.com/startvibecoding/vibecoding`
  - 所有导入、文档和脚本已同步更新

### 🛠 改进

- **平台特定进程处理**
  - 将 `SysProcAttr` 配置提取到构建标签文件（`bash_unix.go`、`bash_windows.go`）
  - 后台子进程清理现在在所有平台上正常工作
  - `Setpgid` 仅在 Unix 系统上设置；Windows 使用 `CREATE_NEW_PROCESS_GROUP`

### 📖 文档

- 更新所有 GitHub URL 至新仓库地址
- 新增 v0.0.6 和 v0.0.7 发布说明

---

## v0.0.6

### 🛠 改进

- **Bash 工具可靠性**
  - 修复后台子进程挂起问题
  - 添加 `WaitDelay` 防止 shell 无限等待后台子进程
  - 正确处理 `exec.ErrWaitDelay` 错误

- **NPM 安装**
  - 新增 npm 包，支持通过 `npm install -g vibecoding-installer` 安装
  - `postinstall` 时自动下载二进制文件

### 📖 文档

- 新增 npm 安装说明
- 移除 docs 根目录下冗余的 markdown 文件
- 新增 v0.0.5 更新日志

---

## v0.0.5

### ✨ 新功能

- **非 root 安装**
  - `install.sh` 现在支持无需 root 或 sudo 权限安装
  - 自动检测可写安装目录：优先使用 `/usr/local/bin`，若不可写则回退到 `~/.vibecoding/bin`
  - 移除所有 `sudo` 调用 — 用户级安装不再需要提升权限

- **自动 PATH 配置**
  - 自动检测用户 shell（bash、zsh、fish）并在相应配置文件中配置 PATH
  - 支持 `.bashrc`、`.bash_profile`、`.zshrc`、`.zshenv`、`config.fish` 和 `.profile`
  - 若 PATH 条目已存在则跳过配置（避免重复）
  - Fish shell 使用 `set -gx PATH` 语法；bash/zsh 使用 `export PATH=...`

### 🛠 改进

- **环境变量**
  - `INSTALL_DIR` — 覆盖安装目录（不变）
  - `AUTO_SETUP_PATH=0` — 禁用自动 PATH 配置
  - 更好的权限问题错误提示

- **安装体验**
  - 开始时显示安装目录和 PATH 自动配置状态
  - 更清晰的彩色状态消息输出

### 📖 文档

- 新增 v0.0.5 发布说明

---

## v0.0.4

### ✨ 新功能

- **Agent 模式审批机制**
  - Agent 模式下执行 bash 命令需要用户审批
  - 支持 `bashWhitelist` 配置，白名单中的命令自动批准
  - 支持 `bashBlacklist` 配置，黑名单中的命令始终需要审批
  - TUI 中显示审批提示，用户输入 `y`/`yes` 或 `n`/`no` 响应
  - 审批请求支持 `abort` 取消

- **模式权限矩阵**
  - Plan 模式: 只读工具 (read, grep, find, ls)
  - Agent 模式: 读写自动执行，bash 需审批
  - YOLO 模式: 所有工具自动执行
  - 更新系统提示词，明确每个模式的权限

### 🛠 改进

- **默认审批白名单**
  - 默认白名单: `go`, `make`, `git`, `npm`, `yarn`, `node`, `python`, `pip`
  - 可在 `settings.json` 中自定义

- **模式切换反馈**
  - 切换模式时显示详细权限说明
  - `/mode` 命令显示当前模式的完整权限列表

### 📖 文档

- 新增审批配置章节
- 更新安全文档，说明审批机制
- 新增 v0.0.4 发布说明

---

## v0.0.3

### ✨ 新功能

- **会话历史加载**
  - 继续或打开会话时显示会话信息（文件路径和消息数量）
  - 在 TUI 中加载并显示历史会话消息
  - 将历史消息加载到 Agent 上下文中以保持连续性
  - 中止时重置 Agent 以确保下次请求状态干净

### 🛠 改进

- **构建与分发系统**
  - 重构 Makefile，按平台划分构建和分发目标
  - 新增 `dist-linux`、`dist-darwin`、`dist-windows` 目标
  - 新增 `build-zip.sh` 用于 Windows zip 打包
  - 新增 `checksums` 目标用于发布校验
  - 更新 `build-deb.sh` 和 `build-tarball.sh` 支持全平台

### 📖 文档

- 文档网站右上角新增 GitHub 仓库跳转按钮
- 新增 v0.0.2 更新日志

---

## v0.0.2

### ✨ 新功能

- **一键安装脚本**
  - `install.sh` 适用于 Linux/macOS，自动从 GitHub Releases 下载
  - `install.ps1` 适用于 Windows PowerShell，支持通过 `VIBECODING_INSTALL_DIR` 自定义安装目录
  - 两个脚本均可自动检测平台/架构、校验完整性并配置 PATH

- **文档站重新设计**
  - 采用 Google Material Design 风格重新设计
  - 默认语言改为英文
  - 新增 Hash 路由，方便文档分享（如 `#/en/README`、`#/zh/configuration`）
  - 头部和 README 新增 Logo

- **品牌素材**
  - 新增 `docs/assets/icon.svg`（512×512）用于打包
  - 新增 `docs/assets/logo.svg`（128×128）用于 README 和小尺寸显示
  - 简洁专业的石板色调设计

- **构建系统**
  - 新增 `make build-windows` 目标（amd64 + arm64）
  - 新增 `make build-linux` 和 `make build-darwin` 目标
  - 更新 `make build-all` 使用平台专用目标

- **文档**
  - 新增 `docs/en/skills.md` 技能系统文档
  - 更新 README 和快速入门中的安装说明

### 🐛 问题修复

- 将素材移至 `docs/assets/` 以支持 GitHub Pages 部署

---

**完整变更日志**: https://gitee.com/startvibecoding/vibecoding/compare/v0.1.26...v0.1.27
