# 多模态图片处理优化方案

> 状态: Phase 1A/1B 已落地；Phase 2/3 部分落地；剩余 tile/多图预算、动态约束发现
> 日期: 2026-07-04
> 目标: 优化图片输入的可靠性、成本、上下文估算和用户体验

## 0. 实现进度快照

更新时间：2026-07-04。

- 已完成：新增统一图片处理基础、`read` 图片自动预处理、WebP decode/inspect 依赖、`ImageContent` 元数据、通用尺寸优先 token 估算、provider/model family policy hint、输出体积硬约束、相关测试。
- 已完成的 family 映射：OpenAI、Anthropic、Anthropic-on-Bedrock、Gemini、Mistral/Pixtral/Devstral、Doubao Seed、Qwen、Kimi、MiniMax、GLM、Grok/xAI、Llama Vision、Gemma Vision、MiMo、Amazon Nova、DeepSeek-on-gateway。
- 已完成的估算改进：agent compaction 路径通过 `ResolveTokenEstimator(settings, model)` 使用模型感知图片估算；Claude/Bedrock Claude 和 Qwen-like family 使用 28px patch，Gemini 使用 384/768 tile，OpenAI/Grok 使用 low/high tile 近似；同一消息多图会累加估算；旧 `EstimateTokens()` 仍保持通用估算。
- 已完成的 provider 参数透传：官方 OpenAI (`api.openai.com`) 和 xAI (`api.x.ai`) 图片请求会从内部 `Detail` 映射到 `low/auto/high`；官方域名按 URL hostname 精确判断；Google Gemini/Vertex 会从内部 `Detail` 映射到全局 `generationConfig.mediaResolution`；OpenAI-compatible 聚合层默认不发送额外 `detail` 字段。
- 已完成的截图接入：browser `screenshot` 直接返回图片时默认用 `detail` 策略进入统一 `imageproc` 流程，并携带发送/原始尺寸元数据；指定 `outputPath` 时仍保存原始截图文件。
- 已完成的裁剪能力：`read` 支持 `crop` 源图像素矩形，`ImageContent` 保留 crop 元数据，输出描述显示裁剪区域。
- 已完成的坐标辅助：`provider.ImageContent` 和 public `agent.ImageContent` 提供发送图坐标、归一化坐标到原图坐标的映射 helper；`read` guideline 明确提示局部图片问题优先使用 `crop`。
- 已验证：`go test ./...` 通过。
- 未完成：tile、多图预算、供应商模型目录动态约束发现；Gemini per-part `resolution` 等实验能力暂未接入。

## 1. 背景

MothX 已支持通过 `read` 工具读取图片文件，并把图片作为 rich content block 传给支持视觉输入的模型。浏览器截图工具也可以直接返回图片内容。此前图片处理主要是“原样透传”，本轮实现已把主要入口升级为可控预处理。

主流多模态模型通常不会直接按原始图片字节理解图片，而是先把图片转换为视觉 token、patch 或 tile。图片过大时，供应商侧往往会进行缩放、切块或拒绝请求。工程侧如果完全依赖供应商隐式处理，会带来请求体过大、延迟抖动、成本不可控、上下文估算不准等问题。

本方案目标是先建立统一、可观测、低风险的图片预处理和估算基础，再逐步支持更高级的局部裁剪、tile、多图预算和 provider-specific 策略。

## 2. 当前实现分析

### 2.1 主要链路

当前图片主要来自三个入口：

1. `read` 工具读取本地图片。
2. 浏览器截图工具直接返回截图。
3. TUI `/paste-image` 把剪贴板图片保存成本地文件路径，再由用户或 agent 后续读取。

当前核心文件：

- `internal/imageproc/imageproc.go` - 统一图片 inspect、crop、resize、transcode 和安全限制。
- `internal/imageproc/policy.go` - provider/model family policy 推断和图片策略。
- `internal/tools/read.go` - 图片文件识别、`imageMode`、`maxLongEdge`、`crop` 参数和 rich image result。
- `internal/tools/tool.go` - `NewImageToolResultWithContent()` 构造 text + image rich content。
- `internal/provider/types.go` - `ImageContent` 包含 base64、MIME、尺寸、字节数、detail、scale、crop 元数据。
- `internal/provider/openai/responses.go` - OpenAI Responses 图片转换和官方 OpenAI/xAI `detail` 透传。
- `internal/provider/openai/provider.go` - OpenAI Chat Completions 图片转换和官方 OpenAI/xAI `detail` 透传。
- `internal/provider/anthropic/provider.go` - Anthropic 图片转换。
- `internal/provider/google/provider.go` - Gemini 图片转换。
- `internal/context/tokenizer.go` - 通用和模型感知图片 token 估算。
- `internal/browser/browser.go` - 浏览器截图统一图片处理。
- `internal/tui/clipboard_image.go` - `/paste-image`。

### 2.2 当前行为

`read` 工具的图片处理流程：

```
1. 根据扩展名判断 jpg/jpeg/png/gif/webp。
2. 根据当前 agent provider/model hint 选择 image policy。
3. 检查文件大小和像素数。
4. inspect 原始尺寸和格式。
5. 可选按源图像素 `crop`。
6. 按 `imageMode` / `maxLongEdge` resize。
7. 必要时转码到 PNG/JPEG 并控制输出体积；非 `raw` 输出超过 `MaxOutputBytes` 时继续缩小重编码，仍无法满足则报错。
8. base64 编码处理后的图片。
9. 返回 text block + image block，包含原图/发送图尺寸、字节数、detail、scale、crop 元数据。
```

provider 适配层当前行为：

- OpenAI Responses: 转成 data URL，放入 `input_image`；官方 OpenAI/xAI baseURL 会发送 `detail`。
- OpenAI Chat Completions: 转成 `image_url`；官方 OpenAI/xAI baseURL 会发送 `detail`。
- Anthropic: 转成 base64 source。
- Gemini: 转成 inline data。
- 其他 OpenAI-compatible 聚合层：默认只发送标准图片字段，不透传非标准 `detail`。

上下文估算当前行为：

- `EstimateTokens()` 仍保持通用兼容逻辑：有宽高时使用 generic 512 tile 估算；无宽高时保留旧 fallback。
- agent compaction 路径通过 `ResolveTokenEstimator(settings, model)` 使用模型感知估算：Claude/Bedrock Claude 和 Qwen-like family 28px patch，Gemini 384/768 tile，OpenAI/Grok low/high tile 近似。

browser 截图当前行为：

- 未指定 `outputPath` 时，截图直接进入 `imageproc`，默认 `detail`，返回 rich image block 和尺寸/字节元数据。
- 指定 `outputPath` 时，保存浏览器返回的原始截图文件，不做预处理。

### 2.3 当前优点

- 实现简单，供应商兼容面较广。
- 不改变原图，避免预处理损失细节。
- `read` 作为显式工具调用，行为可审计。
- 非视觉模型会剥离 image content，避免直接发送给不支持图片的模型。

### 2.4 当前剩余问题

#### 2.4.1 多图请求体和延迟仍缺少总预算

单图 `read` 和 browser screenshot 已按 provider/model policy 做文件大小、像素数、长边和输出体积控制。剩余风险主要在多图场景、`raw` 模式和严格网关的总请求体限制上：多张处理后图片叠加仍可能产生较大的 base64 payload。

影响：

- 请求体膨胀。
- 网络和 provider 解析延迟升高。
- 部分供应商可能拒绝请求或隐式缩放。
- 多图场景容易超出请求限制。

#### 2.4.2 部分模型族成本和上下文估算仍不够精确

图片成本通常由视觉 token、patch 或 tile 决定，而不是 base64 字符数。当前已对 Claude/Bedrock Claude、Qwen-like、Gemini、OpenAI/Grok 接入模型感知估算，但 Doubao/Kimi/MiniMax/GLM/MiMo/Nova/Llama/Gemma 等模型族仍缺少稳定公开公式或模型目录约束。

