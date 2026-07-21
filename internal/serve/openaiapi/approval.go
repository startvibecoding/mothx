package openaiapi

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/startvibecoding/mothx/internal/agent"
	"github.com/startvibecoding/mothx/internal/config"
)

func approvalCommand(args map[string]any) string {
	for _, key := range []string{"command", "cmd"} {
		if value, ok := args[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func approvalPath(args map[string]any) string {
	if value, ok := args["path"].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func suggestedApprovalCommandPrefix(command string) string {
	command = strings.TrimLeft(command, " \t\r\n")
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	count := len(fields)
	if count > 2 {
		count = 2
	}
	prefix := strings.Join(fields[:count], " ")
	if index := strings.Index(command, prefix); index >= 0 {
		prefix = command[index : index+len(prefix)]
	}
	if len(command) > len(prefix) && command[len(prefix)] == ' ' {
		return prefix + " "
	}
	return prefix
}

func approvalToolLabel(name string) string {
	// ASCII-only title case for internal tool names; avoids the deprecated
	// strings.Title which has incorrect Unicode word-boundary semantics.
	words := strings.Split(strings.ReplaceAll(name, "_", " "), " ")
	for i, w := range words {
		words[i] = approvalCapitalize(w)
	}
	return strings.Join(words, " ")
}

func approvalCapitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 'a' - 'A'
	}
	return string(r)
}

func approvalRequestFromEvent(sess *APISession, ev agent.Event) SessionApprovalRequest {
	toolName := ev.ApprovalTool
	args := ev.ApprovalArgs
	summary := "Run " + toolName
	risk := "medium"
	reason := "requires confirmation in agent mode"
	details := map[string]any{}
	switch toolName {
	case "bash":
		command := approvalCommand(args)
		summary = "Run bash: " + command
		risk = "high"
		details["command"] = command
		details["workDir"] = sess.WorkDir
	case "write", "edit", "delete":
		path := approvalPath(args)
		summary = approvalCapitalize(toolName) + " " + path
		risk = "high"
		details["path"] = path
		details["operation"] = toolName
	case "git_access":
		summary = "Allow git metadata access"
		risk = "low"
	}
	actions := []string{"approve_once", "deny_once"}
	if toolName == "bash" && approvalCommand(args) != "" {
		actions = append(actions, "remember_command", "remember_prefix")
	}
	if (toolName == "write" || toolName == "edit") && approvalPath(args) != "" {
		actions = append(actions, "allow_edit_path")
	}
	return SessionApprovalRequest{
		ApprovalID: ev.ApprovalID, SessionID: sess.ID, RunID: sess.ActiveRunID(), Timestamp: time.Now().UTC().Format(time.RFC3339Nano), AgentID: string(ev.AgentID), Mode: sess.Mode,
		Risk: risk, Summary: summary, Reason: reason,
		Tool:    map[string]any{"name": toolName, "label": approvalToolLabel(toolName), "args": args, "details": details},
		Context: map[string]any{"workDir": sess.WorkDir}, Actions: actions,
	}
}

func (s *Server) registerSessionApproval(sess *APISession, a *agent.Agent, ev agent.Event) *SessionApprovalRequest {
	if sess == nil || a == nil || ev.ApprovalID == "" {
		return nil
	}
	request := approvalRequestFromEvent(sess, ev)
	sess.approvalMu.Lock()
	if sess.pendingApprovals == nil {
		sess.pendingApprovals = make(map[string]pendingSessionApproval)
	}
	sess.pendingApprovals[request.ApprovalID] = pendingSessionApproval{Request: request, Agent: a}
	sess.approvalMu.Unlock()
	s.publishSessionStreamEvent(sess.ID, "approval_request", request)
	return &request
}

func (s *Server) resolveSessionApproval(id, approvalID string, response SessionApprovalResponse) (*SessionApprovalResolution, error) {
	if id == "" || approvalID == "" {
		return nil, ErrSessionNotFound
	}
	if response.Action != "approve_once" && response.Action != "deny_once" && response.Action != "remember_command" && response.Action != "remember_prefix" && response.Action != "allow_edit_path" {
		return nil, fmt.Errorf("%w: unsupported approval action", ErrInvalidCapability)
	}
	sess, err := s.pool.getExact(id)
	if err != nil || sess == nil {
		return nil, ErrSessionNotFound
	}
	sess.approvalMu.Lock()
	pending, ok := sess.pendingApprovals[approvalID]
	if ok {
		delete(sess.pendingApprovals, approvalID)
	}
	sess.approvalMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("approval %q is no longer pending", approvalID)
	}
	approved := response.Action != "deny_once"
	if approved {
		if err := s.rememberApprovalRule(pending.Request, response.Action); err != nil {
			sess.approvalMu.Lock()
			sess.pendingApprovals[approvalID] = pending
			sess.approvalMu.Unlock()
			return nil, err
		}
	}
	if pending.Agent != nil {
		pending.Agent.HandleApprovalResponse(approvalID, approved)
	}
	resolution := &SessionApprovalResolution{ApprovalID: approvalID, SessionID: id, Action: response.Action, Status: "resolved"}
	if approved {
		resolution.Message = "approval accepted"
	} else {
		resolution.Message = "approval denied"
	}
	if err := s.recordSessionApprovalResolution(sess, pending.Request, resolution); err != nil {
		return nil, err
	}
	s.publishSessionStreamEvent(id, "approval_response", resolution)
	s.publishSessionStreamEvent(id, "approval_resolved", resolution)
	return resolution, nil
}

func (s *Server) rememberApprovalRule(request SessionApprovalRequest, action string) error {
	if action == "approve_once" {
		return nil
	}
	args, _ := request.Tool["args"].(map[string]any)
	allow := s.getAllow()
	var changed bool
	switch action {
	case "remember_command":
		changed = allow.AddBashCommand(approvalCommand(args))
	case "remember_prefix":
		changed = allow.AddBashPrefix(suggestedApprovalCommandPrefix(approvalCommand(args)))
	case "allow_edit_path":
		changed = allow.AddEditPath(filepath.Clean(approvalPath(args)))
	}
	if !changed {
		return nil
	}
	if s.saveProjectAllow != nil {
		if err := s.saveProjectAllow(allow); err != nil {
			s.rollbackApprovalRule(allow, args, action)
			return fmt.Errorf("save project allow rule: %w", err)
		}
		return nil
	}
	if err := allow.SaveProject(); err != nil {
		s.rollbackApprovalRule(allow, args, action)
		return fmt.Errorf("save project allow rule: %w", err)
	}
	return nil
}

func (s *Server) rollbackApprovalRule(allow *config.AllowConfig, args map[string]any, action string) {
	switch action {
	case "remember_command":
		allow.RemoveBashCommand(approvalCommand(args))
	case "remember_prefix":
		allow.RemoveBashPrefix(suggestedApprovalCommandPrefix(approvalCommand(args)))
	case "allow_edit_path":
		allow.RemoveEditPath(filepath.Clean(approvalPath(args)))
	}
}

func (s *Server) clearSessionApprovals(sess *APISession, status, message string) {
	if sess == nil {
		return
	}
	sess.approvalMu.Lock()
	pending := sess.pendingApprovals
	sess.pendingApprovals = make(map[string]pendingSessionApproval)
	sess.approvalMu.Unlock()
	for approvalID, item := range pending {
		if item.Agent != nil {
			// A rejected response unblocks any tool execution waiting on this approval.
			item.Agent.HandleApprovalResponse(approvalID, false)
		}
		resolution := &SessionApprovalResolution{ApprovalID: approvalID, SessionID: sess.ID, Action: "deny_once", Status: status, Message: message}
		_ = s.recordSessionApprovalResolution(sess, item.Request, resolution)
		s.publishSessionStreamEvent(sess.ID, "approval_resolved", resolution)
	}
}

func (s *Server) recordSessionApprovalResolution(sess *APISession, request SessionApprovalRequest, resolution *SessionApprovalResolution) error {
	if sess == nil || resolution == nil {
		return nil
	}
	sess.approvalMu.Lock()
	runID := sess.activeRunID
	sess.approvalMu.Unlock()
	return s.recordSessionRunEvent(sess, runID, "approval_resolved", resolution.Status, "approval", "", request.Mode, map[string]any{
		"approval":   request,
		"resolution": resolution,
	})
}

// ResolveSessionApproval applies the first accepted WebUI decision and resumes its agent.
func (s *Server) ResolveSessionApproval(sessionID, approvalID string, response SessionApprovalResponse) (*SessionApprovalResolution, error) {
	return s.resolveSessionApproval(sessionID, approvalID, response)
}
