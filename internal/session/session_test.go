package session

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/startvibecoding/mothx/internal/provider"
	_ "modernc.org/sqlite"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)

	if m == nil {
		t.Fatal("expected non-nil manager")
	}

	if m.cwd != "/tmp/test" {
		t.Errorf("expected cwd '/tmp/test', got '%s'", m.cwd)
	}

	if m.sessionDir != sessionDir {
		t.Errorf("expected sessionDir '%s', got '%s'", sessionDir, m.sessionDir)
	}
}

func TestNewDefaultDir(t *testing.T) {
	m := New("/tmp/test", "")

	if m == nil {
		t.Fatal("expected non-nil manager")
	}

	if m.sessionDir == "" {
		t.Error("expected non-empty default session dir")
	}
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)

	if err := m.Init(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.header == nil {
		t.Fatal("expected non-nil header")
	}

	if m.header.Version != CurrentVersion {
		t.Errorf("expected version %d, got %d", CurrentVersion, m.header.Version)
	}

	if m.header.Cwd != "/tmp/test" {
		t.Errorf("expected cwd '/tmp/test', got '%s'", m.header.Cwd)
	}

	if m.header.ID == "" {
		t.Error("expected non-empty ID")
	}

	// Check database was created
	dbPath := filepath.Join(sessionDir, "sessions.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected sessions.db to exist")
	}
}

func TestAppendMessage(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	msg := provider.NewUserMessage("Hello")
	id, err := m.AppendMessage(msg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if id == "" {
		t.Error("expected non-empty ID")
	}

	if len(m.entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(m.entries))
	}

	// Append another message
	msg2 := provider.NewAssistantMessage([]provider.ContentBlock{
		{Type: "text", Text: "Hi there"},
	})
	id2, err := m.AppendMessage(msg2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if id2 == "" {
		t.Error("expected non-empty ID")
	}

	if len(m.entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(m.entries))
	}
}

func TestAppendMessageAutoInitializesSession(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	id, err := m.AppendMessage(provider.NewUserMessage("Hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty message ID")
	}
	if m.GetHeader() == nil {
		t.Fatal("expected session header to be initialized")
	}
	if m.GetFile() == "" {
		t.Fatal("expected session file to be initialized")
	}
	dbPath := filepath.Join(sessionDir, "sessions.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected sessions.db to exist: %v", err)
	}
}

func TestAppendModelChange(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	id, err := m.AppendModelChange("anthropic", "claude-sonnet-4-20250514")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if id == "" {
		t.Error("expected non-empty ID")
	}

	if len(m.entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(m.entries))
	}
}

func TestAppendThinkingLevelChange(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	id, err := m.AppendThinkingLevelChange("high")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if id == "" {
		t.Error("expected non-empty ID")
	}

	if len(m.entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(m.entries))
	}
}

func TestAppendCompaction(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	id, err := m.AppendCompaction("Compacted 10 messages", "entry-1", 1000)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if id == "" {
		t.Error("expected non-empty ID")
	}

	if len(m.entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(m.entries))
	}

	entry, ok := m.entries[0].(CompactionEntry)
	if !ok {
		t.Fatalf("entry type = %T, want CompactionEntry", m.entries[0])
	}
	if entry.SummaryVersion != 1 {
		t.Errorf("SummaryVersion = %d, want 1", entry.SummaryVersion)
	}
}

func TestAppendCompactionMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	_, _ = m.AppendMessage(provider.NewUserMessage("old user"))
	oldAssistantID, _ := m.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old assistant"}}))
	recentUserID, _ := m.AppendMessage(provider.NewUserMessage("recent user"))

	firstCompactionID, err := m.AppendCompaction("summary one", recentUserID, 100)
	if err != nil {
		t.Fatalf("AppendCompaction() error = %v", err)
	}
	first, ok := m.GetLatestCompaction()
	if !ok {
		t.Fatal("expected latest compaction")
	}
	if first.ID != firstCompactionID {
		t.Errorf("latest compaction ID = %q, want %q", first.ID, firstCompactionID)
	}
	if first.SummaryVersion != 1 {
		t.Errorf("first SummaryVersion = %d, want 1", first.SummaryVersion)
	}
	if first.PreviousCompactionID != "" {
		t.Errorf("first PreviousCompactionID = %q, want empty", first.PreviousCompactionID)
	}
	if first.LastSummarizedEntry != oldAssistantID {
		t.Errorf("first LastSummarizedEntry = %q, want %q", first.LastSummarizedEntry, oldAssistantID)
	}

	nextUserID, _ := m.AppendMessage(provider.NewUserMessage("next user"))
	secondCompactionID, err := m.AppendCompaction("summary two", nextUserID, 200)
	if err != nil {
		t.Fatalf("second AppendCompaction() error = %v", err)
	}
	second, ok := m.GetLatestCompaction()
	if !ok {
		t.Fatal("expected latest compaction after second append")
	}
	if second.ID != secondCompactionID {
		t.Errorf("latest compaction ID = %q, want %q", second.ID, secondCompactionID)
	}
	if second.SummaryVersion != 2 {
		t.Errorf("second SummaryVersion = %d, want 2", second.SummaryVersion)
	}
	if second.PreviousCompactionID != firstCompactionID {
		t.Errorf("second PreviousCompactionID = %q, want %q", second.PreviousCompactionID, firstCompactionID)
	}
	if second.LastSummarizedEntry != recentUserID {
		t.Errorf("second LastSummarizedEntry = %q, want %q", second.LastSummarizedEntry, recentUserID)
	}
}