剩余风险：

- 精确公式缺失的模型族仍使用 generic 估算。
- 多图视觉 token 累加还没有独立预算层。
- provider 返回的真实 usage 和本地估算仍可能有偏差。

影响：

- compaction 触发时机不准确。
- 上下文预算和真实 provider 消耗脱节。
- 用户难以理解一次识图为什么变慢或变贵。

#### 2.4.3 任务类型区分仍依赖 agent 主动选择

普通图片描述、UI 截图、OCR、表格、代码截图、坐标定位对分辨率需求不同。当前已提供 `imageMode=fast|auto|detail|raw` 和 `crop`，browser screenshot 默认走 `detail`，但仍依赖 agent 根据任务主动选择合适参数。

- 普通识图仍可能使用过高 detail。
- OCR/小字识别仍可能需要二次 crop/detail。
- 坐标任务已有缩放和 crop 元数据及映射 helper，但工具输出还没有展示常用映射示例。

#### 2.4.4 TUI 粘贴图片体验偏间接

`/paste-image` 只插入本地路径，依赖模型后续决定调用 `read`。这符合 terminal-first 的工具链风格，但对“我刚粘了一张图，马上问它”的场景不够直接。

#### 2.4.5 可观测元数据已落地，但消费路径还不完整

`ImageContent` 已记录发送图/原图尺寸、字节数、scale、detail 和 crop 元数据。剩余工作是让更多消费路径使用这些元数据：

- 坐标映射 helper 的更多消费入口。
- 更清晰的日志/调试输出。
- provider-specific 限制和动态模型目录发现。
- 多图预算统计。

### 2.5 默认供应商多模态图片要求调研

调研时间：2026-07-03。范围以 `internal/config/settings.go` 中默认配置里 `Input` 包含 `image` 的供应商和模型族为准。默认配置里存在大量聚合供应商或 OpenAI-compatible 代理，它们通常转发到底层模型，因此策略需要区分“底层模型族规则”和“网关/聚合层传输限制”。

#### 2.5.1 默认配置审计结果

审计方法：通过 `config.DefaultProviderConfigs()` 读取内置默认配置，筛选 `ModelConfig.Input` 中包含 `image` 的模型。审计结果显示，当前默认配置中共有 **38 个 provider preset**、**231 条视觉模型声明**。这说明图片策略不能只覆盖 OpenAI、Claude、Gemini、Doubao、Qwen 这几类显眼模型，必须按完整默认配置做 family/policy 映射。

按 provider 汇总如下：

| Provider | 数量 | 当前声明支持 image 的模型 |
|----------|------|---------------------------|
| `alibaba-coding-plan` | 4 | `qwen3.5-plus`, `qwen3.6-plus`, `qwen3.7-plus`, `kimi-k2.5` |
| `alibaba-standard` | 3 | `qwen3.6-plus`, `qwen3.7-plus`, `deepseek-v4-pro` |
| `alibaba-token-plan` | 4 | `qwen3.6-plus`, `qwen3.6-flash`, `kimi-k2.6`, `kimi-k2.5` |
| `amazon-bedrock` | 7 | Claude on Bedrock, `amazon.nova-pro-v1:0`, `amazon.nova-lite-v1:0` |
| `anthropic` | 25 | Claude 3/3.5/3.7, Claude 4.x, Claude Fable/Haiku/Opus/Sonnet variants |
| `cloudflare-ai-gateway` | 7 | Claude, OpenAI GPT, Gemini, Llama 4 Scout via AI Gateway |
| `cloudflare-workers-ai` | 4 | Llama 4 Scout, Gemma 4, Kimi K2.7 Code, GLM 5.2 |
| `fireworks` | 2 | Kimi K2.7 Code and fast router |
| `gitee` | 6 | Qwen3.6/3.7, GLM 5.2, Kimi K2.7 Code, MiniMax M3 |
| `github-copilot` | 10 | Claude, GPT, Gemini families |
| `google-gemini` | 16 | Gemini 2.x/3.x, Gemma 4 |
| `google-vertex` | 10 | Gemini 2.5/3.x via Vertex |
| `groq` | 1 | Llama 4 Scout |
| `kimi-coding` | 2 | `k2p7`, `kimi-for-coding` |
| `minimax` | 1 | MiniMax M3 |
| `minimax-anthropic` | 1 | MiniMax M3 via Anthropic-compatible API |
| `minimax-cn-anthropic` | 1 | MiniMax M3 via Anthropic-compatible API |
| `mistral` | 13 | Devstral Small 2, Mistral Large/Medium/Small vision, Pixtral |
| `moark` | 6 | Qwen3.6/3.7, GLM 5.2, Kimi K2.7 Code, MiniMax M3 |
| `moonshotai` | 2 | Kimi K2.7 Code variants |
| `moonshotai-cn` | 2 | Kimi K2.7 Code variants |
| `nvidia` | 2 | Llama 3.2 11B/90B Vision |
| `openai` | 40 | GPT-4 Turbo/4.1/4o, GPT-5.x, o-series vision-capable models |
| `opencode` | 4 | Claude variants |
| `opencode-go` | 3 | GLM 5.2, Kimi K2.6, Kimi K2.7 Code |
| `openrouter` | 14 | Claude, OpenAI GPT, Gemini, Qwen3.7 Plus, Kimi, MiniMax, Llama, Grok |
| `together` | 2 | MiniMax M3, Qwen3.5 397B |
| `vercel-ai-gateway` | 12 | Claude, OpenAI GPT, Gemini, Qwen3.6 Plus, MiniMax, Kimi, Grok, GLM |
| `volcengine` | 2 | Doubao Seed Evolving, Doubao Seed 2.1 Pro |
| `volcengine-agentplan` | 7 | Doubao Seed 2.0 Code/Pro, GLM, Kimi, DeepSeek, MiniMax |
| `volcengine-codingplan` | 6 | Doubao Seed 2.0 Code/Pro, GLM, Kimi, DeepSeek, MiniMax |
| `xai` | 4 | Grok 4.x / Grok Build vision-capable models |
| `xiaomi` | 1 | MiMo V2.5 |
| `xiaomi-token-plan-ams` | 1 | MiMo V2.5 |
| `xiaomi-token-plan-cn` | 1 | MiMo V2.5 |
| `xiaomi-token-plan-sgp` | 1 | MiMo V2.5 |
| `zai` | 2 | GLM 5.2, GLM-5V-Turbo |
| `zai-coding-cn` | 2 | GLM 5.2, GLM-5V-Turbo |

核对结论：

1. **必须显式覆盖的 family**：OpenAI, Anthropic Claude, Gemini/Gemma, Mistral/Pixtral/Devstral, Doubao Seed, Qwen Plus/Qwen VL, Kimi/Kimi Coding, MiniMax, GLM/Z.AI, Grok/xAI, Llama Vision, Amazon Nova, Xiaomi MiMo, DeepSeek-on-gateway。
2. **不能只按 provider API 判断**：`openai-chat` 下面混有 OpenAI、Mistral、Qwen、Doubao、Kimi、MiniMax、GLM、Grok、Llama、MiMo 等；`anthropic-messages` 下面也有 Kimi/MiniMax/Fireworks/Vercel 等非 Claude 模型。
3. **需要标记为待供应商确认的声明**：`deepseek-v4-pro` / `deepseek-v4-flash` 在部分套餐里声明 image，但 DeepSeek 官方 provider 当前默认配置是 text-only；这类应作为 gateway-specific capability，不能套用 DeepSeek 官方 API 假设。
4. **文档必须保留完整审计入口**：后续新增默认模型时，应重新跑同类审计，更新 family/policy 映射，而不是靠人工记忆补表。

#### 2.5.2 默认配置中的视觉模型族

按默认 provider 分组，当前视觉模型主要覆盖：

