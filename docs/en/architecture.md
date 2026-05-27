# System Architecture

## Project Structure

```
vibecoding/
├── agent/                       # Public Agent/Provider interfaces and Builder
├── cmd/vibecoding/              # CLI entry point
│   └── main.go                  # Main program
├── internal/
│   ├── agent/                   # Core Agent loop
│   │   ├── agent.go             # Agent main logic
│   │   ├── factory.go           # AgentFactory for per-agent construction
│   │   ├── manager.go           # AgentManager lifecycle management
│   │   ├── router.go            # EventRouter
│   │   ├── subagent.go          # subagent_* tools
│   │   ├── events.go            # Event type definitions
│   │   ├── provider.go          # Provider interface adapter
│   │   └── system_prompt.go     # System prompt generation
│   ├── config/                  # Configuration management
│   ├── context/                 # Context management and token estimation
│   ├── contextfiles/            # Context file loading
│   ├── platform/                # Cross-platform compatibility utilities
│   ├── provider/                # LLM Provider abstraction
│   │   ├── anthropic/           # Anthropic Messages API
│   │   ├── factory/             # Shared provider/model construction
│   │   ├── vendor*.go           # Vendor adapter registry and defaults
│   │   └── openai/              # OpenAI Chat Completions API
│   ├── cron/                    # Scheduled task store and scheduler
│   ├── sandbox/                 # Sandbox abstraction (bwrap, none)
│   ├── session/                 # Session management (JSONL)
│   ├── skills/                  # Skills system
│   ├── tools/                   # Tool implementations
│   │   ├── bash.go              # Bash command execution
│   │   ├── read.go              # File reading
│   │   ├── write.go             # File writing
│   │   ├── edit.go              # File editing
│   │   ├── grep.go              # Content search
│   │   ├── find.go              # File finding
│   │   └── ls.go                # Directory listing
│   ├── tui/                     # Terminal UI (BubbleTea)
│   └── ua/                      # User-Agent string generation
```

## Core Components

### 1. Provider System

Provider is an abstraction layer for interacting with LLM APIs. CLI and ACP
provider creation both go through `internal/provider/factory`, which applies
vendor adapter defaults before constructing the generic OpenAI-compatible or
Anthropic-compatible protocol provider.

```
┌─────────────────────────────────────────────────────────────┐
│                      Provider Interface                      │
├─────────────────────────────────────────────────────────────┤
│  Chat(ctx, params) <-chan StreamEvent                       │
│  Models() []*Model                                          │
│  GetModel(id string) *Model                                 │
│  Name() string                                              │
└─────────────────────────────────────────────────────────────┘
                              │
                              │
            ┌─────────────────┴─────────────────┐
            ▼                                   ▼
    ┌───────────────────┐             ┌───────────────────┐
    │ Vendor Adapters   │             │ Generic Fallback  │
    │ vendor_*.go       │             │ openai/anthropic  │
    └───────────────────┘             └───────────────────┘
```

Vendor resolution order:

1. Explicit `vendor` field in provider config
2. Base URL detection
3. Generic fallback based on `api`

#### StreamEvent Types

```go
type StreamEvent struct {
    Type        EventType
    TextDelta   string      // Text delta
    ThinkingDelta string    // Thinking delta
    ToolCall    *ToolCall   // Tool call
    Usage       *Usage      // Token usage
    Error       error       // Error
}
```

### 2. Agent Loop

Agent is the core logic that coordinates Provider, Tools, and Session.

```
┌─────────────────────────────────────────────────────────────┐
│                       Agent Loop                             │
├─────────────────────────────────────────────────────────────┤
│  1. Build system prompt (mode + tools + context files + skills)
│  2. Send messages to Provider                               │
│  3. Process stream events (text, thinking, tool calls)      │
│  4. Execute tools and collect results                       │
│  5. Add tool results to messages                            │
│  6. Repeat until complete                                   │
└─────────────────────────────────────────────────────────────┘
```

#### Execution Flow

```
User Input
    │
    ▼
┌───────────────┐
│ Build Context │ ← System Prompt + Tools + Context Files + Skills
└───────┬───────┘
        │
        ▼
┌───────────────┐
│  Provider     │ ← LLM API (OpenAI/Anthropic)
│  Chat()       │
└───────┬───────┘
        │
        ▼
┌───────────────┐     ┌───────────────┐
│ Stream Events │────▶│ Tool Calls?   │
└───────┬───────┘     └───────┬───────┘
        │                     │
        │                     ▼
        │              ┌───────────────┐
        │              │ Execute Tools │
        │              └───────┬───────┘
        │                     │
        │                     ▼
        │              ┌───────────────┐
        └──────────────│ Append Results│
                       └───────────────┘
```

### 3. Multi-Agent Runtime

Multi-agent mode is opt-in with `--multi-agent`. When enabled, the main agent
gets the `subagent_spawn`, `subagent_status`, `subagent_send`, and
`subagent_destroy` tools. Child agents have isolated messages, context, session,
registry, and job manager state.

