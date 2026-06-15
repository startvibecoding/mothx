# 命令行参考

## 概述

```
vibecoding [flags] [message...]
```

别名: `vc`

## 命令行参数

### 基本参数

| 参数 | 简写 | 默认值 | 描述 |
|------|------|--------|------|
| `--provider` | `-p` | 配置文件中的默认值 | LLM 提供商 (deepseek-openai, deepseek-anthropic 或自定义名称) |
| `--model` | `-m` | 配置文件中的默认值 | 模型 ID |
| `--mode` | `-M` | `agent` | 运行模式 (plan, agent, yolo) |
| `--thinking` | `-t` | `off` | 思考级别 (off, minimal, low, medium, high, xhigh) |
| `--multi-agent` | - | `false` | 启用多 Agent 工具和命令 |
| `--delegate` | - | `false` | 启用 Delegate 模式（阻塞式单子 Agent 工具） |

### 会话管理

| 参数 | 简写 | 描述 |
|------|------|------|
| `--continue` | `-c` | 继续最近的会话 |
| `--resume` | `-r` | 通过 ID 或路径恢复会话 |
| `--session` | - | 使用特定的会话文件 |

### 输出控制

| 参数 | 简写 | 描述 |
|------|------|------|
| `--print` | `-P` | 非交互模式，打印响应后退出；如果工具调用需要审批，则直接报错退出，不会自动批准 |
| `--verbose` | - | 详细输出 |
| `--debug` | - | 启用调试日志（同时启用 provider 请求/响应调试输出） |

### 安全

| 参数 | 描述 |
|------|------|
| `--sandbox` | 启用沙箱 (bubblewrap) |
| `--no-sandbox` | 禁用沙箱 (已弃用，默认不启用) |

### 其他

| 参数 | 简写 | 描述 |
|------|------|------|
| `--init-gateway` | - | 生成 `gateway.json` 配置模板 |
| `--init-a2a-master-config` | - | 生成 `a2a-list.json` 配置模板 |
| `--enable-a2a-master` | - | 启用 A2A Master 模式（远程 agent 调度） |
| `--force` | - | 覆盖已存在的配置文件（配合 `--init-*` 使用） |
| `--version` | `-v` | 显示版本 |
| `--help` | `-h` | 显示帮助 |

## 子命令

### `acp` - Agent Client Protocol 服务器

以 ACP 兼容的 stdio 代理模式运行 VibeCoding，用于 IDE 集成。

```
vibecoding acp [flags]
```

支持 VS Code、JetBrains IDE 以及任何 ACP 兼容的编辑器。

| 标志 | 简写 | 默认值 | 描述 |
|------|------|--------|------|
| `--provider` | `-p` | 配置文件中的默认值 | LLM 提供商 |
| `--model` | `-m` | 配置文件中的默认值 | 模型 ID |
| `--mode` | `-M` | `agent` | 运行模式 (plan, agent, yolo) |
| `--thinking` | `-t` | 配置文件中的默认值 | 思考级别 |
| `--sandbox` | - | false | 启用沙箱 |
| `--verbose` | - | false | 详细输出 |
| `--debug` | - | false | 调试日志 |
| `--multi-agent` | - | false | 为 ACP 会话启用多 Agent 工具 |
| `--delegate` | - | false | 为 ACP 会话启用 Delegate 模式 |

详见 [ACP 协议](acp.md) 文档了解 IDE 集成细节。

### `a2a` - A2A 协议服务器

运行 A2A (Agent-to-Agent) 协议服务器，支持独立模式和集成模式。

```
vibecoding a2a [command]
```

| 子命令 | 描述 |
|--------|------|
| `start` | 启动 A2A 服务器 |
| `stop` | 停止 A2A 服务器 |
| `status` | 查看服务器状态 |
| `card` | 显示/生成 Agent Card |
| `send <message>` | 向远程 A2A 服务器发送任务 |
| `discover <url>` | 发现远程 Agent Card |
| `--init-a2a-config` | 生成 `a2a.json` 配置模板 |
| `--force` | 覆盖已存在的配置文件 |

详见 [A2A 协议](a2a.md) 文档。

### `gateway` - OpenAI 兼容 HTTP 网关

以 HTTP 服务器模式运行 VibeCoding，暴露标准 OpenAI Chat Completions API。

```
vibecoding gateway [flags]
```

