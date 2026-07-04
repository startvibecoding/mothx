package stats

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/startvibecoding/mothx/internal/platform"
	"github.com/startvibecoding/mothx/internal/session"
	_ "modernc.org/sqlite"
)

// StatsEntry represents a single recorded LLM request.
type StatsEntry struct {
	ID           int64     `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	SessionID    string    `json:"sessionId"`
	Vendor       string    `json:"vendor"`
	Protocol     string    `json:"protocol"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"inputTokens"`
	OutputTokens int       `json:"outputTokens"`
	TotalTokens  int       `json:"totalTokens"`
	DurationMs   int       `json:"durationMs"`
}

// Aggregate represents aggregated stats for a dimension.
type Aggregate struct {
	Label        string `json:"label"`
	Vendor       string `json:"vendor"`
	Protocol     string `json:"protocol"`
	Model        string `json:"model"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
	TotalTokens  int    `json:"totalTokens"`
	Requests     int    `json:"requests"`
}

// Summary represents overall statistics summary.
type Summary struct {
	TotalRequests int `json:"totalRequests"`
	InputTokens   int `json:"inputTokens"`
	OutputTokens  int `json:"outputTokens"`
	TotalTokens   int `json:"totalTokens"`
}

// Query represents a stats query with filters.
type Query struct {
	From     time.Time
	To       time.Time
	Vendor   string
	Protocol string
	Model    string
	GroupBy  string // "day", "1h", "week", "month", "provider", "model"
}

// DB wraps a SQLite connection for stats queries.
type DB struct {
	db *sql.DB
}

// Open opens the stats database at the given sessions.db path.
// It runs schema migrations to ensure all required tables exist
// (e.g. when opening an old DB from a previous vibecoding version).
func Open(dbPath string) (*DB, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database not found: %s", dbPath)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	_, _ = db.Exec("PRAGMA busy_timeout = 10000;")
	if err := session.ApplyMigrations(db); err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	return &DB{db: db}, nil
}

// OpenDefault opens the default sessions.db in the user's config directory.
func OpenDefault() (*DB, error) {
	dbPath := filepath.Join(platform.SessionDir(), "sessions.db")
	return Open(dbPath)
}

// Close closes the database connection.
func (s *DB) Close() error {
	return s.db.Close()
}

// Summary returns overall summary statistics for the given query.
func (s *DB) Summary(q Query) (*Summary, error) {
	where, args := buildWhereClause(q)
	row := s.db.QueryRow(fmt.Sprintf(
		"SELECT COUNT(*), COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0), COALESCE(SUM(total_tokens),0) FROM request_stats%s",
		where,
	), args...)
	var sum Summary
	err := row.Scan(&sum.TotalRequests, &sum.InputTokens, &sum.OutputTokens, &sum.TotalTokens)
	if err != nil {
		return nil, err
	}
	return &sum, nil
}

// TimeSeries returns time-bucketed stats for charting.
func (s *DB) TimeSeries(q Query) ([]Aggregate, error) {
	where, args := buildWhereClause(q)
	var bucketSQL string
	switch q.GroupBy {
	case "1h":
		bucketSQL = oneHourBucketSQL()
	case "week":
		bucketSQL = "substr(timestamp, 1, 4) || '-W' || substr(timestamp, 6, 2) || '-' || substr(timestamp, 9, 2)"
	case "month":
		bucketSQL = "substr(timestamp, 1, 7)"
	default: // day
		bucketSQL = "substr(timestamp, 1, 10)"
	}

	query := fmt.Sprintf(
		"SELECT %s AS bucket, COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0), COALESCE(SUM(total_tokens),0), COUNT(*) FROM request_stats%s GROUP BY bucket ORDER BY bucket",
		bucketSQL, where,
	)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Aggregate
	for rows.Next() {
		var a Aggregate
		if err := rows.Scan(&a.Label, &a.InputTokens, &a.OutputTokens, &a.TotalTokens, &a.Requests); err != nil {
			continue
		}
		results = append(results, a)
	}
	return results, rows.Err()
}

func oneHourBucketSQL() string {
	return "substr(timestamp, 1, 10) || ' ' || substr(timestamp, 12, 2) || ':00'"
}

