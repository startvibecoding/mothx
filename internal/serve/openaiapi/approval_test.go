package openaiapi

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/startvibecoding/mothx/internal/agent"
	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/sandbox"
	"github.com/startvibecoding/mothx/internal/session"
	"github.com/startvibecoding/mothx/internal/tools"
)

func beginApprovalTestRun(sess *APISession, runID string, a *agent.Agent) context.CancelFunc {
	sess.beginRun(runID)
	_, cancel := context.WithCancel(context.Background())
	if a != nil && !sess.attachRunAgent(runID, a, cancel) {
		panic("failed to attach test run agent")
	}
	return cancel
}

func TestRuntimeSnapshotIncludesPendingApproval(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()
	sess, err := srv.getOrCreateSession("approval-runtime", srv.cfg.GetWorkDir())
	if err != nil {
		t.Fatal(err)
	}
	beginApprovalTestRun(sess, "run_1", nil)
	sess.approvalMu.Lock()
	sess.pendingApprovals = map[string]pendingSessionApproval{
		"approval_1": {Request: SessionApprovalRequest{ApprovalID: "approval_1", SessionID: sess.ID, RunID: "run_1", Summary: "Run bash"}},
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

func TestCancelSessionRunAbortsPendingApproval(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()
	sess, err := srv.getOrCreateSession("approval-cancel", srv.cfg.GetWorkDir())
	if err != nil {
		t.Fatal(err)
	}
	registry := tools.NewRegistry(srv.cfg.GetWorkDir(), sandbox.NewNoneSandbox())
	registry.RegisterDefaults()
	a := agent.New(agent.Config{Mode: "agent"}, registry)
	events := make(chan agent.Event, 1)
	result := make(chan bool, 1)
	go func() { result <- a.RequestApproval(events, "bash", map[string]any{"command": "go test ./..."}) }()

	var event agent.Event
	select {
	case event = <-events:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for blocked approval request")
	}
	beginApprovalTestRun(sess, "run_cancel", a)
	srv.registerSessionApproval(sess, a, event)

	if err := srv.CancelSessionRun(sess.ID); err != nil {
		t.Fatalf("CancelSessionRun: %v", err)
	}
	select {
	case approved := <-result:
		if approved {
			t.Fatal("cancelled approval was approved")
		}
	case <-time.After(time.Second):
		t.Fatal("blocked approval did not exit after cancellation")
	}
	sess.approvalMu.Lock()
	if len(sess.pendingApprovals) != 0 {
		sess.approvalMu.Unlock()
		t.Fatalf("pending approvals remain: %#v", sess.pendingApprovals)
	}
	sess.approvalMu.Unlock()

	stored, err := session.ListSessionRunEvents(srv.settings.GetSessionDir(), sess.ID)
	if err != nil {
		t.Fatalf("ListSessionRunEvents: %v", err)
	}
	var requested, cancelled bool
	for _, item := range stored {
		if item.RunID != "run_cancel" {
			continue
		}
		requested = requested || item.EventType == "approval_requested" && item.Status == "pending"
		cancelled = cancelled || item.EventType == "approval_resolved" && item.Status == "cancelled"
	}
	if !requested || !cancelled {
		t.Fatalf("approval audit events requested=%v cancelled=%v: %#v", requested, cancelled, stored)
	}
}

func TestCancelSessionRunBeforeApprovalRegistrationAbortsAgent(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()
	sess, err := srv.getOrCreateSession("approval-cancel-before-register", srv.cfg.GetWorkDir())
	if err != nil {
		t.Fatal(err)
	}
	a := agent.New(agent.Config{Mode: "agent"}, tools.NewRegistry(srv.cfg.GetWorkDir(), sandbox.NewNoneSandbox()))
	events := make(chan agent.Event, 1)
	result := make(chan bool, 1)
	go func() { result <- a.RequestApproval(events, "bash", map[string]any{"command": "go test ./..."}) }()
	var event agent.Event
	select {
	case event = <-events:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for approval event")
	}
	beginApprovalTestRun(sess, "run_before_register", a)

	if err := srv.CancelSessionRun(sess.ID); err != nil {
		t.Fatalf("CancelSessionRun: %v", err)
	}
	select {
	case approved := <-result:
		if approved {
			t.Fatal("cancelled approval was approved")
		}
	case <-time.After(time.Second):
		t.Fatal("agent remained blocked before approval registration")
	}
	if request := srv.registerSessionApproval(sess, a, event); request != nil {
		t.Fatalf("late approval became pending: %#v", request)
	}
	snapshot, err := srv.GetSessionRuntime(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.PendingApprovals) != 0 || snapshot.ActiveRun == nil || snapshot.ActiveRun.Status != "cancelling" {
		t.Fatalf("runtime after stop = %#v", snapshot)
	}
	stored, err := session.ListSessionRunEvents(srv.settings.GetSessionDir(), sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 2 || stored[0].EventType != "approval_requested" || stored[0].Status != "pending" || stored[1].EventType != "approval_resolved" || stored[1].Status != "cancelled" || stored[1].RunID != "run_before_register" {
		t.Fatalf("late approval audit = %#v", stored)
	}
}

func TestCancelSessionRunDoesNotAffectOtherSessionApproval(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()
	makePending := func(id, runID string) (*APISession, <-chan bool) {
		sess, err := srv.getOrCreateSession(id, srv.cfg.GetWorkDir())
		if err != nil {
			t.Fatal(err)
		}
		a := agent.New(agent.Config{Mode: "agent"}, tools.NewRegistry(srv.cfg.GetWorkDir(), sandbox.NewNoneSandbox()))
		events := make(chan agent.Event, 1)
		result := make(chan bool, 1)
		go func() { result <- a.RequestApproval(events, "bash", map[string]any{"command": id}) }()
		event := <-events
		beginApprovalTestRun(sess, runID, a)
		if srv.registerSessionApproval(sess, a, event) == nil {
			t.Fatalf("failed to register approval for %s", id)
		}
		return sess, result
	}
	sessA, resultA := makePending("approval-isolation-a", "run_a")
	sessB, resultB := makePending("approval-isolation-b", "run_b")

	if err := srv.CancelSessionRun(sessA.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case approved := <-resultA:
		if approved {
			t.Fatal("session A approval was approved")
		}
	case <-time.After(time.Second):
		t.Fatal("session A approval remained blocked")
	}
	select {
	case <-resultB:
		t.Fatal("session B approval was affected by session A stop")
	default:
	}
	snapshotB, err := srv.GetSessionRuntime(sessB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshotB.ActiveRun == nil || snapshotB.ActiveRun.RunID != "run_b" || len(snapshotB.PendingApprovals) != 1 {
		t.Fatalf("session B runtime changed: %#v", snapshotB)
	}
}

func TestResolveSessionApprovalFirstResponseWins(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()
	sess, err := srv.getOrCreateSession("approval-race", srv.cfg.GetWorkDir())
	if err != nil {
		t.Fatal(err)
	}
	beginApprovalTestRun(sess, "run_race", nil)
	sess.approvalMu.Lock()
	sess.pendingApprovals = map[string]pendingSessionApproval{
		"approval_1": {Request: SessionApprovalRequest{ApprovalID: "approval_1", SessionID: sess.ID, RunID: "run_race", Mode: "agent"}},
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
	beginApprovalTestRun(sess, "run_resume", a)
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
	request := SessionApprovalRequest{ApprovalID: "approval_1", SessionID: sess.ID, RunID: "run_rollback", Tool: map[string]any{"args": map[string]any{"command": "go test ./..."}}}
	beginApprovalTestRun(sess, request.RunID, nil)
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
	beginApprovalTestRun(sess, "run_cleanup", nil)
	sess.approvalMu.Lock()
	sess.pendingApprovals = map[string]pendingSessionApproval{
		"approval_1": {Request: SessionApprovalRequest{ApprovalID: "approval_1", SessionID: sess.ID, RunID: "run_cleanup"}},
	}
	sess.approvalMu.Unlock()

	srv.clearSessionApprovals(sess, "cancelled", "run cancelled")
	sess.approvalMu.Lock()
	defer sess.approvalMu.Unlock()
	if len(sess.pendingApprovals) != 0 {
		t.Fatalf("pending approvals remain: %#v", sess.pendingApprovals)
	}
}
