package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/startvibecoding/vibecoding/internal/agent"
	ctxpkg "github.com/startvibecoding/vibecoding/internal/context"
	"github.com/startvibecoding/vibecoding/internal/provider"
	"github.com/startvibecoding/vibecoding/internal/session"
	"github.com/startvibecoding/vibecoding/internal/tools"
	"github.com/startvibecoding/vibecoding/internal/workflow"
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
	if lastUserMsg == "" {
		writeError(w, http.StatusBadRequest, "no user message found", "invalid_request_error")
		return
	}

	// Get or create session
	sessionID := req.XSessionID
	if sessionID == "" {
		// Fall back to the default session for this workDir.
		s.mu.RLock()
		sessionID = s.defaultSessionIDs[workDir]
		s.mu.RUnlock()
	}
	sess, err := s.getOrCreateSession(sessionID, workDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "server_error")
		return
	}
	if sess == nil {
		writeError(w, http.StatusServiceUnavailable, "session pool is at capacity", "server_error")
		return
	}

	sess.Lock()
	defer sess.Unlock()
	sess.Touch()

	// Check for slash command
	if cmdResult := s.handleCommand(sess, lastUserMsg); cmdResult != nil {
		// If /clear, we need to reset agent state on the session
		if strings.HasPrefix(strings.TrimSpace(lastUserMsg), "/clear") {
			if err := s.clearSession(sess, workDir); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error(), "server_error")
				return
			}
		}
		if req.Stream {
			s.writeCommandResponseStreaming(w, cmdResult, currentModel.ID, sess.ID, lastUserMsg)
		} else {
			s.writeCommandResponse(w, cmdResult, currentModel.ID, sess.ID, lastUserMsg)
		}
		return
	}

	// Determine mode
	mode := s.cfg.DefaultMode
	if sess.Mode != "" {
		mode = sess.Mode
	}
	if req.XMode != "" {
		mode = req.XMode
	}

	// Build extra context: system prompt handling
	extraContext := sess.ExtraContext
	if extraContext == "" {
		extraContext = s.extraContext
	}
	if s.cfg.SystemPromptMode == "append" && len(systemMsgs) > 0 {
		extraContext += "\n## Client Instructions\n" + strings.Join(systemMsgs, "\n") + "\n"
	}

	// Build compaction settings
	compactionSettings := ctxpkg.CompactionSettings{
		Enabled:          s.settings.Compaction.Enabled,
		ReserveTokens:    s.settings.Compaction.ReserveTokens,
		KeepRecentTokens: s.settings.Compaction.KeepRecentTokens,
		Tokenizer:        s.settings.Compaction.Tokenizer,
		TokenizerModel:   s.settings.Compaction.TokenizerModel,
		Template:         s.settings.Compaction.Template,
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
		maxTokens = agent.ResolveMaxTokens(s.settings, currentModel)
	}

	// Per-request temperature/top_p override (from OpenAI-compatible client)
	if req.Temperature != nil {
		currentModel.Temperature = req.Temperature
	}
	if req.TopP != nil {
		currentModel.TopP = req.TopP
	}

	// Register sub-agent/delegate tools before agent construction; the agent freezes tools at New().
	if s.cfg.EnableSubAgents && sess.AgentMgr != nil {
		agent.RegisterSubAgentTools(sess.Registry, sess.AgentMgr)
	}
	if sess.DelegateMode && sess.AgentMgr != nil {
		agent.RegisterDelegateSubAgentTool(sess.Registry, sess.AgentMgr)
	} else {
		sess.Registry.Remove("delegate_subagent")
	}
	if sess.Workflows && sess.AgentMgr != nil {
		workflow.RegisterTools(sess.Registry, sess.AgentMgr, nil)
	}

	agentCfg := agent.Config{
		Provider:           currentProvider,
		Vendor:             s.providerName,
		Model:              currentModel,
		Mode:               mode,
		ThinkingLevel:      thinkingLevel,
		MaxTokens:          maxTokens,
		SandboxMgr:         s.sandboxMgr,
		Settings:           s.settings,
		Allow:              s.getAllow(),
		Session:            sess.Manager,
		ExtraContext:       extraContext,
		CompactionSettings: compactionSettings,
		MultiAgent:         s.cfg.EnableSubAgents,
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
	if (s.cfg.EnableSubAgents || sess.DelegateMode || sess.Workflows) && sess.AgentMgr != nil {
		sess.AgentMgr.Register(agent.NewAgentAdapter(a))
		defer func() {
			sess.AgentMgr.Finish(a.ID(), ctx.Err())
		}()
	}

	// Run agent
	eventCh := a.Run(ctx, lastUserMsg)

	if req.Stream {
		s.handleStreamingResponse(w, r, eventCh, currentModel.ID, sess.ID)
	} else {
		s.handleNonStreamingResponse(w, eventCh, currentModel.ID, sess.ID)
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

func (s *Server) handleStreamingResponse(w http.ResponseWriter, r *http.Request, eventCh <-chan agent.Event, modelID, sessionID string) {
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
			return
		default:
		}

		switch ev.Type {
		case agent.EventTextDelta:
			sse.WriteContentDelta(ev.TextDelta)

		case agent.EventToolCall:
			name, callID := resolveToolEvent(ev)
			tc := &toolCallInfo{Name: name, Args: ev.ToolArgs, Status: "running"}
			if callID != "" {
				pendingTools[callID] = tc
			}
			xToolCalls = append(xToolCalls, XToolCall{Name: name, Args: ev.ToolArgs, Status: "running"})
			switch toolMode {
			case "content":
				sse.WriteContentDelta(formatToolRunning(name, ev.ToolArgs))
			case "sse_event":
				sse.WriteToolStatusEvent(name, "running", ev.ToolArgs)
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

			switch toolMode {
			case "content":
				sse.WriteToolResult(tc, toolDetail)
			case "sse_event":
				sse.WriteToolStatusEvent(ev.ToolName, status, nil)
			}

		case agent.EventUsage:
			if ev.Usage != nil {
				totalUsage.PromptTokens += ev.Usage.TotalInputTokens()
				totalUsage.CompletionTokens += ev.Usage.Output
				totalUsage.TotalTokens = totalUsage.PromptTokens + totalUsage.CompletionTokens
			}

		case agent.EventDone:
			sse.WriteDone(&totalUsage)
			return

		case agent.EventError:
			if ev.Error != nil {
				sse.WriteError(ev.Error.Error())
			} else {
				sse.WriteDone(&totalUsage)
			}
			return
		}
	}
	// Channel closed without EventDone
	sse.WriteDone(&totalUsage)
}

func (s *Server) handleNonStreamingResponse(w http.ResponseWriter, eventCh <-chan agent.Event, modelID, sessionID string) {
	var sb strings.Builder
	var totalUsage CompletionUsage
	var xToolCalls []XToolCall
	toolMode := s.cfg.ToolVisibility.Mode
	toolDetail := s.cfg.GetToolDetail()
	pendingTools := make(map[string]*toolCallInfo)

	for ev := range eventCh {
		switch ev.Type {
		case agent.EventTextDelta:
			sb.WriteString(ev.TextDelta)

		case agent.EventToolCall:
			name, callID := resolveToolEvent(ev)
			tc := &toolCallInfo{Name: name, Args: ev.ToolArgs, Status: "running"}
			if callID != "" {
				pendingTools[callID] = tc
			}
			xToolCalls = append(xToolCalls, XToolCall{Name: name, Args: ev.ToolArgs, Status: "running"})

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

			if toolMode == "content" {
				sb.WriteString(formatToolResult(tc, toolDetail))
			}

		case agent.EventUsage:
			if ev.Usage != nil {
				totalUsage.PromptTokens += ev.Usage.TotalInputTokens()
				totalUsage.CompletionTokens += ev.Usage.Output
				totalUsage.TotalTokens = totalUsage.PromptTokens + totalUsage.CompletionTokens
			}

		case agent.EventError:
			if ev.Error != nil {
				writeError(w, http.StatusInternalServerError, ev.Error.Error(), "server_error")
				return
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

func (s *Server) writeCommandResponseStreaming(w http.ResponseWriter, result *CommandResult, modelID, sessionID, cmd string) {
	sse := NewSSEWriter(w, modelID, sessionID)
	sse.WriteRoleDelta()
	sse.WriteContentDelta(result.Message)
	sse.WriteDone(&CompletionUsage{})
}

// getOrCreateSession returns an existing session or creates a new one.
func (s *Server) getOrCreateSession(sessionID, workDir string) (*GatewaySession, error) {
	if sessionID != "" {
		if sess := s.pool.Get(sessionID); sess != nil {
			return sess, nil
		}
	}

	// Serialize creation so concurrent requests don't create duplicate sessions
	// for the same ID or for the shared empty x_session_id path.
	s.sessionCreateMu.Lock()
	defer s.sessionCreateMu.Unlock()

	if sessionID != "" {
		if sess := s.pool.Get(sessionID); sess != nil {
			return sess, nil
		}
		if sess, err := session.OpenByIDExact(s.settings.GetSessionDir(), sessionID); err == nil {
			sessWorkDir := workDir
			if sess.GetHeader() != nil && sess.GetHeader().Cwd != "" {
				sessWorkDir = sess.GetHeader().Cwd
			}
			registry := tools.NewRegistry(sessWorkDir, s.sandboxMgr.GetActive())
			registry.RegisterDefaultsWithPlanTool(s.settings.IsPlanToolEnabled())
			skillsMgr, extraContext, err := buildWorkDirContext(s.settings, sessWorkDir, s.cfg.EnableWorkflows)
			if err != nil {
				return nil, err
			}
			if skillsMgr != nil {
				registry.Register(tools.NewSkillRefTool(skillsMgr))
			}
			gwSess := &GatewaySession{
				ID:           sessionID,
				WorkDir:      sessWorkDir,
				Manager:      sess,
				Registry:     registry,
				SkillsMgr:    skillsMgr,
				ExtraContext: extraContext,
				DelegateMode: s.cfg.EnableDelegate,
				Workflows:    s.cfg.EnableWorkflows,
				LastUsed:     time.Now(),
			}
			if s.cfg.EnableSubAgents || s.cfg.EnableDelegate || s.cfg.EnableWorkflows {
				compactionSettings := ctxpkg.CompactionSettings{
					Enabled:          s.settings.Compaction.Enabled,
					ReserveTokens:    s.settings.Compaction.ReserveTokens,
					KeepRecentTokens: s.settings.Compaction.KeepRecentTokens,
					Tokenizer:        s.settings.Compaction.Tokenizer,
					TokenizerModel:   s.settings.Compaction.TokenizerModel,
					Template:         s.settings.Compaction.Template,
				}
				factory := agent.NewAgentFactoryWithOptions(s.provider, s.model, s.settings, s.sandboxMgr, extraContext, skillsMgr, compactionSettings, nil, agent.AgentFactoryOptions{
					MultiAgentEnabled: true,
					DelegateEnabled:   s.cfg.EnableDelegate,
					WorkflowsEnabled:  s.cfg.EnableWorkflows,
					Allow:             s.getAllow(),
				})
				gwSess.AgentMgr = agent.NewAgentManager(factory)
			}
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
				registry := tools.NewRegistry(sessWorkDir, s.sandboxMgr.GetActive())
				registry.RegisterDefaultsWithPlanTool(s.settings.IsPlanToolEnabled())
				skillsMgr, extraContext, err := buildWorkDirContext(s.settings, sessWorkDir, s.cfg.EnableWorkflows)
				if err != nil {
					return nil, err
				}
				if skillsMgr != nil {
					registry.Register(tools.NewSkillRefTool(skillsMgr))
				}
				gwSess := &GatewaySession{
					ID:           defaultID,
					WorkDir:      sessWorkDir,
					Manager:      sess,
					Registry:     registry,
					SkillsMgr:    skillsMgr,
					ExtraContext: extraContext,
					DelegateMode: s.cfg.EnableDelegate,
					Workflows:    s.cfg.EnableWorkflows,
					LastUsed:     time.Now(),
				}
				if s.cfg.EnableSubAgents || s.cfg.EnableDelegate || s.cfg.EnableWorkflows {
					compactionSettings := ctxpkg.CompactionSettings{
						Enabled:          s.settings.Compaction.Enabled,
						ReserveTokens:    s.settings.Compaction.ReserveTokens,
						KeepRecentTokens: s.settings.Compaction.KeepRecentTokens,
						Tokenizer:        s.settings.Compaction.Tokenizer,
						TokenizerModel:   s.settings.Compaction.TokenizerModel,
						Template:         s.settings.Compaction.Template,
					}
					factory := agent.NewAgentFactoryWithOptions(s.provider, s.model, s.settings, s.sandboxMgr, extraContext, skillsMgr, compactionSettings, nil, agent.AgentFactoryOptions{
						MultiAgentEnabled: true,
						DelegateEnabled:   s.cfg.EnableDelegate,
						WorkflowsEnabled:  s.cfg.EnableWorkflows,
						Allow:             s.getAllow(),
					})
					gwSess.AgentMgr = agent.NewAgentManager(factory)
				}
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

	skillsMgr, extraContext, err := buildWorkDirContext(s.settings, workDir, s.cfg.EnableWorkflows)
	if err != nil {
		return nil, err
	}

	registry := tools.NewRegistry(workDir, s.sandboxMgr.GetActive())
	registry.RegisterDefaultsWithPlanTool(s.settings.IsPlanToolEnabled())
	if skillsMgr != nil {
		registry.Register(tools.NewSkillRefTool(skillsMgr))
	}

	sess := &GatewaySession{
		ID:           id,
		WorkDir:      workDir,
		Manager:      mgr,
		Registry:     registry,
		Mode:         "",
		SkillsMgr:    skillsMgr,
		ExtraContext: extraContext,
		DelegateMode: s.cfg.EnableDelegate,
		Workflows:    s.cfg.EnableWorkflows,
		LastUsed:     time.Now(),
	}

	// Create agent manager if sub-agent, delegate, or workflow mode is enabled.
	if s.cfg.EnableSubAgents || s.cfg.EnableDelegate || s.cfg.EnableWorkflows {
		compactionSettings := ctxpkg.CompactionSettings{
			Enabled:          s.settings.Compaction.Enabled,
			ReserveTokens:    s.settings.Compaction.ReserveTokens,
			KeepRecentTokens: s.settings.Compaction.KeepRecentTokens,
			Tokenizer:        s.settings.Compaction.Tokenizer,
			TokenizerModel:   s.settings.Compaction.TokenizerModel,
			Template:         s.settings.Compaction.Template,
		}
		factory := agent.NewAgentFactoryWithOptions(s.provider, s.model, s.settings, s.sandboxMgr, extraContext, skillsMgr, compactionSettings, nil, agent.AgentFactoryOptions{
			MultiAgentEnabled: true,
			DelegateEnabled:   s.cfg.EnableDelegate,
			WorkflowsEnabled:  s.cfg.EnableWorkflows,
			Allow:             s.getAllow(),
		})
		sess.AgentMgr = agent.NewAgentManager(factory)
	}

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

func (s *Server) clearSession(sess *GatewaySession, workDir string) error {
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
	sess.LastUsed = time.Now()
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
func parseMessages(msgs []RequestMessage) (lastUser string, systemMsgs []string, history []RequestMessage) {
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
		return "", systemMsgs, nil
	}
	lastUser = msgs[lastIdx].Content

	// Everything before the last user message (excluding system) is history
	for i := 0; i < lastIdx; i++ {
		if msgs[i].Role != "system" {
			history = append(history, msgs[i])
		}
	}
	return lastUser, systemMsgs, history
}

// convertHistoryMessages converts OpenAI-format history to internal provider.Message.
func convertHistoryMessages(msgs []RequestMessage) []provider.Message {
	result := make([]provider.Message, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "user":
			result = append(result, provider.NewUserMessage(m.Content))
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
