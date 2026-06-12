package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/startvibecoding/vibecoding/internal/config"
	"github.com/startvibecoding/vibecoding/internal/gateway"
	"github.com/startvibecoding/vibecoding/internal/mcp"
	"github.com/startvibecoding/vibecoding/internal/platform"
	providerfactory "github.com/startvibecoding/vibecoding/internal/provider/factory"
)

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check environment, configuration, and provider status",
		Long:  "Diagnose your VibeCoding environment: OS info, config files, providers, models, sandbox, MCP, and more.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

type checkResult struct {
	Name   string
	Status string // "ok", "warn", "error", "skip"
	Detail string
}

func (r checkResult) Icon() string {
	switch r.Status {
	case "ok":
		return "✅"
	case "warn":
		return "⚠️ "
	case "error":
		return "❌"
	default:
		return "⏭️ "
	}
}

type section struct {
	Title  string
	Checks []checkResult
}

func runDoctor() error {
	fmt.Println()
	fmt.Println("  VibeCoding Doctor")
	fmt.Println("  ─────────────────")

	sections := []section{
		checkEnvironment(),
		checkConfigFiles(),
		checkProviders(),
		checkSandbox(),
		checkMCPServers(),
		checkSessions(),
		checkSkills(),
		checkContextFiles(),
	}

	var totalOK, totalWarn, totalErr int
	for _, sec := range sections {
		fmt.Printf("\n  \033[1m%s\033[0m\n", sec.Title)
		for _, r := range sec.Checks {
			fmt.Printf("    %s %s", r.Icon(), r.Name)
			if r.Detail != "" {
				fmt.Printf(" — %s", r.Detail)
			}
			fmt.Println()
			switch r.Status {
			case "ok":
				totalOK++
			case "warn":
				totalWarn++
			case "error":
				totalErr++
			}
		}
	}

	fmt.Println()
	if totalErr > 0 {
		fmt.Printf("  \033[1mResult:\033[0m %d ok, %d warnings, \033[31m%d errors\033[0m\n", totalOK, totalWarn, totalErr)
	} else if totalWarn > 0 {
		fmt.Printf("  \033[1mResult:\033[0m %d ok, \033[33m%d warnings\033[0m\n", totalOK, totalWarn)
	} else {
		fmt.Printf("  \033[1mResult:\033[0m \033[32mAll %d checks passed\033[0m\n", totalOK)
	}
	fmt.Println()
	return nil
}

// ── Environment ──────────────────────────────────────────────

func checkEnvironment() section {
	s := section{Title: "Environment"}

	s.Checks = append(s.Checks, checkResult{
		Name:   "OS / Arch",
		Status: "ok",
		Detail: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	})

	s.Checks = append(s.Checks, checkResult{
		Name:   "Go version",
		Status: "ok",
		Detail: runtime.Version(),
	})

	shell := platform.DefaultShell()
	if _, err := os.Stat(shell); err != nil {
		s.Checks = append(s.Checks, checkResult{Name: "Shell", Status: "warn", Detail: fmt.Sprintf("%s (not found)", shell)})
	} else {
		s.Checks = append(s.Checks, checkResult{Name: "Shell", Status: "ok", Detail: shell})
	}

	home := platform.HomeDir()
	if _, err := os.Stat(home); err != nil {
		s.Checks = append(s.Checks, checkResult{Name: "Home directory", Status: "error", Detail: fmt.Sprintf("%s (not accessible)", home)})
	} else {
		s.Checks = append(s.Checks, checkResult{Name: "Home directory", Status: "ok", Detail: home})
	}

	if cwd, err := os.Getwd(); err != nil {
		s.Checks = append(s.Checks, checkResult{Name: "Working directory", Status: "error", Detail: err.Error()})
	} else {
		s.Checks = append(s.Checks, checkResult{Name: "Working directory", Status: "ok", Detail: cwd})
	}

	return s
}

// ── Config Files ─────────────────────────────────────────────

