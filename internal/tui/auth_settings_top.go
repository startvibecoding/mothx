package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/tui/components/editor"
)

func (a *App) authSettingsRootOptions() []authOption {
	s := a.effectiveSettings()
	return []authOption{
		{Title: "Providers", Description: fmt.Sprintf("%d provider(s), default %s / %s", len(s.Providers), valueOrDefault(s.DefaultProvider, "(unset)"), valueOrDefault(s.DefaultModel, "(unset)")), Value: "providers"},
		{Title: "Defaults", Description: fmt.Sprintf("mode=%s  thinking=%s", valueOrDefault(s.DefaultMode, "agent"), valueOrDefault(s.DefaultThinkingLevel, "medium")), Value: "defaults"},
		{Title: "Behavior", Description: fmt.Sprintf("theme=%s  planTool=%s  maxOut=%s", valueOrDefault(s.Theme, "dark"), boolPtrSummary(s.EnablePlanTool, true), authItoa(s.MaxOutputTokens)), Value: "behavior"},
		{Title: "Web Search", Description: fmt.Sprintf("enabled=%s  provider=%s", boolPtrSummary(s.WebSearch.Enabled, false), valueOrDefault(s.WebSearch.Provider, "openai")), Value: "webSearch"},
		{Title: "Context Files", Description: fmt.Sprintf("enabled=%s  extra=%d", boolYesNo(s.ContextFiles.Enabled), len(s.ContextFiles.ExtraFiles)), Value: "contextFiles"},
		{Title: "Status Line", Description: fmt.Sprintf("enabled=%s  type=%s", boolYesNo(s.StatusLine.Enabled), valueOrDefault(s.StatusLine.Type, "command")), Value: "statusLine"},
		{Title: "Compaction", Description: fmt.Sprintf("enabled=%s  reserve=%s  keep=%s", boolYesNo(s.Compaction.Enabled), authItoa(s.Compaction.ReserveTokens), authItoa(s.Compaction.KeepRecentTokens)), Value: "compaction"},
		{Title: "Sandbox", Description: fmt.Sprintf("enabled=%s  level=%s  network=%s", boolYesNo(s.Sandbox.Enabled), valueOrDefault(s.Sandbox.Level, "none"), boolYesNo(s.Sandbox.AllowNetwork)), Value: "sandbox"},
		{Title: "Paths", Description: fmt.Sprintf("sessions=%s", shortSettingValue(s.SessionDir)), Value: "paths"},
		{Title: "Retry", Description: fmt.Sprintf("enabled=%s  max=%d  base=%dms", boolYesNo(s.Retry.Enabled), s.Retry.MaxRetries, s.Retry.BaseDelayMs), Value: "retry"},
		{Title: "Approval", Description: fmt.Sprintf("write=%s  whitelist=%d  blacklist=%d", boolPtrSummary(s.Approval.ConfirmBeforeWrite, true), len(s.Approval.BashWhitelist), len(s.Approval.BashBlacklist)), Value: "approval"},
	}
}

func (a *App) selectSettingsRoot(value string) {
	switch value {
	case "providers":
		a.pushAuthView(authViewExistingProvider)
	case "defaults":
		a.pushAuthView(authViewSettingsDefaults)
	case "behavior":
		a.pushAuthView(authViewSettingsBehavior)
	case "webSearch":
		a.pushAuthView(authViewSettingsWebSearch)
	case "contextFiles":
		a.pushAuthView(authViewSettingsContextFiles)
	case "statusLine":
		a.pushAuthView(authViewSettingsStatusLine)
	case "compaction":
		a.pushAuthView(authViewSettingsCompaction)
	case "sandbox":
		a.pushAuthView(authViewSettingsSandbox)
	case "paths":
		a.pushAuthView(authViewSettingsPaths)
	case "retry":
		a.pushAuthView(authViewSettingsRetry)
	case "approval":
		a.pushAuthView(authViewSettingsApproval)
	}
}

