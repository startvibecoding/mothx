package openaiapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/startvibecoding/mothx/internal/session"
)

type sessionStreamEvent struct {
	Name string
	Data any
}

type sessionStreamHub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan sessionStreamEvent]struct{}
}

func newSessionStreamHub() *sessionStreamHub {
	return &sessionStreamHub{subscribers: make(map[string]map[chan sessionStreamEvent]struct{})}
}

func (h *sessionStreamHub) subscribe(sessionID string) (<-chan sessionStreamEvent, func()) {
	ch := make(chan sessionStreamEvent, 128)
	if h == nil || sessionID == "" {
		close(ch)
		return ch, func() {}
	}
	h.mu.Lock()
	if h.subscribers[sessionID] == nil {
		h.subscribers[sessionID] = make(map[chan sessionStreamEvent]struct{})
	}
	h.subscribers[sessionID][ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if subs := h.subscribers[sessionID]; subs != nil {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(h.subscribers, sessionID)
			}
		}
		h.mu.Unlock()
		close(ch)
	}
	return ch, cancel
}

func (h *sessionStreamHub) publish(sessionID string, event sessionStreamEvent) {
	if h == nil || sessionID == "" || event.Name == "" {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subscribers[sessionID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *Server) getSessionStreamHub() *sessionStreamHub {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.streamHub == nil {
		s.streamHub = newSessionStreamHub()
	}
	return s.streamHub
}

func (s *Server) publishSessionStreamEvent(sessionID, eventName string, data any) {
	hub := s.getSessionStreamHub()
	if hub == nil {
		return
	}
	hub.publish(sessionID, sessionStreamEvent{Name: eventName, Data: data})
}

func (s *Server) publishToolEvent(sessionID string, event ToolStatusEvent) {
	if sessionID == "" {
		return
	}
	s.publishSessionStreamEvent(sessionID, "tool_event", event)
}

func (s *Server) publishTranscriptEvent(sessionID string, evt TranscriptStreamEvent) {
	if sessionID == "" {
		return
	}
	if evt.XSessionID == "" {
		evt.XSessionID = sessionID
	}
	s.publishSessionStreamEvent(sessionID, "transcript", evt)
}

func (s *Server) writeTranscriptEvent(sse *SSEWriter, sessionID string, evt TranscriptStreamEvent) {
	if evt.XSessionID == "" {
		evt.XSessionID = sessionID
	}
	if sse != nil {
		sse.WriteTranscriptEvent(evt)
	}
	s.publishTranscriptEvent(sessionID, evt)
}

func (s *Server) publishSessionStreamDone(sessionID string) {
	s.publishSessionStreamEvent(sessionID, "done", map[string]any{"sessionId": sessionID})
}

type sessionStreamCursor struct {
	EntrySeq      int64
	RunSeq        int64
	CapabilitySeq int64
}

// StreamSession streams persisted and live transcript/event updates for one WebUI session.
func (s *Server) StreamSession(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s == nil || s.settings == nil || id == "" {
		writeError(w, http.StatusNotFound, "session not found", "not_found")
		return
	}
	if _, found, err := s.findSessionWorkDir(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "server_error")
		return
	} else if !found {
		writeError(w, http.StatusNotFound, "session not found", "not_found")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is not supported", "server_error")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	cursor := sessionStreamCursor{
		EntrySeq:      streamIntQuery(r, "after_entry_seq", "afterEntrySeq", "entrySeq"),
		RunSeq:        streamIntQuery(r, "after_run_seq", "afterRunSeq", "runSeq"),
		CapabilitySeq: streamIntQuery(r, "after_capability_seq", "afterCapabilitySeq", "capabilitySeq"),
	}
	hub := s.getSessionStreamHub()
	events, cancel := hub.subscribe(id)
	defer cancel()

	if _, err := s.replaySessionStream(w, flusher, id, &cursor, true); err != nil {
		return
	}
	if !s.isSessionRunActive(id) {
		_ = writeSessionSSE(w, flusher, "done", map[string]any{"sessionId": id})
		return
	}

	poll := time.NewTicker(500 * time.Millisecond)
	defer poll.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-events:
			if !ok {
				return
			}
			if evt.Name == "done" {
				_, _ = s.replaySessionStream(w, flusher, id, &cursor, false)
				_ = writeSessionSSE(w, flusher, "done", evt.Data)
				return
			}
			if err := writeSessionSSE(w, flusher, evt.Name, evt.Data); err != nil {
				return
			}
		case <-poll.C:
			if _, err := s.replaySessionStream(w, flusher, id, &cursor, false); err != nil {
				return
			}
			if !s.isSessionRunActive(id) {
				_, _ = s.replaySessionStream(w, flusher, id, &cursor, false)
				_ = writeSessionSSE(w, flusher, "done", map[string]any{"sessionId": id})
				return
			}
		case <-heartbeat.C:
			if err := writeSessionSSE(w, flusher, "heartbeat", map[string]any{"sessionId": id}); err != nil {
				return
			}
		}
	}
}

