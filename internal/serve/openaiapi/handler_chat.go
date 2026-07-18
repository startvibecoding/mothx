package openaiapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	agentpkg "github.com/startvibecoding/mothx/agent"
	"github.com/startvibecoding/mothx/internal/a2a"
	"github.com/startvibecoding/mothx/internal/agent"
	browserfeature "github.com/startvibecoding/mothx/internal/browser"
	"github.com/startvibecoding/mothx/internal/config"
	ctxpkg "github.com/startvibecoding/mothx/internal/context"
	"github.com/startvibecoding/mothx/internal/contextfiles"
	"github.com/startvibecoding/mothx/internal/cron"
	"github.com/startvibecoding/mothx/internal/provider"
	"github.com/startvibecoding/mothx/internal/sandbox"
	"github.com/startvibecoding/mothx/internal/session"
	"github.com/startvibecoding/mothx/internal/skills"
	"github.com/startvibecoding/mothx/internal/tools"
	"github.com/startvibecoding/mothx/internal/util"
	"github.com/startvibecoding/mothx/internal/workflow"
)

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10MB limit
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body", "invalid_request_error")
		return
	}
	defer r.Body.Close()

	var req ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error(), "invalid_request_error")
		return
	}

	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages array is required and must not be empty", "invalid_request_error")
		return
	}

	// Validate x_working_dir
	workDir := s.cfg.GetWorkDir()
	if req.XWorkingDir != "" {
		if err := s.cfg.ValidateWorkDir(req.XWorkingDir); err != nil {
			writeError(w, http.StatusForbidden, err.Error(), "permission_error")
			return
		}
		workDir = req.XWorkingDir
	}

	// Resolve model
	s.mu.RLock()
	currentModel := s.model
	currentProvider := s.provider
	s.mu.RUnlock()

	if req.Model != "" {
		if m := currentProvider.GetModel(req.Model); m != nil {
			currentModel = m
		} else {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("model %q not found — available: %s", req.Model, modelIDs(currentProvider.Models())), "invalid_request_error")
			return
		}
	}
	currentModel = cloneModel(currentModel)

	// Extract last user message
	lastUserMsg, systemMsgs, historyMsgs := parseMessages(req.Messages)
	if strings.TrimSpace(lastUserMsg.Content) == "" && len(lastUserMsg.ContentParts) == 0 {
		writeError(w, http.StatusBadRequest, "no user message found", "invalid_request_error")
		return
	}
	lastUserMessage, err := buildUserMessage(lastUserMsg)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "invalid_request_error")
		return
	}
	if messageHasImage(lastUserMessage) && !modelSupportsInput(currentModel, "image") {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("model %q does not support image input", currentModel.ID), "invalid_request_error")
		return
	}
	if req.XSessionID != "" {
		sessionWorkDir, found, err := s.findSessionWorkDir(req.XSessionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error(), "server_error")
			return
		}
		if found && !sameWorkDir(sessionWorkDir, s.cfg.GetWorkDir()) {
			if err := s.cfg.ValidateWorkDir(sessionWorkDir); err != nil {
				writeError(w, http.StatusForbidden, err.Error(), "permission_error")
				return
			}
		}
		if found && req.XWorkingDir != "" && !sameWorkDir(sessionWorkDir, workDir) {
			writeError(w, http.StatusConflict, fmt.Sprintf("x_working_dir %q does not match session %q workDir %q", workDir, req.XSessionID, sessionWorkDir), "conflict_error")
			return
		}
	}

	// Get or create session
	sessionID := req.XSessionID
	if sessionID == "" {
		// Fall back to the default session for this workDir.
		s.mu.RLock()
		sessionID = s.defaultSessionIDs[workDir]
		s.mu.RUnlock()
	}
	var sess *APISession
	for {
		sess, err = s.getOrCreateSession(sessionID, workDir)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error(), "server_error")
			return
		}
		if sess == nil {
			writeError(w, http.StatusServiceUnavailable, "session pool is at capacity", "server_error")
			return
		}
		if s.pool.Pin(sess) {
			break
		}
	}
	defer s.pool.Unpin(sess)

	sess.Lock()
	defer sess.Unlock()
	sess.Touch()
	runID := newRunID()
	sess.approvalMu.Lock()
	sess.activeRunID = runID
	sess.approvalMu.Unlock()
	sess.SetRunning(true)
	defer func() {
		s.clearSessionApprovals(sess, "cancelled", "run ended before the approval was resolved")
		sess.approvalMu.Lock()
		sess.activeRunID = ""
		sess.approvalMu.Unlock()
		sess.SetRunning(false)
		s.publishSessionStreamDone(sess.ID)
	}()
	if err := s.applySessionToolOptions(sess, req.XTools, runID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "server_error")
		return
	}

	mode := s.cfg.DefaultMode
	if sess.Mode != "" {
		mode = sess.Mode
	}
	if req.XMode != "" {
		mode = strings.TrimSpace(req.XMode)
		if err := validateCapabilityMode(mode); err != nil {
			writeError(w, http.StatusBadRequest, err.Error(), "invalid_request_error")
			return
		}
		// x_mode establishes the selected WebUI runtime mode before the first
		// agent is constructed. Mode is not a tool capability, so this does not
		// synchronize or mutate the session tool registry.
		before := capabilitySnapshotFromSession(sess)
		sess.Mode = mode
		if err := s.persistSessionCapabilitiesWithEvents(sess, before, "x_mode", "webui", runID, map[string]any{
			"source": "chat_completion",
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error(), "server_error")
			return
		}
	}
	command := ""
	if fields := strings.Fields(strings.TrimSpace(lastUserMsg.Content)); len(fields) > 0 && strings.HasPrefix(fields[0], "/") {
		command = fields[0]
	}
	if err := s.recordSessionRunEvent(sess, runID, "started", "running", "chat_completion", currentModel.ID, mode, map[string]any{
		"stream":       req.Stream,
		"transcript":   req.XTranscript,
		"workDir":      sess.WorkDir,
		"provider":     s.providerName,
		"command":      command,
		"messageCount": len(req.Messages),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "server_error")
		return
	}

	// Check for slash command
	if cmdResult := s.handleCommand(sess, lastUserMsg.Content, runID); cmdResult != nil {
		// If /clear, we need to reset agent state on the session
		if strings.HasPrefix(strings.TrimSpace(lastUserMsg.Content), "/clear") {
			if err := s.clearSession(sess, workDir); err != nil {
				_ = s.recordSessionRunEvent(sess, runID, "failed", "failed", "chat_completion", currentModel.ID, mode, map[string]any{
					"command": command,
					"error":   err.Error(),
				})
				writeError(w, http.StatusInternalServerError, err.Error(), "server_error")
				return
			}
		}
		status := "completed"
		eventType := "finished"
		if cmdResult.Error {
			status = "failed"
			eventType = "failed"
		}
		if err := s.recordSessionRunEvent(sess, runID, eventType, status, "chat_completion", currentModel.ID, mode, map[string]any{
			"command": command,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error(), "server_error")
			return
		}
		if req.Stream {
			s.writeCommandResponseStreaming(w, cmdResult, currentModel.ID, sess.ID, lastUserMsg.Content, req.XTranscript)
		} else {
			s.writeCommandResponse(w, cmdResult, currentModel.ID, sess.ID, lastUserMsg.Content)
		}
		return
	}

	// Build extra context: system prompt handling
	extraContext := sess.ExtraContext
	if extraContext == "" {
		extraContext = s.extraContext
	}
	ruleContent := sess.RuleContent
	if s.cfg.SystemPromptMode == "append" && len(systemMsgs) > 0 {
		extraContext += "\n## Client Instructions\n" + strings.Join(systemMsgs, "\n") + "\n"
	}

	runtimeSettings := s.settingsForSession(sess)

	// Build compaction settings
	compactionSettings := ctxpkg.CompactionSettings{
		Enabled:          runtimeSettings.Compaction.Enabled,
		ReserveTokens:    runtimeSettings.Compaction.ReserveTokens,
		KeepRecentTokens: runtimeSettings.Compaction.KeepRecentTokens,
		Tokenizer:        runtimeSettings.Compaction.Tokenizer,
		TokenizerModel:   runtimeSettings.Compaction.TokenizerModel,
		Template:         runtimeSettings.Compaction.Template,
	}
	if compactionSettings.ReserveTokens == 0 {
		compactionSettings.ReserveTokens = 16384
	}
	if compactionSettings.KeepRecentTokens == 0 {
		compactionSettings.KeepRecentTokens = 20000
	}

	// Build agent config
	thinkingLevel := provider.ThinkingLevel(s.cfg.DefaultThinkingLevel)
	if thinkingLevel == "" {
		thinkingLevel = provider.ThinkingLevel(s.settings.DefaultThinkingLevel)
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = agent.ResolveMaxTokens(currentModel)
	}

	// Per-request temperature/top_p override (from OpenAI-compatible client)
	if req.Temperature != nil {
		currentModel.Temperature = req.Temperature
	}
	if req.TopP != nil {
		currentModel.TopP = req.TopP
	}

	// applySessionToolOptions calls syncSessionTools before this point. Tool
	// registration is therefore owned by the session runtime/capability layer,
	// not by mode selection or individual requests. agent.New snapshots this
	// already-synchronized registry below.

	agentCfg := agent.Config{
		Provider:           currentProvider,
		Vendor:             s.providerName,
		Model:              currentModel,
		Mode:               mode,
		ThinkingLevel:      thinkingLevel,
		MaxTokens:          maxTokens,
		SandboxMgr:         s.sandboxMgr,
		Settings:           runtimeSettings,
		Allow:              s.getAllow(),
		Session:            sess.Manager,
		ExtraContext:       extraContext,
		RuleContent:        ruleContent,
		CompactionSettings: compactionSettings,
		MultiAgent:         sess.MultiAgent,
		DelegateMode:       sess.DelegateMode,
		Workflows:          sess.Workflows,
	}

	a := agent.New(agentCfg, sess.Registry)

	// Apply force compact flag from /compact command
	if sess.ForceCompact {
		a.SetForceCompact()
		sess.ForceCompact = false
	}

	replayState := sess.Manager.GetReplayState()
	if len(replayState.Messages) > 0 {
		a.LoadHistoryState(replayState.Messages, replayState.EntryIDs)
	} else if len(historyMsgs) > 0 {
		// Seed brand-new sessions from client-provided history.
		internalMsgs := convertHistoryMessages(historyMsgs)
		a.LoadHistoryMessages(internalMsgs)
	}

	// Setup request timeout
	timeout := time.Duration(s.cfg.RequestTimeoutSecs) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	if (sess.MultiAgent || sess.DelegateMode || sess.Workflows) && sess.AgentMgr != nil {
		sess.AgentMgr.Register(agent.NewAgentAdapter(a))
		defer func() {
			sess.AgentMgr.Finish(a.ID(), ctx.Err())
		}()
	}

	// Run agent
	eventCh := a.RunWithUserMessage(ctx, lastUserMessage)

	if req.Stream {
		usage, status, errMsg := s.handleStreamingResponseWithAgent(w, r, eventCh, currentModel.ID, sess, a, req.XTranscript)
		_ = s.recordSessionRunEvent(sess, runID, runEventTypeForStatus(status), status, "chat_completion", currentModel.ID, mode, usageEventData(usage, errMsg))
	} else {
		usage, status, errMsg := s.handleNonStreamingResponseWithAgent(w, eventCh, currentModel.ID, sess, a)
		_ = s.recordSessionRunEvent(sess, runID, runEventTypeForStatus(status), status, "chat_completion", currentModel.ID, mode, usageEventData(usage, errMsg))
	}
}

