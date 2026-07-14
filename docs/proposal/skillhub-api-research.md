# SkillHub.cn 公开接口调研文档

> 调研日期：2026-07-14
> 站点：https://skillhub.cn （API Host：https://api.skillhub.cn）
> 调研方式：抓取前端 bundle、分析官方 CLI（`~/.skillhub/skills_store_cli.py`）、对接口做实际请求验证。
> 状态：非官方文档，基于黑盒探测 + 前端代码反推，接口契约可能随版本变化。

---

## 0. 基础信息

### 0.1 API Host
- 官网前端：`https://skillhub.cn`（SPA，静态 HTML + JS 包）
- API Host：`https://api.skillhub.cn`（前端 `Wc` 变量指向此地址；也存在备用 `https://api.skillhub.tencent.com`）
- 静态资源 CDN：`https://cloudcache.tencent-cloud.com`
- 包存储（腾讯云 COS）：`https://skillhub-1388575217.cos.ap-guangzhou.myqcloud.com`

### 0.2 通用约定
- 所有公开接口均**无需鉴权**即可 GET；涉及发布、商户、企业、登录态的接口需 Cookie/Token。
- 响应格式：JSON。
  - 老接口（`/api/...`）多采用 `{ "code": 0, "data": {...}, "message": "..." }` 包裹，`code=0` 为成功。
  - v1 接口（`/api/v1/...`）直接返回业务对象，错误时返回 `{ "error": "..." }` 并使用标准 HTTP 状态码（如 404/405）。
- 参数传递：GET 查询参数；批量接口用 POST + JSON body。
- 时间字段：Unix 毫秒时间戳（如 `createdAt: 1773284580768`）。
- CLI User-Agent：`SkillHub-CLI/<version>`（可自定义，普通浏览器 UA 也可访问）。

---

## 1. 技能搜索与列表

### 1.1 站内全量列表/分页检索：`GET /api/skills`

前端 `/skills` 页面使用的主列表接口（非全文检索，字段过滤型）。

**Query 参数**（均可选，来自前端 `fetchSkillsPage` 源码）：

| 参数 | 类型 | 说明 |
|---|---|---|
| `page` | int | 页码，默认 1 |
| `pageSize` | int | 每页条数，前端默认 24 |
| `keyword` | string | 关键词（模糊匹配 name/description，**不是** ES 式全文） |
| `sortBy` | string | 排序字段，**仅支持**：`updated_at` / `downloads` / `stars` / `installs` / `score`。传其他值（含 `name`、`newest`、`created_at`、`trending`）返回 `400 {"code":400,"message":"参数错误：sortBy 不支持（updated_at/downloads/stars/installs/score）"}` |
| `order` | string | `asc` / `desc`，默认 `desc` |
| `category` | string | 分类 key，见第 4 节分类枚举 |
| `source` | string | 来源过滤。前端枚举：`all` / `clawhub` / `enterprise` / `official`（传 `all` 或不传即全部） |
| `labels` | string | 标签过滤（前端直接透传字符串，具体枚举未单独暴露；示例 `requires_api_key=true`） |
| `apiKey` | string | 过滤是否需要 API Key：`all`/`required`/`optional`（前端路由参数） |
| `paid` | string | 付费状态过滤：`all`/`free`/`paid`（前端路由参数） |

**响应**：
```json
{
  "code": 0,
  "message": "",
  "data": {
    "total": 77057,
    "skills": [
      {
        "slug": "self-improving-agent",
        "name": "self-improving agent",
        "category": "ai-agent",
        "subCategories": [],
        "description": "...",
        "description_zh": "...",
        "version": "1.0.0",
        "iconUrl": "https://.../xxx.png",
        "homepage": "https://api.skillhub.cn/<owner>/<slug>",
        "ownerName": "pskoett",
        "downloads": 972888,
        "installs": 123456,
        "stars": 4211,
        "score": 0.0,
        "source": "clawhub",
        "verified": false,
        "claimable": false,
        "claim_state": "unclaimed",
        "claimed_user_handle": null,
        "labels": { "requires_api_key": "false" },
        "tags": [],
        "upstream_owner_login": "...",
        "upstream_url": "...",
        "created_at": 1772739772656,
        "updated_at": 1783990604537,
        "last_synced_at": 1783990604537
      }
    ]
  }
}
```

