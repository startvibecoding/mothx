# Sandbox 架构增强方案：统一权限策略、全拒绝规则与跨平台真实能力

> 状态：Proposal（待讨论）
> 日期：2026-07-16
> 目标版本：待定
> 实施状态：部分完成（截至 2026-07-16）
> 审查结论：Linux 基础 deny 与 Git 单次升级已实现；统一 EffectiveProfile、完整 backend contract、macOS 对齐、Windows Elevated backend、完整可观测性和跨平台执行测试尚未完成。

## 1. 摘要

MothX 已有 Bubblewrap（Linux）、Seatbelt（macOS）和基础 Windows backend；v1.1.68 已接入部分 `settings.sandbox` 配置，并修复 API 多工作目录及 Channels 子 Agent 没有继承 sandbox manager 的问题。

不过当前实现的本质仍是“根据 mode 选择 backend，再拼装命令参数”。它缺少统一、不可变、可验证的权限策略模型；`deniedPaths` 不能覆盖已经 broad-bind 的项目根；跨平台能力与 UI 状态不一致；审批和 sandbox 扩权尚未形成统一边界。

本方案确定采用 Codex 的核心设计思想：**逻辑权限策略与平台执行后端分离**。MothX 需要先生成一个按 workDir 固化的 `EffectiveProfile`，再将它编译为 Linux Bubblewrap、macOS Seatbelt 或 Windows backend 的实际执行方案。每一次命令、主 Agent、sub-agent、workflow 与 cron 都只能使用所属 session/run 的同一 profile snapshot。

本文不按 Phase 划分；第 9 节列出完整开发范围、依赖、验收标准和提交约束，实施时可以拆分 PR，但每一项均属于目标架构的一部分。

## 2. 已确认的产品决策

| 主题 | 决策 |
|---|---|
| CLI 与 Serve 的 backend 不可用行为 | 统一 fail-closed：用户显式启用 sandbox 而 backend/profile 不可执行时，CLI、ACP、OpenAI API、Channels 都返回错误，不得自动退化为 `NoneSandbox`。关闭 sandbox 或 yolo 模式是显式选择。 |
| `deniedPaths` | 定义为**完全拒绝**：禁止读取、写入、创建、删除、重命名、执行及作为 cwd/挂载目标使用。它优先于允许/可写规则。 |
| `.git` | 默认受到完整拒绝保护；读、写或作为 Git 工作目录访问都必须经过一次明确的升级权限申请。升级只对该 command/调用有效，不改变 session 基础 profile。yolo 不受此限制。 |
| `.mothx` | 不作为默认 protected path；遵循普通 profile 和用户显式路径规则。 |
| Windows | 单独形成 Windows sandbox proposal，参考 Codex 的 elevated runner / ACL / WFP 思路；实现仅使用 Go 与 `golang.org/x/sys/windows`，不使用 CGO。当前 env-only backend 不得宣称具有 filesystem sandbox 能力。 |
| Windows 可用模式 | 仅在 Elevated backend setup 已就绪时开放 Windows strict/standard sandbox；Restricted Token 不作为用户可选模式或安全承诺。 |
| 网络禁用 | `NetworkDisabled` 阻断外部网络，但保留 localhost / loopback；未来本地受控代理也通过 loopback 接入。 |
| 实现组织 | 不采用按版本分期的产品承诺；用完整架构设计和可独立审查的开发工作项推进。 |

## 3. 目标、非目标与安全边界

### 3.1 目标

1. 建立独立于 bwrap、Seatbelt、Windows API 的 `EffectiveProfile` 权限模型。
2. 全部运行入口使用相同、不可变、按 `workDir` 实例化的 profile snapshot。
3. 以 fail-closed 规则实现可验证的 allow / deny / nested-path 语义。
4. 将 `.git` 的基础拒绝与按命令升级审批整合，且 yolo 可明确跳过这一层。
5. 每个 backend 只报告实际具备的能力；不允许 no-op 或 env filter 被标记为 strict/standard filesystem sandbox。
6. 通过执行级集成测试证明读、写、删除、执行、网络、子 Agent 继承和升级边界。

### 3.2 非目标