func checkConfigFiles() section {
	s := section{Title: "Configuration Files"}

	type cfgFile struct {
		name string
		path string
	}
	files := []cfgFile{
		{"Global settings", config.GlobalSettingsPath()},
		{"Project settings", config.ProjectSettingsPath()},
		{"Gateway config (global)", gateway.GatewayConfigPath()},
		{"Gateway config (project)", gateway.ProjectGatewayConfigPath()},
		{"MCP config (global)", config.GlobalMCPPath()},
		{"MCP config (project)", filepath.Join(".vibe", "mcp.json")},
	}

	for _, f := range files {
		info, err := os.Stat(f.path)
		if os.IsNotExist(err) {
			s.Checks = append(s.Checks, checkResult{Name: f.name, Status: "skip", Detail: fmt.Sprintf("%s (not found)", f.path)})
			continue
		}
		if err != nil {
			s.Checks = append(s.Checks, checkResult{Name: f.name, Status: "error", Detail: err.Error()})
			continue
		}
		s.Checks = append(s.Checks, checkResult{
			Name:   f.name,
			Status: "ok",
			Detail: fmt.Sprintf("%s (%s)", f.path, formatSize(info.Size())),
		})
	}

	// Validate settings can be parsed
	settings := loadSettingsSilent()
	if settings != nil {
		s.Checks = append(s.Checks, checkResult{Name: "Settings parse", Status: "ok", Detail: "loaded successfully"})
	} else {
		s.Checks = append(s.Checks, checkResult{Name: "Settings parse", Status: "error", Detail: "failed to parse settings"})
	}

	return s
}

// ── Providers & Models ───────────────────────────────────────

func checkProviders() section {
	s := section{Title: "Providers & Models"}

	settings := loadSettingsSilent()
	if settings == nil {
		s.Checks = append(s.Checks, checkResult{Name: "Settings", Status: "error", Detail: "cannot load settings"})
		return s
	}

	s.Checks = append(s.Checks, checkResult{
		Name:   "Default provider",
		Status: "ok",
		Detail: settings.DefaultProvider,
	})
	s.Checks = append(s.Checks, checkResult{
		Name:   "Default model",
		Status: "ok",
		Detail: settings.DefaultModel,
	})
	s.Checks = append(s.Checks, checkResult{
		Name:   "Default mode",
		Status: "ok",
		Detail: valueOrDefault(settings.DefaultMode, "agent"),
	})
	s.Checks = append(s.Checks, checkResult{
		Name:   "Default thinking",
		Status: "ok",
		Detail: valueOrDefault(settings.DefaultThinkingLevel, "off"),
	})

	if len(settings.Providers) == 0 {
		s.Checks = append(s.Checks, checkResult{Name: "Providers", Status: "warn", Detail: "no providers configured"})
		return s
	}

	configuredCount := 0
	for name, pc := range settings.Providers {
		if pc == nil {
			continue
		}

		apiKey := settings.ResolveKey(name)
		if apiKey == "" || strings.HasPrefix(apiKey, "${") {
			continue // skip unconfigured providers
		}

		configuredCount++

		maskedKey := apiKey
		if len(apiKey) > 8 {
			maskedKey = apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
		}

		api := pc.API
		if api == "" {
			api = "(auto-detect)"
		}

		detail := fmt.Sprintf("api=%s, base=%s, key=%s", api, truncateStr(pc.BaseURL, 40), maskedKey)
		s.Checks = append(s.Checks, checkResult{
			Name:   fmt.Sprintf("Provider: %s", name),
			Status: "ok",
			Detail: detail,
		})

		// List models for this provider
		for _, mc := range pc.Models {
			modelDetail := fmt.Sprintf("ctx=%s, max=%s", formatTokenCount(mc.ContextWindow), formatTokenCount(mc.MaxTokens))
			if mc.Reasoning {
				modelDetail += ", reasoning"
			}
			isDefault := name == settings.DefaultProvider && mc.ID == settings.DefaultModel
			if isDefault {
				modelDetail += " ★ default"
			}
			s.Checks = append(s.Checks, checkResult{
				Name:   fmt.Sprintf("  └─ %s", mc.ID),
				Status: "ok",
				Detail: modelDetail,
			})
		}
	}

	if configuredCount == 0 {
		s.Checks = append(s.Checks, checkResult{Name: "Providers", Status: "warn", Detail: "no providers with API keys configured"})
	}

	// Try creating the default provider to verify it works
	_, _, err := providerfactory.Create(settings, settings.DefaultProvider, settings.DefaultModel)
	if err != nil {
		s.Checks = append(s.Checks, checkResult{
			Name:   "Provider init",
			Status: "error",
			Detail: err.Error(),
		})
	} else {
		s.Checks = append(s.Checks, checkResult{
			Name:   "Provider init",
			Status: "ok",
			Detail: fmt.Sprintf("%s/%s created successfully", settings.DefaultProvider, settings.DefaultModel),
		})
	}

	// Env var overrides
	envOverrides := []struct {
		env  string
		name string
	}{
		{"VIBECODING_PROVIDER", "defaultProvider"},
		{"VIBECODING_MODEL", "defaultModel"},
		{"VIBECODING_MODE", "defaultMode"},
		{"VIBECODING_THINKING", "defaultThinkingLevel"},
	}
	for _, eo := range envOverrides {
		if v := os.Getenv(eo.env); v != "" {
			s.Checks = append(s.Checks, checkResult{
				Name:   fmt.Sprintf("Env override: %s", eo.env),
				Status: "warn",
				Detail: fmt.Sprintf("=%s (overrides %s)", v, eo.name),
			})
		}
	}

	return s
}