**实测验证**：
- `?sortBy=stars&order=desc` 正确返回 stars 最高的技能。
- `?category=office-efficiency` 返回 total=3984。
- `?source=clawhub` 返回 total=62564。
- **不支持**按用户过滤（传 `owner=`/`handle=`/`author=`/`user=` 等均被忽略，total 仍是全量）。

---

### 1.2 全文搜索（CLI 使用）：`GET /api/v1/search`

CLI `skillhub search <q>` 调用的接口，做的是真正的关键词相关性排序。

**Query 参数**：

| 参数 | 类型 | 说明 |
|---|---|---|
| `q` | string | 搜索关键词。**为空或省略时返回少量默认结果（不返回全量）** |
| `limit` | int | 返回条数，默认/上限未严格验证，CLI 传用户指定值（默认 20 左右） |

**注意（实测结论）**：
- **不支持**分页参数（`page`/`pageSize`/`offset` 均被忽略，固定返回 `limit` 条）。
- **不支持**分类、排序、来源、用户等过滤参数（`category`/`sortBy`/`owner`/`handle`/`user`/`author` 等 11 个候选参数实测全部被忽略）。
- 不支持翻页，若需要"更多结果"只能通过更精确的关键词二次搜索。

**响应**：
```json
{
  "results": [
    {
      "slug": "ws-excel",
      "name": "Excel",
      "displayName": "Excel",
      "description": "Excel 操作。数据处理、公式、表格操作。",
      "description_zh": "...",
      "summary": "...",
      "version": "1.0.0",
      "category": "office-efficiency",
      "owner_name": "fhbillwer",
      "publisher": null,
      "source": "clawhub",
      "homepage": "https://api.skillhub.cn/fhbillwer/ws-excel",
      "icon_url": "https://.../xxx.png",
      "downloads": 15573,
      "installs": 3459,
      "stars": 7,
      "score": 0.1135,
      "labels": { "requires_api_key": "false" },
      "tags": [],
      "created_at": 1773284580768,
      "updated_at": 1783990604537,
      "updatedAt": 1783990604537
    }
  ]
}
```

**关于"按用户搜索"的结论**：
> `/api/v1/search` 与 `/api/skills` 均**不支持**通过 query 参数按用户/作者/owner 过滤。
> 要"查某用户发布的所有技能"必须使用 §2 的专门接口；要"查某用户的技能并按关键词过滤"只能先取全量再在客户端做本地匹配。

---

### 1.3 首页精选/榜单：`GET /api/v1/showcase/{type}`

| type | 含义 | 对应前端标识 |
|---|---|---|
| `recommended` | 官方推荐（即前端的 featured） | `fetchShowcase("featured")` 内部映射为 `recommended` |
| `hot` | 热门 | `fetchShowcase("hot")` |
| `newest` | 最新 | `fetchShowcase("newest")` |
| `trending` | 趋势 | `fetchShowcase("trending")` |

**参数**：无查询参数（固定返回约 100 条）。

**响应**：
```json
{
  "section": "hot",
  "total": 100,
  "skills": [ /* 与 /api/skills 的 skill 对象结构一致 */ ]
}
```

注意：前端对 `source` 为 `enterprise`/`official` 且无 `publisher` 的技能，还会额外批量调用接口补齐 `publisher` 字段（`needsShowcasePublisher` 逻辑）。

---

## 2. 用户与"按用户过滤"

### 2.1 获取用户信息：`GET /api/v1/users/{handle}`