// ByProvider returns stats grouped by vendor and protocol.
func (s *DB) ByProvider(q Query) ([]Aggregate, error) {
	where, args := buildWhereClause(q)
	query := fmt.Sprintf(
		"SELECT provider, protocol, COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0), COALESCE(SUM(total_tokens),0), COUNT(*) FROM request_stats%s GROUP BY provider, protocol ORDER BY total_tokens DESC",
		where,
	)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Aggregate
	for rows.Next() {
		var a Aggregate
		if err := rows.Scan(&a.Vendor, &a.Protocol, &a.InputTokens, &a.OutputTokens, &a.TotalTokens, &a.Requests); err != nil {
			continue
		}
		a.Label = a.Vendor
		results = append(results, a)
	}
	return results, rows.Err()
}

// ByModel returns stats grouped by model.
func (s *DB) ByModel(q Query) ([]Aggregate, error) {
	where, args := buildWhereClause(q)
	query := fmt.Sprintf(
		"SELECT model, provider, protocol, COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0), COALESCE(SUM(total_tokens),0), COUNT(*) FROM request_stats%s GROUP BY model, provider, protocol ORDER BY total_tokens DESC",
		where,
	)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Aggregate
	for rows.Next() {
		var a Aggregate
		if err := rows.Scan(&a.Model, &a.Vendor, &a.Protocol, &a.InputTokens, &a.OutputTokens, &a.TotalTokens, &a.Requests); err != nil {
			continue
		}
		a.Label = a.Model
		results = append(results, a)
	}
	return results, rows.Err()
}

// RecentPage represents a paginated result of recent stats entries.
type RecentPage struct {
	Items    []StatsEntry `json:"items"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

// Recent returns a paginated list of stats entries, ordered by most recent first.
func (s *DB) Recent(page, pageSize int) (*RecentPage, error) {
	return s.RecentFiltered(Query{}, page, pageSize)
}

// RecentFiltered returns a paginated list of stats entries matching the query,
// ordered by most recent first.
func (s *DB) RecentFiltered(q Query, page, pageSize int) (*RecentPage, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	where, args := buildWhereClause(q)

	// Get total count
	var total int
	if err := s.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM request_stats%s", where), args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	queryArgs := append(append([]interface{}{}, args...), pageSize, offset)
	rows, err := s.db.Query(
		fmt.Sprintf("SELECT id, timestamp, session_id, provider, protocol, model, input_tokens, output_tokens, total_tokens, duration_ms FROM request_stats%s ORDER BY id DESC LIMIT ? OFFSET ?", where),
		queryArgs...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []StatsEntry
	for rows.Next() {
		var e StatsEntry
		var ts string
		var sessionID sql.NullString
		if err := rows.Scan(&e.ID, &ts, &sessionID, &e.Vendor, &e.Protocol, &e.Model, &e.InputTokens, &e.OutputTokens, &e.TotalTokens, &e.DurationMs); err != nil {
			continue
		}
		if sessionID.Valid {
			e.SessionID = sessionID.String
		}
		e.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		if e.Timestamp.IsZero() {
			e.Timestamp, _ = time.Parse(time.RFC3339, ts)
		}
		results = append(results, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &RecentPage{Items: results, Total: total, Page: page, PageSize: pageSize}, nil
}

func buildWhereClause(q Query) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	if !q.From.IsZero() {
		clauses = append(clauses, "timestamp >= ?")
		args = append(args, q.From.Format(time.RFC3339Nano))
	}
	if !q.To.IsZero() {
		clauses = append(clauses, "timestamp < ?")
		args = append(args, q.To.Format(time.RFC3339Nano))
	}
	if q.Vendor != "" {
		clauses = append(clauses, "provider = ?")
		args = append(args, q.Vendor)
	}
	if q.Protocol != "" {
		clauses = append(clauses, "protocol = ?")
		args = append(args, q.Protocol)
	}
	if q.Model != "" {
		clauses = append(clauses, "model = ?")
		args = append(args, q.Model)
	}

	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + clauses[0]
		for _, c := range clauses[1:] {
			where += " AND " + c
		}
	}
	return where, args
}
