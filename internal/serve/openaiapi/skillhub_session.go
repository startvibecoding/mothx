package openaiapi

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/skills"
)

// SkillHubRuntime contains the serve settings needed by marketplace handlers.
type SkillHubRuntime struct {
	GlobalSkillsDir string
	DefaultWorkDir  string
	DefaultMarket   string
	DefaultScope    string
	OfficialHandles []string
}

// SkillHubSessionState reports marketplace activation state for one API session.
type SkillHubSessionState struct {
	SessionID    string   `json:"sessionId"`
	WorkDir      string   `json:"workDir"`
	ActiveSkills []string `json:"activeSkills"`
}

// SkillHubRuntime returns a snapshot of marketplace-related runtime settings.
func (s *Server) SkillHubRuntime() SkillHubRuntime {
	if s == nil {
		return SkillHubRuntime{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	runtime := SkillHubRuntime{}
	if s.cfg != nil {
		runtime.DefaultWorkDir = s.cfg.GetWorkDir()
	}
	if s.settings != nil {
		runtime.GlobalSkillsDir = s.settings.GetGlobalSkillsDir()
		runtime.DefaultMarket = s.settings.SkillHub.DefaultMarket
		runtime.DefaultScope = s.settings.SkillHub.DefaultInstallScope
		runtime.OfficialHandles = append([]string(nil), s.settings.SkillHub.OfficialHandles...)
	}
	if runtime.DefaultMarket == "" {
		runtime.DefaultMarket = "skillhub.cn"
	}
	if runtime.DefaultScope == "" {
		runtime.DefaultScope = "project"
	}
	if len(runtime.OfficialHandles) == 0 {
		runtime.OfficialHandles = []string{config.DefaultSkillHubOfficialHandle}
	}
	return runtime
}

// ResolveSkillHubWorkDir resolves a request workDir through the existing serve whitelist.
func (s *Server) ResolveSkillHubWorkDir(sessionID, requested string) (string, error) {
	if s == nil || s.cfg == nil {
		return "", errors.New("API server not ready")
	}
	if sessionID != "" {
		workDir, found, err := s.findSessionWorkDir(sessionID)
		if err != nil {
			return "", err
		}
		if found {
			if requested != "" && !sameWorkDir(requested, workDir) {
				return "", fmt.Errorf("workDir %q does not match session %q workDir %q", requested, sessionID, workDir)
			}
			if err := s.validatePersistedSessionWorkDir(workDir); err != nil {
				return "", err
			}
			return filepath.Clean(workDir), nil
		}
	}
	workDir := requested
	if workDir == "" {
		workDir = s.cfg.GetWorkDir()
	}
	if !sameWorkDir(workDir, s.cfg.GetWorkDir()) {
		if err := s.cfg.ValidateWorkDir(workDir); err != nil {
			return "", err
		}
	}
	return filepath.Clean(workDir), nil
}

// RefreshSkillHubSession reloads local skills and optionally activates one skill.
func (s *Server) RefreshSkillHubSession(sessionID, requestedWorkDir, activate string) (*SkillHubSessionState, error) {
	workDir, err := s.ResolveSkillHubWorkDir(sessionID, requestedWorkDir)
	if err != nil {
		return nil, err
	}
	sess, err := s.getOrCreateSession(sessionID, workDir)
	if err != nil {
		return nil, err
	}
	if !s.pool.Pin(sess) {
		return nil, errors.New("session is no longer active")
	}
	defer s.pool.Unpin(sess)
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if activate != "" {
		if err := s.activateSkillForSession(sess, activate); err != nil {
			return nil, err
		}
	} else if err := s.refreshSessionContext(sess); err != nil {
		return nil, err
	}
	return skillHubSessionState(sess), nil
}

func (s *Server) activateSkillForSession(sess *APISession, name string) error {
	if sess == nil {
		return errors.New("no active session")
	}
	if sess.ActiveSkills == nil {
		sess.ActiveSkills = make(map[string]bool)
	}
	previous := sess.ActiveSkills[name]
	sess.ActiveSkills[name] = true
	if err := s.refreshSessionContext(sess); err != nil {
		if !previous {
			delete(sess.ActiveSkills, name)
		}
		return err
	}
	if sess.SkillsMgr == nil || sess.SkillsMgr.Get(name) == nil {
		if !previous {
			delete(sess.ActiveSkills, name)
			_ = s.refreshSessionContext(sess)
		}
		return fmt.Errorf("skill not found: %s", name)
	}
	return nil
}

func buildActiveSkillsContext(manager *skills.Manager, active map[string]bool) (string, error) {
	if len(active) == 0 {
		return "", nil
	}
	names := make([]string, 0, len(active))
	for name, enabled := range active {
		if enabled {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var context strings.Builder
	for _, name := range names {
		if manager == nil || manager.Get(name) == nil {
			return "", fmt.Errorf("skill not found: %s", name)
		}
		context.WriteString(manager.BuildSkillContext(name))
	}
	return context.String(), nil
}

func skillHubSessionState(sess *APISession) *SkillHubSessionState {
	names := make([]string, 0, len(sess.ActiveSkills))
	for name, enabled := range sess.ActiveSkills {
		if enabled {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return &SkillHubSessionState{SessionID: sess.ID, WorkDir: sess.WorkDir, ActiveSkills: names}
}
