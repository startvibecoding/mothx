package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/startvibecoding/vibecoding/internal/config"
	providerfactory "github.com/startvibecoding/vibecoding/internal/provider/factory"
	"github.com/startvibecoding/vibecoding/internal/tui/components/editor"
)

type authView int

const (
	authViewMain authView = iota
	authViewExistingProvider
	authViewCustomID
	authViewDefault
	authViewReview
	authViewEditMenu
	// Provider group views
	authViewProviderGroupList
	authViewProviderCredentials
	authViewProviderProtocol
	authViewProviderNetwork
	authViewProviderAdvanced
	authViewHeadersEdit
	authViewResponsesEdit
	authViewAPIChoice
	// Model group views
	authViewModelList
	authViewModelGroupList
	authViewModelBasics
	authViewModelCapabilities
	authViewModelSampling
	authViewModelCost
	authViewModelCompat
	authViewAddModelID
	authViewAddModelName
	authViewSettingsDetail
)

type authOption struct {
	Title       string
	Description string
	Value       string
}

type authDialogState struct {
	Open bool
	View authView

	Cursor int
	Stack  []authView
	Mode   string // existing/custom

	ProviderID string
	Search     string
	SetDefault bool

	ParamField string // current input field name (for provider/model sub-forms)

	// Structured editing state
	Provider       providerEditState
	Models         map[string]*modelEditState // keyed by model ID
	ModelOrder     []string                   // stable iteration order
	CurrentModelID string                     // model currently being edited
	ParamFieldKey  string                     // auxiliary key for multi-field inputs (e.g. header name)
	PreviewExpand  previewExpansion           // which sections are expanded in Review JSON
	Error          string
	Preview        string
}

const (
	authMaxVisibleOptions      = 5
	authMaxPreviewVisibleLines = 18
)

var authDialogStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("63")).
	Padding(1, 2)

func (a *App) openAuthDialog() {
	a.auth = authDialogState{Open: true, View: authViewMain, SetDefault: true}
	a.authInput = editor.New(max(20, a.width-8)).SetMaxLines(3)
	a.input = a.input.Blur()
	a.authInput = a.authInput.Focus()
	a.scheduleRender()
}

func (a *App) closeAuthDialog() {
	a.auth = authDialogState{}
	a.input = a.input.Focus()
	a.scheduleRender()
}

func (a *App) pushAuthView(v authView) {
	a.auth.Stack = append(a.auth.Stack, a.auth.View)
	a.auth.View = v
	a.auth.Cursor = 0
	a.auth.Error = ""
	a.prepareAuthInput()
}

func (a *App) popAuthView() {
	if len(a.auth.Stack) == 0 {
		a.closeAuthDialog()
		return
	}
	last := a.auth.Stack[len(a.auth.Stack)-1]
	a.auth.Stack = a.auth.Stack[:len(a.auth.Stack)-1]
	a.auth.View = last
	a.auth.Cursor = 0
	a.auth.Error = ""
	if a.auth.View == authViewReview {
		a.prepareAuthPreview()
	}
	a.prepareAuthInput()
}

func (a *App) prepareAuthInput() {
	switch a.auth.View {
	case authViewCustomID:
		a.authInput = editor.New(max(20, a.width-8)).SetPlaceholder("provider-id (e.g. openrouter)").SetMaxLines(3).SetValue(a.auth.ProviderID).Focus()
	case authViewAddModelID:
		a.authInput = editor.New(max(20, a.width-8)).SetPlaceholder("model-id").SetMaxLines(3).Focus()
	case authViewAddModelName:
		a.authInput = editor.New(max(20, a.width-8)).SetPlaceholder(a.auth.CurrentModelID).SetMaxLines(3).Focus()
	case authViewModelBasics, authViewModelCapabilities,
		authViewModelSampling, authViewModelCost, authViewModelCompat:
		a.prepareModelInput()
	default:
		a.prepareAuthProviderInput()
	}
}