func TestGetHeader(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	header := m.GetHeader()

	if header == nil {
		t.Fatal("expected non-nil header")
	}

	if header.Cwd != "/tmp/test" {
		t.Errorf("expected cwd '/tmp/test', got '%s'", header.Cwd)
	}
}

func TestGetLeafID(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	// Initially nil
	leafID := m.GetLeafID()
	if leafID != nil {
		t.Error("expected nil leaf ID initially")
	}

	// After append
	m.AppendMessage(provider.NewUserMessage("Hello"))
	leafID = m.GetLeafID()
	if leafID == nil {
		t.Error("expected non-nil leaf ID after append")
	}
}

func TestGetFile(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	file := m.GetFile()
	if file == "" {
		t.Error("expected non-empty file path")
	}
}

func TestGetMessages(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	m.AppendMessage(provider.NewUserMessage("Hello"))
	m.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{
		{Type: "text", Text: "Hi"},
	}))

	messages := m.GetMessages()
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}

func TestGetMessagesAppliesCompaction(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	id1, _ := m.AppendMessage(provider.NewUserMessage("old user"))
	_, _ = m.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old assistant"}}))
	_, _ = m.AppendMessage(provider.NewUserMessage("recent user"))
	_, _ = m.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "recent assistant"}}))
	_, _ = m.AppendCompaction("## Goal\ncompacted", id1, 100)

	messages := m.GetMessages()
	if len(messages) != 5 {
		t.Fatalf("expected 5 replayed messages, got %d", len(messages))
	}
	if !messages[0].SystemInjected || messages[0].Content != "## Goal\ncompacted" {
		t.Fatalf("expected leading injected summary, got %+v", messages[0])
	}
	if messages[1].Content != "old user" {
		t.Fatalf("expected first kept message to remain after summary, got %+v", messages[1])
	}
	if messages[3].Content != "recent user" {
		t.Fatalf("expected recent user message in replay, got %+v", messages[3])
	}

	replay := m.GetReplayState()
	if len(replay.EntryIDs) != len(messages) {
		t.Fatalf("entry ID count = %d, want %d", len(replay.EntryIDs), len(messages))
	}
	if replay.EntryIDs[0] != "" {
		t.Fatalf("summary entry ID = %q, want empty", replay.EntryIDs[0])
	}
	if replay.EntryIDs[1] != id1 {
		t.Fatalf("first kept entry ID = %q, want %q", replay.EntryIDs[1], id1)
	}
}

func TestGetMessagesAppliesSummaryOnlyCompaction(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	_, _ = m.AppendMessage(provider.NewUserMessage("old user"))
	_, _ = m.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old assistant"}}))
	_, _ = m.AppendCompaction("## Goal\nsummary only", "", 100)

	messages := m.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(messages))
	}
	if !messages[0].SystemInjected || messages[0].Content != "## Goal\nsummary only" {
		t.Fatalf("expected summary-only replay, got %#v", messages[0])
	}
	compaction, ok := m.GetLatestCompaction()
	if !ok {
		t.Fatal("expected latest compaction")
	}
	if compaction.LastSummarizedEntry == "" {
		t.Fatal("summary-only compaction should record the last summarized entry")
	}
}

func TestGetMessagesCompactionClearsStaleUsage(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	_, _ = m.AppendMessage(provider.NewUserMessage("old user"))
	_, _ = m.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old assistant"}}))
	recentUserID, _ := m.AppendMessage(provider.NewUserMessage("recent user"))
	recentAssistant := provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "recent assistant"}})
	recentAssistant.Usage = &provider.Usage{Input: 1000, Output: 50, TotalTokens: 1050}
	_, _ = m.AppendMessage(recentAssistant)
	_, _ = m.AppendCompaction("## Goal\ncompacted", recentUserID, 1000)

	messages := m.GetMessages()
	if len(messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(messages))
	}
	if messages[2].Usage != nil {
		t.Fatalf("kept assistant usage = %#v, want nil stale usage", messages[2].Usage)
	}
}

