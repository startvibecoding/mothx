# Serve Mode

`mothx serve` is the only server entry point. It starts the unified runtime for:

- OpenAI-compatible `/v1/chat/completions`
- Web UI management APIs and static assets
- Optional WeChat, Feishu, WebSocket, cron, memory, and hook features

```bash
mothx serve
mothx serve --port 8080 --work-dir /path/to/project
mothx serve --port 0.0.0.0:8080 --work-dir /path/to/project
mothx serve init-config project
```

Configuration lives in `serve.json`:

- Global: `~/.mothx/serve.json`
- Project: `.mothx/serve.json`
- Custom path: `mothx serve --config /path/to/serve.json`

The project config overlays the global config. Use `mothx serve init-config global` or `mothx serve init-config project` to create a template.

To allow access from other machines, bind Serve to an external interface with `--port 0.0.0.0:8080` or set `"listen": "0.0.0.0:8080"` in `serve.json`. Enable Bearer token auth before exposing Serve beyond loopback.

Security is controlled by Bearer tokens, `allowedWorkDirs`, and sandbox settings in `serve.json`.
