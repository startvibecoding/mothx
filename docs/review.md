# Code Review 报告

**提交**: `a61cb91` — refactor: extract configureCommand and handle session stop reason in TUI
**日期**: 2026-06-30
**审查人**: AI Review
**分支**: main

---

## 构建与测试

| 检查项 | 状态 |
|--------|------|
| `go build ./...` | ✅ 通过 |
| `go vet ./...` | ✅ 通过 |
| 全部测试 (`go test ./internal/...`) | ✅ 通过 |

---

## 代码审查

### `internal/tools/bash.go`

**评分**: ✅ 优秀

**变更内容**:

- 将 `setSysProcAttr(cmd)` 和 `cmd.WaitDelay = 100 * time.Millisecond` 提取为独立方法 `configureCommand()`
- 统一了沙箱路径和非沙箱路径的命令配置逻辑
- 消除了 `buildWindowsCommand` 中的重复代码

**结论**: 重构清晰，无问题。

---

### `internal/tools/tools_test.go`

**评分**: ✅ 优秀

**变更内容**:

- 新增 `wrappingTestSandbox` 模拟结构体，实现完整的 `sandbox.Sandbox` 接口
- 新增测试 `TestBashToolSandboxCommandDoesNotWaitForBackgroundChildStdio`，验证沙箱路径下后台子进程不会阻塞 shell 退出

**结论**: 测试设计合理，mock 完整，边界用例覆盖到位。

---

### `internal/tui/events.go`

**评分**: ✅ 优秀

**变更内容**:

- `listenAgentEvents()` 新增 `lastDone` 变量，在事件消费循环中记录最后一个 `EventDone` 或 `EventError` 的 `StopReason`
- 通过 `agentDoneMsg{err, stopReason}` 将停止原因传递到 Update 循环

**结论**: 逻辑正确，事件处理流程完整。

---

## 发现的问题

### 问题 1（中等）: `EventError` 分支中的 stop reason 样式不一致

**位置**: `internal/tui/agent_events.go` 第 181–184 行

```go
if event.Error != nil {
    a.addMessage(errorStyle.Render("Error: ") + a.formatAgentError(event))
} else if event.StopReason != "" {
    a.addMessage(errorStyle.Render("Error: session ended with reason: ") + event.StopReason)
}
```

**问题描述**:

- 当事件同时携带 `Error` 和 `StopReason`（如 `EventError` 通常携带 `context_limit`、`stuck` 等停止原因）时，只显示错误样式，完全忽略 `StopReason`
- `else if` 分支使用 `errorStyle` 渲染 "Error:" 前缀，但语义上是正常会话结束（如 `"context_limit"`），应使用 `statusStyle` 和 "Session ended:" 前缀
- 对比 `app.go:925-926` 中 `agentDoneMsg` 的处理方式，使用了 `statusStyle.Render("Session ended: ")`，语义正确

**建议修复**:

```go
// EventError handler
case agent.EventError:
    a.commitActiveStream()
    if (a.multiAgent || a.delegateMode) && a.agentMgr != nil && a.agent != nil {
        a.agentMgr.MarkError(a.agent.ID(), event.Error)
    }
    a.isThinking = false
    a.finishRequestTimer()
    if event.Error != nil {
        a.addMessage(errorStyle.Render("Error: ") + a.formatAgentError(event))
    }
    if event.StopReason != "" {
        a.addMessage(statusStyle.Render("Session ended: ") + event.StopReason)
    }
    a.pendingAbortReason = ""
    a.currentAssistantIdx = -1
    a.currentThinkIdx = -1
    a.updateViewportContent()
    return tea.Batch(a.timer.Stop(), a.listenAgentEvents())
```

**影响**: 此问题为历史遗留（非本次提交引入），但本次提交使该分支可能被实际命中（因为 `EventError` 现在会携带 `StopReason`）。当前影响有限，建议后续版本统一处理。

---

## 附: 前序提交 `b31901d` (PyPI Installer Packaging)

**变更内容**:

- 新增 `pypi/` 目录，包含 setuptools wheel 打包配置
- 新增构建脚本 `scripts/build-pypi-packages.sh`、`scripts/sync-pypi-version.sh`
- 新增 `make pypi-*` 系列 Makefile 目标
- 更新 README 和文档，补充 PyPI 安装方式（`pipx install vibecoding-installer`）

**结论**: 打包结构规范，平台映射完整，脚本逻辑正确。

---

## 总结

| 项目 | 结果 |
|------|------|
| 本次提交整体质量 | ✅ 通过 |
| 测试覆盖 | ✅ 完整 |
| 遗留问题数 | 1（中等，影响有限） |
| 建议 | 可合并，建议后续修复样式不一致问题 |