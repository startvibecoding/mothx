# SkillHub / ClawHub 市场集成方案

> 状态: Defined（Phase 1–4 的已规划功能已落地；扩展市场与企业 registry 的接入规范见 `docs/proposal/skillhub-market-protocol.md`，adapter 实现不在当前范围）
> 日期: 2026-07-14
> 最近核对: 2026-07-16
> 目标: 在 MothX 中内置 Skill 市场浏览、搜索、下载、安装能力，并同时提供 TUI 与 Web UI 入口。
> 输入资料: `docs/proposal/skillhub-api-research.md`、ClawHub 官方 HTTP API 文档。

## 0. 当前实施状态

### 0.1 已完成

- [x] `internal/skillhub` 统一数据模型、market capability 和 service。
- [x] SkillHub.cn / ClawHub.ai 搜索、列表、详情、文件清单和下载 adapter。
- [x] 兼容两个市场当前实际响应中的对象型 `latestVersion`、对象型 `tags` 和嵌套 `stats`。
- [x] Official 默认账号确定为 `@user_0064faa7`；拉取该用户全部 Skill 后按下载量降序，本地分页。
- [x] SkillHub.cn Browse 使用服务端 `sortBy=downloads&order=desc` 并支持页码分页。
- [x] 列表显示下载量及 `[official]`、`[certified]`、`[verified]`、`[enterprise]`、`[risk]`、`[installed]`、`[update]` 标记。
- [x] 安全 zip 安装：大小/数量限制、路径穿越/绝对路径/Windows drive/symlink 拒绝、根目录识别、`SKILL.md` 校验、metadata、备份与 rename。
- [x] 全局/项目安装目录选择和 managed Skill 本地索引。
- [x] `/skillhub` TUI 覆盖页、双市场 tab、Official/Browse/Search、搜索、详情、文件列表、安装、安装并激活。
- [x] Official、SkillHub.cn Browse 页码分页，以及 ClawHub Browse/Search cursor 前后翻页。
- [x] 详情展示来源、分类、tags、安全扫描状态摘要和 evaluation 维度摘要。
- [x] SkillHub.cn Browse 分类过滤与 `downloads/stars/installs/score/updated_at` 降序切换。
- [x] `d` 详情明细视图：完整文件清单、安全报告 JSON、evaluation JSON 和滚动。
- [x] 详情展示主下载端点和回退来源；TUI `u` 更新 managed Skill，并在原 Skill 已激活时重载上下文。
- [x] `/skillhub search|detail|install|installed` 命令入口。
- [x] 安装后重载 `skillsMgr`、更新 `skill_ref`；激活后重建 extra context 和 agent。
- [x] `settings.skillHub.defaultMarket/defaultInstallScope/officialHandles` additive 配置。
- [x] API fixture、安装、恶意 zip、metadata、官方聚合、分页、下载量和认证字段测试。
- [x] Serve `/api/skillhub/...`：markets、categories、official、search、detail/files、installed、install/update、activate。
- [x] Serve workDir 复用 `allowedWorkDirs` 校验；安装目标只从全局目录或已校验项目目录推导，不接受任意 `TargetDir`。
- [x] Serve session 级 active skills、`SkillsMgr` / `skill_ref` / extra context / `AgentMgr` factory 刷新；`/skill` 也会真正注入当前 session。
- [x] Web UI 一级 `Skills` 工作台、双市场、Official/Browse/Search、分页/筛选/排序、下载量/认证标记、详情、安装、更新与激活。
- [x] Web UI 中英文文案和桌面/移动响应式布局。

### 0.2 部分完成