func (a *App) handleAuthKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if !a.auth.Open {
		return false, nil
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		a.closeAuthDialog()
		return true, nil
	case tea.KeyEsc:
		if a.auth.View == authViewExistingProvider && a.auth.Search != "" {
			a.auth.Search = ""
			a.auth.Cursor = 0
			a.scheduleRender()
			return true, nil
		}
		a.popAuthView()
		return true, nil
	}

	if a.authInputActive() {
		switch msg.Type {
		case tea.KeyEnter:
			a.submitAuthInput()
			return true, nil
		default:
			var cmd tea.Cmd
			a.authInput, cmd = a.authInput.Update(msg)
			a.scheduleRender()
			return true, cmd
		}
	}

	switch msg.Type {
	case tea.KeyBackspace:
		if a.auth.View == authViewExistingProvider && a.auth.Search != "" {
			r := []rune(a.auth.Search)
			a.auth.Search = string(r[:len(r)-1])
			a.auth.Cursor = 0
			a.scheduleRender()
			return true, nil
		}
		// Delete header when Backspace on a header entry
		if a.auth.View == authViewHeadersEdit && !a.authInputActive() {
			opts := a.authOptions()
			if a.auth.Cursor >= 0 && a.auth.Cursor < len(opts) && strings.HasPrefix(opts[a.auth.Cursor].Value, "edit:") {
				key := strings.TrimPrefix(opts[a.auth.Cursor].Value, "edit:")
				delete(a.auth.Provider.Headers, key)
				if len(a.auth.Provider.Headers) == 0 {
					a.auth.Provider.Headers = nil
				}
				// Adjust cursor
				if a.auth.Cursor >= len(opts)-1 {
					a.auth.Cursor = max(0, len(opts)-3) // stay on last header or 0
				}
				a.scheduleRender()
				return true, nil
			}
		}
	case tea.KeyRunes:
		if a.auth.View == authViewExistingProvider && len(msg.Runes) > 0 {
			a.auth.Search += string(msg.Runes)
			a.auth.Cursor = 0
			a.scheduleRender()
			return true, nil
		}
	case tea.KeyUp:
		a.moveAuthCursor(-1)
		return true, nil
	case tea.KeyDown:
		a.moveAuthCursor(1)
		return true, nil
	case tea.KeySpace:
		// In compat view (non-input mode), Space toggles the tri-state field
		if a.auth.View == authViewModelCompat && a.auth.ParamFieldKey == "tristate" {
			a.toggleModelTriState(a.auth.ParamField)
			a.scheduleRender()
			return true, nil
		}
		fallthrough
	case tea.KeyEnter:
		a.selectAuthOption()
		return true, nil
	}
	return true, nil
}

func (a *App) authInputActive() bool {
	switch a.auth.View {
	case authViewCustomID, authViewAddModelID, authViewAddModelName:
		return true
	case authViewProviderCredentials, authViewProviderProtocol, authViewProviderNetwork,
		authViewProviderAdvanced, authViewResponsesEdit:
		return a.auth.ParamField != ""
	case authViewHeadersEdit:
		return a.auth.ParamField == "headerKey" || a.auth.ParamField == "headerValue"
	case authViewModelBasics, authViewModelCapabilities, authViewModelSampling,
		authViewModelCost, authViewModelCompat:
		return a.auth.ParamField != ""
	default:
		return false
	}
}

func (a *App) moveAuthCursor(delta int) {
	opts := a.authOptions()
	if len(opts) == 0 {
		return
	}
	a.auth.Cursor += delta
	if a.auth.Cursor < 0 {
		a.auth.Cursor = len(opts) - 1
	}
	if a.auth.Cursor >= len(opts) {
		a.auth.Cursor = 0
	}
	a.scheduleRender()
}

