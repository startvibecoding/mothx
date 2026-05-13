package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// KillTool stops a running background job.
type KillTool struct {
	registry *Registry
	bashTool *BashTool
}

// NewKillTool creates a new kill tool.
func NewKillTool(r *Registry, bashTool *BashTool) *KillTool {
	return &KillTool{registry: r, bashTool: bashTool}
}

func (t *KillTool) Name() string { return "kill" }

func (t *KillTool) Description() string {
	return "Stop a running background job started with bash async=true."
}

func (t *KillTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"jobId": {
				"type": "integer",
				"description": "The job ID to kill"
			}
		},
		"required": ["jobId"]
	}`)
}

func (t *KillTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	jobID, ok := params["jobId"].(float64)
	if !ok {
		return "", fmt.Errorf("jobId is required")
	}

	jm := t.bashTool.GetJobManager()
	job := jm.GetJob(int(jobID))
	if job == nil {
		return "", fmt.Errorf("job %d not found", int(jobID))
	}

	if job.IsDone() {
		return fmt.Sprintf("Job %d already finished.", int(jobID)), nil
	}

	if err := jm.KillJob(int(jobID)); err != nil {
		return "", fmt.Errorf("failed to kill job %d: %w", int(jobID), err)
	}

	return fmt.Sprintf("Sent kill signal to job %d (PID: %d).", int(jobID), job.PID), nil
}