- 本方案不将 sandbox 视为认证、授权、MCP server 安全或恶意宿主代码防护的替代品。
- 第一版网络只要求 `disabled` / `full`；保留受控代理模式的模型位置，但不实现域名 allowlist。
- 不直接移植 Codex Rust helper 或其 bundled Bubblewrap；参考其架构和安全语义。
- 不改变 yolo 的语义：yolo 允许无 outer sandbox，且 `.git` 不触发 sandbox 升级申请。现有 bash 黑名单等独立审批规则仍按自身规则工作。
- 不改变已有 `settings.json` 字段的既有含义；新增字段必须 optional 且有迁移说明。

### 3.3 威胁模型

防护对象是 Agent 通过内建 `bash` 工具派生的子进程及其后代。目标是限制它们访问未授权的文件、网络与敏感元数据。

不保证防护：

- MothX 进程本身、已获宿主同等权限的恶意插件/MCP server；
- 用户在 yolo 或升级审批中明确授权后的访问；
- 已被合法读取后通过模型输出、网络或侧信道泄露的数据；
- kernel/backend 漏洞、具有管理员/root 权限的攻击者。

## 4. 当前状态与主要不足

### 4.1 当前实现

- `internal/sandbox/sandbox.go`：`strict`、`standard`、`none` level 与 `Manager`。
- `internal/sandbox/bwrap.go`：Linux bwrap 创建 PID / IPC / UTS / network namespace，并挂载项目、系统路径、`/proc`、`/dev`、`/tmp`。
- `internal/sandbox/mac.go`：临时 Seatbelt profile。
- `internal/sandbox/windows.go`：环境变量收缩；没有 filesystem/network isolation。
- v1.1.68：`SandboxSettings.Options()`、OpenAI API per-workDir manager、Channels session manager 复用、额外 allow path 的基本 denied 检查。

### 4.2 缺口

| 优先级 | 缺口 | 风险 |
|---|---|---|
| P0 | 项目 root 已被 bind 后，项目内 `deniedPaths` 仍可见 | secret / `.git` 配置产生错误安全感 |
| P0 | `deniedPaths` 当前只是 allow path 的跳过逻辑，不是完整拒绝 | 可读、可写、可执行与 path overlap 语义缺失 |
| P0 | macOS 未消费统一 options，strict 仍授予真实 home 中 `.config` / `.cache` / `.mothx` 写权限 | 同一设置跨平台行为不同 |
| P0 | Windows env filtering `IsAvailable=true` | 用户/API 误以为拥有 filesystem sandbox |
| P1 | 路径缺少一致 canonicalization、symlink 与不存在目标规则 | mount 逃逸、错误拒绝或入口不一致 |
| P1 | bwrap capability 只做最小 probe，未验证所需隔离 flags | 在 container/root/distro 环境中能力被高估 |
| P1 | sandbox 与审批未形成 command 级升级模型 | `.git` 操作只能永久放开或永久拒绝 |
| P1 | 测试主要检查 command 构造，不证明权限实际生效 | 回归难以发现 |
| P2 | `VIBECODING_SANDBOX_PASS_ENV` 是 settings 之外的隐式扩权 | 策略来源与 UI 不一致 |
| P2 | session/runtime settings 切换的 profile 生命周期不清晰 | 主/子 Agent 的权限可能漂移 |

## 5. Codex 对标结论

调研基于本机 Codex 源码：

- `codex-rs/protocol/src/models.rs` 用 `PermissionProfile::{Managed, Disabled, External}` 作为逻辑权限源。
- `codex-rs/protocol/src/permissions.rs` 将文件系统条目和 access mode 建模为 restricted/unrestricted/external 策略。
- `codex-rs/linux-sandbox/src/bwrap.rs` 从文件系统策略生成 mount，处理 broad root、writable root、deny carveout、protected metadata、symlink 与 missing targets。
- `codex-rs/linux-sandbox/src/landlock.rs` 在适用时配置 `PR_SET_NO_NEW_PRIVS` 和 seccomp 网络限制。
- `codex-rs/sandboxing/src/seatbelt.rs` 固定使用 `/usr/bin/sandbox-exec`，规范化路径，并独立编译 filesystem/network policy。
- Codex 把 approval 与 sandbox 分离：command 获得批准不等于无条件取消 sandbox；deny restriction 不能因“升级”被隐式丢失。