func (a *App) authSettingsTopLevelOptions(v authView) []authOption {
	s := a.effectiveSettings()
	var opts []authOption
	switch v {
	case authViewSettingsDefaults:
		opts = []authOption{
			{Title: "Default Provider / Model", Description: fmt.Sprintf("%s / %s", valueOrDefault(s.DefaultProvider, "(unset)"), valueOrDefault(s.DefaultModel, "(unset)")), Value: "defaults.modelPicker"},
			{Title: "Default Thinking Level", Description: valueOrDefault(s.DefaultThinkingLevel, "medium"), Value: "defaultThinkingLevel"},
			{Title: "Default Mode", Description: valueOrDefault(s.DefaultMode, "agent"), Value: "defaultMode"},
		}
	case authViewSettingsBehavior:
		opts = []authOption{
			{Title: "Theme", Description: valueOrDefault(s.Theme, "dark"), Value: "theme"},
			{Title: "Enable Plan Tool", Description: boolPtrSummary(s.EnablePlanTool, true), Value: "enablePlanTool"},
			{Title: "Max Context Tokens", Description: zeroAsUnset(s.MaxContextTokens), Value: "maxContextTokens"},
			{Title: "Max Output Tokens", Description: zeroAsUnset(s.MaxOutputTokens), Value: "maxOutputTokens"},
			{Title: "Update Check", Description: boolPtrSummary(s.UpdateCheck, true), Value: "updateCheck"},
		}
	case authViewSettingsWebSearch:
		opts = []authOption{
			{Title: "Enabled", Description: boolPtrSummary(s.WebSearch.Enabled, false), Value: "webSearch.enabled"},
			{Title: "Provider", Description: valueOrDefault(s.WebSearch.Provider, "openai"), Value: "webSearch.provider"},
			{Title: "Provider Type", Description: valueOrDefault(s.WebSearch.ProviderType, "responses"), Value: "webSearch.providerType"},
			{Title: "Model", Description: valueOrDefault(s.WebSearch.Model, "(unset)"), Value: "webSearch.model"},
		}
	case authViewSettingsContextFiles:
		opts = []authOption{
			{Title: "Enabled", Description: boolYesNo(s.ContextFiles.Enabled), Value: "contextFiles.enabled"},
			{Title: "Extra Files", Description: listSummary(s.ContextFiles.ExtraFiles), Value: "contextFiles.extraFiles"},
		}
	case authViewSettingsStatusLine:
		opts = []authOption{
			{Title: "Enabled", Description: boolYesNo(s.StatusLine.Enabled), Value: "statusLine.enabled"},
			{Title: "Type", Description: valueOrDefault(s.StatusLine.Type, "command"), Value: "statusLine.type"},
			{Title: "Command", Description: shortSettingValue(valueOrDefault(s.StatusLine.Command, "ccstatusline")), Value: "statusLine.command"},
			{Title: "Padding", Description: authItoa(s.StatusLine.Padding), Value: "statusLine.padding"},
			{Title: "Refresh Interval", Description: fmt.Sprintf("%ds", s.StatusLine.RefreshInterval), Value: "statusLine.refreshInterval"},
			{Title: "Timeout", Description: fmt.Sprintf("%dms", s.StatusLine.TimeoutMs), Value: "statusLine.timeoutMs"},
			{Title: "Fallback", Description: valueOrDefault(s.StatusLine.Fallback, "builtin"), Value: "statusLine.fallback"},
		}
	case authViewSettingsCompaction:
		opts = []authOption{
			{Title: "Enabled", Description: boolYesNo(s.Compaction.Enabled), Value: "compaction.enabled"},
			{Title: "Reserve Tokens", Description: authItoa(s.Compaction.ReserveTokens), Value: "compaction.reserveTokens"},
			{Title: "Keep Recent Tokens", Description: authItoa(s.Compaction.KeepRecentTokens), Value: "compaction.keepRecentTokens"},
			{Title: "Tokenizer", Description: valueOrDefault(s.Compaction.Tokenizer, "(auto)"), Value: "compaction.tokenizer"},
			{Title: "Tokenizer Model", Description: valueOrDefault(s.Compaction.TokenizerModel, "(auto)"), Value: "compaction.tokenizerModel"},
			{Title: "Template", Description: shortSettingValue(s.Compaction.Template), Value: "compaction.template"},
			{Title: "Idle Compression", Description: boolYesNo(s.Compaction.IdleCompressionEnabled), Value: "compaction.idleCompressionEnabled"},
			{Title: "Idle Timeout", Description: fmt.Sprintf("%ds", s.Compaction.IdleTimeoutSeconds), Value: "compaction.idleTimeoutSeconds"},
			{Title: "Idle Min Tokens", Description: authItoa(s.Compaction.IdleMinTokensForCompress), Value: "compaction.idleMinTokensForCompress"},
		}
	case authViewSettingsSandbox:
		opts = []authOption{
			{Title: "Enabled", Description: boolYesNo(s.Sandbox.Enabled), Value: "sandbox.enabled"},
			{Title: "Level", Description: valueOrDefault(s.Sandbox.Level, "none"), Value: "sandbox.level"},
			{Title: "Bwrap Path", Description: valueOrDefault(s.Sandbox.BwrapPath, "(auto)"), Value: "sandbox.bwrapPath"},
			{Title: "Allow Network", Description: boolYesNo(s.Sandbox.AllowNetwork), Value: "sandbox.allowNetwork"},
			{Title: "Allowed Read", Description: listSummary(s.Sandbox.AllowedRead), Value: "sandbox.allowedRead"},
			{Title: "Allowed Write", Description: listSummary(s.Sandbox.AllowedWrite), Value: "sandbox.allowedWrite"},
			{Title: "Denied Paths", Description: listSummary(s.Sandbox.DeniedPaths), Value: "sandbox.deniedPaths"},
			{Title: "Pass Env", Description: listSummary(s.Sandbox.PassEnv), Value: "sandbox.passEnv"},
			{Title: "Tmp Size", Description: valueOrDefault(s.Sandbox.TmpSize, "100m"), Value: "sandbox.tmpSize"},
		}
	case authViewSettingsPaths:
		opts = []authOption{
			{Title: "Session Dir", Description: shortSettingValue(s.SessionDir), Value: "sessionDir"},
			{Title: "Skills Dir", Description: shortSettingValue(s.SkillsDir), Value: "skillsDir"},
			{Title: "Shell Path", Description: valueOrDefault(s.ShellPath, "(default shell)"), Value: "shellPath"},
			{Title: "Shell Command Prefix", Description: valueOrDefault(s.ShellCommandPrefix, "(none)"), Value: "shellCommandPrefix"},
		}
	case authViewSettingsRetry:
		opts = []authOption{
			{Title: "Enabled", Description: boolYesNo(s.Retry.Enabled), Value: "retry.enabled"},
			{Title: "Max Retries", Description: authItoa(s.Retry.MaxRetries), Value: "retry.maxRetries"},
			{Title: "Base Delay", Description: fmt.Sprintf("%dms", s.Retry.BaseDelayMs), Value: "retry.baseDelayMs"},
		}
	case authViewSettingsApproval:
		opts = []authOption{
			{Title: "Confirm Before Write", Description: boolPtrSummary(s.Approval.ConfirmBeforeWrite, true), Value: "approval.confirmBeforeWrite"},
			{Title: "Bash Whitelist", Description: listSummary(s.Approval.BashWhitelist), Value: "approval.bashWhitelist"},
			{Title: "Bash Blacklist", Description: listSummary(s.Approval.BashBlacklist), Value: "approval.bashBlacklist"},
		}
	}
	opts = append(opts, authOption{Title: "Done", Description: "Return to Settings", Value: "done"})
	return opts
}

