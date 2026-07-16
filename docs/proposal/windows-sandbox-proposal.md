# Windows Sandbox 方案：参考 Codex 的 Capability SID、ACL 与 Elevated Runner

> 状态：Proposal（待讨论）
> 日期：2026-07-16
> 关联方案：`docs/proposal/sandbox-architecture-proposal.md`
> 实施状态：未完成（Windows backend 尚处于设计阶段）
> 审查结论：当前 Windows env-only backend 已被标记 unavailable，Elevated runner、ACL、WFP、setup 生命周期和真实 Windows CI 均未实现。

## 1. 背景与结论

MothX 当前 `internal/sandbox/windows.go` 只收缩子进程环境变量。它既不能限制文件系统，也不能限制网络或进程权限，因此不能称为 strict/standard sandbox。

Codex 的 Windows sandbox 不只是单个 `CreateProcess` flag，而是一套 Windows 原生隔离体系：

- 将跨平台 `PermissionProfile` 解析为 Windows 可执行权限；
- 使用 restricted token 和 capability SID，使进程只获得指定 read/write root 的访问能力；
- 通过 ACL 将 deny-read / deny-write 应用于受限身份；
- 支持普通（unelevated/legacy）和 elevated 两条执行路径；
- elevated 模式预配置独立 sandbox identity、runner 和受保护 IPC；
- 使用 Windows Filtering Platform（WFP）支持网络限制；
- 提供 setup、readiness、refresh、日志、错误报告和真实 Windows 测试。

MothX 应借鉴其“**先解析 profile，再由 Windows 专用 helper 以受限身份执行**”的方向。近期必须先将当前 env-only backend 标为 unavailable；真实 Windows sandbox 是一个独立工程，不应通过小改动伪装完成。

## 2. Codex 实现调研

### 2.1 结构与关键文件

| Codex 文件/模块 | 职责 | MothX 可借鉴点 |
|---|---|---|
| `core/src/windows_sandbox.rs` | 选择 Disabled / RestrictedToken / Elevated，协调 setup/readiness | 将 Windows capability 状态上升为明确产品状态 |
| `windows-sandbox-rs/src/resolved_permissions.rs` | 将跨平台 `PermissionProfile` 转成 Windows read/write/network roots | Windows backend 不直接读取 UI/config，而消费统一 profile |
| `windows-sandbox-rs/src/token.rs` | 构造 readonly / workspace-write restricted token 与 capability SID | 用 restricted token 作为进程身份基础 |
| `windows-sandbox-rs/src/workspace_acl.rs`、`acl.rs` | 对 capability SID 配置 ACL / inheritance | 路径权限通过 Windows ACL 表达，不依赖 shell 包装 |
| `windows-sandbox-rs/src/deny_read_acl.rs`、`deny_read_state.rs` | 解析、计划、同步 deny-read ACL | deny 需要实际 OS enforcement 和持久状态管理 |
| `windows-sandbox-rs/src/unified_exec/backends/legacy.rs` | 非 elevated restricted-token session | 低权限模式能力有限，需清晰声明限制 |
| `windows-sandbox-rs/src/unified_exec/backends/elevated.rs` | 使用 runner/IPC 执行 elevated sandbox session | 高保证隔离需要独立身份和 helper 边界 |
| `windows-sandbox-rs/src/wfp*.rs` | WFP 网络过滤与 setup | 网络禁用不是简单清环境变量 |
| `windows-sandbox-rs/src/setup*.rs`、`identity.rs` | provisioning、版本化 setup、credentials | 安装/升级/恢复是产品功能，不是一次性命令 |
| `app-server/.../windows_sandbox_processor.rs` | readiness/setup API 与异步通知 | UI/服务应能显示 NotConfigured/Ready/UpdateRequired |

### 2.2 Codex 的两种模式

Codex 暴露 `Disabled`、`RestrictedToken`（unelevated）和 `Elevated`：