**响应（用户存在，HTTP 200）**：
```json
{
  "user": {
    "handle": "user_0064faa7",
    "displayName": "zhenruyan",
    "avatarUrl": null,
    "bio": null,
    "followersCount": 0,
    "followed": false,
    "createdAt": 1775896147159
  }
}
```

**用户不存在（HTTP 404）**：
```json
{ "error": "user not found" }
```

**关于 handle**：
- 用户可能是自定义 handle（如 `fhbillwer`），也可能是系统生成的 `user_xxxxxxxx` 形式（未设置自定义昵称时）。
- 用户主页 URL：`https://skillhub.cn/{handle}`（SPA，前端把路径段当作 handle 加载）。

---

### 2.2 列出用户所有技能（=按用户过滤的正确方式）：`GET /api/v1/users/{handle}/skills`

**这是"按用户过滤"的唯一官方接口**。

**Query 参数**（均可选）：

| 参数 | 类型 | 默认 | 说明 |
|---|---|---|---|
| `page` | int | 1 | 页码 |
| `pageSize` | int | 20 | 每页条数 |

**实测不支持**：`keyword` / `q` / `category` / `sortBy` / `order` 均被忽略，始终返回该用户全部技能（按默认顺序）。如需按关键词/分类过滤，需在客户端本地过滤。

**响应**：
```json
{
  "count": 3,
  "page": 1,
  "pageSize": 100,
  "skills": [
    {
      "slug": "happyhorse",
      "name": "HappyHorseGiteeAi",
      "description": "Gitee AI 视频生成工具...",
      "description_zh": "...",
      "category": "design-media",
      "version": "1.0.0",
      "iconUrl": "https://...",
      "downloads": 69,
      "installs": 0,
      "stars": 0,
      "labels": { "requires_api_key": "false" },
      "source": "clawhub",
      "createdAt": 1780909163359,
      "updatedAt": 1783990000000
    }
  ]
}
```

**示例**：查 `zhenruyan`（handle=`user_0064faa7`）所有技能：
```bash
curl -s "https://api.skillhub.cn/api/v1/users/user_0064faa7/skills?pageSize=100"
```

**未知 handle**：同 §2.1 返回 HTTP 404 `{"error":"user not found"}`。

---

### 2.3 按用户做关键词过滤的变通方案
```text
1) GET /api/v1/users/{handle}/skills?pageSize=100  → 拿到该用户全部 skills[]
2) 客户端对 item.name / item.description / item.description_zh 做关键词匹配
3) 如需分页，在过滤后的结果上分页即可
```
若用户技能数较多（`count > 100`），可多次按 `page=1..N` 拉取后合并。

---

## 3. 技能详情 / 批量 / 文件 / 下载

### 3.1 技能详情：`GET /api/v1/skills/{slug}`

**响应**：
```json
{
  "skill": {
    "slug": "ws-excel",
    "displayName": "Excel",
    "summary": "...",
    "summary_zh": "...",
    "category": "office-efficiency",
    "subCategories": [],
    "tags": [],
    "iconUrl": "https://...",
    "labels": {},
    "version": "1.0.0",
    "source": "clawhub",
    "sourceUrl": "...",
    "upstream_owner_login": "...",
    "upstream_url": "...",
    "githubAuthorLogin": "...",
    "authorVerifiedHandle": "...",
    "isAuthorVerified": false,
    "verified": false,
    "claim_state": "unclaimed",
    "claimable": false,
    "claimed_user_handle": null,
    "stats": { "downloads": 15573, "installs": 3459, "stars": 7 },
    "createdAt": ...,
    "updatedAt": ...,
    "last_synced_at": ...
  },
  "owner": { /* 发布者信息，同 /api/v1/users/{handle} 中的 user 对象 */ },
  "latestVersion": "1.0.0",
  "contentZhAvailable": true,
  "securityReports": []
}
```

**slug 不存在**：HTTP 404（或 code 非 0，视部署版本）。

> ⚠️ 该接口**不直接返回 README / SKILL.md 正文**。正文需走 §3.3 文件列表 + 下载解压，或直接走下载接口拿 zip。