MothX 应借鉴上述结构，而不复制 Codex 的 Rust helper、内部协议或网络代理实现。

## 6. 目标架构

```text
settings + mode + workDir + request/session override + approval grant
                              │
                              ▼
                      Policy Resolver
          validate / canonicalize / conflict detection / fingerprint
                              │
                              ▼
                 immutable EffectiveProfile snapshot
         filesystem / network / env / temp / protected metadata
                              │
          ┌───────────────────┼────────────────────┐
          ▼                   ▼                    ▼
   Linux profile compiler  macOS compiler     Windows compiler
       bwrap + hardening      Seatbelt       restricted-token / unavailable
          │                   │                    │
          └───────────────────┴────────────────────┘
                              ▼
       PreparedCommand + EffectiveSandboxInfo + cleanup
                              ▼
 bash / async jobs / main agent / sub-agent / workflow / cron
```

### 6.1 核心类型

```go
type AccessMode string

const (
    AccessReadOnly  AccessMode = "read_only"
    AccessReadWrite AccessMode = "read_write"
    AccessDeny      AccessMode = "deny"
)

type NetworkMode string

const (
    NetworkDisabled NetworkMode = "disabled"
    NetworkFull     NetworkMode = "full"
)

type PathRule struct {
    Path       string
    Access     AccessMode
    Provenance RuleProvenance // project, settings, platform-default, approval
}

type EffectiveProfile struct {
    WorkDir            string
    Filesystem         []PathRule
    Network            NetworkMode
    Environment        EnvironmentPolicy
    TemporaryFS        TemporaryFSPolicy
    ProtectGit         bool
    ProfileFingerprint string // redacted, stable hash
    Sources            []ProfileSource
}

type BackendCapabilities struct {
    FilesystemIsolation bool
    NetworkIsolation    bool
    DenyAccess          bool
    ProcessHardening    bool
    GitEscalation       bool
    Reason              string
}
```

`EffectiveProfile` 是 immutable。session 创建、恢复或 workDir 改变时重新解析；每一个 Agent、registry、sub-agent、workflow、cron 使用同一 snapshot 或其仅用于本次 command 的短暂升级副本。

### 6.2 配置兼容映射

| 现有字段 | 目标 profile 映射 |
|---|---|
| `enabled` / `level` | 选择 base profile；strict=项目只读，standard=项目可写，none=disabled |
| `allowNetwork` | `NetworkFull`；否则 `NetworkDisabled` |
| `allowedRead` | `AccessReadOnly` roots |
| `allowedWrite` | `AccessReadWrite` roots |
| `deniedPaths` | `AccessDeny` roots，完整拒绝 |
| `passEnv` | allowlist environment policy |
| `tmpSize` | isolated temporary filesystem policy |
| `bwrapPath` | Linux backend implementation option，不写入逻辑 profile |

建议新增可选字段：

```json
"sandbox": {
  "protectGit": true,
  "fallback": "error",
  "diagnostics": true
}
```

这里 `fallback` 只允许 `error`。保留字段是为了表达与旧项目配置兼容的显式策略，而不是恢复 silent fallback；不支持的旧值加载时应返回迁移错误。

### 6.3 路径解析、冲突和全拒绝语义

所有 path rule 在 profile 解析时完成：

1. 配置中的相对路径相对 session 的 `workDir` 展开；runtime 内部不保留相对路径。
2. 已存在路径使用 `EvalSymlinks` canonicalize；不存在路径使用最长存在父目录 canonicalize 后拼接剩余 component。
3. Windows 使用 volume-aware、case-insensitive 的 canonical comparison；Unix 使用 component-aware comparison。
4. 任何无法绝对化、规范化、stat/解析出安全含义的路径均使 profile 构造失败。
5. `deny` 优先于所有 read/write/platform default 规则。
6. 第一版禁止任意规则冲突：deny root 与 read/write root 存在祖先/后代重叠时，profile 构造失败并给出两条规则和规范化路径。后续若确有需求，再显式实现“deny parent 下 reopen child”的特殊语法；不能靠规则顺序猜测。
7. `workDir`、command cwd、shell executable、所需 runtime library 等若落入 deny，profile 构造失败；不得通过 unsandboxed fallback 继续。
8. deny 对现有目录、现有文件和未来创建路径均有效：禁止 read/write/execute/delete/rename/create/chdir/bind mount。