func (a *App) selectSettingsFieldValue(value string) {
	a.auth.Error = ""
	if value == "done" {
		a.popAuthView()
		return
	}
	if value == "defaults.modelPicker" {
		a.closeAuthDialog()
		a.openDefaultModelDialog("global")
		return
	}

	next := a.cloneEffectiveSettings()
	switch value {
	case "defaultThinkingLevel":
		next.DefaultThinkingLevel = cycleString(next.DefaultThinkingLevel, []string{"off", "minimal", "low", "medium", "high", "xhigh"}, "medium")
		a.saveAuthSettingsPatch("defaultThinkingLevel", map[string]any{"defaultThinkingLevel": next.DefaultThinkingLevel})
	case "defaultMode":
		next.DefaultMode = cycleString(next.DefaultMode, []string{"plan", "agent", "yolo"}, "agent")
		a.saveAuthSettingsPatch("defaultMode", map[string]any{"defaultMode": next.DefaultMode})
	case "enablePlanTool":
		next.EnablePlanTool = cycleSettingsBoolPtr(next.EnablePlanTool, true)
		a.saveAuthSettingsPatch("enablePlanTool", map[string]any{"enablePlanTool": next.EnablePlanTool})
	case "updateCheck":
		next.UpdateCheck = cycleSettingsBoolPtr(next.UpdateCheck, true)
		a.saveAuthSettingsPatch("updateCheck", map[string]any{"updateCheck": next.UpdateCheck})
	case "webSearch.enabled":
		next.WebSearch.Enabled = cycleSettingsBoolPtr(next.WebSearch.Enabled, false)
		a.saveAuthSettingsPatch("webSearch.enabled", map[string]any{"webSearch": next.WebSearch})
	case "contextFiles.enabled":
		next.ContextFiles.Enabled = !next.ContextFiles.Enabled
		a.saveAuthSettingsPatch("contextFiles.enabled", map[string]any{"contextFiles": next.ContextFiles})
	case "statusLine.enabled":
		next.StatusLine.Enabled = !next.StatusLine.Enabled
		normalizeStatusLineDefaults(&next.StatusLine)
		a.saveAuthSettingsPatch("statusLine.enabled", map[string]any{"statusLine": next.StatusLine})
	case "compaction.enabled":
		next.Compaction.Enabled = !next.Compaction.Enabled
		a.saveAuthSettingsPatch("compaction.enabled", map[string]any{"compaction": next.Compaction})
	case "compaction.idleCompressionEnabled":
		next.Compaction.IdleCompressionEnabled = !next.Compaction.IdleCompressionEnabled
		a.saveAuthSettingsPatch("compaction.idleCompressionEnabled", map[string]any{"compaction": next.Compaction})
	case "sandbox.enabled":
		next.Sandbox.Enabled = !next.Sandbox.Enabled
		a.saveAuthSettingsPatch("sandbox.enabled", map[string]any{"sandbox": next.Sandbox})
	case "sandbox.level":
		next.Sandbox.Level = cycleString(next.Sandbox.Level, []string{"none", "standard", "strict"}, "none")
		a.saveAuthSettingsPatch("sandbox.level", map[string]any{"sandbox": next.Sandbox})
	case "sandbox.allowNetwork":
		next.Sandbox.AllowNetwork = !next.Sandbox.AllowNetwork
		a.saveAuthSettingsPatch("sandbox.allowNetwork", map[string]any{"sandbox": next.Sandbox})
	case "retry.enabled":
		next.Retry.Enabled = !next.Retry.Enabled
		a.saveAuthSettingsPatch("retry.enabled", map[string]any{"retry": next.Retry})
	case "approval.confirmBeforeWrite":
		next.Approval.ConfirmBeforeWrite = cycleSettingsBoolPtr(next.Approval.ConfirmBeforeWrite, true)
		a.saveAuthSettingsPatch("approval.confirmBeforeWrite", map[string]any{"approval": next.Approval})
	default:
		a.auth.ParamField = value
		a.prepareAuthSettingsInput()
	}
}