| 标志 | 简写 | 默认值 | 描述 |
|------|------|--------|------|
| `--port` | - | `8080` | 监听端口（覆盖 gateway.json） |
| `--config` | - | - | gateway.json 路径 |
| `--work-dir` | - | 当前目录 | 默认工作目录 |
| `--provider` | `-p` | 配置文件中的默认值 | LLM 提供商 |
| `--model` | `-m` | 配置文件中的默认值 | 模型 ID |
| `--sandbox` | - | false | 启用沙箱 (bwrap) |
| `--multi-agent` | - | false | 启用多 Agent 工具 |
| `--delegate` | - | false | 启用 Delegate 模式 |
| `--verbose` | - | false | 详细输出 |
| `--debug` | - | false | 调试日志 |

详见 [Gateway 模式](gateway.md) 文档。

### `hermes` - 消息平台网关

运行 Hermes 消息平台网关，支持微信/飞书/WebSocket，提供持久化 Agent 会话。

```
vibecoding hermes [command]
```

| 子命令 | 描述 |
|--------|------|
| `start` | 启动 Hermes 服务器 |
| `client` | 以远程 TUI 客户端通过 WebSocket 连接 |
| `--init-hermes-config` | 生成 `hermes.json` 配置模板 |
| `--force` | 覆盖已存在的配置文件 |

详见 [Hermes 模式](hermes.md) 文档。

### `doctor` - 环境诊断

诊断你的 VibeCoding 环境：系统信息、配置文件、Provider、模型、沙箱、MCP 等。

```
vibecoding doctor
```

检查项目：
- **环境**: OS/架构、Go 版本、Shell、Home/工作目录
- **配置文件**: 校验 settings、gateway 和 MCP 配置文件，带解析检查
- **Provider 与模型**: 列出已配置的 Provider（API key 脱敏显示）、模型及其上下文窗口/最大 token/推理标志；验证默认 Provider 初始化
- **沙箱**: 检查 bubblewrap 可用性和版本
- **MCP 服务器**: 列出已配置的 MCP 服务器
- **会话**: 显示会话目录和条目数量
- **技能**: 显示全局和项目技能目录
- **上下文文件**: 发现 AGENTS.md、CLAUDE.md、CURSOR.md、.cursorrules、CONVENTIONS.md

```bash
vibecoding doctor
```

示例输出：
```
  VibeCoding Doctor
  ─────────────────

  Environment
    ✅ OS / Arch — linux/amd64
    ✅ Go version — go1.24.4
    ✅ Shell — /bin/bash
    ✅ Home directory — /home/user
    ✅ Working directory — /home/user/project

  Configuration Files
    ✅ Global settings — /home/user/.vibecoding/settings.json (1.2 KB)
    ⏭️  Project settings — .vibe/settings.json (not found)
    ...

  Providers & Models
    ✅ Default provider — deepseek-openai
    ✅ Default model — deepseek-v4-flash
    ✅ Provider: deepseek-openai — api=openai-chat, base=https://api.deepseek.com, key=sk-a****xyz
    ✅   └─ deepseek-v4-flash — ctx=1M, max=384K ★ default
    ✅ Provider init — deepseek-openai/deepseek-v4-flash created successfully

  Result: All 15 checks passed
```

## 使用示例

### 基本使用

```bash
# 交互模式
vibecoding

# 带初始提示
vibecoding "解释这个代码库"

# 非交互模式
vibecoding -p "写一个 Hello World"
```

### 指定提供商和模型

```bash
# 使用 DeepSeek (OpenAI API)
vibecoding --provider deepseek-openai --model deepseek-v4-flash

# 使用 DeepSeek (Anthropic API)
vibecoding -p deepseek-anthropic -m deepseek-v4-flash

# 使用自定义提供商
vibecoding --provider my-custom-provider
```

### 选择模式

```bash
# Plan 模式 - 只读分析
vibecoding --mode plan

# Agent 模式 - 标准读写 (默认)
vibecoding -M agent

# YOLO 模式 - 完全访问
vibecoding -M yolo
```

### 多 Agent 模式

```bash
# 启用子 Agent 工具和多 Agent 命令
vibecoding --multi-agent

# ACP 会话也可以启用
vibecoding acp --multi-agent
```

启用后，VibeCoding 会注册 `subagent_*` 工具，并支持后台委托调查等多 Agent 工作流。Cron 命令入口也依赖多 Agent 模式。

### Delegate 模式

```bash
# 启用阻塞式单子 Agent 委托
vibecoding --delegate

# ACP 会话也可以启用
vibecoding acp --delegate

# Gateway 也可以启用
vibecoding gateway --delegate
```