func TestGetMessagesAppliesMultipleCompactions(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	_, _ = m.AppendMessage(provider.NewUserMessage("old user"))
	_, _ = m.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old assistant"}}))
	recentUserID, _ := m.AppendMessage(provider.NewUserMessage("recent user"))
	_, _ = m.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "recent assistant"}}))
	_, _ = m.AppendCompaction("## Goal\nsummary one", recentUserID, 100)
	nextUserID, _ := m.AppendMessage(provider.NewUserMessage("next user"))
	_, _ = m.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "next assistant"}}))
	_, _ = m.AppendCompaction("## Goal\nsummary two", nextUserID, 80)

	messages := m.GetMessages()
	if len(messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(messages))
	}
	if !messages[0].SystemInjected || messages[0].Content != "## Goal\nsummary two" {
		t.Fatalf("first message = %#v, want latest summary", messages[0])
	}
	for _, msg := range messages[1:] {
		if msg.Content == "## Goal\nsummary one" || msg.Content == "recent user" {
			t.Fatalf("old compacted content still present in replay: %#v", msg)
		}
	}
	if messages[1].Content != "next user" {
		t.Fatalf("first kept message = %#v, want next user", messages[1])
	}
}

func TestOpen(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	// Create a session
	m1 := New("/tmp/test", sessionDir)
	m1.Init()
	m1.AppendMessage(provider.NewUserMessage("Hello"))

	// Open the session
	m2, err := Open(m1.file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m2 == nil {
		t.Fatal("expected non-nil manager")
	}

	if m2.header.Cwd != "/tmp/test" {
		t.Errorf("expected cwd '/tmp/test', got '%s'", m2.header.Cwd)
	}

	if len(m2.entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(m2.entries))
	}
}

