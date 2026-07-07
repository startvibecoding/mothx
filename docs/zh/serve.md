# Serve 模式

`mothx serve` 是唯一的服务端入口，用来启动统一运行时：

- OpenAI 兼容 `/v1/chat/completions`
- Web UI 管理 API 和静态资源
- 可选的微信、飞书、WebSocket、cron、memory 和 hook 功能

```bash
mothx serve
mothx serve --port 8080 --work-dir /path/to/project
mothx serve init-config project
```

配置统一放在 `serve.json`：

- 全局：`~/.mothx/serve.json`
- 项目：`.mothx/serve.json`
- 自定义路径：`mothx serve --config /path/to/serve.json`

项目配置会覆盖全局配置。使用 `mothx serve init-config global` 或 `mothx serve init-config project` 生成模板。

安全配置由 `serve.json` 中的 Bearer token、`allowedWorkDirs` 和 sandbox 设置共同控制。
