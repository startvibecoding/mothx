package openaiapi

import (
	"fmt"
	"strings"
)

// patchActiveSessionCapabilities updates an already locked active session for
// slash commands. Registry synchronization remains capability-owned: a mode
// change only persists the runtime setting and never re-registers tools.
func (s *Server) patchActiveSessionCapabilities(sess *APISession, patch SessionCapabilityPatch, source, actor, runID string, data map[string]any) (*SessionCapabilities, error) {
	if sess == nil {
		return nil, ErrSessionNotFound
	}
	before := capabilitySnapshotFromSession(sess)
	refreshContext := false
	registryChanged := false

	if patch.Mode != nil {
		mode := strings.TrimSpace(*patch.Mode)
		if err := validateCapabilityMode(mode); err != nil {
			return nil, err
		}
		sess.Mode = mode
	}
	if applyBoolOption(&sess.WebSearch, patch.WebSearch) {
		// Web search is read when building the next agent configuration.
	}
	if applyBoolOption(&sess.Browser, patch.Browser) {
		refreshContext = true
		registryChanged = true
	}
	if applyBoolOption(&sess.A2AMaster, patch.A2AMaster) {
		registryChanged = true
	}
	delegate := patch.DelegateMode
	if delegate == nil {
		delegate = patch.Delegate
	}
	if applyBoolOption(&sess.DelegateMode, delegate) {
		registryChanged = true
	}
	if applyBoolOption(&sess.MultiAgent, patch.MultiAgent) {
		registryChanged = true
	}
	if applyBoolOption(&sess.Workflows, patch.Workflows) {
		refreshContext = true
		registryChanged = true
	}
	if registryChanged {
		if err := s.syncSessionTools(sess, refreshContext); err != nil {
			return nil, fmt.Errorf("sync session tools: %w", err)
		}
	}
	if err := s.persistSessionCapabilitiesWithEvents(sess, before, source, actor, runID, data); err != nil {
		return nil, err
	}
	sess.Touch()
	caps := s.capabilitiesFromSession(sess, true, sess.Manager != nil)
	caps.RuntimeOnly = false
	caps.PersistenceNote = ""
	return &caps, nil
}
