# 配置详解

## 配置文件位置

VibeCoding 使用两个配置文件:

| 文件 | 范围 | 优先级 |
|------|------|--------|
| `~/.vibecoding/settings.json` | 全局 (所有项目) | 低 |
| `.vibe/settings.json` | 项目级 | 高 |

项目级配置会覆盖全局配置。

## 配置结构

### 完整示例

```json
{
  "providers": {
    "anthropic": {
      "baseUrl": "https://api.anthropic.com",
      "apiKey": "sk-ant-...",
      "api": "anthropic-messages",
      "models": [
        {
          "id": "claude-sonnet-4-20250514",
          "name": "Claude Sonnet 4",
          "contextWindow": 200000,
          "maxTokens": 8192,
          "reasoning": true
        }
      ]
    },
    "openai": {
      "baseUrl": "https://api.openai.com/v1",
      "apiKey": "sk-...",
      "api": "openai-chat",
      "models": [
        {
          "id": "gpt-4o",
          "name": "GPT-4o",
          "contextWindow": 128000,
          "maxTokens": 16384
        }
      ]
    },
    "my-custom": {
      "baseUrl": "https://my-api.example.com/v1",
      "api": "openai-chat",
      "models": []
    }
  },
  "defaultProvider": "anthropic",
  "defaultModel": "claude-sonnet-4-20250514",
  "defaultMode": "agent",
  "defaultThinkingLevel": "medium",
  "maxOutputTokens": 8192,
  "maxContextTokens": 200000,
  "compaction": {
    "enabled": true,
    "reserveTokens": 16384,
    "keepRecentTokens": 20000
  },
  "sandbox": {
    "enabled": true,
    "level": "standard",
    "allowNetwork": false
  },
  "contextFiles": {
    "enabled": true,
    "extraFiles": [
      "/path/to/extra-context.md"
    ]
  },
  "skills": {
    "enabled": true,
    "dirs": [
      "~/.vibecoding/skills",
      ".skills"
    ]
  }
}
```

## 配置项详解

### providers

多提供商配置。每个提供商包含:

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| `baseUrl` | string | ✓ | API 基础 URL |
| `apiKey` | string | - | API 密钥 (可选，也可通过环境变量) |
| `api` | string | - | API 类型: `openai-chat` 或 `anthropic-messages` |
| `models` | array | - | 可用模型列表 |

#### api 字段

- `openai-chat`: OpenAI Chat Completions API 格式
- `anthropic-messages`: Anthropic Messages API 格式

如果未指定，会根据 `baseUrl` 自动检测:
- 包含 "anthropic" → `anthropic-messages`
- 其他 → `openai-chat`

#### models 数组

```json
{
  "id": "claude-sonnet-4-20250514",
  "name": "Claude Sonnet 4",
  "contextWindow": 200000,
  "maxTokens": 8192,
  "reasoning": true,
  "input": ["text", "image"],
  "cost": {
    "input": 3.0,
    "output": 15.0,
    "cacheRead": 0.3,
    "cacheWrite": 3.75
  }
}
```

| 字段 | 类型 | 描述 |
|------|------|------|
| `id` | string | 模型 ID |
| `name` | string | 显示名称 |
| `contextWindow` | int | 上下文窗口大小 (token) |
| `maxTokens` | int | 最大输出 token |
| `reasoning` | bool | 是否支持推理/思考 |
| `input` | []string | 支持的输入类型 (text, image) |
| `cost` | object | 定价 (每百万 token) |

### defaultProvider

默认使用的提供商名称。对应 `providers` 中的键名。

```json
{
  "defaultProvider": "anthropic"
}
```

### defaultModel

默认使用的模型 ID。

```json
{
  "defaultModel": "claude-sonnet-4-20250514"
}
```

### defaultMode

默认运行模式。

```json
{
  "defaultMode": "agent"
}
```

可选值:
- `plan`: 只读分析模式
- `agent`: 标准读写模式 (默认)
- `yolo`: 完全访问模式

### defaultThinkingLevel

默认思考级别。

```json
{
  "defaultThinkingLevel": "medium"
}
```

可选值:
- `off`: 关闭思考
- `minimal`: 最小思考
- `low`: 低级别
- `medium`: 中等级别
- `high`: 高级别
- `xhigh`: 最高级别

### maxOutputTokens

最大输出 token 数量。