func cloneModel(model *provider.Model) *provider.Model {
	if model == nil {
		return nil
	}
	copy := *model
	copy.Input = append([]string(nil), model.Input...)
	if model.Compat != nil {
		compat := *model.Compat
		copy.Compat = &compat
	}
	return &copy
}

func sameWorkDir(a, b string) bool {
	if a == "" || b == "" {
		return a == b
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func (s *Server) handleStreamingResponse(w http.ResponseWriter, r *http.Request, eventCh <-chan agent.Event, modelID, sessionID string, transcript bool) (CompletionUsage, string, string) {
	return s.handleStreamingResponseWithAgent(w, r, eventCh, modelID, &APISession{ID: sessionID}, nil, transcript)
}

func (s *Server) handleStreamingResponseWithAgent(w http.ResponseWriter, r *http.Request, eventCh <-chan agent.Event, modelID string, sess *APISession, runningAgent *agent.Agent, transcript bool) (CompletionUsage, string, string) {
	sessionID := sess.ID
	sse := NewSSEWriter(w, modelID, sessionID)
	sse.WriteRoleDelta()

	toolMode := s.cfg.ToolVisibility.Mode
	toolDetail := s.cfg.GetToolDetail()
	var totalUsage CompletionUsage
	var xToolCalls []XToolCall
	// Track in-flight tool calls by callID so we can attach result/diff on end.
	pendingTools := make(map[string]*toolCallInfo)

	for ev := range eventCh {
		select {
		case <-r.Context().Done():
			return totalUsage, "canceled", r.Context().Err().Error()
		default:
		}

		switch ev.Type {
		case agent.EventTextDelta:
			if transcript {
				s.writeTranscriptEvent(sse, sessionID, assistantDeltaTranscriptEvent(ev.TextDelta, ev.AgentID))
			}
			if ev.AgentID == "" {
				sse.WriteContentDelta(ev.TextDelta)
			}

		case agent.EventToolCall:
			name, callID := resolveToolEvent(ev)
			tc := &toolCallInfo{Name: name, Args: ev.ToolArgs, Status: "running"}
			if callID != "" {
				pendingTools[callID] = tc
			}
			xToolCalls = append(xToolCalls, XToolCall{Name: name, Args: ev.ToolArgs, Status: "running"})
			s.publishToolEvent(sessionID, ToolStatusEvent{Tool: name, ToolCallID: callID, AgentID: string(ev.AgentID), Status: "running", Args: ev.ToolArgs})
			if transcript {
				s.writeTranscriptEvent(sse, sessionID, messageTranscriptEvent(transcriptToolCallEntry(name, callID, ev)))
			} else {
				switch toolMode {
				case "content":
					sse.WriteContentDelta(formatToolRunning(name, ev.ToolArgs))
				case "sse_event":
					sse.WriteToolStatusEvent(ToolStatusEvent{
						Tool:       name,
						ToolCallID: callID,
						AgentID:    string(ev.AgentID),
						Status:     "running",
						Args:       ev.ToolArgs,
					})
				}
			}

		case agent.EventToolExecutionEnd:
			status := "completed"
			if ev.ToolError != nil {
				status = "failed"
			}
			// Update xToolCalls status
			for i := len(xToolCalls) - 1; i >= 0; i-- {
				if xToolCalls[i].Name == ev.ToolName && xToolCalls[i].Status == "running" {
					xToolCalls[i].Status = status
					break
				}
			}
			// Build expanded output
			tc := pendingTools[ev.ToolCallID]
			if tc == nil {
				tc = &toolCallInfo{Name: ev.ToolName, Args: ev.ToolArgs}
			}
			tc.Status = status
			tc.Result = ev.ToolResult
			tc.Diff = ev.ToolDiff
			tc.Error = ev.ToolError
			delete(pendingTools, ev.ToolCallID)
			name := ev.ToolName
			if name == "" {
				name = tc.Name
			}
			s.publishToolEvent(sessionID, ToolStatusEvent{
				Tool: name, ToolCallID: ev.ToolCallID, AgentID: string(ev.AgentID), Status: status,
				Args: tc.Args, Summary: summarizeToolStatusResult(ev.ToolResult), IsError: ev.ToolError != nil, HasDetail: ev.ToolCallID != "",
			})

			if transcript {
				s.writeTranscriptEvent(sse, sessionID, messageTranscriptEvent(transcriptToolResultEntry(name, ev, status)))
			} else {
				switch toolMode {
				case "content":
					sse.WriteToolResult(tc, toolDetail)
				case "sse_event":
					sse.WriteToolStatusEvent(ToolStatusEvent{
						Tool:       name,
						ToolCallID: ev.ToolCallID,
						AgentID:    string(ev.AgentID),
						Status:     status,
						Args:       tc.Args,
						Summary:    summarizeToolStatusResult(ev.ToolResult),
						IsError:    ev.ToolError != nil,
						HasDetail:  ev.ToolCallID != "",
					})
				}
			}

		case agent.EventToolApprovalRequest:
			if request := s.registerSessionApproval(sess, runningAgent, ev); request != nil {
				sse.WriteApprovalRequest(*request)
			}

		case agent.EventUsage:
			if ev.Usage != nil {
				totalUsage.PromptTokens += ev.Usage.TotalInputTokens()
				totalUsage.CompletionTokens += ev.Usage.Output
				totalUsage.CacheReadTokens += ev.Usage.CacheRead
				totalUsage.CacheWriteTokens += ev.Usage.CacheWrite
				totalUsage.TotalTokens = totalUsage.PromptTokens + totalUsage.CompletionTokens
			}

		case agent.EventDone:
			if ev.AgentID != "" {
				if transcript {
					s.writeTranscriptEvent(sse, sessionID, subAgentStatusTranscriptEvent(ev.AgentID, "done", ""))
				}
				continue
			}
			sse.WriteDone(&totalUsage)
			return totalUsage, "completed", ""

		case agent.EventError:
			if ev.AgentID != "" {
				if transcript && ev.Error != nil {
					s.writeTranscriptEvent(sse, sessionID, subAgentStatusTranscriptEvent(ev.AgentID, "error", ev.Error.Error()))
				}
				continue
			}
			if ev.Error != nil {
				if transcript {
					s.writeTranscriptEvent(sse, sessionID, assistantDeltaTranscriptEvent("\n\n[Error: "+ev.Error.Error()+"]", ""))
				}
				sse.WriteError(ev.Error.Error())
				return totalUsage, "failed", ev.Error.Error()
			} else {
				sse.WriteDone(&totalUsage)
				return totalUsage, "completed", ""
			}
		}
	}
	// Channel closed without EventDone
	sse.WriteDone(&totalUsage)
	return totalUsage, "completed", ""
}

func assistantDeltaTranscriptEvent(text string, agentID agentpkg.AgentID) TranscriptStreamEvent {
	return TranscriptStreamEvent{
		Type: "assistant_delta",
		Message: &SessionMessageEntry{
			AgentID: string(agentID),
			Role:    "assistant",
			Content: text,
		},
	}
}

func messageTranscriptEvent(entry SessionMessageEntry) TranscriptStreamEvent {
	return TranscriptStreamEvent{
		Type:    "message",
		Message: &entry,
	}
}

func subAgentStatusTranscriptEvent(agentID agentpkg.AgentID, status string, summary string) TranscriptStreamEvent {
	return TranscriptStreamEvent{
		Type: "subagent_status",
		Message: &SessionMessageEntry{
			AgentID: string(agentID),
			Role:    "status",
			Content: status,
			Summary: summary,
			IsError: status == "error",
		},
	}
}

func transcriptToolCallEntry(name, callID string, ev agent.Event) SessionMessageEntry {
	args := rawToolArgs(ev.ToolArgs)
	invalidArgs := ""
	if ev.ToolCall != nil {
		if ev.ToolCall.Name != "" {
			name = ev.ToolCall.Name
		}
		if ev.ToolCall.ID != "" {
			callID = ev.ToolCall.ID
		}
		if len(ev.ToolCall.Arguments) > 0 {
			args = validRawMessage(ev.ToolCall.Arguments)
		}
		invalidArgs = ev.ToolCall.InvalidArguments
	}
	return SessionMessageEntry{
		Role:        "toolCall",
		AgentID:     string(ev.AgentID),
		ToolCallID:  callID,
		ToolName:    name,
		Arguments:   args,
		InvalidArgs: invalidArgs,
		Plan:        planFromToolCall(name, args),
	}
}

func transcriptToolResultEntry(name string, ev agent.Event, status string) SessionMessageEntry {
	isError := status == "failed" || ev.ToolError != nil
	summary := summarizeToolStatusResult(ev.ToolResult)
	if isError && strings.TrimSpace(ev.ToolResult) == "" && ev.ToolError != nil {
		summary = ev.ToolError.Error()
	}
	return SessionMessageEntry{
		Role:       "toolResult",
		AgentID:    string(ev.AgentID),
		ToolCallID: ev.ToolCallID,
		ToolName:   name,
		IsError:    isError,
		Summary:    summary,
		HasDetail:  ev.ToolCallID != "",
	}
}

func rawToolArgs(args map[string]any) json.RawMessage {
	if len(args) == 0 {
		return nil
	}
	data, err := json.Marshal(args)
	if err != nil || !json.Valid(data) {
		return nil
	}
	return data
}

func (s *Server) handleNonStreamingResponse(w http.ResponseWriter, eventCh <-chan agent.Event, modelID, sessionID string) (CompletionUsage, string, string) {
	return s.handleNonStreamingResponseWithAgent(w, eventCh, modelID, &APISession{ID: sessionID}, nil)
}

func (s *Server) handleNonStreamingResponseWithAgent(w http.ResponseWriter, eventCh <-chan agent.Event, modelID string, sess *APISession, runningAgent *agent.Agent) (CompletionUsage, string, string) {
	sessionID := sess.ID
	var sb strings.Builder
	var totalUsage CompletionUsage
	var xToolCalls []XToolCall
	toolMode := s.cfg.ToolVisibility.Mode
	toolDetail := s.cfg.GetToolDetail()
	pendingTools := make(map[string]*toolCallInfo)

	for ev := range eventCh {
		switch ev.Type {
		case agent.EventTextDelta:
			if ev.AgentID == "" {
				sb.WriteString(ev.TextDelta)
			}

		case agent.EventToolCall:
			name, callID := resolveToolEvent(ev)
			tc := &toolCallInfo{Name: name, Args: ev.ToolArgs, Status: "running"}
			if callID != "" {
				pendingTools[callID] = tc
			}
			xToolCalls = append(xToolCalls, XToolCall{Name: name, Args: ev.ToolArgs, Status: "running"})
			s.publishToolEvent(sessionID, ToolStatusEvent{Tool: name, ToolCallID: callID, AgentID: string(ev.AgentID), Status: "running", Args: ev.ToolArgs})

		case agent.EventToolExecutionEnd:
			status := "completed"
			if ev.ToolError != nil {
				status = "failed"
			}
			for i := len(xToolCalls) - 1; i >= 0; i-- {
				if xToolCalls[i].Name == ev.ToolName && xToolCalls[i].Status == "running" {
					xToolCalls[i].Status = status
					break
				}
			}
			// Build expanded output for content/none mode
			tc := pendingTools[ev.ToolCallID]
			if tc == nil {
				tc = &toolCallInfo{Name: ev.ToolName, Args: ev.ToolArgs}
			}
			tc.Status = status
			tc.Result = ev.ToolResult
			tc.Diff = ev.ToolDiff
			tc.Error = ev.ToolError
			delete(pendingTools, ev.ToolCallID)
			name := ev.ToolName
			if name == "" {
				name = tc.Name
			}
			s.publishToolEvent(sessionID, ToolStatusEvent{
				Tool: name, ToolCallID: ev.ToolCallID, AgentID: string(ev.AgentID), Status: status,
				Args: tc.Args, Summary: summarizeToolStatusResult(ev.ToolResult), IsError: ev.ToolError != nil, HasDetail: ev.ToolCallID != "",
			})

			if toolMode == "content" && ev.AgentID == "" {
				sb.WriteString(formatToolResult(tc, toolDetail))
			}

		case agent.EventToolApprovalRequest:
			s.registerSessionApproval(sess, runningAgent, ev)

		case agent.EventUsage:
			if ev.Usage != nil {
				totalUsage.PromptTokens += ev.Usage.TotalInputTokens()
				totalUsage.CompletionTokens += ev.Usage.Output
				totalUsage.CacheReadTokens += ev.Usage.CacheRead
				totalUsage.CacheWriteTokens += ev.Usage.CacheWrite
				totalUsage.TotalTokens = totalUsage.PromptTokens + totalUsage.CompletionTokens
			}

		case agent.EventError:
			if ev.AgentID != "" {
				continue
			}
			if ev.Error != nil {
				writeError(w, http.StatusInternalServerError, ev.Error.Error(), "server_error")
				return totalUsage, "failed", ev.Error.Error()
			}
		}
	}

	finishReason := "stop"
	resp := ChatCompletionResponse{
		ID:      newCompletionID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelID,
		Choices: []ChatCompletionChoice{
			{
				Index:        0,
				Message:      &ResponseMessage{Role: "assistant", Content: sb.String()},
				FinishReason: &finishReason,
			},
		},
		Usage:      &totalUsage,
		XSessionID: sessionID,
		XToolCalls: xToolCalls,
	}
	writeJSON(w, http.StatusOK, resp)
	return totalUsage, "completed", ""
}

func summarizeToolStatusResult(result string) string {
	text := strings.TrimSpace(result)
	if text == "" {
		return "(empty result)"
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		text = text[:idx]
	}
	return util.TruncateWithSuffix(text, 140, "...")
}

func (s *Server) writeCommandResponse(w http.ResponseWriter, result *CommandResult, modelID, sessionID, cmd string) {
	finishReason := "stop"
	resp := ChatCompletionResponse{
		ID:      newCommandCompletionID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelID,
		Choices: []ChatCompletionChoice{
			{
				Index:        0,
				Message:      &ResponseMessage{Role: "assistant", Content: result.Message},
				FinishReason: &finishReason,
			},
		},
		Usage:      &CompletionUsage{},
		XSessionID: sessionID,
		XCommand:   strings.Fields(cmd)[0],
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) writeCommandResponseStreaming(w http.ResponseWriter, result *CommandResult, modelID, sessionID, cmd string, transcript bool) {
	sse := NewSSEWriter(w, modelID, sessionID)
	sse.WriteRoleDelta()
	if transcript {
		s.writeTranscriptEvent(sse, sessionID, assistantDeltaTranscriptEvent(result.Message, ""))
	}
	sse.WriteContentDelta(result.Message)
	sse.WriteDone(&CompletionUsage{})
}

// getOrCreateSession returns an existing session or creates a new one.
func (s *Server) getOrCreateSession(sessionID, workDir string) (*APISession, error) {
	if sessionID != "" {
		if sess := s.pool.Get(sessionID); sess != nil {
			if err := s.validatePersistedSessionWorkDir(sess.WorkDir); err != nil {
				return nil, err
			}
			return sess, nil
		}
	}

	// Serialize creation so concurrent requests don't create duplicate sessions
	// for the same ID or for the shared empty x_session_id path.
	s.sessionCreateMu.Lock()
	defer s.sessionCreateMu.Unlock()

	if sessionID != "" {
		if sess := s.pool.Get(sessionID); sess != nil {
			if err := s.validatePersistedSessionWorkDir(sess.WorkDir); err != nil {
				return nil, err
			}
			return sess, nil
		}
		if sess, err := session.OpenByIDExact(s.settings.GetSessionDir(), sessionID); err == nil {
			sessWorkDir := workDir
			if sess.GetHeader() != nil && sess.GetHeader().Cwd != "" {
				sessWorkDir = sess.GetHeader().Cwd
			}
			if err := s.validatePersistedSessionWorkDir(sessWorkDir); err != nil {
				return nil, err
			}
			resources, err := s.buildSessionResources(sessWorkDir)
			if err != nil {
				return nil, err
			}
			gwSess := &APISession{
				ID:           sessionID,
				WorkDir:      sessWorkDir,
				Manager:      sess,
				Registry:     resources.registry,
				SkillsMgr:    resources.skillsMgr,
				ExtraContext: resources.extraContext,
				RuleContent:  resources.ruleContent,
				DelegateMode: s.cfg.EnableDelegate,
				Workflows:    s.cfg.EnableWorkflows,
				WebSearch:    s.cfg.EnableWebSearch,
				Browser:      s.cfg.EnableBrowser,
				A2AMaster:    s.cfg.EnableA2AMaster,
				MultiAgent:   s.cfg.EnableSubAgents,
				LastUsed:     time.Now(),
			}
			if err := s.applyStoredSessionCapabilities(gwSess); err != nil {
				return nil, err
			}
			if gwSess.MultiAgent || gwSess.DelegateMode || gwSess.Workflows {
				gwSess.AgentMgr = s.newAgentManagerForSession(gwSess)
			}
			s.registerCronTool(gwSess)
			if err := s.pool.Put(gwSess); err != nil {
				return nil, err
			}
			return gwSess, nil
		}
	} else {
		s.mu.RLock()
		defaultID := s.defaultSessionIDs[workDir]
		s.mu.RUnlock()
		if defaultID != "" {
			if sess := s.pool.GetForWorkDir(workDir, defaultID); sess != nil {
				return sess, nil
			}
			if sess, err := session.OpenByIDExact(s.settings.GetSessionDir(), defaultID); err == nil {
				sessWorkDir := workDir
				if sess.GetHeader() != nil && sess.GetHeader().Cwd != "" {
					sessWorkDir = sess.GetHeader().Cwd
				}
				if err := s.validatePersistedSessionWorkDir(sessWorkDir); err != nil {
					return nil, err
				}
				resources, err := s.buildSessionResources(sessWorkDir)
				if err != nil {
					return nil, err
				}
				gwSess := &APISession{
					ID:           defaultID,
					WorkDir:      sessWorkDir,
					Manager:      sess,
					Registry:     resources.registry,
					SkillsMgr:    resources.skillsMgr,
					ExtraContext: resources.extraContext,
					RuleContent:  resources.ruleContent,
					DelegateMode: s.cfg.EnableDelegate,
					Workflows:    s.cfg.EnableWorkflows,
					WebSearch:    s.cfg.EnableWebSearch,
					Browser:      s.cfg.EnableBrowser,
					A2AMaster:    s.cfg.EnableA2AMaster,
					MultiAgent:   s.cfg.EnableSubAgents,
					LastUsed:     time.Now(),
				}
				if err := s.applyStoredSessionCapabilities(gwSess); err != nil {
					return nil, err
				}
				if gwSess.MultiAgent || gwSess.DelegateMode || gwSess.Workflows {
					gwSess.AgentMgr = s.newAgentManagerForSession(gwSess)
				}
				s.registerCronTool(gwSess)
				if err := s.pool.Put(gwSess); err != nil {
					return nil, err
				}
				return gwSess, nil
			}
			sessionID = defaultID
		}
	}

	// Create new session
	mgr := session.New(workDir, s.settings.GetSessionDir())
	if sessionID != "" {
		if err := mgr.InitWithID(sessionID); err != nil {
			return nil, fmt.Errorf("initialize session %q: %w", sessionID, err)
		}
	} else {
		if err := mgr.Init(); err != nil {
			return nil, fmt.Errorf("initialize session: %w", err)
		}
	}

	id := sessionID
	if id == "" && mgr.GetHeader() != nil {
		id = mgr.GetHeader().ID
	}

	resources, err := s.buildSessionResources(workDir)
	if err != nil {
		return nil, err
	}

	sess := &APISession{
		ID:           id,
		WorkDir:      workDir,
		Manager:      mgr,
		Registry:     resources.registry,
		SandboxMgr:   resources.sandboxMgr,
		Mode:         "",
		SkillsMgr:    resources.skillsMgr,
		ExtraContext: resources.extraContext,
		RuleContent:  resources.ruleContent,
		DelegateMode: s.cfg.EnableDelegate,
		Workflows:    s.cfg.EnableWorkflows,
		WebSearch:    s.cfg.EnableWebSearch,
		Browser:      s.cfg.EnableBrowser,
		A2AMaster:    s.cfg.EnableA2AMaster,
		MultiAgent:   s.cfg.EnableSubAgents,
		LastUsed:     time.Now(),
	}
	if err := s.applyStoredSessionCapabilities(sess); err != nil {
		return nil, err
	}

	// Create agent manager if sub-agent, delegate, or workflow mode is enabled.
	if sess.MultiAgent || sess.DelegateMode || sess.Workflows {
		sess.AgentMgr = s.newAgentManagerForSession(sess)
	}
	s.registerCronTool(sess)

	if err := s.pool.Put(sess); err != nil {
		return nil, err
	}

	// If this session was created without a client-supplied ID,
	// remember it as the default so subsequent empty x_session_id
	// requests reuse the same session.
	if sessionID == "" {
		s.mu.Lock()
		if s.defaultSessionIDs == nil {
			s.defaultSessionIDs = make(map[string]string)
		}
		if s.defaultSessionIDs[workDir] == "" {
			s.defaultSessionIDs[workDir] = sess.ID
		}
		s.mu.Unlock()
	}

	return sess, nil
}

// validatePersistedSessionWorkDir applies the current policy when restoring a
// session. The configured default remains trusted even when overrides are
// disabled, preserving the documented default-workdir behavior.
func (s *Server) validatePersistedSessionWorkDir(workDir string) error {
	if sameWorkDir(workDir, s.cfg.GetWorkDir()) {
		return nil
	}
	return s.cfg.ValidateWorkDir(workDir)
}

type sessionResources struct {
	registry     *tools.Registry
	sandboxMgr   *sandbox.Manager
	skillsMgr    *skills.Manager
	extraContext string
	ruleContent  string
}

func (s *Server) buildSessionResources(workDir string) (*sessionResources, error) {
	skillsMgr, extraContext, err := buildWorkDirContext(s.settings, workDir, s.cfg.EnableWorkflows, s.cfg.EnableBrowser)
	if err != nil {
		return nil, err
	}

	sbMgr := sandbox.NewManagerWithOptions(workDir, s.settings.Sandbox.Options())
	if err := sbMgr.SetLevel(s.sandboxMgr.GetActive().Level()); err != nil {
		return nil, fmt.Errorf("sandbox for work directory: %w", err)
	}
	registry := tools.NewRegistry(workDir, sbMgr.GetActive())
	registry.RegisterDefaultsWithPlanTool(s.settings.IsPlanToolEnabled())
	if skillsMgr != nil {
		registry.Register(tools.NewSkillRefTool(skillsMgr))
	}
	if s.cfg.EnableBrowser {
		browserfeature.RegisterTool(registry)
	}
	if err := s.registerA2AMasterTool(registry); err != nil {
		return nil, err
	}

	return &sessionResources{
		registry:     registry,
		skillsMgr:    skillsMgr,
		extraContext: extraContext,
		ruleContent:  contextfiles.LoadRuleFile(workDir),
	}, nil
}

func (s *Server) applySessionToolOptions(sess *APISession, opts *SessionToolOptions, runID string) error {
	if sess == nil {
		return nil
	}
	before := capabilitySnapshotFromSession(sess)
	browserChanged := false
	workflowsChanged := false
	if opts != nil {
		applyBoolOption(&sess.WebSearch, opts.WebSearch)
		browserChanged = applyBoolOption(&sess.Browser, opts.Browser)
		applyBoolOption(&sess.A2AMaster, opts.A2AMaster)
		applyBoolOption(&sess.DelegateMode, opts.Delegate)
		applyBoolOption(&sess.MultiAgent, opts.MultiAgent)
		workflowsChanged = applyBoolOption(&sess.Workflows, opts.Workflows)
	}
	if err := s.syncSessionTools(sess, browserChanged || workflowsChanged); err != nil {
		return err
	}
	if opts != nil {
		return s.persistSessionCapabilitiesWithEvents(sess, before, "x_tools", "webui", runID, map[string]any{
			"source": "chat_completion",
		})
	}
	return nil
}

func applyBoolOption(dst *bool, src *bool) bool {
	if src == nil || dst == nil || *dst == *src {
		return false
	}
	*dst = *src
	return true
}

func (s *Server) syncSessionTools(sess *APISession, refreshContext bool) error {
	if sess == nil || sess.Registry == nil {
		return nil
	}
	s.registerCronTool(sess)

	if refreshContext {
		if err := s.refreshSessionContext(sess); err != nil {
			return err
		}
	}

	if sess.Browser {
		browserfeature.RegisterTool(sess.Registry)
	} else {
		browserfeature.RemoveTool(sess.Registry)
	}

	if sess.A2AMaster {
		if err := s.registerA2ADispatchTool(sess.Registry); err != nil {
			return err
		}
	} else {
		sess.Registry.Remove("a2a_dispatch")
	}

	if sess.MultiAgent || sess.DelegateMode || sess.Workflows {
		if sess.AgentMgr == nil {
			sess.AgentMgr = s.newAgentManagerForSession(sess)
		}
	} else {
		sess.AgentMgr = nil
	}

	if sess.MultiAgent && sess.AgentMgr != nil {
		agent.RegisterSubAgentTools(sess.Registry, sess.AgentMgr)
	} else {
		removeSubAgentTools(sess.Registry)
	}

	if sess.DelegateMode && sess.AgentMgr != nil {
		agent.RegisterDelegateSubAgentTool(sess.Registry, sess.AgentMgr)
	} else {
		sess.Registry.Remove("delegate_subagent")
	}
	if sess.Workflows && sess.AgentMgr != nil {
		workflow.RegisterTools(sess.Registry, sess.AgentMgr, nil)
	} else {
		removeWorkflowTools(sess.Registry)
	}

	return nil
}

func (s *Server) registerCronTool(sess *APISession) {
	if sess == nil || sess.Registry == nil {
		return
	}
	if s == nil || s.cronStore == nil {
		sess.Registry.Remove("cron")
		return
	}
	sess.Registry.Register(cron.NewCronTool(cron.NewSessionScopedStoreWithWorkDir(s.cronStore, sess.ID, sess.WorkDir), s.cronScheduler))
}

func removeSubAgentTools(registry *tools.Registry) {
	if registry == nil {
		return
	}
	for _, name := range []string{"subagent_spawn", "subagent_status", "subagent_send", "subagent_destroy"} {
		registry.Remove(name)
	}
}

func removeWorkflowTools(registry *tools.Registry) {
	if registry == nil {
		return
	}
	for _, name := range []string{"workflow_lint", "workflow_run", "workflow_status", "workflow_cancel"} {
		registry.Remove(name)
	}
}

func (s *Server) refreshSessionContext(sess *APISession) error {
	skillsMgr, extraContext, err := buildWorkDirContext(s.settings, sess.WorkDir, sess.Workflows, sess.Browser)
	if err != nil {
		return err
	}
	activeContext, err := buildActiveSkillsContext(skillsMgr, sess.ActiveSkills)
	if err != nil {
		return err
	}
	sess.SkillsMgr = skillsMgr
	sess.ExtraContext = extraContext + activeContext
	sess.RuleContent = contextfiles.LoadRuleFile(sess.WorkDir)
	if sess.Registry != nil && skillsMgr != nil {
		sess.Registry.Register(tools.NewSkillRefTool(skillsMgr))
	}
	if sess.AgentMgr != nil {
		sess.AgentMgr = s.newAgentManagerForSession(sess)
	}
	return nil
}

func (s *Server) settingsForSession(sess *APISession) *config.Settings {
	if s.settings == nil || sess == nil {
		return s.settings
	}
	runtimeSettings := *s.settings
	runtimeSettings.WebSearch.Enabled = config.BoolPtr(sess.WebSearch)
	return &runtimeSettings
}

func (s *Server) registerA2AMasterTool(registry *tools.Registry) error {
	if !s.cfg.EnableA2AMaster {
		return nil
	}
	return s.registerA2ADispatchTool(registry)
}

func (s *Server) registerA2ADispatchTool(registry *tools.Registry) error {
	a2aListPath := a2a.ProjectAgentListConfigPath()
	if _, err := os.Stat(a2aListPath); err != nil {
		a2aListPath = a2a.AgentListConfigPath()
	}
	a2aListCfg, err := a2a.LoadAgentList(a2aListPath)
	if err != nil {
		return fmt.Errorf("load a2a-list.json: %w", err)
	}
	a2aMgr := a2a.NewA2AManager(a2aListCfg)
	registry.Register(tools.NewA2ADispatchTool(&a2aDispatcherAdapter{mgr: a2aMgr}))
	return nil
}

type a2aDispatcherAdapter struct {
	mgr *a2a.A2AManager
}

func (a *a2aDispatcherAdapter) List() []tools.AgentEntry {
	entries := a.mgr.List()
	result := make([]tools.AgentEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, tools.AgentEntry{Name: e.Name, URL: e.URL})
	}
	return result
}

func (a *a2aDispatcherAdapter) Dispatch(ctx context.Context, name, message string) (string, error) {
	return a.mgr.Dispatch(ctx, name, message)
}

func (s *Server) clearSession(sess *APISession, workDir string) error {
	if sess == nil {
		return fmt.Errorf("no active session to clear")
	}
	sessionDir := s.settings.GetSessionDir()
	if sess.Manager == nil {
		return fmt.Errorf("current session is not initialized")
	}
	if sess.Manager.GetHeader() != nil && sess.Manager.GetHeader().Cwd != "" {
		workDir = sess.Manager.GetHeader().Cwd
	}
	if err := session.DeleteSession(sess.Manager.GetFile(), sessionDir); err != nil {
		return fmt.Errorf("delete current session: %w", err)
	}
	newMgr := session.New(workDir, sessionDir)
	if err := newMgr.InitWithID(sess.ID); err != nil {
		return fmt.Errorf("create fresh session: %w", err)
	}
	sess.Manager = newMgr
	sess.WorkDir = workDir
	sess.Touch()
	sess.ForceCompact = false
	s.mu.Lock()
	if s.defaultSessionIDs == nil {
		s.defaultSessionIDs = make(map[string]string)
	}
	s.defaultSessionIDs[workDir] = sess.ID
	s.mu.Unlock()
	return nil
}

// parseMessages extracts the last user message, system messages, and history messages.
func parseMessages(msgs []RequestMessage) (lastUser RequestMessage, systemMsgs []string, history []RequestMessage) {
	for _, m := range msgs {
		switch m.Role {
		case "system":
			systemMsgs = append(systemMsgs, m.Content)
		}
	}

	// Find the last user message
	lastIdx := -1
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			lastIdx = i
			break
		}
	}
	if lastIdx < 0 {
		return RequestMessage{}, systemMsgs, nil
	}
	lastUser = msgs[lastIdx]

	// Everything before the last user message (excluding system) is history
	for i := 0; i < lastIdx; i++ {
		if msgs[i].Role != "system" {
			history = append(history, msgs[i])
		}
	}
	return lastUser, systemMsgs, history
}

