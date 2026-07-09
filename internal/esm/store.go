package esm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/startvibecoding/mothx/internal/session"
)

var (
	ErrNotFound          = errors.New("esm objective not found")
	ErrObjectiveExists   = errors.New("esm objective already exists")
	ErrInvalidObjective  = errors.New("esm objective cannot be empty")
	ErrInvalidTransition = errors.New("invalid esm status transition")
	ErrBudgetStillHit    = errors.New("esm token budget is still exhausted")
)

// Store persists Enable Supervisor Mode state in the shared sessions database.
type Store struct {
	sessionDir string
	now        func() time.Time
}

// NewStore returns a store backed by the root sessions.db under sessionDir.
func NewStore(sessionDir string) *Store {
	return &Store{
		sessionDir: sessionDir,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Store) db() (*sql.DB, error) {
	return session.OpenRootDB(s.sessionDir)
}

type rowQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getObjective(ctx context.Context, q rowQueryer, sessionID string) (*Objective, error) {
	row := q.QueryRowContext(ctx, `SELECT session_id, esm_id, objective, status, token_budget, tokens_used, time_used_ms, blocked_count, blocked_reason, created_at, updated_at
		FROM session_esm_objectives WHERE session_id = ?`, sessionID)
	return scanObjective(row)
}

func scanObjective(row *sql.Row) (*Objective, error) {
	var obj Objective
	var budget sql.NullInt64
	var created, updated string
	if err := row.Scan(&obj.SessionID, &obj.ESMID, &obj.Objective, &obj.Status, &budget, &obj.TokensUsed, &obj.TimeUsedMS, &obj.BlockedCount, &obj.BlockedReason, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if budget.Valid {
		v := budget.Int64
		obj.TokenBudget = &v
	}
	obj.CreatedAt = parseTime(created)
	obj.UpdatedAt = parseTime(updated)
	return &obj, nil
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t
	}
	return time.Time{}
}

func (s *Store) timestamp() string {
	return s.now().UTC().Format(time.RFC3339Nano)
}

// Get returns the current objective for a session.
func (s *Store) Get(ctx context.Context, sessionID string) (*Objective, error) {
	if sessionID == "" {
		return nil, ErrNotFound
	}
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	return getObjective(ctx, db, sessionID)
}

// Create creates a new objective. A completed row may be replaced; unfinished
// objectives must be edited or cleared explicitly.
func (s *Store) Create(ctx context.Context, sessionID, objective string, budget *int64) (*Objective, error) {
	objective = strings.TrimSpace(objective)
	if sessionID == "" {
		return nil, ErrNotFound
	}
	if objective == "" {
		return nil, ErrInvalidObjective
	}
	if budget != nil && *budget <= 0 {
		return nil, fmt.Errorf("token budget must be positive")
	}
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	existing, err := getObjective(ctx, tx, sessionID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		if IsUnfinishedStatus(existing.Status) {
			return existing, ErrObjectiveExists
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM session_esm_objectives WHERE session_id = ?`, sessionID); err != nil {
			return nil, err
		}
	}

	now := s.timestamp()
	var budgetValue any
	if budget != nil {
		budgetValue = *budget
	}
	esmID := "esm-" + session.GenerateID()
	if _, err := tx.ExecContext(ctx, `INSERT INTO session_esm_objectives
		(session_id, esm_id, objective, status, token_budget, tokens_used, time_used_ms, blocked_count, blocked_reason, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 0, 0, 0, '', ?, ?)`,
		sessionID, esmID, objective, StatusActive, budgetValue, now, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

// Edit updates the objective text for an unfinished objective.
func (s *Store) Edit(ctx context.Context, sessionID, objective string) (*Objective, error) {
	objective = strings.TrimSpace(objective)
	if objective == "" {
		return nil, ErrInvalidObjective
	}
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	current, err := getObjective(ctx, db, sessionID)
	if err != nil {
		return nil, err
	}
	if !IsUnfinishedStatus(current.Status) {
		return current, ErrInvalidTransition
	}
	if _, err := db.ExecContext(ctx, `UPDATE session_esm_objectives
		SET objective = ?, blocked_count = 0, blocked_reason = '', updated_at = ?
		WHERE session_id = ?`, objective, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

// Clear deletes the objective for a session.
func (s *Store) Clear(ctx context.Context, sessionID string) error {
	db, err := s.db()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `DELETE FROM session_esm_objectives WHERE session_id = ?`, sessionID)
	return err
}

// Pause disables idle continuation for an unfinished objective.
func (s *Store) Pause(ctx context.Context, sessionID string) (*Objective, error) {
	return s.setUserStatus(ctx, sessionID, StatusPaused)
}

// MarkUsageLimited records a runtime/provider limit and stops continuation.
func (s *Store) MarkUsageLimited(ctx context.Context, sessionID string) (*Objective, error) {
	return s.setRuntimeStatus(ctx, sessionID, StatusUsageLimited)
}

func (s *Store) setUserStatus(ctx context.Context, sessionID string, status Status) (*Objective, error) {
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	current, err := getObjective(ctx, db, sessionID)
	if err != nil {
		return nil, err
	}
	if !IsUnfinishedStatus(current.Status) {
		return current, ErrInvalidTransition
	}
	if _, err := db.ExecContext(ctx, `UPDATE session_esm_objectives
		SET status = ?, updated_at = ?
		WHERE session_id = ?`, status, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

func (s *Store) setRuntimeStatus(ctx context.Context, sessionID string, status Status) (*Objective, error) {
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	current, err := getObjective(ctx, db, sessionID)
	if err != nil {
		return nil, err
	}
	if current.Status != StatusActive {
		return current, nil
	}
	if _, err := db.ExecContext(ctx, `UPDATE session_esm_objectives
		SET status = ?, updated_at = ?
		WHERE session_id = ?`, status, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

// Resume returns paused/blocked/limited objectives to active when allowed.
func (s *Store) Resume(ctx context.Context, sessionID string) (*Objective, error) {
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	current, err := getObjective(ctx, db, sessionID)
	if err != nil {
		return nil, err
	}
	switch current.Status {
	case StatusActive:
		return current, nil
	case StatusPaused, StatusBlocked, StatusUsageLimited:
	case StatusBudgetLimited:
		if current.TokenBudget != nil && current.TokensUsed >= *current.TokenBudget {
			return current, ErrBudgetStillHit
		}
	default:
		return current, ErrInvalidTransition
	}
	if _, err := db.ExecContext(ctx, `UPDATE session_esm_objectives
		SET status = ?, blocked_count = 0, blocked_reason = '', updated_at = ?
		WHERE session_id = ?`, StatusActive, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

// SetBudget sets or clears the token budget. It does not implicitly resume a
// budget-limited objective; users must run /esm resume after raising/removing it.
func (s *Store) SetBudget(ctx context.Context, sessionID string, budget *int64) (*Objective, error) {
	if budget != nil && *budget <= 0 {
		return nil, fmt.Errorf("token budget must be positive")
	}
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	current, err := getObjective(ctx, db, sessionID)
	if err != nil {
		return nil, err
	}
	if !IsUnfinishedStatus(current.Status) {
		return current, ErrInvalidTransition
	}
	var value any
	if budget != nil {
		value = *budget
	}
	if _, err := db.ExecContext(ctx, `UPDATE session_esm_objectives
		SET token_budget = ?, updated_at = ?
		WHERE session_id = ?`, value, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

// AccountUsage accumulates one agent run's usage and applies token budget
// enforcement after the run finishes.
func (s *Store) AccountUsage(ctx context.Context, sessionID string, tokens, durationMS int64) (*Objective, error) {
	if tokens < 0 {
		tokens = 0
	}
	if durationMS < 0 {
		durationMS = 0
	}
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	current, err := getObjective(ctx, tx, sessionID)
	if err != nil {
		return nil, err
	}
	newTokens := current.TokensUsed + tokens
	newDuration := current.TimeUsedMS + durationMS
	newStatus := current.Status
	if current.TokenBudget != nil && newTokens >= *current.TokenBudget {
		newStatus = StatusBudgetLimited
	}
	if _, err := tx.ExecContext(ctx, `UPDATE session_esm_objectives
		SET tokens_used = ?, time_used_ms = ?, status = ?, updated_at = ?
		WHERE session_id = ?`, newTokens, newDuration, newStatus, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

// UpdateFromModel accepts the two model-controlled transitions: complete and
// blocked. Blocked only becomes terminal after the same blocker repeats 3 times.
func (s *Store) UpdateFromModel(ctx context.Context, sessionID string, status Status, reason string) (*Objective, error) {
	reason = strings.TrimSpace(reason)
	switch status {
	case StatusComplete:
	case StatusBlocked:
		if reason == "" {
			return nil, fmt.Errorf("blocked status requires a concrete reason")
		}
	default:
		return nil, fmt.Errorf("model may only set esm status to %q or %q", StatusComplete, StatusBlocked)
	}

	db, err := s.db()
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	current, err := getObjective(ctx, tx, sessionID)
	if err != nil {
		return nil, err
	}
	if current.Status == StatusBudgetLimited {
		return current, nil
	}
	if current.Status != StatusActive {
		return current, ErrInvalidTransition
	}

	nextStatus := current.Status
	nextCount := current.BlockedCount
	nextReason := current.BlockedReason
	switch status {
	case StatusComplete:
		nextStatus = StatusComplete
		nextCount = 0
		nextReason = ""
	case StatusBlocked:
		nextStatus = StatusActive
		if sameBlockedReason(current.BlockedReason, reason) {
			nextCount++
		} else {
			nextCount = 1
		}
		nextReason = reason
		if nextCount >= 3 {
			nextStatus = StatusBlocked
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE session_esm_objectives
		SET status = ?, blocked_count = ?, blocked_reason = ?, updated_at = ?
		WHERE session_id = ?`, nextStatus, nextCount, nextReason, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

func sameBlockedReason(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) && strings.TrimSpace(a) != ""
}

// IsUsageLimitError applies a conservative text heuristic for provider/account
// limits that should stop unattended continuation.
func IsUsageLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	markers := []string{
		"usage limit",
		"rate limit",
		"quota",
		"insufficient_quota",
		"resource_exhausted",
		"billing",
		"too many requests",
	}
	for _, marker := range markers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}
