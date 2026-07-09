package channels

import (
	"context"
	"strings"
	"testing"

	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/cron"
	"github.com/startvibecoding/mothx/internal/messaging"
	"github.com/startvibecoding/mothx/internal/provider"
	"github.com/startvibecoding/mothx/internal/sandbox"
	"github.com/startvibecoding/mothx/internal/serve/hooks"
	"github.com/startvibecoding/mothx/internal/session"
	"github.com/startvibecoding/mothx/internal/tools"
)

type recordingChannelProvider struct {
	models []*provider.Model
	calls  []provider.ChatParams
}

func newRecordingChannelProvider() *recordingChannelProvider {
	return &recordingChannelProvider{
		models: []*provider.Model{{ID: "m1", Name: "Model 1", ContextWindow: 4096, MaxTokens: 1024}},
	}
}

func (p *recordingChannelProvider) Chat(ctx context.Context, params provider.ChatParams) <-chan provider.StreamEvent {
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

func (p *recordingChannelProvider) Name() string              { return "recording-channel" }
func (p *recordingChannelProvider) API() string               { return "openai-chat" }
func (p *recordingChannelProvider) Models() []*provider.Model { return p.models }
func (p *recordingChannelProvider) GetModel(id string) *provider.Model {
	for _, m := range p.models {
		if m.ID == id {
			return m
		}
	}
	return nil
}

func TestResolveSessionCronOnlyDoesNotExposeSubAgentTools(t *testing.T) {
	workDir := t.TempDir()
	settings := config.DefaultSettings()
	settings.SessionDir = t.TempDir()
	cfg := DefaultConfig()
	cfg.WorkDir = workDir
	cfg.MultiAgent = false
	cfg.Cron.Enabled = true

	store := cron.NewSQLiteCronStore(t.TempDir())
	p := newRecordingChannelProvider()
	d := &Dispatcher{
		cfg:        cfg,
		settings:   settings,
		allow:      &config.AllowConfig{},
		sessionDir: settings.SessionDir,
		security:   NewSecurity(cfg),
		hooksMgr:   hooks.NewManager("", ""),
		provider:   p,
		model:      p.models[0],
		cronStore:  store,
		sessions:   make(map[string]*ChannelSession),
	}

	if d.EnsureAgentManager() == nil {
		t.Fatal("cron should be able to initialize an agent manager without multi-agent")
	}
	sess, err := d.resolveSession("ws", "test-user")
	if err != nil {
		t.Fatalf("resolve session: %v", err)
	}
	if _, ok := sess.Registry.Get("cron"); !ok {
		t.Fatal("cron-only session should expose cron tool")
	}
	for _, name := range []string{"subagent_spawn", "subagent_status", "subagent_send", "subagent_destroy"} {
		if _, ok := sess.Registry.Get(name); ok {
			t.Fatalf("cron-only session should not expose %s", name)
		}
	}
}

func TestBuildAgentLoadsReplayState(t *testing.T) {
	tmpDir := t.TempDir()
	p := newRecordingChannelProvider()
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
		cfg:      DefaultConfig(),
		settings: settings,
		hooksMgr: hooks.NewManager("", ""),
		provider: p,
		model:    p.models[0],
	}
	sess := &ChannelSession{
		ID:       "channels/ws/test-user",
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
		t.Fatal("channel agent did not replay compacted summary")
	}
	if foundOldUser {
		t.Fatal("channel agent still included pre-compaction old user message")
	}
	if !foundRecentUser {
		t.Fatal("channel agent lost recent user message from replay state")
	}
}

func TestBuildAgentUsesCompactionSettings(t *testing.T) {
	tmpDir := t.TempDir()
	p := newRecordingChannelProvider()
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
		cfg:      DefaultConfig(),
		settings: settings,
		hooksMgr: hooks.NewManager("", ""),
		provider: p,
		model:    p.models[0],
	}
	sess := &ChannelSession{
		ID:       "channels/ws/test-user",
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
		t.Fatal("agent should use channel compaction keepRecent settings")
	}
}

func TestCompactCommandRunsImmediately(t *testing.T) {
	tmpDir := t.TempDir()
	p := newRecordingChannelProvider()
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

	sess := &ChannelSession{
		ID:       sessionKey("ws", "test-user"),
		Platform: "ws",
		UserID:   "test-user",
		WorkDir:  tmpDir,
		Manager:  mgr,
		Registry: tools.NewRegistry(tmpDir, sandbox.NewNoneSandbox()),
		Mode:     "agent",
	}
	d := &Dispatcher{
		cfg:      DefaultConfig(),
		settings: settings,
		hooksMgr: hooks.NewManager("", ""),
		provider: p,
		model:    p.models[0],
		sessions: map[string]*ChannelSession{sess.ID: sess},
	}

	reply, err := d.handleCommand(messaging.InboundMessage{Platform: "ws", UserID: "test-user", Text: "/compact"})
	if err != nil {
		t.Fatalf("handleCommand() error = %v", err)
	}
	if !strings.Contains(reply, "compacted") {
		t.Fatalf("reply = %q, want compaction confirmation", reply)
	}
	if sess.ForceCompact {
		t.Fatal("ForceCompact should not be set for immediate compaction")
	}
	replay := mgr.GetReplayState()
	if len(replay.Messages) == 0 || !replay.Messages[0].SystemInjected {
		t.Fatalf("expected compacted summary in replay, got %#v", replay.Messages)
	}
}

func TestCompactCommandForcesSummaryOnlyWhenOnlyRecentContext(t *testing.T) {
	tmpDir := t.TempDir()
	p := newRecordingChannelProvider()
	settings := config.DefaultSettings()

	mgr := session.New(tmpDir, t.TempDir())
	if err := mgr.Init(); err != nil {
		t.Fatalf("init session: %v", err)
	}
	_, _ = mgr.AppendMessage(provider.NewUserMessage("hello"))
	_, _ = mgr.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "hi"}}))

	sess := &ChannelSession{
		ID:       sessionKey("ws", "test-user"),
		Platform: "ws",
		UserID:   "test-user",
		WorkDir:  tmpDir,
		Manager:  mgr,
		Registry: tools.NewRegistry(tmpDir, sandbox.NewNoneSandbox()),
		Mode:     "agent",
	}
	d := &Dispatcher{
		cfg:      DefaultConfig(),
		settings: settings,
		hooksMgr: hooks.NewManager("", ""),
		provider: p,
		model:    p.models[0],
		sessions: map[string]*ChannelSession{sess.ID: sess},
	}

	reply, err := d.handleCommand(messaging.InboundMessage{Platform: "ws", UserID: "test-user", Text: "/compact"})
	if err != nil {
		t.Fatalf("handleCommand() error = %v", err)
	}
	if sess.ForceCompact {
		t.Fatal("ForceCompact should not be set for immediate compaction")
	}
	if !strings.Contains(reply, "compacted") {
		t.Fatalf("reply = %q, want compaction confirmation", reply)
	}
	replay := mgr.GetReplayState()
	if len(replay.Messages) != 1 || !replay.Messages[0].SystemInjected {
		t.Fatalf("expected summary-only replay, got %#v", replay.Messages)
	}
}