```
Main Agent
    │
    ├── AgentManager creates child agents
    ├── EventRouter routes events by AgentID
    └── subagent_* tools manage async child work
```

Child agents cannot create nested sub-agents because their registries filter out
the `subagent_*` tools.

### 4. Cron Scheduler

The `internal/cron` package provides a file-backed cron store and scheduler that
can execute jobs through sub-agents. The TUI exposes `/cron` command entry
points in multi-agent mode; full natural-language parsing and persistent TUI
management remain follow-up wiring.

### 5. Tool System

Tools are the way Agent interacts with the external world.

```
┌─────────────────────────────────────────────────────────────┐
│                    Tool Interface                            │
├─────────────────────────────────────────────────────────────┤
│  Name() string                                              │
│  Description() string                                       │
│  Parameters() json.RawMessage                               │
│  Execute(ctx, params) (*ToolResult, error)                  │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│  File Tools   │   │  Search Tools │   │  System Tools │
│  - read       │   │  - grep       │   │  - bash       │
│  - write      │   │  - find       │   │  - ls         │
│  - edit       │   │               │   │               │
└───────────────┘   └───────────────┘   └───────────────┘
```

### 6. Session Management

Sessions use JSONL format with tree structure and branching support.

```
┌─────────────────────────────────────────────────────────────┐
│                    Session Structure                         │
├─────────────────────────────────────────────────────────────┤
│  {                                                          │
│    "id": "session-abc123",                                  │
│    "type": "session",                                       │
│    "timestamp": "2024-01-01T00:00:00Z",                     │
│    "cwd": "/home/user/project",                             │
│    "provider": "anthropic",                                 │
│    "model": "claude-sonnet-4-20250514"                      │
│  }                                                          │
│  {                                                          │
│    "id": "msg-001",                                         │
│    "parentId": "session-abc123",                            │
│    "type": "message",                                       │
│    "role": "user",                                          │
│    "content": "..."                                         │
│  }                                                          │
│  {                                                          │
│    "id": "msg-002",                                         │
│    "parentId": "msg-001",                                   │
│    "type": "message",                                       │
│    "role": "assistant",                                     │
│    "content": "..."                                         │
│  }                                                          │
└─────────────────────────────────────────────────────────────┘
```

#### Session Types

| type | Description |
|------|-------------|
| `session` | Session metadata |
| `message` | User/assistant messages |
| `model_change` | Model change record |
| `compaction` | Context compression record |
| `label` | Session label |

### 7. Sandbox System

Sandbox implements process isolation through bubblewrap (bwrap).

```
┌─────────────────────────────────────────────────────────────┐
│                     Sandbox Manager                          │
├─────────────────────────────────────────────────────────────┤
│  SetLevel(level)                                            │
│  GetActive() *Sandbox                                       │
│  Execute(cmd) (stdout, stderr, error)                       │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│  LevelNone    │   │ LevelStandard │   │  LevelStrict  │
│  (Unrestricted)│  │ (Project R/W) │   │ (Project R/O) │
└───────────────┘   └───────────────┘   └───────────────┘
```

### 8. TUI System

Terminal user interface based on BubbleTea.

```
┌─────────────────────────────────────────────────────────────┐
│                        TUI App                              │
├─────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   Header Bar                          │  │
│  │  Provider: anthropic │ Model: claude-sonnet-4 │ Mode  │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   Message Area                        │  │
│  │  User: ...                                            │  │
│  │  Assistant: ...                                        │  │
│  │  [tool: bash] running...                              │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   Input Area                          │  │
│  │  > _                                                  │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   Status Bar                          │  │
│  │  Thinking: medium │ Tokens: 1234 in / 567 out │ Cost  │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### Complete Request Flow

```
1. User Input
   │
   ▼
2. TUI captures input
   │
   ▼
3. Agent.Run(ctx, input)
   │
   ▼
4. Build system prompt
   ├── Mode prompt (plan/agent/yolo)
   ├── Tool definitions (JSON Schema)
   ├── Context files (AGENTS.md, CLAUDE.md)
   └── Skills context
   │
   ▼
5. Build message history
   ├── Historical messages (from Session)
   └── New user message
   │
   ▼
6. Provider.Chat(ctx, params)
   │
   ▼
7. SSE streaming response
   ├── TextDelta → Display text
   ├── ThinkingDelta → Display thinking
   └── ToolCall → Execute tool
   │
   ▼
8. Tool execution (via Sandbox)
   │
   ▼
9. Collect tool results
   │
   ▼
10. Continue conversation (back to step 5)
   │
   ▼
11. Complete, save session
```

## Key Design Decisions

### 1. Interface Abstraction

Use interface abstraction for Provider and Tool, facilitating extension and testing.

### 2. Stream Processing

Use Channel for streaming responses, providing real-time feedback.

### 3. Session Tree

Use tree structure for session storage, supporting branching and restoration.

### 4. Layered Configuration

Support global and project configuration, with project configuration overriding global.

### 5. Sandbox Isolation

Implement process-level isolation through bubblewrap, protecting system security.