func buildUserMessage(m RequestMessage) (provider.Message, error) {
	if len(m.ContentParts) == 0 {
		return provider.NewUserMessage(m.Content), nil
	}
	contents, err := requestContentBlocks(m)
	if err != nil {
		return provider.Message{}, err
	}
	msg := provider.NewUserMessage(m.Content)
	msg.Contents = contents
	return msg, nil
}

func requestContentBlocks(m RequestMessage) ([]provider.ContentBlock, error) {
	contents := make([]provider.ContentBlock, 0, len(m.ContentParts))
	for _, part := range m.ContentParts {
		switch part.Type {
		case "text":
			if part.Text != "" {
				contents = append(contents, provider.ContentBlock{Type: "text", Text: part.Text})
			}
		case "image_url":
			if part.ImageURL == nil || part.ImageURL.URL == "" {
				return nil, fmt.Errorf("image_url content part is missing url")
			}
			image, err := imageFromDataURL(part.ImageURL.URL, part.ImageURL.Detail)
			if err != nil {
				return nil, err
			}
			contents = append(contents, provider.ContentBlock{Type: "image", Image: image})
		case "image":
			if part.Image == nil || part.Image.Data == "" || part.Image.MimeType == "" {
				return nil, fmt.Errorf("image content part is missing data or mimeType")
			}
			if err := validateImagePayload(part.Image.MimeType, part.Image.Data); err != nil {
				return nil, err
			}
			contents = append(contents, provider.ContentBlock{Type: "image", Image: &provider.ImageContent{
				Data:     part.Image.Data,
				MimeType: part.Image.MimeType,
				Detail:   part.Image.Detail,
			}})
		default:
			return nil, fmt.Errorf("unsupported content part type %q", part.Type)
		}
	}
	if len(contents) == 0 && m.Content != "" {
		contents = append(contents, provider.ContentBlock{Type: "text", Text: m.Content})
	}
	return contents, nil
}