// ── Sandbox ──────────────────────────────────────────────────

func checkSandbox() section {
	s := section{Title: "Sandbox"}

	if p, err := exec.LookPath("bwrap"); err != nil {
		s.Checks = append(s.Checks, checkResult{
			Name:   "bubblewrap (bwrap)",
			Status: "warn",
			Detail: "not found in PATH — sandbox unavailable",
		})
	} else {
		out, _ := exec.Command(p, "--version").CombinedOutput()
		ver := strings.TrimSpace(string(out))
		s.Checks = append(s.Checks, checkResult{
			Name:   "bubblewrap (bwrap)",
			Status: "ok",
			Detail: fmt.Sprintf("%s (%s)", p, ver),
		})
	}

	settings := loadSettingsSilent()
	if settings != nil {
		s.Checks = append(s.Checks, checkResult{
			Name:   "Sandbox config",
			Status: "ok",
			Detail: fmt.Sprintf("enabled=%v, level=%s", settings.Sandbox.Enabled, valueOrDefault(settings.Sandbox.Level, "none")),
		})
	}

	return s
}

// ── MCP Servers ──────────────────────────────────────────────

func checkMCPServers() section {
	s := section{Title: "MCP Servers"}

	cwd, _ := os.Getwd()
	servers, err := mcp.LoadConfiguredServers(cwd)
	if err != nil {
		s.Checks = append(s.Checks, checkResult{Name: "MCP config", Status: "error", Detail: err.Error()})
		return s
	}

	if len(servers) == 0 {
		s.Checks = append(s.Checks, checkResult{Name: "MCP servers", Status: "skip", Detail: "none configured"})
		return s
	}

	for _, srv := range servers {
		srvType := srv.Type
		if srvType == "" {
			srvType = "stdio"
		}
		var detail string
		if srv.Command != "" {
			detail = fmt.Sprintf("type=%s, cmd=%s", srvType, truncateStr(srv.Command, 40))
		} else if srv.URL != "" {
			detail = fmt.Sprintf("type=%s, url=%s", srvType, truncateStr(srv.URL, 40))
		} else {
			detail = fmt.Sprintf("type=%s", srvType)
		}
		s.Checks = append(s.Checks, checkResult{
			Name:   fmt.Sprintf("Server: %s", srv.Name),
			Status: "ok",
			Detail: detail,
		})
	}

	return s
}

// ── Sessions ─────────────────────────────────────────────────