---

### 3.2 批量查询技能：`POST /api/v1/skills/batch`

**Request**：
```http
POST /api/v1/skills/batch HTTP/1.1
Content-Type: application/json

{ "slugs": ["ws-excel", "happyhorse", "__nope__"] }
```

**Response**：
```json
{
  "count": 2,
  "missing": ["__nope__"],
  "items": [
    { "skill": {...}, "owner": {...}, "latestVersion": "...", "securityReports": [] },
    ...
  ]
}
```

`items[i]` 的结构与 `/api/v1/skills/{slug}` 的顶层一致（含 `skill`/`owner`/`latestVersion`/`securityReports`）。

---

### 3.3 技能文件清单：`GET /api/v1/skills/{slug}/files`

| 参数 | 说明 |
|---|---|
| `version` | 可选，指定版本号；不传返回最新版本 |

**Response**：
```json
{
  "version": "1.0.0",
  "count": 2,
  "files": [
    { "path": "SKILL.md", "sha256": "f7766f8...", "size": 967 },
    { "path": "cli.js",   "sha256": "331f3c3...", "size": 5086 }
  ]
}
```

---

### 3.4 技能评测报告：`GET /api/v1/skills/{slug}/evaluation`

返回官方安全/多维度评测报告，结构：
```json
{
  "createdAt": 1779137766618,
  "dimensions": {
    "adaptability": { "items": { "boundary": { "reason": "...", "score": ... }, ... } },
    ...
  }
}
```
不存在时 HTTP 404。

---

### 3.5 下载技能包：`GET /api/v1/download?slug={slug}`

CLI 元数据里声明的下载端点为 `https://api.skillhub.cn/api/v1/download?slug={slug}`。

**实测注意（坑）**：
- 直接 `curl GET/POST` 都返回 HTTP 405；CLI 的实际下载逻辑有**多级回退**：
  1. 先尝试主下载 URL（上面那个 API 端点）
  2. 失败则回退到 COS 直链：`https://skillhub-1388575217.cos.ap-guangzhou.myqcloud.com/skills/{slug}.zip`
- 浏览器/带签名的场景（企业版/商户版）可能需要签名参数 `fetchSkillSignature` 返回的 `sig`（在 CLI 中有 publicGet + signed 逻辑）。
- 企业源下载接口：`GET {host}/api/v1/orgs/{orgId}/registry/skills/{slug}/download`（需 API Key Header）。

**技能包格式**：zip，根目录至少含 `SKILL.md`，可选 `references/`、`templates/`、`scripts/` 等；并含一个 `_meta.json`：
```json
{ "ownerId": "431657", "publishedAt": 1780909163359, "slug": "happyhorse", "version": "1.0.0" }
```

---

## 4. 分类枚举：`GET /api/v1/categories`

**Response**：
```json
{
  "count": 12,
  "items": [
    { "key": "office-efficiency",    "name": "办公效率",     "nameEn": "Office Efficiency",    "level": 1, "sortOrder": 10, "active": true, "children": [] },
    { "key": "content-creation",     "name": "内容创作",     ... },
    { "key": "dev-programming",      "name": "开发编程",     ... },
    { "key": "data-analysis",        "name": "数据分析",     ... },
    { "key": "design-media",         "name": "设计多媒体",   ... },
    { "key": "ai-agent",             "name": "AI Agent",    ... },
    { "key": "knowledge-management", "name": "知识管理",     ... },
    { "key": "business-ops",         "name": "商业运营",     ... },
    { "key": "education",            "name": "教育学习",     ... },
    { "key": "professional",         "name": "行业专业",     ... },
    { "key": "it-ops-security",      "name": "IT 运维与安全", ... },
    { "key": "life-service",         "name": "生活服务",     ... }
  ]
}
```

`children` 数组用于二级分类（当前探测中均为空，但前端字段已定义）。

---

## 5. 技能集（SkillSet / 场景包）

### 5.1 技能集列表：`GET /api/v1/skillsets`