func (a *App) selectAuthOption() {
	opts := a.authOptions()
	if len(opts) == 0 || a.auth.Cursor < 0 || a.auth.Cursor >= len(opts) {
		return
	}
	opt := opts[a.auth.Cursor]
	switch a.auth.View {
	case authViewMain:
		switch opt.Value {
		case "existing":
			a.auth.Mode = "existing"
			a.pushAuthView(authViewExistingProvider)
		case "custom":
			a.auth.Mode = "custom"
			a.pushAuthView(authViewCustomID)
		}
	case authViewExistingProvider:
		if opt.Value == "back" {
			a.popAuthView()
			return
		}
		a.auth.ProviderID = opt.Value
		a.initAuthForProvider(opt.Value)
		a.pushAuthView(authViewProviderGroupList)
		return
	case authViewProviderGroupList:
		switch opt.Value {
		case "models":
			a.pushAuthView(authViewModelList)
		case "api":
			a.pushAuthView(authViewAPIChoice)
		case "done":
			a.saveAuthProvider()
			return
		case "sep":
			// separator, do nothing
		default:
			a.pushAuthView(authViewProviderFromID(opt.Value))
		}
	case authViewProviderCredentials, authViewProviderProtocol,
		authViewProviderNetwork, authViewProviderAdvanced:
		a.selectProviderFieldValue(opt.Value)
	case authViewResponsesEdit:
		a.selectProviderFieldValue(opt.Value)
	case authViewHeadersEdit:
		a.selectHeaderValue(opt.Value)
	case authViewAPIChoice:
		a.selectAPIChoice(opt.Value)
	case authViewModelGroupList:
		if opt.Value == "done" {
			a.pushAuthView(authViewModelList)
			return
		}
		a.pushAuthView(authModelGroupFromID(opt.Value))
	case authViewModelList:
		a.selectModelList(opt.Value)
	case authViewModelBasics, authViewModelCapabilities,
		authViewModelSampling, authViewModelCost, authViewModelCompat:
		a.selectModelFieldValue(opt.Value)
	case authViewDefault:
		a.auth.SetDefault = opt.Value == "yes"
		a.prepareAuthPreview()
		if a.isReviewEdit() {
			a.returnToReviewAfterEdit()
		} else {
			a.pushAuthView(authViewReview)
		}
	case authViewReview:
		switch opt.Value {
		case "save":
			a.saveAuthProvider()
		case "edit":
			a.pushAuthView(authViewEditMenu)
		}
	case authViewEditMenu:
		a.jumpAuthEdit(opt.Value)
	case authViewSettingsDetail:
		a.selectSettingsDetail(opt.Value)
	}
	a.scheduleRender()
}

func (a *App) returnToReviewAfterEdit() bool {
	if !a.isReviewEdit() {
		return false
	}
	a.auth.Stack = nil
	a.auth.View = authViewReview
	a.auth.Cursor = 0
	a.auth.Error = ""
	a.prepareAuthPreview()
	a.prepareAuthInput()
	a.scheduleRender()
	return true
}

func (a *App) isReviewEdit() bool {
	for _, v := range a.auth.Stack {
		if v == authViewEditMenu {
			return true
		}
	}
	return false
}

func (a *App) jumpAuthEdit(value string) {
	a.auth.Error = ""
	switch value {
	case "providerID":
		a.pushAuthView(authViewCustomID)
	case "providerGroups":
		a.pushAuthView(authViewProviderGroupList)
	case "models":
		if len(a.auth.ModelOrder) > 0 {
			a.auth.CurrentModelID = a.auth.ModelOrder[0]
			a.pushAuthView(authViewModelGroupList)
		} else {
			a.pushAuthView(authViewAddModelID)
		}
	case "modelGroups":
		if a.auth.CurrentModelID != "" {
			a.pushAuthView(authViewModelGroupList)
		} else if len(a.auth.ModelOrder) > 0 {
			a.auth.CurrentModelID = a.auth.ModelOrder[0]
			a.pushAuthView(authViewModelGroupList)
		}
	case "settingsDetail":
		a.pushAuthView(authViewSettingsDetail)
	case "default":
		a.pushAuthView(authViewDefault)
	}
	if strings.HasPrefix(value, "model:") {
		modelID := strings.TrimPrefix(value, "model:")
		a.auth.CurrentModelID = modelID
		a.pushAuthView(authViewModelGroupList)
	}
}


