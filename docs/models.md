# OpenRouter Model Catalog

> Auto-generated from OpenRouter API on 2026-07-21
>
> Total models: **803** | Reasoning: **298** | Free: **17**

## How to Update

```bash
# 1. Fetch latest catalog
curl -s 'https://openrouter.ai/api/frontend/v1/catalog/models' > /tmp/or_models.json

# 2. Run this script
python3 docs/scripts/generate-openrouter-models.py

# 3. Review and commit
# git diff docs/models.md
```

## Legend

| Symbol | Meaning |
|--------|---------|
| 🧠 | Supports reasoning/thinking |
| 🆓 | Free tier available |
| ⚠️ | Mandatory reasoning (cannot disable) |
| T | Text input |
| I | Image input |
| V | Video input |
| A | Audio input |

## Model Groups

| Group | Count |
|-------|-------|
| [Claude](#claude) | 35 |
| [Gpt](#gpt) | 86 |
| [Gemini](#gemini) | 38 |
| [Gemma](#gemma) | 14 |
| [Llama](#llama) | 63 |
| [Qwen](#qwen) | 82 |
| [Deepseek](#deepseek) | 20 |
| [Mistral](#mistral) | 48 |
| [Grok](#grok) | 26 |
| [Nova](#nova) | 6 |
| [Minimax](#minimax) | 11 |
| [Glm](#glm) | 19 |
| [Ernie](#ernie) | 5 |
| [Kimi](#kimi) | 10 |
| [Nemotron](#nemotron) | 12 |
| [Cohere](#cohere) | 13 |
| [Phi](#phi) | 9 |
| [Aetherwiing](#aetherwiing) | 1 |
| [Agentica Org](#agentica-org) | 1 |
| [Ai21](#ai21) | 7 |
| [Aion Labs](#aion-labs) | 5 |
| [Alibaba](#alibaba) | 4 |
| [All Hands](#all-hands) | 1 |
| [Allenai](#allenai) | 8 |
| [Alpindale](#alpindale) | 2 |
| [Anthracite Org](#anthracite-org) | 2 |
| [Arcee Ai](#arcee-ai) | 11 |
| [Arliai](#arliai) | 1 |
| [Baai](#baai) | 3 |
| [Baidu](#baidu) | 2 |
| [Bigcode](#bigcode) | 1 |
| [Black Forest Labs](#black-forest-labs) | 4 |
| [Bytedance](#bytedance) | 5 |
| [Bytedance Research](#bytedance-research) | 1 |
| [Bytedance Seed](#bytedance-seed) | 5 |
| [Canopylabs](#canopylabs) | 1 |
| [Databricks](#databricks) | 1 |
| [Deepcogito](#deepcogito) | 1 |
| [Deepgram](#deepgram) | 1 |
| [Essentialai](#essentialai) | 1 |
| [Featherless](#featherless) | 1 |
| [Fireworks](#fireworks) | 1 |
| [Google](#google) | 10 |
| [Gryphe](#gryphe) | 2 |
| [Hexgrad](#hexgrad) | 1 |
| [Huggingfaceh4](#huggingfaceh4) | 2 |
| [Ibm Granite](#ibm-granite) | 2 |
| [Inception](#inception) | 3 |
| [Inclusionai](#inclusionai) | 3 |
| [Infermatic](#infermatic) | 1 |
| [Inflatebot](#inflatebot) | 1 |
| [Inflection](#inflection) | 2 |
| [Intel](#intel) | 1 |
| [Intfloat](#intfloat) | 3 |
| [Jebcarter](#jebcarter) | 1 |
| [Jondurbin](#jondurbin) | 2 |
| [Koboldai](#koboldai) | 1 |
| [Kwaipilot](#kwaipilot) | 4 |
| [Kwaivgi](#kwaivgi) | 3 |
| [Liquid](#liquid) | 8 |
| [Liuhaotian](#liuhaotian) | 1 |
| [Lizpreciatior](#lizpreciatior) | 1 |
| [Lynn](#lynn) | 2 |
| [Mancer](#mancer) | 1 |
| [Mattshumer](#mattshumer) | 1 |
| [Meituan](#meituan) | 2 |
| [Meta](#meta) | 1 |
| [Microsoft](#microsoft) | 6 |
| [Migtissera](#migtissera) | 1 |
| [Mistralai](#mistralai) | 10 |
| [Moonshotai](#moonshotai) | 1 |
| [Morph](#morph) | 3 |
| [Neversleep](#neversleep) | 1 |
| [Nex Agi](#nex-agi) | 2 |
| [Nothingiisreal](#nothingiisreal) | 1 |
| [Nousresearch](#nousresearch) | 5 |
| [Nvidia](#nvidia) | 1 |
| [Open R1](#open-r1) | 1 |
| [Openai](#openai) | 10 |
| [Openchat](#openchat) | 2 |
| [Opengvlab](#opengvlab) | 3 |
| [Openrouter](#openrouter) | 25 |
| [Perceptron](#perceptron) | 1 |
| [Perplexity](#perplexity) | 9 |
| [Poolside](#poolside) | 5 |
| [Prime Intellect](#prime-intellect) | 1 |
| [Pygmalionai](#pygmalionai) | 1 |
| [Raifle](#raifle) | 1 |
| [Recraft](#recraft) | 11 |
| [Recursal](#recursal) | 1 |
| [Rekaai](#rekaai) | 2 |
| [Relace](#relace) | 2 |
| [Rwkv](#rwkv) | 2 |
| [Sakana](#sakana) | 1 |
| [Sao10K](#sao10k) | 7 |
| [Sarvamai](#sarvamai) | 1 |
| [Sentence Transformers](#sentence-transformers) | 5 |
| [Sesame](#sesame) | 1 |
| [Snowflake](#snowflake) | 1 |
| [Sophosympatheia](#sophosympatheia) | 1 |
| [Sourceful](#sourceful) | 7 |
| [Steelskull](#steelskull) | 1 |
| [Stepfun](#stepfun) | 2 |
| [Stepfun Ai](#stepfun-ai) | 1 |
| [Switchpoint](#switchpoint) | 1 |
| [Tencent](#tencent) | 4 |
| [Thedrummer](#thedrummer) | 5 |
| [Thenlper](#thenlper) | 2 |
| [Thinkingmachines](#thinkingmachines) | 1 |
| [Tngtech](#tngtech) | 1 |
| [Togethercomputer](#togethercomputer) | 2 |
| [Undi95](#undi95) | 2 |
| [Upstage](#upstage) | 1 |
| [Writer](#writer) | 1 |
| [Xiaomi](#xiaomi) | 5 |
| [Xwin Lm](#xwin-lm) | 1 |
| [Yi](#yi) | 12 |
| [Zyphra](#zyphra) | 2 |

---

## Claude <a id="claude"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `anthropic/claude-1` | Anthropic: Claude v1 |  |  | 100K | - | T | 0 | 0 |
| `anthropic/claude-1.2` | Anthropic: Claude v1.2 |  |  | 100K | - | T | 0 | 0 |
| `anthropic/claude-2` | Anthropic: Claude v2 |  |  | 200K | - | T | 0 | 0 |
| `anthropic/claude-2.0` | Anthropic: Claude v2.0 |  |  | 100K | - | T | 0 | 0 |
| `anthropic/claude-2.1` | Anthropic: Claude v2.1 |  |  | 200K | - | T | 0 | 0 |
| `anthropic/claude-3-haiku` | Anthropic: Claude 3 Haiku |  |  | 200K | 4.1K | T/I | $0.250 | $1.25 |
| `anthropic/claude-3-opus` | Anthropic: Claude 3 Opus |  |  | 200K | - | T/I | 0 | 0 |
| `anthropic/claude-3-sonnet` | Anthropic: Claude 3 Sonnet |  |  | 200K | - | T/I | 0 | 0 |
| `anthropic/claude-3.5-haiku` | Anthropic: Claude 3.5 Haiku |  |  | 200K | - | T/I | 0 | 0 |
| `anthropic/claude-3.5-haiku-20241022` | Anthropic: Claude 3.5 Haiku (2024-10-22) |  |  | 200K | - | T/I/file | 0 | 0 |
| `anthropic/claude-3.5-sonnet` | Anthropic: Claude 3.5 Sonnet |  |  | 200K | - | T/I/file | 0 | 0 |
| `anthropic/claude-3.5-sonnet-20240620` | Anthropic: Claude 3.5 Sonnet (2024-06-20) |  |  | 200K | - | T/I/file | 0 | 0 |
| `anthropic/claude-3.7-sonnet` | Anthropic: Claude 3.7 Sonnet | 🧠 |  | 200K | - | T/I/file | 0 | 0 |
| `anthropic/claude-fable-5` | Anthropic: Claude Fable 5 | ⚠️ |  | 1M | 128K | T/I/file | $10.00 | $50.00 |
| `anthropic/claude-haiku-4.5` | Anthropic: Claude Haiku 4.5 | 🧠 |  | 200K | 64K | T/I/file | $1.00 | $5.00 |
| `anthropic/claude-instant-1` | Anthropic: Claude Instant v1 |  |  | 100K | - | T | 0 | 0 |
| `anthropic/claude-instant-1.0` | Anthropic: Claude Instant v1.0 |  |  | 100K | - | T | 0 | 0 |
| `anthropic/claude-instant-1.1` | Anthropic: Claude Instant v1.1 |  |  | 100K | - | T | 0 | 0 |
| `anthropic/claude-opus-4` | Anthropic: Claude Opus 4 | 🧠 |  | 200K | 32K | I/T/file | $15.00 | $75.00 |
| `anthropic/claude-opus-4.1` | Anthropic: Claude Opus 4.1 | 🧠 |  | 200K | 32K | I/T/file | $15.00 | $75.00 |
| `anthropic/claude-opus-4.5` | Anthropic: Claude Opus 4.5 | 🧠 |  | 200K | 64K | file/I/T | $5.00 | $25.00 |
| `anthropic/claude-opus-4.6` | Anthropic: Claude Opus 4.6 | 🧠 |  | 1M | 128K | T/I/file | $5.00 | $25.00 |
| `anthropic/claude-opus-4.6-fast` | Anthropic: Claude Opus 4.6 (Fast) | 🧠 |  | 1M | - | T/I/file | 0 | 0 |
| `anthropic/claude-opus-4.7` | Anthropic: Claude Opus 4.7 | 🧠 |  | 1M | 128K | T/I/file | $5.00 | $25.00 |
| `anthropic/claude-opus-4.7-fast` | Anthropic: Claude Opus 4.7 (Fast) | 🧠 |  | 1M | 128K | T/I/file | $30.00 | $150.00 |
| `anthropic/claude-opus-4.8` | Anthropic: Claude Opus 4.8 | 🧠 |  | 1M | 128K | T/I/file | $5.00 | $25.00 |
| `anthropic/claude-opus-4.8-fast` | Anthropic: Claude Opus 4.8 (Fast) | 🧠 |  | 1M | 128K | T/I/file | $10.00 | $50.00 |
| `anthropic/claude-sonnet-4` | Anthropic: Claude Sonnet 4 | 🧠 |  | 1M | 64K | I/T/file | $3.00 | $15.00 |
| `anthropic/claude-sonnet-4.5` | Anthropic: Claude Sonnet 4.5 | 🧠 |  | 1M | 64K | T/I/file | $3.00 | $15.00 |
| `anthropic/claude-sonnet-4.6` | Anthropic: Claude Sonnet 4.6 | 🧠 |  | 1M | 128K | T/I/file | $3.00 | $15.00 |
| `anthropic/claude-sonnet-5` | Anthropic: Claude Sonnet 5 | 🧠 |  | 1M | 128K | T/I/file | $2.00 | $10.00 |
| `~anthropic/claude-fable-latest` | Anthropic: Claude Fable Latest | ⚠️ |  | 1M | 128K | T/I/file | $10.00 | $50.00 |
| `~anthropic/claude-haiku-latest` | Anthropic Claude Haiku Latest | 🧠 |  | 200K | 64K | T/I/file | $1.00 | $5.00 |
| `~anthropic/claude-opus-latest` | Anthropic: Claude Opus Latest | 🧠 |  | 1M | 128K | T/I/file | $5.00 | $25.00 |
| `~anthropic/claude-sonnet-latest` | Anthropic Claude Sonnet Latest | 🧠 |  | 1M | 128K | T/I/file | $2.00 | $10.00 |

---

## Gpt <a id="gpt"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `openai/chatgpt-4o-latest` | OpenAI: ChatGPT-4o |  |  | 128K | - | T/I | 0 | 0 |
| `openai/gpt-3.5-turbo` | OpenAI: GPT-3.5 Turbo |  |  | 16.4K | 4.1K | T | $0.500 | $1.50 |
| `openai/gpt-3.5-turbo-0125` | OpenAI: GPT-3.5 Turbo 16k |  |  | 16.4K | - | T | 0 | 0 |
| `openai/gpt-3.5-turbo-0301` | OpenAI: GPT-3.5 Turbo (older v0301) |  |  | 4.1K | - | T | 0 | 0 |
| `openai/gpt-3.5-turbo-0613` | OpenAI: GPT-3.5 Turbo (older v0613) |  |  | 4.1K | 4.1K | T | $1.00 | $2.00 |
| `openai/gpt-3.5-turbo-1106` | OpenAI: GPT-3.5 Turbo 16k (older v1106) |  |  | 16.4K | - | T | 0 | 0 |
| `openai/gpt-3.5-turbo-16k` | OpenAI: GPT-3.5 Turbo 16k |  |  | 16.4K | 4.1K | T | $3.00 | $4.00 |
| `openai/gpt-3.5-turbo-instruct` | OpenAI: GPT-3.5 Turbo Instruct |  |  | 4.1K | 4.1K | T | $1.50 | $2.00 |
| `openai/gpt-4` | OpenAI: GPT-4 |  |  | 8.2K | 4.1K | T | $30.00 | $60.00 |
| `openai/gpt-4-0314` | OpenAI: GPT-4 (older v0314) |  |  | 8.2K | - | T | 0 | 0 |
| `openai/gpt-4-1106-preview` | OpenAI: GPT-4 Turbo (older v1106) |  |  | 128K | - | T | 0 | 0 |
| `openai/gpt-4-32k` | OpenAI: GPT-4 32k |  |  | 32.8K | - | T | 0 | 0 |
| `openai/gpt-4-32k-0314` | OpenAI: GPT-4 32k (older v0314) |  |  | 32.8K | - | T | 0 | 0 |
| `openai/gpt-4-turbo` | OpenAI: GPT-4 Turbo |  |  | 128K | 4.1K | T/I | $10.00 | $30.00 |
| `openai/gpt-4-turbo-preview` | OpenAI: GPT-4 Turbo Preview |  |  | 128K | 4.1K | T | $10.00 | $30.00 |
| `openai/gpt-4-vision-preview` | OpenAI: GPT-4 Vision |  |  | 128K | - | T/I | 0 | 0 |
| `openai/gpt-4.1` | OpenAI: GPT-4.1 |  |  | 1.0M | - | I/T/file | $2.00 | $8.00 |
| `openai/gpt-4.1-mini` | OpenAI: GPT-4.1 Mini |  |  | 1.0M | 32.8K | I/T/file | $0.400 | $1.60 |
| `openai/gpt-4.1-nano` | OpenAI: GPT-4.1 Nano |  |  | 1.0M | 32.8K | I/T/file | $0.100 | $0.400 |
| `openai/gpt-4.5-preview` | OpenAI: GPT-4.5 (Preview) |  |  | 128K | - | T/I | 0 | 0 |
| `openai/gpt-4o` | OpenAI: GPT-4o |  |  | 128K | 16.4K | T/I/file | $2.50 | $10.00 |
| `openai/gpt-4o-2024-05-13` | OpenAI: GPT-4o (2024-05-13) |  |  | 128K | 4.1K | T/I/file | $5.00 | $15.00 |
| `openai/gpt-4o-2024-08-06` | OpenAI: GPT-4o (2024-08-06) |  |  | 128K | 16.4K | T/I/file | $2.50 | $10.00 |
| `openai/gpt-4o-2024-11-20` | OpenAI: GPT-4o (2024-11-20) |  |  | 128K | 16.4K | T/I/file | $2.50 | $10.00 |
| `openai/gpt-4o-mini` | OpenAI: GPT-4o-mini |  |  | 128K | 16.4K | T/I/file | $0.150 | $0.600 |
| `openai/gpt-4o-mini-2024-07-18` | OpenAI: GPT-4o-mini (2024-07-18) |  |  | 128K | 16.4K | T/I/file | $0.150 | $0.600 |
| `openai/gpt-4o-mini-search-preview` | OpenAI: GPT-4o-mini Search Preview |  |  | 128K | 16.4K | T | $0.150 | $0.600 |
| `openai/gpt-4o-mini-transcribe` | OpenAI: GPT-4o Mini Transcribe |  |  | 128K | - | A | $1.25 | $5.00 |
| `openai/gpt-4o-search-preview` | OpenAI: GPT-4o Search Preview |  |  | 128K | 16.4K | T | $2.50 | $10.00 |
| `openai/gpt-4o-transcribe` | OpenAI: GPT-4o Transcribe |  |  | 128K | - | A | $2.50 | $10.00 |
| `openai/gpt-5` | OpenAI: GPT-5 | ⚠️ |  | 400K | 128K | T/I/file | $1.25 | $10.00 |
| `openai/gpt-5-chat` | OpenAI: GPT-5 Chat |  |  | 128K | 16.4K | file/I/T | $1.25 | $10.00 |
| `openai/gpt-5-codex` | OpenAI: GPT-5 Codex | ⚠️ |  | 400K | 128K | T/I | $1.25 | $10.00 |
| `openai/gpt-5-image` | OpenAI: GPT-5 Image | ⚠️ |  | 400K | 128K | I/T/file | $10.00 | $10.00 |
| `openai/gpt-5-image-mini` | OpenAI: GPT-5 Image Mini | ⚠️ |  | 400K | 128K | file/I/T | $2.50 | $2.00 |
| `openai/gpt-5-mini` | OpenAI: GPT-5 Mini | ⚠️ |  | 400K | 128K | T/I/file | $0.250 | $2.00 |
| `openai/gpt-5-nano` | OpenAI: GPT-5 Nano | ⚠️ |  | 400K | 128K | T/I/file | $0.050 | $0.400 |
| `openai/gpt-5-pro` | OpenAI: GPT-5 Pro | ⚠️ |  | 400K | 128K | I/T/file | $15.00 | $120.00 |
| `openai/gpt-5.1` | OpenAI: GPT-5.1 | 🧠 |  | 400K | 128K | I/T/file | $1.25 | $10.00 |
| `openai/gpt-5.1-chat` | OpenAI: GPT-5.1 Chat |  |  | 128K | 32K | file/I/T | $1.25 | $10.00 |
| `openai/gpt-5.1-codex` | OpenAI: GPT-5.1-Codex | ⚠️ |  | 400K | 128K | T/I | $1.25 | $10.00 |
| `openai/gpt-5.1-codex-max` | OpenAI: GPT-5.1-Codex-Max | ⚠️ |  | 400K | 128K | T/I | $1.25 | $10.00 |
| `openai/gpt-5.1-codex-mini` | OpenAI: GPT-5.1-Codex-Mini | 🧠 |  | 400K | 100K | I/T | $0.250 | $2.00 |
| `openai/gpt-5.2` | OpenAI: GPT-5.2 | 🧠 |  | 400K | 128K | file/I/T | $1.75 | $14.00 |
| `openai/gpt-5.2-chat` | OpenAI: GPT-5.2 Chat |  |  | 128K | 16.4K | file/I/T | $1.75 | $14.00 |
| `openai/gpt-5.2-codex` | OpenAI: GPT-5.2-Codex | ⚠️ |  | 400K | 128K | T/I | $1.75 | $14.00 |
| `openai/gpt-5.2-pro` | OpenAI: GPT-5.2 Pro | ⚠️ |  | 400K | 128K | I/T/file | $21.00 | $168.00 |
| `openai/gpt-5.3-chat` | OpenAI: GPT-5.3 Chat |  |  | 128K | 16.4K | T/I/file | $1.75 | $14.00 |
| `openai/gpt-5.3-codex` | OpenAI: GPT-5.3-Codex | 🧠 |  | 400K | 128K | T/I/file | $1.75 | $14.00 |
| `openai/gpt-5.4` | OpenAI: GPT-5.4 | 🧠 |  | 1.1M | 128K | T/I/file | $2.50 | $15.00 |
| `openai/gpt-5.4-image-2` | OpenAI: GPT-5.4 Image 2 | 🧠 |  | 272K | 128K | I/T/file | $8.00 | $15.00 |
| `openai/gpt-5.4-mini` | OpenAI: GPT-5.4 Mini | 🧠 |  | 400K | 128K | file/I/T | $0.750 | $4.50 |
| `openai/gpt-5.4-nano` | OpenAI: GPT-5.4 Nano | 🧠 |  | 400K | 128K | file/I/T | $0.200 | $1.25 |
| `openai/gpt-5.4-pro` | OpenAI: GPT-5.4 Pro | ⚠️ |  | 1.1M | 128K | T/I/file | $30.00 | $180.00 |
| `openai/gpt-5.5` | OpenAI: GPT-5.5 | 🧠 |  | 1.1M | 128K | file/I/T | $5.00 | $30.00 |
| `openai/gpt-5.5-pro` | OpenAI: GPT-5.5 Pro | ⚠️ |  | 1.1M | 128K | file/I/T | $30.00 | $180.00 |
| `openai/gpt-5.6-luna` | OpenAI: GPT-5.6 Luna | 🧠 |  | 1.1M | 128K | file/I/T | $1.00 | $6.00 |
| `openai/gpt-5.6-luna-pro` | OpenAI: GPT-5.6 Luna Pro | 🧠 |  | 1.1M | 128K | file/I/T | $1.00 | $6.00 |
| `openai/gpt-5.6-sol` | OpenAI: GPT-5.6 Sol | 🧠 |  | 1.1M | 128K | file/I/T | $5.00 | $30.00 |
| `openai/gpt-5.6-sol-pro` | OpenAI: GPT-5.6 Sol Pro | 🧠 |  | 1.1M | 128K | file/I/T | $5.00 | $30.00 |
| `openai/gpt-5.6-terra` | OpenAI: GPT-5.6 Terra | 🧠 |  | 1.1M | 128K | file/I/T | $2.50 | $15.00 |
| `openai/gpt-5.6-terra-pro` | OpenAI: GPT-5.6 Terra Pro | 🧠 |  | 1.1M | 128K | file/I/T | $2.50 | $15.00 |
| `openai/gpt-audio` | OpenAI: GPT Audio |  |  | 128K | 16.4K | T/A | $2.50 | $10.00 |
| `openai/gpt-audio-mini` | OpenAI: GPT Audio Mini |  |  | 128K | 16.4K | T/A | $0.600 | $2.40 |
| `openai/gpt-chat-latest` | OpenAI: GPT Chat Latest |  |  | 400K | 128K | T/I/file | $5.00 | $30.00 |
| `openai/gpt-image-1` | OpenAI: GPT Image 1 |  |  | 400K | - | T/I | $10.00 | $10.00 |
| `openai/gpt-image-1-mini` | OpenAI: GPT Image 1 Mini |  |  | 400K | - | T/I | $2.50 | $2.50 |
| `openai/gpt-image-2` | OpenAI: GPT Image 2 |  |  | 400K | - | T/I | $8.00 | $8.00 |
| `openai/gpt-oss-120b` | OpenAI: gpt-oss-120b | ⚠️ |  | 131.1K | - | T | $0.030 | $0.180 |
| `openai/gpt-oss-20b` | OpenAI: gpt-oss-20b (free) | ⚠️ | 🆓 | 131.1K | 32.8K | T | 0 | 0 |
| `openai/gpt-oss-20b` | OpenAI: gpt-oss-20b | ⚠️ |  | 131.1K | - | T | $0.029 | $0.140 |
| `openai/gpt-oss-safeguard-20b` | OpenAI: gpt-oss-safeguard-20b | ⚠️ |  | 131.1K | 65.5K | T | $0.075 | $0.300 |
| `openai/o1-mini` | OpenAI: o1-mini | 🧠 |  | 128K | - | T | 0 | 0 |
| `openai/o1-mini-2024-09-12` | OpenAI: o1-mini (2024-09-12) |  |  | 128K | - | T | 0 | 0 |
| `openai/o1-preview` | OpenAI: o1-preview |  |  | 128K | - | T | 0 | 0 |
| `openai/o1-preview-2024-09-12` | OpenAI: o1-preview (2024-09-12) |  |  | 128K | - | T | 0 | 0 |
| `openai/o1-pro` | OpenAI: o1-pro | 🧠 |  | 200K | 100K | T/I/file | $150.00 | $600.00 |
| `openai/o3-deep-research` | OpenAI: o3 Deep Research | 🧠 |  | 200K | 100K | I/T/file | $10.00 | $40.00 |
| `openai/o3-mini` | OpenAI: o3 Mini | 🧠 |  | 200K | 100K | T/file | $1.10 | $4.40 |
| `openai/o3-mini-high` | OpenAI: o3 Mini High | ⚠️ |  | 200K | 100K | T/file | $1.10 | $4.40 |
| `openai/o3-pro` | OpenAI: o3 Pro | 🧠 |  | 200K | 100K | T/file/I | $20.00 | $80.00 |
| `openai/o4-mini` | OpenAI: o4 Mini | 🧠 |  | 200K | 100K | I/T/file | $1.10 | $4.40 |
| `openai/o4-mini-deep-research` | OpenAI: o4 Mini Deep Research | 🧠 |  | 200K | 100K | file/I/T | $2.00 | $8.00 |
| `openai/o4-mini-high` | OpenAI: o4 Mini High | ⚠️ |  | 200K | 100K | I/T/file | $1.10 | $4.40 |
| `~openai/gpt-latest` | OpenAI GPT Latest | 🧠 |  | 1.1M | 128K | file/I/T | $5.00 | $30.00 |
| `~openai/gpt-mini-latest` | OpenAI GPT Mini Latest | 🧠 |  | 400K | 128K | file/I/T | $0.750 | $4.50 |

---

## Gemini <a id="gemini"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `google/gemini-2.0-flash-001` | Google: Gemini 2.0 Flash |  |  | 1M | - | T/I/file/A/V | 0 | 0 |
| `google/gemini-2.0-flash-exp` | Google: Gemini 2.0 Flash Experimental |  |  | 1.0M | - | T/I | 0 | 0 |
| `google/gemini-2.0-flash-lite-001` | Google: Gemini 2.0 Flash Lite |  |  | 1.0M | - | T/I/file/A/V | 0 | 0 |
| `google/gemini-2.5-flash` | Google: Gemini 2.5 Flash | 🧠 |  | 1.0M | 65.5K | file/I/T/A/V | $0.300 | $2.50 |
| `google/gemini-2.5-flash-image` | Google: Nano Banana (Gemini 2.5 Flash Image) |  |  | 32.8K | 32.8K | I/T | $0.300 | $2.50 |
| `google/gemini-2.5-flash-image-preview` | Google: Gemini 2.5 Flash Image Preview (Nano Ba... |  |  | 32.8K | - | I/T | 0 | 0 |
| `google/gemini-2.5-flash-lite` | Google: Gemini 2.5 Flash Lite | 🧠 |  | 1.0M | 65.5K | T/I/file/A/V | $0.100 | $0.400 |
| `google/gemini-2.5-flash-lite-preview-09-2025` | Google: Gemini 2.5 Flash Lite Preview 09-2025 | 🧠 |  | 1.0M | - | T/I/file/A/V | 0 | 0 |
| `google/gemini-2.5-flash-preview-09-2025` | Google: Gemini 2.5 Flash Preview 09-2025 | 🧠 |  | 1.0M | - | I/file/T/A/V | 0 | 0 |
| `google/gemini-2.5-pro` | Google: Gemini 2.5 Pro | ⚠️ |  | 1.0M | 65.5K | T/I/file/A/V | $1.25 | $10.00 |
| `google/gemini-2.5-pro-exp-03-25` | Google: Gemini 2.5 Pro Experimental |  |  | 1.0M | - | T/I/file | 0 | 0 |
| `google/gemini-2.5-pro-preview` | Google: Gemini 2.5 Pro Preview 06-05 | ⚠️ |  | 1.0M | 65.5K | file/I/T/A | $1.25 | $10.00 |
| `google/gemini-2.5-pro-preview-05-06` | Google: Gemini 2.5 Pro Preview 05-06 | ⚠️ |  | 1.0M | 65.5K | T/I/file/A/V | $1.25 | $10.00 |
| `google/gemini-3-flash-preview` | Google: Gemini 3 Flash Preview | 🧠 |  | 1.0M | 65.5K | T/I/file/A/V | $0.500 | $3.00 |
| `google/gemini-3-pro-image` | Google: Nano Banana Pro (Gemini 3 Pro Image) | ⚠️ |  | 65.5K | 32.8K | I/T | $2.00 | $12.00 |
| `google/gemini-3-pro-image-preview` | Google: Nano Banana Pro (Gemini 3 Pro Image Pre... | ⚠️ |  | 65.5K | 32.8K | I/T | $2.00 | $12.00 |
| `google/gemini-3-pro-preview` | Google: Gemini 3 Pro Preview | ⚠️ |  | 1.0M | - | T/I/file/A/V | 0 | 0 |
| `google/gemini-3.1-flash-image` | Google: Nano Banana 2 (Gemini 3.1 Flash Image) | 🧠 |  | 131.1K | 32.8K | I/T | $0.500 | $3.00 |
| `google/gemini-3.1-flash-image-preview` | Google: Nano Banana 2 (Gemini 3.1 Flash Image P... | 🧠 |  | 131.1K | 32.8K | I/T | $0.500 | $3.00 |
| `google/gemini-3.1-flash-lite` | Google: Gemini 3.1 Flash Lite | 🧠 |  | 1.0M | 65.5K | T/I/V/file/A | $0.250 | $1.50 |
| `google/gemini-3.1-flash-lite-image` | Google: Nano Banana 2 Lite (Gemini 3.1 Flash Li... | 🧠 |  | 65.5K | 66K | I/T | $0.250 | $1.50 |
| `google/gemini-3.1-flash-lite-preview` | Google: Gemini 3.1 Flash Lite Preview | 🧠 |  | 1.0M | 65.5K | T/I/V/file/A | $0.250 | $1.50 |
| `google/gemini-3.1-flash-tts-preview` | Google: Gemini 3.1 Flash TTS Preview |  |  | 32.8K | 16.4K | T | $1.00 | $20.00 |
| `google/gemini-3.1-pro-preview` | Google: Gemini 3.1 Pro Preview | ⚠️ |  | 1.0M | 65.5K | A/file/I/T/V | $2.00 | $12.00 |
| `google/gemini-3.1-pro-preview-customtools` | Google: Gemini 3.1 Pro Preview Custom Tools | ⚠️ |  | 1.0M | 65.5K | T/A/I/V/file | $2.00 | $12.00 |
| `google/gemini-3.5-flash` | Google: Gemini 3.5 Flash | ⚠️ |  | 1.0M | 65.5K | T/I/V/file/A | $1.50 | $9.00 |
| `google/gemini-embedding-001` | Google: Gemini Embedding 001 |  |  | 20K | - | T | $0.150 | 0 |
| `google/gemini-embedding-2` | Google: Gemini Embedding 2 |  |  | 8.2K | - | T/I/file/A/V | $0.200 | 0 |
| `google/gemini-embedding-2-preview` | Google: Gemini Embedding 2 Preview |  |  | 8.2K | - | T/I/file/A/V | $0.200 | 0 |
| `google/gemini-exp-1114` | Google: Gemini Experimental 1114 |  |  | 41.0K | - | T/I | 0 | 0 |
| `google/gemini-exp-1121` | Google: Gemini Experimental 1121 |  |  | 41.0K | - | T/I | 0 | 0 |
| `google/gemini-flash-1.5` | Google: Gemini 1.5 Flash  |  |  | 1M | - | T/I | 0 | 0 |
| `google/gemini-flash-1.5-8b` | Google: Gemini 1.5 Flash 8B |  |  | 1M | - | T/I | 0 | 0 |
| `google/gemini-flash-1.5-exp` | Google: Gemini 1.5 Flash Experimental |  |  | 1M | - | T/I | 0 | 0 |
| `google/gemini-pro-1.5` | Google: Gemini 1.5 Pro |  |  | 2M | - | T/I | 0 | 0 |
| `google/gemini-pro-1.5-exp` | Google: Gemini 1.5 Pro Experimental |  |  | 1M | - | T/I | 0 | 0 |
| `~google/gemini-flash-latest` | Google Gemini Flash Latest | ⚠️ |  | 1.0M | 65.5K | T/I/V/file/A | $1.50 | $9.00 |
| `~google/gemini-pro-latest` | Google Gemini Pro Latest | ⚠️ |  | 1.0M | 65.5K | A/file/I/T/V | $2.00 | $12.00 |

---

## Gemma <a id="gemma"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `google/gemma-2-27b-it` | Google: Gemma 2 27B |  |  | 8.2K | 2.0K | T | $0.650 | $0.650 |
| `google/gemma-2-9b-it` | Google: Gemma 2 9B |  |  | 8.2K | - | T | 0 | 0 |
| `google/gemma-2b-it` | Google: Gemma 1 2B |  |  | 8.2K | - | T | 0 | 0 |
| `google/gemma-3-12b-it` | Google: Gemma 3 12B |  |  | 131.1K | 16.4K | T/I | $0.050 | $0.150 |
| `google/gemma-3-1b-it` | Google: Gemma 3 1B |  |  | 32K | - | T/I | 0 | 0 |
| `google/gemma-3-27b-it` | Google: Gemma 3 27B |  |  | 131.1K | 16.4K | T/I | $0.080 | $0.160 |
| `google/gemma-3-4b-it` | Google: Gemma 3 4B |  |  | 131.1K | 16.4K | T/I | $0.050 | $0.100 |
| `google/gemma-3n-e2b-it` | Google: Gemma 3n 2B |  |  | 8.2K | - | T | 0 | 0 |
| `google/gemma-3n-e4b-it` | Google: Gemma 3n 4B |  |  | 32.8K | - | T | $0.060 | $0.120 |
| `google/gemma-4-26b-a4b-it` | Google: Gemma 4 26B A4B  (free) | 🧠 | 🆓 | 262.1K | 32.8K | I/T/V | 0 | 0 |
| `google/gemma-4-26b-a4b-it` | Google: Gemma 4 26B A4B  | 🧠 |  | 262.1K | - | I/T/V | $0.060 | $0.330 |
| `google/gemma-4-31b-it` | Google: Gemma 4 31B (free) | 🧠 | 🆓 | 262.1K | 32.8K | I/T/V | 0 | 0 |
| `google/gemma-4-31b-it` | Google: Gemma 4 31B | 🧠 |  | 262.1K | 8.2K | I/T/V | $0.100 | $0.350 |
| `google/gemma-7b-it` | Google: Gemma 7B |  |  | 8.2K | - | T | 0 | 0 |

---

## Llama <a id="llama"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `aion-labs/aion-rp-llama-3.1-8b` | AionLabs: Aion-RP 1.0 (8B) |  |  | 32.8K | 32.8K | T | $0.800 | $1.60 |
| `alfredpros/codellama-7b-instruct-solidity` | AlfredPros: CodeLLaMa 7B Instruct Solidity |  |  | 4.1K | - | T | 0 | 0 |
| `allenai/llama-3.1-tulu-3-405b` | Llama 3.1 Tulu 3 405B |  |  | 0 | - | T | 0 | 0 |
| `cognitivecomputations/dolphin-llama-3-70b` | Dolphin Llama 3 70B 🐬 |  |  | 8.2K | - | T | 0 | 0 |
| `deepcogito/cogito-v2-preview-llama-109b-moe` | Cogito V2 Preview Llama 109B | 🧠 |  | 131.1K | - | I/T | 0 | 0 |
| `deepcogito/cogito-v2-preview-llama-405b` | Deep Cogito: Cogito V2 Preview Llama 405B | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `deepcogito/cogito-v2-preview-llama-70b` | Deep Cogito: Cogito V2 Preview Llama 70B | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `deepseek/deepseek-r1-distill-llama-70b` | DeepSeek: R1 Distill Llama 70B | 🧠 |  | 128K | 8.2K | T | $0.800 | $0.800 |
| `deepseek/deepseek-r1-distill-llama-8b` | DeepSeek: R1 Distill Llama 8B | 🧠 |  | 0 | - | T | 0 | 0 |
| `eva-unit-01/eva-llama-3.33-70b` | EVA Llama 3.33 70B |  |  | 16.4K | - | T | 0 | 0 |
| `latitudegames/wayfarer-large-70b-llama-3.3` | LatitudeGames: Wayfarer Large 70B Llama 3.3 |  |  | 128K | - | T | 0 | 0 |
| `meta-llama/codellama-34b-instruct` | Meta: CodeLlama 34B Instruct |  |  | 8.2K | - | T | 0 | 0 |
| `meta-llama/codellama-70b-instruct` | Meta: CodeLlama 70B Instruct |  |  | 2.0K | - | T | 0 | 0 |
| `meta-llama/llama-2-13b-chat` | Meta: Llama 2 13B Chat |  |  | 4.1K | - | T | 0 | 0 |
| `meta-llama/llama-2-70b-chat` | Meta: Llama 2 70B Chat |  |  | 4.1K | - | T | 0 | 0 |
| `meta-llama/llama-3-70b` | Meta: Llama 3 70B (Base) |  |  | 8.2K | - | T | 0 | 0 |
| `meta-llama/llama-3-70b-instruct` | Meta: Llama 3 70B Instruct |  |  | 8.2K | - | T | 0 | 0 |
| `meta-llama/llama-3-8b` | Meta: Llama 3 8B (Base) |  |  | 8.2K | - | T | 0 | 0 |
| `meta-llama/llama-3-8b-instruct` | Meta: Llama 3 8B Instruct |  |  | 8.2K | - | T | 0 | 0 |
| `meta-llama/llama-3.1-405b` | Meta: Llama 3.1 405B (base) |  |  | 131.1K | - | T | 0 | 0 |
| `meta-llama/llama-3.1-405b-instruct` | Meta: Llama 3.1 405B Instruct |  |  | 131.1K | - | T | 0 | 0 |
| `meta-llama/llama-3.1-70b-instruct` | Meta: Llama 3.1 70B Instruct |  |  | 131.1K | 16.4K | T | $0.400 | $0.400 |
| `meta-llama/llama-3.1-8b-instruct` | Meta: Llama 3.1 8B Instruct |  |  | 131.1K | 16.4K | T | $0.020 | $0.030 |
| `meta-llama/llama-3.2-11b-vision-instruct` | Meta: Llama 3.2 11B Vision Instruct |  |  | 131.1K | - | T/I | 0 | 0 |
| `meta-llama/llama-3.2-1b-instruct` | Meta: Llama 3.2 1B Instruct |  |  | 131.1K | 60K | T | $0.027 | $0.201 |
| `meta-llama/llama-3.2-3b-instruct` | Meta: Llama 3.2 3B Instruct |  |  | 131.1K | 131.1K | T | $0.050 | $0.330 |
| `meta-llama/llama-3.2-90b-vision-instruct` | Meta: Llama 3.2 90B Vision Instruct |  |  | 131.1K | - | T/I | 0 | 0 |
| `meta-llama/llama-3.3-70b-instruct` | Meta: Llama 3.3 70B Instruct |  |  | 131.1K | 16.4K | T | $0.100 | $0.320 |
| `meta-llama/llama-3.3-8b-instruct` | Meta: Llama 3.3 8B Instruct |  |  | 128K | - | T | 0 | 0 |
| `meta-llama/llama-4-maverick` | Meta: Llama 4 Maverick |  |  | 1.0M | 16.4K | T/I | $0.200 | $0.800 |
| `meta-llama/llama-4-scout` | Meta: Llama 4 Scout |  |  | 10M | 16.4K | T/I | $0.100 | $0.300 |
| `meta-llama/llama-guard-2-8b` | Meta: LlamaGuard 2 8B |  |  | 8.2K | - | T | 0 | 0 |
| `meta-llama/llama-guard-3-8b` | Llama Guard 3 8B |  |  | 0 | - | T | 0 | 0 |
| `meta-llama/llama-guard-4-12b` | Meta: Llama Guard 4 12B |  |  | 163.8K | 16.4K | I/T | $0.180 | $0.180 |
| `neversleep/llama-3-lumimaid-70b` | NeverSleep: Llama 3 Lumimaid 70B |  |  | 8.2K | - | T | 0 | 0 |
| `neversleep/llama-3-lumimaid-8b` | NeverSleep: Llama 3 Lumimaid 8B |  |  | 24.6K | - | T | 0 | 0 |
| `neversleep/llama-3.1-lumimaid-70b` | NeverSleep: Lumimaid v0.2 70B |  |  | 131.1K | - | T | 0 | 0 |
| `neversleep/llama-3.1-lumimaid-8b` | NeverSleep: Lumimaid v0.2 8B |  |  | 131.1K | - | T | 0 | 0 |
| `nousresearch/deephermes-3-llama-3-8b-preview` | Nous: DeepHermes 3 Llama 3 8B Preview |  |  | 131.1K | - | T | 0 | 0 |
| `nousresearch/hermes-2-pro-llama-3-8b` | NousResearch: Hermes 2 Pro - Llama-3 8B |  |  | 8.2K | - | T | 0 | 0 |
| `nousresearch/hermes-2-theta-llama-3-8b` | Nous: Hermes 2 Theta 8B |  |  | 16.4K | - | T | 0 | 0 |
| `nousresearch/hermes-3-llama-3.1-405b` | Nous: Hermes 3 405B Instruct |  |  | 131.1K | 16.4K | T | $1.00 | $1.00 |
| `nousresearch/hermes-3-llama-3.1-70b` | Nous: Hermes 3 70B Instruct |  |  | 131.1K | 16.4K | T | $0.700 | $0.700 |
| `nousresearch/nous-hermes-llama2-13b` | Nous: Hermes 13B |  |  | 4.1K | - | T | 0 | 0 |
| `nousresearch/nous-hermes-llama2-70b` | Nous: Hermes 70B |  |  | 4.1K | - | T | 0 | 0 |
| `nvidia/llama-3.1-nemotron-70b-instruct` | NVIDIA: Llama 3.1 Nemotron 70B Instruct |  |  | 131.1K | - | T | 0 | 0 |
| `nvidia/llama-3.1-nemotron-nano-8b-v1` | NVIDIA: Llama 3.1 Nemotron Nano 8B v1 |  |  | 131.1K | - | T | 0 | 0 |
| `nvidia/llama-3.1-nemotron-ultra-253b-v1` | NVIDIA: Llama 3.1 Nemotron Ultra 253B v1 | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `nvidia/llama-3.3-nemotron-super-49b-v1` | NVIDIA: Llama 3.3 Nemotron Super 49B v1 |  |  | 131.1K | - | T | 0 | 0 |
| `nvidia/llama-3.3-nemotron-super-49b-v1.5` | NVIDIA: Llama 3.3 Nemotron Super 49B V1.5 | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `nvidia/llama-nemotron-embed-vl-1b-v2` | NVIDIA: Llama Nemotron Embed VL 1B V2 (free) |  | 🆓 | 131.1K | - | T/I | 0 | 0 |
| `nvidia/llama-nemotron-rerank-vl-1b-v2` | NVIDIA: Llama Nemotron Rerank VL 1B V2 (free) |  | 🆓 | 10.2K | - | T/I | 0 | 0 |
| `perplexity/llama-3-sonar-large-32k-chat` | Perplexity: Llama3 Sonar 70B |  |  | 32.8K | - | T | 0 | 0 |
| `perplexity/llama-3-sonar-large-32k-online` | Perplexity: Llama3 Sonar 70B Online |  |  | 28K | - | T | 0 | 0 |
| `perplexity/llama-3-sonar-small-32k-chat` | Perplexity: Llama3 Sonar 8B |  |  | 32.8K | - | T | 0 | 0 |
| `perplexity/llama-3-sonar-small-32k-online` | Perplexity: Llama3 Sonar 8B Online |  |  | 28K | - | T | 0 | 0 |
| `perplexity/llama-3.1-sonar-large-128k-online` | Perplexity: Llama 3.1 Sonar 70B Online |  |  | 127.1K | - | T | 0 | 0 |
| `perplexity/llama-3.1-sonar-small-128k-online` | Perplexity: Llama 3.1 Sonar 8B Online |  |  | 127.1K | - | T | 0 | 0 |
| `phind/phind-codellama-34b` | Phind: CodeLlama 34B v2 |  |  | 4.1K | - | T | 0 | 0 |
| `scb10x/llama3.1-typhoon2-70b-instruct` | Typhoon2 70B Instruct |  |  | 8.2K | - | T | 0 | 0 |
| `scb10x/llama3.1-typhoon2-8b-instruct` | Typhoon2 8B Instruct |  |  | 8.2K | - | T | 0 | 0 |
| `sentientagi/dobby-mini-unhinged-plus-llama-3.1-8b` | SentientAGI: Dobby Mini Plus Llama 3.1 8B |  |  | 131.1K | - | T | 0 | 0 |
| `tokyotech-llm/llama-3.1-swallow-8b-instruct-v0.3` | Swallow: Llama 3.1 Swallow 8B Instruct V0.3 |  |  | 16.4K | - | T | 0 | 0 |

---

## Qwen <a id="qwen"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `deepseek/deepseek-r1-0528-qwen3-8b` | DeepSeek: DeepSeek R1 0528 Qwen3 8B | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `deepseek/deepseek-r1-distill-qwen-1.5b` | DeepSeek: R1 Distill Qwen 1.5B | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `deepseek/deepseek-r1-distill-qwen-14b` | DeepSeek: R1 Distill Qwen 14B | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `deepseek/deepseek-r1-distill-qwen-32b` | DeepSeek: R1 Distill Qwen 32B | 🧠 |  | 128K | - | T | 0 | 0 |
| `deepseek/deepseek-r1-distill-qwen-7b` | DeepSeek: R1 Distill Qwen 7B | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `eva-unit-01/eva-qwen-2.5-14b` | EVA Qwen2.5 14B |  |  | 32.8K | - | T | 0 | 0 |
| `eva-unit-01/eva-qwen-2.5-32b` | EVA Qwen2.5 32B |  |  | 32K | - | T | 0 | 0 |
| `eva-unit-01/eva-qwen-2.5-72b` | EVA Qwen2.5 72B |  |  | 32K | - | T | 0 | 0 |
| `qwen/qwen-110b-chat` | Qwen 1.5 110B Chat |  |  | 32.8K | - | T | 0 | 0 |
| `qwen/qwen-14b-chat` | Qwen 1.5 14B Chat |  |  | 32.8K | - | T | 0 | 0 |
| `qwen/qwen-2-72b-instruct` | Qwen 2 72B Instruct |  |  | 32.8K | - | T | 0 | 0 |
| `qwen/qwen-2-7b-instruct` | Qwen 2 7B Instruct |  |  | 32.8K | - | T | 0 | 0 |
| `qwen/qwen-2.5-72b-instruct` | Qwen2.5 72B Instruct |  |  | 131.1K | 16.4K | T | $0.360 | $0.400 |
| `qwen/qwen-2.5-7b-instruct` | Qwen: Qwen2.5 7B Instruct |  |  | 131.1K | 32.8K | T | $0.040 | $0.100 |
| `qwen/qwen-2.5-coder-32b-instruct` | Qwen2.5 Coder 32B Instruct |  |  | 128K | 32.8K | T | $0.660 | $1.00 |
| `qwen/qwen-2.5-vl-7b-instruct` | Qwen: Qwen2.5-VL 7B Instruct |  |  | 32.8K | - | T/I | 0 | 0 |
| `qwen/qwen-32b-chat` | Qwen 1.5 32B Chat |  |  | 32.8K | - | T | 0 | 0 |
| `qwen/qwen-4b-chat` | Qwen 1.5 4B Chat |  |  | 32.8K | - | T | 0 | 0 |
| `qwen/qwen-72b-chat` | Qwen 1.5 72B Chat |  |  | 32.8K | - | T | 0 | 0 |
| `qwen/qwen-7b-chat` | Qwen 1.5 7B Chat |  |  | 32.8K | - | T | 0 | 0 |
| `qwen/qwen-max` | Qwen: Qwen-Max  |  |  | 32.8K | - | T | 0 | 0 |
| `qwen/qwen-plus` | Qwen: Qwen-Plus |  |  | 1M | 32.8K | T | $0.260 | $0.780 |
| `qwen/qwen-plus-2025-07-28` | Qwen: Qwen Plus 0728 (thinking) | 🧠 |  | 1M | 32.8K | T | $0.260 | $0.780 |
| `qwen/qwen-plus-2025-07-28` | Qwen: Qwen Plus 0728 | 🧠 |  | 1M | 32.8K | T | $0.260 | $0.780 |
| `qwen/qwen-turbo` | Qwen: Qwen-Turbo |  |  | 1M | - | T | 0 | 0 |
| `qwen/qwen-vl-max` | Qwen: Qwen VL Max |  |  | 131.1K | - | T/I | 0 | 0 |
| `qwen/qwen-vl-plus` | Qwen: Qwen VL Plus |  |  | 7.5K | - | T/I | 0 | 0 |
| `qwen/qwen2.5-32b-instruct` | Qwen: Qwen2.5 32B Instruct |  |  | 131.1K | - | T | 0 | 0 |
| `qwen/qwen2.5-coder-7b-instruct` | Qwen: Qwen2.5 Coder 7B Instruct |  |  | 131.1K | - | T | 0 | 0 |
| `qwen/qwen2.5-vl-32b-instruct` | Qwen: Qwen2.5 VL 32B Instruct |  |  | 32.8K | - | T/I | 0 | 0 |
| `qwen/qwen2.5-vl-3b-instruct` | Qwen: Qwen2.5 VL 3B Instruct |  |  | 64K | - | T/I | 0 | 0 |
| `qwen/qwen2.5-vl-72b-instruct` | Qwen: Qwen2.5 VL 72B Instruct |  |  | 131.1K | - | T/I | $0.250 | $0.750 |
| `qwen/qwen3-0.6b-04-28` | Qwen: Qwen3 0.6B | 🧠 |  | 32K | - | T | 0 | 0 |
| `qwen/qwen3-1.7b` | Qwen: Qwen3 1.7B | 🧠 |  | 32K | - | T | 0 | 0 |
| `qwen/qwen3-14b` | Qwen: Qwen3 14B | 🧠 |  | 131.7K | 41.0K | T | $0.100 | $0.240 |
| `qwen/qwen3-235b-a22b` | Qwen: Qwen3 235B A22B | 🧠 |  | 131.1K | 8.2K | T | $0.455 | $1.82 |
| `qwen/qwen3-235b-a22b-2507` | Qwen: Qwen3 235B A22B Instruct 2507 |  |  | 262.1K | 16.4K | T | $0.090 | $0.550 |
| `qwen/qwen3-235b-a22b-thinking-2507` | Qwen: Qwen3 235B A22B Thinking 2507 | ⚠️ |  | 262.1K | - | T | $0.150 | $1.50 |
| `qwen/qwen3-30b-a3b` | Qwen: Qwen3 30B A3B | 🧠 |  | 131.1K | 16.4K | T | $0.120 | $0.500 |
| `qwen/qwen3-30b-a3b-instruct-2507` | Qwen: Qwen3 30B A3B Instruct 2507 |  |  | 131.1K | 32K | T | $0.048 | $0.193 |
| `qwen/qwen3-30b-a3b-thinking-2507` | Qwen: Qwen3 30B A3B Thinking 2507 | ⚠️ |  | 131.1K | 32.8K | T | $0.130 | $1.56 |
| `qwen/qwen3-32b` | Qwen: Qwen3 32B | 🧠 |  | 131.1K | 16.4K | T | $0.080 | $0.280 |
| `qwen/qwen3-4b` | Qwen: Qwen3 4B | 🧠 |  | 128K | - | T | 0 | 0 |
| `qwen/qwen3-8b` | Qwen: Qwen3 8B | 🧠 |  | 131.1K | 8.2K | T | $0.117 | $0.455 |
| `qwen/qwen3-asr-flash-2026-02-10` | Qwen: Qwen3 ASR Flash |  |  | 0 | - | A | $35.00 | 0 |
| `qwen/qwen3-coder` | Qwen: Qwen3 Coder 480B A35B |  |  | 1.0M | 65.5K | T | $0.220 | $1.80 |
| `qwen/qwen3-coder-30b-a3b-instruct` | Qwen: Qwen3 Coder 30B A3B Instruct |  |  | 160K | 32.8K | T | $0.070 | $0.270 |
| `qwen/qwen3-coder-flash` | Qwen: Qwen3 Coder Flash |  |  | 1M | 65.5K | T | $0.195 | $0.975 |
| `qwen/qwen3-coder-next` | Qwen: Qwen3 Coder Next |  |  | 262.1K | 262.1K | T | $0.110 | $0.800 |
| `qwen/qwen3-coder-plus` | Qwen: Qwen3 Coder Plus | 🧠 |  | 1M | 65.5K | T | $0.650 | $3.25 |
| `qwen/qwen3-embedding-0.6b` | Qwen: Qwen3 Embedding 0.6B |  |  | 8.2K | - | T | 0 | 0 |
| `qwen/qwen3-embedding-4b` | Qwen: Qwen3 Embedding 4B |  |  | 32.8K | - | T | $0.020 | 0 |
| `qwen/qwen3-embedding-8b` | Qwen: Qwen3 Embedding 8B |  |  | 32K | 32K | T | $0.010 | 0 |
| `qwen/qwen3-max` | Qwen: Qwen3 Max | 🧠 |  | 262.1K | 32.8K | T | $0.780 | $3.90 |
| `qwen/qwen3-max-thinking` | Qwen: Qwen3 Max Thinking | 🧠 |  | 262.1K | 32.8K | T | $0.780 | $3.90 |
| `qwen/qwen3-next-80b-a3b-instruct` | Qwen: Qwen3 Next 80B A3B Instruct |  |  | 262.1K | 16.4K | T | $0.090 | $1.10 |
| `qwen/qwen3-next-80b-a3b-thinking` | Qwen: Qwen3 Next 80B A3B Thinking | ⚠️ |  | 262.1K | 32.8K | T | $0.098 | $0.780 |
| `qwen/qwen3-vl-235b-a22b-instruct` | Qwen: Qwen3 VL 235B A22B Instruct |  |  | 262.1K | 16.4K | T/I | $0.200 | $0.880 |
| `qwen/qwen3-vl-235b-a22b-thinking` | Qwen: Qwen3 VL 235B A22B Thinking | ⚠️ |  | 131.1K | 32.8K | T/I | $0.260 | $2.60 |
| `qwen/qwen3-vl-30b-a3b-instruct` | Qwen: Qwen3 VL 30B A3B Instruct |  |  | 262.1K | 32.8K | T/I | $0.130 | $0.520 |
| `qwen/qwen3-vl-30b-a3b-thinking` | Qwen: Qwen3 VL 30B A3B Thinking | ⚠️ |  | 131.1K | 32.8K | T/I | $0.130 | $1.56 |
| `qwen/qwen3-vl-32b-instruct` | Qwen: Qwen3 VL 32B Instruct |  |  | 262.1K | 32.8K | T/I | $0.104 | $0.416 |
| `qwen/qwen3-vl-8b-instruct` | Qwen: Qwen3 VL 8B Instruct |  |  | 256K | 32.8K | I/T | $0.117 | $0.455 |
| `qwen/qwen3-vl-8b-thinking` | Qwen: Qwen3 VL 8B Thinking | ⚠️ |  | 256K | 32.8K | I/T | $0.117 | $1.36 |
| `qwen/qwen3.5-122b-a10b` | Qwen: Qwen3.5-122B-A10B | 🧠 |  | 262.1K | 262.1K | T/I/V | $0.260 | $2.08 |
| `qwen/qwen3.5-27b` | Qwen: Qwen3.5-27B | 🧠 |  | 262.1K | 65.5K | T/I/V | $0.195 | $1.56 |
| `qwen/qwen3.5-35b-a3b` | Qwen: Qwen3.5-35B-A3B | 🧠 |  | 262.1K | 81.9K | T/I/V | $0.140 | $1.00 |
| `qwen/qwen3.5-397b-a17b` | Qwen: Qwen3.5 397B A17B | 🧠 |  | 256K | - | T/I/V | $0.385 | $2.45 |
| `qwen/qwen3.5-9b` | Qwen: Qwen3.5-9B | 🧠 |  | 262.1K | 262.1K | T/I/V | $0.100 | $0.150 |
| `qwen/qwen3.5-flash-02-23` | Qwen: Qwen3.5-Flash | 🧠 |  | 1M | 65.5K | T/I/V | $0.065 | $0.260 |
| `qwen/qwen3.5-plus-02-15` | Qwen: Qwen3.5 Plus 2026-02-15 | 🧠 |  | 1M | 65.5K | T/I/V | $0.260 | $1.56 |
| `qwen/qwen3.5-plus-20260420` | Qwen: Qwen3.5 Plus 2026-04-20 | 🧠 |  | 1M | 65.5K | T/I/V | $0.300 | $1.80 |
| `qwen/qwen3.6-27b` | Qwen: Qwen3.6 27B | 🧠 |  | 262.1K | 131.1K | T/I/V | $0.289 | $2.40 |
| `qwen/qwen3.6-35b-a3b` | Qwen: Qwen3.6 35B A3B | 🧠 |  | 262.1K | - | T/I/V | $0.130 | $1.00 |
| `qwen/qwen3.6-flash` | Qwen: Qwen3.6 Flash | 🧠 |  | 1M | 65.5K | T/I/V | $0.188 | $1.12 |
| `qwen/qwen3.6-max-preview` | Qwen: Qwen3.6 Max Preview | 🧠 |  | 262.1K | 65.5K | T | $1.04 | $6.24 |
| `qwen/qwen3.6-plus` | Qwen: Qwen3.6 Plus | 🧠 |  | 1M | 65.5K | T/I/V | $0.325 | $1.95 |
| `qwen/qwen3.6-plus-preview` | Qwen: Qwen3.6 Plus Preview | ⚠️ |  | 1M | - | T | 0 | 0 |
| `qwen/qwen3.7-max` | Qwen: Qwen3.7 Max | 🧠 |  | 1M | 65.5K | T | $1.48 | $4.42 |
| `qwen/qwen3.7-plus` | Qwen: Qwen3.7 Plus | 🧠 |  | 1M | 65.5K | T/I | $0.320 | $1.28 |
| `qwen/qwq-32b` | Qwen: QwQ 32B | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `qwen/qwq-32b-preview` | Qwen: QwQ 32B Preview | 🧠 |  | 32.8K | - | T | 0 | 0 |

---

## Deepseek <a id="deepseek"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `deepcogito/cogito-v2-preview-deepseek-671b` | Deep Cogito: Cogito V2 Preview Deepseek 671B | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `deepseek/deepseek-chat` | DeepSeek: DeepSeek V3 |  |  | 131.1K | 16K | T | $0.200 | $0.800 |
| `deepseek/deepseek-chat-v2.5` | DeepSeek V2.5 |  |  | 128K | - | T | 0 | 0 |
| `deepseek/deepseek-chat-v3-0324` | DeepSeek: DeepSeek V3 0324 |  |  | 163.8K | 16.4K | T | $0.240 | $0.900 |
| `deepseek/deepseek-chat-v3.1` | DeepSeek: DeepSeek V3.1 | 🧠 |  | 163.8K | 32.8K | T | $0.250 | $0.950 |
| `deepseek/deepseek-prover-v2` | DeepSeek: DeepSeek Prover V2 |  |  | 163.8K | - | T | 0 | 0 |
| `deepseek/deepseek-r1` | DeepSeek: R1 | ⚠️ |  | 163.8K | 16K | T | $0.700 | $2.50 |
| `deepseek/deepseek-r1-0528` | DeepSeek: R1 0528 | ⚠️ |  | 163.8K | 32.8K | T | $0.500 | $2.15 |
| `deepseek/deepseek-r1-zero` | DeepSeek: DeepSeek R1 Zero | 🧠 |  | 163.8K | - | T | 0 | 0 |
| `deepseek/deepseek-v3-base` | DeepSeek: DeepSeek V3 Base |  |  | 131.1K | - | T | 0 | 0 |
| `deepseek/deepseek-v3.1-base` | DeepSeek: DeepSeek V3.1 Base |  |  | 163.8K | - | T | 0 | 0 |
| `deepseek/deepseek-v3.1-terminus` | DeepSeek: DeepSeek V3.1 Terminus | 🧠 |  | 163.8K | 32.8K | T | $0.270 | $0.950 |
| `deepseek/deepseek-v3.2` | DeepSeek: DeepSeek V3.2 | 🧠 |  | 131.1K | 64K | T | $0.214 | $0.322 |
| `deepseek/deepseek-v3.2-exp` | DeepSeek: DeepSeek V3.2 Exp | 🧠 |  | 163.8K | 65.5K | T | $0.270 | $0.410 |
| `deepseek/deepseek-v3.2-speciale` | DeepSeek: DeepSeek V3.2 Speciale | ⚠️ |  | 131.1K | - | T | 0 | 0 |
| `deepseek/deepseek-v4-flash` | DeepSeek: DeepSeek V4 Flash | 🧠 |  | 1.0M | 65.5K | T | $0.090 | $0.180 |
| `deepseek/deepseek-v4-pro` | DeepSeek: DeepSeek V4 Pro | 🧠 |  | 1.0M | 384K | T | $0.435 | $0.870 |
| `nex-agi/deepseek-v3.1-nex-n1` | Nex AGI: DeepSeek V3.1 Nex N1 |  |  | 8.2K | - | T | 0 | 0 |
| `tngtech/deepseek-r1t-chimera` | TNG: DeepSeek R1T Chimera | 🧠 |  | 163.8K | - | T | 0 | 0 |
| `tngtech/deepseek-r1t2-chimera` | TNG: DeepSeek R1T2 Chimera | 🧠 |  | 163.8K | - | T | 0 | 0 |

---

## Mistral <a id="mistral"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `cognitivecomputations/dolphin-mistral-24b-venice-edition` | Venice: Uncensored |  |  | 128K | 8.2K | T | $0.200 | $0.900 |
| `cognitivecomputations/dolphin-mixtral-8x22b` | Dolphin 2.9.2 Mixtral 8x22B 🐬 |  |  | 65.5K | - | T | 0 | 0 |
| `cognitivecomputations/dolphin-mixtral-8x7b` | Dolphin 2.6 Mixtral 8x7B 🐬 |  |  | 32.8K | - | T | 0 | 0 |
| `cognitivecomputations/dolphin3.0-mistral-24b` | Dolphin3.0 Mistral 24B |  |  | 32.8K | - | T | 0 | 0 |
| `cognitivecomputations/dolphin3.0-r1-mistral-24b` | Dolphin3.0 R1 Mistral 24B | 🧠 |  | 32.8K | - | T | 0 | 0 |
| `mistralai/codestral-2501` | Mistral: Codestral 2501 |  |  | 256K | - | T | 0 | 0 |
| `mistralai/codestral-2508` | Mistral: Codestral 2508 |  |  | 256K | - | T/file | $0.300 | $0.900 |
| `mistralai/codestral-embed-2505` | Mistral: Codestral Embed 2505 |  |  | 8.2K | - | T | $0.150 | 0 |
| `mistralai/codestral-mamba` | Mistral: Codestral Mamba |  |  | 256K | - | T | 0 | 0 |
| `mistralai/devstral-2512` | Mistral: Devstral 2 2512 |  |  | 262.1K | - | T/file | $0.400 | $2.00 |
| `mistralai/devstral-medium` | Mistral: Devstral Medium |  |  | 131.1K | - | T/file | 0 | 0 |
| `mistralai/devstral-small` | Mistral: Devstral Small 1.1 |  |  | 131.1K | - | T/file | 0 | 0 |
| `mistralai/devstral-small-2505` | Mistral: Devstral Small 2505 |  |  | 131.1K | - | T | 0 | 0 |
| `mistralai/mistral-7b-instruct` | Mistral: Mistral 7B Instruct |  |  | 32.8K | - | T | 0 | 0 |
| `mistralai/mistral-7b-instruct-v0.1` | Mistral: Mistral 7B Instruct v0.1 |  |  | 4.1K | - | T | 0 | 0 |
| `mistralai/mistral-7b-instruct-v0.2` | Mistral: Mistral 7B Instruct v0.2 |  |  | 32.8K | - | T | 0 | 0 |
| `mistralai/mistral-7b-instruct-v0.3` | Mistral: Mistral 7B Instruct v0.3 |  |  | 32.8K | - | T | 0 | 0 |
| `mistralai/mistral-embed-2312` | Mistral: Mistral Embed 2312 |  |  | 8.2K | - | T | $0.100 | 0 |
| `mistralai/mistral-large` | Mistral Large |  |  | 128K | - | T/file | $2.00 | $6.00 |
| `mistralai/mistral-large-2407` | Mistral Large 2407 |  |  | 131.1K | - | T/file | $2.00 | $6.00 |
| `mistralai/mistral-large-2411` | Mistral Large 2411 |  |  | 128K | - | T/file | 0 | 0 |
| `mistralai/mistral-large-2512` | Mistral: Mistral Large 3 2512 |  |  | 262.1K | - | T/I/file | $0.500 | $1.50 |
| `mistralai/mistral-medium` | Mistral Medium |  |  | 32K | - | T | 0 | 0 |
| `mistralai/mistral-medium-3` | Mistral: Mistral Medium 3 |  |  | 131.1K | - | T/I/file | $0.400 | $2.00 |
| `mistralai/mistral-medium-3-5` | Mistral: Mistral Medium 3.5 | 🧠 |  | 262.1K | - | T/I/file | $1.50 | $7.50 |
| `mistralai/mistral-medium-3.1` | Mistral: Mistral Medium 3.1 |  |  | 131.1K | - | T/I/file | $0.400 | $2.00 |
| `mistralai/mistral-nemo` | Mistral: Mistral Nemo |  |  | 131.1K | - | T | $0.018 | $0.030 |
| `mistralai/mistral-saba` | Mistral: Saba |  |  | 32.8K | - | T/file | $0.200 | $0.600 |
| `mistralai/mistral-small` | Mistral Small |  |  | 32K | - | T | 0 | 0 |
| `mistralai/mistral-small-24b-instruct-2501` | Mistral: Mistral Small 3 |  |  | 32.8K | 16.4K | T | $0.050 | $0.080 |
| `mistralai/mistral-small-2603` | Mistral: Mistral Small 4 | 🧠 |  | 262.1K | - | T/I | $0.150 | $0.600 |
| `mistralai/mistral-small-3.1-24b-instruct` | Mistral: Mistral Small 3.1 24B |  |  | 128K | 128K | T/I | $0.351 | $0.555 |
| `mistralai/mistral-small-3.2-24b-instruct` | Mistral: Mistral Small 3.2 24B |  |  | 128K | 16.4K | I/T | $0.075 | $0.200 |
| `mistralai/mistral-small-creative` | Mistral: Mistral Small Creative |  |  | 32.8K | - | T | 0 | 0 |
| `mistralai/mistral-tiny` | Mistral Tiny |  |  | 32K | - | T | 0 | 0 |
| `mistralai/mixtral-8x22b` | Mistral: Mixtral 8x22B (base) |  |  | 65.5K | - | T | 0 | 0 |
| `mistralai/mixtral-8x22b-instruct` | Mistral: Mixtral 8x22B Instruct |  |  | 65.5K | - | T/file | $2.00 | $6.00 |
| `mistralai/mixtral-8x7b-instruct` | Mistral: Mixtral 8x7B Instruct |  |  | 32.8K | - | T | 0 | 0 |
| `mistralai/pixtral-12b` | Mistral: Pixtral 12B |  |  | 4.1K | - | T/I | 0 | 0 |
| `mistralai/pixtral-large-2411` | Mistral: Pixtral Large 2411 |  |  | 128K | - | T/I/file | 0 | 0 |
| `neversleep/noromaid-mixtral-8x7b-instruct` | Noromaid Mixtral 8x7B Instruct |  |  | 8K | - | T | 0 | 0 |
| `nousresearch/deephermes-3-mistral-24b-preview` | Nous: DeepHermes 3 Mistral 24B Preview | 🧠 |  | 32.8K | - | T | 0 | 0 |
| `nousresearch/nous-hermes-2-mistral-7b-dpo` | Nous: Hermes 2 Mistral 7B DPO |  |  | 8.2K | - | T | 0 | 0 |
| `nousresearch/nous-hermes-2-mixtral-8x7b-dpo` | Nous: Hermes 2 Mixtral 8x7B DPO |  |  | 32.8K | - | T | 0 | 0 |
| `nousresearch/nous-hermes-2-mixtral-8x7b-sft` | Nous: Hermes 2 Mixtral 8x7B SFT |  |  | 32.8K | - | T | 0 | 0 |
| `open-orca/mistral-7b-openorca` | Mistral OpenOrca 7B |  |  | 8.2K | - | T | 0 | 0 |
| `teknium/openhermes-2-mistral-7b` | OpenHermes 2 Mistral 7B |  |  | 8.2K | - | T | 0 | 0 |
| `teknium/openhermes-2.5-mistral-7b` | OpenHermes 2.5 Mistral 7B |  |  | 4.1K | - | T | 0 | 0 |

---

## Grok <a id="grok"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `x-ai/grok-2` | xAI: Grok 2 |  |  | 32.8K | - | T | 0 | 0 |
| `x-ai/grok-2-1212` | xAI: Grok 2 1212 |  |  | 131.1K | - | T | 0 | 0 |
| `x-ai/grok-2-mini` | xAI: Grok 2 mini |  |  | 32.8K | - | T | 0 | 0 |
| `x-ai/grok-2-vision-1212` | xAI: Grok 2 Vision 1212 |  |  | 32.8K | - | T/I | 0 | 0 |
| `x-ai/grok-3` | xAI: Grok 3 |  |  | 131.1K | - | T | 0 | 0 |
| `x-ai/grok-3-beta` | xAI: Grok 3 Beta |  |  | 131.1K | - | T | 0 | 0 |
| `x-ai/grok-3-mini` | xAI: Grok 3 Mini | ⚠️ |  | 131.1K | - | T | 0 | 0 |
| `x-ai/grok-3-mini-beta` | xAI: Grok 3 Mini Beta | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `x-ai/grok-4` | xAI: Grok 4 | ⚠️ |  | 256K | - | I/T/file | 0 | 0 |
| `x-ai/grok-4-fast` | xAI: Grok 4 Fast | 🧠 |  | 2M | - | T/I/file | 0 | 0 |
| `x-ai/grok-4.1-fast` | xAI: Grok 4.1 Fast | 🧠 |  | 2M | - | T/I/file | 0 | 0 |
| `x-ai/grok-4.20` | xAI: Grok 4.20 | 🧠 |  | 2M | - | T/I/file | $1.25 | $2.50 |
| `x-ai/grok-4.20-beta` | xAI: Grok 4.20 Beta | 🧠 |  | 2M | - | T/I/file | 0 | 0 |
| `x-ai/grok-4.20-multi-agent` | xAI: Grok 4.20 Multi-Agent | ⚠️ |  | 2M | - | T/I/file | $1.25 | $2.50 |
| `x-ai/grok-4.20-multi-agent-beta` | xAI: Grok 4.20 Multi-Agent Beta | ⚠️ |  | 2M | - | T/I/file | 0 | 0 |
| `x-ai/grok-4.3` | xAI: Grok 4.3 | 🧠 |  | 1M | - | T/I/file | $1.25 | $2.50 |
| `x-ai/grok-4.5` | xAI: Grok 4.5 | ⚠️ |  | 500K | - | T/I/file | $2.00 | $6.00 |
| `x-ai/grok-beta` | xAI: Grok Beta |  |  | 131.1K | - | T | 0 | 0 |
| `x-ai/grok-build-0.1` | xAI: Grok Build 0.1 | ⚠️ |  | 256K | - | T/I/file | $1.00 | $2.00 |
| `x-ai/grok-code-fast-1` | xAI: Grok Code Fast 1 | 🧠 |  | 256K | - | T | 0 | 0 |
| `x-ai/grok-imagine-image-quality` | xAI: Grok Imagine Image Quality |  |  | 65.5K | - | T/I | 0 | 0 |
| `x-ai/grok-imagine-video` | xAI: Grok Imagine Video |  |  | 0 | - | T/I | 0 | 0 |
| `x-ai/grok-imagine-video-1.5` | xAI: Grok Imagine Video 1.5 |  |  | 0 | - | T/I | 0 | 0 |
| `x-ai/grok-vision-beta` | xAI: Grok Vision Beta |  |  | 8.2K | - | T/I | 0 | 0 |
| `x-ai/grok-voice-tts-1.0` | xAI: Grok Voice TTS 1.0 |  |  | 15K | - | T | $15.00 | 0 |
| `~x-ai/grok-latest` | xAI: Grok Latest | ⚠️ |  | 500K | - | T/I/file | $2.00 | $6.00 |

---

## Nova <a id="nova"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `amazon/nova-2-lite-v1` | Amazon: Nova 2 Lite | 🧠 |  | 1M | 65.5K | T/I/V/file | $0.300 | $2.50 |
| `amazon/nova-lite-v1` | Amazon: Nova Lite 1.0 |  |  | 300K | 5.1K | T/I | $0.060 | $0.240 |
| `amazon/nova-micro-v1` | Amazon: Nova Micro 1.0 |  |  | 128K | 5.1K | T | $0.035 | $0.140 |
| `amazon/nova-premier-v1` | Amazon: Nova Premier 1.0 |  |  | 1M | 32K | T/I | $2.50 | $12.50 |
| `amazon/nova-pro-v1` | Amazon: Nova Pro 1.0 |  |  | 300K | 5.1K | T/I | $0.800 | $3.20 |
| `deepgram/nova-3` | Deepgram: Nova-3 |  |  | 0 | - | A | $4300.00 | 0 |

---

## Minimax <a id="minimax"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `minimax/hailuo-2.3` | MiniMax: Hailuo 2.3 |  |  | 0 | - | T/I | 0 | 0 |
| `minimax/minimax-01` | MiniMax: MiniMax-01 |  |  | 1.0M | 1.0M | T/I | $0.200 | $1.10 |
| `minimax/minimax-m1` | MiniMax: MiniMax M1 | 🧠 |  | 1M | 40K | T | $0.400 | $2.20 |
| `minimax/minimax-m2` | MiniMax: MiniMax M2 | ⚠️ |  | 204.8K | 131.1K | T | $0.255 | $1.02 |
| `minimax/minimax-m2-her` | MiniMax: MiniMax M2-her |  |  | 65.5K | 2.0K | T | $0.300 | $1.20 |
| `minimax/minimax-m2.1` | MiniMax: MiniMax M2.1 | ⚠️ |  | 204.8K | 131.1K | T | $0.300 | $1.20 |
| `minimax/minimax-m2.5` | MiniMax: MiniMax M2.5 | ⚠️ |  | 204.8K | 196.6K | T | $0.150 | $0.900 |
| `minimax/minimax-m2.7` | MiniMax: MiniMax M2.7 | ⚠️ |  | 204.8K | 196.6K | T | $0.240 | $0.960 |
| `minimax/minimax-m3` | MiniMax: MiniMax M3 | 🧠 |  | 1.0M | 131.1K | T/I/V | $0.300 | $1.20 |
| `minimax/speech-2.8-hd` | MiniMax: Speech 2.8 HD |  |  | 0 | - | T | $100.00 | 0 |
| `minimax/speech-2.8-turbo` | MiniMax: Speech 2.8 Turbo |  |  | 0 | - | T | $60.00 | 0 |

---

## Glm <a id="glm"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `thudm/glm-4-32b` | THUDM: GLM 4 32B |  |  | 32.8K | - | T | 0 | 0 |
| `thudm/glm-4-9b` | THUDM: GLM 4 9B |  |  | 32K | - | T | 0 | 0 |
| `thudm/glm-4.1v-9b-thinking` | THUDM: GLM 4.1V 9B Thinking | 🧠 |  | 65.5K | - | I/T | 0 | 0 |
| `thudm/glm-z1-32b` | THUDM: GLM Z1 32B | 🧠 |  | 32.8K | - | T | 0 | 0 |
| `thudm/glm-z1-9b` | THUDM: GLM Z1 9B | 🧠 |  | 32K | - | T | 0 | 0 |
| `thudm/glm-z1-rumination-32b` | THUDM: GLM Z1 Rumination 32B  | 🧠 |  | 32K | - | T | 0 | 0 |
| `z-ai/glm-4-32b` | Z.ai: GLM 4 32B  |  |  | 128K | - | T | 0 | 0 |
| `z-ai/glm-4.5` | Z.ai: GLM 4.5 | 🧠 |  | 131.1K | 98.3K | T | $0.600 | $2.20 |
| `z-ai/glm-4.5-air` | Z.ai: GLM 4.5 Air | 🧠 |  | 131.1K | 98.3K | T | $0.130 | $0.850 |
| `z-ai/glm-4.5v` | Z.ai: GLM 4.5V | 🧠 |  | 65.5K | 16.4K | T/I | $0.600 | $1.80 |
| `z-ai/glm-4.6` | Z.ai: GLM 4.6 | 🧠 |  | 200K | 16.4K | T | $0.430 | $1.75 |
| `z-ai/glm-4.6v` | Z.ai: GLM 4.6V | 🧠 |  | 131.1K | 32.8K | I/T/V | $0.300 | $0.900 |
| `z-ai/glm-4.7` | Z.ai: GLM 4.7 | 🧠 |  | 202.8K | 131.1K | T | $0.400 | $1.75 |
| `z-ai/glm-4.7-flash` | Z.ai: GLM 4.7 Flash | 🧠 |  | 202.8K | 16.4K | T | $0.060 | $0.400 |
| `z-ai/glm-5` | Z.ai: GLM 5 | 🧠 |  | 202.8K | 128K | T | $0.600 | $1.92 |
| `z-ai/glm-5-turbo` | Z.ai: GLM 5 Turbo | 🧠 |  | 202.8K | 131.1K | T | $1.20 | $4.00 |
| `z-ai/glm-5.1` | Z.ai: GLM 5.1 | 🧠 |  | 202.8K | 128K | T | $0.966 | $3.04 |
| `z-ai/glm-5.2` | Z.ai: GLM 5.2 | 🧠 |  | 1.0M | 32.8K | T | $0.930 | $3.00 |
| `z-ai/glm-5v-turbo` | Z.ai: GLM 5V Turbo | 🧠 |  | 202.8K | 131.1K | I/T/V | $1.20 | $4.00 |

---

## Ernie <a id="ernie"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `baidu/ernie-4.5-21b-a3b` | Baidu: ERNIE 4.5 21B A3B | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `baidu/ernie-4.5-21b-a3b-thinking` | Baidu: ERNIE 4.5 21B A3B Thinking | ⚠️ |  | 131.1K | - | T | 0 | 0 |
| `baidu/ernie-4.5-300b-a47b` | Baidu: ERNIE 4.5 300B A47B  |  |  | 131.1K | - | T | 0 | 0 |
| `baidu/ernie-4.5-vl-28b-a3b` | Baidu: ERNIE 4.5 VL 28B A3B | 🧠 |  | 131.1K | - | T/I | 0 | 0 |
| `baidu/ernie-4.5-vl-424b-a47b` | Baidu: ERNIE 4.5 VL 424B A47B  | 🧠 |  | 131.1K | 16K | I/T | $0.420 | $1.25 |

---

## Kimi <a id="kimi"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `moonshotai/kimi-dev-72b` | MoonshotAI: Kimi Dev 72B | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `moonshotai/kimi-k2` | MoonshotAI: Kimi K2 0711 |  |  | 131.1K | 100.4K | T | $0.570 | $2.30 |
| `moonshotai/kimi-k2-0905` | MoonshotAI: Kimi K2 0905 |  |  | 262.1K | 100.4K | T | $0.600 | $2.50 |
| `moonshotai/kimi-k2-thinking` | MoonshotAI: Kimi K2 Thinking | ⚠️ |  | 262.1K | 100.4K | T | $0.600 | $2.50 |
| `moonshotai/kimi-k2.5` | MoonshotAI: Kimi K2.5 | 🧠 |  | 262.1K | - | T/I | $0.375 | $2.02 |
| `moonshotai/kimi-k2.6` | MoonshotAI: Kimi K2.6 | 🧠 |  | 262.1K | 262.1K | T/I | $0.660 | $3.41 |
| `moonshotai/kimi-k2.7-code` | MoonshotAI: Kimi K2.7 Code | ⚠️ |  | 262.1K | 262.1K | T/I | $0.720 | $3.50 |
| `moonshotai/kimi-k3` | MoonshotAI: Kimi K3 | 🧠 |  | 1.0M | - | T/I | $3.00 | $15.00 |
| `moonshotai/kimi-vl-a3b-thinking` | MoonshotAI: Kimi VL A3B Thinking | 🧠 |  | 131.1K | - | I/T | 0 | 0 |
| `~moonshotai/kimi-latest` | MoonshotAI Kimi Latest | 🧠 |  | 1.0M | - | T/I | $3.00 | $15.00 |

---

## Nemotron <a id="nemotron"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `nvidia/nemotron-3-embed-1b` | NVIDIA: Nemotron 3 Embed 1B (free) |  | 🆓 | 32.8K | - | T | 0 | 0 |
| `nvidia/nemotron-3-nano-30b-a3b` | NVIDIA: Nemotron 3 Nano 30B A3B (free) | 🧠 | 🆓 | 256K | - | T | 0 | 0 |
| `nvidia/nemotron-3-nano-30b-a3b` | NVIDIA: Nemotron 3 Nano 30B A3B | 🧠 |  | 262.1K | 228K | T | $0.050 | $0.200 |
| `nvidia/nemotron-3-nano-omni-30b-a3b-reasoning` | NVIDIA: Nemotron 3 Nano Omni (free) | 🧠 | 🆓 | 256K | 65.5K | T/A/I/V | 0 | 0 |
| `nvidia/nemotron-3-super-120b-a12b` | NVIDIA: Nemotron 3 Super (free) | 🧠 | 🆓 | 1M | 262.1K | T | 0 | 0 |
| `nvidia/nemotron-3-super-120b-a12b` | NVIDIA: Nemotron 3 Super | 🧠 |  | 1M | - | T | $0.080 | $0.450 |
| `nvidia/nemotron-3-ultra-550b-a55b` | NVIDIA: Nemotron 3 Ultra (free) | 🧠 | 🆓 | 1M | 65.5K | T | 0 | 0 |
| `nvidia/nemotron-3-ultra-550b-a55b` | NVIDIA: Nemotron 3 Ultra | 🧠 |  | 1M | 16.4K | T | $0.500 | $2.20 |
| `nvidia/nemotron-3.5-content-safety` | NVIDIA: Nemotron 3.5 Content Safety (free) | 🧠 | 🆓 | 128K | 8.2K | T/I | 0 | 0 |
| `nvidia/nemotron-4-340b-instruct` | NVIDIA: Nemotron-4 340B Instruct |  |  | 4.1K | - | T | 0 | 0 |
| `nvidia/nemotron-nano-12b-v2-vl` | NVIDIA: Nemotron Nano 12B 2 VL (free) | 🧠 | 🆓 | 128K | 128K | I/T/V | 0 | 0 |
| `nvidia/nemotron-nano-9b-v2` | NVIDIA: Nemotron Nano 9B V2 (free) | 🧠 | 🆓 | 128K | - | T | 0 | 0 |

---

## Cohere <a id="cohere"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `cohere/command` | Cohere: Command |  |  | 4.1K | - | T | 0 | 0 |
| `cohere/command-a` | Cohere: Command A |  |  | 256K | 8.2K | T | $2.50 | $10.00 |
| `cohere/command-r` | Cohere: Command R |  |  | 128K | - | T | 0 | 0 |
| `cohere/command-r-03-2024` | Cohere: Command R (03-2024) |  |  | 128K | - | T | 0 | 0 |
| `cohere/command-r-08-2024` | Cohere: Command R (08-2024) |  |  | 128K | 4K | T | $0.150 | $0.600 |
| `cohere/command-r-plus` | Cohere: Command R+ |  |  | 128K | - | T | 0 | 0 |
| `cohere/command-r-plus-04-2024` | Cohere: Command R+ (04-2024) |  |  | 128K | - | T | 0 | 0 |
| `cohere/command-r-plus-08-2024` | Cohere: Command R+ (08-2024) |  |  | 128K | 4K | T | $2.50 | $10.00 |
| `cohere/command-r7b-12-2024` | Cohere: Command R7B (12-2024) |  |  | 128K | 4K | T | $0.037 | $0.150 |
| `cohere/north-mini-code` | Cohere: North Mini Code (free) | 🧠 | 🆓 | 256K | 64K | T | 0 | 0 |
| `cohere/rerank-4-fast` | Cohere: Rerank 4 Fast |  |  | 32.8K | - | T | 0 | 0 |
| `cohere/rerank-4-pro` | Cohere: Rerank 4 Pro |  |  | 32.8K | - | T | 0 | 0 |
| `cohere/rerank-v3.5` | Cohere: Rerank v3.5 |  |  | 4.1K | - | T | 0 | 0 |

---

## Phi <a id="phi"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `microsoft/phi-3-medium-128k-instruct` | Microsoft: Phi-3 Medium 128K Instruct |  |  | 128K | - | T | 0 | 0 |
| `microsoft/phi-3-medium-4k-instruct` | Microsoft: Phi-3 Medium 4K Instruct |  |  | 4K | - | T | 0 | 0 |
| `microsoft/phi-3-mini-128k-instruct` | Microsoft: Phi-3 Mini 128K Instruct |  |  | 128K | - | T | 0 | 0 |
| `microsoft/phi-3.5-mini-128k-instruct` | Microsoft: Phi-3.5 Mini 128K Instruct |  |  | 128K | - | T | 0 | 0 |
| `microsoft/phi-4` | Microsoft: Phi 4 |  |  | 16.4K | 16.4K | T | $0.070 | $0.140 |
| `microsoft/phi-4-mini-instruct` | Microsoft: Phi 4 Mini Instruct |  |  | 131.1K | - | T | 0 | 0 |
| `microsoft/phi-4-multimodal-instruct` | Microsoft: Phi 4 Multimodal Instruct |  |  | 131.1K | - | T/I | 0 | 0 |
| `microsoft/phi-4-reasoning` | Microsoft: Phi 4 Reasoning | 🧠 |  | 32.8K | - | T | 0 | 0 |
| `microsoft/phi-4-reasoning-plus` | Microsoft: Phi 4 Reasoning Plus | 🧠 |  | 32.8K | - | T | 0 | 0 |

---

## Aetherwiing <a id="aetherwiing"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `aetherwiing/mn-starcannon-12b` | Aetherwiing: Starcannon 12B |  |  | 12K | - | T | 0 | 0 |

---

## Agentica Org <a id="agentica-org"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `agentica-org/deepcoder-14b-preview` | Agentica: Deepcoder 14B Preview | 🧠 |  | 96K | - | T | 0 | 0 |

---

## Ai21 <a id="ai21"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `ai21/jamba-1-5-large` | AI21: Jamba 1.5 Large |  |  | 256K | - | T | 0 | 0 |
| `ai21/jamba-1-5-mini` | AI21: Jamba 1.5 Mini |  |  | 256K | - | T | 0 | 0 |
| `ai21/jamba-1.6-large` | AI21: Jamba 1.6 Large |  |  | 256K | - | T | 0 | 0 |
| `ai21/jamba-1.6-mini` | AI21: Jamba Mini 1.6 |  |  | 256K | - | T | 0 | 0 |
| `ai21/jamba-instruct` | AI21: Jamba Instruct |  |  | 256K | - | T | 0 | 0 |
| `ai21/jamba-large-1.7` | AI21: Jamba Large 1.7 |  |  | 256K | 4.1K | T | $2.00 | $8.00 |
| `ai21/jamba-mini-1.7` | AI21: Jamba Mini 1.7 |  |  | 256K | - | T | 0 | 0 |

---

## Aion Labs <a id="aion-labs"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `aion-labs/aion-1.0` | AionLabs: Aion-1.0 | 🧠 |  | 32.8K | - | T | 0 | 0 |
| `aion-labs/aion-1.0-mini` | AionLabs: Aion-1.0-Mini | 🧠 |  | 16.4K | - | T | 0 | 0 |
| `aion-labs/aion-2.0` | AionLabs: Aion-2.0 | ⚠️ |  | 131.1K | 32.8K | T | $0.800 | $1.60 |
| `aion-labs/aion-3.0` | AionLabs: Aion-3.0 | ⚠️ |  | 131.1K | 32.8K | T | $3.00 | $6.00 |
| `aion-labs/aion-3.0-mini` | AionLabs: Aion-3.0-Mini | ⚠️ |  | 131.1K | 32.8K | T | $0.700 | $1.40 |

---

## Alibaba <a id="alibaba"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `alibaba/happyhorse-1.0` | Alibaba: HappyHorse 1.0 |  |  | 0 | - | T/I | 0 | 0 |
| `alibaba/happyhorse-1.1` | Alibaba: HappyHorse 1.1 |  |  | 0 | - | T/I | 0 | 0 |
| `alibaba/wan-2.6` | Alibaba: Wan 2.6 |  |  | 0 | - | T/I | 0 | 0 |
| `alibaba/wan-2.7` | Alibaba: Wan 2.7 |  |  | 0 | - | T/I | 0 | 0 |

---

## All Hands <a id="all-hands"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `all-hands/openhands-lm-32b-v0.1` | OpenHands LM 32B V0.1 |  |  | 131.1K | - | T | 0 | 0 |

---

## Allenai <a id="allenai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `allenai/molmo-2-8b` | AllenAI: Molmo2 8B |  |  | 36.9K | - | T/I/V | 0 | 0 |
| `allenai/olmo-2-0325-32b-instruct` | AllenAI: Olmo 2 32B Instruct |  |  | 128K | - | T | 0 | 0 |
| `allenai/olmo-3-32b-think` | AllenAI: Olmo 3 32B Think | ⚠️ |  | 65.5K | 65.5K | T | $0.150 | $0.500 |
| `allenai/olmo-3-7b-instruct` | AllenAI: Olmo 3 7B Instruct |  |  | 65.5K | - | T | 0 | 0 |
| `allenai/olmo-3-7b-think` | AllenAI: Olmo 3 7B Think | ⚠️ |  | 65.5K | - | T | 0 | 0 |
| `allenai/olmo-3.1-32b-instruct` | AllenAI: Olmo 3.1 32B Instruct |  |  | 65.5K | - | T | 0 | 0 |
| `allenai/olmo-3.1-32b-think` | AllenAI: Olmo 3.1 32B Think | ⚠️ |  | 65.5K | - | T | 0 | 0 |
| `allenai/olmo-7b-instruct` | OLMo 7B Instruct |  |  | 2.0K | - | T | 0 | 0 |

---

## Alpindale <a id="alpindale"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `alpindale/goliath-120b` | Goliath 120B |  |  | 6.1K | - | T | 0 | 0 |
| `alpindale/magnum-72b` | Magnum 72B |  |  | 16.4K | - | T | 0 | 0 |

---

## Anthracite Org <a id="anthracite-org"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `anthracite-org/magnum-v2-72b` | Magnum v2 72B |  |  | 32.8K | - | T | 0 | 0 |
| `anthracite-org/magnum-v4-72b` | Magnum v4 72B |  |  | 32.8K | 2.0K | T | $3.00 | $5.00 |

---

## Arcee Ai <a id="arcee-ai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `arcee-ai/afm-4.5b` | Arcee AI: AFM 4.5B |  |  | 65.5K | - | T | 0 | 0 |
| `arcee-ai/arcee-blitz` | Arcee AI: Arcee Blitz |  |  | 32.8K | - | T | 0 | 0 |
| `arcee-ai/caller-large` | Arcee AI: Caller Large |  |  | 32.8K | - | T | 0 | 0 |
| `arcee-ai/coder-large` | Arcee AI: Coder Large |  |  | 32.8K | - | T | 0 | 0 |
| `arcee-ai/maestro-reasoning` | Arcee AI: Maestro Reasoning |  |  | 131.1K | - | T | 0 | 0 |
| `arcee-ai/spotlight` | Arcee AI: Spotlight |  |  | 131.1K | - | I/T | 0 | 0 |
| `arcee-ai/trinity-large-preview` | Arcee AI: Trinity Large Preview |  |  | 131K | - | T | 0 | 0 |
| `arcee-ai/trinity-large-thinking` | Arcee AI: Trinity Large Thinking | ⚠️ |  | 262.1K | 262.1K | T | $0.220 | $0.850 |
| `arcee-ai/trinity-mini` | Arcee AI: Trinity Mini | ⚠️ |  | 131.1K | - | T | 0 | 0 |
| `arcee-ai/virtuoso-large` | Arcee AI: Virtuoso Large |  |  | 131.1K | 64K | T | $0.750 | $1.20 |
| `arcee-ai/virtuoso-medium-v2` | Arcee AI: Virtuoso Medium V2 |  |  | 131.1K | - | T | 0 | 0 |

---

## Arliai <a id="arliai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `arliai/qwq-32b-arliai-rpr-v1` | ArliAI: QwQ 32B RpR v1 | 🧠 |  | 32.8K | - | T | 0 | 0 |

---

## Baai <a id="baai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `baai/bge-base-en-v1.5` | BAAI: bge-base-en-v1.5 |  |  | 8.2K | - | T | $0.0050 | 0 |
| `baai/bge-large-en-v1.5` | BAAI: bge-large-en-v1.5 |  |  | 8.2K | - | T | $0.010 | 0 |
| `baai/bge-m3` | BAAI: bge-m3 |  |  | 8.2K | - | T | $0.010 | 0 |

---

## Baidu <a id="baidu"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `baidu/cobuddy` | Baidu Qianfan: CoBuddy | ⚠️ |  | 131.1K | - | T | 0 | 0 |
| `baidu/qianfan-ocr-fast` | Baidu: Qianfan-OCR-Fast | 🧠 |  | 65.5K | - | I/T | 0 | 0 |

---

## Bigcode <a id="bigcode"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `bigcode/starcoder2-15b-instruct` | StarCoder2 15B Instruct |  |  | 16.4K | - | T | 0 | 0 |

---

## Black Forest Labs <a id="black-forest-labs"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `black-forest-labs/flux.2-flex` | Black Forest Labs: FLUX.2 Flex |  |  | 67.3K | - | T/I | 0 | 0 |
| `black-forest-labs/flux.2-klein-4b` | Black Forest Labs: FLUX.2 Klein 4B |  |  | 41.0K | - | T/I | 0 | 0 |
| `black-forest-labs/flux.2-max` | Black Forest Labs: FLUX.2 Max |  |  | 46.9K | - | T/I | 0 | 0 |
| `black-forest-labs/flux.2-pro` | Black Forest Labs: FLUX.2 Pro |  |  | 46.9K | - | T/I | 0 | 0 |

---

## Bytedance <a id="bytedance"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `bytedance/seed-oss-36b-instruct` | ByteDance: Seed OSS 36B Instruct | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `bytedance/seedance-1-5-pro` | ByteDance: Seedance 1.5 Pro |  |  | 0 | - | T/I | 0 | 0 |
| `bytedance/seedance-2.0` | ByteDance: Seedance 2.0 |  |  | 0 | - | T/I/V/A | 0 | 0 |
| `bytedance/seedance-2.0-fast` | ByteDance: Seedance 2.0 Fast |  |  | 0 | - | T/I/V/A | 0 | 0 |
| `bytedance/ui-tars-1.5-7b` | ByteDance: UI-TARS 7B  |  |  | 128K | 2.0K | I/T | $0.100 | $0.200 |

---

## Bytedance Research <a id="bytedance-research"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `bytedance-research/ui-tars-72b` | Bytedance: UI-TARS 72B  |  |  | 32.8K | - | T/I | 0 | 0 |

---

## Bytedance Seed <a id="bytedance-seed"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `bytedance-seed/seed-1.6` | ByteDance Seed: Seed 1.6 | 🧠 |  | 262.1K | 32.8K | I/T/V | $0.250 | $2.00 |
| `bytedance-seed/seed-1.6-flash` | ByteDance Seed: Seed 1.6 Flash | 🧠 |  | 262.1K | 32.8K | I/T/V | $0.075 | $0.300 |
| `bytedance-seed/seed-2.0-lite` | ByteDance Seed: Seed-2.0-Lite | 🧠 |  | 262.1K | 131.1K | T/I/V | $0.250 | $2.00 |
| `bytedance-seed/seed-2.0-mini` | ByteDance Seed: Seed-2.0-Mini | 🧠 |  | 262.1K | 131.1K | T/I/V | $0.100 | $0.400 |
| `bytedance-seed/seedream-4.5` | ByteDance Seed: Seedream 4.5 |  |  | 4.1K | - | I/T | 0 | 0 |

---

## Canopylabs <a id="canopylabs"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `canopylabs/orpheus-3b-0.1-ft` | Canopy Labs: Orpheus 3B |  |  | 4.1K | - | T | $7.00 | 0 |

---

## Databricks <a id="databricks"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `databricks/dbrx-instruct` | Databricks: DBRX 132B Instruct |  |  | 32.8K | - | T | 0 | 0 |

---

## Deepcogito <a id="deepcogito"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `deepcogito/cogito-v2.1-671b` | Deep Cogito: Cogito v2.1 671B | 🧠 |  | 128K | - | T | $1.25 | $1.25 |

---

## Deepgram <a id="deepgram"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `deepgram/aura-2` | Deepgram: Aura-2 |  |  | 0 | - | T | $30.00 | 0 |

---

## Essentialai <a id="essentialai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `essentialai/rnj-1-instruct` | EssentialAI: Rnj 1 Instruct | 🧠 |  | 32.8K | - | T | 0 | 0 |

---

## Featherless <a id="featherless"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `featherless/qwerky-72b` | Qrwkv 72B |  |  | 32.8K | - | T | 0 | 0 |

---

## Fireworks <a id="fireworks"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `fireworks/firellava-13b` | Fireworks: FireLLaVA 13B |  |  | 4.1K | - | T/I | 0 | 0 |

---

## Google <a id="google"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `google/chirp-3` | Google: Chirp 3 |  |  | 0 | - | A | $16000.00 | 0 |
| `google/lyria-3-clip-preview` | Google: Lyria 3 Clip Preview |  |  | 1.0M | 65.5K | T/I | 0 | 0 |
| `google/lyria-3-pro-preview` | Google: Lyria 3 Pro Preview |  |  | 1.0M | 65.5K | T/I | 0 | 0 |
| `google/palm-2-chat-bison` | Google: PaLM 2 Chat |  |  | 9.2K | - | T | 0 | 0 |
| `google/palm-2-chat-bison-32k` | Google: PaLM 2 Chat 32k |  |  | 32.8K | - | T | 0 | 0 |
| `google/palm-2-codechat-bison` | Google: PaLM 2 Code Chat |  |  | 7.2K | - | T | 0 | 0 |
| `google/palm-2-codechat-bison-32k` | Google: PaLM 2 Code Chat 32k |  |  | 32.8K | - | T | 0 | 0 |
| `google/veo-3.1` | Google: Veo 3.1 |  |  | 0 | - | T/I | 0 | 0 |
| `google/veo-3.1-fast` | Google: Veo 3.1 Fast |  |  | 0 | - | T/I | 0 | 0 |
| `google/veo-3.1-lite` | Google: Veo 3.1 Lite |  |  | 0 | - | T/I | 0 | 0 |

---

## Gryphe <a id="gryphe"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `gryphe/mythomax-l2-13b` | MythoMax 13B |  |  | 4.1K | 4.1K | T | $0.060 | $0.060 |
| `gryphe/mythomist-7b` | MythoMist 7B |  |  | 32.8K | - | T | 0 | 0 |

---

## Hexgrad <a id="hexgrad"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `hexgrad/kokoro-82m` | hexgrad: Kokoro 82M |  |  | 4.1K | - | T | $0.620 | 0 |

---

## Huggingfaceh4 <a id="huggingfaceh4"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `huggingfaceh4/zephyr-7b-beta` | Hugging Face: Zephyr 7B |  |  | 4.1K | - | T | 0 | 0 |
| `huggingfaceh4/zephyr-orpo-141b-a35b` | Zephyr 141B-A35B |  |  | 65.5K | - | T | 0 | 0 |

---

## Ibm Granite <a id="ibm-granite"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `ibm-granite/granite-4.0-h-micro` | IBM: Granite 4.0 Micro |  |  | 131K | 131K | T | $0.017 | $0.112 |
| `ibm-granite/granite-4.1-8b` | IBM: Granite 4.1 8B |  |  | 131.1K | 131.1K | T | $0.050 | $0.100 |

---

## Inception <a id="inception"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `inception/mercury` | Inception: Mercury |  |  | 128K | - | T | 0 | 0 |
| `inception/mercury-2` | Inception: Mercury 2 | 🧠 |  | 128K | 50K | T | $0.250 | $0.750 |
| `inception/mercury-coder` | Inception: Mercury Coder |  |  | 128K | - | T | 0 | 0 |

---

## Inclusionai <a id="inclusionai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `inclusionai/ling-2.6-1t` | inclusionAI: Ling-2.6-1T |  |  | 262.1K | 32.8K | T | $0.075 | $0.625 |
| `inclusionai/ling-2.6-flash` | inclusionAI: Ling-2.6-flash |  |  | 262.1K | 32.8K | T | $0.010 | $0.030 |
| `inclusionai/ring-2.6-1t` | inclusionAI: Ring-2.6-1T | ⚠️ |  | 262.1K | 65.5K | T | $0.075 | $0.625 |

---

## Infermatic <a id="infermatic"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `infermatic/mn-inferor-12b` | Infermatic: Mistral Nemo Inferor 12B |  |  | 32K | - | T | 0 | 0 |

---

## Inflatebot <a id="inflatebot"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `inflatebot/mn-mag-mell-r1` | Inflatebot: Mag Mell R1 12B |  |  | 32K | - | T | 0 | 0 |

---

## Inflection <a id="inflection"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `inflection/inflection-3-pi` | Inflection: Inflection 3 Pi |  |  | 8K | 1.0K | T | $2.50 | $10.00 |
| `inflection/inflection-3-productivity` | Inflection: Inflection 3 Productivity |  |  | 8K | 1.0K | T | $2.50 | $10.00 |

---

## Intel <a id="intel"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `intel/neural-chat-7b` | Neural Chat 7B v3.1 |  |  | 4.1K | - | T | 0 | 0 |

---

## Intfloat <a id="intfloat"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `intfloat/e5-base-v2` | Intfloat: E5-Base-v2 |  |  | 8.2K | - | T | $0.0050 | 0 |
| `intfloat/e5-large-v2` | Intfloat: E5-Large-v2 |  |  | 8.2K | - | T | $0.010 | 0 |
| `intfloat/multilingual-e5-large` | Intfloat: Multilingual-E5-Large |  |  | 8.2K | - | T | $0.010 | 0 |

---

## Jebcarter <a id="jebcarter"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `jebcarter/psyfighter-13b` | Psyfighter 13B |  |  | 4.1K | - | T | 0 | 0 |

---

## Jondurbin <a id="jondurbin"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `jondurbin/airoboros-l2-70b` | Airoboros 70B |  |  | 4.1K | - | T | 0 | 0 |
| `jondurbin/bagel-34b` | Bagel 34B v0.2 |  |  | 200K | - | T | 0 | 0 |

---

## Koboldai <a id="koboldai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `koboldai/psyfighter-13b-2` | Psyfighter v2 13B |  |  | 4.1K | - | T | 0 | 0 |

---

## Kwaipilot <a id="kwaipilot"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `kwaipilot/kat-coder-air-v2.5` | Kwaipilot: KAT-Coder-Air V2.5 |  |  | 256K | 80K | T | $0.150 | $0.600 |
| `kwaipilot/kat-coder-pro` | Kwaipilot: KAT-Coder-Pro V1 |  |  | 262.1K | - | T | 0 | 0 |
| `kwaipilot/kat-coder-pro-v2` | Kwaipilot: KAT-Coder-Pro V2 |  |  | 256K | 80K | T | $0.300 | $1.20 |
| `kwaipilot/kat-coder-pro-v2.5` | Kwaipilot: KAT-Coder-Pro V2.5 |  |  | 256K | 80K | T | $0.740 | $2.96 |

---

## Kwaivgi <a id="kwaivgi"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `kwaivgi/kling-v3.0-pro` | Kling: Video v3.0 Pro |  |  | 0 | - | T/I | 0 | 0 |
| `kwaivgi/kling-v3.0-std` | Kling: Video v3.0 Standard |  |  | 0 | - | T/I | 0 | 0 |
| `kwaivgi/kling-video-o1` | Kling: Video O1 |  |  | 0 | - | T/I | 0 | 0 |

---

## Liquid <a id="liquid"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `liquid/lfm-2-24b-a2b` | LiquidAI: LFM2-24B-A2B |  |  | 128K | - | T | 0 | 0 |
| `liquid/lfm-2.2-6b` | LiquidAI: LFM2-2.6B |  |  | 32.8K | - | T | 0 | 0 |
| `liquid/lfm-2.5-1.2b-instruct` | LiquidAI: LFM2.5-1.2B-Instruct |  |  | 32.8K | - | T | 0 | 0 |
| `liquid/lfm-2.5-1.2b-thinking` | LiquidAI: LFM2.5-1.2B-Thinking | ⚠️ |  | 32.8K | - | T | 0 | 0 |
| `liquid/lfm-3b` | Liquid: LFM 3B |  |  | 32.8K | - | T | 0 | 0 |
| `liquid/lfm-40b` | Liquid: LFM 40B MoE |  |  | 32.8K | - | T | 0 | 0 |
| `liquid/lfm-7b` | Liquid: LFM 7B |  |  | 32.8K | - | T | 0 | 0 |
| `liquid/lfm2-8b-a1b` | LiquidAI: LFM2-8B-A1B |  |  | 8.2K | - | T | 0 | 0 |

---

## Liuhaotian <a id="liuhaotian"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `liuhaotian/llava-13b` | LLaVA 13B |  |  | 2.0K | - | T/I | 0 | 0 |

---

## Lizpreciatior <a id="lizpreciatior"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `lizpreciatior/lzlv-70b-fp16-hf` | lzlv 70B |  |  | 4.1K | - | T | 0 | 0 |

---

## Lynn <a id="lynn"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `lynn/soliloquy-l3` | Lynn: Llama 3 Soliloquy 8B v2 |  |  | 24.6K | - | T | 0 | 0 |
| `lynn/soliloquy-v3` | Lynn: Llama 3 Soliloquy 7B v3 32K |  |  | 32.8K | - | T | 0 | 0 |

---

## Mancer <a id="mancer"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `mancer/weaver` | Mancer: Weaver (alpha) |  |  | 8K | 2K | T | $0.500 | $0.750 |

---

## Mattshumer <a id="mattshumer"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `mattshumer/reflection-70b` | Reflection 70B |  |  | 131.1K | - | T | 0 | 0 |

---

## Meituan <a id="meituan"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `meituan/longcat-2.0` | Meituan: LongCat 2.0 | 🧠 |  | 1.0M | 262.1K | T | $0.300 | $1.20 |
| `meituan/longcat-flash-chat` | Meituan: LongCat Flash Chat |  |  | 131.1K | - | T | 0 | 0 |

---

## Meta <a id="meta"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `meta/muse-spark-1.1` | Meta: Muse Spark 1.1 | ⚠️ |  | 1.0M | - | T/I/V/file/A | $1.25 | $4.25 |

---

## Microsoft <a id="microsoft"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `microsoft/mai-ds-r1` | Microsoft: MAI DS R1 | 🧠 |  | 163.8K | - | T | 0 | 0 |
| `microsoft/mai-image-2.5` | Microsoft: MAI-Image-2.5 |  |  | 4.1K | 1.0K | T/I | $5.00 | 0 |
| `microsoft/mai-transcribe-1.5` | Microsoft: MAI-Transcribe 1.5 |  |  | 0 | - | A | $360000.00 | 0 |
| `microsoft/mai-voice-2` | Microsoft: MAI-Voice-2 |  |  | 0 | - | T | $22.00 | 0 |
| `microsoft/wizardlm-2-7b` | WizardLM-2 7B |  |  | 32K | - | T | 0 | 0 |
| `microsoft/wizardlm-2-8x22b` | WizardLM-2 8x22B |  |  | 65.5K | 8K | T | $0.620 | $0.620 |

---

## Migtissera <a id="migtissera"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `migtissera/synthia-70b` | Synthia 70B |  |  | 8.2K | - | T | 0 | 0 |

---

## Mistralai <a id="mistralai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `mistralai/magistral-medium-2506` | Mistral: Magistral Medium 2506 | 🧠 |  | 41.0K | - | T | 0 | 0 |
| `mistralai/magistral-small-2506` | Mistral: Magistral Small 2506 | 🧠 |  | 40K | - | T | 0 | 0 |
| `mistralai/ministral-14b-2512` | Mistral: Ministral 3 14B 2512 |  |  | 262.1K | - | T/I | $0.200 | $0.200 |
| `mistralai/ministral-3b` | Mistral: Ministral 3B |  |  | 128K | - | T | 0 | 0 |
| `mistralai/ministral-3b-2512` | Mistral: Ministral 3 3B 2512 |  |  | 131.1K | - | T/I | $0.100 | $0.100 |
| `mistralai/ministral-8b` | Mistral: Ministral 8B |  |  | 128K | - | T | 0 | 0 |
| `mistralai/ministral-8b-2512` | Mistral: Ministral 3 8B 2512 |  |  | 262.1K | - | T/I | $0.150 | $0.150 |
| `mistralai/voxtral-mini-transcribe` | Mistral: Voxtral Mini Transcribe |  |  | 0 | - | A | $3000.00 | 0 |
| `mistralai/voxtral-mini-tts-2603` | Mistral: Voxtral Mini TTS |  |  | 4.1K | - | T | $16.00 | 0 |
| `mistralai/voxtral-small-24b-2507` | Mistral: Voxtral Small 24B 2507 |  |  | 32K | - | T/A/file | $0.100 | $0.300 |

---

## Moonshotai <a id="moonshotai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `moonshotai/moonlight-16b-a3b-instruct` | MoonshotAI: Moonlight 16B A3B Instruct |  |  | 8.2K | - | T | 0 | 0 |

---

## Morph <a id="morph"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `morph/morph-v2` | Morph: Fast Apply |  |  | 32K | - | T | 0 | 0 |
| `morph/morph-v3-fast` | Morph: Morph V3 Fast |  |  | 81.9K | 38K | T | $0.800 | $1.20 |
| `morph/morph-v3-large` | Morph: Morph V3 Large |  |  | 262.1K | 131.1K | T | $0.900 | $1.90 |

---

## Neversleep <a id="neversleep"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `neversleep/noromaid-20b` | Noromaid 20B |  |  | 8.2K | - | T | 0 | 0 |

---

## Nex Agi <a id="nex-agi"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `nex-agi/nex-n2-mini` | Nex AGI: Nex-N2-Mini | 🧠 |  | 262.1K | 262.1K | T/I | $0.025 | $0.100 |
| `nex-agi/nex-n2-pro` | Nex AGI: Nex-N2-Pro | 🧠 |  | 262.1K | 262.1K | T/I | $0.250 | $1.00 |

---

## Nothingiisreal <a id="nothingiisreal"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `nothingiisreal/mn-celeste-12b` | Mistral Nemo 12B Celeste |  |  | 32K | - | T | 0 | 0 |

---

## Nousresearch <a id="nousresearch"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `nousresearch/hermes-4-405b` | Nous: Hermes 4 405B | 🧠 |  | 131.1K | - | T | $1.00 | $3.00 |
| `nousresearch/hermes-4-70b` | Nous: Hermes 4 70B | 🧠 |  | 131.1K | - | T | $0.130 | $0.400 |
| `nousresearch/nous-capybara-34b` | Nous: Capybara 34B |  |  | 200K | - | T | 0 | 0 |
| `nousresearch/nous-capybara-7b` | Nous: Capybara 7B |  |  | 8.2K | - | T | 0 | 0 |
| `nousresearch/nous-hermes-2-vision-7b` | Nous: Hermes 2 Vision 7B (alpha) |  |  | 4.1K | - | T/I | 0 | 0 |

---

## Nvidia <a id="nvidia"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `nvidia/parakeet-tdt-0.6b-v3` | NVIDIA: Parakeet TDT 0.6B v3 |  |  | 0 | - | A | $1500.00 | 0 |

---

## Open R1 <a id="open-r1"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `open-r1/olympiccoder-32b` | OlympicCoder 32B | 🧠 |  | 32.8K | - | T | 0 | 0 |

---

## Openai <a id="openai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `openai/codex-mini` | OpenAI: Codex Mini | 🧠 |  | 200K | - | I/T | 0 | 0 |
| `openai/o1` | OpenAI: o1 | 🧠 |  | 200K | 100K | T/I/file | $15.00 | $60.00 |
| `openai/o3` | OpenAI: o3 | 🧠 |  | 200K | 100K | I/T/file | $2.00 | $8.00 |
| `openai/sora-2-pro` | OpenAI: Sora 2 Pro |  |  | 0 | - | T/I | 0 | 0 |
| `openai/text-embedding-3-large` | OpenAI: Text Embedding 3 Large |  |  | 8.2K | - | T | $0.130 | 0 |
| `openai/text-embedding-3-small` | OpenAI: Text Embedding 3 Small |  |  | 8.2K | - | T | $0.020 | 0 |
| `openai/text-embedding-ada-002` | OpenAI: Text Embedding Ada 002 |  |  | 8.2K | - | T | $0.100 | 0 |
| `openai/whisper-1` | OpenAI: Whisper 1 |  |  | 0 | - | A | $6000.00 | 0 |
| `openai/whisper-large-v3` | OpenAI: Whisper Large V3 |  |  | 0 | - | A | $1500.00 | 0 |
| `openai/whisper-large-v3-turbo` | OpenAI: Whisper Large V3 Turbo |  |  | 0 | - | A | $40000.00 | 0 |

---

## Openchat <a id="openchat"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `openchat/openchat-7b` | OpenChat 3.5 7B |  |  | 8.2K | - | T | 0 | 0 |
| `openchat/openchat-8b` | OpenChat 3.6 8B |  |  | 8.2K | - | T | 0 | 0 |

---

## Opengvlab <a id="opengvlab"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `opengvlab/internvl3-14b` | OpenGVLab: InternVL3 14B |  |  | 32K | - | I/T | 0 | 0 |
| `opengvlab/internvl3-2b` | OpenGVLab: InternVL3 2B |  |  | 32K | - | I/T | 0 | 0 |
| `opengvlab/internvl3-78b` | OpenGVLab: InternVL3 78B |  |  | 0 | - | I/T | 0 | 0 |

---

## Openrouter <a id="openrouter"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `openrouter/andromeda-alpha` | Andromeda Alpha | 🧠 |  | 128K | - | I/T | 0 | 0 |
| `openrouter/aurora-alpha` | Aurora Alpha | ⚠️ |  | 128K | - | T | 0 | 0 |
| `openrouter/auto` | Auto Router |  |  | 2M | - | T/I/A/file/V | 0 | 0 |
| `openrouter/auto-beta` | Auto Router (Beta) |  |  | 2M | - | T/I/A/file/V | 0 | 0 |
| `openrouter/bert-nebulon-alpha` | Bert-Nebulon Alpha |  |  | 256K | - | T/I | 0 | 0 |
| `openrouter/bodybuilder` | Body Builder (beta) |  |  | 128K | - | T | 0 | 0 |
| `openrouter/cinematika-7b` | Cinematika 7B (alpha) |  |  | 8K | - | T | 0 | 0 |
| `openrouter/cypher-alpha` | Cypher Alpha | 🧠 |  | 1M | - | T | 0 | 0 |
| `openrouter/elephant-alpha` | Elephant Alpha |  |  | 262.1K | - | T | 0 | 0 |
| `openrouter/free` | Free Models Router | 🧠 |  | 200K | - | T/I | 0 | 0 |
| `openrouter/fusion` | OpenRouter: Fusion |  |  | 1M | - | T | 0 | 0 |
| `openrouter/healer-alpha` | Healer Alpha | 🧠 |  | 262.1K | - | T/I/A/V | 0 | 0 |
| `openrouter/horizon-alpha` | Horizon Alpha |  |  | 256K | - | I/T | 0 | 0 |
| `openrouter/horizon-beta` | Horizon Beta |  |  | 256K | - | I/T | 0 | 0 |
| `openrouter/hunter-alpha` | Hunter Alpha | 🧠 |  | 1.0M | - | T/I | 0 | 0 |
| `openrouter/optimus-alpha` | Optimus Alpha |  |  | 1M | - | I/T | 0 | 0 |
| `openrouter/owl-alpha` | Owl Alpha |  |  | 1.0M | - | T | 0 | 0 |
| `openrouter/pareto-code` | Pareto Code Router |  |  | 2M | - | T | 0 | 0 |
| `openrouter/polaris-alpha` | Polaris Alpha |  |  | 256K | - | T/I | 0 | 0 |
| `openrouter/pony-alpha` | Pony Alpha | 🧠 |  | 200K | - | T | 0 | 0 |
| `openrouter/quasar-alpha` | Quasar Alpha |  |  | 1M | - | I/T | 0 | 0 |
| `openrouter/sherlock-dash-alpha` | Sherlock Dash Alpha |  |  | 1.8M | - | T/I | 0 | 0 |
| `openrouter/sherlock-think-alpha` | Sherlock Think Alpha | 🧠 |  | 1.8M | - | T/I | 0 | 0 |
| `openrouter/sonoma-dusk-alpha` | Sonoma Dusk Alpha |  |  | 2M | - | T/I | 0 | 0 |
| `openrouter/sonoma-sky-alpha` | Sonoma Sky Alpha | 🧠 |  | 2M | - | T/I | 0 | 0 |

---

## Perceptron <a id="perceptron"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `perceptron/perceptron-mk1` | Perceptron: Perceptron Mk1 | 🧠 |  | 32.8K | 8.2K | T/I/V | $0.150 | $1.50 |

---

## Perplexity <a id="perplexity"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `perplexity/pplx-embed-v1-0.6b` | Perplexity: Embed V1 0.6B |  |  | 32K | - | T | $0.0040 | 0 |
| `perplexity/pplx-embed-v1-4b` | Perplexity: Embed V1 4B |  |  | 32K | - | T | $0.030 | 0 |
| `perplexity/r1-1776` | Perplexity: R1 1776 | 🧠 |  | 128K | - | T | 0 | 0 |
| `perplexity/sonar` | Perplexity: Sonar |  |  | 127.1K | - | T/I | $1.00 | $1.00 |
| `perplexity/sonar-deep-research` | Perplexity: Sonar Deep Research | 🧠 |  | 128K | - | T | $2.00 | $8.00 |
| `perplexity/sonar-pro` | Perplexity: Sonar Pro |  |  | 200K | 8K | T/I | $3.00 | $15.00 |
| `perplexity/sonar-pro-search` | Perplexity: Sonar Pro Search | ⚠️ |  | 200K | 8K | T/I | $3.00 | $15.00 |
| `perplexity/sonar-reasoning` | Perplexity: Sonar Reasoning | 🧠 |  | 127K | - | T | 0 | 0 |
| `perplexity/sonar-reasoning-pro` | Perplexity: Sonar Reasoning Pro | 🧠 |  | 128K | - | T/I | $2.00 | $8.00 |

---

## Poolside <a id="poolside"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `poolside/laguna-m.1` | Poolside: Laguna M.1 (free) | 🧠 | 🆓 | 262.1K | 32.8K | T | 0 | 0 |
| `poolside/laguna-m.1` | Poolside: Laguna M.1 | 🧠 |  | 262.1K | 32.8K | T | $0.200 | $0.400 |
| `poolside/laguna-xs-2.1` | Poolside: Laguna XS 2.1 (free) | 🧠 | 🆓 | 262.1K | 32.8K | T | 0 | 0 |
| `poolside/laguna-xs-2.1` | Poolside: Laguna XS 2.1 | 🧠 |  | 262.1K | 32.8K | T | $0.060 | $0.120 |
| `poolside/laguna-xs.2` | Poolside: Laguna XS.2 | 🧠 |  | 262.1K | - | T | 0 | 0 |

---

## Prime Intellect <a id="prime-intellect"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `prime-intellect/intellect-3` | Prime Intellect: INTELLECT-3 | ⚠️ |  | 131.1K | - | T | 0 | 0 |

---

## Pygmalionai <a id="pygmalionai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `pygmalionai/mythalion-13b` | Pygmalion: Mythalion 13B |  |  | 8.2K | - | T | 0 | 0 |

---

## Raifle <a id="raifle"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `raifle/sorcererlm-8x22b` | SorcererLM 8x22B |  |  | 16K | - | T | 0 | 0 |

---

## Recraft <a id="recraft"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `recraft/recraft-v3` | Recraft: Recraft V3 |  |  | 65.5K | - | T/I | 0 | 0 |
| `recraft/recraft-v4` | Recraft: Recraft V4 |  |  | 65.5K | - | T/I | 0 | 0 |
| `recraft/recraft-v4-pro` | Recraft: Recraft V4 Pro |  |  | 65.5K | - | T/I | 0 | 0 |
| `recraft/recraft-v4-pro-vector` | Recraft: Recraft V4 Pro Vector |  |  | 65.5K | - | T/I | 0 | 0 |
| `recraft/recraft-v4-vector` | Recraft: Recraft V4 Vector |  |  | 65.5K | - | T/I | 0 | 0 |
| `recraft/recraft-v4.1` | Recraft: Recraft V4.1 |  |  | 65.5K | - | T/I | 0 | 0 |
| `recraft/recraft-v4.1-pro` | Recraft: Recraft V4.1 Pro |  |  | 65.5K | - | T/I | 0 | 0 |
| `recraft/recraft-v4.1-pro-vector` | Recraft: Recraft V4.1 Pro Vector |  |  | 65.5K | - | T/I | 0 | 0 |
| `recraft/recraft-v4.1-utility` | Recraft: Recraft V4.1 Utility |  |  | 65.5K | - | T/I | 0 | 0 |
| `recraft/recraft-v4.1-utility-pro` | Recraft: Recraft V4.1 Utility Pro |  |  | 65.5K | - | T/I | 0 | 0 |
| `recraft/recraft-v4.1-vector` | Recraft: Recraft V4.1 Vector |  |  | 65.5K | - | T/I | 0 | 0 |

---

## Recursal <a id="recursal"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `recursal/eagle-7b` | RWKV v5: Eagle 7B |  |  | 10K | - | T | 0 | 0 |

---

## Rekaai <a id="rekaai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `rekaai/reka-edge` | Reka Edge | 🧠 |  | 16.4K | 16.4K | I/T/V | $0.100 | $0.100 |
| `rekaai/reka-flash-3` | Reka Flash 3 | ⚠️ |  | 65.5K | 65.5K | T | $0.100 | $0.200 |

---

## Relace <a id="relace"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `relace/relace-apply-3` | Relace: Relace Apply 3 |  |  | 256K | 128K | T | $0.850 | $1.25 |
| `relace/relace-search` | Relace: Relace Search |  |  | 256K | 128K | T | $1.00 | $3.00 |

---

## Rwkv <a id="rwkv"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `recursal/rwkv-5-3b-ai-town` | RWKV v5 3B AI Town |  |  | 10K | - | T | 0 | 0 |
| `rwkv/rwkv-5-world-3b` | RWKV v5 World 3B |  |  | 10K | - | T | 0 | 0 |

---

## Sakana <a id="sakana"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `sakana/fugu-ultra` | Sakana: Fugu Ultra | ⚠️ |  | 1M | 128K | T/I | $5.00 | $30.00 |

---

## Sao10K <a id="sao10k"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `sao10k/fimbulvetr-11b-v2` | Fimbulvetr 11B v2 |  |  | 8.2K | - | T | 0 | 0 |
| `sao10k/l3-euryale-70b` | Sao10k: Llama 3 Euryale 70B v2.1 |  |  | 8.2K | - | T | 0 | 0 |
| `sao10k/l3-lunaris-8b` | Sao10K: Llama 3 8B Lunaris |  |  | 8.2K | 16.4K | T | $0.040 | $0.050 |
| `sao10k/l3-stheno-8b` | Sao10K: Llama 3 Stheno 8B v3.3 32K |  |  | 32K | - | T | 0 | 0 |
| `sao10k/l3.1-70b-hanami-x1` | Sao10K: Llama 3.1 70B Hanami x1 |  |  | 16K | - | T | 0 | 0 |
| `sao10k/l3.1-euryale-70b` | Sao10K: Llama 3.1 Euryale 70B v2.2 |  |  | 131.1K | 16.4K | T | $0.850 | $0.850 |
| `sao10k/l3.3-euryale-70b` | Sao10K: Llama 3.3 Euryale 70B |  |  | 131.1K | 16.4K | T | $0.650 | $0.750 |

---

## Sarvamai <a id="sarvamai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `sarvamai/sarvam-m` | Sarvam AI: Sarvam-M |  |  | 32.8K | - | T | 0 | 0 |

---

## Sentence Transformers <a id="sentence-transformers"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `sentence-transformers/all-minilm-l12-v2` | Sentence Transformers: all-MiniLM-L12-v2 |  |  | 8.2K | - | T | $0.0050 | 0 |
| `sentence-transformers/all-minilm-l6-v2` | Sentence Transformers: all-MiniLM-L6-v2 |  |  | 8.2K | - | T | $0.0050 | 0 |
| `sentence-transformers/all-mpnet-base-v2` | Sentence Transformers: all-mpnet-base-v2 |  |  | 8.2K | - | T | $0.0050 | 0 |
| `sentence-transformers/multi-qa-mpnet-base-dot-v1` | Sentence Transformers: multi-qa-mpnet-base-dot-v1 |  |  | 8.2K | - | T | $0.0050 | 0 |
| `sentence-transformers/paraphrase-minilm-l6-v2` | Sentence Transformers: paraphrase-MiniLM-L6-v2 |  |  | 8.2K | - | T | $0.0050 | 0 |

---

## Sesame <a id="sesame"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `sesame/csm-1b` | Sesame: CSM 1B |  |  | 4.1K | - | T | $7.00 | 0 |

---

## Snowflake <a id="snowflake"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `snowflake/snowflake-arctic-instruct` | Snowflake: Arctic Instruct |  |  | 4.1K | - | T | 0 | 0 |

---

## Sophosympatheia <a id="sophosympatheia"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `sophosympatheia/midnight-rose-70b` | Midnight Rose 70B |  |  | 4.1K | - | T | 0 | 0 |

---

## Sourceful <a id="sourceful"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `sourceful/riverflow-v2-fast` | Sourceful: Riverflow V2 Fast |  |  | 8.2K | - | T/I | 0 | 0 |
| `sourceful/riverflow-v2-fast-preview` | Sourceful: Riverflow V2 Fast Preview |  |  | 8.2K | - | T/I | 0 | 0 |
| `sourceful/riverflow-v2-max-preview` | Sourceful: Riverflow V2 Max Preview |  |  | 8.2K | - | T/I | 0 | 0 |
| `sourceful/riverflow-v2-pro` | Sourceful: Riverflow V2 Pro |  |  | 8.2K | - | T/I | 0 | 0 |
| `sourceful/riverflow-v2-standard-preview` | Sourceful: Riverflow V2 Standard Preview |  |  | 8.2K | - | T/I | 0 | 0 |
| `sourceful/riverflow-v2.5-fast` | Sourceful: Riverflow V2.5 Fast | ⚠️ |  | 32.8K | - | T/I | 0 | 0 |
| `sourceful/riverflow-v2.5-pro` | Sourceful: Riverflow V2.5 Pro | ⚠️ |  | 32.8K | - | T/I | 0 | 0 |

---

## Steelskull <a id="steelskull"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `steelskull/l3.3-electra-r1-70b` | SteelSkull: L3.3 Electra R1 70B | 🧠 |  | 128K | - | T | 0 | 0 |

---

## Stepfun <a id="stepfun"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `stepfun/step-3.5-flash` | StepFun: Step 3.5 Flash | ⚠️ |  | 262.1K | 65.5K | T | $0.100 | $0.300 |
| `stepfun/step-3.7-flash` | StepFun: Step 3.7 Flash | ⚠️ |  | 256K | 256K | T/I/V | $0.200 | $1.15 |

---

## Stepfun Ai <a id="stepfun-ai"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `stepfun-ai/step3` | StepFun: Step3 | ⚠️ |  | 65.5K | - | I/T | 0 | 0 |

---

## Switchpoint <a id="switchpoint"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `switchpoint/router` | Switchpoint Router | 🧠 |  | 131.1K | - | T | 0 | 0 |

---

## Tencent <a id="tencent"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `tencent/hunyuan-a13b-instruct` | Tencent: Hunyuan A13B Instruct | 🧠 |  | 131.1K | 131.1K | T | $0.140 | $0.570 |
| `tencent/hy3` | Tencent: Hy3 (free) | 🧠 | 🆓 | 262.1K | 262.1K | T | 0 | 0 |
| `tencent/hy3` | Tencent: Hy3 | 🧠 |  | 262.1K | 128K | T | $0.132 | $0.528 |
| `tencent/hy3-preview` | Tencent: Hy3 preview | 🧠 |  | 262.1K | - | T | $0.063 | $0.210 |

---

## Thedrummer <a id="thedrummer"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `thedrummer/cydonia-24b-v4.1` | TheDrummer: Cydonia 24B V4.1 |  |  | 131.1K | 131.1K | T | $0.300 | $0.500 |
| `thedrummer/rocinante-12b` | TheDrummer: Rocinante 12B |  |  | 65.5K | 65.5K | T | $0.250 | $0.500 |
| `thedrummer/skyfall-36b-v2` | TheDrummer: Skyfall 36B V2 |  |  | 32.8K | 32.8K | T | $0.550 | $0.800 |
| `thedrummer/unslopnemo-12b` | TheDrummer: UnslopNemo 12B |  |  | 32.8K | 32.8K | T | $0.400 | $0.400 |
| `thedrummer/valkyrie-49b-v1` | TheDrummer: Valkyrie 49B V1 | 🧠 |  | 131.1K | - | T | 0 | 0 |

---

## Thenlper <a id="thenlper"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `thenlper/gte-base` | Thenlper: GTE-Base |  |  | 8.2K | - | T | $0.0050 | 0 |
| `thenlper/gte-large` | Thenlper: GTE-Large |  |  | 8.2K | - | T | $0.010 | 0 |

---

## Thinkingmachines <a id="thinkingmachines"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `thinkingmachines/inkling` | Thinking Machines: Inkling | 🧠 |  | 1.0M | - | T/I/A | $1.00 | $4.05 |

---

## Tngtech <a id="tngtech"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `tngtech/tng-r1t-chimera` | TNG: R1T Chimera | ⚠️ |  | 163.8K | - | T | 0 | 0 |

---

## Togethercomputer <a id="togethercomputer"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `togethercomputer/stripedhyena-hessian-7b` | StripedHyena Hessian 7B (base) |  |  | 32.8K | - | T | 0 | 0 |
| `togethercomputer/stripedhyena-nous-7b` | StripedHyena Nous 7B |  |  | 32.8K | - | T | 0 | 0 |

---

## Undi95 <a id="undi95"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `undi95/remm-slerp-l2-13b` | ReMM SLERP 13B |  |  | 6.1K | 2.0K | T | $0.450 | $0.650 |
| `undi95/toppy-m-7b` | Toppy M 7B |  |  | 4.1K | - | T | 0 | 0 |

---

## Upstage <a id="upstage"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `upstage/solar-pro-3` | Upstage: Solar Pro 3 | 🧠 |  | 128K | - | T | $0.150 | $0.600 |

---

## Writer <a id="writer"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `writer/palmyra-x5` | Writer: Palmyra X5 |  |  | 1.0M | 8.2K | T | $0.600 | $6.00 |

---

## Xiaomi <a id="xiaomi"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `xiaomi/mimo-v2-flash` | Xiaomi: MiMo-V2-Flash | 🧠 |  | 262.1K | - | T | 0 | 0 |
| `xiaomi/mimo-v2-omni` | Xiaomi: MiMo-V2-Omni | 🧠 |  | 262.1K | - | T/A/I/V | 0 | 0 |
| `xiaomi/mimo-v2-pro` | Xiaomi: MiMo-V2-Pro | 🧠 |  | 1.0M | - | T | 0 | 0 |
| `xiaomi/mimo-v2.5` | Xiaomi: MiMo-V2.5 | 🧠 |  | 1.0M | - | T/A/I/V | $0.105 | $0.280 |
| `xiaomi/mimo-v2.5-pro` | Xiaomi: MiMo-V2.5-Pro | 🧠 |  | 1.1M | - | T | $0.348 | $0.696 |

---

## Xwin Lm <a id="xwin-lm"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `xwin-lm/xwin-lm-70b` | Xwin 70B |  |  | 8.2K | - | T | 0 | 0 |

---

## Yi <a id="yi"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `01-ai/yi-1.5-34b-chat` | Yi 1.5 34B Chat |  |  | 4.1K | - | T | 0 | 0 |
| `01-ai/yi-34b` | Yi 34B (base) |  |  | 4.1K | - | T | 0 | 0 |
| `01-ai/yi-34b-200k` | Yi 34B 200K |  |  | 200K | - | T | 0 | 0 |
| `01-ai/yi-34b-chat` | Yi 34B Chat |  |  | 4.1K | - | T | 0 | 0 |
| `01-ai/yi-6b` | Yi 6B (base) |  |  | 4.1K | - | T | 0 | 0 |
| `01-ai/yi-large` | 01.AI: Yi Large |  |  | 32.8K | - | T | 0 | 0 |
| `01-ai/yi-large-fc` | 01.AI: Yi Large FC |  |  | 16.4K | - | T | 0 | 0 |
| `01-ai/yi-large-turbo` | 01.AI: Yi Large Turbo |  |  | 4.1K | - | T | 0 | 0 |
| `01-ai/yi-vision` | 01.AI: Yi Vision |  |  | 16.4K | - | T/I | 0 | 0 |
| `alibaba/tongyi-deepresearch-30b-a3b` | Tongyi DeepResearch 30B A3B | 🧠 |  | 131.1K | - | T | 0 | 0 |
| `liuhaotian/llava-yi-34b` | LLaVA v1.6 34B |  |  | 4.1K | - | T/I | 0 | 0 |
| `nousresearch/nous-hermes-yi-34b` | Nous: Hermes 2 Yi 34B |  |  | 4.1K | - | T | 0 | 0 |

---

## Zyphra <a id="zyphra"></a>

| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |
|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|
| `zyphra/zonos-v0.1-hybrid` | Zyphra: Zonos v0.1 Hybrid |  |  | 4.1K | - | T | $7.00 | 0 |
| `zyphra/zonos-v0.1-transformer` | Zyphra: Zonos v0.1 Transformer |  |  | 4.1K | - | T | $7.00 | 0 |

---

## Summary Statistics

### Context Length Distribution

| Range | Count |
|-------|-------|
| 1M+ | 114 |
| 500K-1M | 2 |
| 200K-500K | 157 |
| 100K-200K | 207 |
| 32K-100K | 145 |
| <32K | 178 |

### Input Modality Distribution

| Modalities | Count |
|------------|-------|
| text | 474 |
| image/text | 158 |
| file/image/text | 87 |
| image/text/video | 28 |
| audio/file/image/text/video | 22 |
| audio | 11 |
| file/text | 11 |
| audio/image/text/video | 6 |
| audio/text | 2 |
| audio/image/text | 1 |
| file/image/text/video | 1 |
| audio/file/text | 1 |
| audio/file/image/text | 1 |
