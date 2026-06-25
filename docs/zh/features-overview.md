# ✨ VibeCoding 核心特性详解

> 一个工具，满足你所有的 AI 编程需求。

---

## 🤖 多提供商支持

### 支持的提供商

| 提供商 | 模型 | API 格式 |
|--------|------|----------|
| **DeepSeek**（默认） | deepseek-v4-flash, deepseek-v4-pro | OpenAI Chat / Anthropic Messages |
| **OpenAI** | GPT-4o, o1 等 | OpenAI Chat |
| **Anthropic** | Claude Sonnet, Opus 等 | Anthropic Messages |
| **厂商适配器** | Google Gemini、Google Vertex、小米、火山引擎/豆包、Kimi、MiniMax、Seed、Qianfan、Bailian、Gitee、OpenRouter、Together、Groq、Fireworks、Mistral、GitHub Copilot、Cloudflare AI Gateway、Cloudflare Workers AI、Amazon Bedrock 等 | OpenAI Chat 或 Anthropic Messages |
| **自定义** | 任何兼容模型 | 通用 OpenAI Chat 或 Anthropic Messages fallback |

### 快速切换

```bash
# 使用 DeepSeek
vibecoding --provider deepseek-openai --model deepseek-v4-flash

# 使用 OpenAI
vibecoding --provider openai --model gpt-4o

# 使用 Anthropic
vibecoding --provider anthropic --model claude-3-5-sonnet-20241022
```

### 厂商适配器

VibeCoding 支持 20+ 厂商适配器，包括：

- **国内**：小米、火山引擎/豆包、Kimi、MiniMax、Seed、Qianfan、Bailian、Gitee
- **国际**：Google Gemini、Google Vertex、OpenRouter、Together、Groq、Fireworks、Mistral、GitHub Copilot、Cloudflare AI Gateway、Cloudflare Workers AI、Amazon Bedrock

无需额外配置，只需设置 API 密钥即可使用。

---

## ⚡ 极速响应

### SSE 流式传输

VibeCoding 使用 SSE（Server-Sent Events）流式传输，实时返回 token，无需等待完整响应。

### 缓存命中优化

- **实时缓存命中率显示**：在 TUI 底部显示缓存命中百分比
- **每轮缓存统计**：每轮对话的缓存命中详情
- **统一缓存指标**：TUI 和 print 模式使用一致的统计口径

### 性能对比

| 特性 | VibeCoding | 其他工具 |
|------|-----------|----------|
| 流式传输 | ✅ SSE | 部分支持 |
| 缓存命中率 | ✅ 实时显示 | ❌ 不显示 |
| Token 统计 | ✅ 统一口径 | ❌ 不一致 |

---

## 🧠 思考模式

### 支持的模型

- **DeepSeek**：deepseek-v4-pro（支持思考模式）
- **OpenAI**：o1 系列（支持思考模式）
- **Anthropic**：Claude 系列（支持思考模式）

### 思考级别

```bash
# 设置思考级别
vibecoding --thinking off      # 关闭思考
vibecoding --thinking minimal  # 最小思考
vibecoding --thinking low      # 低思考
vibecoding --thinking medium   # 中等思考
vibecoding --thinking high     # 高思考
vibecoding --thinking xhigh    # 超高思考
```

### 使用场景

- **复杂问题**：使用 `high` 或 `xhigh` 获得更深入的推理
- **简单问题**：使用 `off` 或 `minimal` 获得更快的响应
- **调试代码**：使用 `medium` 平衡速度和质量

---

## 🛡️ 沙箱隔离

### bwrap 进程级隔离