**Query 参数**（来自前端代码）：

| 参数 | 说明 |
|---|---|
| `page` | 页码 |
| `pageSize` | 每页条数 |
| `keyword` | 关键词 |
| `scene` | 场景 key（如 `tech`/`life`/...，枚举见前端 `fL` 常量） |

**Response**：
```json
{
  "total": 32,
  "skillSets": [
    {
      "id": 32,
      "slug": "tech-test-automation",
      "displayName": "自动化测试",
      "displayNameEn": "",
      "summary": "...",
      "summaryEn": "",
      "remark": "",
      "scene": "tech",
      "subScene": "test-automation",
      "scope": 1,
      "iconUrl": "https://...",
      "published": true,
      "skillSlugs": ["superpowers-tdd", "..."],
      "content": "---\nscene: \"tech\"\n...",
      "contentEn": "...",
      "createdAt": ...,
      "updatedAt": ...
    }
  ]
}
```

### 5.2 技能集详情：`GET /api/v1/skillsets/{slug}`

直接返回单个 skillSet 对象（结构同上，不包 `data` 外层）。

---

## 6. 鉴权相关（仅列出可见部分，未做深度验证）

以下接口前端 bundle 中存在但需要登录态（Cookie / `Authorization: Bearer <token>`），本次调研未验证：

| 方法 | 路径 | 用途 |
|---|---|---|
| POST | `/api/v1/auth/tokens` | 手机号/验证码登录换取 token |
| POST | `/api/v1/auth/sms/send` | 发送短信验证码 |
| POST | `/api/v1/auth/sms/verify` | 校验验证码 |
| GET  | `/api/v1/auth/me` | 当前登录用户信息（返回 `{ "user": { id, handle, role, ... } }`） |
| POST | `/api/v1/auth/logout` | 退出 |
| GET  | `/api/v1/auth/github/authorize` | GitHub OAuth 入口 |
| GET  | `/api/v1/auth/github/status` | GitHub 绑定状态 |
| POST | `/api/v1/auth/github/unbind` | 解绑 GitHub |
| GET  | `/api/v1/merchant/info` | 商户信息 |
| GET  | `/api/v1/merchant/developer-keys` | 商户 API Key 列表 |
| POST | `/api/v1/community/skills/publish` | 社区技能发布 |
| POST | `/api/v1/community/skill-icons/upload` | 上传技能图标 |
| POST | `/api/v1/skills/claim/batch` | 批量认领上游技能 |
| GET  | `/api/v1/dashboard/skills` | 个人发布的技能列表（支持 `keyword`/`status`/`page`/`pageSize`） |
| GET  | `/api/v1/orgs/{orgId}/registry/search` | 企业库搜索（参数：`q`、`pageSize`，需 API Key） |
| GET  | `/api/v1/orgs/{orgId}/registry/skills/{slug}/download` | 企业源技能下载 |

CLI 使用的凭证文件：`~/.skillhub/credentials.json`，结构：
```json
{
  "user": { "host": "...", "token": "sh_pat_...", "userId": 123, "handle": "...", "loggedInAt": ... },
  "orgs": {
    "<orgSlug>": { "host": "...", "orgId": "...", "orgSlug": "...", "orgOrgId": "...", "apiKey": "..." }
  }
}
```

---

## 7. CLI 用法速查（与接口对应）

`skillhub` CLI（`~/.skillhub/skills_store_cli.py`，当前版本 2026.6.23）核心命令到 API 的映射：

| CLI 命令 | 调用的 API |
|---|---|
| `skillhub search <q>` | 企业源：`/api/v1/orgs/{orgId}/registry/search?q=...&pageSize=...` + 社区源：`/api/v1/search?q=...&limit=...` 再合并 |
| `skillhub install <slug> [--org community\|orgSlug]` | 1) 下载回退链：`/api/v1/download?slug=...` → COS 直链 → 企业源下载；2) 校验 ed25519 签名；3) 解压到 `--dir` |
| `skillhub login --key <token> --host <host>` | `GET {host}/api/v1/auth/me` 验证 token，写入 `credentials.json` |
| `skillhub self-upgrade` | 拉 `self_update_url` 指向的 `version.json`，对比后下载新的 `skills_store_cli.py` |
| `skillhub publish <dir>` | 企业/商户发布接口（需 token） |