```json
{
  "maxOutputTokens": 8192
}
```

### maxContextTokens

最大上下文 token 数量。

```json
{
  "maxContextTokens": 200000
}
```

### compaction

上下文压缩配置，用于管理长对话。

```json
{
  "compaction": {
    "enabled": true,
    "reserveTokens": 16384,
    "keepRecentTokens": 20000
  }
}
```

| 字段 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `enabled` | bool | true | 是否启用压缩 |
| `reserveTokens` | int | 16384 | 为模型响应保留的 token |
| `keepRecentTokens` | int | 20000 | 保留的最近消息 token |

### sandbox

沙箱配置。

```json
{
  "sandbox": {
    "enabled": true,
    "level": "standard",
    "allowNetwork": false
  }
}
```

| 字段 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `enabled` | bool | false | 是否启用沙箱 |
| `level` | string | standard | 沙箱级别 (none, standard, strict) |
| `allowNetwork` | bool | false | 是否允许网络访问 |

### contextFiles

上下文文件配置。

```json
{
  "contextFiles": {
    "enabled": true,
    "extraFiles": [
      "/path/to/extra-context.md",
      "~/.vibecoding/global-context.md"
    ]
  }
}
```

| 字段 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `enabled` | bool | true | 是否自动加载上下文文件 |
| `extraFiles` | []string | [] | 额外的上下文文件路径 |

#### 自动加载的上下文文件

VibeCoding 会自动搜索并加载以下文件:

1. **全局文件** (在 `~/.vibecoding/`):
   - `AGENTS.md`
   - `CLAUDE.md`

2. **项目文件** (从当前目录向上搜索):
   - `AGENTS.md`
   - `CLAUDE.md`
   - `.vibe/AGENTS.md`
   - `.vibe/CLAUDE.md`

### skills

技能系统配置。

```json
{
  "skills": {
    "enabled": true,
    "dirs": [
      "~/.vibecoding/skills",
      ".skills"
    ]
  }
}
```

## 认证配置

### 方式一: 环境变量

```bash
export ANTHROPIC_API_KEY=sk-ant-...
export OPENAI_API_KEY=sk-...
```

### 方式二: 认证文件

创建 `~/.vibecoding/auth.json`:

```json
{
  "anthropic": {
    "type": "api_key",
    "key": "sk-ant-..."
  },
  "openai": {
    "type": "api_key",
    "key": "sk-..."
  }
}
```

### 方式三: 配置文件内嵌

在 `settings.json` 的 providers 中直接配置:

```json
{
  "providers": {
    "anthropic": {
      "apiKey": "sk-ant-..."
    }
  }
}
```

### 密钥解析顺序

1. 环境变量 (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`)
2. 认证文件 (`~/.vibecoding/auth.json`)
3. 配置文件内嵌 (`settings.json`)

## 环境变量覆盖

可以通过环境变量覆盖任何设置:

| 环境变量 | 覆盖的配置 |
|----------|-----------|
| `VIBECODING_DIR` | 配置目录 |
| `VIBECODING_PROVIDER` | defaultProvider |
| `VIBECODING_MODEL` | defaultModel |
| `VIBECODING_MODE` | defaultMode |
| `VIBECODING_THINKING` | defaultThinkingLevel |

## 配置示例

### 最小配置

```json
{
  "defaultProvider": "anthropic",
  "defaultModel": "claude-sonnet-4-20250514"
}
```

### 多提供商配置

```json
{
  "providers": {
    "anthropic": {
      "baseUrl": "https://api.anthropic.com",
      "api": "anthropic-messages"
    },
    "openai": {
      "baseUrl": "https://api.openai.com/v1",
      "api": "openai-chat"
    }
  },
  "defaultProvider": "anthropic",
  "defaultModel": "claude-sonnet-4-20250514"
}
```

### 自定义 API 端点

```json
{
  "providers": {
    "my-proxy": {
      "baseUrl": "https://my-proxy.example.com/v1",
      "api": "openai-chat",
      "apiKey": "my-key",
      "models": [
        {
          "id": "gpt-4o",
          "name": "GPT-4o (via proxy)"
        }
      ]
    }
  },
  "defaultProvider": "my-proxy"
}
```

### 启用沙箱

```json
{
  "sandbox": {
    "enabled": true,
    "level": "standard"
  }
}
```
