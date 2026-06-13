# Tool System Reference Guide

VibeCoding provides a powerful and extensible set of built-in tools for file operations, codebase exploration, shell execution, multi-agent coordination, and workspace interaction.

---

## 1. Tool Execution Safety and Sandboxing

Before exploring individual tools, it is crucial to understand the three safety levels under which VibeCoding can execute tools:

| Safety Level | Tool Modification Privileges | Network Access | Sandbox Implementation |
|--------------|------------------------------|----------------|------------------------|
| **none** | Can read, write, and execute shell commands anywhere. | Fully enabled. | Direct host execution. |
| **standard** | Read/write in project directory; read-only in system directories. | Fully disabled. | Restricts host access via `bwrap` (Bubblewrap) if available. |
| **strict** | Read-only in project and system directories. No writes/edits allowed. | Fully disabled. | Restricts host access via `bwrap` (Bubblewrap) if available. |

### Sandbox Mechanics (`bwrap`)
When `sandbox.enabled` is `true` in `settings.json`, VibeCoding isolates commands run via `bash` (or any external process):
* **Allowed Paths**: The sandbox mounts system folders (`/usr`, `/lib`, `/bin`, etc.) as read-only, and mounts the active workspace directory as read-write (`standard`) or read-only (`strict`).
* **Denied Paths**: Sensitive directories (like `~/.ssh`, `/etc/shadow`, etc.) are completely hidden or blocked.
* **Network Isolation**: Direct networking is blocked by creating a separate network namespace (`--unshare-net`).

---

## 2. Comprehensive Tool Directory