func (a *App) prepareAuthSettingsInput() {
	prompt := a.authSettingsInputPrompt()
	value := a.authSettingsInputValue()
	a.authInput = editor.New(max(20, a.width-8)).SetPlaceholder(prompt).SetMaxLines(3).Focus()
	if value != "" {
		a.authInput = a.authInput.SetValue(value)
	}
}

func (a *App) authSettingsInputPrompt() string {
	switch a.auth.ParamField {
	case "theme":
		return "Enter theme:"
	case "maxContextTokens":
		return "Enter max context tokens (0 = unset):"
	case "maxOutputTokens":
		return "Enter max output tokens (0 = unset):"
	case "webSearch.provider":
		return "Enter web search provider:"
	case "webSearch.providerType":
		return "Enter web search provider type:"
	case "webSearch.model":
		return "Enter web search model (empty = unset):"
	case "contextFiles.extraFiles":
		return "Enter extra context files, comma or newline separated:"
	case "statusLine.type":
		return "Enter status line type:"
	case "statusLine.command":
		return "Enter status line command:"
	case "statusLine.padding":
		return "Enter status line padding:"
	case "statusLine.refreshInterval":
		return "Enter refresh interval seconds (0 = event-driven):"
	case "statusLine.timeoutMs":
		return "Enter timeout in milliseconds:"
	case "statusLine.fallback":
		return "Enter status line fallback:"
	case "compaction.reserveTokens":
		return "Enter reserve tokens:"
	case "compaction.keepRecentTokens":
		return "Enter keep recent tokens:"
	case "compaction.tokenizer":
		return "Enter tokenizer (empty = auto):"
	case "compaction.tokenizerModel":
		return "Enter tokenizer model (empty = auto):"
	case "compaction.template":
		return "Enter compaction template (empty = default):"
	case "compaction.idleTimeoutSeconds":
		return "Enter idle timeout seconds:"
	case "compaction.idleMinTokensForCompress":
		return "Enter idle min tokens:"
	case "sandbox.bwrapPath":
		return "Enter bwrap path (empty = auto):"
	case "sandbox.allowedRead", "sandbox.allowedWrite", "sandbox.deniedPaths", "sandbox.passEnv":
		return "Enter values, comma or newline separated:"
	case "sandbox.tmpSize":
		return "Enter tmp size:"
	case "sessionDir":
		return "Enter session directory:"
	case "skillsDir":
		return "Enter skills directory:"
	case "shellPath":
		return "Enter shell path (empty = default shell):"
	case "shellCommandPrefix":
		return "Enter shell command prefix (empty = none):"
	case "retry.maxRetries":
		return "Enter max retries:"
	case "retry.baseDelayMs":
		return "Enter base delay in milliseconds:"
	case "approval.bashWhitelist", "approval.bashBlacklist":
		return "Enter one command prefix per line. Trailing spaces are significant:"
	default:
		return "Input:"
	}
}

