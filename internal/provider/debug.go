package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

var debugLogMu sync.Mutex

// DebugResponse is the complete response reconstructed from a streamed provider response.
// It intentionally records the final result rather than individual SSE fragments.
type DebugResponse struct {
	Provider   string          `json:"provider"`
	API        string          `json:"api"`
	Content    string          `json:"content,omitempty"`
	Reasoning  string          `json:"reasoning,omitempty"`
	ToolCalls  []ToolCallBlock `json:"toolCalls,omitempty"`
	StopReason string          `json:"stopReason,omitempty"`
	Usage      *Usage          `json:"usage,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// DebugJSON writes a JSON request or complete response to stderr and debug.log
// when --debug has enabled VIBECODING_DEBUG.
func DebugJSON(label string, body []byte) {
	if os.Getenv("VIBECODING_DEBUG") == "" {
		return
	}

	line := fmt.Sprintf("[DEBUG] %s: %s\n", label, body)
	debugLogMu.Lock()
	defer debugLogMu.Unlock()

	_, _ = fmt.Fprint(os.Stderr, line)
	file, err := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(line)
}

// DebugCompleteResponse marshals a reconstructed non-SSE response for debug logging.
func DebugCompleteResponse(response DebugResponse) {
	body, err := json.Marshal(response)
	if err != nil {
		DebugJSON("Response JSON marshal error", []byte(fmt.Sprintf(`{"error":%q}`, err.Error())))
		return
	}
	DebugJSON("Response JSON", body)
}