- [x] 详情：摘要与 `d` 明细视图已覆盖来源、分类、tags、完整文件清单、安全报告、evaluation 和实际下载回退来源。
- [x] 本地状态：识别 managed Skill 并按远端版本计算 `UpdateAvailable`；无 metadata 的手写 Skill 按 `local` 合并到市场状态。
- [x] 分页：SkillHub.cn Browse / Official 页码分页和 ClawHub Browse/Search cursor 翻页均已接入 TUI。
- [~] 排序：SkillHub.cn Browse 和 Official 已按下载量排序；Search 保持相关性；ClawHub 当前公开 API 未验证出可靠的下载量全局排序，不在 UI 中宣称支持。
- [x] 配置：默认项、Official handles、自定义内置市场、启停、API URL、Bearer token 和 Web UI 设置编辑入口已实现；扩展市场与企业 registry/token 的协议和安全语义见 `docs/proposal/skillhub-market-protocol.md`，本阶段不实现新 adapter。
- [x] 覆盖/更新：安装器仅允许覆盖相同 market/id 的 managed Skill，支持备份和回滚；TUI 提供 Update 操作和版本更新提示。
- [~] 安全目录边界：TUI 只使用全局或当前项目 skills 目录；Serve 已校验 `allowedWorkDirs` 并禁止客户端指定 `TargetDir`；核心安装请求仍保留 `TargetDir` 以支持受信任调用和测试。

### 0.3 剩余目标

- [x] 短 TTL 内存缓存，并在安装/卸载后失效；缓存读写做 slice 隔离。
- [x] SkillHub.cn showcase service 编排，以及 ClawHub 文件内容读取 service 编排。
- [x] managed 与无 metadata 的本地 Skill 状态扫描和远端结果合并。
- [x] 卸载、SkillSet 批量安装、失败回滚、多 Skill 激活。
- [x] settings.skillHub markets 配置、启停、API URL、Bearer token 和 Web UI 设置编辑入口。
- [x] TUI SkillSet/uninstall 命令、Web UI Showcase/批量安装/文件内容/卸载入口。
- [x] 本地 HTTP fixture 的自定义 URL、Bearer token、ClawHub 文件内容测试。
- [x] 扩展市场和企业 registry/token 的协议与安全语义已定义，详见 `docs/proposal/skillhub-market-protocol.md`；adapter 实现不在当前范围。

### 0.4 建议后续顺序

2. 当前方案完成；未来需要接入新市场时，按 `docs/proposal/skillhub-market-protocol.md` 实现 adapter。

## 1. 目标

做一个统一的 Skill Marketplace，用户可以在同一个入口里通过 tab 切换不同市场：

- `skillhub.cn`: 中国区 SkillHub 市场。
- `clawhub.ai`: 海外 ClawHub 市场。

核心能力：

1. 浏览市场首页/列表。
2. 搜索 skill。
3. 查看详情、文件清单、评测/安全信息。
4. 下载并安装到全局或当前项目 skill 目录。
5. 展示本地已安装状态。
6. 安装后可在当前 TUI / Web UI session 中激活 skill。
7. 展示官方推荐 skill。官方推荐走 `skillhub.cn` 的用户技能列表接口，按配置的官方用户 handle 拉取并聚合。

MVP 不做：

- 发布 skill。
- 登录、商户、企业私有 registry。
- 评分、收藏、评论。
- 自动执行 skill 包内脚本。
- 后台自动更新所有 skill。

这些可以作为后续阶段扩展。

## 2. 产品形态

### 2.1 统一名称

用户侧名称建议用 `SkillHub`，界面里称为 `Skills` 或 `SkillHub`。

内部代码建议用更中性的 `skillhub` 包名，但数据模型里保留 `Market` 概念，避免把实现绑死在单个站点上。

### 2.2 TUI

新增 `/skillhub` 命令，打开一个 Bubble Tea 子页面。

页面结构：