func (a *App) submitAuthInput() {
	value := strings.TrimSpace(a.authInput.Value())
	a.auth.Error = ""
	switch a.auth.View {
	case authViewCustomID:
		if value == "" || strings.ContainsAny(value, " /\\\t\n") {
			a.auth.Error = "Provider ID must be non-empty and contain no spaces or slashes."
			return
		}
		a.auth.ProviderID = value
		if a.returnToReviewAfterEdit() {
			return
		}
		a.initAuthForCustom(value)
		a.pushAuthView(authViewProviderGroupList)
	case authViewProviderCredentials, authViewProviderProtocol,
		authViewProviderNetwork, authViewProviderAdvanced,
		authViewResponsesEdit:
		if err := a.authProviderSubmitInput(); err != nil {
			if err == errStayInInput {
				a.scheduleRender()
				return
			}
			a.auth.Error = err.Error()
			return
		}
		a.scheduleRender()
	case authViewHeadersEdit:
		if a.auth.ParamField == "headerKey" || a.auth.ParamField == "headerValue" {
			if err := a.authProviderSubmitInput(); err != nil {
				if err == errStayInInput {
					a.scheduleRender()
					return
				}
				a.auth.Error = err.Error()
				return
			}
		}
		a.scheduleRender()
	case authViewAddModelID:
		if value == "" || strings.ContainsAny(value, " /\\\t\n") {
			a.auth.Error = "Model ID must be non-empty and contain no spaces or slashes."
			return
		}
		a.auth.CurrentModelID = value
		a.pushAuthView(authViewAddModelName)
	case authViewAddModelName:
		name := value
		if name == "" {
			name = a.auth.CurrentModelID
		}
		me := a.initModelFromDefault(a.auth.CurrentModelID)
		me.Name = name
		if a.auth.Models == nil {
			a.auth.Models = map[string]*modelEditState{}
		}
		a.auth.Models[a.auth.CurrentModelID] = me
		a.auth.ModelOrder = append(a.auth.ModelOrder, a.auth.CurrentModelID)
		a.popAuthView() // pop AddModelName
		a.popAuthView() // pop AddModelID → back to model list or settings detail
	case authViewModelBasics, authViewModelCapabilities,
		authViewModelSampling, authViewModelCost, authViewModelCompat:
		if err := a.authModelSubmitInput(); err != nil {
			if err == errStayInInput {
				a.scheduleRender()
				return
			}
			a.auth.Error = err.Error()
			return
		}
		a.scheduleRender()
	}
	a.scheduleRender()
}

func (a *App) authOptions() []authOption {
	switch a.auth.View {
	case authViewMain:
		return []authOption{
			{Title: "Existing Providers", Description: "Add or update token/model under an existing provider", Value: "existing"},
			{Title: "Custom Provider", Description: "Add provider by API type, base URL, token and models", Value: "custom"},
		}
	case authViewExistingProvider:
		ids := sortedAuthProviderIDs(a.settings)
		ids = filterAuthProviderIDs(ids, a.auth.Search)
		opts := make([]authOption, 0, len(ids))
		for _, id := range ids {
			pc := a.settings.GetProviderConfig(id)
			desc := ""
			if pc != nil {
				desc = fmt.Sprintf("%s · %s · %d models", pc.API, pc.BaseURL, len(pc.Models))
			}
			opts = append(opts, authOption{Title: id, Description: desc, Value: id})
		}
		opts = append(opts, authOption{Title: "← Back", Description: "Return to main menu", Value: "back"})
		return opts
	case authViewProviderGroupList:
		return a.authProviderGroupOptions()
	case authViewProviderCredentials:
		return a.authProviderCredentialsOptions()
	case authViewProviderProtocol:
		return a.authProviderProtocolOptions()
	case authViewProviderNetwork:
		return a.authProviderNetworkOptions()
	case authViewProviderAdvanced:
		return a.authProviderAdvancedOptions()
	case authViewResponsesEdit:
		return a.authResponsesOptions()
	case authViewHeadersEdit:
		return a.authHeadersOptions()
	case authViewAPIChoice:
		return a.authViewAPIChoiceOptions()
	case authViewModelList:
		return a.authModelListOptions()
	case authViewModelGroupList:
		return a.authModelGroupOptions()
	case authViewModelBasics:
		return a.authModelBasicsOptions()
	case authViewModelCapabilities:
		return a.authModelCapabilitiesOptions()
	case authViewModelSampling:
		return a.authModelSamplingOptions()
	case authViewModelCost:
		return a.authModelCostOptions()
	case authViewModelCompat:
		return a.authModelCompatOptions()
	case authViewDefault:
		return []authOption{
			{Title: "Yes", Description: "Use this provider/model for future requests", Value: "yes"},
			{Title: "No", Description: "Save provider without changing defaults", Value: "no"},
		}
	case authViewReview:
		return []authOption{
			{Title: "Save", Description: "Write settings.json and switch current TUI provider", Value: "save"},
			{Title: "Edit", Description: "Go back and modify provider setup", Value: "edit"},
		}
	case authViewEditMenu:
		pe := &a.auth.Provider
		items := []authOption{
			{Title: "Provider Settings", Description: pe.summaryShort(), Value: "providerGroups"},
			{Title: "Models", Description: fmt.Sprintf("%d model(s)", len(a.auth.ModelOrder)), Value: "models"},
			{Title: "Default setting", Description: fmt.Sprintf("set default: %v", a.auth.SetDefault), Value: "default"},
		}
		for _, id := range a.auth.ModelOrder {
			if me, ok := a.auth.Models[id]; ok {
				items = append(items, authOption{
					Title:       "  └ " + id,
					Description: me.summaryShort(),
					Value:       "model:" + id,
				})
			}
		}
		if a.auth.Mode == "custom" {
			items = append([]authOption{{Title: "Provider ID", Description: a.auth.ProviderID, Value: "providerID"}}, items...)
		}
		return items
	case authViewSettingsDetail:
		return a.authSettingsDetailOptions()
	default:
		return nil
	}
}

