package openaiapi

import (
	"encoding/json"
	"fmt"
	"time"
)

// --- OpenAI-compatible request types ---

// ChatCompletionRequest represents the OpenAI chat completions request.
type ChatCompletionRequest struct {
	Model       string           `json:"model,omitempty"`
	Messages    []RequestMessage `json:"messages"`
	Stream      bool             `json:"stream,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	TopP        *float64         `json:"top_p,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`

	// VibeCoding extensions
	XSessionID  string              `json:"x_session_id,omitempty"`
	XMode       string              `json:"x_mode,omitempty"`
	XWorkingDir string              `json:"x_working_dir,omitempty"`
	XTools      *SessionToolOptions `json:"x_tools,omitempty"`
	XTranscript bool                `json:"x_transcript,omitempty"`
}

// SessionToolOptions are per-session runtime tool toggles supplied by WebUI.
type SessionToolOptions struct {
	WebSearch  *bool `json:"webSearch,omitempty"`
	Browser    *bool `json:"browser,omitempty"`
	A2AMaster  *bool `json:"a2aMaster,omitempty"`
	Delegate   *bool `json:"delegate,omitempty"`
	MultiAgent *bool `json:"multiAgent,omitempty"`
	Workflows  *bool `json:"workflows,omitempty"`
}

