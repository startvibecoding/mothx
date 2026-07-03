package hermes

import (
	"context"
	"strings"
	"testing"

	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/hermes/hooks"
	"github.com/startvibecoding/mothx/internal/messaging"
	"github.com/startvibecoding/mothx/internal/provider"
	"github.com/startvibecoding/mothx/internal/sandbox"
	"github.com/startvibecoding/mothx/internal/session"
	"github.com/startvibecoding/mothx/internal/tools"
)

type recordingHermesProvider struct {
	models []*provider.Model
	calls  []provider.ChatParams
}

func newRecordingHermesProvider() *recordingHermesProvider {
	return &recordingHermesProvider{
		models: []*provider.Model{{ID: "m1", Name: "Model 1", ContextWindow: 4096, MaxTokens: 1024}},
	}
}

func (p *recordingHermesProvider) Chat(ctx context.Context, params provider.ChatParams) <-chan provider.StreamEvent {
	p.calls = append(p.calls, provider.ChatParams{
		Messages: append([]provider.Message(nil), params.Messages...),
	})
	ch := make(chan provider.StreamEvent, 3)
	go func() {
		defer close(ch)
		ch <- provider.StreamEvent{Type: provider.StreamStart}
		ch <- provider.StreamEvent{Type: provider.StreamTextDelta, TextDelta: "ok"}
		ch <- provider.StreamEvent{Type: provider.StreamDone}
	}()
	return ch
}

func (p *recordingHermesProvider) Name() string              { return "recording-hermes" }
func (p *recordingHermesProvider) API() string               { return "openai-chat" }
func (p *recordingHermesProvider) Models() []*provider.Model { return p.models }
func (p *recordingHermesProvider) GetModel(id string) *provider.Model {
	for _, m := range p.models {
		if m.ID == id {
			return m
		}
	}
	return nil
}

func TestBuildAgentLoadsReplayState(t *testing.T) {
	tmpDir := t.TempDir()
	p := newRecordingHermesProvider()
	settings := config.DefaultSettings()
	settings.SessionDir = t.TempDir()

	mgr := session.New(tmpDir, settings.SessionDir)
	if err := mgr.Init(); err != nil {
		t.Fatalf("init session: %v", err)
	}
	oldUser := provider.NewUserMessage("old user context")
	oldAssistant := provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old assistant context"}})
	recentUser := provider.NewUserMessage("recent user context")
	recentAssistant := provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "recent assistant context"}})
	_, _ = mgr.AppendMessage(oldUser)
	_, _ = mgr.AppendMessage(oldAssistant)
	recentUserID, _ := mgr.AppendMessage(recentUser)
	_, _ = mgr.AppendMessage(recentAssistant)
	_, _ = mgr.AppendCompaction("## Goal\ncompacted checkpoint", recentUserID, 100)

	registry := tools.NewRegistry(tmpDir, sandbox.NewNoneSandbox())
	d := &Dispatcher{
		cfg:      DefaultHermesConfig(),
		settings: settings,
		hooksMgr: hooks.NewManager("", ""),
		provider: p,
		model:    p.models[0],
	}
	sess := &HermesSession{
		ID:       "hermes/ws/test-user",
		Platform: "ws",
		UserID:   "test-user",
		WorkDir:  tmpDir,
		Manager:  mgr,
		Registry: registry,
		Mode:     "agent",
	}

	a, cleanup := d.buildAgent(context.Background(), sess, nil)
	defer cleanup(nil)

	for range a.Run(context.Background(), "continue") {
	}

	if len(p.calls) != 1 {
		t.Fatalf("provider call count = %d, want 1", len(p.calls))
	}

	foundSummary := false
	foundOldUser := false
	foundRecentUser := false
	for _, msg := range p.calls[0].Messages {
		if msg.SystemInjected && msg.Content == "## Goal\ncompacted checkpoint" {
			foundSummary = true
		}
		if msg.Content == oldUser.Content {
			foundOldUser = true
		}
		if msg.Content == recentUser.Content {
			foundRecentUser = true
		}
	}

	if !foundSummary {
		t.Fatal("hermes agent did not replay compacted summary")
	}
	if foundOldUser {
		t.Fatal("hermes agent still included pre-compaction old user message")
	}
	if !foundRecentUser {
		t.Fatal("hermes agent lost recent user message from replay state")
	}
}