func defaultBaseURLForAPI(api string) string {
	switch api {
	case "anthropic-messages":
		return "https://api.anthropic.com"
	case "google-gemini":
		return "https://generativelanguage.googleapis.com/v1beta/models"
	case "google-vertex":
		return "https://aiplatform.googleapis.com/v1/publishers/google/models"
	default:
		return "https://api.openai.com/v1"
	}
}

func baseURLOptionsForProvider(providerID string) []authOption {
	switch providerID {
	case "longcat":
		return []authOption{
			{Title: "OpenAI Compatible", Description: "https://api.longcat.chat/openai", Value: "https://api.longcat.chat/openai"},
			{Title: "Anthropic Compatible", Description: "https://api.longcat.chat/anthropic", Value: "https://api.longcat.chat/anthropic"},
		}
	case "minimax":
		return []authOption{
			{Title: "International", Description: "https://api.minimax.io/v1", Value: "https://api.minimax.io/v1"},
			{Title: "China", Description: "https://api.minimaxi.com/v1", Value: "https://api.minimaxi.com/v1"},
		}
	case "zai":
		return []authOption{
			{Title: "Standard API Key", Description: "https://api.z.ai/api/paas/v4", Value: "https://api.z.ai/api/paas/v4"},
			{Title: "Coding Plan", Description: "https://api.z.ai/api/coding/paas/v4", Value: "https://api.z.ai/api/coding/paas/v4"},
		}
	case "alibaba-coding-plan":
		return []authOption{
			{Title: "China (Beijing)", Description: "https://coding.dashscope.aliyuncs.com/v1", Value: "https://coding.dashscope.aliyuncs.com/v1"},
			{Title: "Singapore (International)", Description: "https://coding-intl.dashscope.aliyuncs.com/v1", Value: "https://coding-intl.dashscope.aliyuncs.com/v1"},
		}
	case "alibaba-standard":
		return []authOption{
			{Title: "China (Beijing)", Description: "https://dashscope.aliyuncs.com/compatible-mode/v1", Value: "https://dashscope.aliyuncs.com/compatible-mode/v1"},
			{Title: "Singapore", Description: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", Value: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"},
			{Title: "US (Virginia)", Description: "https://dashscope-us.aliyuncs.com/compatible-mode/v1", Value: "https://dashscope-us.aliyuncs.com/compatible-mode/v1"},
			{Title: "China (Hong Kong)", Description: "https://cn-hongkong.dashscope.aliyuncs.com/compatible-mode/v1", Value: "https://cn-hongkong.dashscope.aliyuncs.com/compatible-mode/v1"},
		}
	default:
		return nil
	}
}

func normalizeAuthModelIDs(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '\n' || r == '\t' })
	seen := map[string]bool{}
	var ids []string
	for _, f := range fields {
		id := strings.TrimSpace(f)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func parsePositiveInt(s string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("invalid positive integer")
	}
	return v, nil
}

func parseFloatRange(s string, min, max float64) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < min || v > max {
		return 0, fmt.Errorf("invalid float range")
	}
	return v, nil
}