func (a *App) authSettingsInputValue() string {
	s := a.effectiveSettings()
	switch a.auth.ParamField {
	case "theme":
		return s.Theme
	case "maxContextTokens":
		return intInputValue(s.MaxContextTokens)
	case "maxOutputTokens":
		return intInputValue(s.MaxOutputTokens)
	case "webSearch.provider":
		return s.WebSearch.Provider
	case "webSearch.providerType":
		return s.WebSearch.ProviderType
	case "webSearch.model":
		return s.WebSearch.Model
	case "contextFiles.extraFiles":
		return strings.Join(s.ContextFiles.ExtraFiles, ", ")
	case "statusLine.type":
		return s.StatusLine.Type
	case "statusLine.command":
		return s.StatusLine.Command
	case "statusLine.padding":
		return intInputValue(s.StatusLine.Padding)
	case "statusLine.refreshInterval":
		return intInputValue(s.StatusLine.RefreshInterval)
	case "statusLine.timeoutMs":
		return intInputValue(s.StatusLine.TimeoutMs)
	case "statusLine.fallback":
		return s.StatusLine.Fallback
	case "compaction.reserveTokens":
		return intInputValue(s.Compaction.ReserveTokens)
	case "compaction.keepRecentTokens":
		return intInputValue(s.Compaction.KeepRecentTokens)
	case "compaction.tokenizer":
		return s.Compaction.Tokenizer
	case "compaction.tokenizerModel":
		return s.Compaction.TokenizerModel
	case "compaction.template":
		return s.Compaction.Template
	case "compaction.idleTimeoutSeconds":
		return intInputValue(s.Compaction.IdleTimeoutSeconds)
	case "compaction.idleMinTokensForCompress":
		return intInputValue(s.Compaction.IdleMinTokensForCompress)
	case "sandbox.bwrapPath":
		return s.Sandbox.BwrapPath
	case "sandbox.allowedRead":
		return strings.Join(s.Sandbox.AllowedRead, ", ")
	case "sandbox.allowedWrite":
		return strings.Join(s.Sandbox.AllowedWrite, ", ")
	case "sandbox.deniedPaths":
		return strings.Join(s.Sandbox.DeniedPaths, ", ")
	case "sandbox.passEnv":
		return strings.Join(s.Sandbox.PassEnv, ", ")
	case "sandbox.tmpSize":
		return s.Sandbox.TmpSize
	case "sessionDir":
		return s.SessionDir
	case "skillsDir":
		return s.SkillsDir
	case "shellPath":
		return s.ShellPath
	case "shellCommandPrefix":
		return s.ShellCommandPrefix
	case "retry.maxRetries":
		return intInputValue(s.Retry.MaxRetries)
	case "retry.baseDelayMs":
		return intInputValue(s.Retry.BaseDelayMs)
	case "approval.bashWhitelist":
		return strings.Join(s.Approval.BashWhitelist, "\n")
	case "approval.bashBlacklist":
		return strings.Join(s.Approval.BashBlacklist, "\n")
	default:
		return ""
	}
}