```text
SkillHub
[ SkillHub.cn ] [ ClawHub.ai ]

[ Official ] [ Browse ] [ Search ]

Search: go testing________________________  Sort: Downloads  Scope: Project  Page: 1

┌ Results ─────────────────────────────┐ ┌ Detail ───────────────────────────────┐
│ [official] go-expert  DL 12,300 1.2.0 │ │ go-expert                              │
│ [certified] tdd-helper DL 823    0.4.1 │ │ Author: xxx     Downloads: 12.3k       │
│             ...                       │ │ Summary: ...                           │
└───────────────────────────────────────┘ │ Files: SKILL.md, references/...        │
                                          │ Security: verified / reports / warnings│
                                          │ [Install] [Install & Activate]         │
                                          └────────────────────────────────────────┘
```

建议按键：

| 按键 | 行为 |
|---|---|
| `Tab` / `Shift+Tab` | 切换市场 tab |
| `[` / `]` | 切换当前市场下的 Official / Browse / Search 子视图 |
| `/` | 聚焦搜索框 |
| `Enter` | 打开当前 skill 详情 |
| `d` | 打开/关闭可滚动的详情、安全报告和 evaluation 明细 |
| `Left` / `Right` | SkillHub.cn Browse / Official 页码翻页；ClawHub Browse / Search cursor 翻页 |
| `c` / `s` | SkillHub.cn Browse 切换分类 / 排序 |
| `i` | 安装到当前 scope |
| `u` | 更新有新版本的 managed Skill |
| `a` | 安装并激活到当前 session |
| `g` / `p` | 切换全局 / 项目安装 scope |
| `r` | 刷新 |
| `Esc` | 返回聊天 |

同时保留轻量命令，方便高级用户和脚本化：

```text
/skillhub search <query>
/skillhub install <market>/<slug> [--global|--project] [--activate]
/skillhub detail <market>/<slug>
/skillhub installed
```

当前排序与标记规则：

- SkillHub.cn Browse 使用服务端下载量降序。
- SkillHub.cn Browse 可循环切换分类，以及 `downloads/stars/installs/score/updated_at` 降序。
- Official 默认聚合 `@user_0064faa7`，拉取全部发布 Skill 后按下载量降序并本地分页。
- Search 保持相关性排序，因为 SkillHub.cn 搜索接口会忽略下载量排序参数。
- ClawHub 当前只展示 `stats.downloads`，不宣称支持按下载量全局排序。
- `[official]` 表示配置的 Official 账号来源；`[certified]` 表示 `publisher.verified`；`[verified]` 表示 Skill 认证字段；`[risk]` 表示可疑风险。来源与认证不得混为同一状态。

### 2.3 Web UI

新增 `Skills` 侧边栏入口，路由建议为 `#/skills`。

页面结构：

- 顶部 market tabs: `SkillHub.cn` / `ClawHub.ai`。
- `SkillHub.cn` 下增加二级 tab：`Official` / `Browse` / `Search`。`Official` 是默认入口，用官方用户列表聚合推荐 skill。
- 搜索框 + 分类 + 排序 + 安装 scope。
- 列表区：名称、作者、版本、下载数、star、安全状态、是否已安装。
- 详情抽屉或右侧面板：简介、标签、文件清单、评测、安全报告、下载来源。
- 操作按钮：`Install`、`Install & Activate`、`Update`、`Uninstall`。

Web UI 不直接访问第三方站点下载 zip，统一走 serve 后端代理，原因：

- 避免浏览器 CORS 和下载重定向差异。
- 安装需要写本地文件，必须由本地 serve 进程执行。
- 后端可以统一做 zip 安全校验、路径校验、覆盖策略和本地状态刷新。

## 3. 后端架构

新增包：

```text
internal/skillhub/
  types.go             # 统一数据模型
  service.go           # 业务编排: search/detail/install/installed
  install.go           # 下载、校验、解压、原子安装
  local.go             # 扫描本地 installed 状态
  http.go              # HTTP JSON 请求和错误处理
  client_helpers.go    # adapter 公共辅助函数
  skillhubcn.go        # skillhub.cn adapter
  clawhub.go           # clawhub.ai adapter
  cache.go             # 短 TTL 内存缓存
```