func (a *App) buildAuthSettingsFrom(base *config.Settings) (*config.Settings, string) {
	next := *base
	next.Providers = make(map[string]*config.ProviderConfig, len(base.Providers)+1)
	for k, v := range base.Providers {
		if v == nil {
			continue
		}
		cp := cloneProviderConfig(v)
		next.Providers[k] = &cp
	}
	pc := next.Providers[a.auth.ProviderID]
	if pc == nil {
		pc = &config.ProviderConfig{}
		next.Providers[a.auth.ProviderID] = pc
	}

	// Write provider-level fields from structured state
	pc.API = a.auth.Provider.API
	if pc.API == "" {
		pc.API = "openai-chat"
	}
	pc.BaseURL = a.auth.Provider.BaseURL
	pc.HTTPProxy = a.auth.Provider.HTTPProxy
	pc.ForceHTTP11 = a.auth.Provider.ForceHTTP11
	pc.APIKey = a.auth.Provider.APIKey
	pc.Vendor = a.auth.Provider.Vendor
	pc.Headers = config.CloneStringMap(a.auth.Provider.Headers)
	pc.ThinkingFormat = a.auth.Provider.ThinkingFormat
	pc.CacheControl = config.CloneBoolPtr(a.auth.Provider.CacheControl)
	pc.Responses = a.auth.Provider.Responses.toConfig()

	// Write models from structured per-model state
	pc.Models = make([]config.ModelConfig, 0, len(a.auth.ModelOrder))
	for _, id := range a.auth.ModelOrder {
		if me, ok := a.auth.Models[id]; ok {
			pc.Models = append(pc.Models, me.toConfig())
		}
	}

	modelID := ""
	if len(a.auth.ModelOrder) > 0 {
		modelID = a.auth.ModelOrder[0]
	}
	if a.auth.SetDefault {
		next.DefaultProvider = a.auth.ProviderID
		next.DefaultModel = modelID
	}
	return &next, modelID
}

func (a *App) buildAuthSettings() (*config.Settings, string) {
	return a.buildAuthSettingsFrom(a.settings)
}

func cloneProviderConfig(v *config.ProviderConfig) config.ProviderConfig {
	cp := *v
	cp.Models = append([]config.ModelConfig(nil), v.Models...)
	if v.Headers != nil {
		cp.Headers = make(map[string]string, len(v.Headers))
		for hk, hv := range v.Headers {
			cp.Headers[hk] = hv
		}
	}
	return cp
}

func (a *App) prepareAuthPreview() {
	next, _ := a.buildAuthSettings()
	a.auth.Preview = previewBuildFoldedJSON(next, a.auth.ProviderID, true)
}

func (a *App) togglePreviewFold(section string) {
	switch section {
	case "cost":
		a.auth.PreviewExpand.CostExpand = !a.auth.PreviewExpand.CostExpand
	case "compat":
		a.auth.PreviewExpand.CompatExpand = !a.auth.PreviewExpand.CompatExpand
	}
}

func maskAuthSecret(s string) string {
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		return s
	}
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

func (a *App) saveAuthProvider() {
	globalSparse, err := config.LoadGlobalSettingsSparse()
	if err != nil {
		a.auth.Error = fmt.Sprintf("Load global settings failed: %v", err)
		return
	}
	nextGlobal, modelID := a.buildAuthSettingsFrom(globalSparse)
	runtimeSettings := *a.settings
	runtimePatched, _ := a.buildAuthSettingsFrom(&runtimeSettings)
	p, m, err := providerfactory.Create(runtimePatched, a.auth.ProviderID, modelID)
	if err != nil {
		a.auth.Error = fmt.Sprintf("Provider validation failed: %v", err)
		return
	}
	if err := config.SaveGlobalSettings(nextGlobal); err != nil {
		a.auth.Error = fmt.Sprintf("Save failed: %v", err)
		return
	}
	a.settings = runtimePatched
	a.provider = p
	a.model = m
	a.resetAgent(fmt.Errorf("provider changed"))
	providerID := a.auth.ProviderID
	model := m.ID
	a.closeAuthDialog()
	a.addCommandStatus(fmt.Sprintf("✅ Provider saved: %s / %s", providerID, model), "Next message will use the new provider/model.")
}