// CapabilityFeature describes serve-level capability availability and defaults.
type CapabilityFeature struct {
	Available bool   `json:"available"`
	Default   bool   `json:"default"`
	Locked    bool   `json:"locked,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// CapabilityOverview is returned by GET /api/capabilities.
type CapabilityOverview struct {
	Modes    []string                     `json:"modes"`
	Features map[string]CapabilityFeature `json:"features"`
	Defaults SessionCapabilities          `json:"defaults"`
}

// SessionCapabilities are the effective runtime capabilities for a session.
type SessionCapabilities struct {
	ID              string `json:"id,omitempty"`
	WorkDir         string `json:"workDir,omitempty"`
	Active          bool   `json:"active"`
	Mode            string `json:"mode"`
	DelegateMode    bool   `json:"delegateMode"`
	Delegate        bool   `json:"delegate"`
	MultiAgent      bool   `json:"multiAgent"`
	Workflows       bool   `json:"workflows"`
	WebSearch       bool   `json:"webSearch"`
	Browser         bool   `json:"browser"`
	A2AMaster       bool   `json:"a2aMaster"`
	Model           string `json:"model,omitempty"`
	ThinkingLevel   string `json:"thinkingLevel,omitempty"`
	Persisted       bool   `json:"persisted"`
	RuntimeOnly     bool   `json:"runtimeOnly,omitempty"`
	PersistenceNote string `json:"persistenceNote,omitempty"`
}

// SessionCapabilityPatch updates mutable session runtime capabilities.
type SessionCapabilityPatch struct {
	Mode         *string `json:"mode,omitempty"`
	DelegateMode *bool   `json:"delegateMode,omitempty"`
	Delegate     *bool   `json:"delegate,omitempty"`
	MultiAgent   *bool   `json:"multiAgent,omitempty"`
	Workflows    *bool   `json:"workflows,omitempty"`
	WebSearch    *bool   `json:"webSearch,omitempty"`
	Browser      *bool   `json:"browser,omitempty"`
	A2AMaster    *bool   `json:"a2aMaster,omitempty"`
}

// SessionRuntimePatch is the structured WebUI runtime patch payload. Mode is
// kept separate from capabilities while capability toggles remain session-level
// user intent.
type SessionRuntimePatch struct {
	Mode         *string             `json:"mode,omitempty"`
	Capabilities map[string]bool     `json:"capabilities,omitempty"`
	Tools        *SessionToolOptions `json:"tools,omitempty"`
}

// SessionRuntimeSnapshot is the structured WebUI view for runtime state.
type SessionRuntimeSnapshot struct {
	SessionID        string                            `json:"sessionId"`
	Mode             string                            `json:"mode"`
	Model            string                            `json:"model,omitempty"`
	ThinkingLevel    string                            `json:"thinkingLevel,omitempty"`
	WorkDir          string                            `json:"workDir,omitempty"`
	Capabilities     map[string]SessionCapabilityState `json:"capabilities"`
	PendingApprovals []SessionApprovalRequest          `json:"pendingApprovals"`
	ActiveRun        *SessionActiveRun                 `json:"activeRun,omitempty"`
}

// SessionCapabilityState describes availability, desired enabled state and
// effective runtime state for one WebUI capability.
type SessionCapabilityState struct {
	Available      bool   `json:"available"`
	Enabled        bool   `json:"enabled"`
	Effective      bool   `json:"effective"`
	DisabledReason string `json:"disabledReason,omitempty"`
}

// SessionActiveRun describes the currently running session run, if any.
type SessionActiveRun struct {
	RunID  string `json:"runId,omitempty"`
	Status string `json:"status"`
}

// SessionApprovalRequest is the WebUI approval-center event shape.
type SessionApprovalRequest struct {
	ApprovalID string         `json:"approvalId"`
	SessionID  string         `json:"sessionId"`
	RunID      string         `json:"runId,omitempty"`
	Timestamp  string         `json:"timestamp,omitempty"`
	AgentID    string         `json:"agentId,omitempty"`
	Mode       string         `json:"mode,omitempty"`
	Risk       string         `json:"risk,omitempty"`
	Summary    string         `json:"summary,omitempty"`
	Reason     string         `json:"reason,omitempty"`
	Tool       map[string]any `json:"tool,omitempty"`
	Context    map[string]any `json:"context,omitempty"`
	Actions    []string       `json:"actions,omitempty"`
}

// RequestMessage represents a message in the OpenAI request.
type RequestMessage struct {
	Role         string               `json:"role"`
	Content      string               `json:"content"`
	ContentParts []RequestContentPart `json:"-"`
	Name         string               `json:"name,omitempty"`
}

// RequestContentPart represents one OpenAI-compatible multimodal content part.
type RequestContentPart struct {
	Type     string            `json:"type"`
	Text     string            `json:"text,omitempty"`
	ImageURL *RequestImageURL  `json:"image_url,omitempty"`
	Image    *RequestImageData `json:"image,omitempty"`
}

// RequestImageURL represents an OpenAI image_url content part.
type RequestImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// RequestImageData represents an internal image content part shape.
type RequestImageData struct {
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
	Detail   string `json:"detail,omitempty"`
}

// UnmarshalJSON accepts both classic string content and OpenAI-style content arrays.
func (m *RequestMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
		Name    string          `json:"name,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Role = raw.Role
	m.Name = raw.Name
	m.Content = ""
	m.ContentParts = nil
	if len(raw.Content) == 0 || string(raw.Content) == "null" {
		return nil
	}
	var text string
	if err := json.Unmarshal(raw.Content, &text); err == nil {
		m.Content = text
		return nil
	}
	var parts []RequestContentPart
	if err := json.Unmarshal(raw.Content, &parts); err != nil {
		return fmt.Errorf("content must be a string or content array")
	}
	m.ContentParts = parts
	for _, part := range parts {
		if part.Type == "text" && part.Text != "" {
			if m.Content != "" {
				m.Content += "\n"
			}
			m.Content += part.Text
		}
	}
	return nil
}

// --- OpenAI-compatible response types ---

// ChatCompletionResponse is the non-streaming response.
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   *CompletionUsage       `json:"usage,omitempty"`

	// VibeCoding extensions
	XSessionID string      `json:"x_session_id,omitempty"`
	XCommand   string      `json:"x_command,omitempty"`
	XToolCalls []XToolCall `json:"x_tool_calls,omitempty"`
}

// ChatCompletionChoice is a single choice in the response.
type ChatCompletionChoice struct {
	Index        int              `json:"index"`
	Message      *ResponseMessage `json:"message,omitempty"`
	Delta        *ResponseMessage `json:"delta,omitempty"`
	FinishReason *string          `json:"finish_reason"`
}

