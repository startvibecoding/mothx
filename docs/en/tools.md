# Tool System

VibeCoding provides a set of built-in tools for file operations, code search, and command execution.

## Tool Overview

| Tool | Description | Sandbox Restriction |
|------|-------------|-------------------|
| `read` | Read file content | Read-only directories accessible |
| `write` | Create/overwrite files | Only standard/yolo |
| `edit` | Precise text replacement | Only standard/yolo |
| `bash` | Execute shell commands | Subject to sandbox restrictions |
| `grep` | Regex content search | Read-only |
| `find` | Filename search | Read-only |
| `ls` | List directory contents | Read-only |

## Tool Details

### read - File Reading

Read file content with pagination support.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | ✓ | File path |
| `offset` | int | - | Starting line number (from 1) |
| `limit` | int | - | Maximum lines to read |

**Example:**

```json
{
  "path": "/home/user/project/main.go",
  "offset": 10,
  "limit": 50
}
```

**Returns:** File content text

---

### write - File Writing

Create new files or overwrite existing files.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | ✓ | File path |
| `content` | string | ✓ | File content |

**Example:**

```json
{
  "path": "/home/user/project/README.md",
  "content": "# My Project\n\nThis is a new project."
}
```

**Returns:** Success/failure message

---

### edit - File Editing

Precise text replacement for modifying existing files.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | ✓ | File path |
| `edits` | array | ✓ | List of edit operations |

**edits array elements:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `oldText` | string | ✓ | Exact text to find |
| `newText` | string | ✓ | Replacement text |

**Example:**

```json
{
  "path": "/home/user/project/main.go",
  "edits": [
    {
      "oldText": "func main() {\n\tfmt.Println(\"old\")\n}",
      "newText": "func main() {\n\tfmt.Println(\"new\")\n}"
    }
  ]
}
```

**Best Practices:**

1. `oldText` must exactly match the text in the file, including spaces and newlines
2. Use `read` first to get file content and ensure `oldText` is correct
3. Use sufficiently long `oldText` to ensure unique matching
4. A single call can contain multiple edit operations

---

### bash - Command Execution

Execute shell commands.

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `command` | string | ✓ | - | Command to execute |
| `timeout` | int | - | 120 | Timeout in seconds |

**Example:**

```json
{
  "command": "go test ./...",
  "timeout": 300
}
```

**Returns:** stdout and stderr output

**Sandbox Behavior:**

| Sandbox Level | File System | Network | Description |
|---------------|------------|---------|-------------|
| none | Full access | Allowed | No restrictions |
| standard | Project read/write | Disabled | Can only modify project files |
| strict | Project read-only | Disabled | Can only read project files |

---

### grep - Content Search

Search file content using regular expressions.

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `pattern` | string | ✓ | - | Regular expression |
| `path` | string | - | Current directory | Search path |
| `include` | string | - | - | File matching pattern (e.g., `*.go`) |
| `maxResults` | int | - | 100 | Maximum number of results |

**Example:**

```json
{
  "pattern": "func\\s+\\w+\\(",
  "path": "/home/user/project",
  "include": "*.go",
  "maxResults": 50
}
```

**Returns:** Matching lines with file paths and line numbers

---

### find - File Search

Search files by filename pattern.

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `pattern` | string | ✓ | - | Glob pattern |
| `path` | string | - | Current directory | Search path |
| `maxDepth` | int | - | Unlimited | Maximum directory depth |
| `maxResults` | int | - | 100 | Maximum number of results |

**Example:**

```json
{
  "pattern": "*.go",
  "path": "/home/user/project",
  "maxDepth": 3
}
```

**Returns:** List of matching file paths

---

### ls - Directory Listing

List directory contents.

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | - | Current directory | Directory path |

**Example:**

```json
{
  "path": "/home/user/project"
}
```

**Returns:** Directory contents list with file types and sizes

---

## Tool Usage Patterns

### Read-Modify-Write Pattern

This is the most common code editing pattern:

```
1. read   → Get file content
2. edit   → Make precise modifications
3. bash   → Verify changes (e.g., go build)
```

**Example Conversation:**

```
User: Fix the bug in main.go

Assistant:
  1. read("main.go")           # Read file
  2. Analyze code, find bug
  3. edit("main.go", edits)    # Fix bug
  4. bash("go build ./...")    # Verify compilation
```

### Search-Locate-Modify Pattern

When file location is unknown:

```
1. grep   → Search for relevant code
2. read   → View context
3. edit   → Modify code
```

### Project Exploration Pattern

Understanding project structure:

```
1. ls     → List root directory
2. find   → Find specific files
3. read   → Read key files
```

## Tool Error Handling

Tool execution failures return error messages:

```json
{
  "error": "open /path/to/file: no such file or directory"
}
```

Common error types:

| Error | Cause | Solution |
|-------|-------|----------|
| File not found | Wrong path or file deleted | Check path, use `find` to locate |
| Permission denied | Sandbox restriction or file permissions | Check sandbox level, verify file permissions |
| Timeout | Command execution too long | Increase timeout or optimize command |
| Edit failed | `oldText` doesn't match | Re-`read` to get latest content |

## Tool Limitations

### Sandbox Restrictions

In sandbox mode:

- **standard**: Project directory read/write, system directory read-only, no network
- **strict**: All directories read-only, no network

### Timeout Limits

- Default timeout: 120 seconds
- Maximum timeout: 600 seconds
- Long-running commands need larger timeout setting

### Output Limits

- Single output has size limit
- Excess content is truncated
- Use `offset` and `limit` for paginated reading of large files

## Extending Tools

### Custom Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() json.RawMessage
    Execute(ctx context.Context, params json.RawMessage) (string, error)
}
```

### Register Custom Tool

```go
registry := tools.NewRegistry(workdir, sandbox)
registry.Register(&MyCustomTool{})
```

## Best Practices

1. **Read before modifying**: Use `read` to view file content, then use `edit` to modify
2. **Precise matching**: `edit`'s `oldText` must match exactly
3. **Verify changes**: Use `bash` to verify after modification (e.g., compile, test)
4. **Paginated reading**: Use `offset` and `limit` for large files
5. **Limit searches**: Use `include` and `maxResults` to limit search scope