package session

import (
	"database/sql"
	"fmt"
	"time"
)

// migration represents a single schema migration.
type migration struct {
	Name string
	SQL  string
}

// schemaMigrationsTable creates the migration tracking table.
const schemaMigrationsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
	name TEXT PRIMARY KEY,
	applied_at TEXT NOT NULL
);`

// migrations lists all schema migrations in order.
// Each migration is idempotent and safe to skip if already applied.
var migrations = []migration{
	{
		Name: "001_create_sessions_table",
		SQL: `CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			cwd TEXT,
			timestamp TEXT,
			parent_session TEXT,
			version INTEGER
		);`,
	},
	{
		Name: "002_create_entries_table",
		SQL: `CREATE TABLE IF NOT EXISTS entries (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT REFERENCES sessions(id) ON DELETE CASCADE,
			id TEXT UNIQUE,
			type TEXT NOT NULL,
			parent_id TEXT,
			timestamp TEXT NOT NULL,
			data TEXT NOT NULL
		);`,
	},
	{
		Name: "003_create_entries_indexes",
		SQL: `CREATE INDEX IF NOT EXISTS idx_entries_session_id ON entries(session_id);
		       CREATE INDEX IF NOT EXISTS idx_entries_type ON entries(type);
		       CREATE INDEX IF NOT EXISTS idx_sessions_cwd ON sessions(cwd);`,
	},
	{
		Name: "004_create_request_stats_table",
		SQL: `CREATE TABLE IF NOT EXISTS request_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			session_id TEXT,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			input_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			total_tokens INTEGER NOT NULL DEFAULT 0,
			duration_ms INTEGER NOT NULL DEFAULT 0
		);`,
	},
	{
		Name: "005_create_request_stats_indexes",
		SQL: `CREATE INDEX IF NOT EXISTS idx_request_stats_timestamp ON request_stats(timestamp);
		       CREATE INDEX IF NOT EXISTS idx_request_stats_provider ON request_stats(provider);
		       CREATE INDEX IF NOT EXISTS idx_request_stats_model ON request_stats(model);`,
	},
	{
		Name: "006_add_request_stats_protocol_column",
		SQL:  `ALTER TABLE request_stats ADD COLUMN protocol TEXT NOT NULL DEFAULT '';`,
	},
	{
		Name: "007_create_session_capabilities_table",
		SQL: `CREATE TABLE IF NOT EXISTS session_capabilities (
			session_id TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
			mode TEXT NOT NULL DEFAULT '',
			delegate_mode INTEGER NOT NULL DEFAULT 0,
			multi_agent INTEGER NOT NULL DEFAULT 0,
			workflows INTEGER NOT NULL DEFAULT 0,
			web_search INTEGER NOT NULL DEFAULT 0,
			browser INTEGER NOT NULL DEFAULT 0,
			a2a_master INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL
		);`,
	},
	{
		Name: "008_create_session_event_tables",
		SQL: `CREATE TABLE IF NOT EXISTS session_run_events (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			id TEXT UNIQUE NOT NULL,
			session_id TEXT REFERENCES sessions(id) ON DELETE CASCADE,
			run_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			mode TEXT NOT NULL DEFAULT '',
			timestamp TEXT NOT NULL,
			data TEXT NOT NULL DEFAULT '{}'
		);
		CREATE INDEX IF NOT EXISTS idx_session_run_events_session_id ON session_run_events(session_id);
		CREATE INDEX IF NOT EXISTS idx_session_run_events_run_id ON session_run_events(run_id);
		CREATE INDEX IF NOT EXISTS idx_session_run_events_type ON session_run_events(event_type);
		CREATE TABLE IF NOT EXISTS session_capability_events (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			id TEXT UNIQUE NOT NULL,
			session_id TEXT REFERENCES sessions(id) ON DELETE CASCADE,
			run_id TEXT NOT NULL DEFAULT '',
			event_type TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT '',
			actor TEXT NOT NULL DEFAULT '',
			capability TEXT NOT NULL,
			old_value TEXT NOT NULL DEFAULT '',
			new_value TEXT NOT NULL DEFAULT '',
			timestamp TEXT NOT NULL,
			data TEXT NOT NULL DEFAULT '{}'
		);
		CREATE INDEX IF NOT EXISTS idx_session_capability_events_session_id ON session_capability_events(session_id);
		CREATE INDEX IF NOT EXISTS idx_session_capability_events_run_id ON session_capability_events(run_id);
		CREATE INDEX IF NOT EXISTS idx_session_capability_events_capability ON session_capability_events(capability);`,
	},
	{
		Name: "009_create_cron_jobs_table",
		SQL: `CREATE TABLE IF NOT EXISTS cron_jobs (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL DEFAULT '',
			prompt TEXT NOT NULL DEFAULT '',
			schedule TEXT NOT NULL DEFAULT '',
			oneshot INTEGER NOT NULL DEFAULT 0,
			mode TEXT NOT NULL DEFAULT 'yolo',
			work_dir TEXT NOT NULL DEFAULT '',
			a2a_target TEXT NOT NULL DEFAULT '',
			a2a_token TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			last_run TEXT NOT NULL DEFAULT '',
			next_run TEXT NOT NULL DEFAULT '',
			run_count INTEGER NOT NULL DEFAULT 0,
			last_status TEXT NOT NULL DEFAULT '',
			last_error TEXT NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_session_id ON cron_jobs(session_id);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_enabled ON cron_jobs(enabled);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_next_run ON cron_jobs(next_run);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_created_at ON cron_jobs(created_at);`,
	},
}

// ensureSchemaMigrations creates the schema_migrations tracking table if it doesn't exist.
func ensureSchemaMigrations(db *sql.DB) error {
	_, err := db.Exec(schemaMigrationsTable)
	return err
}

// isMigrationApplied checks whether a named migration has already been recorded.
func isMigrationApplied(db *sql.DB, name string) (bool, error) {
	var dummy string
	err := db.QueryRow("SELECT name FROM schema_migrations WHERE name = ?", name).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ApplyMigrations runs any pending migrations in order.
// Already-applied migrations are skipped based on the schema_migrations table.
// It is safe to call on every DB open — it is a no-op if all migrations are applied.
func ApplyMigrations(db *sql.DB) error {
	if err := ensureSchemaMigrations(db); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	for _, m := range migrations {
		applied, err := isMigrationApplied(db, m.Name)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", m.Name, err)
		}
		if applied {
			continue
		}

		if _, err := db.Exec(m.SQL); err != nil {
			return fmt.Errorf("apply migration %s: %w", m.Name, err)
		}

		_, err = db.Exec(
			"INSERT INTO schema_migrations (name, applied_at) VALUES (?, ?)",
			m.Name, time.Now().UTC().Format(time.RFC3339Nano),
		)
		if err != nil {
			return fmt.Errorf("record migration %s: %w", m.Name, err)
		}
	}
	return nil
}
