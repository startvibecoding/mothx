# SkillHub 扩展市场与企业 Registry 协议定义

> 状态：Defined
> 日期：2026-07-16
> 范围：定义后续 adapter 的配置、HTTP、认证与安全契约；不在本阶段实现 adapter，也不改变现有 `settings.json` schema。

## 1. 目标和边界

本规范为第三方公共市场和企业私有 registry 定义统一接入契约。新市场必须显式选择协议，客户端不得从 URL、响应字段或错误内容猜测其协议或能力。

- 只定义浏览、搜索、详情、文件、评测和下载。
- 不定义登录、发布、用户管理、token 签发或刷新。
- 所有下载仍交给现有安全安装器：大小限制、拒绝路径穿越/绝对路径/Windows drive/symlink、`SKILL.md` 校验和安全覆盖策略。
- 企业属性不改变安装目录、`allowedWorkDirs`、sandbox 或 session 激活规则。

## 2. 市场标识、协议和能力

市场 ID 必须稳定、全小写，推荐 DNS 风格（例如 `acme.skills`），且不得与 `skillhub.cn`、`clawhub.ai` 冲突。ID 是本地 `.mothx-skillhub.json` 中的来源标识；站点改名或 API URL 迁移不得改变 ID。

支持的 `protocol`：

| protocol | 含义 |
|---|---|
| `skillhub-v1` | 现有 SkillHub.cn HTTP 语义 |
| `clawhub-v1` | 现有 ClawHub HTTP 语义 |
| `mothx-market-v1` | 本规范定义的通用协议 |

内置 ID 省略 `protocol` 时沿用既有 adapter；任何非内置 ID 必须显式设置 `protocol: "mothx-market-v1"`。未知协议必须显示为未支持，且不能出现在可用市场 tab 中。

`mothx-market-v1` 通过 `GET /v1/market` 协商能力。未知 major `schemaVersion` 必须拒绝接入；客户端不能根据缺失端点推断 capability。

```json
{
  "schemaVersion": "mothx-market/v1",
  "id": "acme.skills",
  "name": "Acme Skills",
  "capabilities": {
    "search": true,
    "list": true,
    "cursorPagination": true,
    "detail": true,
    "download": true,
    "fileList": true,
    "fileContent": false,
    "categories": true,
    "evaluation": false
  }
}
```

能力名称与 `internal/skillhub.MarketCapabilities` 对应。`detail` 和 `download` 是可安装市场的必要能力；缺少其中任一项时市场只能浏览或必须被禁用。分页只能选择 cursor 或 page 之一，不能混用。

## 3. 配置定义

未来 schema 保持现有 `apiToken` 的 Bearer 含义兼容，并扩展为以下模型：

```json
{
  "id": "acme.skills",
  "name": "Acme Skills",
  "siteURL": "https://skills.acme.example",
  "apiURL": "https://skills.acme.example",
  "protocol": "mothx-market-v1",
  "registry": "enterprise",
  "enabled": true,
  "auth": {
    "type": "header",
    "headerName": "X-API-Key",
    "token": "${ACME_SKILLS_API_KEY}"
  }
}
```

- `registry` 只允许 `public` 或 `enterprise`；它影响 UI 与认证政策，不影响安装权限。
- `auth.type` 只允许 `none`、`bearer`、`header`。
- `header` 必须提供符合 HTTP token 语法、且不是 `Authorization` 的 `headerName`。
- `auth.token` 支持现有受控 secret / 环境变量引用；不得写入 API 响应、TUI、Web UI、日志、错误或诊断。
- `apiToken` 与 `auth` 同时出现为配置错误。
- `apiURL` 必须为绝对 HTTPS URL。只有 `localhost`、`127.0.0.1`、`::1` 可用 HTTP。
- `siteURL` 仅用于展示和跳转，不能用作 API 或下载回退地址。

## 4. `mothx-market-v1` HTTP 契约

