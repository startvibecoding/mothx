package session

import (
	"database/sql"
	"fmt"
	"strings"
)

const currentSchema = `
CREATE TABLE sessions (
	id TEXT PRIMARY KEY,
	cwd TEXT,
	timestamp TEXT,
	parent_session TEXT,
	version INTEGER
);
CREATE TABLE entries (
	seq INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT REFERENCES sessions(id) ON DELETE CASCADE,
	id TEXT UNIQUE,
	type TEXT NOT NULL,
	parent_id TEXT,
	timestamp TEXT NOT NULL,
	data TEXT NOT NULL
);
CREATE INDEX idx_entries_session_id ON entries(session_id);
CREATE INDEX idx_entries_type ON entries(type);
CREATE INDEX idx_sessions_cwd ON sessions(cwd);
CREATE TABLE request_stats (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	timestamp TEXT NOT NULL,
	session_id TEXT,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	input_tokens INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	duration_ms INTEGER NOT NULL DEFAULT 0,
	protocol TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_request_stats_timestamp ON request_stats(timestamp);
CREATE INDEX idx_request_stats_provider ON request_stats(provider);
CREATE INDEX idx_request_stats_model ON request_stats(model);
CREATE TABLE session_capabilities (
	session_id TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
	mode TEXT NOT NULL DEFAULT '',
	delegate_mode INTEGER NOT NULL DEFAULT 0,
	multi_agent INTEGER NOT NULL DEFAULT 0,
	workflows INTEGER NOT NULL DEFAULT 0,
	web_search INTEGER NOT NULL DEFAULT 0,
	browser INTEGER NOT NULL DEFAULT 0,
	a2a_master INTEGER NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL
);
CREATE TABLE session_run_events (
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
CREATE INDEX idx_session_run_events_session_id ON session_run_events(session_id);
CREATE INDEX idx_session_run_events_run_id ON session_run_events(run_id);
CREATE INDEX idx_session_run_events_type ON session_run_events(event_type);
CREATE TABLE session_capability_events (
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
CREATE INDEX idx_session_capability_events_session_id ON session_capability_events(session_id);
CREATE INDEX idx_session_capability_events_run_id ON session_capability_events(run_id);
CREATE INDEX idx_session_capability_events_capability ON session_capability_events(capability);
CREATE TABLE cron_jobs (
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
CREATE INDEX idx_cron_jobs_session_id ON cron_jobs(session_id);
CREATE INDEX idx_cron_jobs_enabled ON cron_jobs(enabled);
CREATE INDEX idx_cron_jobs_next_run ON cron_jobs(next_run);
CREATE INDEX idx_cron_jobs_created_at ON cron_jobs(created_at);
CREATE TABLE session_esm_objectives (
	session_id TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
	esm_id TEXT NOT NULL,
	objective TEXT NOT NULL,
	status TEXT NOT NULL,
	token_budget INTEGER,
	tokens_used INTEGER NOT NULL DEFAULT 0,
	time_used_ms INTEGER NOT NULL DEFAULT 0,
	blocked_count INTEGER NOT NULL DEFAULT 0,
	blocked_reason TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	blocked_run_id TEXT NOT NULL DEFAULT '',
	completion_reason TEXT NOT NULL DEFAULT '',
	completion_run_id TEXT NOT NULL DEFAULT '',
	completion_review TEXT NOT NULL DEFAULT '',
	phase TEXT NOT NULL DEFAULT '',
	progress_summary TEXT NOT NULL DEFAULT '',
	remaining_work TEXT NOT NULL DEFAULT '[]',
	completion_rejection_count INTEGER NOT NULL DEFAULT 0,
	completion_rejection_run_id TEXT NOT NULL DEFAULT '',
	recovery_count INTEGER NOT NULL DEFAULT 0,
	recovery_reason TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_session_esm_objectives_status ON session_esm_objectives(status);
CREATE TABLE sub_session (
	id TEXT PRIMARY KEY,
	cwd TEXT,
	timestamp TEXT,
	parent_session TEXT,
	version INTEGER
);
CREATE TABLE sub_entries (
	seq INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT REFERENCES sub_session(id) ON DELETE CASCADE,
	id TEXT UNIQUE,
	type TEXT NOT NULL,
	parent_id TEXT,
	timestamp TEXT NOT NULL,
	data TEXT NOT NULL
);
CREATE INDEX idx_sub_entries_session_id ON sub_entries(session_id);
CREATE INDEX idx_sub_entries_type ON sub_entries(type);
CREATE INDEX idx_sub_session_cwd ON sub_session(cwd);
`

