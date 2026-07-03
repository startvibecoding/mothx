# Development Guide

This document describes how to contribute code to MothX.

## Development Environment Setup

### System Requirements

- Go 1.24+
- Git
- Make (optional)
- bubblewrap (optional, for sandbox testing)

### Get Source Code

```bash
git clone https://github.com/startvibecoding/mothx.git
cd mothx
```

### Install Dependencies

```bash
go mod download
```

### Build Project

```bash
# Build
make build

# Install to $GOPATH/bin
make install
```

### Run Tests

```bash
# Run all tests
make test

# Run tests for specific package
go test ./internal/tools/

# Run specific test
go test -run TestReadTool ./internal/tools/
```

## Project Structure

```
mothx/
├── agent/                     # Public Agent/Provider interfaces and Builder
├── bootstrap/                 # Bootstrap package for embedding
├── cmd/mothx/            # CLI entry point
│   ├── main.go                # Main program
│   └── main_doctor.go         # `doctor` subcommand
├── internal/
│   ├── a2a/                   # A2A protocol server and master mode
│   ├── acp/                   # ACP / MCP integration
│   ├── agent/                 # Core Agent loop
│   │   ├── agent.go           # Agent main logic
│   │   ├── factory.go         # AgentFactory
│   │   ├── manager.go         # AgentManager
│   │   ├── router.go          # EventRouter
│   │   ├── subagent.go        # subagent_* tools
│   │   ├── events.go          # Event type definitions
│   │   ├── provider.go        # Provider adapter
│   │   └── system_prompt.go   # System prompt generation
│   ├── config/                # Configuration management
│   ├── context/               # Context management and token estimation
│   ├── contextfiles/          # Context file loading
│   ├── cron/                  # Scheduled task store and scheduler
│   ├── gateway/               # OpenAI-compatible HTTP gateway
│   ├── hermes/                # Messaging gateway (WeChat/Feishu/WebSocket)
│   ├── mcp/                   # MCP server integration
│   ├── memory/                # Persistent memory (memory.md)
│   ├── messaging/             # Messaging platform abstraction
│   ├── platform/              # Cross-platform compatibility
│   ├── provider/              # LLM Provider abstraction
│   │   ├── anthropic/         # Anthropic Messages API
│   │   ├── google/            # Google Gemini/Vertex API
│   │   ├── openai/            # OpenAI Chat Completions API
│   │   ├── factory/           # Shared provider/model construction
│   │   └── vendor*.go         # Vendor adapter registry
│   ├── sandbox/               # Sandbox abstraction (bwrap, none)
│   ├── session/               # Session management (SQLite)
│   ├── skills/                # Skills system
│   ├── stats/                 # Usage stats web dashboard
│   ├── systeminit/            # AGENTS.md generation
│   ├── tools/                 # Tool implementations
│   │   ├── bash.go            # Bash command execution
│   │   ├── read.go            # File reading
│   │   ├── write.go           # File writing
│   │   ├── edit.go            # File editing
│   │   ├── grep.go            # Content search (pure-Go ripgrep)
│   │   ├── find.go            # File finding (pure-Go fd)
│   │   ├── ls.go              # Directory listing
│   │   ├── plan.go            # Task planning
│   │   ├── question.go        # User clarification
│   │   ├── skill_ref.go       # Skill reference loading
│   │   └── a2a_dispatch.go    # A2A remote agent dispatch
│   ├── tui/                   # Terminal UI (BubbleTea)
│   └── workflow/              # Workflow runtime (Elisp DSL)
├── example/                   # SDK examples
└── pkg/sdk/                   # Public SDK interface
```

## Core Interfaces

### Provider Interface

```go
type Provider interface {
    Name() string
    Models() []*Model
    GetModel(id string) *Model
    Chat(ctx context.Context, params ChatParams) <-chan StreamEvent
}
```

### Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() json.RawMessage
    Execute(ctx context.Context, params json.RawMessage) (string, error)
}
```

## Adding New Tools

### Step 1: Create Tool File

```go
// internal/tools/mytool.go
package tools

import (
    "context"
    "encoding/json"
)

type MyTool struct {
    workdir string
}

func NewMyTool(workdir string) *MyTool {
    return &MyTool{workdir: workdir}
}

func (t *MyTool) Name() string {
    return "mytool"
}

func (t *MyTool) Description() string {
    return "Description of my tool"
}

func (t *MyTool) Parameters() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "param1": {
                "type": "string",
                "description": "First parameter"
            }
        },
        "required": ["param1"]
    }`)
}

func (t *MyTool) Execute(ctx context.Context, params json.RawMessage) (string, error) {
    var p struct {
        Param1 string `json:"param1"`
    }
    if err := json.Unmarshal(params, &p); err != nil {
        return "", err
    }
    // Implement tool logic
    return "result", nil
}
```

### Step 2: Register Tool

In `internal/tools/tool.go`'s `RegisterDefaults()` method:

```go
func (r *Registry) RegisterDefaults() {
    r.Register(&ReadTool{workdir: r.workdir})
    r.Register(&WriteTool{workdir: r.workdir})
    r.Register(&EditTool{workdir: r.workdir})
    r.Register(&BashTool{workdir: r.workdir, sandbox: r.sandbox})
    r.Register(&GrepTool{workdir: r.workdir})
    r.Register(&FindTool{workdir: r.workdir})
    r.Register(&LsTool{workdir: r.workdir})
    r.Register(&MyTool{workdir: r.workdir}) // Add new tool
}
```

### Step 3: Update System Prompt

Add tool description in `internal/agent/system_prompt.go`.

### Step 4: Write Tests

```go
// internal/tools/mytool_test.go
package tools

import (
    "context"
    "testing"
)

func TestMyTool_Execute(t *testing.T) {
    tool := NewMyTool("/tmp")
    params := `{"param1": "value"}`
    
    result, err := tool.Execute(context.Background(), json.RawMessage(params))
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    if result != "expected" {
        t.Errorf("expected 'expected', got '%s'", result)
    }
}
```

## Adding Provider Support

Most new services should be added as vendor adapters, not new protocol
providers. If the service speaks OpenAI Chat Completions or Anthropic Messages,
reuse the generic provider and register vendor defaults in `internal/provider`.

### Add an OpenAI/Anthropic-Compatible Vendor

1. Create `internal/provider/vendor_myvendor.go`.
2. Register URL detection and defaults with `RegisterVendorAdapter`.
3. Add model `compat` flags only for behavior that differs from the generic protocol.
4. Add focused tests in `internal/provider` and, if request formatting changes, in `internal/provider/openai` or `internal/provider/anthropic`.

```go
package provider

func init() {
    RegisterVendorAdapter(simpleVendorAdapter{
        name:           "myvendor",
        domains:        []string{"api.myvendor.example"},
        thinkingFormat: "deepseek", // optional
        defaultAPI:     "openai-chat",
    })
}
```

Provider creation for CLI and ACP goes through `internal/provider/factory`, so
do not add vendor-specific creation code to `cmd/mothx/main.go` or
`internal/acp/acp.go`.

### Add a New Protocol Provider

Only add a new provider package when the service has a native protocol that is
not covered by OpenAI Chat Completions or Anthropic Messages.

1. Create `internal/provider/myprotocol`.
2. Implement `provider.Provider`.
3. Add construction support in `internal/provider/factory`.
4. Keep settings JSON compatibility stable.
5. Add provider and factory tests.

## Testing

### Run All Tests

```bash
make test
```

### Run Tests for Specific Package

```bash
go test ./internal/agent/
go test ./internal/tools/
```

### Generate Coverage Report

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Code Standards

### Formatting

```bash
make fmt
```

Or manually:

```bash
gofmt -w .
goimports -w .
```

### Linting

```bash
make lint
```

### Naming Conventions

- Package names: lowercase words, e.g., `tools`, `agent`
- Interface names: verbs or nouns, e.g., `Provider`, `Tool`
- Struct names: PascalCase, e.g., `ReadTool`, `AgentConfig`
- Function names: PascalCase, e.g., `NewAgent`, `Execute`
- Variable names: camelCase, e.g., `workdir`, `maxTokens`

### Error Handling

```go
// Good practice
result, err := doSomething()
if err != nil {
    return fmt.Errorf("do something: %w", err)
}

// Bad practice
result, _ := doSomething()
```

## Git Workflow

### Commit Convention

Use Conventional Commits:

```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `refactor`: Refactoring
- `test`: Tests
- `chore`: Miscellaneous

Examples:

```
feat(tools): add new search tool
fix(agent): fix streaming issue
docs(readme): update installation guide
```

### Pull Request

1. Fork project
2. Create feature branch
3. Commit changes
4. Run tests
5. Create Pull Request

## Debugging

### Enable Debug Logging

```bash
mothx --debug
```

### Using dlv

```bash
# Install dlv
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug
dlv debug ./cmd/mothx -- --debug
```

## Release Process

### Version Numbers

Use Semantic Versioning:

```
MAJOR.MINOR.PATCH

Example: 1.0.0, 1.1.0, 1.0.1
```

### Create Release

```bash
# Update version number
git tag -a v1.0.0 -m "Release v1.0.0"

# Push tags
git push --tags

# Build release packages
make build-all
```

### Publish to npm

```bash
# Publish release version
make npm-publish-all

# Publish pre-release version
make npm-publish-pre
```

Users can install via npm:

```bash
npm install -g mothx
```

### Publish to PyPI

```bash
# Install publishing tools
python3 -m pip install --user "setuptools>=77.0.0" build twine

# Publish release wheels
make pypi-publish

# Publish pre-release wheels
make pypi-publish-pre
```

Users can install via PyPI:

```bash
pipx install vibecoding-installer
```

## Frequently Asked Questions

### Q: How to add a new tool?

A: See [Adding New Tools](#adding-new-tools) section.

### Q: How to add a new Provider?

A: See [Adding New Providers](#adding-new-providers) section.

### Q: What to do if tests fail?

A:
1. Check Go version
2. Run `go mod tidy`
3. Check error logs

### Q: How to debug sandbox issues?

A:
1. Use `--debug` flag
2. Check if bwrap is installed: `bwrap --version`
3. Check system logs