### 3.1 统一接口

```go
type MarketClient interface {
    Market() MarketInfo
    Search(ctx context.Context, q SearchQuery) (SearchPage, error)
    UserSkills(ctx context.Context, handle string, q UserSkillsQuery) (SearchPage, error)
    Detail(ctx context.Context, id SkillID) (SkillDetail, error)
    Files(ctx context.Context, id SkillID, version string) ([]SkillFile, error)
    Evaluation(ctx context.Context, id SkillID) (any, error)
    Download(ctx context.Context, id SkillID, version string) (io.ReadCloser, DownloadMeta, error)
    Categories(ctx context.Context) ([]Category, error)
}
```

不同市场能力不一致，统一模型要显式表达 capability：

```go
type MarketCapabilities struct {
    Search           bool
    List             bool
    CursorPagination bool
    PagePagination   bool
    Categories       bool
    Showcase         bool
    AuthorFilter     bool
    UserSkills       bool
    FileList         bool
    FileContent      bool
    Evaluation       bool
}
```

UI 根据 capability 控制过滤项和分页行为，不要把某个市场不支持的参数硬塞给后端。

### 3.2 统一数据模型

核心字段：

```go
type SkillSummary struct {
    Market      string
    ID          string
    Slug        string
    Name        string
    DisplayName string
    Description string
    Version     string
    Author      string
    Category    string
    Tags        []string
    IconURL     string
    Homepage    string
    SourceURL   string
    Source      string
    PublisherName string
    CertifiedName string
    PublisherVerified bool
    Downloads   int64
    Installs    int64
    Stars       int64
    Score       float64
    Verified    bool
    Suspicious  bool
    UpdatedAt   time.Time
    Installed   *InstalledState
}
```

详情在 summary 基础上补：

- `Readme` / `SkillContentPreview`，如果市场支持文件读取。
- `Files`，用于展示安装内容和校验。
- `SecurityReports` / `Evaluation`，如果市场支持。
- `DownloadSources`，用于展示直接下载、回退下载、签名状态。

### 3.3 skillhub.cn adapter

基于现有调研：

- 搜索优先使用 `GET https://api.skillhub.cn/api/v1/search?q=...&limit=...`。
- 空查询/列表使用 `GET https://api.skillhub.cn/api/skills?page=...&pageSize=...`。
- 首页榜单使用 `GET /api/v1/showcase/{recommended,hot,newest,trending}`。
- 官方推荐使用 `GET /api/v1/users/{handle}/skills?page=...&pageSize=...`，按配置的官方 handle 列表逐个拉取后在客户端聚合。
- 分类使用 `GET /api/v1/categories`。
- 详情使用 `GET /api/v1/skills/{slug}`。
- 文件清单使用 `GET /api/v1/skills/{slug}/files`。
- 下载使用两级回退：
  1. `GET /api/v1/download?slug={slug}`
  2. `GET https://skillhub-1388575217.cos.ap-guangzhou.myqcloud.com/skills/{slug}.zip`

注意点：

- `/api/v1/search` 不支持分页、分类、排序、用户过滤。
- `/api/skills` 的 `sortBy` 只能是 `updated_at/downloads/stars/installs/score`。
- `/api/v1/users/{handle}/skills` 是按用户拉 skill 的正确接口，但只支持 `page/pageSize`；关键词、分类和排序需要本地过滤。
- 当前默认 Official handle 为 `user_0064faa7`。实现会拉取其完整列表、按 `downloads` 降序后再做 TUI 分页。
- `/api/v1/skills/{slug}` 当前部署的 `latestVersion` 和 `tags` 可能是对象，统计位于 `stats`；adapter 已同时兼容旧字符串/数组形式。
- v1 错误是 HTTP 状态码 + `{"error":"..."}`；旧 `/api/skills` 是 `code != 0`。
- 时间戳是毫秒。

