package openaiapi

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/startvibecoding/mothx/internal/agent"
	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/sandbox"
	"github.com/startvibecoding/mothx/internal/tools"
)

func TestRuntimeSnapshotIncludesPendingApproval(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()
	sess, err := srv.getOrCreateSession("approval-runtime", srv.cfg.GetWorkDir())
	if err != nil {
		t.Fatal(err)
	}
	sess.SetRunning(true)
	sess.approvalMu.Lock()
	sess.activeRunID = "run_1"
	sess.pendingApprovals = map[string]pendingSessionApproval{
		"approval_1": {Request: SessionApprovalRequest{ApprovalID: "approval_1", SessionID: sess.ID, Summary: "Run bash"}},
	}
	sess.approvalMu.Unlock()

	snapshot, err := srv.GetSessionRuntime(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ActiveRun == nil || snapshot.ActiveRun.RunID != "run_1" {
		t.Fatalf("active run = %#v, want run_1", snapshot.ActiveRun)
	}
	if len(snapshot.PendingApprovals) != 1 || snapshot.PendingApprovals[0].ApprovalID != "approval_1" {
		t.Fatalf("pending approvals = %#v", snapshot.PendingApprovals)
	}
}

func TestResolveSessionApprovalFirstResponseWins(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()
	sess, err := srv.getOrCreateSession("approval-race", srv.cfg.GetWorkDir())
	if err != nil {
		t.Fatal(err)
	}
	sess.approvalMu.Lock()
	sess.pendingApprovals = map[string]pendingSessionApproval{
		"approval_1": {Request: SessionApprovalRequest{ApprovalID: "approval_1", SessionID: sess.ID, Mode: "agent"}},
	}
	sess.approvalMu.Unlock()

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, action := range []string{"approve_once", "deny_once"} {
		wg.Add(1)
		go func(action string) {
			defer wg.Done()
			_, err := srv.ResolveSessionApproval(sess.ID, "approval_1", SessionApprovalResponse{Action: action})
			results <- err
		}(action)
	}
	wg.Wait()
	close(results)
	var successes int
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful responses = %d, want exactly 1", successes)
	}
}

func TestResolveSessionApprovalResumesBlockedAgent(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()
	sess, err := srv.getOrCreateSession("approval-resume", srv.cfg.GetWorkDir())
	if err != nil {
		t.Fatal(err)
	}
	registry := tools.NewRegistry(srv.cfg.GetWorkDir(), sandbox.NewNoneSandbox())
	registry.RegisterDefaults()
	a := agent.New(agent.Config{Mode: "agent"}, registry)
	result := make(chan bool, 1)
	events := make(chan agent.Event, 1)
	go func() { result <- a.RequestApproval(events, "bash", map[string]any{"command": "go test ./..."}) }()

	var event agent.Event
	select {
	case event = <-events:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for blocked approval request")
	}
	srv.registerSessionApproval(sess, a, event)
	if _, err := srv.ResolveSessionApproval(sess.ID, event.ApprovalID, SessionApprovalResponse{Action: "approve_once"}); err != nil {
		t.Fatal(err)
	}
	select {
	case approved := <-result:
		if !approved {
			t.Fatal("agent received denied approval, want approved")
		}
	case <-time.After(time.Second):
		t.Fatal("blocked agent did not resume after approval")
	}
}

func TestResolveSessionApprovalRollsBackRuleWhenSaveFails(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()
	srv.allow = &config.AllowConfig{}
	srv.saveProjectAllow = func(*config.AllowConfig) error { return errors.New("disk full") }
	sess, err := srv.getOrCreateSession("approval-rollback", srv.cfg.GetWorkDir())
	if err != nil {
		t.Fatal(err)
	}
	request := SessionApprovalRequest{ApprovalID: "approval_1", SessionID: sess.ID, Tool: map[string]any{"args": map[string]any{"command": "go test ./..."}}}
	sess.approvalMu.Lock()
	sess.pendingApprovals = map[string]pendingSessionApproval{"approval_1": {Request: request}}
	sess.approvalMu.Unlock()

	if _, err := srv.ResolveSessionApproval(sess.ID, request.ApprovalID, SessionApprovalResponse{Action: "remember_command"}); err == nil {
		t.Fatal("expected rule persistence failure")
	}
	if srv.allow.MatchBashCommand("go test ./...") {
		t.Fatal("failed rule persistence must rollback in-memory allow rule")
	}
	sess.approvalMu.Lock()
	defer sess.approvalMu.Unlock()
	if _, ok := sess.pendingApprovals[request.ApprovalID]; !ok {
		t.Fatal("approval must remain pending after rule persistence failure")
	}
}

func TestClearSessionApprovalsResolvesAndRemovesPending(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()
	sess, err := srv.getOrCreateSession("approval-cleanup", srv.cfg.GetWorkDir())
	if err != nil {
		t.Fatal(err)
	}
	sess.approvalMu.Lock()
	sess.pendingApprovals = map[string]pendingSessionApproval{
		"approval_1": {Request: SessionApprovalRequest{ApprovalID: "approval_1", SessionID: sess.ID}},
	}
	sess.approvalMu.Unlock()

	srv.clearSessionApprovals(sess, "cancelled", "run cancelled")
	sess.approvalMu.Lock()
	defer sess.approvalMu.Unlock()
	if len(sess.pendingApprovals) != 0 {
		t.Fatalf("pending approvals remain: %#v", sess.pendingApprovals)
	}
}