func TestOpenNonExistent(t *testing.T) {
	_, err := Open("/nonexistent/path.db")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestListForDir(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	// Create sessions
	m1 := New("/tmp/test1", sessionDir)
	m1.Init()

	time.Sleep(10 * time.Millisecond)

	m2 := New("/tmp/test1", sessionDir)
	m2.Init()

	m3 := New("/tmp/test2", sessionDir)
	m3.Init()

	// List for test1
	sessions, err := ListForDir("/tmp/test1", sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}

	// List for test2
	sessions, err = ListForDir("/tmp/test2", sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}

	// List for non-existent
	sessions, err = ListForDir("/tmp/nonexistent", sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestContinueRecent(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	// Create a session
	m1 := New("/tmp/test", sessionDir)
	m1.Init()

	// Continue recent
	m2, err := ContinueRecent("/tmp/test", sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m2 == nil {
		t.Fatal("expected non-nil manager")
	}

	if m2.file != m1.file {
		t.Errorf("expected file '%s', got '%s'", m1.file, m2.file)
	}
}

func TestContinueRecentNew(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	// Continue recent for non-existing dir
	m, err := ContinueRecent("/tmp/nonexistent", sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m == nil {
		t.Fatal("expected non-nil manager")
	}

	if m.file == "" {
		t.Fatal("expected new session file")
	}
	if m.header == nil {
		t.Fatal("expected new session header")
	}
	dbPath := filepath.Join(sessionDir, "sessions.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected sessions.db to exist: %v", err)
	}
	if _, err := m.AppendMessage(provider.NewUserMessage("Hello")); err != nil {
		t.Fatalf("append message to new continued session: %v", err)
	}
}

func TestContinueRecentDefaultDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Test with empty session dir (should use default)
	m, err := ContinueRecent("/tmp/test", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestOpenByPathOrID(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m1 := New("/tmp/test", sessionDir)
	if err := m1.InitWithID("session-test-id"); err != nil {
		t.Fatalf("init session: %v", err)
	}

	byPath, err := OpenByPathOrID("/tmp/test", sessionDir, m1.file)
	if err != nil {
		t.Fatalf("open by path: %v", err)
	}
	if byPath.file != m1.file {
		t.Errorf("expected file %q, got %q", m1.file, byPath.file)
	}

	byID, err := OpenByPathOrID("/tmp/test", sessionDir, "session-test-id")
	if err != nil {
		t.Fatalf("open by id: %v", err)
	}
	if byID.file != m1.file {
		t.Errorf("expected file %q, got %q", m1.file, byID.file)
	}

	shortID := sessionFileID(m1.file)
	byShortID, err := OpenByPathOrID("/tmp/test", sessionDir, shortID)
	if err != nil {
		t.Fatalf("open by short id: %v", err)
	}
	if byShortID.file != m1.file {
		t.Errorf("expected file %q, got %q", m1.file, byShortID.file)
	}
}

func TestOpenByPathOrIDAmbiguousPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	ids := []string{"abcdef01", "abcdef02"}
	for _, id := range ids {
		m := New("/tmp/test", sessionDir)
		if err := m.InitWithID(id); err != nil {
			t.Fatalf("init session %s: %v", id, err)
		}
	}

	_, err := OpenByPathOrID("/tmp/test", sessionDir, "abc")
	if err == nil {
		t.Fatal("expected ambiguous prefix error")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("err = %q, want ambiguous", err)
	}
}

func TestOpenByIDRecreatesMissingHandle(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	if err := m.InitWithID("custom-session-123"); err != nil {
		t.Fatalf("init session: %v", err)
	}

	reopened, err := OpenByID("/tmp/test", sessionDir, "custom-session-123")
	if err != nil {
		t.Fatalf("open by ID: %v", err)
	}
	if reopened.GetHeader().ID != "custom-session-123" {
		t.Fatalf("header ID = %q", reopened.GetHeader().ID)
	}
	dbPath := filepath.Join(sessionDir, "sessions.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected sessions.db: %v", err)
	}
}

func TestOpenByIDExactIgnoresCwd(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test-a", sessionDir)
	if err := m.InitWithID("exact-session"); err != nil {
		t.Fatalf("init session: %v", err)
	}

	reopened, err := OpenByIDExact(sessionDir, "exact-session")
	if err != nil {
		t.Fatalf("open exact: %v", err)
	}
	if reopened.GetHeader() == nil || reopened.GetHeader().ID != "exact-session" {
		t.Fatalf("unexpected reopened header: %#v", reopened.GetHeader())
	}
	if reopened.GetHeader().Cwd != "/tmp/test-a" {
		t.Fatalf("cwd = %q, want /tmp/test-a", reopened.GetHeader().Cwd)
	}
}

func TestLoadRejectsCorruptSessionFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "session.db")
	// Handle file with a session ID that won't exist in any DB
	if err := os.WriteFile(path, []byte("nonexistent-session-id"), 0600); err != nil {
		t.Fatalf("write handle: %v", err)
	}

	// Should fail because there's no sessions.db with this session ID
	_, err := Open(path)
	if err == nil {
		t.Fatal("expected error for session not in DB")
	}
	if !strings.Contains(err.Error(), "not registered in DB") {
		t.Fatalf("err = %q, want 'not registered in DB'", err)
	}
}

func TestAppendEntriesMaintainParentChain(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	if err := m.Init(); err != nil {
		t.Fatalf("init session: %v", err)
	}

	firstID, err := m.AppendMessage(provider.NewUserMessage("first"))
	if err != nil {
		t.Fatalf("append first: %v", err)
	}
	secondID, err := m.AppendModelChange("openai", "model")
	if err != nil {
		t.Fatalf("append second: %v", err)
	}

	if len(m.entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(m.entries))
	}
	second, ok := m.entries[1].(ModelChangeEntry)
	if !ok {
		t.Fatalf("entry type = %T, want ModelChangeEntry", m.entries[1])
	}
	if second.ParentID == nil || *second.ParentID != firstID {
		t.Fatalf("second parent = %#v, want %s", second.ParentID, firstID)
	}
	if leaf := m.GetLeafID(); leaf == nil || *leaf != secondID {
		t.Fatalf("leaf = %#v, want %s", leaf, secondID)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()

	if id1 == "" {
		t.Error("expected non-empty ID")
	}

	if id2 == "" {
		t.Error("expected non-empty ID")
	}

	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}

func TestSessionInfo(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	// Create sessions
	m1 := New("/tmp/test", sessionDir)
	m1.Init()

	time.Sleep(10 * time.Millisecond)

	m2 := New("/tmp/test", sessionDir)
	m2.Init()

	// List and check info
	sessions, _ := ListForDir("/tmp/test", sessionDir)

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Check that sessions have required fields
	for _, s := range sessions {
		if s.Path == "" {
			t.Error("expected non-empty path")
		}
		if s.ModTime.IsZero() {
			t.Error("expected non-zero mod time")
		}
	}
}

func TestDeleteSession(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	path := m.GetFile()
	dbPath := filepath.Join(sessionDir, "sessions.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("sessions.db should exist: %v", err)
	}

	err := DeleteSession(path, sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sessions, err := ListForDir("/tmp/test", sessionDir)
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(sessions) != 0 {
		t.Error("expected 0 sessions after deletion")
	}
}

func TestDeleteSessionNonExistent(t *testing.T) {
	sessionDir := t.TempDir()
	err := DeleteSession(filepath.Join(sessionDir, "missing.db"), sessionDir)
	if err != nil {
		t.Errorf("expected no error for non-existent file, got %v", err)
	}
}

func TestDeleteSessionRejectsPathOutsideSessionDir(t *testing.T) {
	sessionDir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.db")
	if err := os.WriteFile(outside, []byte("session-id"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := DeleteSession(outside, sessionDir); err == nil {
		t.Fatal("expected outside session path to be rejected")
	}
}

func TestDeleteSessionRejectsSharedDB(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	if err := m.Init(); err != nil {
		t.Fatalf("init session: %v", err)
	}
	sharedDB := filepath.Join(sessionDir, "sessions.db")
	if _, err := os.Stat(sharedDB); err != nil {
		t.Fatalf("expected shared DB: %v", err)
	}

	if err := DeleteSession(sharedDB, sessionDir); err == nil {
		t.Fatal("expected shared DB deletion to be rejected")
	}
}

func TestListForDirDetailed(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	// Create a session with messages
	m := New("/tmp/test", sessionDir)
	m.Init()
	m.AppendMessage(provider.NewUserMessage("Hello world"))
	m.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{
		{Type: "text", Text: "Hi there"},
	}))
	m.AppendMessage(provider.NewUserMessage("Another message"))

	details, err := ListForDirDetailed("/tmp/test", sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(details) != 1 {
		t.Fatalf("expected 1 session detail, got %d", len(details))
	}

	d := details[0]
	if d.MessageCount != 3 {
		t.Errorf("expected 3 messages, got %d", d.MessageCount)
	}
	if d.Preview != "Hello world" {
		t.Errorf("expected preview 'Hello world', got %q", d.Preview)
	}
	if d.ID == "" {
		t.Error("expected non-empty ID")
	}
	if d.Cwd != "/tmp/test" {
		t.Errorf("expected cwd '/tmp/test', got %q", d.Cwd)
	}
}

func TestListAllDetailedAcrossWorkDirs(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m1 := New("/tmp/project-a", sessionDir)
	if err := m1.InitWithID("session-a"); err != nil {
		t.Fatalf("init session a: %v", err)
	}
	if _, err := m1.AppendMessage(provider.NewUserMessage("Project A task")); err != nil {
		t.Fatalf("append session a message: %v", err)
	}

	m2 := New("/tmp/project-b", sessionDir)
	if err := m2.InitWithID("session-b"); err != nil {
		t.Fatalf("init session b: %v", err)
	}
	if _, err := m2.AppendMessage(provider.NewUserMessage("Project B task")); err != nil {
		t.Fatalf("append session b message: %v", err)
	}

	details, err := ListAllDetailed(sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(details) != 2 {
		t.Fatalf("expected 2 session details, got %d", len(details))
	}

	byID := make(map[string]SessionDetail)
	for _, d := range details {
		byID[d.ID] = d
	}
	if byID["session-a"].Cwd != "/tmp/project-a" || byID["session-a"].Preview != "Project A task" {
		t.Fatalf("session-a detail = %#v", byID["session-a"])
	}
	if byID["session-b"].Cwd != "/tmp/project-b" || byID["session-b"].Preview != "Project B task" {
		t.Fatalf("session-b detail = %#v", byID["session-b"])
	}
}

func TestListForDirDetailedLongPreview(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()
	// Message longer than 60 chars
	longMsg := strings.Repeat("a", 100)
	m.AppendMessage(provider.NewUserMessage(longMsg))

	details, err := ListForDirDetailed("/tmp/test", sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(details) != 1 {
		t.Fatalf("expected 1 session, got %d", len(details))
	}

	if len(details[0].Preview) > 64 { // 60 + "..."
		t.Errorf("preview should be truncated, got length %d", len(details[0].Preview))
	}
	if !strings.HasSuffix(details[0].Preview, "...") {
		t.Error("expected truncated preview to end with '...'")
	}
}

func TestListForDirDetailedLongPreviewKeepsUTF8(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()
	longMsg := "a" + strings.Repeat("你好", 40)
	m.AppendMessage(provider.NewUserMessage(longMsg))

	details, err := ListForDirDetailed("/tmp/test", sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(details) != 1 {
		t.Fatalf("expected 1 session, got %d", len(details))
	}
	if !strings.HasSuffix(details[0].Preview, "...") {
		t.Error("expected truncated preview to end with '...'")
	}
	if !utf8.ValidString(details[0].Preview) {
		t.Fatalf("preview should be valid UTF-8, got %q", details[0].Preview)
	}
	if strings.ContainsRune(details[0].Preview, utf8.RuneError) {
		t.Fatalf("preview should not contain replacement rune, got %q", details[0].Preview)
	}
}

func TestListForDirDetailedEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	details, err := ListForDirDetailed("/tmp/nonexistent", sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(details) != 0 {
		t.Errorf("expected 0 details, got %d", len(details))
	}
}

func TestListForDirDetailedContentBlocks(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()
	// User message with content blocks (no Content field)
	m.AppendMessage(provider.Message{
		Role: "user",
		Contents: []provider.ContentBlock{
			{Type: "text", Text: "Block content"},
		},
	})

	details, err := ListForDirDetailed("/tmp/test", sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(details) != 1 {
		t.Fatalf("expected 1 session, got %d", len(details))
	}
	if details[0].Preview != "Block content" {
		t.Errorf("expected preview 'Block content', got %q", details[0].Preview)
	}
}

func TestAppendSessionInfo(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	m.Init()

	id, err := m.AppendSessionInfo("My Session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty ID")
	}
	if len(m.entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(m.entries))
	}
}

func TestEncodePath(t *testing.T) {
	// Same path should produce same encoding
	e1 := encodePath("/tmp/test")
	e2 := encodePath("/tmp/test")
	if e1 != e2 {
		t.Error("expected same encoding for same path")
	}

	// Different paths should produce different encodings
	e3 := encodePath("/tmp/test2")
	if e1 == e3 {
		t.Error("expected different encoding for different path")
	}

	// Paths that are similar but different should not collide
	e4 := encodePath("/tmp/test-1")
	e5 := encodePath("/tmp/test:1")
	if e4 == e5 {
		t.Error("expected different encoding for paths with different special chars")
	}
}

func TestInitWithID(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	err := m.InitWithID("custom-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	header := m.GetHeader()
	if header.ID != "custom-id" {
		t.Errorf("expected ID 'custom-id', got %q", header.ID)
	}
}

func TestSessionFileID(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/20240101-120000_abcd1234.db", "abcd1234"},
		{"/path/to/session.db", ""},
		{"simple_id.db", "id"},
	}

	for _, tt := range tests {
		result := sessionFileID(tt.path)
		if result != tt.expected {
			t.Errorf("sessionFileID(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestOpenByPathOrIDEmptyValue(t *testing.T) {
	_, err := OpenByPathOrID("/tmp", "/tmp/sessions", "")
	if err == nil {
		t.Error("expected error for empty value")
	}
}

func TestSessionRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	// Create session with various entry types
	m1 := New("/tmp/test", sessionDir)
	m1.Init()
	m1.AppendMessage(provider.NewUserMessage("Hello"))
	m1.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{
		{Type: "text", Text: "Hi"},
	}))
	m1.AppendModelChange("anthropic", "claude-sonnet-4-20250514")
	m1.AppendThinkingLevelChange("high")
	m1.AppendCompaction("Summary", "", 1000)
	m1.AppendSessionInfo("Test Session")

	// Re-open and verify all entries loaded
	m2, err := Open(m1.GetFile())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m2.entries) != 6 {
		t.Errorf("expected 6 entries, got %d", len(m2.entries))
	}

	msgs := m2.GetMessages()
	if len(msgs) != 1 {
		t.Errorf("expected 1 compacted message, got %d", len(msgs))
	}
	if len(msgs) == 1 && (!msgs[0].SystemInjected || msgs[0].Content != "Summary") {
		t.Errorf("expected summary-only replay, got %#v", msgs[0])
	}
}

// TestWriteEntryDurable verifies that entries are fsynced and survive reopen.
func TestWriteEntryDurable(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	m := New("/tmp/test", sessionDir)
	if err := m.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Append several messages
	for i := 0; i < 5; i++ {
		msg := provider.NewUserMessage(fmt.Sprintf("message %d", i))
		if _, err := m.AppendMessage(msg); err != nil {
			t.Fatalf("append message %d: %v", i, err)
		}
	}

	// Re-open from disk — all 5 messages + 1 header should be present
	reopened, err := Open(m.GetFile())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	loadedMsgs := reopened.GetMessages()
	if len(loadedMsgs) != 5 {
		t.Errorf("expected 5 messages after reopen, got %d", len(loadedMsgs))
	}

	// Verify content of last message
	last := loadedMsgs[4]
	if last.Content != "message 4" {
		t.Errorf("last message content = %q, want 'message 4'", last.Content)
	}
}

// TestApplyMigrationsOnOldDB verifies that creating a session on an old-format
// DB (without schema_migrations) triggers migration and creates all tables.
func TestApplyMigrationsOnOldDB(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")
	dbPath := filepath.Join(sessionDir, "sessions.db")

	// Simulate an old DB: create sessions + entries only, no schema_migrations
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		cwd TEXT,
		timestamp TEXT,
		parent_session TEXT,
		version INTEGER
	);`)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE entries (
		seq INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT REFERENCES sessions(id) ON DELETE CASCADE,
		id TEXT UNIQUE,
		type TEXT NOT NULL,
		parent_id TEXT,
		timestamp TEXT NOT NULL,
		data TEXT NOT NULL
	);`)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.Close()

	// Reset the in-memory cache so the session package treats this as a new DB
	dbLock.Lock()
	delete(initializedDBs, dbPath)
	delete(cachedDBs, dbPath)
	dbLock.Unlock()

	// Creating a new session should trigger migrations
	m := New("/tmp/test-migration", sessionDir)
	if err := m.Init(); err != nil {
		t.Fatalf("Init on old DB should trigger migration: %v", err)
	}

	// Re-open DB to verify tables exist
	db2, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()

	var migrationCount int
	if err := db2.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrationCount); err != nil {
		t.Fatalf("schema_migrations should exist: %v", err)
	}
	if migrationCount != len(migrations) {
		t.Errorf("expected %d migrations applied, got %d", len(migrations), migrationCount)
	}

	// request_stats should exist
	var tblName string
	if err := db2.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='request_stats'").Scan(&tblName); err != nil {
		t.Fatalf("request_stats table should exist after migration: %v", err)
	}
	if err := db2.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='cron_jobs'").Scan(&tblName); err != nil {
		t.Fatalf("cron_jobs table should exist after migration: %v", err)
	}
	if err := db2.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='session_esm_objectives'").Scan(&tblName); err != nil {
		t.Fatalf("session_esm_objectives table should exist after migration: %v", err)
	}

	// Old data should still work
	var entryCount int
	if err := db2.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&entryCount); err != nil {
		t.Fatalf("sessions table should still work: %v", err)
	}
	if entryCount != 1 {
		t.Errorf("expected 1 session header row after init, got %d", entryCount)
	}
}

