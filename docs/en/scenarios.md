# Scenarios & Walkthroughs

This document demonstrates MothX's various usage modes through practical scenarios.

---

## Scenario 1: Daily Coding (Single Agent Mode)

The most basic usage — conversing with AI in the terminal for everyday development tasks.

### Setup

```bash
# Set API key
export DEEPSEEK_API_KEY=sk-...

# Enter project directory
cd ~/projects/myapp
```

### A: Code Explanation

```bash
$ mothx -P "explain the core logic of internal/agent/agent.go"
```

MothX will:
1. Load context files (`AGENTS.md`, `CLAUDE.md`)
2. Read the file with `read`
3. Explore related code with `ls`, `grep`
4. Output a structured explanation

### B: Bug Fix

```bash
$ mothx
```

```
> fix the null pointer error in handler.go

[Agent]
  📄 read("internal/handler.go", offset=45, limit=20)
  → Found: line 52 doesn't check if err is nil

  📝 edit("internal/handler.go", edits=[
    {oldText: "resp, _ := client.Do(req)",
     newText: "resp, err := client.Do(req)\n\tif err != nil {\n\t\treturn nil, err\n\t}"}
  ])
  → Edit successful

  🔧 bash("go build ./...")
  → Build passed

✅ Fixed: added error check at line 52
```

### C: Adding a New Feature

```
> add a /health endpoint to the HTTP handler

[Agent]
  📄 plan(title="Add /health endpoint", steps=[
    {title: "Read existing handler structure", status: "running"},
    {title: "Add health handler", status: "pending"},
    {title: "Register route", status: "pending"},
    {title: "Add tests", status: "pending"},
    {title: "Verify build", status: "pending"}
  ])

  📄 read("internal/server/routes.go")
  📝 edit(...)  # Add handler
  📝 edit(...)  # Register route
  🔧 bash("go test ./internal/server/ -run TestHealth")
  🔧 bash("go build ./...")

  📄 plan(..., steps=[..., {title: "Verify build", status: "done"}])

✅ Done: /health endpoint added and tested
```

### Mode Selection Guide

| Task | Recommended Mode | Command |
|------|-----------------|---------|
| Read code, learn project | `plan` | `mothx --mode plan` |
| Write code, fix bugs | `agent` (default) | `mothx` |
| Install deps, system ops | `yolo` | `mothx --mode yolo` |

---

## Scenario 2: Non-Interactive Mode (CI/Script Integration)

Use MothX in CI pipelines or scripts.

### A: Code Review

```bash
# Review PR in CI
git diff main..feature | mothx -P "review this diff, point out potential issues"
```

### B: Automated Refactoring

```bash
# Batch refactoring
mothx -P "change all fmt.Errorf calls to use %w for error wrapping" --mode yolo
```

### C: Generate Documentation

```bash
# Generate README for a package
mothx -P "generate README.md for internal/cache package with usage examples" --mode yolo
```

---

## Scenario 3: Multi-Agent Mode (Complex Task Delegation)

Enable sub-agent tools with `--multi-agent` to split and execute complex tasks in parallel.

### Launch

```bash
$ mothx --multi-agent
```

### Scenario: Parallel Refactoring and Testing

```
> I need: 1) rename internal/cache to internal/store
>        2) ensure all tests pass at the same time

[Agent]
  🤖 subagent_spawn(task="Rename internal/cache to internal/store, update all import paths",
                     mode="agent",
                     tools=["read", "write", "edit", "bash", "grep", "find"])

  → Handle: "agent-1"

  🤖 subagent_spawn(task="Run full test suite, report failures",
                     mode="agent",
                     tools=["read", "bash", "grep", "find"])

  → Handle: "agent-2"

  ... wait for sub-agents ...

  🤖 subagent_status(handle="agent-1")
  → Status: completed
  → Result: "Renamed cache to store, updated 15 files' import paths"

  🤖 subagent_status(handle="agent-2")
  → Status: completed
  → Result: "3 tests failed: TestCacheGet, TestCacheSet, TestCacheDelete"

  🤖 subagent_send(handle="agent-1", message="Fix the 3 failing tests reported by agent-2")

  ... sub-agent continues ...

✅ Done: package renamed, all tests pass
```

### Sub-Agent Tools Summary