// ResponseMessage is the assistant's response message.
type ResponseMessage struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// CompletionUsage tracks token counts.
type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CacheReadTokens  int `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
}

// XToolCall is a VibeCoding extension for exposing tool call info.
type XToolCall struct {
	Name   string         `json:"name"`
	Args   map[string]any `json:"args,omitempty"`
	Status string         `json:"status"` // "running", "completed", "failed"
}

// --- Streaming chunk types ---

// ChatCompletionChunk is the streaming chunk response.
type ChatCompletionChunk struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   *CompletionUsage       `json:"usage,omitempty"`

	// VibeCoding extensions
	XSessionID string `json:"x_session_id,omitempty"`
}

// --- SSE tool_status event (for sse_event mode) ---

// ToolStatusEvent is sent via SSE event: tool_status.
type ToolStatusEvent struct {
	SessionID  string         `json:"sessionId,omitempty"`
	RunID      string         `json:"runId,omitempty"`
	Timestamp  string         `json:"timestamp,omitempty"`
	Tool       string         `json:"tool"`
	ToolCallID string         `json:"toolCallId,omitempty"`
	AgentID    string         `json:"agentId,omitempty"`
	Status     string         `json:"status"` // "running", "completed", "failed"
	Args       map[string]any `json:"args,omitempty"`
	Summary    string         `json:"summary,omitempty"`
	IsError    bool           `json:"isError,omitempty"`
	HasDetail  bool           `json:"hasDetail,omitempty"`
}

// TranscriptStreamEvent is a WebUI-oriented SSE event that mirrors the
// /api/sessions/{id}/messages entry shape while the response is still running.
type TranscriptStreamEvent struct {
	Type       string               `json:"type"` // "assistant_delta" or "message"
	XSessionID string               `json:"x_session_id,omitempty"`
	RunID      string               `json:"runId,omitempty"`
	Timestamp  string               `json:"timestamp,omitempty"`
	Message    *SessionMessageEntry `json:"message,omitempty"`
}

// SessionRunEventEntry is the Web/API view of a run lifecycle event.
type SessionRunEventEntry struct {
	Seq       int64          `json:"seq,omitempty"`
	ID        string         `json:"id"`
	SessionID string         `json:"sessionId"`
	RunID     string         `json:"runId"`
	EventType string         `json:"eventType"`
	Source    string         `json:"source,omitempty"`
	Status    string         `json:"status,omitempty"`
	Model     string         `json:"model,omitempty"`
	Mode      string         `json:"mode,omitempty"`
	Timestamp string         `json:"timestamp"`
	Data      map[string]any `json:"data,omitempty"`
}

// SessionCapabilityEventEntry is the Web/API view of a capability transition.
type SessionCapabilityEventEntry struct {
	Seq        int64          `json:"seq,omitempty"`
	ID         string         `json:"id"`
	SessionID  string         `json:"sessionId"`
	RunID      string         `json:"runId,omitempty"`
	EventType  string         `json:"eventType"`
	Source     string         `json:"source,omitempty"`
	Actor      string         `json:"actor,omitempty"`
	Capability string         `json:"capability"`
	OldValue   string         `json:"oldValue"`
	NewValue   string         `json:"newValue"`
	Timestamp  string         `json:"timestamp"`
	Data       map[string]any `json:"data,omitempty"`
}

// --- Model list types ---

// ModelListResponse is the response for GET /v1/models.
type ModelListResponse struct {
	Object string      `json:"object"`
	Data   []ModelItem `json:"data"`
}

// ModelItem represents one model in the list.
type ModelItem struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	OwnedBy string   `json:"owned_by"`
	Input   []string `json:"input,omitempty"`
}

// --- Health ---

// HealthResponse is the response for GET /health.
type HealthResponse struct {
	Status   string `json:"status"`
	Version  string `json:"version"`
	Sessions int    `json:"sessions"`
}

// --- Error response ---

// ErrorResponse is the standard OpenAI error format.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information.
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// --- Helpers ---

func newCompletionID() string {
	return fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
}

func newCommandCompletionID() string {
	return fmt.Sprintf("chatcmpl-cmd-%d", time.Now().UnixNano())
}

func stringPtr(s string) *string {
	return &s
}

func marshalJSON(v any) []byte {
	data, _ := json.Marshal(v)
	return data
}