1. **RestrictedToken / legacy**
   - 在当前用户上下文中创建 restricted token，配合 capability SID 和 ACL 启动命令。
   - 实现成本较低，不需要先完成 elevated provisioning。
   - 能力有明确限制：Codex 代码中拒绝 restricted read-only profile 和额外 deny-read override，原因是 write-restricted token 不能让 capability-SID deny-read ACL 成为可靠的权威边界。

2. **Elevated**
   - 需要预先 setup/provision；创建并维护 sandbox identity、受保护凭据/目录、runner 和 IPC。
   - 以单独 sandbox account/token 运行 runner；通过命名管道与主进程通信。
   - 可以表达更完整的 read root、write root、deny-read/deny-write 与受管网络策略。
   - setup 具备版本/ready 状态；失效时需要 refresh/re-provision。

这说明 Windows 平台不能把“restricted token”简单等同于完整文件系统隔离。MothX 的产品结论是：**Restricted Token 不作为用户可选模式或安全承诺；只有 Elevated backend setup ready 后才开放 Windows strict/standard sandbox。**

### 2.3 Codex 的重要实现经验

1. **权限先解析，再执行**：Windows backend 接收 resolved profile，而不是直接从 command 推断工作区。
2. **全盘写不可表达时明确拒绝**：Codex 的 `token_mode_for_permission_profile` 对 full-disk write 返回错误，而不是假装限制仍有效。
3. **deny 有 backend 前提**：legacy backend 对 deny-read 明确报错，要求 elevated backend。
4. **helper/IPC 是安全边界的一部分**：elevated runner 的命名管道、凭据、日志、进程终止和 stdio forwarding 都有专门实现。
5. **网络规则是系统资源**：WFP filter 的安装、刷新和清理需要 setup lifecycle，不能仅在 child env 加 `NO_PROXY` 或删除 proxy。
6. **readiness 是 API**：前端/服务端知道 sandbox 是否 configured、ready 或 update-required，而不是只显示一个 bool。

## 3. MothX Windows 目标语义

### 3.1 与统一 `EffectiveProfile` 的映射

Windows backend 消费主方案中的 immutable `EffectiveProfile`：

| EffectiveProfile | Windows target |
|---|---|
| `AccessReadOnly` root | capability SID + ACL read/execute grant |
| `AccessReadWrite` root | write capability SID + inheritable ACL grant |
| `AccessDeny` root | elevated backend 对 sandbox identity 添加 deny read/write/execute/delete ACL；legacy backend 不支持则 unavailable |
| `NetworkDisabled` | elevated backend 通过 WFP 对 sandbox identity/runner block outbound；未实际执行 WFP 时不可声称网络已隔离 |
| `NetworkFull` | 不安装 outbound block；仍保留 filesystem token/ACL 限制 |
| `ProtectGit=true` | `.git` 解析为 `AccessDeny`；单次 Git approval 生成当前 command 最小 profile transform |
| yolo / profile disabled | 不启动 Windows sandbox helper |

### 3.2 产品状态

```go
type WindowsSandboxState string

const (
    WindowsSandboxUnavailable     WindowsSandboxState = "unavailable"
    WindowsSandboxNotConfigured   WindowsSandboxState = "not_configured"
    WindowsSandboxSetupRequired   WindowsSandboxState = "setup_required"
    WindowsSandboxRestrictedToken WindowsSandboxState = "restricted_token"
    WindowsSandboxElevated        WindowsSandboxState = "elevated"
    WindowsSandboxUpdateRequired  WindowsSandboxState = "update_required"
)
```

- `restricted_token` 只能接受 capability 中实际支持的 profile；不能执行 deny、完整 read-only 或 WFP enforcement 时返回 profile unsupported。
- `elevated` 才可作为 strict/standard + deny + network-disabled 的目标 backend。
- 任何 sandbox enabled 入口在 state 不是能表达 profile 的 `elevated`（或未来经过明确验证的等效 backend）时 fail-closed。
- 当前 MothX env-only 方案只能在 `sandbox.enabled=false` 时作为普通 process env hygiene 使用，不能参与上述状态机。