| Tool | Purpose |
|------|---------|
| `subagent_spawn` | Start sub-agent, returns handle |
| `subagent_status` | Query sub-agent status and results |
| `subagent_send` | Send follow-up instructions |
| `subagent_destroy` | Stop and clean up sub-agent |

### Multi-Agent + Cron Scheduling

```bash
# Daily code review
mothx hermes cron add "daily-review" \
  "review the last 24 hours of git changes, output an issue report" \
  --schedule "@daily"
```

---

## Scenario 4: VS Code ACP Integration

Use MothX directly in VS Code as an AI coding assistant.

### Step 1: Install

```bash
npm install -g mothx
```

### Step 2: Configure VS Code

Edit VS Code's `settings.json`:

```json
{
  "acp.agents": {
    "mothx": {
      "command": "mothx",
      "args": ["acp", "--mode", "agent", "--multi-agent"],
      "description": "MothX AI Assistant"
    }
  }
}
```

### Step 3: Use

1. Open your project in VS Code
2. Open the ACP panel (via extension)
3. Ask questions or request code changes directly

**Experience in VS Code:**

```
You: change ParseConfig in utils.go to support YAML format

MothX:
  [tool_call: read utils.go]
  [tool_call: edit utils.go]
  [tool_call: bash "go test ./..."]
  ✅ YAML support added, all tests pass
```

### ACP Mode Special Capabilities

| Capability | Description |
|------------|-------------|
| Session Management | IDE auto-manages session create/load/continue |
| Permission Requests | IDE popup for high-risk operations |
| MCP Integration | IDE can pass MCP server configs |
| Multi-Agent | Enable sub-agent tools via `--multi-agent` |

---

## Scenario 5: A2A Standalone Server Mode

Run MothX as an A2A server for other agents to call.

### A: Start Standalone A2A Server

```bash
# Initialize config
mothx a2a --init-a2a-config

# Edit a2a.json (optional)
vim ~/.vibecoding/a2a.json

# Start server
mothx a2a start --port 8093 --work-dir ~/projects/myapp
```

### B: Other Agents Call It

```bash
# Using mothx client
mothx a2a send "list all Go files in the project" --target http://localhost:8093

# Using curl
curl -X POST http://localhost:8093/a2a \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {
        "role": "user",
        "parts": [{"type": "text", "text": "run all tests"}]
      }
    },
    "id": 1
  }'

# Discover remote agent capabilities
mothx a2a discover http://localhost:8093
```

### C: A2A Server with Authentication

```bash
# Start with auth token
mothx a2a start --auth-token "my-secret-token-xxx"

# Client call with token
mothx a2a send "review main.go" \
  --target http://remote-server:8093 \
  --auth-token "my-secret-token-xxx"
```

---

## Scenario 6: A2A Master Mode (Cross-Machine Agent Dispatch)

Manage multiple remote A2A agents, letting the LLM automatically dispatch tasks.

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Local (MothX + A2A Master)                         │
│                                                         │
│  mothx --enable-a2a-master                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │  LLM auto-decides → a2a_dispatch tool            │   │
│  └─────────────────────────────────────────────────┘   │
│           │                   │                         │
│           ▼                   ▼                         │
│  ┌──────────────┐   ┌──────────────┐                   │
│  │ code-reviewer│   │  ci-agent    │                   │
│  │ 192.168.1.10 │   │ 192.168.1.20 │                   │
│  │ :8093        │   │ :8093        │                   │
│  └──────────────┘   └──────────────┘                   │
└─────────────────────────────────────────────────────────┘
```

### Step 1: Start A2A Servers on Remote Machines

**Machine A (Code Review Agent):**
```bash
# 192.168.1.10
mothx a2a start --port 8093 --work-dir ~/projects/shared
```

**Machine B (CI Agent):**
```bash
# 192.168.1.20
mothx a2a start --port 8093 --work-dir ~/ci-runner --auth-token "ci-secret"
```

### Step 2: Initialize Master Config Locally

```bash
# Generate sample config
mothx --init-a2a-master-config

# Edit a2a-list.json
vim ~/.vibecoding/a2a-list.json
```

```json
{
  "agents": [
    {
      "name": "code-reviewer",
      "url": "http://192.168.1.10:8093"
    },
    {
      "name": "ci-agent",
      "url": "http://192.168.1.20:8093",
      "auth_token": "ci-secret"
    }
  ]
}
```

### Step 3: Enable Master Mode

```bash
$ mothx --enable-a2a-master --verbose
```

```
A2A master mode enabled: 2 agents loaded from /home/user/.vibecoding/a2a-list.json

