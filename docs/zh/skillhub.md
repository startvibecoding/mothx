# 在线Skill市场集成

VibeCoding（项目代号 Hermas / Claw）计划支持从在线技能市场安装技能。中国用户将使用 **SkillHub**，海外用户将使用 **ClawHub**。

| 平台 | 地址 | 区域 |
|------|------|------|
| **SkillHub** | [https://skillhub.cn](https://skillhub.cn/) | 中国 |
| **ClawHub** | [https://clawhub.ai](https://clawhub.ai/) | 海外 |

> **注意：** Hub 集成尚未实现。当前 VibeCoding 仅支持本地技能。本文档描述已实现的本地技能系统和 Cron 基础设施。

本指南涵盖：

1. [当前技能系统](#当前技能系统) — 已实现的功能
2. [Cron 基础设施](#cron-基础设施) — 已有的定时任务基础

---

## 当前技能系统

本地技能系统已完整实现，可以直接使用。

### 技能工作原理

技能是存储为 `SKILL.md` 文件的可复用提示片段。启动时加载并注入系统提示词。

```
┌─────────────────────────────────────────────────────────────┐
│                     技能系统                                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  全局技能                       项目技能                      │
│  ~/.vibecoding/skills/         .skills/                      │
│  ┌─────────────────────┐      ┌─────────────────────┐       │
│  │ coding-standards/   │      │ project-specific/   │       │
│  │   SKILL.md          │      │   SKILL.md          │       │
│  │                     │      │                     │       │
│  │ git-workflow/       │      │ testing-rules/      │       │
│  │   SKILL.md          │      │   SKILL.md          │       │
│  └─────────────────────┘      └─────────────────────┘       │
│            │                            │                    │
│            └──────────┬─────────────────┘                    │
│                       ▼                                      │
│              ┌─────────────────┐                             │
│              │  系统提示词       │                             │
│              └─────────────────┘                             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 技能目录

| 类型 | 位置 | 作用域 |
|------|------|--------|
| 全局 | `~/.vibecoding/skills/`（Linux/macOS）或 `%APPDATA%\vibecoding\skills\`（Windows） | 所有项目 |
| 项目 | `.skills/`（项目根目录） | 当前项目，覆盖同名全局技能 |

### 创建技能

创建一个包含 `SKILL.md` 文件的目录：

```bash
mkdir -p ~/.vibecoding/skills/go-expert
cat > ~/.vibecoding/skills/go-expert/SKILL.md << 'EOF'
# Go Expert

专家级 Go 编码规范。

## 规则

- 使用 `gofmt` 格式化代码
- 遵循 Effective Go 指南
- 返回错误，不要 panic
- 使用 `fmt.Errorf` 和 `%w` 包装错误

## 测试

- 编写表驱动测试
- 使用 `t.Run` 子测试
- 目标覆盖率 >80%
EOF
```

### 使用技能

```
> /skills
已加载 2 个技能:
  - go-expert (全局)
  - project-conventions (项目)

> /skill:go-expert
已加载技能: go-expert
```

技能可以包含通过 `skill_ref` 工具按需加载的参考文件：

```
### 1. API 指南 (references/api-guide.md) [待按需加载]
```

### 配置

在 `settings.json` 中配置全局技能目录：

```json
{
  "skillsDir": "~/.vibecoding/skills"
}
```

项目技能自动从 `.skills/` 加载，无需额外配置。

---

## Cron 基础设施

VibeCoding 已有内部 cron 基础设施（`internal/cron` 包）和 TUI 命令入口。Cron 存储将任务持久化到 `~/.vibecoding/cron.json`，调度器每 30 秒检查一次到期任务。

> **注意：** 完整的 cron 集成（自然语言调度解析、TUI 中的实际子 Agent 执行接线）仍在开发中。`/cron` TUI 命令已作为入口点存在，但尚未完全连接到 cron 存储和调度器。

### `/cron` TUI 命令

需要多 Agent 模式（`--multi-agent` 或 Ctrl+P 切换）：

```
> /cron add <描述>             — 添加定时任务
> /cron list                   — 列出定时任务
> /cron enable <id>            — 启用任务
> /cron disable <id>           — 禁用任务
> /cron remove <id>            — 删除任务
> /cron run <id>               — 立即运行任务
```

### Cron 任务数据模型

每条 cron 任务记录存储：

| 字段 | 描述 |
|------|------|
| `id` | 唯一任务 ID（如 `cron-1716883200`） |
| `name` | 任务简短描述 |
| `prompt` | 发送给子 Agent 的任务提示词 |
| `schedule` | 5 字段 cron 表达式 |
| `mode` | `agent` 或 `yolo` |
| `enabled` | 任务是否激活 |
| `last_run` | 上次执行时间戳 |
| `next_run` | 计算得出的下次执行时间 |
| `run_count` | 总执行次数 |
| `last_status` | `success`、`failed` 或 `running` |

### 调度器架构

```
调度器循环 (每 30 秒)
    │
    ├── 从存储列出所有已启用任务
    │
    ├── 检查每个任务：是否到期？
    │   ├── 从未运行 → 到期
    │   ├── NextRun 已过 → 到期
    │   └── 上次运行超过 1 小时 → 到期（兜底）
    │
    └── 到期任务 → 创建子 Agent
              │
              ├── 标记任务为 "running"
              ├── 通过 AgentManager 创建 Agent
              ├── 使用任务 prompt 运行 Agent
              ├── 收集结果
              └── 更新任务状态 (success/failed)
```

---

## 相关文档

- [技能系统](skills.md) — 本地技能格式和管理
- [配置详解](configuration.md) — 完整设置参考
- [安全与沙箱](security.md) — 沙箱和审批控制