var requiredSchema = map[string][]string{
	"sessions":                  {"id", "cwd", "timestamp", "parent_session", "version"},
	"entries":                   {"seq", "session_id", "id", "type", "parent_id", "timestamp", "data"},
	"request_stats":             {"id", "timestamp", "session_id", "provider", "protocol", "model", "input_tokens", "output_tokens", "total_tokens", "duration_ms"},
	"session_capabilities":      {"session_id", "mode", "delegate_mode", "multi_agent", "workflows", "web_search", "browser", "a2a_master", "updated_at"},
	"session_run_events":        {"seq", "id", "session_id", "run_id", "event_type", "source", "status", "model", "mode", "timestamp", "data"},
	"session_capability_events": {"seq", "id", "session_id", "run_id", "event_type", "source", "actor", "capability", "old_value", "new_value", "timestamp", "data"},
	"cron_jobs":                 {"id", "session_id", "name", "prompt", "schedule", "oneshot", "mode", "work_dir", "a2a_target", "a2a_token", "enabled", "created_at", "last_run", "next_run", "run_count", "last_status", "last_error"},
	"session_esm_objectives":    {"session_id", "esm_id", "objective", "status", "token_budget", "tokens_used", "time_used_ms", "blocked_count", "blocked_reason", "created_at", "updated_at", "blocked_run_id", "completion_reason", "completion_run_id", "completion_review", "phase", "progress_summary", "remaining_work", "completion_rejection_count", "completion_rejection_run_id", "recovery_count", "recovery_reason"},
	"sub_session":               {"id", "cwd", "timestamp", "parent_session", "version"},
	"sub_entries":               {"seq", "session_id", "id", "type", "parent_id", "timestamp", "data"},
}

// EnsureCurrentSchema creates the current schema only for an empty database.
// Existing databases are validated but never migrated or otherwise modified.
func EnsureCurrentSchema(db *sql.DB) error {
	var tableCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations'`).Scan(&tableCount); err != nil {
		return fmt.Errorf("inspect database schema: %w", err)
	}
	if tableCount == 0 {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin schema initialization: %w", err)
		}
		if err := tx.QueryRow(`SELECT COUNT(*) FROM sqlite_master
			WHERE type = 'table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations'`).Scan(&tableCount); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("recheck database schema: %w", err)
		}
		if tableCount == 0 {
			if _, err := tx.Exec(currentSchema); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("initialize database schema: %w", err)
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit database schema: %w", err)
		}
	}

	for table, requiredColumns := range requiredSchema {
		rows, err := db.Query("PRAGMA table_info(" + table + ")")
		if err != nil {
			return fmt.Errorf("inspect table %s: %w", table, err)
		}
		columns := make(map[string]bool)
		for rows.Next() {
			var cid, notNull, primaryKey int
			var name, columnType string
			var defaultValue any
			if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				rows.Close()
				return fmt.Errorf("inspect table %s: %w", table, err)
			}
			columns[name] = true
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("inspect table %s: %w", table, err)
		}
		var missing []string
		for _, column := range requiredColumns {
			if !columns[column] {
				missing = append(missing, column)
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("database schema is incompatible: table %s is missing columns %s", table, strings.Join(missing, ", "))
		}
	}
	return nil
}