### 3.4 clawhub.ai adapter

按 ClawHub 官方 HTTP API：

- 列表/搜索使用 `GET /api/v1/skills`。
- 支持 `query`、`limit`、`cursor`、`order`、`author`、`verifiedOnly`、`nonSuspiciousOnly` 等参数。
- 详情使用 `GET /api/v1/skills/{skill-id}`。
- 文件清单使用 `GET /api/v1/skills/{skill-id}/files`。
- 文件内容使用 `GET /api/v1/skills/{skill-id}/files/{path}`。
- 下载使用 `GET /api/v1/skills/{skill-id}/download`。
- 安全扫描/验证接口可用于展示风险状态。

注意点：

- ClawHub 的分页是 cursor，不是 page/pageSize。
- 错误响应可能是 `text/plain`，不能假设都是 JSON。
- `skill-id` 需要统一支持 slug 或 `org/slug` 形式，避免跨组织重名。
- 当前列表与详情响应提供嵌套 `stats.downloads`，但没有稳定返回可直接展示的 verified/suspicious 布尔字段；没有字段时 UI 不生成认证标记。
- 当前实测 `order=downloads` 未形成可靠的下载量全局排序，因此首版只展示下载量，保留 API 默认顺序。
- TUI 已保存 cursor 历史栈，支持 `Right` 前进和 `Left` 返回上一 cursor 页。

## 4. Serve API

新增路由挂在 `internal/serve` 的 `rt.routes(...)` 下：

```http
GET  /api/skillhub/markets
GET  /api/skillhub/categories?market=skillhub.cn
GET  /api/skillhub/official?market=skillhub.cn&q=&limit=50
GET  /api/skillhub/search?market=skillhub.cn&q=go&limit=20&cursor=...
GET  /api/skillhub/skills/{market}/{id}
GET  /api/skillhub/skills/{market}/{id}/files
GET  /api/skillhub/installed?workDir=/path/to/project
POST /api/skillhub/install
POST /api/skillhub/activate
DELETE /api/skillhub/installed/{name}?scope=project&workDir=...
```

`POST /api/skillhub/install` 请求：

```json
{
  "market": "skillhub.cn",
  "id": "go-expert",
  "version": "",
  "scope": "project",
  "workDir": "/repo",
  "overwrite": false,
  "activate": false
}
```

响应：

```json
{
  "name": "go-expert",
  "market": "skillhub.cn",
  "version": "1.2.0",
  "scope": "project",
  "dir": "/repo/.mothx/skills/go-expert",
  "installed": true,
  "activated": false,
  "warnings": []
}
```

`POST /api/skillhub/activate` 需要对当前 session 生效：

- TUI 已有 `activeSkills` 和 `rebuildExtraContext()`，安装后可复用。
- Serve 需要补齐 session 级 active skills 状态，激活后重建 `sess.ExtraContext`，并重置当前 session agent，使下一次请求使用新的 frozen system prompt。
- Web UI 应明确展示“已安装”和“已在当前 session 激活”是两个不同状态。

## 5. 安装策略

### 5.1 安装位置

复用现有 skills 加载逻辑：

- 全局：`settings.GetGlobalSkillsDir()`。
- 项目：`skills.ProjectSkillDirs(workDir)[0]`，即当前项目最高优先级 skill 目录。

不要手写 `.skills` 路径。这样能保持和现有优先级一致：

```text
.mothx/skills > .skills > skills > global
```

### 5.2 解压与校验

安装流程：

1. 下载 zip 到临时目录。
2. 限制响应大小、解压后总大小、文件数量、单文件大小。
3. 解压前检查每个 entry：
   - 禁止绝对路径。
   - 禁止 `..` 路径穿越。
   - 禁止 Windows drive path。
   - 默认拒绝 symlink。
