package openaiapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/startvibecoding/mothx/internal/session"
)

type capabilitySnapshot struct {
	Mode         string
	DelegateMode bool
	MultiAgent   bool
	Workflows    bool
	WebSearch    bool
	Browser      bool
	A2AMaster    bool
}

func newRunID() string {
	return "run_" + session.GenerateID()
}

func capabilitySnapshotFromSession(sess *APISession) capabilitySnapshot {
	if sess == nil {
		return capabilitySnapshot{}
	}
	return capabilitySnapshot{
		Mode:         sess.Mode,
		DelegateMode: sess.DelegateMode,
		MultiAgent:   sess.MultiAgent,
		Workflows:    sess.Workflows,
		WebSearch:    sess.WebSearch,
		Browser:      sess.Browser,
		A2AMaster:    sess.A2AMaster,
	}
}

func (c capabilitySnapshot) values() map[string]string {
	return map[string]string{
		"mode":         c.Mode,
		"delegateMode": strconv.FormatBool(c.DelegateMode),
		"multiAgent":   strconv.FormatBool(c.MultiAgent),
		"workflows":    strconv.FormatBool(c.Workflows),
		"webSearch":    strconv.FormatBool(c.WebSearch),
		"browser":      strconv.FormatBool(c.Browser),
		"a2aMaster":    strconv.FormatBool(c.A2AMaster),
	}
}

func (s *Server) persistSessionCapabilitiesWithEvents(sess *APISession, before capabilitySnapshot, source, actor, runID string, data map[string]any) error {
	if err := s.persistSessionCapabilities(sess); err != nil {
		return err
	}
	return s.recordSessionCapabilityChanges(sess, before, source, actor, runID, data)
}

func (s *Server) recordSessionCapabilityChanges(sess *APISession, before capabilitySnapshot, source, actor, runID string, data map[string]any) error {
	if s == nil || s.settings == nil || sess == nil || sess.ID == "" {
		return nil
	}
	after := capabilitySnapshotFromSession(sess)
	beforeValues := before.values()
	afterValues := after.values()
	eventData := rawEventData(data)
	for _, capability := range []string{"mode", "delegateMode", "multiAgent", "workflows", "webSearch", "browser", "a2aMaster"} {
		oldValue := beforeValues[capability]
		newValue := afterValues[capability]
		if oldValue == newValue {
			continue
		}
		if _, err := session.SaveSessionCapabilityEvent(s.settings.GetSessionDir(), session.SessionCapabilityEvent{
			SessionID:  sess.ID,
			RunID:      runID,
			EventType:  "changed",
			Source:     source,
			Actor:      actor,
			Capability: capability,
			OldValue:   oldValue,
			NewValue:   newValue,
			Timestamp:  time.Now(),
			Data:       eventData,
		}); err != nil {
			return fmt.Errorf("save capability event: %w", err)
		}
	}
	return nil
}

func (s *Server) recordSessionRunEvent(sess *APISession, runID, eventType, status, source, modelID, mode string, data map[string]any) error {
	if s == nil || s.settings == nil || sess == nil || sess.ID == "" || runID == "" {
		return nil
	}
	if _, err := session.SaveSessionRunEvent(s.settings.GetSessionDir(), session.SessionRunEvent{
		SessionID: sess.ID,
		RunID:     runID,
		EventType: eventType,
		Source:    source,
		Status:    status,
		Model:     modelID,
		Mode:      mode,
		Timestamp: time.Now(),
		Data:      rawEventData(data),
	}); err != nil {
		return fmt.Errorf("save run event: %w", err)
	}
	return nil
}

func rawEventData(data map[string]any) json.RawMessage {
	if len(data) == 0 {
		return nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	return raw
}

func runEventTypeForStatus(status string) string {
	switch status {
	case "failed":
		return "failed"
	case "canceled":
		return "canceled"
	default:
		return "finished"
	}
}

func usageEventData(usage CompletionUsage, errMsg string) map[string]any {
	data := map[string]any{
		"usage": map[string]any{
			"prompt_tokens":     usage.PromptTokens,
			"completion_tokens": usage.CompletionTokens,
			"total_tokens":      usage.TotalTokens,
		},
	}
	if errMsg != "" {
		data["error"] = errMsg
	}
	return data
}