func (s *Server) replaySessionStream(w http.ResponseWriter, flusher http.Flusher, sessionID string, cursor *sessionStreamCursor, includeMessages bool) (bool, error) {
	if s == nil || s.settings == nil || cursor == nil {
		return false, nil
	}
	sessionDir := s.settings.GetSessionDir()
	changed := false

	if includeMessages {
		messages, err := session.ListSessionMessagesAfter(sessionDir, sessionID, cursor.EntrySeq, 200)
		if err != nil {
			_ = writeSessionSSE(w, flusher, "error", map[string]any{"error": err.Error()})
			return changed, err
		}
		for _, item := range messages {
			for _, entry := range providerMessageToSessionEntries(item.Message, item.Seq, item.EntryID) {
				evt := messageTranscriptEvent(entry)
				evt.XSessionID = sessionID
				if err := writeSessionSSE(w, flusher, "transcript", evt); err != nil {
					return changed, err
				}
				changed = true
			}
			if item.Seq > cursor.EntrySeq {
				cursor.EntrySeq = item.Seq
			}
		}
	}

	runEvents, err := session.ListSessionRunEventsAfter(sessionDir, sessionID, cursor.RunSeq, 200)
	if err != nil {
		_ = writeSessionSSE(w, flusher, "error", map[string]any{"error": err.Error()})
		return changed, err
	}
	for _, item := range runEvents {
		if err := writeSessionSSE(w, flusher, "run_event", sessionRunEventToEntry(item.Event, item.Seq)); err != nil {
			return changed, err
		}
		if item.Seq > cursor.RunSeq {
			cursor.RunSeq = item.Seq
		}
		changed = true
	}

	capabilityEvents, err := session.ListSessionCapabilityEventsAfter(sessionDir, sessionID, cursor.CapabilitySeq, 200)
	if err != nil {
		_ = writeSessionSSE(w, flusher, "error", map[string]any{"error": err.Error()})
		return changed, err
	}
	for _, item := range capabilityEvents {
		if err := writeSessionSSE(w, flusher, "capability_event", sessionCapabilityEventToEntry(item.Event, item.Seq)); err != nil {
			return changed, err
		}
		if item.Seq > cursor.CapabilitySeq {
			cursor.CapabilitySeq = item.Seq
		}
		changed = true
	}

	return changed, nil
}

func (s *Server) isSessionRunActive(id string) bool {
	if s == nil || s.pool == nil || id == "" {
		return false
	}
	sess, err := s.pool.getExact(id)
	if err != nil || sess == nil {
		return false
	}
	return sess.IsRunning()
}

func streamIntQuery(r *http.Request, keys ...string) int64 {
	values := r.URL.Query()
	for _, key := range keys {
		raw := values.Get(key)
		if raw == "" {
			continue
		}
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 {
			return 0
		}
		return n
	}
	return 0
}

func writeSessionSSE(w http.ResponseWriter, flusher http.Flusher, event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return err
	}
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}