| Tool | Category | Action | Execution Security | Availability |
|------|----------|--------|-------------------|--------------|
| [`read`](#read---file-reading) | File System | Read file content (supports images and pagination) | Read-only access allowed | All modes |
| [`write`](#write---file-writing) | File System | Create or overwrite files | Requires Standard/Yolo | CLI, ACP, Gateway, Hermes |
| [`edit`](#edit---precise-file-modification) | File System | Atomic text replacements | Requires Standard/Yolo | CLI, ACP, Gateway, Hermes |
| [`ls`](#ls---directory-listing) | Exploration | List directory contents | Read-only access allowed | All modes |
| [`find`](#find---file-search) | Exploration | Search for files by pattern | Read-only access allowed | All modes |
| [`grep`](#grep---text-content-search) | Exploration | Regex search within files | Read-only access allowed | All modes |
| [`bash`](#bash---command-execution) | Execution | Run shell commands (supports sync & async) | Subject to Sandbox Level | CLI, ACP, Gateway, Hermes |
| [`jobs`](#jobs---background-job-management) | Execution | List or check status of background jobs | Read-only access allowed | All modes |
| [`kill`](#kill---stop-background-job) | Execution | Stop a running background job | Requires Standard/Yolo | CLI, ACP, Gateway, Hermes |
| [`plan`](#plan---task-planning) | Session | Update a visible progress/task plan | Read-only access allowed | All modes |
| [`question`](#question---user-clarification) | Session | Prompt user for multiple-choice input | Plan mode only | TUI Only |
| [`memory`](#memory---persistent-memory) | Workflow | Read/write to persistent `memory.md` | Session-scoped read/write | Hermes Mode |
| [`cron`](#cron---scheduled-background-tasks) | Workflow | Schedule background tasks via sub-agents | Session-scoped scheduling | Hermes & Multi-Agent |
| [`subagent_spawn`](#subagent_---delegated-work) | Multi-Agent | Spawn isolated sub-agents | Sub-agent scoped limits | Multi-Agent Mode |
| [`subagent_status`](#subagent_---delegated-work) | Multi-Agent | Query sub-agent status | Read-only | Multi-Agent Mode |
| [`subagent_send`](#subagent_---delegated-work) | Multi-Agent | Send commands to sub-agents | Send message | Multi-Agent Mode |
| [`subagent_destroy`](#subagent_---delegated-work) | Multi-Agent | Remove sub-agents & clean up | Destroy | Multi-Agent Mode |
| [`a2a_dispatch`](#a2a_dispatch---remote-agent-dispatch) | Multi-Agent | Dispatch tasks to a remote A2A agent | Network request | A2A Master Mode |
| [`skill_ref`](#skill_ref---skill-reference-loading) | Skills | Load external skill documentation | Read-only | All modes |

---

## 3. Tool Details & Parameters

### read - File Reading

Read file contents with support for pagination (vital for handling large source files without blowing up the context window) and image rendering.

#### Parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | ✓ | Path to the file. |
| `offset` | integer | - | Starting line number (1-indexed). |
| `limit` | integer | - | Maximum number of lines to read. |

#### Image File Support:
If the file path ends with a supported image extension (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`), the tool automatically encodes the image content in Base64 and returns a rich content block with its MIME type, which enables multimodal models to "see" the image.

#### Example Payload:
```json
{
  "path": "src/main.go",
  "offset": 120,
  "limit": 50
}
```

---

### write - File Writing

Create a new file or completely overwrite an existing file. This operation is atomic (uses a temporary file first, then renames it) to prevent corruption.

#### Parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | ✓ | Path to write. |
| `content` | string | ✓ | Full text content of the file. |

#### Example Payload:
```json
{
  "path": "config.json",
  "content": "{\n  \"port\": 8080,\n  \"debug\": true\n}"
}
```

---

### edit - Precise File Modification

The `edit` tool performs highly precise, **atomic multi-replacement operations** on a single file. Understanding how this tool operates under the hood is critical:

#### Edit Lifecycle & Safety Rules:
1. **Pre-Flight Validation**: Before any modification is made, VibeCoding reads the file and searches for each `oldText` block in `edits[]`.
2. **Uniqueness Check**: Every `oldText` pattern **must match exactly once** in the target file. If a pattern matches 0 times or matches multiple times, the **entire** batch of edits fails immediately with an error, and the file is untouched.
3. **Overlap Resolution**: VibeCoding sorts the matches by their start index and checks if any edit overlaps with another. If an overlap is detected, the operation aborts to prevent corrupt edits.
4. **Atomic Application**: Edits are applied in sorted order based on their indices in the original file. No partial edits are written. The file is modified atomically via a temporary file write.

#### Parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | ✓ | File to edit. |
| `edits` | array of objects | ✓ | List of replacement blocks. |

**Edit Object Structure:**

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `oldText` | string | ✓ | Exact block of text to replace. |
| `newText` | string | ✓ | The replacement text. |

#### Example Payload:
```json
{
  "path": "server.go",
  "edits": [
    {
      "oldText": "func Start() {\n\tlog.Println(\"Starting server...\")",
      "newText": "func Start() {\n\tlog.Println(\"Booting microservice...\")"
    }
  ]
}
```

---

### bash - Command Execution

Executes a command using the shell (e.g., `/bin/bash` or `PowerShell`). It supports both synchronous (blocking) and asynchronous (background) execution modes.

#### Parameters:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `command` | string | ✓ | - | Shell command string to execute. |
| `timeout` | integer | - | 120 | Maximum execution time in seconds (max 600). |
| `async` | boolean | - | false | If `true` (or if command ends with `&`), the process starts in the background and returns a `jobId` immediately. |

#### Example Payload (Synchronous):
```json
{
  "command": "go test -v ./internal/provider/...",
  "timeout": 60
}
```

#### Example Payload (Asynchronous):
```json
{
  "command": "npm run dev",
  "async": true
}
```

---

### jobs - Background Job Management

List, query, or clean up asynchronous shell processes spawned with `bash` (`async=true`).

#### Parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `jobId` | integer | - | Query detailed logs, start time, CPU elapsed time, and status for a specific job. |
| `cleanup` | boolean | - | Remove finished jobs from the manager list. |

#### Example Payload (List All):
```json
{}
```

#### Example Payload (Query Specific Job):
```json
{
  "jobId": 1
}
```

---

### kill - Stop Background Job

Forcefully stop (send SIGTERM / SIGKILL) an asynchronous background job started via the `bash` tool.

#### Parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `jobId` | integer | ✓ | ID of the job to terminate. |

#### Example Payload:
```json
{
  "jobId": 1
}
```

---

### ls - Directory Listing

List files and directories inside a target path, giving exact file sizes, modification statuses, and types.

#### Parameters:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | - | `.` | Directory to list. |

#### Example Payload:
```json
{
  "path": "./internal"
}
```

---

### find - File Search

Find file paths matching glob patterns. This tool uses an embedded `fd` binary (falls back to system `find` if unsupported) and is extremely fast.

#### Parameters:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `pattern` | string | ✓ | - | Glob pattern to search (e.g. `*.go`, `*test*`). |
| `path` | string | - | `.` | Root directory for search. |
| `maxDepth` | integer | - | - | Maximum search depth. |
| `maxResults`| integer | - | 100 | Limit on number of results. |

#### Example Payload:
```json
{
  "pattern": "*_test.go",
  "path": "internal/provider",
  "maxDepth": 3
}
```

---

### grep - Text Content Search

Perform fast regex-based searches across codebase files. Uses embedded `rg` (ripgrep) for near-instant results.

#### Parameters:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `pattern` | string | ✓ | - | Regular expression pattern. |
| `path` | string | - | `.` | Directory or file path to search. |
| `include` | string | - | - | Glob pattern for files to include (e.g., `*.go`). |
| `maxResults`| integer | - | 100 | Limit on returned matches. |

#### Example Payload:
```json
{
  "pattern": "type ProviderConfig struct",
  "include": "*.go"
}
```

---

### plan - Task Planning

Renders a beautiful multi-step checklist panel in the TUI, helping users and developers monitor exactly what steps the assistant intends to execute and which ones have succeeded or failed.

#### Parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `steps` | array of objects | ✓ | Ordered steps of the task. |
| `title` | string | - | Short descriptive title of the plan. |
| `note` | string | - | Current risk warning, blocker, or status note. |

**Step Object Structure:**

| Property | Type | Required | Allowed Values | Description |
|----------|------|----------|----------------|-------------|
| `title` | string | ✓ | - | Concise description of the step. |
| `status` | string | ✓ | `pending`, `running`, `done`, `failed` | Execution state. |

#### Example Payload:
```json
{
  "title": "Database Migration",
  "steps": [
    { "title": "Check current schema", "status": "done" },
    { "title": "Run migration script", "status": "running" },
    { "title": "Verify data consistency", "status": "pending" }
  ]
}
```

---

### question - User Clarification

In `plan` mode inside the TUI, the assistant can halt execution and ask the user a multiple-choice question to clarify requirements. The interactive TUI lets the user navigate options and select the correct path.

#### Parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `question` | string | ✓ | The question being asked. |
| `options` | array of strings | ✓ | List of choices for the user. |

#### Example Payload:
```json
{
  "question": "Which styling library would you prefer for the frontend components?",
  "options": [
    "Tailwind CSS",
    "Styled Components",
    "Vanilla CSS / CSS Modules"
  ]
}
```

---

### memory - Persistent Memory

In Hermes messaging mode, the assistant can record user preferences, decisions, and system rules in `.vibe/memory.md`. This memory is loaded at the beginning of each session.

#### Parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | ✓ | Action to take: `read`, `add`, `update`, `delete`. |
| `section` | string | - | Heading section (e.g. `Lessons Learned`, `Project Rules`). |
| `content` | string | - | Text block to write or match. |
| `old` | string | - | Target text to replace (used in `update`). |
| `new` | string | - | New replacement text (used in `update`). |

#### Example Payload (Add Memory):
```json
{
  "action": "add",
  "section": "Project Rules",
  "content": "All API endpoints must be documented in OpenAPI 3.0 format."
}
```

---

### cron - Scheduled Background Tasks

Enables creating and managing cron-like scheduled tasks that execute in the background via sub-agents. 

#### Parameters:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `action` | string | ✓ | - | Action: `list`, `create`, `enable`, `disable`, `remove`, `run`. |
| `id` | string | - | - | Job ID (required for edit/remove actions). |
| `name` | string | - | - | Short mnemonic name. |
| `prompt` | string | - | - | The prompt instruction sent to the sub-agent. |
| `schedule` | string | - | - | Cron notation or interval (e.g. `@daily`, `@every 15m`). |
| `oneshot` | boolean | - | false | Run only once and then disable itself. |
| `mode` | string | - | `yolo` | Sub-agent execution mode (`agent` or `yolo`). |

#### Example Payload:
```json
{
  "action": "create",
  "name": "daily-git-pull",
  "prompt": "Pull the latest upstream branch and verify compilation.",
  "schedule": "@daily",
  "mode": "yolo"
}
```

---

### subagent_* - Delegated Work

When VibeCoding is launched in multi-agent mode (`--multi-agent`), the main agent can spin up completely separate sub-agents to parallelize work. Sub-agents run in their own session contexts, have separate memories, separate temporary files, and isolated logs.

#### `subagent_spawn`
Launches a sub-agent to handle a focused prompt task asynchronously.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `task` | string | ✓ | Prompt describing the precise task to fulfill. |
| `mode` | string | - | Run mode (`plan`, `agent`, or `yolo`; defaults to `agent`). |
| `work_dir` | string | - | Specific subdirectory to run the sub-agent in. |
| `tools` | array of strings | - | Whitelist of allowed tool names (empty for all). |
| `max_iterations` | integer | - | Maximum agent loop cycles. |

#### `subagent_status`
Queries progress and returns the result once complete.
```json
{ "handle": "subagent-job-1" }
```

#### `subagent_send`
Appends additional guidance or questions to an active sub-agent.
```json
{ "handle": "subagent-job-1", "message": "Also fix imports in helper_test.go." }
```

#### `subagent_destroy`
Cleans up logs and releases the sub-agent container/process context.
```json
{ "handle": "subagent-job-1" }
```

---

### a2a_dispatch - Remote Agent Dispatch

Only available in A2A Master Mode. Dispatches a task request over network to a remote agent endpoint registered in the local configuration files.

#### Parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `agent_name` | string | ✓ | Remote agent alias registered in `a2a-list.json`. |
| `message` | string | ✓ | Full instruction to send to the remote agent. |

#### Example Payload:
```json
{
  "agent_name": "kubernetes-deployer",
  "message": "Deploy image tag v1.2.3 to development namespace."
}
```

---

### skill_ref - Skill Reference Loading

Load reference context from a localized skill. This keeps large reference schemas, api endpoints, and documentation out of the main agent prompt until specifically requested.

#### Parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `skill` | string | ✓ | Skill directory name. |
| `ref` | string | ✓ | Path of the reference file inside the skill folder. |

#### Example Payload:
```json
{
  "skill": "kubernetes-skill",
  "ref": "references/helm-deployment.md"
}
```

---

## 4. Best Practices for Developers & Users

To get the absolute best out of VibeCoding's tool loop, keep the following principles in mind:

1. **The Read-Before-Modify Standard**: Always ensure you (or the model) call `read` on files before attempting to modify them with `edit`. Having the exact, complete block in context ensures the `oldText` parameter matches perfectly and prevents atomic editing failures.
2. **Minimize `bash` usage for exploration**: Avoid running shell commands like `grep`, `find`, or `cat` inside a bash tool. VibeCoding's dedicated `grep`, `find`, and `read` tools are heavily optimized, parsed directly into structured data, and do not suffer from sub-shell spawn latencies or sandbox restrictions.
3. **Handle Long-Running Jobs with Care**: When spinning up servers or continuous builders using `bash` in async mode (`async=true`), regularly call `jobs` to clean up exited jobs (`cleanup=true`) to keep the system overhead down, and use `kill` to shut down background servers once they are no longer required.
