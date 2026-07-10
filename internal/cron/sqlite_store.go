package cron

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/startvibecoding/mothx/internal/session"
)

// SQLiteCronStore persists cron jobs in the shared sessions.db database.
type SQLiteCronStore struct {
	sessionDir string
}

// NewSQLiteCronStore creates a SQLite-backed cron store rooted at sessionDir.
func NewSQLiteCronStore(sessionDir string) *SQLiteCronStore {
	return &SQLiteCronStore{sessionDir: sessionDir}
}

func (s *SQLiteCronStore) db() (*sql.DB, error) {
	return session.OpenRootDB(s.sessionDir)
}

// List returns all cron jobs.
func (s *SQLiteCronStore) List() ([]CronJob, error) {
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id, session_id, name, prompt, schedule, oneshot, mode, work_dir, a2a_target, a2a_token,
		enabled, created_at, last_run, next_run, run_count, last_status, last_error
		FROM cron_jobs ORDER BY created_at DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []CronJob
	for rows.Next() {
		job, err := scanCronJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

// Get returns a cron job by ID.
func (s *SQLiteCronStore) Get(id string) (*CronJob, error) {
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	row := db.QueryRow(`SELECT id, session_id, name, prompt, schedule, oneshot, mode, work_dir, a2a_target, a2a_token,
		enabled, created_at, last_run, next_run, run_count, last_status, last_error
		FROM cron_jobs WHERE id = ?`, id)
	job, err := scanCronJob(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cron job %q not found", id)
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// Create adds a new cron job.
func (s *SQLiteCronStore) Create(job CronJob) (*CronJob, error) {
	if job.ID == "" {
		job.ID = newCronID()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	db, err := s.db()
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`INSERT INTO cron_jobs (
		id, session_id, name, prompt, schedule, oneshot, mode, work_dir, a2a_target, a2a_token,
		enabled, created_at, last_run, next_run, run_count, last_status, last_error
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.SessionID, job.Name, job.Prompt, job.Schedule, boolToInt(job.OneShot), job.Mode, job.WorkDir, job.A2ATarget, job.A2AToken,
		boolToInt(job.Enabled), formatCronTime(job.CreatedAt), formatCronTime(job.LastRun), formatCronTime(job.NextRun), job.RunCount, job.LastStatus, job.LastError)
	if err != nil {
		return nil, fmt.Errorf("create cron job %q: %w", job.ID, err)
	}
	return &job, nil
}

// Update updates an existing cron job.
func (s *SQLiteCronStore) Update(job CronJob) error {
	db, err := s.db()
	if err != nil {
		return err
	}
	res, err := db.Exec(`UPDATE cron_jobs SET
		session_id = ?, name = ?, prompt = ?, schedule = ?, oneshot = ?, mode = ?, work_dir = ?, a2a_target = ?, a2a_token = ?,
		enabled = ?, created_at = ?, last_run = ?, next_run = ?, run_count = ?, last_status = ?, last_error = ?
		WHERE id = ?`,
		job.SessionID, job.Name, job.Prompt, job.Schedule, boolToInt(job.OneShot), job.Mode, job.WorkDir, job.A2ATarget, job.A2AToken,
		boolToInt(job.Enabled), formatCronTime(job.CreatedAt), formatCronTime(job.LastRun), formatCronTime(job.NextRun), job.RunCount, job.LastStatus, job.LastError,
		job.ID)
	if err != nil {
		return fmt.Errorf("update cron job %q: %w", job.ID, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("cron job %q not found", job.ID)
	}
	return nil
}

// Delete removes a cron job.
func (s *SQLiteCronStore) Delete(id string) error {
	db, err := s.db()
	if err != nil {
		return err
	}
	res, err := db.Exec("DELETE FROM cron_jobs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete cron job %q: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("cron job %q not found", id)
	}
	return nil
}

// ClaimDue atomically marks a due job as running. Only the caller that updates
// a row may execute it, preventing duplicate runs across scheduler instances.
func (s *SQLiteCronStore) ClaimDue(id string, now time.Time) (bool, error) {
	db, err := s.db()
	if err != nil {
		return false, err
	}
	stamp := formatCronTime(now)
	res, err := db.Exec(`UPDATE cron_jobs
		SET last_status = 'running', last_run = ?, last_error = ''
		WHERE id = ? AND enabled = 1 AND last_status != 'running'
		AND (last_run = '' OR (next_run != '' AND next_run <= ?))`, stamp, id, stamp)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

type cronRow interface {
	Scan(dest ...any) error
}

func scanCronJob(row cronRow) (CronJob, error) {
	var job CronJob
	var oneShot, enabled int
	var createdAt, lastRun, nextRun string
	err := row.Scan(
		&job.ID,
		&job.SessionID,
		&job.Name,
		&job.Prompt,
		&job.Schedule,
		&oneShot,
		&job.Mode,
		&job.WorkDir,
		&job.A2ATarget,
		&job.A2AToken,
		&enabled,
		&createdAt,
		&lastRun,
		&nextRun,
		&job.RunCount,
		&job.LastStatus,
		&job.LastError,
	)
	if err != nil {
		return CronJob{}, err
	}
	job.OneShot = oneShot != 0
	job.Enabled = enabled != 0
	job.CreatedAt = parseCronTime(createdAt)
	job.LastRun = parseCronTime(lastRun)
	job.NextRun = parseCronTime(nextRun)
	return job, nil
}

func formatCronTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseCronTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// SessionScopedStore limits a CronStore to one session.
type SessionScopedStore struct {
	base      CronStore
	sessionID string
	workDir   string
}

// NewSessionScopedStore returns a cron store view bound to sessionID.
func NewSessionScopedStore(base CronStore, sessionID string) *SessionScopedStore {
	return NewSessionScopedStoreWithWorkDir(base, sessionID, "")
}

// NewSessionScopedStoreWithWorkDir returns a cron store view bound to sessionID.
// Jobs created through the scoped view inherit workDir when they do not specify
// one explicitly.
func NewSessionScopedStoreWithWorkDir(base CronStore, sessionID, workDir string) *SessionScopedStore {
	return &SessionScopedStore{base: base, sessionID: sessionID, workDir: workDir}
}

func (s *SessionScopedStore) List() ([]CronJob, error) {
	if s == nil || s.base == nil {
		return nil, fmt.Errorf("cron store unavailable")
	}
	jobs, err := s.base.List()
	if err != nil {
		return nil, err
	}
	filtered := jobs[:0]
	for _, job := range jobs {
		if job.SessionID == s.sessionID {
			filtered = append(filtered, job)
		}
	}
	return filtered, nil
}

func (s *SessionScopedStore) Get(id string) (*CronJob, error) {
	if s == nil || s.base == nil {
		return nil, fmt.Errorf("cron store unavailable")
	}
	job, err := s.base.Get(id)
	if err != nil {
		return nil, err
	}
	if job.SessionID != s.sessionID {
		return nil, fmt.Errorf("cron job %q not found in this session", id)
	}
	return job, nil
}

func (s *SessionScopedStore) Create(job CronJob) (*CronJob, error) {
	if s == nil || s.base == nil {
		return nil, fmt.Errorf("cron store unavailable")
	}
	if s.sessionID == "" {
		return nil, fmt.Errorf("cron tool requires a session")
	}
	if job.SessionID != "" && job.SessionID != s.sessionID {
		return nil, fmt.Errorf("cron job session mismatch")
	}
	job.SessionID = s.sessionID
	if job.WorkDir == "" {
		job.WorkDir = s.workDir
	}
	return s.base.Create(job)
}

func (s *SessionScopedStore) Update(job CronJob) error {
	if s == nil || s.base == nil {
		return fmt.Errorf("cron store unavailable")
	}
	if _, err := s.Get(job.ID); err != nil {
		return err
	}
	job.SessionID = s.sessionID
	if job.WorkDir == "" {
		job.WorkDir = s.workDir
	}
	return s.base.Update(job)
}

func (s *SessionScopedStore) Delete(id string) error {
	if s == nil || s.base == nil {
		return fmt.Errorf("cron store unavailable")
	}
	if _, err := s.Get(id); err != nil {
		return err
	}
	return s.base.Delete(id)
}