### 6.4 `.git` 保护与升级审批

#### 基础规则

当 sandbox 已启用且 mode 不是 yolo：

- 每个 workspace root 自动附加 `<root>/.git` 的 `AccessDeny` 规则；
- 对 git worktree 的 `.git` 文件和它指向的真实 git dir 也解析后加入 deny；
- `.git` 的 directory traversal、读取、写入、删除、执行 hook、`git -C ...` / `--git-dir` 访问都应被拒绝；
- 不能通过 symlink、alternate object directory、`GIT_DIR` / `GIT_WORK_TREE` 或 nested worktree 绕过。识别不到的非标准 Git layout 应 fail-closed，提示用户批准明确升级或关闭保护。

#### 升级触发

在 agent/plan 模式中，bash tool 在 profile resolver 后、执行前分析命令和环境：

1. 当 command 需要 `.git` 访问，产生 `GitAccessEscalationRequest`。
2. 请求 UI/API 展示：命令、workDir、被访问的 git root、推断操作类型（read/write/unknown）、为何需要解除保护。
3. 用户可选择：`Allow once` 或 `Deny`。不提供“本 session 永久允许”，避免 `.git` 基础保护被悄然失效。
4. `Allow once` 仅为当前 command 创建 `EscalatedProfile`：只将本 command 所需的 canonical git root 从 deny 集移除，并根据实际 profile 允许只读或读写；不改变 session snapshot，不影响并发 command 与 sub-agent。
5. 若 command 同时需要 profile 之外的 write/network 权限，必须分别走对应升级审批；Git 升级不自动授予其他 root 或网络。
6. yolo 不注入 `.git` deny，因此不请求 Git escalation。

#### 命令识别原则

不能把安全性建立在对 `git` 字符串的完整解析上。命令分类仅用于改善审批文案；真正安全边界仍由基础 deny rule 和批准后的最小 profile 决定。任何不确定的 shell 复合命令都显示 `unknown Git access` 并仅允许用户单次明确升级。

### 6.5 审批、升级和 sandbox 的关系

- 审批决定**该 tool call 是否运行**。
- profile 决定**该 tool call 运行时的实际权限**。
- 升级为一次性、最小范围的 profile transform，带 `callID`、profile fingerprint、原因和过期时间。
- 存在 deny rule 时，任何普通“command approved”均不得自动转为 `NoneSandbox`。
- yolo 是唯一可通过 mode 明确跳过 outer sandbox 的路径；它不从 agent-mode 的批准缓存间接获得同等效果。
- Serve/Channels 没有交互审批能力时，遇到 `.git` escalation 必须拒绝该 command，并返回结构化说明；不能自动允许。后续可考虑只由管理员预配置精确 allow rule。

### 6.6 Backend contract 与真实能力

```go
type Backend interface {
    Name() string
    Probe(context.Context) BackendCapabilities
    Compile(ctx context.Context, profile EffectiveProfile, req CommandRequest) (PreparedCommand, error)
}
```

`PreparedCommand` 包含 argv、env、cwd、cleanup、实际 profile fingerprint、backend capabilities 和可安全记录的 diagnostic summary。

每个入口在创建 session 时 probe 并冻结实际 backend/capability；profile 无法由该 backend 表达时返回 error。CLI、ACP、OpenAI API、Channels 行为一致：**显式启用 sandbox 即 fail-closed**。无论从 settings 还是 CLI flag 启用，都不得静默回退。

`FormatSandboxInfo`、TUI、Serve startup 与 API status 必须展示：请求 profile、实际 backend、filesystem/network/deny capability、是否有 process hardening，以及失败原因；不只展示 `strict` / `standard` 文案。

## 7. 各平台后端要求

