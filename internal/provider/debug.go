package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

var debugLogMu sync.Mutex

// DebugLogOnlyEnv suppresses debug stderr output while retaining debug.log.
// The interactive TUI sets it so asynchronous provider output cannot disrupt
// Bubble Tea rendering.
const DebugLogOnlyEnv = "VIBECODING_DEBUG_LOG_ONLY"

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

// DebugJSON writes a JSON request or complete response to debug.log when
// --debug has enabled VIBECODING_DEBUG. Non-TUI callers also receive it on
// stderr unless DebugLogOnlyEnv is set.
func DebugJSON(label string, body []byte) {
	if os.Getenv("VIBECODING_DEBUG") == "" {
		return
	}

	line := fmt.Sprintf("[DEBUG] %s: %s\n", label, body)
	debugLogMu.Lock()
	defer debugLogMu.Unlock()

	file, err := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err == nil {
		_, _ = file.WriteString(line)
		_ = file.Close()
	}
	if os.Getenv(DebugLogOnlyEnv) == "" {
		_, _ = fmt.Fprint(os.Stderr, line)
	}
}

// DebugCompleteResponse marshals a reconstructed non-SSE response for debug logging.
func DebugCompleteResponse(response DebugResponse) {
	body, err := json.Marshal(response)
	if err != nil {
		fallback, _ := json.Marshal(struct {
			Error    string `json:"error"`
			Response string `json:"response"`
		}{
			Error:    err.Error(),
			Response: debugResponseDump(response),
		})
		DebugJSON("Response JSON marshal error", fallback)
		return
	}
	DebugJSON("Response JSON", body)
}

// debugResponseDump preserves fields that json.Marshal cannot encode, notably
// invalid json.RawMessage tool arguments reconstructed from a stream.
func debugResponseDump(response DebugResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "provider=%q api=%q content=%q reasoning=%q stopReason=%q usage=%+v error=%q", response.Provider, response.API, response.Content, response.Reasoning, response.StopReason, response.Usage, response.Error)
	for i, call := range response.ToolCalls {
		fmt.Fprintf(&b, " toolCall[%d]={id=%q name=%q arguments=%q invalidArguments=%q thoughtSignature=%q}", i, call.ID, call.Name, string(call.Arguments), call.InvalidArguments, call.ThoughtSignature)
	}
	return b.String()
}