## 4. 推荐架构

```text
MothX Go process
   │
   ├─ Policy resolver -> EffectiveProfile (immutable)
   │
   ├─ Windows backend capability/readiness service
   │      ├─ setup version / binary integrity / identity status
   │      └─ returns State + Capabilities + diagnostics
   │
   └─ Windows runner client
          │ protected named pipe (authenticated, versioned protocol)
          ▼
      mothx-windows-sandbox-runner.exe
          │
          ├─ restricted token / sandbox identity
          ├─ capability SIDs + ACL grants
          ├─ deny ACL application/verification
          ├─ WFP network policy (when requested)
          ├─ CreateProcessAsUser / Job Object process-tree control
          └─ framed stdout/stderr/exit/cancel messages
```

### 4.1 Go 主程序职责

- 解析、canonicalize、验证 `EffectiveProfile`，不把原始 JSON 或机密传给 runner。
- 查询 readiness，验证 runner 版本、签名/哈希、setup manifest。
- 以结构化请求发送 command、cwd、最小 env、profile fingerprint 和已解析 roots。
- 提供同步/异步 bash、取消和 job status 的统一适配。
- 将 Windows backend capability 回报给 TUI/Serve/doctor。

### 4.2 Windows helper 职责

建议 helper 独立 Go module 或独立仓库子目录，使用 build tag `windows` 和 `golang.org/x/sys/windows`。**不得使用 CGO、C/C++ bridge 或外部 native bridge。** 提升权限和 Windows syscall/unsafe 代码必须集中在 helper，不得散布到主程序。

helper 负责：

- 创建/打开受限 token 或 sandbox account token；
- 管理 capability SID；
- 对 allow roots 创建最小 ACL；
- 对 deny roots 创建拒绝 ACL，并验证最终 DACL 的规则和 inheritance；
- 使用 Job Object 限制/清理进程树；
- 使用 `CreateProcessAsUser` 或等效 API 启动 command；
- private desktop 默认关闭；保留实现开关，但不在首版产品配置中暴露。非交互 shell、关闭 stdin、Job Object 和 token/ACL/WFP 限制仍必须始终生效。
- WFP filter 安装、refcount、移除；
- named pipe server 认证、framed protocol、stdout/stderr、取消、超时；
- 审计日志和安全清理。

## 5. Setup、提升权限和生命周期

### 5.1 为什么需要 setup

完整 ACL deny 与 WFP 可能需要创建独立 identity、安装持久规则、配置受保护目录或进行管理员授权。Codex elevated 模式同样把这作为显式 setup，而不是普通命令启动时临时提权。

MothX 应提供：

```text
mothx sandbox status
mothx sandbox setup --windows-elevated --current-user
mothx sandbox repair --windows-elevated
mothx sandbox teardown --windows-elevated
```

要求：

- setup/repair/teardown 必须是用户明确发起的本地操作；Serve/Channels 不得自动尝试提升权限。
- setup 使用版本化 manifest；应用/runner 升级后状态可为 `update_required`。
- 所有 ACL/WFP 变更必须有 ownership marker，teardown 只能删除 MothX 自己创建的规则。
- 失败时输出可操作错误（权限不足、组策略禁止、WFP 不可用、identity 创建失败），不回退为普通执行。

### 5.2 受限 token 与 capability SID

- read-only profile 使用受限的 sandbox identity/token，去除不必要 privilege，并让 capability SID 对只读 roots 拥有最小 access mask。该 token 机制是 Elevated backend 的内部实现细节，不作为单独产品模式暴露。
- standard/workspace-write profile 使用专用 write capability SID，仅授予明确 write roots。
- 不使用“当前用户全权限 token + 检查 command 字符串”的方案；这不是安全边界。
- capability SID 规则需支持目录 inheritance，防止新建子文件天然脱离策略。
- shell、runtime、必要 DLL、工作 cwd、TEMP/TMP、runner pipe 等运行依赖必须作为 read/write roots 的显式 platform defaults 解析和审计。

