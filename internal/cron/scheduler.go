package cron

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/startvibecoding/mothx/internal/agent"
	"github.com/startvibecoding/mothx/internal/session"
)

// Scheduler checks for due cron jobs and executes them via sub-agents.
type Scheduler struct {
	store      CronStore
	manager    *agent.AgentManager
	interval   time.Duration
	sessionDir string
	quit       chan struct{}
	running    bool
	claims     map[string]struct{}
	mu         sync.Mutex
}

var a2aHTTPClient = &http.Client{Timeout: 30 * time.Second}

const maxA2AResponseBytes = 1 << 20

type dueJobClaimer interface {
	ClaimDue(id string, now time.Time) (bool, error)
}

// NewScheduler creates a new cron scheduler.
func NewScheduler(store CronStore, manager *agent.AgentManager, interval time.Duration) *Scheduler {
	return NewSchedulerWithSessionDir(store, manager, interval, "")
}

// NewSchedulerWithSessionDir creates a scheduler that can attach scheduled
// local runs to existing sessions by session ID.
func NewSchedulerWithSessionDir(store CronStore, manager *agent.AgentManager, interval time.Duration, sessionDir string) *Scheduler {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Scheduler{
		store:      store,
		manager:    manager,
		interval:   interval,
		sessionDir: sessionDir,
		quit:       make(chan struct{}),
		claims:     make(map[string]struct{}),
	}
}

// Start begins the scheduler loop.
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.quit = make(chan struct{})
	s.mu.Unlock()

	go s.loop()
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.quit)
}

// IsRunning returns whether the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Scheduler) loop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Check immediately on start
	s.checkAndRun()

	for {
		select {
		case <-s.quit:
			return
		case <-ticker.C:
			s.checkAndRun()
		}
	}
}

// checkAndRun checks all enabled jobs and runs any that are due.
func (s *Scheduler) checkAndRun() {
	jobs, err := s.store.List()
	if err != nil {
		log.Printf("[cron] failed to list jobs: %v", err)
		return
	}

	now := time.Now()
	for _, job := range jobs {
		if !job.Enabled {
			continue
		}
		if job.LastStatus == "running" {
			continue // Don't start a job that's already running
		}
		if s.isDue(job, now) {
			claimed, release, err := s.claimJob(job.ID, now)
			if err != nil {
				log.Printf("[cron] claim job %s: %v", job.ID, err)
				continue
			}
			if claimed {
				go func() {
					defer release()
					s.executeJob(job)
				}()
			}
		}
	}
}

func (s *Scheduler) claimJob(id string, now time.Time) (bool, func(), error) {
	if claimer, ok := s.store.(dueJobClaimer); ok {
		claimed, err := claimer.ClaimDue(id, now)
		return claimed, func() {}, err
	}

	// In-memory stores cannot coordinate across processes, but retain the
	// previous single-scheduler behavior without allowing overlapping ticks.
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, claimed := s.claims[id]; claimed {
		return false, func() {}, nil
	}
	s.claims[id] = struct{}{}
	return true, func() {
		s.mu.Lock()
		delete(s.claims, id)
		s.mu.Unlock()
	}, nil
}

// isDue checks if a job should run now.
func (s *Scheduler) isDue(job CronJob, now time.Time) bool {
	// If never run, run now
	if job.LastRun.IsZero() {
		return true
	}
	// If NextRun is set and has passed
	if !job.NextRun.IsZero() && now.After(job.NextRun) {
		return true
	}
	return false
}

// executeJob runs a cron job by spawning a sub-agent or sending to A2A server.
func (s *Scheduler) executeJob(job CronJob) {
	var lastErr error

	// A2A target mode: send task to remote A2A server
	if job.A2ATarget != "" {
		lastErr = s.executeA2AJob(job)
	} else {
		// Local agent mode
		multiAgentPrompt := false
		var sess *session.Manager
		workDir := job.WorkDir
		if job.SessionID != "" && s.sessionDir != "" {
			if opened, err := session.OpenByIDExact(s.sessionDir, job.SessionID); err == nil {
				sess = opened
				if workDir == "" {
					if header := opened.GetHeader(); header != nil && header.Cwd != "" {
						workDir = header.Cwd
					}
				}
			}
		}
		a, err := s.manager.Create(agent.AgentOptions{
			IsSubAgent: sess == nil,
			Mode:       job.Mode,
			WorkDir:    workDir,
			Session:    sess,
			MultiAgent: &multiAgentPrompt,
		})
		if err != nil {
			s.updateJob(job.ID, func(current *CronJob) {
				current.LastStatus = "failed"
				current.LastError = fmt.Sprintf("create agent: %v", err)
			})
			return
		}

		ch := a.Run(context.Background(), job.Prompt)
		for event := range ch {
			if event.Error != nil {
				lastErr = event.Error
			}
		}
		s.manager.Destroy(a.ID())
	}

	s.updateJob(job.ID, func(current *CronJob) {
		current.RunCount++
		if lastErr != nil {
			current.LastStatus = "failed"
			current.LastError = lastErr.Error()
		} else {
			current.LastStatus = "success"
			current.LastError = ""
		}

		// Compute next run from the latest stored schedule.
		next, isOneShot, err := ParseSchedule(current.Schedule, time.Now())
		if err != nil {
			isOneShot = true
		}
		if isOneShot || current.OneShot {
			current.Enabled = false
			current.NextRun = time.Time{}
		} else {
			current.NextRun = next
		}
	})
}

func (s *Scheduler) updateJob(id string, update func(*CronJob)) {
	current, err := s.store.Get(id)
	if err != nil {
		return
	}
	update(current)
	_ = s.store.Update(*current)
}

// executeA2AJob sends a task to a remote A2A server.
func (s *Scheduler) executeA2AJob(job CronJob) error {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "message/send",
		"params": map[string]any{
			"message": map[string]any{
				"role":  "user",
				"parts": []map[string]string{{"type": "text", "text": job.Prompt}},
			},
		},
		"id": 1,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", job.A2ATarget+"/a2a", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if job.A2AToken != "" {
		req.Header.Set("Authorization", "Bearer "+job.A2AToken)
	}

	resp, err := a2aHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("a2a request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("a2a request: status %d", resp.StatusCode)
	}

	var result struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxA2AResponseBytes)).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if result.Error != nil {
		return fmt.Errorf("a2a error: %s", result.Error.Message)
	}
	return nil
}