4. 自动识别 zip 根目录：
   - 如果根目录下直接有 `SKILL.md` / `skill.md`，安装该根。
   - 如果只有一个顶层目录且里面有 `SKILL.md` / `skill.md`，剥离顶层目录安装。
5. 校验必须存在 `SKILL.md` 或 `skill.md`。
6. 写入元数据文件，例如 `.mothx-skillhub.json`：

```json
{
  "market": "skillhub.cn",
  "id": "go-expert",
  "slug": "go-expert",
  "version": "1.2.0",
  "installedAt": "2026-07-14T10:00:00Z",
  "sourceURL": "https://..."
}
```

7. 安装到临时目录后用 rename 原子替换。

### 5.3 覆盖与更新

MVP 策略：

- 如果目标目录不存在，直接安装。
- 如果目标目录存在且没有 `.mothx-skillhub.json`，默认拒绝覆盖，提示这是本地手写 skill。
- 如果目标目录存在且来自同一 market/id：
  - 版本相同：显示已安装。
  - 版本不同：提供 `Update`。
- 覆盖时先备份到同级 `.backup/<name>-<timestamp>` 或临时目录，安装失败则回滚。

### 5.4 激活语义

安装不等于激活。

- `Install`: 只写入 skill 目录并刷新本地 skill manager。
- `Install & Activate`: 写入后立即加入当前 session 的 active skills，并重建 agent。

原因：agent 的 system prompt 是 frozen 的，安装后必须重置 agent，下一轮请求才能带上新 skill。

## 6. 配置

建议新增 additive 配置，不改变现有字段含义：

```json
{
  "skillHub": {
    "defaultMarket": "skillhub.cn",
    "defaultInstallScope": "project",
    "officialHandles": ["user_0064faa7"],
    "markets": [
      {
        "id": "skillhub.cn",
        "name": "SkillHub.cn",
        "siteURL": "https://skillhub.cn",
        "apiURL": "https://api.skillhub.cn",
        "enabled": true
      },
      {
        "id": "clawhub.ai",
        "name": "ClawHub.ai",
        "siteURL": "https://clawhub.ai",
        "apiURL": "https://clawhub.ai",
        "enabled": true
      }
    ]
  }
}
```

当前实现范围：`defaultMarket`、`defaultInstallScope`、`officialHandles` 和 `markets` 已进入 `settings.json` schema；内置 SkillHub.cn / ClawHub.ai 支持启停、API URL 和 Bearer token 配置，Web UI 提供编辑入口。扩展市场与企业 registry/token 的协议和安全语义见 `docs/proposal/skillhub-market-protocol.md`；本阶段不实现新的运行时 adapter。

配置位置：

- `settings.json`: app 级能力，TUI 和 serve 都会用，适合作为默认配置来源。
- `serve.json`: 不新增重复字段；serve 读取 settings 后注入 runtime。

如果不想第一阶段改 settings schema，也可以先用代码内置默认市场，后续再做可配置市场。

官方推荐默认 handle 已确定为 `user_0064faa7`。JSON 配置和 Web UI 可覆盖 handle 列表；TUI 使用同一配置来源。

## 7. 安全边界

Skill 市场安装会写本地文件，必须和普通下载区别对待：

- 只允许写入全局 skillsDir 或当前 workDir 的项目 skill 目录。
- Web API 的 `workDir` 必须走现有 `allowedWorkDirs` 校验。
- 下载使用固定超时和大小限制。
- 不执行包内脚本、postinstall、binary。
- zip 中 executable bit 可以保留但要提示；MVP 可以不做执行权限保留，降低风险。
- 详情页展示来源、版本、作者、安全报告和 warning。
- 对同名本地 skill 默认不覆盖。

### 7.1 Bash / Sandbox 运行时策略（待实施）

