package stats

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func createTestDB(t *testing.T) *DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sessions.db")

	// Create a minimal db file (simulates an old DB without request_stats)
	f, err := os.Create(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	// Open auto-creates request_stats table via ensureTable()
	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSummary(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	now := time.Now().Format(time.RFC3339Nano)
	_, err := db.db.Exec(
		"INSERT INTO request_stats (timestamp, session_id, provider, protocol, model, input_tokens, output_tokens, total_tokens, duration_ms) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		now, "sess1", "openai", "openai-chat", "gpt-4", 1000, 500, 1500, 2000,
	)
	if err != nil {
		t.Fatal(err)
	}

	summary, err := db.Summary(Query{})
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalRequests != 1 {
		t.Errorf("expected 1 request, got %d", summary.TotalRequests)
	}
	if summary.InputTokens != 1000 {
		t.Errorf("expected 1000 input tokens, got %d", summary.InputTokens)
	}
	if summary.OutputTokens != 500 {
		t.Errorf("expected 500 output tokens, got %d", summary.OutputTokens)
	}
}

func TestTimeSeries(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	for i := 0; i < 3; i++ {
		ts := time.Date(2026, 6, 28+i, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
		_, err := db.db.Exec(
			"INSERT INTO request_stats (timestamp, session_id, provider, protocol, model, input_tokens, output_tokens, total_tokens, duration_ms) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			ts, "sess1", "openai", "openai-chat", "gpt-4", 100*(i+1), 50*(i+1), 150*(i+1), 1000,
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	data, err := db.TimeSeries(Query{GroupBy: "day"})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 3 {
		t.Errorf("expected 3 data points, got %d", len(data))
	}
}

func TestTimeSeriesOneHour(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	rows := []struct {
		ts          time.Time
		totalTokens int
	}{
		{time.Date(2026, 6, 28, 12, 40, 0, 0, time.UTC), 100},
		{time.Date(2026, 6, 28, 14, 59, 59, 0, time.UTC), 200},
		{time.Date(2026, 6, 28, 15, 0, 0, 0, time.UTC), 300},
	}
	for _, row := range rows {
		_, err := db.db.Exec(
			"INSERT INTO request_stats (timestamp, session_id, provider, protocol, model, input_tokens, output_tokens, total_tokens, duration_ms) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			row.ts.Format(time.RFC3339Nano), "sess1", "openai", "openai-chat", "gpt-4", row.totalTokens, 0, row.totalTokens, 1000,
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	data, err := db.TimeSeries(Query{GroupBy: "1h"})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 3 {
		t.Fatalf("expected 3 data points, got %d", len(data))
	}
	if data[0].Label != "2026-06-28 12:00" || data[0].TotalTokens != 100 {
		t.Errorf("expected first bucket 2026-06-28 12:00 with 100 tokens, got %s with %d", data[0].Label, data[0].TotalTokens)
	}
	if data[1].Label != "2026-06-28 14:00" || data[1].TotalTokens != 200 {
		t.Errorf("expected second bucket 2026-06-28 14:00 with 200 tokens, got %s with %d", data[1].Label, data[1].TotalTokens)
	}
	if data[2].Label != "2026-06-28 15:00" || data[2].TotalTokens != 300 {
		t.Errorf("expected third bucket 2026-06-28 15:00 with 300 tokens, got %s with %d", data[2].Label, data[2].TotalTokens)
	}
}

func TestByProvider(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	now := time.Now().Format(time.RFC3339Nano)
	_, _ = db.db.Exec("INSERT INTO request_stats (timestamp, provider, protocol, model, input_tokens, output_tokens, total_tokens) VALUES (?, ?, ?, ?, ?, ?, ?)",
		now, "openai", "openai-chat", "gpt-4", 1000, 500, 1500)
	_, _ = db.db.Exec("INSERT INTO request_stats (timestamp, provider, protocol, model, input_tokens, output_tokens, total_tokens) VALUES (?, ?, ?, ?, ?, ?, ?)",
		now, "anthropic", "anthropic-messages", "claude-3", 2000, 800, 2800)

	data, err := db.ByProvider(Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 2 {
		t.Errorf("expected 2 providers, got %d", len(data))
	}
	// anthropic should be first (higher total tokens)
	if data[0].Vendor != "anthropic" {
		t.Errorf("expected anthropic first, got %s", data[0].Vendor)
	}
	if data[0].Protocol != "anthropic-messages" {
		t.Errorf("expected anthropic-messages protocol, got %s", data[0].Protocol)
	}
}

func TestByModel(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	now := time.Now().Format(time.RFC3339Nano)
	_, _ = db.db.Exec("INSERT INTO request_stats (timestamp, provider, protocol, model, input_tokens, output_tokens, total_tokens) VALUES (?, ?, ?, ?, ?, ?, ?)",
		now, "openai", "openai-chat", "gpt-4", 1000, 500, 1500)
	_, _ = db.db.Exec("INSERT INTO request_stats (timestamp, provider, protocol, model, input_tokens, output_tokens, total_tokens) VALUES (?, ?, ?, ?, ?, ?, ?)",
		now, "openai", "openai-chat", "gpt-3.5", 500, 200, 700)

	data, err := db.ByModel(Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 2 {
		t.Errorf("expected 2 models, got %d", len(data))
	}
}

func TestRecent(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	now := time.Now().Format(time.RFC3339Nano)
	for i := 0; i < 5; i++ {
		_, err := db.db.Exec("INSERT INTO request_stats (timestamp, provider, protocol, model, input_tokens, output_tokens, total_tokens) VALUES (?, ?, ?, ?, ?, ?, ?)",
			now, "openai", "openai-chat", "gpt-4", 100, 50, 150)
		if err != nil {
			t.Fatal(err)
		}
	}

	page, err := db.Recent(1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 3 {
		t.Errorf("expected 3 recent entries, got %d", len(page.Items))
	}
	if page.Total != 5 {
		t.Errorf("expected total 5, got %d", page.Total)
	}
	if page.Page != 1 {
		t.Errorf("expected page 1, got %d", page.Page)
	}
	if page.PageSize != 3 {
		t.Errorf("expected pageSize 3, got %d", page.PageSize)
	}
}

func TestRecentFiltered(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	_, _ = db.db.Exec("INSERT INTO request_stats (timestamp, provider, protocol, model, input_tokens, output_tokens, total_tokens) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"2026-07-02T10:00:00Z", "openai", "openai-chat", "gpt-4", 100, 50, 150)
	_, _ = db.db.Exec("INSERT INTO request_stats (timestamp, provider, protocol, model, input_tokens, output_tokens, total_tokens) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"2026-07-03T10:00:00Z", "moark", "openai-chat", "qwen3.6-plus", 200, 100, 300)

	page, err := db.RecentFiltered(Query{
		From:   time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
		To:     time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC),
		Vendor: "moark",
	}, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 {
		t.Fatalf("expected total 1, got %d", page.Total)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Items))
	}
	if page.Items[0].Vendor != "moark" || page.Items[0].Model != "qwen3.6-plus" {
		t.Errorf("unexpected item: %+v", page.Items[0])
	}
}

// TestMigrationFromOldDB verifies that opening a DB created with an older
// schema (no schema_migrations table, no request_stats table) auto-migrates
// and works correctly.
func TestMigrationFromOldDB(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sessions.db")

	// Create a DB simulating an old vibecoding version:
	// only sessions + entries, no schema_migrations, no request_stats.
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
		t.Fatal(err)
	}
	db.Close()

	// Open with stats.Open — triggers ApplyMigrations
	sdb, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open should migrate old DB, got error: %v", err)
	}
	defer sdb.Close()

	// schema_migrations table should exist and record all migrations
	var migrationCount int
	err = sdb.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrationCount)
	if err != nil {
		t.Fatalf("schema_migrations table should exist after migration: %v", err)
	}
	if migrationCount != 8 {
		t.Errorf("expected 8 migrations recorded, got %d", migrationCount)
	}

	// Verify a specific migration was recorded
	var appliedAt string
	err = sdb.db.QueryRow("SELECT applied_at FROM schema_migrations WHERE name = '004_create_request_stats_table'").Scan(&appliedAt)
	if err != nil {
		t.Fatalf("migration 004 should be recorded: %v", err)
	}
	if appliedAt == "" {
		t.Error("migration 004 should have applied_at timestamp")
	}

	// request_stats table should now exist and be usable
	_, err = sdb.db.Exec("INSERT INTO request_stats (timestamp, provider, protocol, model, input_tokens, output_tokens, total_tokens) VALUES (?, ?, ?, ?, ?, ?, ?)",
		time.Now().Format(time.RFC3339Nano), "anthropic", "anthropic-messages", "claude-3", 500, 200, 700)
	if err != nil {
		t.Fatal(err)
	}

	summary, err := sdb.Summary(Query{})
	if err != nil {
		t.Fatalf("Summary on migrated DB failed: %v", err)
	}
	if summary.TotalRequests != 1 {
		t.Errorf("expected 1 request after migration, got %d", summary.TotalRequests)
	}
	if summary.InputTokens != 500 {
		t.Errorf("expected 500 input tokens, got %d", summary.InputTokens)
	}

	// Old tables should still be intact
	var sessionCount int
	err = sdb.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&sessionCount)
	if err != nil {
		t.Fatalf("Old sessions table should still exist: %v", err)
	}
}

// TestIdempotentMigrations verifies that calling ApplyMigrations multiple
// times is safe and does not re-apply already-applied migrations.
func TestIdempotentMigrations(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sessions.db")

	// Create an empty file (Open requires the file to exist)
	f, err := os.Create(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	// First open — runs all migrations
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	var count1 int
	db1.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count1)
	if count1 != 8 {
		t.Errorf("expected 8 migrations after first open, got %d", count1)
	}
	db1.Close()

	// Second open — should be a no-op
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()

	var count2 int
	db2.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count2)
	if count2 != 8 {
		t.Errorf("expected 8 migrations after second open (no re-apply), got %d", count2)
	}
}
