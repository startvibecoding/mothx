package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fuckvibecoding/vibecoding/internal/sandbox"
)

// BashTool executes shell commands.
type BashTool struct {
	registry *Registry
}

// NewBashTool creates a new bash tool.
func NewBashTool(r *Registry) *BashTool {
	return &BashTool{registry: r}
}

func (t *BashTool) Name() string { return "bash" }

func (t *BashTool) Description() string {
	return "Execute a bash command. Use this to run shell commands, scripts, build commands, etc. The command runs in the current working directory. Set timeout for long-running commands (default 120s, max 600s)."
}

func (t *BashTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The bash command to execute"
			},
			"timeout": {
				"type": "integer",
				"description": "Timeout in seconds (default 120, max 600)"
			}
		},
		"required": ["command"]
	}`)
}

func (t *BashTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	command, _ := params["command"].(string)
	if command == "" {
		return "", fmt.Errorf("command is required")
	}

	timeout := 120 * time.Second
	if v, ok := params["timeout"].(float64); ok && v > 0 {
		if v > 600 {
			v = 600
		}
		timeout = time.Duration(v) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	shell := "/bin/bash"
	if s := os.Getenv("SHELL"); s != "" {
		shell = s
	}

	workDir := t.registry.GetWorkDir()

	var cmd *exec.Cmd
	sb := t.registry.GetSandbox()
	if sb != nil && sb.IsAvailable() {
		opts := sandbox.ExecOpts{
			WorkDir: workDir,
			Timeout: timeout,
			EnvVars: make(map[string]string),
		}
		cmd = sb.WrapCommand(ctx, shell, command, opts)
	} else {
		cmd = exec.CommandContext(ctx, shell, "-c", command)
		cmd.Dir = workDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += "STDERR:\n" + stderr.String()
	}

	// Truncate large outputs
	const maxOutput = 50000
	if len(output) > maxOutput {
		truncated := len(output) - maxOutput
		output = output[:maxOutput] + fmt.Sprintf("\n... (truncated %d bytes)", truncated)
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Sprintf("Exit code: %d\n%s", exitErr.ExitCode(), output), nil
		}
		return "", fmt.Errorf("command failed: %w\n%s", err, output)
	}

	if output == "" {
		return "(no output)", nil
	}
	return output, nil
}

// SetTool is an interface for tools that need sandbox updates.
type SetTool interface {
	SetSandbox(sb sandbox.Sandbox)
}

// FileTool is a base for file-related tools.
type FileTool struct {
	registry *Registry
}

func (t *FileTool) resolvePath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		return strings.Replace(path, "~", home, 1)
	}
	if !strings.HasPrefix(path, "/") {
		return t.registry.GetWorkDir() + "/" + path
	}
	return path
}