func (a *App) authSettingsSubmitInput() error {
	field := a.auth.ParamField
	rawValue := a.authInput.Value()
	value := strings.TrimSpace(rawValue)
	next := a.cloneEffectiveSettings()
	updates := map[string]any{}

	switch field {
	case "theme":
		next.Theme = value
		updates["theme"] = next.Theme
	case "maxContextTokens":
		v, err := parseNonNegativeInt(value)
		if err != nil {
			return err
		}
		next.MaxContextTokens = v
		updates["maxContextTokens"] = next.MaxContextTokens
	case "maxOutputTokens":
		v, err := parseNonNegativeInt(value)
		if err != nil {
			return err
		}
		next.MaxOutputTokens = v
		updates["maxOutputTokens"] = next.MaxOutputTokens
	case "webSearch.provider":
		next.WebSearch.Provider = value
		updates["webSearch"] = next.WebSearch
	case "webSearch.providerType":
		next.WebSearch.ProviderType = value
		updates["webSearch"] = next.WebSearch
	case "webSearch.model":
		next.WebSearch.Model = value
		updates["webSearch"] = next.WebSearch
	case "contextFiles.extraFiles":
		next.ContextFiles.ExtraFiles = parseSettingsList(value)
		updates["contextFiles"] = next.ContextFiles
	case "statusLine.type":
		next.StatusLine.Type = value
		normalizeStatusLineDefaults(&next.StatusLine)
		updates["statusLine"] = next.StatusLine
	case "statusLine.command":
		next.StatusLine.Command = value
		normalizeStatusLineDefaults(&next.StatusLine)
		updates["statusLine"] = next.StatusLine
	case "statusLine.padding":
		v, err := parseNonNegativeInt(value)
		if err != nil {
			return err
		}
		next.StatusLine.Padding = v
		updates["statusLine"] = next.StatusLine
	case "statusLine.refreshInterval":
		v, err := parseNonNegativeInt(value)
		if err != nil {
			return err
		}
		next.StatusLine.RefreshInterval = v
		updates["statusLine"] = next.StatusLine
	case "statusLine.timeoutMs":
		v, err := parsePositiveInt(value)
		if err != nil {
			return err
		}
		next.StatusLine.TimeoutMs = v
		updates["statusLine"] = next.StatusLine
	case "statusLine.fallback":
		next.StatusLine.Fallback = value
		normalizeStatusLineDefaults(&next.StatusLine)
		updates["statusLine"] = next.StatusLine
	case "compaction.reserveTokens":
		v, err := parseNonNegativeInt(value)
		if err != nil {
			return err
		}
		next.Compaction.ReserveTokens = v
		updates["compaction"] = next.Compaction
	case "compaction.keepRecentTokens":
		v, err := parseNonNegativeInt(value)
		if err != nil {
			return err
		}
		next.Compaction.KeepRecentTokens = v
		updates["compaction"] = next.Compaction
	case "compaction.tokenizer":
		next.Compaction.Tokenizer = value
		updates["compaction"] = next.Compaction
	case "compaction.tokenizerModel":
		next.Compaction.TokenizerModel = value
		updates["compaction"] = next.Compaction
	case "compaction.template":
		next.Compaction.Template = value
		updates["compaction"] = next.Compaction
	case "compaction.idleTimeoutSeconds":
		v, err := parseNonNegativeInt(value)
		if err != nil {
			return err
		}
		next.Compaction.IdleTimeoutSeconds = v
		updates["compaction"] = next.Compaction
	case "compaction.idleMinTokensForCompress":
		v, err := parseNonNegativeInt(value)
		if err != nil {
			return err
		}
		next.Compaction.IdleMinTokensForCompress = v
		updates["compaction"] = next.Compaction
	case "sandbox.bwrapPath":
		next.Sandbox.BwrapPath = value
		updates["sandbox"] = next.Sandbox
	case "sandbox.allowedRead":
		next.Sandbox.AllowedRead = parseSettingsList(value)
		updates["sandbox"] = next.Sandbox
	case "sandbox.allowedWrite":
		next.Sandbox.AllowedWrite = parseSettingsList(value)
		updates["sandbox"] = next.Sandbox
	case "sandbox.deniedPaths":
		next.Sandbox.DeniedPaths = parseSettingsList(value)
		updates["sandbox"] = next.Sandbox
	case "sandbox.passEnv":
		next.Sandbox.PassEnv = parseSettingsList(value)
		updates["sandbox"] = next.Sandbox
	case "sandbox.tmpSize":
		next.Sandbox.TmpSize = value
		updates["sandbox"] = next.Sandbox
	case "sessionDir":
		next.SessionDir = value
		updates["sessionDir"] = next.SessionDir
	case "skillsDir":
		next.SkillsDir = value
		updates["skillsDir"] = next.SkillsDir
	case "shellPath":
		next.ShellPath = value
		updates["shellPath"] = next.ShellPath
	case "shellCommandPrefix":
		next.ShellCommandPrefix = value
		updates["shellCommandPrefix"] = next.ShellCommandPrefix
	case "retry.maxRetries":
		v, err := parseNonNegativeInt(value)
		if err != nil {
			return err
		}
		next.Retry.MaxRetries = v
		updates["retry"] = next.Retry
	case "retry.baseDelayMs":
		v, err := parseNonNegativeInt(value)
		if err != nil {
			return err
		}
		next.Retry.BaseDelayMs = v
		updates["retry"] = next.Retry
	case "approval.bashWhitelist":
		next.Approval.BashWhitelist = parseApprovalPrefixes(rawValue)
		updates["approval"] = next.Approval
	case "approval.bashBlacklist":
		next.Approval.BashBlacklist = parseApprovalPrefixes(rawValue)
		updates["approval"] = next.Approval
	default:
		return errors.New("unknown settings field")
	}

	if err := a.saveAuthSettingsPatch(field, updates); err != nil {
		return err
	}
	a.clearAuthParamField()
	return nil
}