所有路径相对 `apiURL`，采用 UTF-8 JSON；下载端点返回 zip 二进制或同源重定向。客户端发送 `Accept: application/json` 和 `User-Agent: MothX/<version>`。

| 方法 | 路径 | capability | 语义 |
|---|---|---|---|
| GET | `/v1/market` | — | 协议版本、市场信息和能力协商 |
| GET | `/v1/skills` | `list` 或 `search` | 列表/搜索 |
| GET | `/v1/skills/{id}` | `detail` | Skill 详情 |
| GET | `/v1/skills/{id}/files?version=` | `fileList` | 文件清单 |
| GET | `/v1/skills/{id}/files/content?path=&version=` | `fileContent` | 单一文本文件 |
| GET | `/v1/skills/{id}/evaluation` | `evaluation` | 评测数据 |
| GET | `/v1/categories` | `categories` | 分类树 |
| GET | `/v1/skills/{id}/download?version=` | `download` | zip 下载 |

`id` 必须整体 URL path-escape，服务端只 decode 一次。`GET /v1/skills` 可接受 `q`、`limit`（1–100，默认 20）、`cursor`、`page`、`sort`、`order`、`category`、`author`、`verifiedOnly` 和 `nonSuspiciousOnly`。服务端必须在 capability 中表明分页模式与所支持的筛选项；不支持的筛选项不得静默产生看似成功但错误的过滤结果。

列表响应必须映射到现有 `SearchPage`：`items`、`total`、`page`、`pageSize`、`nextCursor`。技能列表/详情必须可映射至现有 `SkillSummary` / `SkillDetail`，至少提供 `id`、`name` 或 `displayName`、`description`、`version`、`author`、`downloads` 与 `updatedAt`。`files`、`securityReports`、`evaluation`、`downloadSources` 可选。

下载响应必须带 `Content-Type: application/zip` 或 `application/octet-stream`。详情或文件清单提供 `sha256` 时，客户端安装前必须校验。

错误必须使用非 2xx 与稳定机器码：

```json
{ "error": { "code": "skill_not_found", "message": "skill not found" } }
```

保留错误码：`unauthenticated`、`forbidden`、`skill_not_found`、`invalid_cursor`、`rate_limited`、`unsupported_capability`、`invalid_request`、`internal_error`。401 对应 `unauthenticated`，403 对应 `forbidden`，404 对应 `skill_not_found`；429 应返回 `Retry-After`。

## 5. 企业 registry 与 token 语义

企业 registry 只支持只读查询和下载。凭证由运行 MothX 的操作者在本地配置；MothX 不保存、刷新、交换或上传凭证。

- 企业 registry 必须设置 `registry: "enterprise"`、显式 `protocol` 和非 `none` 的 `auth`；公共 registry 可以不认证。
- `bearer` 固定发送 `Authorization: Bearer <token>`；`header` 仅向 `apiURL` 同源请求发送指定头。
- 不支持 Cookie 注入、查询串 token、URL 嵌入凭证或任意自定义认证值。
- 推荐最小 token scope：`skills:read` 与 `skills:download`。token 缺失、无效或过期返回 401；权限不足返回 403。
- 认证头只可发往 `apiURL` 同源请求。下载重定向跨源时必须移除认证头；认证下载必须使用同源端点或同源签名 URL。
- 错误仅可展示 market ID 和可操作摘要；不得输出 token、认证头、完整服务端 body 或签名下载 URL。
- `.mothx-skillhub.json` 只记录 market、id、版本和不含凭证的 `sourceURL`；不得保存 token、认证头或签名 URL。

## 6. 未来实现验收

新增 adapter 必须通过本地 HTTP fixture 验证：协议协商、未知协议/能力拒绝、列表/详情/文件/下载映射、401/403/429、Bearer 与 header 注入、跨源重定向不泄漏认证头、token 不进入日志或 metadata、企业安装仍受 `allowedWorkDirs` 与 zip 安全校验限制。