VibeCoding 使用 [bubblewrap](https://github.com/containers/bubblewrap) 实现进程级隔离，确保安全执行。

### 三种模式

| 模式 | 文件系统 | 网络 | 用途 |
|------|---------|------|------|
| **Plan** | 只读 | ❌ | 分析、规划 |
| **Agent** | 读写 | ❌ | 日常开发 |
| **YOLO** | 完全 | ✅ | 系统级操作 |

### 安全特性

- **bashBlacklist 优先**：黑名单命令（如 `rm -rf`、`sudo`）始终被拦截
- **审批门控**：Agent 模式下，bash 命令需要审批
- **`--print` 快速失败**：遇到需审批命令时直接失败，不会卡住

### 安装 bwrap

```bash
# Debian/Ubuntu
sudo apt install bubblewrap

# Fedora
sudo dnf install bubblewrap

# Arch
sudo pacman -S bubblewrap
```

---

## 📝 会话管理

### SQLite 持久化

VibeCoding 使用 SQLite 存储会话历史。支持：

- **分支**：从任意节点创建新分支
- **压缩**：自动压缩旧会话，节省空间
- **树形结构**：可视化会话树

### 会话命令

```bash
# 继续最近的会话
vibecoding --continue

# 恢复特定会话
vibecoding --resume <session-id>

# 使用特定会话句柄文件
vibecoding --session <session-file.db>
```

### 会话存储位置

| 平台 | 位置 |
|------|------|
| Linux/macOS | `~/.vibecoding/sessions/` |
| Windows | `%APPDATA%\vibecoding\sessions\` |

---

## 🧩 技能系统

### 什么是技能？

技能是可复用的提示片段，用于：

- **项目约定**：编码规范、命名规则、架构模式
- **团队共享**：统一团队的开发风格
- **快速上手**：新成员快速了解项目规范

### 创建技能

```bash
# 全局技能（所有项目可用）
~/.vibecoding/skills/<name>/SKILL.md

# 项目技能（覆盖全局）
.skills/<name>/SKILL.md
```

### 使用技能

```bash
# 列出可用技能
> /skills

# 激活技能
> /skill my-conventions

# 加载技能引用
> /skill_ref my-references
```

### 在线 Skill 市场

VibeCoding 兼容 SkillHub / ClawHub，支持：

- **技能安装**：一键安装在线技能
- **Cron 基础设施**：定时执行技能
- **社区共享**：分享你的技能

---

## 💻 IDE 集成

### ACP 协议

VibeCoding 支持 Agent Client Protocol (ACP)，可集成到：

- **VS Code**
- **Zed**
- **JetBrains IDEs**（IntelliJ IDEA、WebStorm 等）

### VS Code 集成

在 `settings.json` 中添加：

```json
{
  "acp.agents": {
    "vibecoding": {
      "command": "vibecoding",
      "args": ["acp", "--mode", "agent", "--multi-agent"]
    }
  }
}
```

### JetBrains 集成

导航到 `Settings → Tools → ACP Agents` 并添加：

- **Name**: VibeCoding
- **Command**: `vibecoding`
- **Arguments**: `acp --mode agent`

---

## 🌐 网关模式

### OpenAI 兼容 HTTP API

VibeCoding 可以作为 OpenAI 兼容的 HTTP 服务器使用：

```bash
# 启动网关
vibecoding gateway

# 配置文件
~/.vibecoding/gateway.json
.vibe/gateway.json
```

### 使用场景

- **团队共享**：团队成员共享 AI 编程助手
- **CI/CD 集成**：在 CI/CD 流程中使用 AI
- **API 服务**：提供 AI 编程 API 服务

---

## 📱 消息平台

### 微信/飞书/WebSocket

VibeCoding 可以部署为聊天机器人：

```bash
# 启动消息网关
vibecoding hermes

# 配置文件
~/.vibecoding/hermes.json
.vibe/hermes.json
```

### 功能特性

- **多用户支持**：每个用户独立会话
- **持久化会话**：会话历史自动保存
- **进度回调**：实时显示工具执行进度
- **思考显示**：显示 AI 的思考过程

---

## 🤝 多 Agent

### 子 Agent 委托

```bash
# 启用多 Agent 模式
vibecoding --multi-agent

# 子 Agent 工具
subagent_spawn    # 创建子 Agent
subagent_status   # 查看子 Agent 状态
subagent_send     # 发送消息给子 Agent
subagent_destroy  # 销毁子 Agent
```

### Delegate 模式

```bash
# 启用一个阻塞式委托子 Agent
vibecoding --delegate

# 在 TUI 或 Gateway 会话中切换
/delegate on
/delegate status
/delegate off
```

`--delegate` 会注册 `delegate_subagent`，这是一次只运行一个边界清晰调查任务的同步工具。适合主 Agent 只需要最终摘要结果的多步探索任务。

### A2A 协议

```bash
# 启用 A2A Master 模式
vibecoding --enable-a2a-master

# 生成示例配置
vibecoding --init-a2a-master-config
```

### 动态 Workflow

```bash
# 启用 Workflow 模式
vibecoding --workflows

# Workflow 工具
workflow_run      # 运行 Elisp workflow
workflow_status   # 查看 workflow 状态
workflow_cancel   # 取消 workflow 运行
```

使用 Elisp DSL 编排多个 worker agent，支持 phase 阶段划分、parallel 并行执行、并发控制、结果汇总和运行日志。适合代码审计、架构调研、多角色评审、生成-评审循环等复杂多智能体场景。

详见 [Workflow 模式](workflow.md) 文档了解完整语法和最佳实践。

### 使用场景

- **大型任务**：将大任务拆分成多个子任务
- **并行执行**：多个子 Agent 并行工作
- **远程协作**：通过 A2A 协议管理远程 Agent

---

## 🎨 丰富 TUI

### 功能特性

- **Markdown 渲染**：实时渲染 Markdown 内容
- **语法高亮**：代码块语法高亮
- **思考显示**：显示 AI 的思考过程
- **工具弹窗**：查看工具执行详情
- **状态栏**：显示缓存命中率、token 统计等

### 快捷键

| 快捷键 | 功能 |
|--------|------|
| `Ctrl+O` | 打开工具详情 |
| `Ctrl+G` | 切换紧凑显示 |
| `Ctrl+T` | 切换思考显示 |
| `Tab` | 切换模式 |
| `Esc` | 中止当前操作 |

### 斜杠命令

| 命令 | 描述 |
|------|------|
| `/systeminit [guidance]` | 生成或刷新项目 `AGENTS.md` |
| `/reload` | 以新 session 重启（TUI） |
| `/btw <问题>` | 不中断主任务提问旁路问题 |
| `/alloweditpath [add\|remove\|clear] <glob>` | 管理可编辑路径白名单 |
| `/allowautoedit [on\|off] [global]` | 切换 Agent 模式全自动编辑 |

---

## 🔒 安全可控

### 审批机制

```json
{
  "approval": {
    "bashWhitelist": ["go ", "make ", "git ", "npm "],
    "bashBlacklist": ["rm -rf", "sudo"],
    "confirmBeforeWrite": true
  }
}
```

### 安全特性

- **bashBlacklist 优先**：黑名单命令始终被拦截
- **YOLO 模式安全**：即使在 YOLO 模式下，黑名单仍然有效
- **`--print` 快速失败**：遇到需审批命令时直接失败

---

## 📊 统一缓存指标

### 实时显示

- **缓存命中率**：在 TUI 底部实时显示
- **Token 统计**：每轮对话的 token 使用详情
- **统一口径**：TUI 和 print 模式使用一致的统计

### 使用场景

- **成本优化**：监控缓存命中率，优化 API 调用成本
- **性能监控**：监控 token 使用情况，优化响应速度

---

## 🐞 一致的调试输出

### `--debug` 模式

```bash
# 启用调试模式
vibecoding --debug

# 调试输出包括
- Provider 请求/响应
- ACP 模式调试
- 工具执行详情
```

### 使用场景

- **问题排查**：快速定位 API 调用问题
- **开发调试**：调试自定义工具或技能
- **性能分析**：分析 API 调用性能

---

## 🩺 环境诊断

### `doctor` 子命令

```bash
# 运行环境诊断
vibecoding doctor
```

### 检查项目

- **配置文件**：检查 settings.json、gateway.json、hermes.json
- **Provider**：检查 API 密钥和连接
- **沙箱**：检查 bwrap 安装和配置
- **MCP**：检查 MCP 服务器配置
- **会话**：检查会话存储
- **技能**：检查技能加载
- **上下文文件**：检查 AGENTS.md、CLAUDE.md 等

### 版本更新通知

VibeCoding 会通过 npm registry 检测是否有新版本，并在启动时给出非阻塞的更新提醒。检测在后台进行（最多每 24 小时一次），前台不会阻塞。

可通过 `settings.json` 中设置 `"updateCheck": false` 关闭，或通过 `VIBECODING_NO_UPDATE_CHECK=1` 关闭。

---

## 🎯 总结

VibeCoding 是一个**全能终端 AI 编程助手**，具有以下核心优势：

1. **一个二进制文件搞定一切**：无需安装多个工具
2. **多提供商支持**：20+ 厂商适配器，一键切换
3. **安全可控**：沙箱隔离、审批门控、黑名单保护
4. **丰富的功能**：会话管理、技能系统、IDE 集成
5. **高性能**：SSE 流式传输、缓存命中优化

**准备好开始了吗？** [5 分钟快速上手](../zh/quick-start-tutorial.md)