func (a *App) saveAuthSettingsPatch(label string, updates map[string]any) error {
	if err := config.SaveGlobalSettingsPatch(updates); err != nil {
		a.auth.Error = fmt.Sprintf("Failed to save settings: %v", err)
		a.scheduleRender()
		return err
	}
	effective, err := config.LoadSettings()
	if err != nil {
		a.auth.Error = fmt.Sprintf("Failed to reload settings: %v", err)
		a.scheduleRender()
		return err
	}
	a.settings = effective
	a.applyRuntimeSettingsAfterSave(label, effective)
	a.auth.Error = ""
	a.addCommandStatus(fmt.Sprintf("Settings saved: %s", label))
	return nil
}

func (a *App) applyRuntimeSettingsAfterSave(label string, effective *config.Settings) {
	if label == "defaultMode" && effective != nil && strings.TrimSpace(effective.DefaultMode) != "" {
		a.mode = effective.DefaultMode
	}
	if strings.HasPrefix(label, "statusLine.") {
		a.statusLineIntervalInit = false
		if !a.statusLineEnabled() {
			a.statusLineOutput = ""
			a.statusLineLastError = ""
			a.statusLineLastSuccess = ""
			a.statusLineLastAttempt = ""
			a.statusLinePending = nil
			a.statusLineInFlight = false
		} else if a.ready && a.width > 0 {
			a.requestStatusLineRefresh(true)
		}
	}
	if strings.HasPrefix(label, "sandbox.") || label == "sessionDir" || label == "skillsDir" || label == "shellPath" || label == "shellCommandPrefix" {
		a.addCommandStatus("Note: /reload may be needed for this setting to fully affect existing tools or sessions.")
	}
	a.resetAgent(fmt.Errorf("settings changed"))
	a.scheduleRender()
}