func imageFromDataURL(dataURL, detail string) (*provider.ImageContent, error) {
	const marker = ";base64,"
	if !strings.HasPrefix(dataURL, "data:image/") {
		return nil, fmt.Errorf("image_url must be a data:image URL")
	}
	idx := strings.Index(dataURL, marker)
	if idx < 0 {
		return nil, fmt.Errorf("image_url must contain base64 image data")
	}
	mimeType := dataURL[len("data:"):idx]
	data := dataURL[idx+len(marker):]
	if err := validateImagePayload(mimeType, data); err != nil {
		return nil, err
	}
	return &provider.ImageContent{Data: data, MimeType: mimeType, Detail: detail}, nil
}

func validateImagePayload(mimeType, data string) error {
	switch mimeType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
	default:
		return fmt.Errorf("unsupported image MIME type %q", mimeType)
	}
	if _, err := base64.StdEncoding.DecodeString(data); err != nil {
		return fmt.Errorf("invalid base64 image data: %w", err)
	}
	return nil
}

func messageHasImage(msg provider.Message) bool {
	for _, block := range msg.Contents {
		if block.Type == "image" && block.Image != nil {
			return true
		}
	}
	return false
}

func modelSupportsInput(model *provider.Model, input string) bool {
	if model == nil {
		return false
	}
	for _, item := range model.Input {
		if item == input {
			return true
		}
	}
	return false
}

// convertHistoryMessages converts OpenAI-format history to internal provider.Message.
func convertHistoryMessages(msgs []RequestMessage) []provider.Message {
	result := make([]provider.Message, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "user":
			msg, err := buildUserMessage(m)
			if err == nil {
				result = append(result, msg)
			}
		case "assistant":
			result = append(result, provider.NewAssistantMessage([]provider.ContentBlock{
				{Type: "text", Text: m.Content},
			}))
		}
	}
	return result
}

// resolveToolEvent extracts tool name and call ID from an agent event,
// falling back to ToolCall fields when top-level fields are empty.
func resolveToolEvent(ev agent.Event) (name string, callID string) {
	name = ev.ToolName
	callID = ev.ToolCallID
	if ev.ToolCall != nil {
		if name == "" {
			name = ev.ToolCall.Name
		}
		if callID == "" {
			callID = ev.ToolCall.ID
		}
	}
	return name, callID
}

// modelIDs returns a comma-separated list of model IDs for error messages.
func modelIDs(models []*provider.Model) string {
	ids := make([]string, len(models))
	for i, m := range models {
		ids[i] = m.ID
	}
	return strings.Join(ids, ", ")
}
