package esm

import (
	"context"
	"errors"
	"testing"

	"github.com/startvibecoding/mothx/internal/session"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	sessionDir := t.TempDir()
	m := session.New("/tmp/esm-test", sessionDir)
	if err := m.Init(); err != nil {
		t.Fatalf("Init session: %v", err)
	}
	return NewStore(sessionDir), m.GetHeader().ID
}

func TestStoreCreateBudgetAndResume(t *testing.T) {
	ctx := context.Background()
	store, sessionID := newTestStore(t)

	obj, err := store.Create(ctx, sessionID, "ship esm", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if obj.Status != StatusActive || obj.Objective != "ship esm" {
		t.Fatalf("created objective = %#v", obj)
	}

	if _, err := store.Create(ctx, sessionID, "replace", nil); !errors.Is(err, ErrObjectiveExists) {
		t.Fatalf("duplicate Create error = %v, want ErrObjectiveExists", err)
	}

	budget := int64(100)
	if obj, err = store.SetBudget(ctx, sessionID, &budget); err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
	if obj.TokenBudget == nil || *obj.TokenBudget != budget {
		t.Fatalf("budget = %#v, want %d", obj.TokenBudget, budget)
	}

	if obj, err = store.AccountUsage(ctx, sessionID, 60, 1000); err != nil {
		t.Fatalf("AccountUsage first: %v", err)
	}
	if obj.Status != StatusActive || obj.TokensUsed != 60 {
		t.Fatalf("after first usage = %#v", obj)
	}

	if obj, err = store.AccountUsage(ctx, sessionID, 40, 500); err != nil {
		t.Fatalf("AccountUsage second: %v", err)
	}
	if obj.Status != StatusBudgetLimited || obj.TokensUsed != 100 || obj.TimeUsedMS != 1500 {
		t.Fatalf("after budget usage = %#v", obj)
	}

	if _, err := store.Resume(ctx, sessionID); !errors.Is(err, ErrBudgetStillHit) {
		t.Fatalf("Resume while exhausted error = %v, want ErrBudgetStillHit", err)
	}

	raised := int64(200)
	if _, err := store.SetBudget(ctx, sessionID, &raised); err != nil {
		t.Fatalf("raise budget: %v", err)
	}
	obj, err = store.Resume(ctx, sessionID)
	if err != nil {
		t.Fatalf("Resume after raise: %v", err)
	}
	if obj.Status != StatusActive {
		t.Fatalf("status after resume = %s, want active", obj.Status)
	}
}

func TestStoreBlockedAuditAndComplete(t *testing.T) {
	ctx := context.Background()
	store, sessionID := newTestStore(t)
	if _, err := store.Create(ctx, sessionID, "finish migration", nil); err != nil {
		t.Fatalf("Create: %v", err)
	}

	for i, runID := range []string{"run-1", "run-2"} {
		obj, err := store.UpdateFromModelForRun(ctx, sessionID, StatusBlocked, "missing API token", runID)
		if err != nil {
			t.Fatalf("UpdateFromModel blocked %d: %v", i+1, err)
		}
		if obj.Status != StatusActive || obj.BlockedCount != i+1 {
			t.Fatalf("blocked audit %d = %#v", i+1, obj)
		}
	}
	obj, err := store.UpdateFromModelForRun(ctx, sessionID, StatusBlocked, "missing API token", "run-3")
	if err != nil {
		t.Fatalf("UpdateFromModel blocked 3: %v", err)
	}
	if obj.Status != StatusBlocked || obj.BlockedCount != 3 {
		t.Fatalf("third blocked audit = %#v", obj)
	}

	obj, err = store.Resume(ctx, sessionID)
	if err != nil {
		t.Fatalf("Resume blocked: %v", err)
	}
	if obj.BlockedCount != 0 || obj.BlockedReason != "" || obj.Status != StatusActive {
		t.Fatalf("resume did not reset blocked audit: %#v", obj)
	}

	obj, err = store.UpdateFromModel(ctx, sessionID, StatusComplete, "all checks pass")
	if err != nil {
		t.Fatalf("UpdateFromModel complete: %v", err)
	}
	if obj.Status != StatusCompleteCandidate || obj.CompletionReason != "all checks pass" {
		t.Fatalf("complete candidate = %#v", obj)
	}

	obj, err = store.MarkCompleteFromAudit(ctx, sessionID, "auditor verified every requirement")
	if err != nil {
		t.Fatalf("MarkCompleteFromAudit: %v", err)
	}
	if obj.Status != StatusComplete {
		t.Fatalf("status = %s, want complete after audit", obj.Status)
	}
}

func TestStoreBlockedAuditRequiresConsecutiveRuns(t *testing.T) {
	ctx := context.Background()
	store, sessionID := newTestStore(t)
	if _, err := store.Create(ctx, sessionID, "finish migration", nil); err != nil {
		t.Fatalf("Create: %v", err)
	}

	obj, err := store.UpdateFromModelForRun(ctx, sessionID, StatusBlocked, "missing API token", "run-1")
	if err != nil {
		t.Fatalf("blocked run 1: %v", err)
	}
	if obj.BlockedCount != 1 || obj.BlockedRunID != "run-1" || obj.Status != StatusActive {
		t.Fatalf("blocked run 1 = %#v", obj)
	}

	obj, err = store.UpdateFromModelForRun(ctx, sessionID, StatusBlocked, "missing API token", "run-1")
	if err != nil {
		t.Fatalf("duplicate blocked run 1: %v", err)
	}
	if obj.BlockedCount != 1 {
		t.Fatalf("duplicate blocked same run count = %d, want 1", obj.BlockedCount)
	}

	obj, err = store.FinishRun(ctx, sessionID, "run-2")
	if err != nil {
		t.Fatalf("FinishRun run 2: %v", err)
	}
	if obj.BlockedCount != 0 || obj.BlockedReason != "" || obj.BlockedRunID != "" {
		t.Fatalf("non-blocked run did not reset audit: %#v", obj)
	}

	for i, runID := range []string{"run-3", "run-4"} {
		obj, err = store.UpdateFromModelForRun(ctx, sessionID, StatusBlocked, "missing API token", runID)
		if err != nil {
			t.Fatalf("blocked %s: %v", runID, err)
		}
		if obj.BlockedCount != i+1 || obj.Status != StatusActive {
			t.Fatalf("blocked %s = %#v", runID, obj)
		}
	}
	obj, err = store.UpdateFromModelForRun(ctx, sessionID, StatusBlocked, "missing API token", "run-5")
	if err != nil {
		t.Fatalf("blocked run 5: %v", err)
	}
	if obj.BlockedCount != 3 || obj.Status != StatusBlocked {
		t.Fatalf("third consecutive blocker = %#v", obj)
	}
}

func TestStoreCompleteRequiresEvidence(t *testing.T) {
	ctx := context.Background()
	store, sessionID := newTestStore(t)
	if _, err := store.Create(ctx, sessionID, "finish migration", nil); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := store.UpdateFromModel(ctx, sessionID, StatusComplete, ""); err == nil {
		t.Fatal("UpdateFromModel complete without evidence succeeded, want error")
	}
}

func TestStoreRejectCompletionCandidateReturnsActive(t *testing.T) {
	ctx := context.Background()
	store, sessionID := newTestStore(t)
	if _, err := store.Create(ctx, sessionID, "finish migration", nil); err != nil {
		t.Fatalf("Create: %v", err)
	}

	obj, err := store.UpdateFromModelForRun(ctx, sessionID, StatusComplete, "worker evidence", "run-1")
	if err != nil {
		t.Fatalf("UpdateFromModelForRun complete: %v", err)
	}
	if obj.Status != StatusCompleteCandidate || obj.CompletionRunID != "run-1" {
		t.Fatalf("candidate = %#v", obj)
	}

	obj, err = store.RejectCompletionCandidate(ctx, sessionID, "missing requirement")
	if err != nil {
		t.Fatalf("RejectCompletionCandidate: %v", err)
	}
	if obj.Status != StatusActive || obj.CompletionReview != "missing requirement" {
		t.Fatalf("rejected candidate = %#v", obj)
	}
}

func TestStoreRecordCompletionReviewWhileActive(t *testing.T) {
	ctx := context.Background()
	store, sessionID := newTestStore(t)
	if _, err := store.Create(ctx, sessionID, "finish migration", nil); err != nil {
		t.Fatalf("Create: %v", err)
	}

	obj, err := store.RecordCompletionReview(ctx, sessionID, "worker completion lacked tool-backed evidence")
	if err != nil {
		t.Fatalf("RecordCompletionReview: %v", err)
	}
	if obj.Status != StatusActive || obj.CompletionReview != "worker completion lacked tool-backed evidence" {
		t.Fatalf("recorded review = %#v", obj)
	}
}