func TestBuildAgentUsesCompactionSettings(t *testing.T) {
	tmpDir := t.TempDir()
	p := newRecordingHermesProvider()
	settings := config.DefaultSettings()
	settings.Compaction.KeepRecentTokens = 1

	mgr := session.New(tmpDir, t.TempDir())
	if err := mgr.Init(); err != nil {
		t.Fatalf("init session: %v", err)
	}
	_, _ = mgr.AppendMessage(provider.NewUserMessage("old user context"))
	_, _ = mgr.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old assistant context"}}))
	_, _ = mgr.AppendMessage(provider.NewUserMessage("recent user context"))
	_, _ = mgr.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "recent assistant context"}}))

	d := &Dispatcher{
		cfg:      DefaultHermesConfig(),
		settings: settings,
		hooksMgr: hooks.NewManager("", ""),
		provider: p,
		model:    p.models[0],
	}
	sess := &HermesSession{
		ID:       "hermes/ws/test-user",
		Platform: "ws",
		UserID:   "test-user",
		WorkDir:  tmpDir,
		Manager:  mgr,
		Registry: tools.NewRegistry(tmpDir, sandbox.NewNoneSandbox()),
		Mode:     "agent",
	}

	a, cleanup := d.buildAgent(context.Background(), sess, nil)
	defer cleanup(nil)

	if !a.CanCompact() {
		t.Fatal("agent should use Hermes compaction keepRecent settings")
	}
}

func TestCompactCommandOnlySetsFlagWhenCompactable(t *testing.T) {
	tmpDir := t.TempDir()
	p := newRecordingHermesProvider()
	settings := config.DefaultSettings()
	settings.Compaction.KeepRecentTokens = 1

	mgr := session.New(tmpDir, t.TempDir())
	if err := mgr.Init(); err != nil {
		t.Fatalf("init session: %v", err)
	}
	_, _ = mgr.AppendMessage(provider.NewUserMessage("old user context"))
	_, _ = mgr.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old assistant context"}}))
	_, _ = mgr.AppendMessage(provider.NewUserMessage("recent user context"))
	_, _ = mgr.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "recent assistant context"}}))

	sess := &HermesSession{
		ID:       sessionKey("ws", "test-user"),
		Platform: "ws",
		UserID:   "test-user",
		WorkDir:  tmpDir,
		Manager:  mgr,
		Registry: tools.NewRegistry(tmpDir, sandbox.NewNoneSandbox()),
		Mode:     "agent",
	}
	d := &Dispatcher{
		cfg:      DefaultHermesConfig(),
		settings: settings,
		hooksMgr: hooks.NewManager("", ""),
		provider: p,
		model:    p.models[0],
		sessions: map[string]*HermesSession{sess.ID: sess},
	}

	reply, err := d.handleCommand(messaging.InboundMessage{Platform: "ws", UserID: "test-user", Text: "/compact"})
	if err != nil {
		t.Fatalf("handleCommand() error = %v", err)
	}
	if !strings.Contains(reply, "compaction") {
		t.Fatalf("reply = %q, want compaction confirmation", reply)
	}
	if !sess.ForceCompact {
		t.Fatal("ForceCompact should be set for compactable conversation")
	}
}

func TestCompactCommandDoesNotSetFlagWhenOnlyRecentContext(t *testing.T) {
	tmpDir := t.TempDir()
	p := newRecordingHermesProvider()
	settings := config.DefaultSettings()

	mgr := session.New(tmpDir, t.TempDir())
	if err := mgr.Init(); err != nil {
		t.Fatalf("init session: %v", err)
	}
	_, _ = mgr.AppendMessage(provider.NewUserMessage("hello"))
	_, _ = mgr.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "hi"}}))

	sess := &HermesSession{
		ID:       sessionKey("ws", "test-user"),
		Platform: "ws",
		UserID:   "test-user",
		WorkDir:  tmpDir,
		Manager:  mgr,
		Registry: tools.NewRegistry(tmpDir, sandbox.NewNoneSandbox()),
		Mode:     "agent",
	}
	d := &Dispatcher{
		cfg:      DefaultHermesConfig(),
		settings: settings,
		hooksMgr: hooks.NewManager("", ""),
		provider: p,
		model:    p.models[0],
		sessions: map[string]*HermesSession{sess.ID: sess},
	}

	reply, err := d.handleCommand(messaging.InboundMessage{Platform: "ws", UserID: "test-user", Text: "/compact"})
	if err != nil {
		t.Fatalf("handleCommand() error = %v", err)
	}
	if !strings.Contains(reply, "only recent context") {
		t.Fatalf("reply = %q, want only recent context message", reply)
	}
	if sess.ForceCompact {
		t.Fatal("ForceCompact should not be set for non-compactable conversation")
	}
}
