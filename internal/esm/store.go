package esm

import (
	"context"
	"database/sql"
	"encoding/json"
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
	row := q.QueryRowContext(ctx, `SELECT session_id, esm_id, objective, status, token_budget, tokens_used, time_used_ms, blocked_count, blocked_reason, blocked_run_id, completion_reason, completion_run_id, completion_review, phase, progress_summary, remaining_work, completion_rejection_count, completion_rejection_run_id, created_at, updated_at
		FROM session_esm_objectives WHERE session_id = ?`, sessionID)
	return scanObjective(row)
}

func scanObjective(row *sql.Row) (*Objective, error) {
	var obj Objective
	var budget sql.NullInt64
	var remainingWork string
	var created, updated string
	if err := row.Scan(&obj.SessionID, &obj.ESMID, &obj.Objective, &obj.Status, &budget, &obj.TokensUsed, &obj.TimeUsedMS, &obj.BlockedCount, &obj.BlockedReason, &obj.BlockedRunID, &obj.CompletionReason, &obj.CompletionRunID, &obj.CompletionReview, &obj.Phase, &obj.ProgressSummary, &remainingWork, &obj.RejectionCount, &obj.RejectionRunID, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if budget.Valid {
		v := budget.Int64
		obj.TokenBudget = &v
	}
	if err := json.Unmarshal([]byte(remainingWork), &obj.RemainingWork); err != nil {
		return nil, fmt.Errorf("decode esm remaining work: %w", err)
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
		(session_id, esm_id, objective, status, token_budget, tokens_used, time_used_ms, blocked_count, blocked_reason, phase, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 0, 0, 0, '', ?, ?, ?)`,
		sessionID, esmID, objective, StatusActive, budgetValue, PhaseWorker, now, now); err != nil {
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
		SET objective = ?, blocked_count = 0, blocked_reason = '', blocked_run_id = '', completion_reason = '', completion_run_id = '', completion_review = '', phase = ?, progress_summary = '', remaining_work = '[]', completion_rejection_count = 0, completion_rejection_run_id = '', updated_at = ?
		WHERE session_id = ?`, objective, PhaseWorker, s.timestamp(), sessionID); err != nil {
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
		SET status = ?, blocked_count = 0, blocked_reason = '', blocked_run_id = '', completion_reason = '', completion_run_id = '', phase = ?, completion_rejection_count = 0, completion_rejection_run_id = '', updated_at = ?
		WHERE session_id = ?`, StatusActive, PhaseWorker, s.timestamp(), sessionID); err != nil {
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

// SetPhase records the current role in the worker/critic/audit pipeline.
func (s *Store) SetPhase(ctx context.Context, sessionID string, phase Phase) (*Objective, error) {
	switch phase {
	case PhaseWorker, PhaseCritic, PhaseAudit, PhaseComplete:
	default:
		return nil, fmt.Errorf("invalid esm phase %q", phase)
	}
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	current, err := getObjective(ctx, db, sessionID)
	if err != nil {
		return nil, err
	}
	validTransition := false
	switch phase {
	case PhaseWorker:
		validTransition = current.Status == StatusActive
	case PhaseCritic, PhaseAudit:
		validTransition = current.Status == StatusCompleteCandidate
	case PhaseComplete:
		validTransition = current.Status == StatusComplete
	}
	if !validTransition {
		return current, ErrInvalidTransition
	}
	if _, err := db.ExecContext(ctx, `UPDATE session_esm_objectives
		SET phase = ?, updated_at = ?
		WHERE session_id = ?`, phase, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

// RecordWorkerProgress persists the latest structured worker result so later
// runs and the TUI can show concrete progress and remaining work.
func (s *Store) RecordWorkerProgress(ctx context.Context, sessionID, summary string, remainingWork []string) (*Objective, error) {
	encoded, err := encodeStringSlice(remainingWork)
	if err != nil {
		return nil, err
	}
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	current, err := getObjective(ctx, db, sessionID)
	if err != nil {
		return nil, err
	}
	if current.Status != StatusActive {
		return current, ErrInvalidTransition
	}
	if _, err := db.ExecContext(ctx, `UPDATE session_esm_objectives
		SET phase = ?, progress_summary = ?, remaining_work = ?, updated_at = ?
		WHERE session_id = ?`, PhaseWorker, strings.TrimSpace(summary), encoded, s.timestamp(), sessionID); err != nil {
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
// blocked. Complete records a candidate only; the orchestrator must audit the
// candidate before marking the objective terminal complete. Blocked transitions
// should normally use UpdateFromModelForRun so the repeated-blocker audit is
// counted across consecutive agent runs.
func (s *Store) UpdateFromModel(ctx context.Context, sessionID string, status Status, reason string) (*Objective, error) {
	return s.UpdateFromModelForRun(ctx, sessionID, status, reason, "")
}

// UpdateFromModelForRun accepts model-controlled complete/blocked transitions.
// Complete becomes complete_candidate, never terminal complete. The terminal
// complete state is reserved for the ESM orchestrator after an independent
// audit sub-agent passes.
// Blocked only becomes terminal after the same blocker repeats in three
// consecutive ESM agent runs. A run can contribute at most once to the audit.
func (s *Store) UpdateFromModelForRun(ctx context.Context, sessionID string, status Status, reason, runID string) (*Objective, error) {
	reason = strings.TrimSpace(reason)
	runID = strings.TrimSpace(runID)
	switch status {
	case StatusComplete:
		if reason == "" {
			return nil, fmt.Errorf("complete status requires verification evidence")
		}
	case StatusBlocked:
		if reason == "" {
			return nil, fmt.Errorf("blocked status requires a concrete reason")
		}
		if runID == "" {
			return nil, fmt.Errorf("blocked status requires an ESM run id")
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
	nextRunID := current.BlockedRunID
	nextCompletionReason := current.CompletionReason
	nextCompletionRunID := current.CompletionRunID
	nextCompletionReview := current.CompletionReview
	nextPhase := current.Phase
	nextRejectionCount := current.RejectionCount
	nextRejectionRunID := current.RejectionRunID
	switch status {
	case StatusComplete:
		nextStatus = StatusCompleteCandidate
		nextCount = 0
		nextReason = ""
		nextRunID = ""
		nextCompletionReason = reason
		nextCompletionRunID = runID
		nextCompletionReview = ""
		nextPhase = PhaseCritic
	case StatusBlocked:
		nextStatus = StatusActive
		if current.BlockedRunID == runID && sameBlockedReason(current.BlockedReason, reason) {
			nextCount = current.BlockedCount
		} else if sameBlockedReason(current.BlockedReason, reason) {
			nextCount++
		} else {
			nextCount = 1
		}
		nextReason = reason
		nextRunID = runID
		nextCompletionReason = ""
		nextCompletionRunID = ""
		nextCompletionReview = ""
		nextRejectionCount = 0
		nextRejectionRunID = ""
		if nextCount >= 3 {
			nextStatus = StatusBlocked
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE session_esm_objectives
		SET status = ?, blocked_count = ?, blocked_reason = ?, blocked_run_id = ?, completion_reason = ?, completion_run_id = ?, completion_review = ?, phase = ?, completion_rejection_count = ?, completion_rejection_run_id = ?, updated_at = ?
		WHERE session_id = ?`, nextStatus, nextCount, nextReason, nextRunID, nextCompletionReason, nextCompletionRunID, nextCompletionReview, nextPhase, nextRejectionCount, nextRejectionRunID, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

// MarkCompleteFromAudit marks a completion candidate terminal complete after an
// independent ESM audit has verified the objective against the current state.
func (s *Store) MarkCompleteFromAudit(ctx context.Context, sessionID, review string) (*Objective, error) {
	review = strings.TrimSpace(review)
	if review == "" {
		return nil, fmt.Errorf("complete audit requires a review")
	}
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	current, err := getObjective(ctx, db, sessionID)
	if err != nil {
		return nil, err
	}
	if current.Status != StatusCompleteCandidate {
		return current, ErrInvalidTransition
	}
	if _, err := db.ExecContext(ctx, `UPDATE session_esm_objectives
		SET status = ?, blocked_count = 0, blocked_reason = '', blocked_run_id = '', completion_review = ?, phase = ?, remaining_work = '[]', completion_rejection_count = 0, completion_rejection_run_id = '', updated_at = ?
		WHERE session_id = ?`, StatusComplete, review, PhaseComplete, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

// RejectCompletionCandidate records a failed completion candidate. Repeated
// rejections pause unattended continuation at CompletionRejectionLimit.
func (s *Store) RejectCompletionCandidate(ctx context.Context, sessionID, review string) (*Objective, error) {
	current, err := s.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return s.RejectCompletionCandidateForRun(ctx, sessionID, current.CompletionRunID, review, nil)
}

// RejectCompletionCandidateForRun records a critic/audit rejection with its
// structured missing work. A run contributes at most once to the streak.
func (s *Store) RejectCompletionCandidateForRun(ctx context.Context, sessionID, runID, review string, missingWork []string) (*Objective, error) {
	return s.recordCompletionRejection(ctx, sessionID, runID, review, missingWork, StatusCompleteCandidate)
}

// RejectWorkerReport records a worker report rejected before supervisor review
// while the objective is still active.
func (s *Store) RejectWorkerReport(ctx context.Context, sessionID, runID, review string, remainingWork []string) (*Objective, error) {
	return s.recordCompletionRejection(ctx, sessionID, runID, review, remainingWork, StatusActive)
}

func (s *Store) recordCompletionRejection(ctx context.Context, sessionID, runID, review string, remainingWork []string, expectedStatus Status) (*Objective, error) {
	review = strings.TrimSpace(review)
	if review == "" {
		return nil, fmt.Errorf("completion rejection requires an audit review")
	}
	runID = strings.TrimSpace(runID)
	encoded, err := encodeStringSlice(remainingWork)
	if err != nil {
		return nil, err
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
	if current.Status != expectedStatus {
		if runID != "" && current.RejectionRunID == runID {
			return current, nil
		}
		return current, ErrInvalidTransition
	}
	nextCount := current.RejectionCount
	if runID == "" || current.RejectionRunID != runID {
		nextCount++
	}
	nextStatus := StatusActive
	if nextCount >= CompletionRejectionLimit {
		nextStatus = StatusPaused
	}
	if _, err := tx.ExecContext(ctx, `UPDATE session_esm_objectives
		SET status = ?, completion_review = ?, remaining_work = ?, completion_rejection_count = ?, completion_rejection_run_id = ?, updated_at = ?
		WHERE session_id = ?`, nextStatus, review, encoded, nextCount, runID, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

// RecordCompletionReview stores a completion rejection/review note without
// changing the current lifecycle state. This lets later worker runs learn why a
// previous completion claim was not accepted.
func (s *Store) RecordCompletionReview(ctx context.Context, sessionID, review string) (*Objective, error) {
	review = strings.TrimSpace(review)
	if review == "" {
		return nil, fmt.Errorf("completion review cannot be empty")
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
		SET completion_review = ?, updated_at = ?
		WHERE session_id = ?`, review, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

// FinishRun clears repeated blocker/rejection streaks when an active ESM run
// finishes without reporting the same condition.
func (s *Store) FinishRun(ctx context.Context, sessionID, runID string) (*Objective, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return s.Get(ctx, sessionID)
	}
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
	nextBlockedCount := current.BlockedCount
	nextBlockedReason := current.BlockedReason
	nextBlockedRunID := current.BlockedRunID
	if current.BlockedCount > 0 && current.BlockedRunID != "" && current.BlockedRunID != runID {
		nextBlockedCount = 0
		nextBlockedReason = ""
		nextBlockedRunID = ""
	}
	nextRejectionCount := current.RejectionCount
	nextRejectionRunID := current.RejectionRunID
	if current.RejectionCount > 0 && current.RejectionRunID != "" && current.RejectionRunID != runID {
		nextRejectionCount = 0
		nextRejectionRunID = ""
	}
	if nextBlockedCount == current.BlockedCount && nextRejectionCount == current.RejectionCount {
		return current, nil
	}
	if _, err := db.ExecContext(ctx, `UPDATE session_esm_objectives
		SET blocked_count = ?, blocked_reason = ?, blocked_run_id = ?, completion_rejection_count = ?, completion_rejection_run_id = ?, updated_at = ?
		WHERE session_id = ?`, nextBlockedCount, nextBlockedReason, nextBlockedRunID, nextRejectionCount, nextRejectionRunID, s.timestamp(), sessionID); err != nil {
		return nil, err
	}
	return s.Get(ctx, sessionID)
}

func encodeStringSlice(values []string) (string, error) {
	values = trimStringSlice(values)
	if values == nil {
		values = []string{}
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("encode esm remaining work: %w", err)
	}
	return string(encoded), nil
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
