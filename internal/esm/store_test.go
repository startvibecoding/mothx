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

	for i := 1; i <= 2; i++ {
		obj, err := store.UpdateFromModel(ctx, sessionID, StatusBlocked, "missing API token")
		if err != nil {
			t.Fatalf("UpdateFromModel blocked %d: %v", i, err)
		}
		if obj.Status != StatusActive || obj.BlockedCount != i {
			t.Fatalf("blocked audit %d = %#v", i, obj)
		}
	}
	obj, err := store.UpdateFromModel(ctx, sessionID, StatusBlocked, "missing API token")
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
	if obj.Status != StatusComplete {
		t.Fatalf("status = %s, want complete", obj.Status)
	}
}
