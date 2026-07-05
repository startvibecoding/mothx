package serve

import (
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type serveLogEvent struct {
	Type      string       `json:"type"`
	Message   string       `json:"message,omitempty"`
	Timestamp time.Time    `json:"timestamp,omitempty"`
	Status    *serveStatus `json:"status,omitempty"`
}

const logHistoryLimit = 200

type logHub struct {
	mu          sync.Mutex
	subscribers map[chan serveLogEvent]struct{}
	history     []serveLogEvent
	historySize int
	closed      bool
}

func newLogHub() *logHub {
	return &logHub{subscribers: make(map[chan serveLogEvent]struct{}), historySize: logHistoryLimit}
}

func installLogHub(hub *logHub) func() {
	if hub == nil {
		return func() {}
	}
	previous := log.Writer()
	log.SetOutput(io.MultiWriter(previous, hub))
	return func() {
		log.SetOutput(previous)
		hub.close()
	}
}

func (h *logHub) Write(p []byte) (int, error) {
	for _, line := range strings.Split(string(p), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		h.publish(serveLogEvent{Type: "log", Message: line, Timestamp: time.Now()})
	}
	return len(p), nil
}

func (h *logHub) subscribe() (<-chan serveLogEvent, []serveLogEvent, func()) {
	ch := make(chan serveLogEvent, 32)
	h.mu.Lock()
	if h.closed {
		close(ch)
		h.mu.Unlock()
		return ch, nil, func() {}
	}
	h.subscribers[ch] = struct{}{}
	history := append([]serveLogEvent(nil), h.history...)
	h.mu.Unlock()

	return ch, history, func() {
		h.mu.Lock()
		if _, ok := h.subscribers[ch]; ok {
			delete(h.subscribers, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
}

func (h *logHub) publish(ev serveLogEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.rememberLocked(ev)
	for ch := range h.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (h *logHub) rememberLocked(ev serveLogEvent) {
	if h.historySize <= 0 || ev.Type == "heartbeat" {
		return
	}
	h.history = append(h.history, ev)
	if len(h.history) > h.historySize {
		copy(h.history, h.history[len(h.history)-h.historySize:])
		h.history = h.history[:h.historySize]
	}
}

func (h *logHub) close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for ch := range h.subscribers {
		close(ch)
		delete(h.subscribers, ch)
	}
}

func (rt *channelRuntime) handleLogs(sessions activeSessionManager) websocket.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		if rt.logHub == nil {
			_ = websocket.JSON.Send(ws, serveLogEvent{
				Type:      "error",
				Message:   "log stream not configured",
				Timestamp: time.Now(),
			})
			return
		}

		ch, history, unsubscribe := rt.logHub.subscribe()
		defer unsubscribe()

		status := rt.statusSnapshot(sessions)
		if err := websocket.JSON.Send(ws, serveLogEvent{
			Type:      "connected",
			Timestamp: time.Now(),
			Status:    &status,
		}); err != nil {
			return
		}
		for _, ev := range history {
			if err := websocket.JSON.Send(ws, ev); err != nil {
				return
			}
		}

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					return
				}
				if err := websocket.JSON.Send(ws, ev); err != nil {
					return
				}
			case now := <-ticker.C:
				if err := websocket.JSON.Send(ws, serveLogEvent{Type: "heartbeat", Timestamp: now}); err != nil {
					return
				}
			}
		}
	})
}
