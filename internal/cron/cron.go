// Package cron implements scheduled task management for vibecoding.
// Cron jobs are persisted in sessions.db and executed by spawning agents.
package cron

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

var fallbackCronCounter uint64

// CronJob represents a scheduled task.
type CronJob struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id,omitempty"`
	Name       string    `json:"name"`              // Short description
	Prompt     string    `json:"prompt"`            // Task prompt for sub-agent
	Schedule   string    `json:"schedule"`          // Schedule: @daily, @every 30m, 5-field cron, or empty for one-shot
	OneShot    bool      `json:"oneshot,omitempty"` // If true, auto-disable after first run
	Mode       string    `json:"mode"`              // "agent" or "yolo"
	WorkDir    string    `json:"work_dir,omitempty"`
	A2ATarget  string    `json:"a2a_target,omitempty"` // A2A server URL (if set, send task via A2A protocol)
	A2AToken   string    `json:"a2a_token,omitempty"`  // Bearer token for A2A server
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	LastRun    time.Time `json:"last_run,omitempty"`
	NextRun    time.Time `json:"next_run,omitempty"`
	RunCount   int       `json:"run_count"`
	LastStatus string    `json:"last_status,omitempty"` // "success", "failed", "running"
	LastError  string    `json:"last_error,omitempty"`
}

// CronStore is the interface for cron job persistence.
type CronStore interface {
	List() ([]CronJob, error)
	Get(id string) (*CronJob, error)
	Create(job CronJob) (*CronJob, error)
	Update(job CronJob) error
	Delete(id string) error
}

func newCronID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "cron-" + hex.EncodeToString(b[:])
	}
	n := atomic.AddUint64(&fallbackCronCounter, 1)
	return fmt.Sprintf("cron-%d-%d", time.Now().UnixNano(), n)
}