Skill 安装器的目录与 zip 安全边界应保持严格，但不应将 Bash 工具可用性绑定到某个 sandbox 后端是否可用。当前的 bwrap 配置或平台兼容问题会使 `ps`、`hostname` 等基础诊断命令整体失败，尤其影响 Windows + BusyBox 环境；应按以下模型简化。

**职责分离：**

- mode 只决定 Bash 是否注册和是否需要审批：`plan` 不注册 Bash；`agent` 按 approval 策略执行；`yolo` 自动执行。
- sandbox 是独立、可选的执行防护，只决定命令是否经隔离 backend 执行。
- backend 只负责声明并验证能否真实实施所要求的文件系统、网络和 namespace 约束；不得把仅裁剪环境变量的实现宣称为安全 sandbox。

建议将面向用户的策略收敛为：

```json
{
  "sandbox": {
    "enabled": false,
    "strict": false
  }
}
```

- `enabled=false`：使用本机 shell 直接执行；这是 CLI、TUI、Serve 和 channel 的默认值。
- `enabled=true, strict=false`：尽力执行**进程空间隔离**。Linux 上 bwrap 完整 profile 可用则启用；bwrap 不存在、运行 profile 校验失败、Windows/BusyBox 不支持时，回退到直接执行，并仅提示一次 sandbox 未生效的明确警告。
- `enabled=true, strict=true`：强制进程空间隔离。backend 必须通过完整 profile 校验；不可用时拒绝执行并返回可操作错误，绝不静默降级。
- sandbox 不隔离网络：不使用 `--unshare-net`，命令保留宿主网络访问能力。普通 sandbox 项目可写，strict sandbox 项目只读。

bwrap 可用性不能只检查二进制存在或运行 `/bin/true`，必须验证实际使用的 profile：所需 flags、user/pid/ipc/uts namespace、`--new-session`、`--die-with-parent`、隔离的 `/proc`、`/dev`、受限 `/tmp`、HOME、系统只读挂载、项目 bind/mask、hostname、workdir 与 shell 执行。该 profile 不使用 `--unshare-net`，以保留宿主网络访问。可用 bwrap 环境中必须回归验证 `ps`、`hostname`、`/proc` 读取、strict 项目只读和 deny path；Windows/BusyBox 不应尝试调用 bwrap。

默认值已确定：CLI、TUI、Serve 与 channel 均为 `enabled=false`，保证本地开发、诊断和 unattended agent 的可用性；只有用户或企业策略明确要求时才启用 sandbox，且只有明确要求强隔离时才设置 `strict=true`。

## 8. 本地状态同步

`internal/skills.Manager` 当前只负责加载本地 skill，不记录市场来源。

新增 `skillhub.LocalIndex`：

- 扫描全局和项目 skill 目录。
- 读取 `.mothx-skillhub.json` 得到 market/id/version。
- 没有 metadata 的本地 skill 仍显示为 `local`。
- 给远端搜索结果补 `InstalledState`：

```go
type InstalledState struct {
    Installed bool
    Scope     string
    Dir       string
    Version   string
    Active    bool
    UpdateAvailable bool
}
```

安装、卸载、更新后：

- TUI: 重新 `skillsMgr.Load()`，刷新 `/skills` 列表和 marketplace installed badge。
- Serve: 重新构建对应 workDir 的 `SkillsMgr`，并更新 session registry 里的 `skill_ref` tool。

## 9. 分期

### Phase 1: Core + installer（核心验收已完成，增强项待补）

交付：

- `internal/skillhub` 数据模型和 service。
- `skillhub.cn` adapter。
- `clawhub.ai` adapter。
- 官方推荐聚合逻辑：按 `officialHandles` 调用 `skillhub.cn` 用户技能列表接口，并补齐 installed 状态。
- 安装器、zip 安全校验、本地 installed 扫描。
- focused tests：API fixture、zip traversal、覆盖策略、metadata、安装目录选择。

验收：

