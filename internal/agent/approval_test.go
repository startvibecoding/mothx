package agent

import (
	"testing"

	"github.com/startvibecoding/vibecoding/internal/config"
	"github.com/startvibecoding/vibecoding/internal/provider"
	"github.com/startvibecoding/vibecoding/internal/sandbox"
	"github.com/startvibecoding/vibecoding/internal/tools"
)

func newApprovalTestAgent(t *testing.T, mode string, approval config.ApprovalSettings) *Agent {
	t.Helper()
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{{ID: "model1", Name: "Model 1"}}, nil)
	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(t.TempDir(), sb)
	registry.RegisterDefaults()
	return New(Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     mode,
		Settings: &config.Settings{Approval: approval},
	}, registry)
}

func TestNeedsApproval_NonBashNeverNeedsApproval(t *testing.T) {
	a := newApprovalTestAgent(t, "agent", config.ApprovalSettings{})
	if a.NeedsApproval("read", map[string]any{"path": "README.md"}) {
		t.Fatal("non-bash tool should not require approval")
	}
}

func TestNeedsApproval_AgentModeWhitelistSkipsApproval(t *testing.T) {
	a := newApprovalTestAgent(t, "agent", config.ApprovalSettings{
		BashWhitelist: []string{"go ", "make "},
	})
	if a.NeedsApproval("bash", map[string]any{"command": "go test ./..."}) {
		t.Fatal("whitelisted bash command should not require approval in agent mode")
	}
}

func TestNeedsApproval_AgentModeBlacklistForcesApproval(t *testing.T) {
	a := newApprovalTestAgent(t, "agent", config.ApprovalSettings{
		BashWhitelist: []string{"go ", "rm "},
		BashBlacklist: []string{"rm -rf"},
	})
	if !a.NeedsApproval("bash", map[string]any{"command": "rm -rf /tmp/demo"}) {
		t.Fatal("blacklisted bash command should require approval in agent mode")
	}
}

func TestNeedsApproval_AgentModeNonWhitelistedNeedsApproval(t *testing.T) {
	a := newApprovalTestAgent(t, "agent", config.ApprovalSettings{
		BashWhitelist: []string{"go "},
	})
	if !a.NeedsApproval("bash", map[string]any{"command": "python script.py"}) {
		t.Fatal("non-whitelisted bash command should require approval in agent mode")
	}
}

func TestNeedsApproval_YoloModeAllowsUnlessBlacklisted(t *testing.T) {
	a := newApprovalTestAgent(t, "yolo", config.ApprovalSettings{
		BashBlacklist: []string{"rm -rf"},
	})
	if a.NeedsApproval("bash", map[string]any{"command": "go test ./..."}) {
		t.Fatal("non-blacklisted bash command should not require approval in yolo mode")
	}
	if !a.NeedsApproval("bash", map[string]any{"command": "rm -rf /"}) {
		t.Fatal("blacklisted bash command should still require approval in yolo mode")
	}
}

func TestNeedsApproval_BlacklistOverridesWhitelist(t *testing.T) {
	a := newApprovalTestAgent(t, "agent", config.ApprovalSettings{
		BashWhitelist: []string{"rm ", "rm -rf"},
		BashBlacklist: []string{"rm -rf"},
	})
	if !a.NeedsApproval("bash", map[string]any{"command": "rm -rf build"}) {
		t.Fatal("blacklist should override whitelist")
	}
}
