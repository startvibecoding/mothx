<p align="center">
  <img src="assets/mothx-small.png" alt="MothX" width="128" height="128">
</p>

<h1 align="center">MothX</h1>

<p align="center">
  <strong>🚀 一个二进制文件搞定一切 — 你的终端 AI 编程助手</strong>
</p>

<p align="center">
  别再在 Claude Code、Codex、Claw、Hermes 之间来回切换了。<br>
  MothX 把所有功能打包成一个文件 — 提供商、工具、沙箱、会话、技能，一应俱全。
</p>

<p align="center">
  <a href="https://www.npmjs.com/package/mothx"><img src="https://img.shields.io/npm/dm/mothx.svg" alt="npm downloads"></a>
  <a href="https://pypi.org/project/vibecoding-installer/"><img src="https://img.shields.io/pypi/v/vibecoding-installer.svg" alt="PyPI version"></a>
  <a href="https://gitee.com/startvibecoding/mothx/releases/latest"><img src="https://img.shields.io/badge/Gitee-release-blue" alt="Gitee release"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
  <a href="https://goreportcard.com/report/github.com/startvibecoding/mothx"><img src="https://goreportcard.com/badge/github.com/startvibecoding/mothx" alt="Go Report Card"></a>
  <a href="https://pkg.go.dev/github.com/startvibecoding/mothx"><img src="https://pkg.go.dev/badge/github.com/startvibecoding/mothx?status.svg" alt="GoDoc"></a>
</p>

<p align="center">
  <strong>GitHub: <a href="https://github.com/startvibecoding/mothx">海外用户</a></strong>
</p>

> **改名说明：** MothX 原名 VibeCoding。本过渡版本仍保留 `vibecoding` 命令、`vibecoding-installer` 包以及 `.vibecoding` 配置目录作为兼容入口；新安装和后续更新请使用 `mothx`。

---

## ✨ 为什么选择 MothX？

**问题：** 你在多个 AI 编程工具之间疲于奔命 — Claude Code 做这个，Codex 做那个，Claw 又是另一个。每个都有自己的配置、自己的坑、自己的依赖。

**解决方案：** MothX 是**全能终端 AI 编程助手**，一个工具搞定所有事情。一个二进制文件，一份配置，零烦恼。

### 🎯 核心亮点

| 特性 | 对你意味着什么 |
|------|---------------|
| **⚙️ 工作流模式** | 动态 Elisp 工作流，支持阶段划分、并行执行、多 Worker 协调 — 自动化复杂开发流水线 |
| **🤖 多提供商** | DeepSeek、OpenAI、Anthropic、火山引擎/豆包、LongCat 和 25+ 厂商适配器 — 一键切换模型 |
| **⚡ 极速响应** | SSE 流式传输、实时 token 送达、缓存命中优化 |
| **🧠 思考模式** | 复杂问题的扩展推理（DeepSeek、o1、Claude） |
| **🛡️ 沙箱隔离** | bwrap 进程级隔离 — 安全文件操作、网络控制、审批门控 |
| **📝 会话管理** | 持久化 SQLite 历史，支持分支、压缩和树形结构 |
| **🧩 技能系统** | 可复用的提示片段，用于项目约定 — 团队共享 |
| **📊 统计面板** | 使用统计图表、按厂商/模型分类统计、CLI 模式 |
| **🔄 工作流** | Elisp DSL 实现多阶段、并行、多 Agent 编排 |
| **💻 IDE 集成** | ACP 协议支持 VS Code、Zed、JetBrains — 原生编辑器集成 |
| **🌐 网关模式** | OpenAI 兼容 HTTP API，支持多工作区会话隔离 |
| **📱 消息平台** | 微信、飞书、WebSocket，支持远程 TUI 客户端 |
| **🤝 多 Agent** | `--multi-agent` 异步子 Agent、`--delegate` 阻塞式委托，以及 A2A Master 模式 |
| **🎨 丰富 TUI** | Markdown 渲染、语法高亮、思考显示、工具弹窗、多行输入 |
| **🔒 安全可控** | 项目级 bash 自动审批规则、bashBlacklist 优先于白名单、交互式审批对话框 |

---

## 🚀 30 秒上手

```bash
# 安装（任选其一）
npm install -g mothx                         # npm（推荐）
npm install -g vibecoding-installer          # 兼容包，本过渡版本保留
pipx install vibecoding-installer           # PyPI
curl -fsSL https://gitee.com/startvibecoding/mothx/raw/main/install.sh | bash  # Linux/macOS

# 设置 API 密钥
export DEEPSEEK_API_KEY=sk-...

# 运行
mothx
```

就这么简单，你已经在用 AI 编程了。

**卸载:**

```bash
# npm
npm uninstall -g mothx
npm uninstall -g vibecoding-installer        # 如果曾安装旧兼容包

# PyPI
pipx uninstall vibecoding-installer

# Linux/macOS（一键安装）
curl -fsSL https://gitee.com/startvibecoding/mothx/raw/main/install.sh | bash -s -- --uninstall

# Windows（一键安装）
irm https://gitee.com/startvibecoding/mothx/raw/main/install.ps1 | iex; Uninstall-MothX
```

---

## 🎮 三种模式，应对各种场景

```
🗒️  Plan    → 只读分析和规划。安全、沙箱化、没有意外。
🔧  Agent   → 标准读写。Bash 需要审批。（默认）
🚀  YOLO    → 完全系统访问。没有限制。给勇者的。
```