| 默认 provider | 典型视觉模型族 | 说明 |
|---------------|----------------|------|
| `openai` | GPT-4o、GPT-4.1、GPT-5.x、o-series | 走 OpenAI Responses API，支持 `detail`。 |
| `anthropic` | Claude 3/3.5/3.7、Claude 4.x、Claude 5/Fable 等 | 走 Anthropic Messages API，图片按 28x28 patch 计视觉 token。 |
| `google-gemini` / `google-vertex` | Gemini 2.x/3.x、Gemma 4 | Gemini 图片按 384/768 tile 规则计 token，Gemini 3 有 `media_resolution`。 |
| `mistral` | Mistral Large/Medium/Small vision、Pixtral、Ministral vision | OpenAI-compatible chat completions，官方文档确认 URL/base64 图片输入。 |
| `volcengine*` | Doubao Seed 2.0 Code/Pro、Doubao Seed 2.1 Pro、Doubao Seed Evolving、GLM、Kimi、MiniMax、DeepSeek 等 | Ark/套餐聚合层，多个底层模型族混合，按兼容接口发送；Doubao Seed 是默认配置里的重点视觉模型族，需要显式 policy。 |
| `alibaba-*` / `gitee` / `moark` | Qwen3.6 Plus、Qwen3.7 Plus、Qwen3.6 Max、Qwen3.6 Flash、Kimi、GLM、MiniMax 等 | 多为 OpenAI-compatible 聚合层，底层模型族差异较大；Qwen3.6/3.7 Plus 是默认配置里的重点视觉模型族，需要显式 policy。 |
| `minimax*` | MiniMax-M3 | 原生多模态，公开模型卡强调 text/image/video 混合训练。 |
| `zai*` | GLM-5.2、GLM-5V、GLM-V 系列 | GLM-V 模型强调 GUI、文档、grounding 和高分辨率。 |
| `xai` / 聚合中的 `x-ai/*` | Grok 4.x | xAI Responses 风格支持 `input_image` 和 `detail`。 |
| `groq` | Llama 4 Scout、Qwen 视觉模型 | Groq 对 base64 request size、URL image size、像素数和图片数有明确限制。 |
| `cloudflare-*` | Llama 4 Scout、Gemma 4、Kimi、GLM、Claude/Gemini/OpenAI 代理 | Workers AI 和 AI Gateway 混合，底层规则与 Cloudflare 网关能力共同决定。 |
| `amazon-bedrock` | Claude、Amazon Nova Pro/Lite | Bedrock 对 Claude 图片大小更严，Nova 有自己的 25MB payload 和 8000x8000 限制。 |
| `openrouter` / `vercel-ai-gateway` / `github-copilot` / `opencode*` | Claude/OpenAI/Gemini/Qwen/Kimi/MiniMax/GLM 等 | 聚合层。应按模型 ID 前缀推断底层规则，不能假设统一支持 `detail`。 |

#### 2.5.3 官方要求摘要

