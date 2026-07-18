# Serve 模式

`mothx serve` 是唯一的服务端入口，用来启动统一运行时：

- OpenAI 兼容 `/v1/chat/completions` API
- Web UI 管理面板（聊天界面、会话管理、设置编辑）
- 微信、飞书、WebSocket 消息通道
- Cron 定时任务、Memory 持久记忆、Hooks 钩子
- Stats 使用统计仪表盘
- 多 Agent 子代理支持

```bash
# 启动 Serve（默认 127.0.0.1:7878）
mothx serve

# 指定端口和工作目录
mothx serve --port 8080 --work-dir /path/to/project

# 绑定到外部网卡（允许其他机器访问）
mothx serve --port 0.0.0.0:8080 --work-dir /path/to/project

# 关闭认证并绑定到所有网卡（仅限可信网络）
mothx serve --unsafe --work-dir /path/to/project

# 初始化配置文件
mothx serve init-config global   # 生成 ~/.mothx/serve.json
mothx serve init-config project  # 生成 .mothx/serve.json
```

## 配置

配置统一放在 `serve.json`：

- 全局：`~/.mothx/serve.json`
- 项目：`.mothx/serve.json`
- 自定义路径：`mothx serve --config /path/to/serve.json`

项目配置会覆盖全局配置。

### 核心配置项

```json
{
  "api": {
    "listen": "127.0.0.1:7878",
    "token": "your-secret-token",
    "allowedWorkDirs": ["/path/to/project"]
  },
  "features": {
    "multiAgent": true,
    "workflows": true
  },
  "sandbox": {
    "enabled": false
  },
  "channels": {
    "wechat": { "enabled": false },
    "feishu": { "enabled": false },
    "websocket": { "enabled": false }
  },
  "webUI": {
    "enabled": true
  },
  "cron": {
    "enabled": true
  },
  "memory": {
    "path": ".mothx/memory.md"
  },
  "security": {
    "token": "your-secret-token"
  },
  "agent": {
    "mode": "yolo"
  }
}
```

### 配置热重载

通过 Web UI 保存设置后，Serve 会自动热重载 provider/model 配置，无需重启服务。

## 网络访问

如果需要允许其他机器访问，把 Serve 绑定到外部网卡，例如 `--port 0.0.0.0:8080`，或在 `serve.json` 中设置 `"listen": "0.0.0.0:8080"`。对外暴露前应开启 Bearer token 认证。仅在可信本地网络中，可以使用 `--unsafe` 临时关闭认证，并把 loopback/default 监听地址绑定到 `0.0.0.0`。

## 安全

安全配置由以下三层独立控制：

1. **Bearer Token 认证**：`api.token` 或 `security.token`
2. **工作目录白名单**：`api.allowedWorkDirs`
3. **沙箱隔离**：`sandbox.enabled`（bwrap）

## Web UI

访问 `http://127.0.0.1:7878` 打开 Web UI，提供：

- **聊天界面**：SSE 流式输出，工具调用/结果渲染，计划卡片，以及 `plan`、`agent`、`yolo` 模式的会话运行时菜单
- **审批中心**：查看待处理工具审批，一次性批准或拒绝，持久化命令/路径放行规则，并查看会话审批审计历史
- **会话管理**：分页浏览，键盘快捷键，历史会话，运行时快照，能力开关及可安全重连恢复的审批状态
- **设置编辑**：Provider/Model 配置，Defaults，Web 搜索，上下文文件，压缩，沙箱，重试，审批，Provider 配置
- **通道管理**：微信 QR 登录，飞书配置，WebSocket 开关
- **服务配置**：Features，API，Cron，Memory，Security，Agent，Hooks，Channels，Lobster 模式

## 消息通道

### 微信

- 支持 QR 码登录（扫码认证）
- 登录状态轮询，错误处理
- API 端点：`/api/channels/wechat/login`

### 飞书

- 支持 appId/appSecret/workspace/allowedUsers 配置
- 自动消息路由和会话持久化

### WebSocket

- 挂载到 `/ws` 端点
- 复用 Channels 事件协议实现实时通信

## Stats 仪表盘

访问 `http://127.0.0.1:7878` 可查看使用统计（tokens、请求数、持续时间），支持按时间范围、provider、model 筛选。

## CLI 标志

| 标志 | 描述 | 默认值 |
|------|------|--------|
| `--port` | 监听地址（host:port） | `127.0.0.1:7878` |
| `--work-dir` | 工作目录 | 当前目录 |
| `--config` | 配置文件路径 | `~/.mothx/serve.json` 或 `.mothx/serve.json` |
| `--web-ui-dir` | Web UI 静态资源目录 | 内置路径 |
| `--unsafe` | 关闭认证并绑定到所有网卡 | 关闭 |
| `--debug` | 启用 pprof 性能分析服务器 | 关闭 |