随时切换模式：`/mode plan|agent|yolo` 或按 `Tab`。

---

## 🏗️ 架构概览

```
mothx/
├── cmd/mothx/        # CLI 入口
├── internal/
│   ├── agent/             # 核心 Agent 循环
│   ├── provider/          # LLM 提供商抽象（20+ 厂商）
│   ├── tools/             # 内置工具（read, write, bash, grep, find, ...）
│   ├── sandbox/           # bwrap 沙箱实现
│   ├── session/           # SQLite 会话存储
│   ├── skills/            # 技能系统
│   ├── tui/               # 终端 UI（BubbleTea + Lipgloss）
│   ├── gateway/           # OpenAI 兼容 HTTP 网关
│   ├── hermes/            # 消息平台网关（微信/飞书/WebSocket）
│   ├── a2a/               # A2A 协议服务器和 Master 模式
│   └── acp/               # ACP / MCP 集成
└── pkg/sdk/               # 公共 SDK 接口
```

---

## 📚 文档

### 🚀 入门指南
- [5 分钟快速上手](quick-start-tutorial.md) — 别读长文档，直接上手！
- [快速入门](getting-started.md) — 安装、配置、首次运行
- [命令行参考](cli-reference.md) — 所有命令和参数

### ✨ 核心特性
- [核心特性详解](features-overview.md) — 多提供商、沙箱、会话、技能等

### 🎯 使用场景
- [使用场景与实战示例](use-cases.md) — 从日常开发到企业部署

### ⚙️ 配置
- [配置详解](configuration.md) — 设置、环境变量、认证

### 🏗️ 架构
- [系统架构](architecture.md) — 核心组件、数据流
- [工具系统](tools.md) — 内置工具指南
- [技能系统](skills.md) — 可复用提示片段
- [在线 Skill 市场](skillhub.md) — SkillHub / ClawHub 集成
- [工作流模式](workflow.md) — Elisp DSL 多 Agent 编排

### 🔒 安全
- [安全与沙箱](security.md) — 沙箱模式、权限、审批

### 💻 IDE 集成
- [ACP 协议](acp.md) — VS Code、Zed、JetBrains 集成

### 🌐 网关模式
- [Gateway 模式](gateway.md) — OpenAI 兼容 HTTP API
- [Hermes 模式](hermes.md) — 微信/飞书/WebSocket 聊天机器人
- [A2A 协议](a2a.md) — Agent-to-Agent 协议

### 📖 教程
- [场景演示](scenarios.md) — 实际用法示例
- [FAQ](faq.md) — 常见问题解答

### 🇬🇧 English Docs
- [English Documentation](../en/README.md) — Full English documentation

---

## 🎯 使用场景

### 💻 日常开发
```bash
mothx -P "把这个函数重构成泛型版本"
mothx -P "为 UserService 结构体写测试"
mothx -P "解释这个正则表达式的作用"
```

### 🔍 代码审查
```bash
mothx --mode plan "审查这个 PR 并提出改进建议"
```

### 🚀 CI/CD 集成
```bash
mothx -p "从 git log 生成更新日志" > CHANGELOG.md
```

### 🌐 API 服务器
```bash
mothx gateway  # 启动 OpenAI 兼容 HTTP 服务器
```

### 📱 聊天机器人
```bash
mothx hermes   # 部署为微信/飞书机器人
```

---

## 🛠️ 内置工具

| 工具 | 描述 |
|------|------|
| `read` | 读取文件内容（支持图片） |
| `write` | 创建/覆盖文件（原子操作） |
| `edit` | 精确多处文本替换 |
| `bash` | 执行 Shell 命令（同步/异步，默认 45 秒超时） |
| `grep` | 搜索文件内容（纯 Go ripgrep） |
| `find` | 按模式查找文件（纯 Go fd） |
| `ls` | 列出目录内容 |
| `plan` | 发布任务计划 |
| `question` | 向用户提问（多选） |
| `jobs` | 管理后台任务 |
| `kill` | 停止后台任务 |
| `skill_ref` | 加载技能引用 |
| `delegate_subagent` | 阻塞式单子 Agent 委托 |
| `workflow_run` | 运行 Elisp 工作流编排 |

---

## 🔧 配置

### 配置文件

| 位置 | 平台 | 作用域 |
|------|------|--------|
| `~/.vibecoding/settings.json` | Linux/macOS | 全局 |
| `%APPDATA%\vibecoding\settings.json` | Windows | 全局 |
| `.vibe/settings.json` | 所有 | 项目（覆盖全局） |

### 环境变量

| 变量 | 描述 |
|------|------|
| `DEEPSEEK_API_KEY` | DeepSeek API 密钥 |
| `VIBECODING_DIR` | 覆盖配置目录 |
| `VIBECODING_PROVIDER` | 覆盖默认提供商 |
| `VIBECODING_MODEL` | 覆盖默认模型 |
| `VIBECODING_MODE` | 覆盖默认模式 |
| `VIBECODING_DEBUG` | 启用调试输出 |

---

## 🤝 贡献

我们欢迎贡献！详见 [开发指南](development.md)。

```bash
git clone https://gitee.com/startvibecoding/mothx.git
cd mothx
make build
make test
```

---

## 📄 许可证

MIT — 详见 [LICENSE](../../LICENSE)。

---

<p align="center">
  <strong>准备好开始了吗？⭐ Star 这个仓库，开始编程吧！</strong>
</p>