### 7.1 Linux / Bubblewrap

1. 使用 capability probe 验证所需 `bwrap` 功能和实际 namespace 创建：至少检查 `--unshare-user`、`--unshare-pid`、`--new-session`、`--die-with-parent`、`--proc`、tmpfs、ro-bind/bind 和网络 namespace。
2. 使用 `--unshare-user`、`--unshare-pid`、`--new-session`、`--die-with-parent`；保留 IPC/UTS 和 network isolation。
3. filesystem 编译顺序：只读基线 → platform defaults → 读路径 → 可写路径 → deny masks → cwd；deny 必须在所有 broad bind 后重新施加。
4. `AccessDeny` 目录通过空 mount/mask 隐藏；文件通过空只读 file/data bind 隐藏；缺失目标需在安全父路径下创建临时 mount target 并在 process 退出后做 inode-safe cleanup。
5. 对 symlink 的 source/target 进行 canonical policy 比较，禁止通过 writable symlink target 穿透 deny。
6. 评估 Go 实现中的 `PR_SET_NO_NEW_PRIVS`；若增加 seccomp，必须有独立 syscall allow/deny 设计、架构覆盖和执行级测试，不能将“尝试设置”宣传为完整进程安全。
7. 移除 `VIBECODING_SANDBOX_PASS_ENV` 的运行时扩权语义；所有 pass-through 仅来源于 profile env allowlist。
8. 网络 disabled 使用 network namespace 隔离；full network 不 unshare network。未来 proxy-only 需要独立本地 bridge，不能仅依赖 `HTTP_PROXY`。

### 7.2 macOS / Seatbelt

1. 固定使用 `/usr/bin/sandbox-exec`，不从 PATH 解析。
2. `EffectiveProfile` 编译为 profile；filesystem、network、temporary home 独立表达。
3. strict/standard 不再对真实 `~/.config`、`~/.cache`、`~/.mothx` 给予默认写权限；使用 sandbox-local HOME、XDG config/cache 与临时目录。
4. `NetworkDisabled` 显式 deny；`NetworkFull` 按 Seatbelt 能力允许，而非当前无条件 deny。
5. `.git` deny 与一次性 Git escalation 必须生成对应 profile carveout；临时 profile 文件权限、生命周期和 cleanup 可审计。
6. 在真实 Darwin runner 做 command-level 测试；sandbox-exec 不可用时 profile 不可用，统一 fail-closed。

### 7.3 Windows

详细实施参见 `docs/proposal/windows-sandbox-proposal.md`。在其完成前：

- 当前 `winSandbox` 更名/标记为 environment restriction，`FilesystemIsolation=false`、`NetworkIsolation=false`、`DenyAccess=false`；
- strict/standard 请求不能返回“sandbox available”；
- 显式 sandbox 启动必须返回 Windows backend unavailable，而非执行 env-only 伪 sandbox；
- yolo/明确关闭 sandbox 仍正常运行。

## 8. 完整开发工作项

### 8.1 策略与生命周期

- [ ] 新增 `internal/sandbox/policy.go`：profile 类型、rule provenance、profile fingerprint、resolver。
- [ ] 新增 `internal/sandbox/path.go`：跨平台 canonicalization、ancestor/descendant/overlap 比较、缺失路径解析。
- [ ] 新增 profile validation：deny precedence、冲突拒绝、cwd/shell/runtime 可达性校验、Git worktree 解析。
- [ ] 将 `SandboxSettings` 映射为 profile input；分离 backend options（如 bwrap binary）和 logical permission rules。
- [ ] Session 创建/恢复、API workDir override、Channels session、ACP、CLI、factory/sub-agent、cron/workflow 统一创建并传递 snapshot。
- `/home` 兼容语义：Linux bwrap 会先将真实用户 home 隔离为 tmpfs，因此历史默认 `deniedPaths: ["/home"]` 在项目位于 `/home/...` 时视为冗余并自动忽略；其他用户明确配置的 deny 规则仍继续 fail-closed。

### 8.2 执行与升级