### 5.3 deny 与 `.git`

- elevated backend 才允许 `AccessDeny`。
- 对 `.git` 及用户 deny root，为 sandbox identity 创建拒绝 read/write/execute/delete/child-create 的 ACL。
- 所有 deny ACL 需要在 command 前验证：canonical path、DACL owner、ACE order、inheritance、junction/reparse point。
- 命令结束后：session-scoped deny rule 应按 refcount/transaction 清理；若采用持久 sandbox identity，应确保 deny 不影响真实用户，且 repair/teardown 可以恢复。
- Git Allow once 使用临时 profile/ACL transform，仅在该 command runner 生命周期内取消该 `.git` deny，结束后无条件恢复；不得修改基础 session profile。

### 5.4 网络

- `NetworkDisabled` 的完成标准是 Windows sandbox identity/runner 被 WFP 拦截 outbound，而不是 unset env 或防火墙提示文本。
- WFP rules 必须按 profile/session/job 关联，确保并发 session 不相互删除规则。
- runner crash、app crash、取消、setup repair 后必须清理 orphan filters；需要 ownership tag 和 startup reconciliation。
- `NetworkFull` 不安装 block filter；仍可保留 filesystem sandbox。
- proxy-only / allowlist 暂不实现，模型中预留即可。

## 6. 进程、IPC 与作业控制

### 6.1 Runner protocol

使用 named pipe，协议应包含：

```go
type RunnerSpawnRequest struct {
    Version            int
    RequestID          string
    Command            []string
    Cwd                string
    Env                map[string]string // already filtered
    Profile            ResolvedWindowsProfile
    ProfileFingerprint string
    Timeout            time.Duration
    TTY                bool
}
```

响应采用 frame：`Ready`、`Stdout`、`Stderr`、`Exit`、`Error`、`Audit`、`CancelAck`。

安全要求：

- pipe name 含随机 nonce，使用安全 descriptor 限制当前真实用户和 sandbox helper identity；
- handshake 绑定 protocol version、request nonce、expected runner binary version；
- 不接受未认证 client；不从 pipe 接收任意 profile override；
- command/env/paths 长度有硬上限；所有日志脱敏；
- client 断开或 cancel 时 helper 使用 Job Object 终止整个 process tree；
- async job 在主程序重启后的 orphan 处理必须有明确定义（建议终止，不恢复）。

### 6.2 Shell 兼容

MothX Windows 当前可能使用 BusyBox 或 PowerShell。runner 接收最终 argv，不依赖 shell 文本解析：

- BusyBox：`busybox sh -c <command>`；
- PowerShell：`powershell.exe -NoProfile -NonInteractive -Command <command>`；
- `cmd.exe`：`cmd.exe /c <command>`。

helper 不决定 approval 或 Git command 分类；主程序先做 profile/approval 决策，helper 只强制已解析权限。

## 7. 具体开发工作项

### 7.1 立即修正（与主方案同步）

- [ ] `internal/sandbox/windows.go` 不再把 env filtering 标记为 strict/standard available。
- [ ] `Probe()` 返回 `FilesystemIsolation=false`、`NetworkIsolation=false`、`DenyAccess=false` 及明确原因。
- [ ] sandbox enabled 的 Windows CLI、ACP、OpenAI API、Channels 统一 fail-closed。
- [ ] doctor/TUI/Serve status 展示 `Windows sandbox unavailable: real backend not configured`。

### 7.2 Profile 与 API

- [ ] 定义 `ResolvedWindowsProfile`：canonical read roots、write roots、deny roots、network、temp roots、Git escalation transform。
- [ ] 定义 capability matrix；拒绝 legacy/restricted-token 无法表达的 profile。
- [ ] 为 `sandbox status/setup/repair/teardown` 增加 CLI、TUI、Serve 管理 API；管理 API 只能在本地/admin policy 允许时调用。
- [ ] session 保存 backend state / profile fingerprint，不存储 token、ACL SDDL 或完整 env。