> review internal/handler.go for code quality, then run tests to make sure nothing breaks

[Agent]
  I'll dispatch tasks to both remote agents:

  🔧 a2a_dispatch(agent_name="code-reviewer",
                   message="Review internal/handler.go for code quality, focus on
                           error handling, performance, and security")

  → code-reviewer returns: "Found 3 issues: 1) Line 45 doesn't handle timeout..."

  🔧 a2a_dispatch(agent_name="ci-agent",
                   message="Run the full test suite, report results")

  → ci-agent returns: "47/47 tests passed, coverage 82%"

✅ Summary:
- Code review found 3 issues (details listed)
- All tests pass, coverage 82%
- Recommend fixing timeout handling on line 45 first
```

---

## Scenario 7: Gateway Mode (HTTP API)

Run MothX as an OpenAI-compatible HTTP service for other applications to call.

### Initialize and Start

```bash
# Generate config template
mothx --init-gateway

# Edit gateway.json (set token, port, etc.)
vim ~/.vibecoding/gateway.json

# Start gateway
mothx gateway --port 8080 --work-dir ~/projects/myapp
```

### Call It

```bash
# curl (OpenAI-compatible format)
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-flash",
    "messages": [
      {"role": "user", "content": "explain this project architecture"}
    ]
  }'

# Python OpenAI SDK
from openai import OpenAI
client = OpenAI(base_url="http://localhost:8080/v1", api_key="your-token")
response = client.chat.completions.create(
    model="deepseek-v4-flash",
    messages=[{"role": "user", "content": "write an HTTP middleware"}]
)
```

---

## Scenario 8: Hermes Messaging Gateway

Connect MothX to WeChat/Feishu for unattended AI coding assistant.

### Start

```bash
# Configure hermes.json
vim ~/.vibecoding/hermes.json

# Start
mothx hermes start
```

### Typical Config

```json
{
  "server": { "port": 8090, "auth_token": "my-token" },
  "platforms": {
    "wechat": { "enabled": true },
    "feishu": { "enabled": true, "app_id": "...", "app_secret": "..." }
  },
  "default_mode": "yolo",
  "security": {
    "smart_approvals": true,
    "allowed_work_dirs": ["/srv/projects"]
  },
  "a2a": { "enabled": true },
  "cron": { "enabled": true },
  "memory": { "enabled": true }
}
```

### Usage in Messaging Platform

```
User: /new
Bot:  New session created

User: add rate limiting middleware to the api package
Bot:  [executing...]
      ✅ Added rate limiting middleware with configurable requests/sec
      Modified: internal/api/middleware.go, internal/api/routes.go

User: run tests
Bot:  [running go test ./...]
      ✅ All passed (12/12)
```

---

## Scenario 9: Combined Modes (Multi-Tool Workflow)

Combine multiple modes for a complete development workflow.

### Example: Develop + Review + Deploy

```bash
# 1. Local development (TUI mode)
cd ~/projects/myapp
mothx --mode yolo

# 2. Pre-commit review (Plan mode)
mothx --mode plan "review all changes in git diff"

# 3. Post-push CI review (Gateway mode)
# In CI script:
curl http://gateway:8080/v1/chat/completions \
  -d '{"messages": [{"role": "user", "content": "review PR #42"}]}'

# 4. Scheduled security scan (Hermes + Cron)
mothx hermes cron add "security-scan" \
  "scan for security vulnerabilities and sensitive data leaks" \
  --schedule "@weekly"
```

---

## Quick Reference

| Scenario | Command |
|----------|---------|
| Daily coding | `mothx` |
| Read-only analysis | `mothx --mode plan` |
| Full access | `mothx --mode yolo` |
| Non-interactive | `mothx -P "..."` |
| Multi-agent | `mothx --multi-agent` |
| A2A server | `mothx a2a start` |
| A2A master | `mothx --enable-a2a-master` |
| HTTP gateway | `mothx gateway` |
| Messaging gateway | `mothx hermes start` |
| IDE integration | `mothx acp` |
| Continue session | `mothx -c` |
| Resume session | `mothx -r <id>` |
| Init gateway config | `mothx --init-gateway` |
| Init A2A config | `mothx a2a --init-a2a-config` |
| Init master config | `mothx --init-a2a-master-config` |