- [ ] 将 `Sandbox` 接口从直接 `WrapCommand` 扩展为 profile-aware `Compile` / `PreparedCommand`；保留兼容适配层直到所有工具迁移。
- [ ] Bash/job manager 记录 command 的 effective fingerprint 和 backend；async process cleanup 持有 profile resources。
- [ ] 新增 `GitAccessEscalationRequest`、approval UI/API payload、single-call grant 和失效机制。
- [ ] 在 TUI 支持 Allow once / Deny；ACP 通过 permission channel 传递；Serve/Channels 结构化拒绝。
- [ ] yolo 显式跳过 Git deny 注入和 Git escalation。

### 8.3 Linux backend

- [x] bwrap capability probe 已验证 `--unshare-user`、`--new-session`、`--die-with-parent`、`--proc`、tmpfs、bind/ro-bind 和 network namespace flags；命令已显式加入 `--unshare-user` 与 `--new-session`。
- [x] `/proc` 默认通过 `--proc /proc` 挂载，提供 sandbox 内独立 proc namespace 和只读信息访问；不使用 `--remount-ro`，因为 bwrap 需要先初始化 `/proc/<pid>`，过早 remount 会导致命令启动失败。
- [x] mount compiler、deny file/dir masks、缺失 target 基础 mask 和 Git/worktree 保护已实现；inode-safe cleanup、复杂 symlink 防绕过仍待完善。
- [x] user namespace/new session/process tree hardening 的 bwrap flags 已加入；`no_new_privs`/seccomp 仍待评估。
- [ ] env allowlist 收口并删除隐式环境变量扩权。
- [ ] no_new_privs/seccomp 的可行性验证及独立实现/测试（若决定纳入）。

### 8.4 macOS backend

- [ ] profile compiler 与 Linux 语义对齐。
- [ ] sandbox-local home/cache/config 与 profile cleanup。
- [ ] network mode、Git deny/upgrade、固定 binary、Darwin integration test。

### 8.5 Windows backend

- [ ] 立即修正 capability 报告和 fail-closed 行为。
- [ ] 按 Windows 专项 proposal 实现受限 token/capability/ACL/runner/WFP 方案。

### 8.6 可观测性、文档和迁移

- [ ] 实现 `EffectiveSandboxInfo`，在 TUI、doctor、CLI startup、Serve status 与 API session status 中一致输出。
- [ ] 记录无 secret 的 profile fingerprint、backend、capabilities、升级 reason/outcome；不记录完整 env 或 token。
- [ ] 更新 settings UI：只展示当前 OS/backend 实际支持的项；不允许保存不可表达的 profile。
- [ ] 更新用户文档：威胁模型、fail-closed 行为、`.git` 单次审批、yolo 例外、平台能力矩阵。
- [ ] 增加 config migration/validation errors 的可操作文案和示例。

## 9. 测试、验收与质量门槛

### 9.1 单元测试

- profile 默认值、mode/settings/override 优先级、fingerprint 稳定性；
- path normalization：相对路径、symlink、broken symlink、missing child、Windows case/volume；
- conflict detection：deny/read/write overlap、deny cwd、deny shell/runtime；
- `.git`：普通 repo、worktree、`.git` file、nested repo、`GIT_DIR` 解析；
- backend compiler：mount/profile 顺序、deny 最后施加、env 去重、network mode；
- capability probe 和 unavailable reason；
- grant scope/TTL/callID 匹配，确保升级不能复用到另一次调用。

### 9.2 Linux 执行级集成测试

测试仅使用临时目录和 localhost，不依赖公共网络：

| 场景 | 必须断言 |
|---|---|
| strict | 可读普通项目文件；不可创建、修改、删除、rename 项目文件；无网络 |
| standard | 可读写普通项目文件；不可写 profile 外路径；无网络 |
| deny file | `cat`、shell redirection、rename、remove、execute 均失败 |
| deny directory | traversal、read、create、delete 均失败 |
| symlink | 项目内 symlink 指向 deny path 时访问失败 |
| `.git` 基础 profile | `git status` / `.git` read 均因 deny 失败 |
| `.git` Allow once | 单次批准 command 可完成；下一条 `.git` command 再次需要审批 |
| Linux proc | sandbox 内 `/proc` 可读取，PID namespace 隔离；不强制 remount-ro，以避免破坏 bwrap PID 初始化 |
| full network | local listener 可达；disabled network 不可达 |
| proc | `/proc/self/status` 和 `/proc/1` 可读，且 `ps` 能正常工作；proc 写保护不由 remount-ro 提供 |
| async job | 启动、取消、cleanup 后临时资源正确清理，profile 不漂移 |