func TestSessionCapabilitiesSaveLoadAndDelete(t *testing.T) {
	sessionDir := t.TempDir()
	m := New("/tmp/caps-project", sessionDir)
	if err := m.InitWithID("caps-session"); err != nil {
		t.Fatalf("InitWithID: %v", err)
	}

	caps := SessionCapabilities{
		SessionID:    "caps-session",
		Mode:         "agent",
		DelegateMode: true,
		MultiAgent:   true,
		Workflows:    true,
		WebSearch:    true,
		Browser:      true,
		A2AMaster:    true,
	}
	if err := SaveSessionCapabilities(sessionDir, caps); err != nil {
		t.Fatalf("SaveSessionCapabilities: %v", err)
	}

	loaded, ok, err := LoadSessionCapabilities(sessionDir, "caps-session")
	if err != nil {
		t.Fatalf("LoadSessionCapabilities: %v", err)
	}
	if !ok {
		t.Fatal("expected capabilities to be found")
	}
	if loaded.Mode != "agent" || !loaded.DelegateMode || !loaded.MultiAgent || !loaded.Workflows ||
		!loaded.WebSearch || !loaded.Browser || !loaded.A2AMaster {
		t.Fatalf("loaded capabilities = %#v", loaded)
	}

	if err := DeleteSession(m.GetFile(), sessionDir); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, ok, err := LoadSessionCapabilities(sessionDir, "caps-session"); err != nil || ok {
		t.Fatalf("capabilities after delete: ok=%v err=%v", ok, err)
	}
}

