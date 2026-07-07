# /auth & /settings V2 代码审查

> 审查日期: 2026-07-01
> 审查范围: `internal/tui/auth_*.go`, `internal/config/settings.go` (导出 helper), `internal/serve/openaiapi/commands.go`
> 对照基线: `docs/proposal/auth-settings-v2-proposal.md`
> 状态: 所有问题已修复 (2026-07-01)

---

## 1. Phase 6 完成状态（对照 proposal §15） ✅ 全部完成

| 项目 | 状态 | 说明 |
|------|------|------|
| 单元测试（默认透传正确性） | ✅ | `TestResolveProviderConfigMergesDefaults`, `TestDefaultModelConfigLookup` |
| 单元测试（全字段 round-trip） | ✅ | `TestProviderEditStateRoundTrip`, `TestModelEditStateRoundTrip` |
| 单元测试（compat 三态） | ✅ | `TestCycleTriState`, `TestToggleModelTriState`, `TestCompatEditStateActiveCount`, `TestCompatResetToAuto` |
| 单元测试（headers 编辑流） | ✅ | `TestHeadersEditFlow` |
| 单元测试（cost enabled/disabled 切换） | ✅ | `TestCostEnabledToggle` |
| 单元测试（`previewBuildFoldedJSON` + fold/expand） | ✅ | `TestPreviewBuildFoldedJSONMultipleModels`, `TestPreviewFoldMaskedKey` |
| 单元测试（tri-state pointer 循环 nil→true→false→nil） | ✅ | `TestCycleTriState`, `TestToggleModelTriState` |
| 移除老扁平字段 | ✅ | `authDialogState` 无老扁平字段 |
| 旧 settings.json 兼容 | ✅ | Name 回退 id、Cost=nil 跳过、Compat=nil 回退 zero |

---

## 2. 方案中未实现的功能

| 功能 | 方案章节 | 状态 | 说明 |
|------|----------|------|------|
| Headers 编辑器的删除操作 | §7.4 | ✅ 已实现 | Backspace 删除当前 header entry |
| Serve `/settings` 命令 | §17 | ⚠ N/A | HTTP 模式无 TUI 对话框，可接受 |
| Model List Done → Default → Review | §2.4.1 | ⚠ 偏差 | `popAuthView()` 返回上一级，不强制走 Default/Review |

---

## 3. 方案之外的新增实现

| 新增项 | 文件 | 说明 |
|--------|------|------|
| `authViewAddModelName` 两步输入 | `auth_dialog.go` | 方案仅一个输入，实现拆为 ID→Name |
| `authViewAPIChoice` 独立选择器 | `auth_provider.go` | API Type 从 Protocol 组分离为独立视图 |
| `previewExpansion` 折叠机制 | `auth_render.go` | cost/compat 在 Review JSON 中折叠/展开 |
| `previewBuildFoldedJSON` | `auth_render.go` | 带折叠标记的 JSON 预览 |
| `compatEditState.Active` 标记 | `auth_state.go` | 追踪 compat 是否被显式编辑 |
| `← Back` 返回按钮 | `auth_dialog.go` | Existing Providers 列表末尾 |
| `✓ Done` 保存按钮 | `auth_provider.go` | Provider Group List 末尾直接保存 |
| Provider Group Done 直接 save | `auth_dialog.go` | `saveAuthProvider()` 而非进入 Review |
| `filterAuthProviderIDs` 搜索过滤 | `auth_render.go` | |
| `sortedAuthProviderIDs` 排序 | `auth_render.go` | |
| `authProviderSortPriority` 硬编码优先级 | `auth_render.go` | |

---

## 4. 代码坏味道（Code Smells）

### 4.1 死代码（Dead Code） ✅ 全部已清理

已删除：
- `selectModelGroup` (auth_model.go) — 从未调用
- `submitAddModelID` / `submitAddModelName` (auth_model.go) — 空壳函数
- `*App.renderAuthPreview` 方法 (auth_render.go) — 被同名包级函数遮蔽
- `authProviderBaseURLSuggestions` (auth_provider.go) — 从未调用
- `resolveProviderBaseURL` (auth_provider.go) — 从未调用
- `renderAuthPreviewTrunc` (auth_render.go) — 包装函数

保留：`baseURLOptionsForProvider` — 仅测试用，不影响生产代码

### 4.2 重复代码（Duplication） ✅ 已清理

已统一：config 包导出 `CloneStringMap`, `CloneBoolPtr`, `CloneFloat64Ptr`, `CloneStringSlice`，tui 重复定义已删除。

保留：`cloneProviderConfig` 签名不同（tui 返回值 vs config 返回指针），用途不同。

### 4.3 命名与遮蔽问题 ✅ 已修复

- ~~`max` 函数遮蔽 Go 1.21+ 内置~~ → 已删除（项目使用 Go 1.26）
- ~~`renderAuthPreview` 同名遮蔽~~ → 方法版本已删除

### 4.4 结构与设计问题（保留，影响低）

| 问题 | 影响 | 建议 |
|------|------|------|
| `authViewAPIChoice` 不在 Provider Group 内 | 低 | 保留现状，枚举选择器比文本输入体验好 |
| `providerGroupProtocol` 不包含 API Type | 低 | 保留现状 |
| `modelEditState` Cost 字段平铺而非嵌套 | 低 | 改动范围大，保留现状 |
| summary 函数模式重复 | 低 | 重构收益低，保留现状 |

### 4.5 潜在 Bug ✅ 已修复

已修复：
- ~~`insertFoldMarkers` 只替换第一个匹配~~ → 改为 `strings.ReplaceAll`
- ~~`renderFoldedPreview` 名称解析错误~~ → 使用 `strings.Index` 提取名称

保留（风险低）：
- `previewBuildFoldedJSON` 未检查 nil map entry — 当前调用链安全
- `renderFoldedPreview` 基于字符串匹配 — 概率极低

### 4.6 空注释 ✅ 已清理

~~`auth_settings.go` 空注释~~ → 已删除

---

## 5. 修复清单

| 修复项 | 文件 | 变更 |
|--------|------|------|
| 删除 `selectModelGroup` | auth_model.go | 移除函数及 `selectAuthOption` 中的 case |
| 删除 `submitAddModelID`/`submitAddModelName` | auth_model.go, auth_dialog.go | 移除函数及调用点 |
| 删除 `*App.renderAuthPreview` | auth_render.go | 移除死方法 |
| 删除 `authProviderBaseURLSuggestions` | auth_provider.go | 移除函数 |
| 删除 `resolveProviderBaseURL` | auth_provider.go | 移除函数，清理 config import |
| 删除 `renderAuthPreviewTrunc` | auth_render.go | 移除包装函数 |
| 删除 `max` 函数 | auth_dialog.go | 使用 Go 1.26 内置 |
| 清理空注释 | auth_settings.go | 删除遗留注释 |
| `insertFoldMarkers` 改用 `ReplaceAll` | auth_render.go | 支持多模型折叠 |
| 修复 `renderFoldedPreview` 名称解析 | auth_render.go | 使用 `strings.Index` 提取 |
| 导出 clone helpers | config/settings.go | `CloneStringMap` 等 4 个函数 |
| 删除 tui clone 重复 | auth_state.go | 改用 `config.CloneXxx` |
| Headers Backspace 删除 | auth_dialog.go | 新增 Backspace handler |
| 新增 8 个测试 | auth_dialog_test.go | Phase 6 覆盖 |