bwrap 不可用时 integration test 应显式 skip 并输出原因；产品运行在 sandbox enabled 时仍 fail-closed，不能把 test skip 当作 runtime fallback。

### 9.3 macOS / Windows 执行级测试

- macOS：在真实 Darwin CI runner 验证同一 matrix 的 filesystem/network/Git 语义。
- Windows：按 Windows 专项方案在真实 runner 验证 restricted token、ACL deny、network enforcement、runner 进程树和 setup/refresh 行为。
- CI 必须标识“未运行该 OS backend 测试”，不得以 Linux unit test 替代跨平台 enforcement 验证。

### 9.4 审查质量门槛

- 不接受只修改提示文本或 command argv 的安全修复；必须有执行级回归测试。
- 所有新增 profile 字段必须说明默认值、序列化、兼容性和跨平台能力。
- 任何 permission broadening 需要 threat-model 说明和至少一条否定测试。
- 不允许 silent fallback、path canonicalization error ignore、或 deny rule 被 approval/none backend 隐式绕过。

## 10. 迁移风险与处理

1. **历史配置 overlap**：新 resolver 应返回规范化后冲突路径和对应字段；用户必须拆分规则或删除冲突，不能靠加载顺序猜测。
2. **工具依赖真实 home/cache**：macOS 应提供 sandbox-local HOME/XDG，而不是恢复真实 home 写权限。
3. **Git 工作流摩擦**：默认保护 `.git` 会让 `git status` 等命令申请一次性升级。这是明确的产品取舍；TUI 必须让请求内容清晰、操作快速。
4. **受限 container 中 bwrap 不可用**：所有显式 sandbox 入口统一报错，doctor 提供安装/内核配置诊断；不得运行无隔离工具调用。
5. **性能**：profile 在 session/run 创建时解析并缓存；Git layout 与 canonical paths 缓存需以 workDir metadata 失效；glob deny（未来）必须有上限并 fail-closed。
6. **外部工具**：MCP / browser / 其他独立进程不应因 profile 声明而被误认为已被 bwrap 限制；状态需分别展示其是否受 sandbox 管理。

## 11. 仍待确认的设计细节

已确认的产品决策见第 2 节。实施前仍需确认以下技术细节：

1. Git escalation 的 command scope 是否仅允许单个 tool call（本方案建议是），以及 ACP 是否暴露 Allow once 的完整交互。
2. `.git` baseline deny 是否覆盖 bare repo/worktree 外部 git dir；建议覆盖解析到的实际 git dir，但若路径在 workspace 外，审批 UI 必须额外警告。
3. Linux `no_new_privs` 和 seccomp 是否纳入首批 backend hardening；建议 no_new_privs 优先，seccomp 在 syscall 审计后决定。
4. 是否在未来引入 `proxy_only` network mode；当前仅在数据模型预留，不新增用户配置。网络禁用的 loopback 保留规则已确认。
5. Windows 实现语言已确定为纯 Go + `golang.org/x/sys/windows`，不使用 CGO；strict/standard 仅在 Elevated backend ready 后开放，Restricted Token 不作为产品模式。
6. private desktop 默认关闭；是否保留为高级选项及其默认值变更条件仍可在实现设计中确认。

## 13. 实施完成度审查（2026-07-16）

本节根据当前仓库代码复核 proposal，不代表目标设计已经全部落地。

### 13.1 已完成