func TestSessionRunAndCapabilityEventsSaveListAndDelete(t *testing.T) {
	sessionDir := t.TempDir()
	m := New("/tmp/events-project", sessionDir)
	if err := m.InitWithID("events-session"); err != nil {
		t.Fatalf("InitWithID: %v", err)
	}

	if _, err := SaveSessionRunEvent(sessionDir, SessionRunEvent{
		SessionID: "events-session",
		RunID:     "run-1",
		EventType: "started",
		Source:    "chat_completion",
		Status:    "running",
		Model:     "m1",
		Mode:      "agent",
		Data:      []byte(`{"stream":true}`),
	}); err != nil {
		t.Fatalf("SaveSessionRunEvent started: %v", err)
	}
	if _, err := SaveSessionRunEvent(sessionDir, SessionRunEvent{
		SessionID: "events-session",
		RunID:     "run-1",
		EventType: "finished",
		Source:    "chat_completion",
		Status:    "completed",
		Data:      []byte(`{"usage":{"total_tokens":3}}`),
	}); err != nil {
		t.Fatalf("SaveSessionRunEvent finished: %v", err)
	}
	if _, err := SaveSessionCapabilityEvent(sessionDir, SessionCapabilityEvent{
		SessionID:  "events-session",
		RunID:      "run-1",
		EventType:  "changed",
		Source:     "api_patch",
		Actor:      "webui",
		Capability: "browser",
		OldValue:   "false",
		NewValue:   "true",
		Data:       []byte(`{"reason":"test"}`),
	}); err != nil {
		t.Fatalf("SaveSessionCapabilityEvent: %v", err)
	}

	runEvents, err := ListSessionRunEvents(sessionDir, "events-session")
	if err != nil {
		t.Fatalf("ListSessionRunEvents: %v", err)
	}
	if len(runEvents) != 2 {
		t.Fatalf("run events len = %d, want 2: %#v", len(runEvents), runEvents)
	}
	if runEvents[0].RunID != "run-1" || runEvents[0].EventType != "started" || runEvents[0].Status != "running" {
		t.Fatalf("first run event = %#v", runEvents[0])
	}
	if string(runEvents[0].Data) != `{"stream":true}` {
		t.Fatalf("first run event data = %s", runEvents[0].Data)
	}

	capabilityEvents, err := ListSessionCapabilityEvents(sessionDir, "events-session")
	if err != nil {
		t.Fatalf("ListSessionCapabilityEvents: %v", err)
	}
	if len(capabilityEvents) != 1 {
		t.Fatalf("capability events len = %d, want 1: %#v", len(capabilityEvents), capabilityEvents)
	}
	if capabilityEvents[0].Capability != "browser" || capabilityEvents[0].OldValue != "false" || capabilityEvents[0].NewValue != "true" {
		t.Fatalf("capability event = %#v", capabilityEvents[0])
	}

	runEventsWithSeq, err := ListSessionRunEventsWithSeq(sessionDir, "events-session")
	if err != nil {
		t.Fatalf("ListSessionRunEventsWithSeq: %v", err)
	}
	if len(runEventsWithSeq) != 2 || runEventsWithSeq[0].Seq <= 0 || runEventsWithSeq[1].Seq <= runEventsWithSeq[0].Seq {
		t.Fatalf("run events with seq = %#v", runEventsWithSeq)
	}
	runAfterFirst, err := ListSessionRunEventsAfter(sessionDir, "events-session", runEventsWithSeq[0].Seq, 10)
	if err != nil {
		t.Fatalf("ListSessionRunEventsAfter: %v", err)
	}
	if len(runAfterFirst) != 1 || runAfterFirst[0].Event.EventType != "finished" {
		t.Fatalf("run events after first = %#v", runAfterFirst)
	}

	capabilityEventsWithSeq, err := ListSessionCapabilityEventsWithSeq(sessionDir, "events-session")
	if err != nil {
		t.Fatalf("ListSessionCapabilityEventsWithSeq: %v", err)
	}
	if len(capabilityEventsWithSeq) != 1 || capabilityEventsWithSeq[0].Seq <= 0 || capabilityEventsWithSeq[0].Event.Capability != "browser" {
		t.Fatalf("capability events with seq = %#v", capabilityEventsWithSeq)
	}

	if err := DeleteSession(m.GetFile(), sessionDir); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	runEvents, err = ListSessionRunEvents(sessionDir, "events-session")
	if err != nil {
		t.Fatalf("ListSessionRunEvents after delete: %v", err)
	}
	if len(runEvents) != 0 {
		t.Fatalf("run events after delete len = %d, want 0", len(runEvents))
	}
	capabilityEvents, err = ListSessionCapabilityEvents(sessionDir, "events-session")
	if err != nil {
		t.Fatalf("ListSessionCapabilityEvents after delete: %v", err)
	}
	if len(capabilityEvents) != 0 {
		t.Fatalf("capability events after delete len = %d, want 0", len(capabilityEvents))
	}
}

