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