### 7.3 Helper / runner

- [ ] 新建 `internal/windowsandbox/` Go package 与 `cmd/mothx-windows-sandbox-runner/` helper（或确认单独 module 布局）。
- [ ] Windows-only token API：restricted token、privilege removal、capability SID 创建/打开。
- [ ] ACL API：canonical path、DACL read/modify、allow/deny ACE、inheritance、transaction/rollback、ownership marker。
- [ ] Job Object、CreateProcessAsUser、stdin/stdout/stderr forwarding、ConPTY（如果 TTY 需要）。
- [ ] named pipe authenticated framed transport、protocol tests、timeout/cancel/runner crash handling。
- [ ] runner binary integrity/version checks；禁止从 PATH 启动未知 helper。

### 7.4 Elevated setup

- [ ] 设计独立 sandbox identity 与 credentials 存储（使用 DPAPI 或 Windows Credential Manager）；密钥不得写入普通 config/log。
- [ ] setup manifest、version、readiness、repair、teardown 与 orphan cleanup。
- [ ] 管理员提升 UX：明确说明要创建的 identity/ACL/WFP 规则；失败不降级。
- [ ] private desktop 默认关闭；实现层保留内部开关和回归测试，确保关闭时不削弱 token、ACL、WFP、Job Object 或非交互 stdin 限制。

### 7.5 WFP 网络 enforcement

- [ ] 为 sandbox identity 建立 outbound block WFP filter。
- [ ] session/job refcount 和 crash cleanup。
- [ ] loopback（`127.0.0.1`、`::1`、localhost）默认允许；WFP 规则必须阻断外部 TCP/UDP/DNS/IPv6 流量，同时精确排除 loopback，以支持本地开发、测试和未来本地受控代理。
- [ ] 在真实 Windows runner 验证外部 TCP/UDP/DNS/IPv6 被阻断、loopback 可达，防止仅阻断一部分路径。

### 7.6 `.git` escalation

- [ ] 解析 repo/worktree `.git` 路径、junction/reparse point 与外部 git dir。
- [ ] elevated profile 添加 `.git` deny ACL。
- [ ] TUI/ACP Allow once 请求将该 command 的 git root 临时移出 deny ACL；runner finally 块恢复。
- [ ] Serve/Channels 无交互时结构化拒绝 Git access；yolo 不注入 deny。

### 7.7 可观测性与测试

- [ ] 安全日志：setup、runner launch、profile fingerprint、ACL/WFP apply/rollback、grant outcome；所有值脱敏。
- [ ] 单元测试：profile mapping、ACL plan、ACE ordering、path canonicalization、pipe framing、capability matrix。
- [ ] Windows integration tests：真实 token/ACL/WFP/runner 测试，需独立 Windows CI runner。
- [ ] 故障测试：runner crash、主进程 crash、cancel、setup version stale、ACL apply 失败 rollback、WFP cleanup。

## 8. 验收矩阵

| 场景 | Restricted Token | Elevated | 期望 |
|---|---:|---:|---|
| strict read-only | 不作为完成目标 | 必须支持 | workspace 可读、不可写 |
| standard workspace-write | 可选支持 | 必须支持 | 仅 workspace/显式 write roots 可写 |
| deny user path | 不支持时 fail-closed | 必须支持 | read/write/execute/delete 均失败 |
| `.git` baseline | 不支持时 fail-closed | 必须支持 | git 元数据完全不可访问 |
| `.git` Allow once | 不支持 | 必须支持 | 当前 command 成功，下一调用重新请求 |
| network disabled | 不作为产品模式 | 必须支持 | 外部 TCP/UDP/DNS/IPv6 被拒绝；loopback 按策略保留可达 |
| network full | 可选 | 必须支持 | 网络可用但 filesystem 限制继续生效 |
| sub-agent/cron/workflow | 不得能力降级 | 必须支持 | profile fingerprint 与父 session 一致 |
| yolo | 不应用 | 不应用 | 不注入 `.git` deny 或 outer sandbox |