func checkSessions() section {
	s := section{Title: "Sessions"}

	settings := loadSettingsSilent()
	if settings == nil {
		s.Checks = append(s.Checks, checkResult{Name: "Session dir", Status: "error", Detail: "cannot load settings"})
		return s
	}

	sessionDir := settings.GetSessionDir()
	info, err := os.Stat(sessionDir)
	if os.IsNotExist(err) {
		s.Checks = append(s.Checks, checkResult{
			Name:   "Session directory",
			Status: "skip",
			Detail: fmt.Sprintf("%s (not created yet)", sessionDir),
		})
	} else if err != nil {
		s.Checks = append(s.Checks, checkResult{Name: "Session directory", Status: "error", Detail: err.Error()})
	} else if info.IsDir() {
		entries, _ := os.ReadDir(sessionDir)
		s.Checks = append(s.Checks, checkResult{
			Name:   "Session directory",
			Status: "ok",
			Detail: fmt.Sprintf("%s (%d entries)", sessionDir, len(entries)),
		})
	}

	return s
}

// ── Skills ───────────────────────────────────────────────────

func checkSkills() section {
	s := section{Title: "Skills"}

	settings := loadSettingsSilent()
	if settings == nil {
		s.Checks = append(s.Checks, checkResult{Name: "Skills dir", Status: "error", Detail: "cannot load settings"})
		return s
	}

	skillsDir := settings.GetGlobalSkillsDir()
	info, err := os.Stat(skillsDir)
	if os.IsNotExist(err) {
		s.Checks = append(s.Checks, checkResult{
			Name:   "Global skills dir",
			Status: "skip",
			Detail: fmt.Sprintf("%s (not created)", skillsDir),
		})
	} else if err != nil {
		s.Checks = append(s.Checks, checkResult{Name: "Global skills dir", Status: "error", Detail: err.Error()})
	} else if info.IsDir() {
		entries, _ := os.ReadDir(skillsDir)
		s.Checks = append(s.Checks, checkResult{
			Name:   "Global skills dir",
			Status: "ok",
			Detail: fmt.Sprintf("%s (%d entries)", skillsDir, len(entries)),
		})
	}

	// Project-level skills
	cwd, _ := os.Getwd()
	projSkills := filepath.Join(cwd, ".vibe", "skills")
	if info, err := os.Stat(projSkills); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(projSkills)
		s.Checks = append(s.Checks, checkResult{
			Name:   "Project skills dir",
			Status: "ok",
			Detail: fmt.Sprintf("%s (%d entries)", projSkills, len(entries)),
		})
	}

	return s
}

// ── Context Files ────────────────────────────────────────────

func checkContextFiles() section {
	s := section{Title: "Context Files"}

	settings := loadSettingsSilent()
	if settings != nil {
		s.Checks = append(s.Checks, checkResult{
			Name:   "Context files feature",
			Status: boolToStatus(settings.ContextFiles.Enabled),
			Detail: fmt.Sprintf("enabled=%v", settings.ContextFiles.Enabled),
		})
	}

	cwd, _ := os.Getwd()
	knownFiles := []string{"AGENTS.md", "CLAUDE.md", "CURSOR.md", ".cursorrules", "CONVENTIONS.md"}
	found := 0
	for _, name := range knownFiles {
		path := filepath.Join(cwd, name)
		if info, err := os.Stat(path); err == nil {
			s.Checks = append(s.Checks, checkResult{
				Name:   name,
				Status: "ok",
				Detail: formatSize(info.Size()),
			})
			found++
		}
	}
	if found == 0 {
		s.Checks = append(s.Checks, checkResult{
			Name:   "Project context files",
			Status: "skip",
			Detail: "none found in working directory",
		})
	}

	// Global context files
	globalDir := config.ConfigDir()
	globalFiles := []string{"AGENTS.md", "CLAUDE.md"}
	for _, name := range globalFiles {
		path := filepath.Join(globalDir, name)
		if info, err := os.Stat(path); err == nil {
			s.Checks = append(s.Checks, checkResult{
				Name:   name + " (global)",
				Status: "ok",
				Detail: fmt.Sprintf("%s (%s)", path, formatSize(info.Size())),
			})
		}
	}

	return s
}

// ── Helpers ──────────────────────────────────────────────────

func loadSettingsSilent() *config.Settings {
	s, _ := config.LoadSettings()
	return s
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}


func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func valueOrDefault(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func boolToStatus(v bool) string {
	if v {
		return "ok"
	}
	return "skip"
}
