# Serve Mode

`mothx serve` is the only server entry point. It starts the unified runtime for:

- OpenAI-compatible `/v1/chat/completions` API
- Web UI management panel (chat interface, session management, settings editor)
- WeChat, Feishu, WebSocket messaging channels
- Cron scheduled tasks, Memory persistent memory, Hooks
- Stats usage statistics dashboard
- Multi-Agent sub-agent support

```bash
# Start Serve (default 127.0.0.1:7878)
mothx serve

# Specify port and work directory
mothx serve --port 8080 --work-dir /path/to/project

# Bind to external interface (allow access from other machines)
mothx serve --port 0.0.0.0:8080 --work-dir /path/to/project

# Initialize configuration files
mothx serve init-config global   # generates ~/.mothx/serve.json
mothx serve init-config project  # generates .mothx/serve.json
```

## Configuration

Configuration lives in `serve.json`:

- Global: `~/.mothx/serve.json`
- Project: `.mothx/serve.json`
- Custom path: `mothx serve --config /path/to/serve.json`

The project config overlays the global config.

### Core Configuration Fields

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

### Configuration Hot-Reload

After saving settings via the Web UI, Serve automatically hot-reloads provider/model configuration without requiring a restart.

## Network Access

To allow access from other machines, bind Serve to an external interface with `--port 0.0.0.0:8080` or set `"listen": "0.0.0.0:8080"` in `serve.json`. Enable Bearer token auth before exposing Serve beyond loopback.

## Security

Security is controlled by three independent layers:

1. **Bearer Token Auth**: `api.token` or `security.token`
2. **Work Directory Whitelist**: `api.allowedWorkDirs`
3. **Sandbox Isolation**: `sandbox.enabled` (bwrap)

## Web UI

Access `http://127.0.0.1:7878` to open the Web UI, providing:

- **Chat Interface**: SSE streaming output, tool call/result rendering, plan cards
- **Session Management**: Pagination, keyboard shortcuts, historical sessions
- **Settings Editor**: Provider/Model configuration, Defaults, Web Search, Context Files, Compaction, Sandbox, Retry, Approval, Provider Config
- **Channel Management**: WeChat QR login, Feishu config, WebSocket toggle
- **Serve Config**: Features, API, Cron, Memory, Security, Agent, Hooks, Channels, Lobster Mode

## Messaging Channels

### WeChat

- QR code login support (scan-to-authenticate)
- Login status polling with error handling
- API endpoint: `/api/channels/wechat/login`

### Feishu

- Supports appId/appSecret/workspace/allowedUsers configuration
- Automatic message routing and session persistence

### WebSocket

- Mounted at `/ws` endpoint
- Reuses Channels event protocol for real-time communication

## Stats Dashboard

Access `http://127.0.0.1:7878` to view usage statistics (tokens, requests, duration), filterable by time range, provider, and model.

## CLI Flags

| Flag | Description | Default |
|------|-------------|--------|
| `--port` | Listen address (host:port) | `127.0.0.1:7878` |
| `--work-dir` | Working directory | Current directory |
| `--config` | Configuration file path | `~/.mothx/serve.json` or `.mothx/serve.json` |
| `--web-ui-dir` | Web UI static assets directory | Built-in path |
| `--debug` | Enable pprof profiling server | Disabled |
