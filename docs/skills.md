# 技能系统

VibeCoding 的技能系统允许您创建可重用的提示片段，称为技能 (Skills)。

## 概述

技能是存储为 `SKILL.md` 文件的提示片段，可以被加载并注入到系统提示中。

```
┌─────────────────────────────────────────────────────────────┐
│                     Skills System                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Global Skills                 Project Skills                │
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
│                       │                                      │
│                       ▼                                      │
│              ┌─────────────────┐                             │
│              │  System Prompt  │                             │
│              └─────────────────┘                             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 技能目录

### 全局技能

位置: `~/.vibecoding/skills/`

全局技能对所有项目可用。

```bash
~/.vibecoding/skills/
├── coding-standards/
│   └── SKILL.md
├── git-workflow/
│   └── SKILL.md
└── review-checklist/
    └── SKILL.md
```

### 项目技能

位置: `.skills/` (在项目根目录)

项目技能仅对当前项目可用，并且会覆盖同名的全局技能。

```bash
.skills/
├── project-conventions/
│   └── SKILL.md
└── testing-rules/
    └── SKILL.md
```

## SKILL.md 格式

每个技能是一个 Markdown 文件，包含技能的描述和内容。

### 示例

```markdown
# Coding Standards

This skill defines coding standards for the project.

## Go Code Style

- Use `gofmt` for formatting
- Follow Effective Go guidelines
- Use meaningful variable names
- Add comments for exported functions

## Error Handling

- Always handle errors
- Use `fmt.Errorf` with `%w` for wrapping
- Don't panic in library code

## Testing

- Write table-driven tests
- Use `t.Run` for subtests
- Aim for >80% coverage
```

### 结构

1. **标题** (H1): 技能名称
2. **描述** (第一段): 技能的简短描述
3. **内容**: 详细的内容和规则

## 配置

### 技能目录

在 `settings.json` 中配置技能目录:

```json
{
  "skillsDir": "~/.vibecoding/skills"
}
```

### 项目本地技能

项目技能自动从 `.skills/` 目录加载，无需额外配置。

## 使用

### 加载技能

技能在启动时自动加载:

```go
skillsMgr := skills.NewManager(globalSkillsDir, projectSkillsDir)
if err := skillsMgr.Load(); err != nil {
    // 处理错误
}
```

### 查看已加载技能

在交互模式中使用 `/skills` 命令:

```
> /skills
Loaded 3 skills:
  - coding-standards (global)
  - git-workflow (global)
  - project-conventions (project)
```

### 技能注入

技能内容会被注入到系统提示中，供 LLM 参考。

## 最佳实践

### 1. 保持技能简洁

每个技能专注于一个主题:

```markdown
# Git Commit Rules

## Commit Message Format

<type>(<scope>): <subject>

## Types

- feat: New feature
- fix: Bug fix
- docs: Documentation
- style: Formatting
- refactor: Code restructuring
- test: Adding tests
- chore: Maintenance
```

### 2. 使用全局技能共享通用规则

将适用于所有项目的规则放在全局技能目录:

```bash
~/.vibecoding/skills/
└── general-rules/
    └── SKILL.md
```

### 3. 使用项目技能覆盖特定规则

项目技能可以覆盖全局技能:

```bash
.skills/
└── general-rules/  # 覆盖全局的 general-rules
    └── SKILL.md
```

### 4. 组织技能目录

按主题组织技能:

```bash
~/.vibecoding/skills/
├── coding/
│   ├── go-style/
│   │   └── SKILL.md
│   └── python-style/
│       └── SKILL.md
├── git/
│   ├── commit-rules/
│   │   └── SKILL.md
│   └── branch-strategy/
│       └── SKILL.md
└── review/
    └── checklist/
        └── SKILL.md
```

## 示例技能

### 代码审查技能

```markdown
# Code Review Checklist

## Before Review

- [ ] Code compiles without errors
- [ ] Tests pass
- [ ] No lint warnings

## Review Points

- [ ] Clear variable/function names
- [ ] Proper error handling
- [ ] No hardcoded values
- [ ] Comments for complex logic
- [ ] Test coverage adequate

## Security

- [ ] No SQL injection
- [ ] No XSS vulnerabilities
- [ ] Proper input validation
- [ ] Secrets not exposed
```

### 测试技能

```markdown
# Testing Guidelines

## Test Structure

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    interface{}
        expected interface{}
    }{
        // test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

## Coverage

- Aim for >80% code coverage
- Focus on edge cases
- Test error paths
```

## 故障排除

### 技能未加载

1. 检查目录权限
2. 确认 `SKILL.md` 文件名正确
3. 使用 `/skills` 命令查看加载状态

### 技能未生效

1. 确认技能内容格式正确
2. 检查系统提示是否包含技能内容
3. 重启 VibeCoding 重新加载

## 相关文档

- [配置详解](configuration.md) - 技能目录配置
- [系统架构](architecture.md) - 技能系统架构