func TestListSessionMessagesWithSeqAndAfter(t *testing.T) {
	sessionDir := t.TempDir()
	m := New("/tmp/seq-project", sessionDir)
	if err := m.InitWithID("seq-session"); err != nil {
		t.Fatalf("InitWithID: %v", err)
	}
	firstID, err := m.AppendMessage(provider.NewUserMessage("first"))
	if err != nil {
		t.Fatalf("AppendMessage first: %v", err)
	}
	secondID, err := m.AppendMessage(provider.Message{Role: "assistant", Content: "second"})
	if err != nil {
		t.Fatalf("AppendMessage second: %v", err)
	}

	messages, err := ListSessionMessagesWithSeq(sessionDir, "seq-session")
	if err != nil {
		t.Fatalf("ListSessionMessagesWithSeq: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("messages len = %d, want 2: %#v", len(messages), messages)
	}
	if messages[0].EntryID != firstID || messages[0].Message.Content != "first" || messages[0].Seq <= 0 {
		t.Fatalf("first sequenced message = %#v", messages[0])
	}
	if messages[1].EntryID != secondID || messages[1].Message.Content != "second" || messages[1].Seq <= messages[0].Seq {
		t.Fatalf("second sequenced message = %#v", messages[1])
	}

	afterFirst, err := ListSessionMessagesAfter(sessionDir, "seq-session", messages[0].Seq, 10)
	if err != nil {
		t.Fatalf("ListSessionMessagesAfter: %v", err)
	}
	if len(afterFirst) != 1 || afterFirst[0].EntryID != secondID {
		t.Fatalf("messages after first = %#v", afterFirst)
	}
}