CLI 元数据（`~/.skillhub/metadata.json`）：
```json
{
  "skills_index_url": "https://skillhub-1388575217.cos.ap-guangzhou.myqcloud.com/skills.json",
  "skills_search_url": "https://api.skillhub.cn/api/v1/search",
  "skills_primary_download_url_template": "https://api.skillhub.cn/api/v1/download?slug={slug}",
  "skills_download_url_template": "https://skillhub-1388575217.cos.ap-guangzhou.myqcloud.com/skills/{slug}.zip",
  "self_update_manifest_url": "https://skillhub-1388575217.cos.ap-guangzhou.myqcloud.com/version.json"
}
```

---

## 8. 接口速查表（公开/无需鉴权）

| 方法 | 路径 | 主要用途 | 支持用户过滤 |
|---|---|---|---|
| GET | `/api/skills` | 分页列表/字段过滤 | ❌ 不支持 |
| GET | `/api/v1/search` | 全文搜索 | ❌ 不支持 |
| GET | `/api/v1/showcase/{recommended,hot,newest,trending}` | 首页榜单 | ❌ |
| GET | `/api/v1/categories` | 分类枚举 | – |
| GET | `/api/v1/skillsets` | 技能集列表 | ❌ |
| GET | `/api/v1/skillsets/{slug}` | 技能集详情 | – |
| GET | `/api/v1/skills/{slug}` | 技能详情 | – |
| POST | `/api/v1/skills/batch` | 批量查详情 | – |
| GET | `/api/v1/skills/{slug}/files` | 文件清单/sha256 | – |
| GET | `/api/v1/skills/{slug}/evaluation` | 评测报告 | – |
| GET | `/api/v1/download?slug=...` | 下载（可能需签名，失败回退 COS） | – |
| GET | `/api/v1/users/{handle}` | 用户信息 | ✅（路径参数） |
| **GET** | **`/api/v1/users/{handle}/skills`** | **用户发布的技能** | ✅ **这是按用户查询的官方接口** |

---

## 9. 已知限制 / 踩坑提醒

1. **没有"按用户搜索"的组合接口**：要么在全量搜索里拿到混合结果（无法筛到某用户），要么拿用户全量技能再本地过滤关键词。
2. **`/api/v1/search` 不支持分页**：`limit` 之外没有 `page`/`offset`，结果数也有限。
3. **`/api/skills` 的 `sortBy` 白名单极严**：传错直接 400，不支持 `name`/`newest`/`created_at`/`trending` 等看似合理的值。
4. **`/api/skills` 对未知查询参数静默忽略**：即使传了 `owner=fhbillwer` 也不会报错，会返回全量数据，容易误判为"支持"——必须用返回值里的 `ownerName` 实际校验。
5. **下载接口有回退**：CLI 会依次尝试 `/api/v1/download` 与 COS 直链，其中前者可能对浏览器 UA 返回 405（可能需要签名/特定 header），自己写脚本时建议直接用 COS 模板 URL，或复用 CLI 的回退逻辑。
6. **用户 handle 不一定等于 displayName**：如 `zhenruyan` 的 handle 是 `user_0064faa7`，需要先搜或查主页确认。当前未发现"按 displayName 查 handle"的公开接口，建议通过搜索 API 搜技能拿 `owner_name`/`homepage` 字段再反推 handle。
7. **时间单位是毫秒**，不是秒（Java/JS 风格）。
8. **v1 接口错误用 HTTP 状态码 + `{"error":"..."}`；`/api/skills` 用 200 + `code!=0`**，两种风格混用，客户端需分别处理。