Delegate 模式会注册 `delegate_subagent` 工具，用于同步阻塞式子 Agent 委托。与可并行异步运行子 Agent 的多 Agent 模式不同，Delegate 模式同一时间只运行一个子 Agent，并等待其完成。适合父 Agent 只需要摘要结果的边界清晰调查任务。

可在 TUI 或 Gateway 中通过 `/delegate [on|off|status]` 运行时切换。

### A2A Master 模式

```bash
# 生成示例配置
vibecoding --init-a2a-master-config

# 启用 master 模式
vibecoding --enable-a2a-master

# 启用 master 模式 + 详细日志
vibecoding --enable-a2a-master --verbose
```

启用后，VibeCoding 会加载 `a2a-list.json` 中的远程 agent 列表，注册 `a2a_dispatch` tool，LLM 可自动向远程 agent 分发任务。

### 初始化配置

```bash
# 生成 gateway.json 模板
vibecoding --init-gateway

# 生成 a2a.json 模板
vibecoding a2a --init-a2a-config

# 生成 a2a-list.json 模板
vibecoding --init-a2a-master-config

# 强制覆盖已存在的文件
vibecoding --init-gateway --force
```

### 思考级别

```bash
# 关闭思考
vibecoding --thinking off

# 中等级别
vibecoding -t medium

# 最高级别
vibecoding --thinking xhigh
```

### 会话管理

```bash
# 继续最近的会话
vibecoding --continue
vibecoding -c

# 恢复特定会话
vibecoding --resume session-abc123
vibecoding -r ~/.vibecoding/sessions/my-session.jsonl

# 使用特定会话文件
vibecoding --session ./my-session.jsonl
```

### 沙箱

```bash
# 启用沙箱
vibecoding --sandbox

# 禁用沙箱 (默认)
vibecoding
```

### 管道输入

```bash
# 从 stdin 读取
echo "解释这段代码" | vibecoding -P

# 直接读取文件内容
vibecoding -p "解释这个文件: main.go"
```

### ACP 服务器

```bash
# 启动 ACP 服务器（用于 IDE 集成）
vibecoding acp

# 使用特定模型
vibecoding acp --provider deepseek-openai --model deepseek-v4-flash

# 启用沙箱
vibecoding acp --sandbox --mode agent
```

## 交互式命令

在交互会话中可用的命令:

### 模式与模型

| 命令 | 描述 |
|------|------|
| `/mode [plan\|agent\|yolo]` | 切换或显示当前模式 |
| `/model [model_id]` | 切换或显示当前模型 |
| `/think` | 循环切换思考级别 |

### 会话管理

| 命令 | 描述 |
|------|------|
| `/sessions` | 列出当前项目的会话 |
| `/sessions ls` | 列出所有项目的会话 |
| `/sessions set <id>` | 通过 ID 前缀切换到指定会话 |
| `/sessions clear` | 创建新的空白会话 |
| `/sessions del <id>` | 通过 ID 前缀删除会话 |
| `/clear` | 清空对话 |

### 技能

| 命令 | 描述 |
|------|------|
| `/skills` | 列出可用技能 |
| `/skill <name>` | 激活指定技能 |
| `/skill:<name>` | 激活技能（替代语法） |

### 通用

| 命令 | 描述 |
|------|------|
| `/help` | 显示帮助 |
| `/quit` | 退出 |

## 键盘快捷键

| 快捷键 | 功能 |
|--------|------|
| `Ctrl+C` | 中断当前操作 / 清空输入 |
| `Ctrl+D` | 退出 |
| `Tab` | 循环切换思考级别 |
| `Ctrl+T` | 切换思考内容显示 |

## 环境变量

可以通过环境变量覆盖默认设置:

| 变量 | 描述 |
|------|------|
| `DEEPSEEK_API_KEY` | DeepSeek API 密钥 |
| `VIBECODING_DIR` | 覆盖配置目录 |
| `VIBECODING_PROVIDER` | 覆盖默认提供商 |
| `VIBECODING_MODEL` | 覆盖默认模型 |
| `VIBECODING_MODE` | 覆盖默认模式 |
| `VIBECODING_THINKING` | 覆盖默认思考级别 |
| `VIBECODING_USER_AGENT` | 自定义 User-Agent 字符串 |

## 退出码

| 码 | 描述 |
|----|------|
| 0 | 成功 |
| 1 | 一般错误 |
| 2 | 用法错误 |
| 130 | 用户中断 (Ctrl+C) |