| 模型族 / 供应商 | 默认配置涉及 provider | 官方要求和处理规则 | 对本方案的影响 |
|-----------------|----------------------|--------------------|----------------|
| OpenAI GPT-4o/4.1/5/o-series | `openai`，以及 OpenRouter/Vercel/Cloudflare/GitHub Copilot 等聚合层 | 支持 PNG/JPEG/WebP/非动画 GIF；支持 URL、base64 data URL、File API；请求总 payload 可到 512MB，最多 1500 张图。`detail` 支持 `low/high/original/auto`。`low` 使用 512x512；GPT-5.4/5.5 支持 `original`，高保真/原始模式有 patch budget 和最大边限制；旧 4o/4.1/o 系列 high 使用 2048 方框、短边 768、512 tile 计费。来源：[OpenAI Images and vision](https://developers.openai.com/api/docs/guides/images-vision)。 | 内部 `Detail` 要能表达 `low/high/original/auto`。只有官方 OpenAI 或明确兼容的 vendor 才发送 `detail`，聚合层默认保守。OpenAI 的强 payload 上限不应决定全局默认，因为其他供应商更严格。 |
| Anthropic Claude | `anthropic`，Bedrock/Vercel/OpenRouter/GitHub Copilot/opencode 等 Claude 代理 | 支持 JPEG/PNG/GIF/WebP，动画 GIF 仅第一帧。Claude API 单图 10MB base64，Bedrock/GCP 单图 5MB base64；标准端点请求体 32MB。超过 20 张图时建议每边不超过 2000px。视觉 token 为 `ceil(width/28) * ceil(height/28)`；标准档长边 1568px/1568 visual tokens，高分辨率档长边 2576px/4784 visual tokens。官方建议不需要截图/密集文档高保真时先 downsample。来源：[Claude vision](https://platform.claude.com/docs/en/build-with-claude/vision)。 | 全局默认长边 1568px 与 Claude 标准档对齐。面向 Bedrock/Claude 代理时，base64 输出体积应更保守，单图目标最好低于约 3.5MB raw bytes。 |
| Google Gemini / Vertex Gemini | `google-gemini`、`google-vertex`，以及聚合层里的 Google 模型 | 支持 PNG/JPEG/WebP/HEIC/HEIF。每请求最多 3600 张图。双边都 <=384px 时 258 tokens；更大图按 768x768 tile，每 tile 258 tokens。Gemini 3 引入 `media_resolution`，可控制每张图片/每帧 token 上限；高分辨率提升小字和细节但增加 token 与延迟。来源：[Gemini image understanding](https://ai.google.dev/gemini-api/docs/image-understanding)。 | 已支持 Gemini tile 估算；Google provider 会把内部 `Detail` 映射到全局 `mediaResolution`。聚合层仍不假设透传 Google 专有参数。 |
| Mistral / Pixtral / Ministral vision | `mistral`，以及 OpenRouter/Vercel 等聚合层 | 官方 Vision 文档列出 Mistral Large 3、Medium 3.1、Small 3.2、Ministral 3 等推荐视觉模型；支持 Chat Completions API，通过 URL 或 base64 发送图片。FAQ 列出价格、token、格式、大小、图片数问题，但当前公开页面未展开具体数值。来源：[Mistral Vision](https://docs.mistral.ai/studio-api/conversations/vision)。Pixtral 论文说明 Pixtral 可按自然分辨率和宽高比处理图片，并在 128K context 中处理任意数量图片。来源：[Pixtral 12B](https://arxiv.org/abs/2410.07073)。 | 对 Mistral 使用通用 OpenAI-compatible 图片格式。没有官方稳定 token 公式时，用 generic tile/patch 估算并保留 provider override 扩展点。 |
| Doubao Seed vision | `volcengine-agentplan`、`volcengine-codingplan`、`volcengine` | 默认配置中显式支持图片的 Doubao 模型包括 `doubao-seed-2-0-code`、`doubao-seed-2-0-pro`、`doubao-seed-evolving`、`doubao-seed-2-1-pro-260628`。这些模型通过 Volcengine Ark OpenAI-compatible 入口接入。本轮公开资料检索未找到稳定、可直接引用的 per-model 图片尺寸/token 公式，因此不能把 OpenAI/Claude/Gemini 的细节规则直接套用到 Seed。 | 单独设为 `doubao-seed` family：发送格式按 OpenAI-compatible，第一阶段使用保守 `auto` 预处理；token 估算先走 generic tile/patch，后续如果 Ark 模型目录能返回图片约束，再补 vendor override。Seed Code/Pro 常用于编码和 GUI/截图类任务，`detail` 模式需要保留到 2048/2560 长边。 |
| Qwen/Qwen-VL 系列 | `alibaba-*`、`gitee`、`moark`、OpenRouter/Vercel、部分 plan provider | Qwen2-VL/Qwen2.5-VL 强调动态分辨率；Transformers 文档建议通过 `min_pixels`/`max_pixels` 控制质量和计算量，可把每张图编码到 256-1024 个视觉 token，patch 相关步长为 28。来源：[Qwen2-VL Transformers docs](https://huggingface.co/docs/transformers/en/model_doc/qwen2_vl) 和 [Qwen2.5-VL technical report](https://arxiv.org/abs/2502.13923)。 | 对 Qwen-like 模型应保留 `maxPixels`/视觉 token budget 的策略接口。OpenAI-compatible API 未必暴露这些参数，因此客户端预缩放仍有价值。 |
| Qwen3.6/3.7 Plus vision | `alibaba-coding-plan`、`alibaba-token-plan`、`alibaba-standard`、`gitee`、`moark`、OpenRouter/Vercel 等 | 默认配置里 `qwen3.6-plus`、`qwen3.7-plus` 多处声明支持 `text,image`，部分套餐还声明 `video`。它们经 Bailian/DashScope compatible-mode 或聚合网关接入。本轮公开资料检索未找到稳定、可直接引用的 Qwen3.6/3.7 Plus per-model 图片尺寸/token 公式；可参考 Qwen-VL 动态分辨率思想，但不能等同于具体 API 限制。 | 单独设为 `qwen-plus` family：发送格式按 OpenAI-compatible；预处理优先使用 `maxPixels`/`maxLongEdge` 而不是 provider-only 参数；token 估算可以先用 Qwen-like 28px patch 近似或 generic，后续补 Bailian/模型目录 override。由于这些模型常用于大上下文和文档/截图，`detail` 应可保留 2048/2560 长边。 |
| Kimi / Kimi Coding / Kimi-VL family | `moonshotai`、`moonshotai-cn`、`kimi-coding`、`fireworks`、`opencode-go`、`volcengine*`、`cloudflare-workers-ai`、OpenRouter/Vercel 等 | 默认配置中 Kimi K2.5/K2.6/K2.7 Code 多处声明 image。公开 Kimi-VL 技术报告说明 Kimi-VL 使用 native-resolution vision encoder，覆盖 OCR、文档、GUI、长视频/多图等视觉任务。来源：[Kimi-VL Technical Report](https://arxiv.org/abs/2504.07491)。但 Kimi K2.7 Code API 的逐模型图片尺寸/token 公式本轮未找到稳定公开文档。 | 单独设为 `kimi` family：发送格式按接入 API 决定，token 估算先 generic；对 coding/GUI 截图保留 `detail`。`anthropic-messages` 兼容入口里的 Kimi 不能直接套 Claude 28px patch 规则。 |
| MiniMax-M3 | `minimax`、`minimax-*`、`gitee`、`moark`、`together`、`volcengine*`、OpenRouter/Vercel 等 | MiniMax-M3 官方模型卡说明它是 1M context 的原生多模态模型，支持 text/image/video 混合输入，OpenAI-compatible 示例使用 `image_url`。来源：[MiniMax-M3 model card](https://huggingface.co/MiniMaxAI/MiniMax-M3)。 | 缺少公开稳定图片 token 公式时使用 generic 估算。由于 context 很大，也仍需控制 base64 请求体。 |
| xAI Grok vision | `xai` 或聚合层 `x-ai/*` | xAI 支持 base64 data URL 或公网 URL，Responses 风格 `input_image` 可带 `detail`。通用限制：单图最大 20MiB，不限制图片数量，支持 jpg/jpeg/png，文本和图片顺序均可。来源：[xAI Image Understanding](https://docs.x.ai/developers/model-capabilities/images/understanding)。 | 可把 xAI 归入支持 `detail` 的 Responses-compatible 特例，但格式上仅默认 JPEG/PNG 最稳。 |
| Groq Llama 4 Scout / Qwen vision | `groq` | Groq Llama 4 Scout：URL 图片请求最大 20MB；每图最大 33MP；base64 图片请求最大 4MB；最多 5 张图。Groq Qwen vision 也最多 5 张图，URL 图片请求最大 20MB。来源：[Groq Images and Vision](https://console.groq.com/docs/vision)。 | 全局 base64 输出体积默认不宜超过 4MB；多图预算第一阶段即应为后续预留。 |
| Amazon Nova / Bedrock Claude | `amazon-bedrock` | Nova：多图 payload 总限制 25MB，可用 S3 URI 绕开 payload；最大分辨率 8000x8000；会按接近宽高比自动 rescale，并给出若干 token 估算示例；bounding box 输出为 [0,1000) 坐标。来源：[Amazon Nova image understanding](https://docs.aws.amazon.com/nova/latest/userguide/modalities-image.html)。Bedrock 上 Claude 单图 base64 限制比 Claude API 更严，见 Claude 官方文档。 | Bedrock provider 应采用更严格 payload 策略。坐标映射要记录原图和发送图尺寸，兼容 [0,1000) 归一化坐标。 |
| Cloudflare Workers AI Llama 4 Scout | `cloudflare-workers-ai` | Workers AI 模型页确认 Llama 4 Scout 原生多模态、Vision=Yes、context window 约 131K；但模型页未给出详细图片大小/格式限制。来源：[Cloudflare Workers AI Llama 4 Scout](https://developers.cloudflare.com/workers-ai/models/llama-4-scout-17b-16e-instruct/)。 | 对 Cloudflare Workers AI 暂用网关保守策略，不发送非标准 `detail`。 |
| Llama Vision / Gemma / open VLM gateway models | `groq`、`nvidia`、`cloudflare-*`、OpenRouter、Together 等 | 默认配置覆盖 Llama 4 Scout、Llama 3.2 11B/90B Vision、Gemma 4 视觉模型。Groq 和 Cloudflare 有部分托管限制文档；NVIDIA/Together/OpenRouter 等更多依赖网关自身限制和底层模型卡。 | 需要 `llama-vision` / `gemma-vision` family。没有明确限制时按网关 policy + generic 估算。 |
| Z.AI / GLM-V | `zai*`、OpenRouter/Vercel、Cloudflare Workers AI 等 | GLM-4.5V 模型卡说明支持图像、视频、文档、GUI、grounding；GLM-4.1V 说明支持任意宽高比和最高 4K 图像分辨率；输出 bbox 为相对 0-1000 的归一化坐标。来源：[GLM-4.5V model card](https://huggingface.co/zai-org/GLM-4.5V)。 | `detail`/`raw` 对 GUI、grounding、文档很重要。坐标映射必须作为元数据保留。 |
| Xiaomi MiMo | `xiaomi`、`xiaomi-token-plan-*` | 默认配置中 `mimo-v2.5` 声明 `text,image`。本轮未找到稳定公开 API 限制文档，因此不能推断具体 token/尺寸规则。 | 单独设为 `mimo` family，先走 OpenAI-compatible 保守策略；如果官方文档/模型目录提供视觉约束，再加 override。 |
| DeepSeek-on-gateway | `alibaba-standard`、`volcengine-agentplan`、`volcengine-codingplan` | 默认配置中 `deepseek-v4-pro` 或 `deepseek-v4-flash` 在部分套餐 provider 里声明 image，但 DeepSeek 官方 provider 默认仍是 text-only。本轮官方 DeepSeek API 检索未确认 V4 通用 vision API。 | 标记为 `gateway-asserted-vision`，只按对应 gateway policy 发送。不要把它归入 DeepSeek 官方模型族规则；建议后续专门核对这些默认声明是否准确。 |

#### 2.5.4 调研后的工程结论

1. **默认长边 1568px 是合理基线**：它与 Claude 标准档对齐，也足以覆盖大多数普通识图和轻量截图；对 Gemini/OpenAI/Groq 不会过度保守。
2. **`detail` 模式建议长边 2048px 起步**：OpenAI high 和很多模型的高保真路径会在 2048 附近出现关键阈值；但对 Claude 高分辨率、GLM-V 4K、OpenAI original 这类场景，应保留更高上限作为显式 `raw/original` 或后续配置。
3. **base64 输出体积默认应低于 4MB**：Groq base64 请求限制是 4MB，Bedrock Claude 的 5MB base64 单图限制也偏紧。考虑 base64 膨胀，处理后 raw bytes 目标值应优先控制在 3MB 左右，`detail` 可放宽。
4. **图片格式默认保持 PNG/JPEG/WebP 输入识别，但发送格式要保守**：OpenAI/Claude/OpenRouter 支持 GIF/WebP，Gemini 支持 HEIC/HEIF，但 xAI 仅列出 JPEG/PNG，Groq 示例也偏 JPEG。跨 provider 默认发送 JPEG/PNG 更稳。
5. **Doubao Seed 和 Qwen Plus 要单独建 family，不要埋在 generic 聚合层里**：它们是默认配置里的高频视觉模型，且官方公开资料中没有像 OpenAI/Claude/Gemini 那样清晰的通用图片 token 公式。第一阶段应按 OpenAI-compatible 发送格式 + 保守预处理，Phase 2 再补模型目录/供应商返回的精确约束。
6. **Kimi、MiniMax、GLM、MiMo、Llama Vision、DeepSeek-on-gateway 也要进入 family 映射**：这些都已经在默认配置中出现，不能落到一个没有审计依据的通用分支。
7. **聚合供应商不能统一套用底层参数**：OpenRouter/Vercel/Cloudflare/GitHub Copilot/opencode 等可能转发到底层模型，但不保证透传 `detail`、`media_resolution` 或 provider-specific 参数。第一阶段应把这些参数作为内部元数据，只有明确支持时才写进请求。
8. **坐标任务需要元数据而不是只看图片**：OpenAI、Claude、Gemini、Nova、GLM-V 都涉及不同的归一化或缩放行为。必须记录原图尺寸、发送图尺寸和缩放比例，并提供坐标映射 helper。

## 3. 设计目标

### 3.1 用户目标

1. 默认情况下，图片识别更稳定、更快、更省上下文。
2. 对截图、OCR、小字和 UI 任务保留足够细节。
3. 用户不需要理解每个供应商的视觉 token 规则，也能得到合理行为。
4. 当图片被缩放或处理时，工具结果能明确说明发生了什么。
5. 保留显式 `read` 路径，不强制把所有图片自动附到用户消息里。

### 3.2 工程目标

1. 图片预处理逻辑集中在独立包中，避免散落在 provider 适配层。
2. provider 层只负责把标准 `ImageContent` 转为供应商请求格式。
3. 图片 token 估算尽量基于尺寸和 provider/model 策略。
4. 新字段使用 `omitempty`，保持 session 兼容。
5. 默认策略保守，不引入复杂 tile 逻辑作为第一阶段依赖。

### 3.3 非目标

第一阶段不解决：

- 完整 OCR 引擎。
- 本地视觉模型。
- 自动图像语义分类。
- 智能检测 ROI 并自动裁剪。
- 高级 tile 拼接和跨 tile 空间推理。
- 改变 settings.json 现有字段语义。

## 4. 总体方案

新增统一图片处理层：

```
internal/imageproc/
├── imageproc.go       # public API: PrepareFile/PrepareBytes plus inspect/resize/transcode helpers
└── policy.go          # mode/policy/defaults and provider/model family hints

internal/context/tokenizer.go # generic and provider/model-aware visual token estimation
```

核心思想：

1. 读取图片前先 inspect 宽高和格式。
2. 根据 mode/policy 决定是否缩放、转码或拒绝。
3. 生成处理后的图片 payload 和元数据。
4. `read`、browser screenshot、未来 `/attach-image` 复用同一套逻辑。
5. 上下文估算使用图片元数据，不再只看 base64 长度。

## 5. 数据结构

### 5.1 ImageContent 扩展

建议在 `provider.ImageContent` 上增加可选元数据：

```go
type ImageContent struct {
    Data     string `json:"data"`     // base64 encoded
    MimeType string `json:"mimeType"` // e.g. "image/png"

    Width  int `json:"width,omitempty"`
    Height int `json:"height,omitempty"`
    Bytes  int `json:"bytes,omitempty"`

    OriginalWidth  int `json:"originalWidth,omitempty"`
    OriginalHeight int `json:"originalHeight,omitempty"`
    OriginalBytes  int `json:"originalBytes,omitempty"`

    Detail string  `json:"detail,omitempty"` // auto, fast, detail, raw
    Scale  float64 `json:"scale,omitempty"`

    Cropped    bool `json:"cropped,omitempty"`
    CropX      int  `json:"cropX,omitempty"`
    CropY      int  `json:"cropY,omitempty"`
    CropWidth  int  `json:"cropWidth,omitempty"`
    CropHeight int  `json:"cropHeight,omitempty"`
}
```

说明：

- `Width/Height/Bytes` 表示实际发送给模型的图片。
- `Original*` 表示源图片。
- `Detail` 是对处理策略的描述，不一定所有 provider 都直接支持。
- `Scale` 和 `Crop*` 用于后续坐标映射。
- 字段全部 `omitempty`，旧 session 不受影响。

### 5.2 图片处理请求

```go
type Mode string

const (
    ModeAuto   Mode = "auto"
    ModeFast   Mode = "fast"
    ModeDetail Mode = "detail"
    ModeRaw    Mode = "raw"
)

type Policy struct {
    Mode              Mode
    MaxFileBytes      int64
    MaxPixels         int
    MaxLongEdge       int
    MaxOutputBytes    int
    Crop              *Crop
}

type Result struct {
    Data     []byte
    MimeType string
    Meta     Meta
}

type Meta struct {
    Width          int
    Height         int
    Bytes          int
    OriginalWidth  int
    OriginalHeight int
    OriginalBytes  int
    Resized        bool
    Transcoded     bool
    Scale          float64
    Detail         string
    Cropped        bool
    CropX          int
    CropY          int
    CropWidth      int
    CropHeight     int
}
```

### 5.3 默认策略

建议默认值：

| Mode | 用途 | 长边 | 输出体积 | 说明 |
|------|------|------|----------|------|
| `fast` | 普通图片快速理解 | 768 或 1024 | 1-2MB | 低成本、低延迟 |
| `auto` | 默认 | 1568 | 约 3MB raw bytes | 对齐 Claude 标准档，并兼顾 Groq/Bedrock 等严格 base64 限制；非 `raw` 输出超限时继续缩小重编码 |
| `detail` | OCR/截图/表格/代码 | 2048 或 2560 | 4-8MB | 保留细节；对 Groq/Bedrock 等严格供应商需按 provider 下调；非 `raw` 输出超限时继续缩小重编码 |
| `raw` | 明确要求原图 | 不缩放 | 现有上限 | 只做安全校验 |

第一阶段默认使用 `auto`。`raw` 不应绕过安全限制。WebP 输入第一阶段明确引入 `golang.org/x/image/webp` 做 decode/inspect；发送侧仍优先 JPEG/PNG，不引入 WebP encoder。

## 6. 工具层设计

### 6.1 read 工具参数扩展

保持现有参数兼容，新增可选参数：

```json
{
  "imageMode": {
    "type": "string",
    "enum": ["auto", "fast", "detail", "raw"],
    "description": "Image processing mode for image files"
  },
  "maxLongEdge": {
    "type": "integer",
    "description": "Override maximum long edge for image resizing"
  },
  "crop": {
    "type": "object",
    "properties": {
      "x": {"type": "integer"},
      "y": {"type": "integer"},
      "width": {"type": "integer"},
      "height": {"type": "integer"}
    }
  }
}
```

Phase 1 可以先实现 `imageMode` 和 `maxLongEdge`，`crop` 留到 Phase 2。

当前实现状态：`imageMode`、`maxLongEdge`、`crop` 均已在 `read` 中落地。`crop` 在源图像像素坐标上执行，先 crop 再 resize/transcode，越界裁剪会返回错误。

### 6.2 read 输出说明

当前描述：

```
[Image file: path, size: N bytes, type: image/png]
```

建议改为：

```
[Image file: path, original: 3024x1964 4.8MB image/png, sent: 1568x1019 620KB image/jpeg, mode: auto]
```

如果没有缩放：

```
[Image file: path, 1280x720 312KB image/png, mode: auto]
```

这样用户和 agent 都能看见图片是否被处理。

### 6.3 browser screenshot 接入

浏览器截图已接入统一处理。当前行为：

1. 未指定 `outputPath` 时，截图 bytes 进入 `imageproc.PrepareBytes()`。
2. 默认使用 `detail` policy；也支持 `imageMode` 和 `maxLongEdge`。
3. 输出描述包含原始截图尺寸、发送尺寸和处理模式。
4. 指定 `outputPath` 时仍保存浏览器返回的原始截图，不做转码或缩放。

截图往往包含文字和 UI，默认不应压得过低。

### 6.4 TUI 粘贴图片

保留 `/paste-image` 当前语义：保存文件并插入路径。

后续可新增：

- `/attach-image`：保存并直接附加到下一条用户消息。
- `/paste-image --attach`：兼容已有命令但提供直发能力。

第一阶段不强制改变 TUI 输入消息结构，避免扩大改动面。

## 7. Provider 层设计

### 7.1 OpenAI

OpenAI Responses 图片 block 可增加 `detail`。官方 OpenAI 支持 `low/high/original/auto`，其中 `original` 只适合明确支持的较新模型或供应商：

```go
type responsesContentBlock struct {
    Type     string `json:"type"`
    Text     string `json:"text,omitempty"`
    ImageURL string `json:"image_url,omitempty"`
    Detail   string `json:"detail,omitempty"`
}
```

映射建议：

- `ModeFast` -> `detail: "low"`
- `ModeAuto` -> `detail: "auto"`
- `ModeDetail` -> `detail: "high"`
- `ModeRaw` -> 当前实现退回 `detail: "high"`；后续仅在模型和 vendor 明确支持时再映射到 `original`

是否发送 `detail` 需要确认对应模型和 API 版本兼容。若兼容性不确定，可以先只保留 `ImageContent.Detail`，不在请求中发送。

### 7.2 Anthropic

Anthropic 图片 block 当前没有通用 `detail` 字段。优化重点应放在发送前的尺寸控制和 token 估算。

### 7.3 Gemini

Gemini inline data 没有 OpenAI 风格的 per-image `detail` 字段。当前实现会在 Google Gemini/Vertex 请求的 `generationConfig.mediaResolution` 中做全局映射：`fast/low` -> `MEDIA_RESOLUTION_LOW`，`detail/high/raw/original` -> `MEDIA_RESOLUTION_HIGH`，`auto` 不发送该字段。Gemini 3 / v1alpha 的 per-part `resolution` 暂不接入。

### 7.4 OpenAI-compatible 供应商

许多 OpenAI-compatible 供应商对 `detail` 支持不一致。默认不应向所有兼容供应商发送额外字段，除非 vendor adapter 明确标记支持。

### 7.5 Provider 策略分层

建议新增内部 provider image policy，不直接暴露为用户配置：

| Policy family | 适用 provider/model | 关键约束 |
|---------------|---------------------|----------|
| `openai-responses` | `openai` 官方 provider | 可发送 `detail`，支持 `original` 时再启用。 |
| `anthropic-direct` | `anthropic` | 单图 base64 10MB，标准长边 1568，高分辨率模型长边 2576。 |
| `anthropic-bedrock` | `amazon-bedrock` 中 Claude | 单图 base64 5MB，输出体积更保守。 |
| `gemini` | `google-gemini` / `google-vertex` | 按 384/768 tile 估算；Google API 请求会按图片 detail 映射全局 `mediaResolution`。 |
| `doubao-seed` | `doubao-seed-2-0-code`、`doubao-seed-2-0-pro`、`doubao-seed-evolving`、`doubao-seed-2-1-pro-*` | OpenAI-compatible 发送格式；无公开稳定图片 token 公式时用 generic，`detail` 保留更高长边。 |
| `qwen-plus` | `qwen3.6-plus`、`qwen3.7-plus`、`qwen3.6-flash/max` 等默认视觉 Qwen | OpenAI-compatible 发送格式；当前使用 Qwen-like 28px patch 近似估算，并保留 `maxPixels`/模型目录 override 扩展点。 |
| `kimi` | `kimi-k2.5`、`kimi-k2.6`、`kimi-k2.7-code`、`k2p7`、`kimi-for-coding`、Fireworks Kimi routers | 不按 Claude/OpenAI token 规则硬套；根据接入 API 发送，估算先 generic。 |
| `minimax` | MiniMax M3 across MiniMax/Gitee/Moark/Together/Volcengine/OpenRouter/Vercel | 原生多模态，缺少稳定公开 token 公式时按 generic 控制体积。 |
| `glm` | `glm-5.2`、`glm-5v-turbo`、GLM via Z.AI/Cloudflare/Gitee/Moark/Volcengine/OpenRouter/Vercel | GUI/文档/grounding 优先保留 detail；坐标映射元数据必须保留。 |
| `mimo` | Xiaomi `mimo-v2.5` and token-plan variants | OpenAI-compatible 发送格式；无公开稳定限制时保守处理。 |
| `llama-vision` | Llama 4 Scout, Llama 3.2 Vision via Groq/Cloudflare/NVIDIA/OpenRouter | 网关限制优先；无明确规则时 generic 估算。 |
| `gemma-vision` | Gemini/Gemma 4 via Google/Cloudflare | Google API 走 Gemini policy；其他网关走 conservative generic。 |
| `deepseek-gateway-vision` | DeepSeek V4 Pro/Flash only where provider preset declares image | gateway-specific capability；不套 DeepSeek 官方 text-only provider 假设，建议后续核对声明准确性。 |
| `groq` | `groq` | base64 request 4MB，URL 20MB，最多 5 图，最大 33MP；当前实现额外按 Groq provider/baseURL 把输出 raw bytes cap 到 3MiB，为 base64 膨胀留空间。 |
| `xai` | `xai` | JPEG/PNG，单图 20MiB，可发送 `detail`。 |
| `openai-compatible-generic` | 未匹配到上述 family 的 OpenAI-compatible 模型 | 不发送非标准字段，按保守体积和尺寸处理。 |

## 8. Token 估算设计

### 8.1 当前问题

当前图片估算逻辑依赖 base64 长度，不能反映视觉 token 的真实规律。应改为优先使用图片宽高。

### 8.2 估算接口

可以在现有 `TokenEstimator` 基础上扩展 provider/model-aware 图片估算：

```go
type ImageTokenEstimator interface {
    EstimateImageTokens(image *provider.ImageContent, model *provider.Model) int
}
```

或保持内部函数，不额外暴露接口：

```go
func estimateImageTokens(image *provider.ImageContent, model *provider.Model) int
```

### 8.3 估算规则

建议分三层：

1. provider/model 明确规则。
2. vendor family 规则。
3. 通用兜底规则。

示例：

```go
func estimateAnthropicImageTokens(w, h int) int {
    return ceilDiv(w, 28) * ceilDiv(h, 28)
}

func estimateGeminiImageTokens(w, h int) int {
    if w <= 384 && h <= 384 {
        return 258
    }
    return ceilDiv(w, 768) * ceilDiv(h, 768) * 258
}

func estimateGenericImageTokens(w, h int) int {
    return ceilDiv(w, 512) * ceilDiv(h, 512) * 800
}
```

如果图片缺少宽高，保留当前最小估算逻辑作为 fallback。

### 8.4 模型族推断

默认配置中聚合供应商很多，不能只看 provider ID。当前已在 `internal/imageproc/policy.go` 增加 `Hint` + `InferFamily()` + `PolicyForHint()`，并在 agent 构造时把 provider/model 上下文回填到 `tools.Registry`。

当前推断顺序：

1. model ID 前缀/名称模式优先：`doubao-seed-*`、`seed-2-*`、`qwen*`、`kimi-*`、`k2p7`、`MiniMax-*`、`glm-*`、`mimo-*`、`grok-*`、`amazon.nova*`、`llama-*vision/scout*`、`gemma-*`、`gemini-*`、`pixtral-*`、`mistral-*`、`devstral-*`、`claude-*`、`anthropic/claude-*`、`gpt-*`、`o-series`、`deepseek-v4-*` on known gateway providers。
2. provider/vendor/baseURL 兜底：`xiaomi`、`minimax`、`kimi/moonshot`、`zai/bigmodel`、`xai`、`amazon-bedrock`、`google-gemini`、`google-vertex`、`mistral`、`volcengine`、`alibaba/bailian/dashscope`、`anthropic`、`openai-responses`。
3. fallback generic。

这个顺序是有意的：`anthropic-messages` 下面可能是 MiniMax/Kimi/Fireworks，`openai-chat` 下面可能是 Qwen/Doubao/Kimi/MiniMax/GLM/Grok/Llama/MiMo。先按模型 ID 识别可以避免把这些模型误套到 Claude 或 OpenAI 官方策略上。

当前策略输出仍然保守：

1. 通用默认仍是 `fast=1024/2MB`、`auto=1568/3MB`、`detail=2048/6MB`、`raw=原图但安全校验`。
2. OpenAI/Anthropic/Gemini/Grok 可把输入文件上限提高到 20MiB。
3. Anthropic-on-Bedrock 使用更严格的 4MiB 文件上限和 3MiB 输出目标。
4. Doubao Seed、Qwen、Kimi、GLM 在显式 `detail` 模式下把长边提高到 2560，默认 `auto` 仍保持 1568。
5. Amazon Nova、Mistral/MiniMax、MiMo/Llama/Gemma 等 family 当前只做保守输出体积 cap，不透传 provider-specific 参数；Groq 这类严格网关额外按 provider/baseURL 应用更低输出 cap。非 `raw` 输出如果编码后仍超 cap，会继续缩小重编码；仍无法满足时返回错误。
6. `deepseek-v4-*` 只有在已知 gateway provider 下才归为 `deepseek-gateway-vision`；直接 DeepSeek provider 仍不按视觉模型处理。

已完成：

1. `ResolveTokenEstimator()` 在有 model 上下文时返回模型感知估算器。
2. Claude/Bedrock Claude、Qwen-like family 图片按 `ceil(width/28) * ceil(height/28)` 估算。
3. Gemini 图片按小图 258 tokens、大图 768 tile * 258 tokens 估算。
4. OpenAI/Grok 图片按 `fast/low=85 tokens`、其他模式 `85 + 170 * 512-tile` 近似估算。
5. 同一消息内多张图片会逐张累加估算。

仍未完成：

1. Doubao/Kimi/MiniMax/GLM/MiMo/Nova/Llama/Gemma 的更精确官方公式或模型目录约束；Qwen 当前只有 28px patch 近似。
2. 从供应商模型目录/API 动态发现图片限制。
3. Gemini per-part `resolution` 等实验 API 能力。

## 9. 配置设计

第一阶段建议不修改 `settings.json` schema，使用内置默认策略即可。

如果后续需要开放配置，建议增加独立顶层配置：

```json
{
  "image": {
    "mode": "auto",
    "maxPixels": 40000000,
    "maxLongEdge": 1568,
    "maxOutputBytes": 4194304,
    "screenshotMode": "detail"
  }
}
```

但这属于后续阶段。当前 repository 指南要求不要轻易改变 settings schema，因此第一版应避免新增配置字段。

## 10. 分阶段计划

### Phase 1: 统一预处理基础（已落地）

目标：把“原样透传”升级为“可控的自动处理”。

改动：

1. 新增 `internal/imageproc`。
2. 支持 inspect PNG/JPEG/WebP/GIF 的尺寸和格式。
3. 支持 JPEG/PNG 常见格式的 resize 和 encode。
4. `read` 工具接入 `imageMode=auto|fast|detail|raw`。
5. 扩展 `ImageContent` 元数据。
6. 工具输出描述展示原图和发送图信息。
7. 内置 provider image policy 的保守默认：`auto` 长边 1568，输出 raw bytes 目标约 3MB。
8. 增加单元测试。

验收：

- 已验证：大图会被等比缩放。
- 已验证：小图不被无意义重编码。
- 已验证：超大像素图片被拒绝。
- 已验证：旧文本读取行为不变。
- 已验证：旧图片 session 仍可读取。

### Phase 1B: 默认模型族 policy（已落地）

目标：默认配置中声明视觉能力的模型不再全落到无区分 generic policy。

改动：

1. 新增 `imageproc.Hint`、`InferFamily()`、`PolicyForHint()`。
2. `tools.Registry` 保存 provider/model image hint，并为 `read` 提供当前 `ImagePolicy()`。
3. `agent.New()` / `NewWithLoopConfig()` 根据 `Config.Provider`、`Config.Vendor`、`Config.Model`、`Config.Settings` 自动回填 hint。
4. `read` 的图片大小检查改为使用当前 policy 的 `MaxFileBytes`。
5. 覆盖 Seed、Qwen、Kimi、MiniMax、GLM、MiMo、Grok、Llama、Gemma、Amazon Nova、Bedrock Claude、DeepSeek-on-gateway 等默认配置 family；Seed/Qwen/Kimi/GLM 的 `detail` 长边提升到 2560。
6. 增加 policy 推断和 agent/registry 集成测试。

验收：

- 已验证：`seed-2-pro` / `doubao-seed-2-0-code` 命中 `doubao-seed`。
- 已验证：`qwen3.6-plus` / `qwen3.7-plus` 命中 `qwen`。
- 已验证：`MiniMax-M3` 即使走 `anthropic-messages` 也命中 `minimax`，不会误判 Claude。
- 已验证：Bedrock Claude 使用更严格文件/输出体积。
- 已验证：直接 DeepSeek provider 不因 `deepseek-v4-*` 名称误判为视觉模型；只有 gateway provider 命中 `deepseek-gateway-vision`。

### Phase 2: 上下文估算和 provider detail（部分落地）

目标：让 compaction 和成本估算更贴近真实视觉 token。

改动：

1. 已完成：图片 token 估算优先使用宽高。
2. 已完成：OpenAI/Grok tile 近似、Claude/Qwen 28px patch、Gemini 768 tile 接入估算。
3. 已完成：provider/model-aware 估算规则接入 `ResolveTokenEstimator()`。
4. 已完成：OpenAI/xAI 支持 `detail`，仅在官方 baseURL 确认兼容时发送；Gemini 支持全局 `mediaResolution`。
5. 已完成：browser screenshot 接入统一预处理。
6. 更新相关测试。

验收：

- 图片估算不再主要受 base64 长度影响。
- 不同尺寸图片估算随 tile/patch 规则变化。
- browser screenshot 不再原样无限制透传。

### Phase 3: 裁剪和显式细节控制（部分落地）

目标：支持 OCR、局部 UI 和坐标任务。

改动：

1. 已完成：`read` 支持 `crop` 参数。
2. 已完成：输出描述包含 crop 坐标；`ImageContent` 保存 `Cropped/CropX/CropY/CropWidth/CropHeight`。
3. 已完成：`provider.ImageContent` 和 public `agent.ImageContent` 提供坐标映射 helper，可把发送图坐标或 `[0,1000]` 归一化坐标映射回原图坐标。
4. 已完成：`read` tool guideline 更明确地提醒 agent 对小字/局部区域使用 `detail` / `crop`。

验收：

- 已验证：用户可要求“看这张图左上角”，agent 能读取局部图。
- 已验证：模型输出坐标时，可基于已有 crop/scale 元数据通过 helper 映射回原图坐标。

### Phase 4: 多图预算和 tile

目标：处理超大图、长截图、多图批量任务。

改动：

1. 全图缩略图 + 局部 tile。
2. 多图总视觉 token 预算。
3. tile 描述包含位置和序号。
4. 对长截图支持纵向切片。

验收：

- 超长网页截图不会直接爆请求体。
- agent 能同时获得全局上下文和局部细节。

## 11. 风险与权衡

### 11.1 预处理可能损失细节

缓解：

- 默认长边不要过低。
- 提供 `detail` 和 `raw`。
- 工具输出明确说明处理尺寸。
- OCR/截图默认更偏 detail。

### 11.2 转码可能改变图片语义

例如透明 PNG、像素 art、UI 细线、压缩伪影。

缓解：

- 小 PNG 默认不转码。
- 检测 alpha 时优先保留 PNG。
- 文字截图使用高质量编码。
- 只在体积收益明确时转码。

### 11.3 Go 标准库格式支持有限

标准库支持 JPEG/PNG/GIF，WebP 需要额外依赖。当前 `read` 已支持 WebP 扩展名，但如果要解码尺寸和重采样，需要决定是否引入依赖。

候选：

- 已采用：引入 `golang.org/x/image/webp` 读取 WebP，WebP 输入可以进入统一 inspect/resize/transcode 流程。
- 若编码 WebP 需求强，再评估额外 encoder。

### 11.4 GIF 动图语义

多数视觉 API 对 GIF 支持不一致，可能只取首帧或拒绝。

缓解：

- 当前 GIF 通过 Go 标准库 decoder 读取；非 `raw` 模式会按首帧进入统一 resize/transcode 流程。
- 后续应在输出说明里明确 GIF/动图的首帧处理方式。
- 后续再考虑视频/多帧抽样。

### 11.5 Provider 兼容性

OpenAI-compatible 供应商不一定接受 `detail` 字段。

缓解：

- `detail` 先作为内部元数据。
- 只有官方 OpenAI 或明确支持的 vendor 才发送。

## 12. 测试状态

### 12.1 imageproc 单元测试

- 已覆盖：大 JPEG 等比缩放。
- 已覆盖：`raw` 模式不缩放但仍校验安全限制。
- 已覆盖：超大像素拒绝。
- 已覆盖：裁剪输出尺寸、crop 元数据和越界裁剪错误。
- 已覆盖：Groq provider/baseURL 会应用更严格的输出体积 cap。
- 已覆盖：输出超出 `MaxOutputBytes` 时继续缩小重编码并满足 cap。
- 已覆盖：alpha PNG resize 后仍保持 PNG，不被错误转成 JPEG。
- 已覆盖：WebP decode 后转码为 provider 兼容的 JPEG/PNG。

### 12.2 read 工具测试

- 已覆盖：默认图片输出包含 image block。
- 已覆盖：输出描述包含原图和发送图尺寸。
- 已覆盖：`maxLongEdge` resize。
- 已覆盖：registry image policy 被 `read` 使用。
- 已覆盖：`crop` 参数、crop 元数据和输出描述。
- 已覆盖：大文件错误信息。
- 已覆盖：`imageMode=fast/detail/raw` 的端到端差异测试。
- 已覆盖：坏图片错误信息。

### 12.3 provider 测试

- 已覆盖：OpenAI Chat Completions 图片 `detail` 只在官方 OpenAI/xAI baseURL 出现。
- 已覆盖：OpenAI Responses 图片 `detail` 只在官方 OpenAI/xAI baseURL 出现。
- 已覆盖：伪官方域名（例如包含 `api.openai.com` 的非官方 host）不会透传 `detail`。
- 已覆盖：Gemini/Vertex 全局 `mediaResolution` 按图片 detail 映射，`auto` 不发送该字段。
- 已通过全量测试：Anthropic/Gemini 现有图片序列化未被新增元数据破坏。

### 12.4 context/tokenizer 测试

- 已覆盖：无宽高旧图片 fallback 到兼容逻辑。
- 已覆盖：有宽高图片按 generic 规则估算。
- 已覆盖：Claude/Bedrock Claude、Qwen、Gemini、OpenAI low/high 的模型感知图片估算。
- 已覆盖：同一消息内多图估算累加。
- 待补充：更多 provider family 的精确公式或模型目录约束。

### 12.5 coordinate helper 测试

- 已覆盖：发送图坐标映射回原图坐标。
- 已覆盖：裁剪后发送图坐标映射回原图坐标。
- 已覆盖：`[0,1000]` 归一化矩形映射回原图坐标。

### 12.6 browser 测试

- 已覆盖：browser tool 注册/移除。
- 已覆盖：截图 bytes 进入统一图片处理后返回 rich image block、尺寸元数据和 resize 描述。

## 13. 待讨论问题

1. 默认 `auto` 长边是否继续保持 `1568`，还是在更多实测后调到 `2048`？
2. 截图默认已使用 `detail`；是否需要为 full-page/长截图单独使用更激进的压缩或 tile？
3. `read` 是否应该默认把 PNG 截图转 JPEG，还是只在超过体积阈值时转？
4. WebP 已引入 decode/inspect；是否需要 WebP encoder 或继续统一输出 PNG/JPEG？
5. `/paste-image` 是否新增直接 attach 模式，还是继续保持路径优先？
6. 是否需要在 settings 中暴露 image 配置，还是先使用内置策略？
7. OpenAI/xAI 官方 baseURL 已透传 `detail`；是否需要显式 vendor capability 表让部分聚合层也能安全透传？
8. 坐标映射 helper 已落地；是否需要进一步在工具输出中展示常用映射示例？
9. 是否要把 Groq/Bedrock 这类严格体积限制作为全局默认，还是只作为 provider policy？
10. 对 OpenRouter/Vercel/GitHub Copilot/opencode 这类聚合层，是否只按底层 model ID 推断，还是增加显式 vendor capability 表？
11. Doubao Seed、Qwen Plus、Kimi、MiMo、MiniMax、DeepSeek-on-gateway 的精确图片限制是否应通过 provider 模型目录/API 动态发现，而不是硬编码在客户端？
12. 默认配置里 `deepseek-v4-pro` / `deepseek-v4-flash` 的 image 声明是否准确，是否需要单独开配置核对任务修正 `Input`？

## 14. 已采纳决策和剩余建议

已采纳：

1. 不改 settings schema。
2. 新增 `internal/imageproc`。
3. `read` 默认 `auto`，最长边先用 `1568`。
4. `auto` 处理后 raw bytes 目标约 `3MB`，为 Groq base64 4MB 和 Bedrock Claude 5MB base64 留余量。
5. 不做 tile，不做自动 ROI；先实现显式 `crop`。
6. 截图直接返回图片时默认 `detail` 并进入统一预处理；保存到 `outputPath` 时保留原始截图。
7. WebP 第一版引入 `golang.org/x/image/webp` 读取和 inspect；发送侧优先 JPEG/PNG。
8. 默认配置中所有 38 个视觉 provider preset 都必须能落入一个明确 policy family；第一版精确公式缺失的 family 先走 generic/保守估算。
9. `ImageContent` 加元数据，为后续 token 估算和坐标映射铺路。
10. `detail` 默认作为内部元数据；只有 OpenAI 官方和 xAI 这类已确认支持的 provider 发送。

剩余建议：

1. 先不要开放 settings schema，等默认策略跑一段时间后再决定是否暴露配置。
2. tile/多图预算应等 crop/detail 的实际使用反馈稳定后再做。
3. Gemini per-part `resolution` 应等待 v1alpha/Gemini 3 相关 API 稳定后再评估。
4. 对 DeepSeek-on-gateway 的 image 声明需要单独核对默认配置准确性。
