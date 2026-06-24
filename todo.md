# 代码审查问题清单

> 生成时间：2026-06-24

---

## 严重问题

### 1. 静默忽略错误（广泛存在）

多处代码忽略 JSON 序列化和 IO 操作的错误，可能导致下游逻辑基于零值或空数据继续执行。

**受影响文件：**

- `internal/workflow/tools.go` — 多处 `json.Marshal` 错误被忽略（line 75, 286, 328, 335, 377）
- `internal/mcp/mcp.go` — `io.ReadAll` 错误被忽略（line 631, 741, 806）
- `internal/messaging/wechat/protocol.go` — `rand.Read` 和 JSON 解码错误被忽略（line 59, 101, 107）
- `internal/provider/openai/responses.go` — `io.ReadAll` 错误被忽略（line 208）
- `internal/provider/anthropic/provider.go` — `io.ReadAll` 错误被忽略（line 367）
- `internal/provider/google/provider.go` — `io.ReadAll` 错误被忽略（line 290）
- `internal/provider/openai/provider.go` — `io.ReadAll` 错误被忽略（line 385）

**典型问题代码：**

```go
// protocol.go:59
rand.Read(buf[:])  // error silently ignored

// protocol.go:101
json.NewDecoder(resp.Body).Decode(&result)  // error ignored

// mcp.go:631
data, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
```

**建议：** 使用 `if err != nil { return ..., err }` 模式，或至少记录日志。

---

### 2. Webhook 路由 JSON 解析失败被静默忽略

**文件：** `internal/hermes/webhook/router.go:102`

```go
json.Unmarshal(body, &generic)  // error ignored
```

如果 JSON 解析失败，`eventType` 为空，事件会被静默过滤掉，调用方不知道是解析失败还是确实没有匹配的事件类型。

---

### 3. goroutine 缺少 panic 恢复

**文件：** `internal/mcp/mcp.go:279-282`

```go
go func() {
    _ = cmd.Wait()
    client.closePending(fmt.Errorf("MCP server %q exited", cfg.Name))
}()
```

如果 `cmd.Wait()` panic（虽然罕见），goroutine 会直接终止，没有恢复机制，可能导致 client 状态不一致。

---

## 中等问题

### 4. TUI 下 `/sessions set` 指令后上下文使用量不显示

**文件：** `internal/tui/commands.go:673-743`（`sessionsSet` 函数）

切换会话后，`a.contextUsage` 被设置为 `nil`，而状态栏渲染依赖此字段：

```go
// commands.go:729
a.contextUsage = nil  // ← 被清空

// render.go:264
if a.contextUsage != nil && a.contextUsage.ContextWindow > 0 {
    // 显示上下文使用量
}
```

**结果：** 切换会话后，状态栏的上下文 token 使用量（如 "45.2%/128K"）消失，直到新 agent 启动并发送 `ContextUsage` 事件才恢复。

**修复方向：** 在 `sessionsSet` 加载历史消息后，计算新会话的上下文使用量并设置 `a.contextUsage`。

---

### 5. Session Compact 潜在边界 panic

**文件：** `internal/session/session.go:533`

```go
nextEntryIDs = append(nextEntryIDs, append([]string(nil), state.entryIDs[firstKept:]...)...)
```

如果 `firstKept` 超过 `len(state.entryIDs)`，会产生 panic。虽然有调用方检查，但属于边界情况风险。

---

### 6. BashTool Execute 异步命令 cancel 未 deferred

**文件：** `internal/tools/bash.go:148-152`

```go
if async {
    cmdCtx, cancel = context.WithCancel(context.Background())
} else {
    cmdCtx, cancel = context.WithTimeout(ctx, timeout)
    defer cancel()  // 非异步时会 deferred
}
```

异步命令的 `cancel` 未 deferred，如果函数提前返回，cancel 不会被调用。虽然对后台进程影响较小，但可能导致资源泄漏。

---

### 7. WeChat 登录轮询无总超时

**文件：** `internal/messaging/wechat/auth.go:145,153`

```go
time.Sleep(2 * time.Second)
// retry logic in loop
```

虽然有重试次数限制，但总等待时间可能较长，用户体验不佳。

---

## 问题统计

| 严重程度 | 数量 | 主要类型 |
|---------|------|---------|
| 严重 | ~15 | 静默忽略错误 |
| 中等 | 4 | 竞态条件、边界问题、用户体验 |
| 轻微 | 2 | 边缘情况 |

---

## 修复优先级

1. **P0（立即修复）：** 问题 1（mcp.go、wechat/protocol.go）、问题 3
2. **P1（尽快修复）：** 问题 2（webhook router）、问题 4（TUI session switch）
3. **P2（计划修复）：** 问题 5、6、7