真正 release-ready 的 Windows strict/standard 以 Elevated 行为为唯一标准。Restricted Token 仅可作为 Elevated backend 内部实现细节或研发原型，不能作为用户可见、可配置或可宣传的 sandbox 模式。

## 9. 风险与不采纳的捷径

1. **仅 `CreateRestrictedToken`**：不足以完整表达 deny-read 与网络限制；不能当作最终方案。
2. **仅环境变量或 proxy env**：没有 OS enforcement，不构成 filesystem/network sandbox。
3. **直接改真实用户 ACL**：可能锁住用户文件或与其他应用冲突；必须使用 sandbox identity/capability SID、ownership marker 与可恢复 transaction。
4. **每条命令临时管理员提升**：UX 差、攻击面大、难以审计；应通过显式 setup 创建长期受控 runner 环境。
5. **通过解析 `git` 字符串判断安全性**：解析仅用于审批说明，真正限制必须由 ACL/profile 强制。
6. **无 Windows CI 即宣称完成**：不可接受；ACL、token、WFP 必须在真实 Windows 版本上验证。

## 10. 已确认技术决策

1. Helper 仅使用纯 Go 与 `golang.org/x/sys/windows`，不使用 CGO、C/C++ bridge 或其他 native bridge。
2. Windows strict/standard 仅在 Elevated backend setup ready 后开放；Restricted Token 不作为用户可选模式或安全承诺。
3. private desktop 默认关闭，首版不暴露用户配置；核心 token、ACL、WFP、Job Object 与 non-interactive 限制不依赖它。
4. `NetworkDisabled` 阻断外部网络，保留 loopback（`127.0.0.1`、`::1`、localhost）。
5. Elevated 是一次性 setup/provision 以创建低权限、独立 runner 环境；它不是让 Agent 或 child process 以管理员权限执行。

## 12. 实施状态审查（2026-07-16）

- [x] 已确认纯 Go + `golang.org/x/sys/windows`、不使用 CGO 的实现约束。
- [x] 已确认仅 Elevated backend ready 后开放 strict/standard；Restricted Token 不作为产品模式。
- [x] 已确认 private desktop 默认关闭，NetworkDisabled 保留 loopback、阻断外部网络。
- [x] 当前 env-only `winSandbox` 不再报告为可用 filesystem sandbox，Windows sandbox enabled 会 fail-closed。
- [ ] 尚未创建 `internal/windowsandbox` 或 `mothx-windows-sandbox-runner.exe`。
- [ ] 尚未实现 sandbox identity、ACL/capability SID、deny ACL、WFP、Job Object、CreateProcessAsUser、named pipe runner。
- [ ] 尚未实现 setup/status/repair/teardown/readiness API 或版本化 manifest。
- [ ] 尚未实现 Windows Git Allow once ACL transform。
- [ ] 尚未有真实 Windows CI 的 token/ACL/WFP/runner 集成测试。

因此 Windows 专项方案整体状态为：**设计完成，代码实现未开始**。

Windows sandbox 完成不等于 helper 能启动。完成标准是：

- Windows sandbox enabled 时，profile 无法表达或 setup 未 ready 一律 fail-closed；
- Elevated backend 在真实 Windows CI 上验证 strict、standard、deny、`.git` 单次升级与 network disabled；
- runner、ACL、WFP、identity、IPC、Job Object 都具备 crash/cancel/setup-upgrade cleanup；
- UI/API 真实报告 backend level/capabilities；
- yolo 不受到 `.git` baseline 或 outer sandbox 的影响；
- 无机密泄漏到 config、session、audit log 或 named-pipe protocol diagnostics。