func (a *App) effectiveSettings() *config.Settings {
	if a.settings != nil {
		return a.settings
	}
	return config.DefaultSettings()
}

func (a *App) cloneEffectiveSettings() *config.Settings {
	src := a.effectiveSettings()
	data, err := json.Marshal(src)
	if err != nil {
		cp := *src
		return &cp
	}
	var out config.Settings
	if err := json.Unmarshal(data, &out); err != nil {
		cp := *src
		return &cp
	}
	return &out
}

func normalizeStatusLineDefaults(s *config.StatusLineSettings) {
	if s == nil {
		return
	}
	if s.Type == "" {
		s.Type = "command"
	}
	if s.Enabled && strings.TrimSpace(s.Command) == "" {
		s.Command = "ccstatusline"
	}
	if s.TimeoutMs == 0 {
		s.TimeoutMs = 800
	}
	if s.Fallback == "" {
		s.Fallback = "builtin"
	}
}

func parseNonNegativeInt(s string) (int, error) {
	if strings.TrimSpace(s) == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v < 0 {
		return 0, fmt.Errorf("invalid non-negative integer")
	}
	return v, nil
}

func parseSettingsList(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '\n' || r == '\t' })
	seen := map[string]bool{}
	var out []string
	for _, f := range fields {
		value := strings.TrimSpace(f)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func parseApprovalPrefixes(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	seen := map[string]bool{}
	var out []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || seen[line] {
			continue
		}
		seen[line] = true
		out = append(out, line)
	}
	return out
}

func cycleString(current string, values []string, fallback string) string {
	current = strings.TrimSpace(current)
	for i, value := range values {
		if current == value {
			return values[(i+1)%len(values)]
		}
	}
	return fallback
}

func cycleSettingsBoolPtr(v *bool, defaultValue bool) *bool {
	if v == nil {
		b := !defaultValue
		return &b
	}
	if *v != defaultValue {
		b := defaultValue
		return &b
	}
	return nil
}

func boolPtrSummary(v *bool, defaultValue bool) string {
	if v == nil {
		if defaultValue {
			return "auto(on)"
		}
		return "auto(off)"
	}
	if *v {
		return "on"
	}
	return "off"
}

func zeroAsUnset(v int) string {
	if v == 0 {
		return "unset"
	}
	return strconv.Itoa(v)
}

func intInputValue(v int) string {
	return strconv.Itoa(v)
}

func listSummary(values []string) string {
	if len(values) == 0 {
		return "(empty)"
	}
	if len(values) == 1 {
		return shortSettingValue(values[0])
	}
	return fmt.Sprintf("%d entries", len(values))
}

func shortSettingValue(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(unset)"
	}
	if len([]rune(s)) <= 42 {
		return s
	}
	r := []rune(s)
	return string(r[:39]) + "..."
}