- [x] CLI、ACP、OpenAI API 在 sandbox enabled 且 backend 不可用时 fail-closed。
- [x] Channels 创建 session 时启用 sandbox 失败会返回错误；无 sandbox 时显式使用 `LevelNone`。
- [x] Linux options 接入：bwrap path、network、allow read/write、deny paths、pass env、tmp size、protectGit。
- [x] 基础路径绝对化、symlink 解析、缺失路径父级规范化和 allow/deny overlap 拒绝。
- [x] Linux broad bind 后对现有 deny 目录/文件执行 mask；缺失 deny 目录使用 `--dir` + `--tmpfs`。
- [x] 项目 `.git` 和 worktree `.git` 文件指向的真实 gitdir 解析与保护。
- [x] TUI/ACP 复用现有 approval 机制触发 `git_access`，Allow once 不写入持久 allow 规则。
- [x] Channels 对无交互 `git_access` 明确拒绝；yolo 不触发 Git sandbox upgrade。
- [x] 目标包测试和 Linux bwrap 执行级基础测试已通过。

### 13.2 部分完成 / 存在偏差

- [ ] 当前仍没有统一的 `EffectiveProfile`、`BackendCapabilities`、`PreparedCommand` 和 profile fingerprint；Options 仍是 backend-oriented。
- [ ] Git Allow once 目前只在 Linux `BwrapSandbox` 有实际 transform；macOS/Windows 尚无对应实现。
- [ ] Git 命令识别仍是保守正则提示，不解析 shell AST、环境中的 `GIT_DIR`、`GIT_WORK_TREE` 或完整外部 bare repo；不能宣称已覆盖主方案 §6.4 的全部 fail-closed 语义。
- [ ] deny mask 对 symlink/junction、复杂 nested carveout、删除/rename/execute 的跨平台执行验证不完整。
- [x] bwrap capability probe、`--unshare-user`、`--new-session` 已实现；`PR_SET_NO_NEW_PRIVS`/seccomp 仍待评估。
- [ ] `VIBECODING_SANDBOX_PASS_ENV` 仍然可以扩展环境白名单，尚未移除隐式扩权入口。
- [ ] Effective sandbox 状态、能力和失败原因尚未统一输出到 TUI、doctor、Serve status/API session status。
- [x] Channels 的全局 sub-agent factory 现在复用 dispatcher 的 sandbox manager，并在创建时设置 standard/none level。
- [x] Public AgentFactory 在 `cfg.SandboxEnabled` 时按 mode 设置 strict/standard/none level，并在不可用时返回错误。

### 13.3 未完成

- [ ] macOS profile 与统一 options 对齐、sandbox-local HOME/XDG、NetworkFull、Git transform 和 Darwin integration tests。
- [ ] Windows 纯 Go Elevated runner、ACL、capability SID、WFP、setup/readiness/repair/teardown、IPC、Job Object。
- [ ] Windows localhost 保留、外部网络阻断和真实 Windows CI 验证。
- [ ] 完整 Git escalation payload（root、operation、scope、callID、TTL、fingerprint）及 ACP 专用权限类型。
- [ ] CLI/Serve/Channels/ACP/factory/sub-agent/workflow/cron 全链路 immutable profile snapshot。
- [ ] 完整 Linux deny 测试矩阵：symlink、delete、rename、execute、未来创建、local listener network、sub-agent/profile consistency。

### 13.4 审查结论

当前实现可称为：**Linux bwrap 基础隔离 + 项目 Git deny/交互式单次放行的实验性实现**，不能称为 proposal 中定义的跨平台完整 Sandbox 方案，也不能宣称 Windows/macOS 已达到目标安全语义。

下一步应继续统一 EffectiveProfile 和 backend capability contract；Channels 全局 sub-agent manager 与 public AgentFactory 的已知旁路已修复。


本方案完成的最低定义不是“有一个新的 sandbox 包”，而是：

- 启用 sandbox 的 CLI、ACP、Serve、Channels 均 fail-closed；
- strict/standard 的实际 filesystem/network/deny 结果可由执行级测试证明；
- `.git` 在非-yolo profile 下默认完整拒绝，且仅通过单次、最小范围升级访问；
- 主 Agent、sub-agent、workflow、cron 不存在 profile 漂移或无 sandbox 旁路；
- Linux/macOS/Windows 的 UI 与 API 只报告真实能力；未实现平台明确 unavailable；
- settings、doctor、日志和用户文档都能解释“请求了什么、实际生效什么、为什么失败或需要升级”。
