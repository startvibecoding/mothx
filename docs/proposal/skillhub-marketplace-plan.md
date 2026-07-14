# SkillHub / ClawHub 市场集成方案

> 状态: Draft
> 日期: 2026-07-14
> 目标: 在 MothX 中内置 Skill 市场浏览、搜索、下载、安装能力，并同时提供 TUI 与 Web UI 入口。
> 输入资料: `docs/proposal/skillhub-api-research.md`、ClawHub 官方 HTTP API 文档。

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

Search: go testing________________________  Sort: Relevance  Scope: Project

┌ Results ─────────────────────────────┐ ┌ Detail ───────────────────────────────┐
│ [installed] go-expert        1.2.0    │ │ go-expert                              │
│             Go testing...    823 ★    │ │ Author: xxx     Downloads: 12.3k       │
│             tdd-helper       0.4.1    │ │ Category: dev-programming              │
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
| `i` | 安装到当前 scope |
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
  cache.go             # 短 TTL 内存缓存
  skillhubcn/client.go # skillhub.cn adapter
  clawhub/client.go    # clawhub.ai adapter
```

### 3.1 统一接口

```go
type MarketClient interface {
    Market() Market
    Search(ctx context.Context, q SearchQuery) (SearchPage, error)
    UserSkills(ctx context.Context, handle string, q UserSkillsQuery) (SearchPage, error)
    Detail(ctx context.Context, id SkillID) (SkillDetail, error)
    Files(ctx context.Context, id SkillID, version string) ([]SkillFile, error)
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
    "officialHandles": ["mothx-official"],
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

配置位置：

- `settings.json`: app 级能力，TUI 和 serve 都会用，适合作为默认配置来源。
- `serve.json`: 不新增重复字段；serve 读取 settings 后注入 runtime。

如果不想第一阶段改 settings schema，也可以先用代码内置默认市场，后续再做可配置市场。

官方推荐的 handle 列表建议先内置默认值，后续再暴露到设置页。这样首版可以先落地官方推荐，不要求用户理解 SkillHub 用户 handle。

## 7. 安全边界

Skill 市场安装会写本地文件，必须和普通下载区别对待：

- 只允许写入全局 skillsDir 或当前 workDir 的项目 skill 目录。
- Web API 的 `workDir` 必须走现有 `allowedWorkDirs` 校验。
- 下载使用固定超时和大小限制。
- 不执行包内脚本、postinstall、binary。
- zip 中 executable bit 可以保留但要提示；MVP 可以不做执行权限保留，降低风险。
- 详情页展示来源、版本、作者、安全报告和 warning。
- 对同名本地 skill 默认不覆盖。

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

### Phase 1: Core + installer

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

### Phase 2: TUI

交付：

- `/skillhub` 页面。
- tab 切换两个市场。
- 搜索、详情、安装、安装并激活。
- 命令形式 fallback。

验收：

- 在 TUI 里能从两个市场搜索并安装 skill。
- 安装后 `/skills` 能看到新 skill。
- `Install & Activate` 后下一轮 agent prompt 包含该 skill。

### Phase 3: Serve API + Web UI

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

### Phase 4: 更新、卸载、扩展市场

交付：

- 更新检查。
- 卸载。
- 批量安装 skillset。
- 企业 registry / token 支持。
- 可配置自定义 market。

## 10. 需要提前确认的问题

1. 项目安装目录默认是否使用当前代码最高优先级 `.mothx/skills`，还是为了用户直觉强制用 `.skills`。建议用 `skills.ProjectSkillDirs(workDir)[0]` 保持一致。
2. Web UI 是否把 `Skills` 放一级导航，还是放到 Settings / Project Context 下。建议放一级导航，因为它是高频安装入口。
3. SkillHub.cn 搜索结果中的 `source=clawhub` 与独立 `clawhub.ai` 市场可能有重复 skill，是否需要跨市场去重。MVP 不去重，只在详情页显示 market/source。
4. 是否允许未登录下载 ClawHub 全部公开 skill。按官方文档可以先实现公开 API，登录能力后置。
5. 现有 Serve `/skill` 激活逻辑需要补齐 session active skills 状态；否则 Web UI 的 `Activate` 只能显示成功但不会改变 frozen prompt。
6. 官方推荐默认 handle 用哪个账号或哪些账号。技术方案按 `officialHandles` 支持多个 handle，产品上需要确定初始列表。

## 11. 参考

- `docs/proposal/skillhub-api-research.md`
- ClawHub HTTP API: https://docs.openclaw.ai/clawhub/http-api