- 能在测试 fixture 下搜索两个市场。
- 能通过配置的官方 handle 拉到 SkillHub.cn 官方推荐列表。
- 能把一个 zip skill 安装到临时项目目录。
- 恶意 zip 不会写出目标目录。

Phase 1 的范围已完成；扩展市场与企业 registry 的 adapter 仅在未来按 `docs/proposal/skillhub-market-protocol.md` 接入时实现。

### Phase 2: TUI（主流程已完成，体验与集成测试待补）

交付：

- `/skillhub` 页面。
- tab 切换两个市场。
- 搜索、详情、安装、安装并激活。
- 命令形式 fallback。

验收：

- 在 TUI 里能从两个市场搜索并安装 skill。
- 安装后 `/skills` 能看到新 skill。
- `Install & Activate` 后下一轮 agent prompt 包含该 skill。

TUI 命令和本地 HTTP fixture 已覆盖核心状态机。

### Phase 3: Serve API + Web UI（主流程已完成）

交付：

- `/api/skillhub/...` 路由。
- `ui/src/views/Skills.svelte`。
- 侧边栏 Skills 入口和 i18n。
- Web UI 详情、安装、激活、已安装状态。

验收：

- Web UI 可 tab 切换 `SkillHub.cn` / `ClawHub.ai`。
- 搜索结果和 TUI 使用同一后端模型。
- 安装写入当前 session workDir 对应项目 skill 目录。
- 激活后当前 session agent 会重建。

当前状态：Serve API、目录校验、session 激活刷新、Web UI 工作台、双市场浏览/搜索/详情、Showcase、批量安装、文件内容和卸载入口均已完成。扩展市场和企业 registry 的接入规范已定义，adapter 不在当前范围。

### Phase 4: 卸载、批量与扩展市场（后端基础能力已完成，扩展项收尾中）

交付：

- [x] 卸载。
- [x] 批量安装 SkillSet、失败回滚和多 Skill 激活。
- [x] 企业 registry / token 规范：认证、最小权限、同源认证头和凭证不落盘/不回显语义已在 `docs/proposal/skillhub-market-protocol.md` 定义；本阶段不实现 adapter。
- [x] 可配置自定义 market 规范：`mothx-market-v1` 协议、能力协商和配置语义已在 `docs/proposal/skillhub-market-protocol.md` 定义；本阶段不实现 adapter。

## 10. 决策与待确认问题

1. **已确认**：项目安装目录使用 `skills.ProjectSkillDirs(workDir)[0]`，即 `.mothx/skills`，保持现有优先级。
2. **待确认**：Web UI 是否把 `Skills` 放一级导航，还是放到 Settings / Project Context 下。当前建议仍是一级导航。
3. **已确认（MVP）**：SkillHub.cn 的 `source=clawhub` 与独立 `clawhub.ai` 不跨市场去重，保留 market/source 标识。
4. **部分验证**：ClawHub 公开列表、详情和下载接口按无需登录设计；正式接入 Serve/Web 前仍需做真实公开包下载集成测试。
5. **已实现**：Serve 保存 session 级 active skills，激活/安装/更新后重建 `SkillsMgr`、`skill_ref`、`sess.ExtraContext` 和 `AgentMgr` factory。Serve 每次请求创建新的主 agent，因此下一轮会直接使用刷新后的 frozen system prompt。
6. **已确认**：Official 默认 handle 为 `user_0064faa7`，配置结构仍支持多个 handle。
7. **已确认**：列表展示下载量。SkillHub.cn Browse 与 Official 按下载量降序；Search 保持相关性；ClawHub 未确认可靠全局下载排序前只展示下载量。
8. **已确认**：Official 来源、publisher 认证、Skill verified 和 risk 必须使用独立标记，不能相互替代。

## 11. 参考

- `docs/proposal/skillhub-market-protocol.md`
- `docs/proposal/skillhub-api-research.md`
- ClawHub HTTP API: https://docs.openclaw.ai/clawhub/http-api
