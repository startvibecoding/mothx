# Serve Mode

`mothx serve` is the only server entry point. It starts the unified runtime for:

- OpenAI-compatible `/v1/chat/completions`
- Web UI management APIs and static assets
- Optional WeChat, Feishu, WebSocket, cron, memory, and hook features

```bash
mothx serve
mothx serve --port 8080 --work-dir /path/to/project
mothx serve init-config project
```

Configuration lives in `serve.json`:

- Global: `~/.mothx/serve.json`
- Project: `.mothx/serve.json`
- Custom path: `mothx serve --config /path/to/serve.json`

The project config overlays the global config. Use `mothx serve init-config global` or `mothx serve init-config project` to create a template.

Security is controlled by Bearer tokens, `allowedWorkDirs`, and sandbox settings in `serve.json`.
