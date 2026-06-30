package tui

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
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
	authViewAPI
	authViewBaseURLChoice
	authViewBaseURL
	authViewHTTPProxy
	authViewForceHTTP11
	authViewAPIKey
	authViewModels
	authViewAdvanced
	authViewParamInput
	authViewDefault
	authViewReview
	authViewEditMenu
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

	ProviderID  string
	API         string
	BaseURL     string
	HTTPProxy   string
	ForceHTTP11 bool
	APIKey      string
	ModelIDs    string
	Search      string
	SetDefault  bool

	ContextWindow string
	MaxTokens     string
	Reasoning     bool
	InputTypes    string
	Temperature   string
	TopP          string
	ParamField    string

	// Track which advanced params were explicitly edited by the user.
	// Pre-filled defaults from settings.go provider config have these as false,
	// so applyAuthModelParams preserves per-model values for unedited params.
	ContextWindowEdited bool
	MaxTokensEdited     bool
	ReasoningEdited     bool
	InputTypesEdited    bool
	TemperatureEdited   bool
	TopPEdited          bool

	Error   string
	Preview string
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
	placeholder := ""
	value := ""
	switch a.auth.View {
	case authViewCustomID:
		placeholder = "provider-id (e.g. openrouter)"
		value = a.auth.ProviderID
	case authViewBaseURL:
		placeholder = "https://api.openai.com/v1"
		value = a.auth.BaseURL
	case authViewHTTPProxy:
		placeholder = "http://127.0.0.1:7890 (empty = none)"
		value = a.auth.HTTPProxy
	case authViewAPIKey:
		placeholder = "sk-... or ${ENV_VAR}"
		value = a.auth.APIKey
	case authViewModels:
		placeholder = "model-id-1, model-id-2"
		value = a.auth.ModelIDs
	case authViewParamInput:
		placeholder, value = a.authParamPlaceholderAndValue()
	}
	a.authInput = editor.New(max(20, a.width-8)).SetPlaceholder(placeholder).SetMaxLines(3).Focus()
	if value != "" {
		a.authInput = a.authInput.SetValue(value)
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
	case tea.KeyEnter, tea.KeySpace:
		a.selectAuthOption()
		return true, nil
	}
	return true, nil
}

func (a *App) authInputActive() bool {
	switch a.auth.View {
	case authViewCustomID, authViewBaseURL, authViewHTTPProxy, authViewAPIKey, authViewModels, authViewParamInput:
		return true
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
		a.auth.ProviderID = opt.Value
		pc := a.settings.GetProviderConfig(opt.Value)
		if pc != nil {
			a.auth.API = pc.API
			a.auth.BaseURL = pc.BaseURL
			a.auth.HTTPProxy = pc.HTTPProxy
			a.auth.ForceHTTP11 = pc.ForceHTTP11
			a.auth.APIKey = pc.APIKey
			var ids []string
			for _, m := range pc.Models {
				ids = append(ids, m.ID)
			}
			a.auth.ModelIDs = strings.Join(ids, ", ")
			// Pre-fill advanced model params from the first existing model
			// so the review screen shows the settings.go preset values.
			// Edited flags stay false so saving preserves per-model values
			// for params the user didn't explicitly change.
			if len(pc.Models) > 0 {
				a.auth.ContextWindow = ""
				a.auth.MaxTokens = ""
				a.auth.Reasoning = false
				a.auth.InputTypes = ""
				a.auth.Temperature = ""
				a.auth.TopP = ""
				a.auth.ContextWindowEdited = false
				a.auth.MaxTokensEdited = false
				a.auth.ReasoningEdited = false
				a.auth.InputTypesEdited = false
				a.auth.TemperatureEdited = false
				a.auth.TopPEdited = false
				m := pc.Models[0]
				if m.ContextWindow > 0 {
					a.auth.ContextWindow = strconv.Itoa(m.ContextWindow)
				}
				if m.MaxTokens > 0 {
					a.auth.MaxTokens = strconv.Itoa(m.MaxTokens)
				}
				a.auth.Reasoning = m.Reasoning
				if len(m.Input) > 0 {
					a.auth.InputTypes = strings.Join(m.Input, ",")
				}
				if m.Temperature != nil {
					a.auth.Temperature = strconv.FormatFloat(*m.Temperature, 'f', -1, 64)
				}
				if m.TopP != nil {
					a.auth.TopP = strconv.FormatFloat(*m.TopP, 'f', -1, 64)
				}
			}
		}
		if a.auth.API == "" {
			a.auth.API = "openai-chat"
		}
		a.pushAuthView(authViewAPIKey)
	case authViewAPI:
		a.auth.API = opt.Value
		if !a.isReviewEdit() {
			a.auth.BaseURL = defaultBaseURLForAPI(opt.Value)
			a.pushAuthView(authViewBaseURL)
		} else {
			a.returnToReviewAfterEdit()
		}
	case authViewBaseURLChoice:
		if opt.Value != "custom" {
			a.auth.BaseURL = opt.Value
		}
		a.pushAuthView(authViewBaseURL)
	case authViewForceHTTP11:
		a.auth.ForceHTTP11 = opt.Value == "yes"
		if a.returnToReviewAfterEdit() {
			break
		}
		if a.auth.Mode == "existing" {
			a.pushAuthView(authViewModels)
		} else {
			a.pushAuthView(authViewAPIKey)
		}
	case authViewAdvanced:
		switch opt.Value {
		case "continue":
			if a.isReviewEdit() {
				a.returnToReviewAfterEdit()
			} else {
				a.pushAuthView(authViewDefault)
			}
		case "reasoning":
			a.auth.Reasoning = !a.auth.Reasoning
			a.auth.ReasoningEdited = true
		case "skip":
			a.auth.ContextWindow = ""
			a.auth.MaxTokens = ""
			a.auth.Reasoning = false
			a.auth.InputTypes = ""
			a.auth.Temperature = ""
			a.auth.TopP = ""
			a.auth.ContextWindowEdited = false
			a.auth.MaxTokensEdited = false
			a.auth.ReasoningEdited = false
			a.auth.InputTypesEdited = false
			a.auth.TemperatureEdited = false
			a.auth.TopPEdited = false
			if a.isReviewEdit() {
				a.returnToReviewAfterEdit()
			} else {
				a.pushAuthView(authViewDefault)
			}
		default:
			a.auth.ParamField = opt.Value
			a.pushAuthView(authViewParamInput)
		}
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
	case "api":
		a.pushAuthView(authViewAPI)
	case "apiKey":
		a.pushAuthView(authViewAPIKey)
	case "baseUrl":
		if len(baseURLOptionsForProvider(a.auth.ProviderID)) > 0 {
			a.pushAuthView(authViewBaseURLChoice)
		} else {
			a.pushAuthView(authViewBaseURL)
		}
	case "httpProxy":
		a.pushAuthView(authViewHTTPProxy)
	case "forceHTTP11":
		a.auth.ForceHTTP11 = !a.auth.ForceHTTP11
		if a.returnToReviewAfterEdit() {
			return
		}
	case "models":
		a.pushAuthView(authViewModels)
	case "advanced":
		a.pushAuthView(authViewAdvanced)
	case "default":
		a.pushAuthView(authViewDefault)
	}
}

func (a *App) returnToAdvancedAfterParamEdit() {
	for i := len(a.auth.Stack) - 1; i >= 0; i-- {
		if a.auth.Stack[i] == authViewAdvanced {
			a.auth.Stack = a.auth.Stack[:i]
			a.auth.View = authViewAdvanced
			a.auth.Cursor = 0
			a.auth.Error = ""
			a.prepareAuthInput()
			a.scheduleRender()
			return
		}
	}
	a.pushAuthView(authViewAdvanced)
}

func (a *App) authParamPlaceholderAndValue() (string, string) {
	switch a.auth.ParamField {
	case "contextWindow":
		return "128000", a.auth.ContextWindow
	case "maxTokens":
		return "8192", a.auth.MaxTokens
	case "input":
		return "text,image", valueOrDefaultText(a.auth.InputTypes, "text")
	case "temperature":
		return "0.7 (empty = auto)", a.auth.Temperature
	case "topP":
		return "1.0 (empty = auto)", a.auth.TopP
	default:
		return "value", ""
	}
}

func authParamPrompt(field string) string {
	switch field {
	case "contextWindow":
		return "Enter context window tokens (empty = auto/existing):"
	case "maxTokens":
		return "Enter max output tokens (empty = auto/existing):"
	case "input":
		return "Enter input modalities separated by commas (text,image,audio,video,pdf):"
	case "temperature":
		return "Enter temperature 0..2 (empty = API default):"
	case "topP":
		return "Enter top_p 0..1 (empty = API default):"
	default:
		return "Enter value:"
	}
}

func (a *App) setAuthParamValue(value string) error {
	switch a.auth.ParamField {
	case "contextWindow":
		if value != "" {
			if _, err := parsePositiveInt(value); err != nil {
				return fmt.Errorf("context window must be a positive integer")
			}
		}
		a.auth.ContextWindow = value
		a.auth.ContextWindowEdited = true
	case "maxTokens":
		if value != "" {
			if _, err := parsePositiveInt(value); err != nil {
				return fmt.Errorf("max tokens must be a positive integer")
			}
		}
		a.auth.MaxTokens = value
		a.auth.MaxTokensEdited = true
	case "input":
		ids := normalizeAuthModelIDs(value)
		if len(ids) == 0 {
			ids = []string{"text"}
		}
		a.auth.InputTypes = strings.Join(ids, ",")
		a.auth.InputTypesEdited = true
	case "temperature":
		if value != "" {
			if _, err := parseFloatRange(value, 0, 2); err != nil {
				return fmt.Errorf("temperature must be between 0 and 2")
			}
		}
		a.auth.Temperature = value
		a.auth.TemperatureEdited = true
	case "topP":
		if value != "" {
			if _, err := parseFloatRange(value, 0, 1); err != nil {
				return fmt.Errorf("top_p must be between 0 and 1")
			}
		}
		a.auth.TopP = value
		a.auth.TopPEdited = true
	}
	return nil
}

func (a *App) authAdvancedSummary() string {
	parts := []string{}
	if a.auth.ContextWindow != "" {
		parts = append(parts, "ctx="+a.auth.ContextWindow)
	}
	if a.auth.MaxTokens != "" {
		parts = append(parts, "max="+a.auth.MaxTokens)
	}
	if a.auth.Reasoning {
		parts = append(parts, "reasoning")
	}
	if a.auth.InputTypes != "" {
		parts = append(parts, "input="+a.auth.InputTypes)
	}
	if a.auth.Temperature != "" {
		parts = append(parts, "temp="+a.auth.Temperature)
	}
	if a.auth.TopP != "" {
		parts = append(parts, "top_p="+a.auth.TopP)
	}
	if len(parts) == 0 {
		return "auto/existing"
	}
	return strings.Join(parts, ", ")
}

func valueOrAuto(s string) string {
	return valueOrDefaultText(s, "auto")
}

func valueOrDefaultText(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
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
		a.pushAuthView(authViewAPI)
	case authViewBaseURL:
		if value == "" {
			a.auth.Error = "Base URL is required."
			return
		}
		if u, err := url.Parse(value); err != nil || u.Scheme == "" || u.Host == "" {
			a.auth.Error = "Base URL must be a valid absolute URL."
			return
		}
		a.auth.BaseURL = value
		if a.returnToReviewAfterEdit() {
			return
		}
		a.pushAuthView(authViewHTTPProxy)
	case authViewHTTPProxy:
		if value != "" {
			if u, err := url.Parse(value); err != nil || u.Scheme == "" || u.Host == "" {
				a.auth.Error = "HTTP proxy must be a valid URL or empty."
				return
			}
		}
		a.auth.HTTPProxy = value
		if a.returnToReviewAfterEdit() {
			return
		}
		a.pushAuthView(authViewForceHTTP11)
	case authViewAPIKey:
		if value == "" {
			a.auth.Error = "API key is required."
			return
		}
		a.auth.APIKey = value
		if a.returnToReviewAfterEdit() {
			return
		}
		if a.auth.Mode == "custom" {
			a.pushAuthView(authViewModels)
		} else {
			a.pushAuthView(authViewBaseURL)
		}
	case authViewModels:
		ids := normalizeAuthModelIDs(value)
		if len(ids) == 0 {
			a.auth.Error = "At least one model ID is required."
			return
		}
		a.auth.ModelIDs = strings.Join(ids, ", ")
		if a.returnToReviewAfterEdit() {
			return
		}
		a.pushAuthView(authViewAdvanced)
	case authViewParamInput:
		if err := a.setAuthParamValue(value); err != nil {
			a.auth.Error = err.Error()
			return
		}
		a.returnToAdvancedAfterParamEdit()
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
		return opts
	case authViewAPI:
		return []authOption{
			{Title: "OpenAI-compatible", Description: "api: openai-chat", Value: "openai-chat"},
			{Title: "OpenAI Responses", Description: "api: openai-responses", Value: "openai-responses"},
			{Title: "Anthropic-compatible", Description: "api: anthropic-messages", Value: "anthropic-messages"},
			{Title: "Gemini-compatible", Description: "api: google-gemini", Value: "google-gemini"},
			{Title: "Google Vertex", Description: "api: google-vertex", Value: "google-vertex"},
		}
	case authViewBaseURLChoice:
		opts := baseURLOptionsForProvider(a.auth.ProviderID)
		items := make([]authOption, 0, len(opts)+1)
		for _, opt := range opts {
			items = append(items, opt)
		}
		items = append(items, authOption{Title: "Custom", Description: "Enter or edit the base URL manually", Value: "custom"})
		return items
	case authViewForceHTTP11:
		return []authOption{
			{Title: "No", Description: "Use the default HTTP transport behavior", Value: "no"},
			{Title: "Yes", Description: "Disable HTTP/2 for this provider", Value: "yes"},
		}
	case authViewAdvanced:
		return []authOption{
			{Title: "Continue", Description: "Apply these parameters", Value: "continue"},
			{Title: "Skip/Clear", Description: "Use existing model params or defaults", Value: "skip"},
			{Title: "Context window", Description: valueOrAuto(a.auth.ContextWindow), Value: "contextWindow"},
			{Title: "Max output tokens", Description: valueOrAuto(a.auth.MaxTokens), Value: "maxTokens"},
			{Title: "Reasoning", Description: fmt.Sprintf("enabled: %v", a.auth.Reasoning), Value: "reasoning"},
			{Title: "Input modalities", Description: valueOrDefaultText(a.auth.InputTypes, "text"), Value: "input"},
			{Title: "Temperature", Description: valueOrAuto(a.auth.Temperature), Value: "temperature"},
			{Title: "Top P", Description: valueOrAuto(a.auth.TopP), Value: "topP"},
		}
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
		items := []authOption{
			{Title: "API Key", Description: maskAuthSecret(a.auth.APIKey), Value: "apiKey"},
			{Title: "Base URL", Description: a.auth.BaseURL, Value: "baseUrl"},
			{Title: "HTTP Proxy", Description: valueOrDefaultText(a.auth.HTTPProxy, "none"), Value: "httpProxy"},
			{Title: "Force HTTP/1.1", Description: fmt.Sprintf("enabled: %v", a.auth.ForceHTTP11), Value: "forceHTTP11"},
			{Title: "Model IDs", Description: a.auth.ModelIDs, Value: "models"},
			{Title: "Advanced parameters", Description: a.authAdvancedSummary(), Value: "advanced"},
			{Title: "Default setting", Description: fmt.Sprintf("set default: %v", a.auth.SetDefault), Value: "default"},
		}
		if a.auth.Mode == "custom" {
			items = append([]authOption{{Title: "Provider ID", Description: a.auth.ProviderID, Value: "providerID"}, {Title: "API Type", Description: a.auth.API, Value: "api"}}, items...)
		}
		return items
	default:
		return nil
	}
}

func (a *App) renderAuthDialog() string {
	if !a.auth.Open {
		return ""
	}
	width := a.width - 4
	if width < 50 {
		width = 50
	}
	if width > 100 {
		width = 100
	}
	var lines []string
	lines = append(lines, authTitle(a.auth.View))
	lines = append(lines, "")
	if a.authInputActive() {
		lines = append(lines, a.authInputPrompt(a.auth.View))
		lines = append(lines, a.authInput.View())
		lines = append(lines, "")
		lines = append(lines, statusStyle.Render("Enter to submit, Esc to go back"))
	} else if a.auth.View == authViewReview {
		lines = append(lines, renderAuthPreview(a.auth.Preview)...)
		lines = append(lines, "")
		lines = append(lines, a.renderAuthOptions())
		lines = append(lines, statusStyle.Render("Enter to save, Esc to go back"))
	} else {
		if a.auth.View == authViewExistingProvider {
			query := a.auth.Search
			if query == "" {
				query = "type to search"
			}
			lines = append(lines, statusStyle.Render("Search: "+query), "")
		}
		lines = append(lines, a.renderAuthOptions())
		lines = append(lines, "")
		lines = append(lines, statusStyle.Render("Enter to select, ↑↓ to navigate, Esc to go back"))
	}
	if a.auth.Error != "" {
		lines = append(lines, "", errorStyle.Render(a.auth.Error))
	}
	return authDialogStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func sortedAuthProviderIDs(settings *config.Settings) []string {
	if settings == nil {
		return nil
	}
	ids := make([]string, 0, len(settings.Providers))
	for id := range settings.Providers {
		ids = append(ids, id)
	}
	sort.SliceStable(ids, func(i, j int) bool {
		pi, pj := authProviderSortPriority(ids[i]), authProviderSortPriority(ids[j])
		if pi != pj {
			return pi < pj
		}
		return ids[i] < ids[j]
	})
	return ids
}

func filterAuthProviderIDs(ids []string, query string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return ids
	}
	type scored struct {
		id    string
		score int
	}
	var matches []scored
	for _, id := range ids {
		lower := strings.ToLower(id)
		score := -1
		switch {
		case lower == query:
			score = 0
		case strings.HasPrefix(lower, query):
			score = 1
		case strings.Contains(lower, query):
			score = 2
		}
		if score >= 0 {
			matches = append(matches, scored{id: id, score: score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score < matches[j].score
		}
		pi, pj := authProviderSortPriority(matches[i].id), authProviderSortPriority(matches[j].id)
		if pi != pj {
			return pi < pj
		}
		return matches[i].id < matches[j].id
	})
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m.id
	}
	return out
}

func authProviderSortPriority(id string) int {
	name := strings.ToLower(id)
	switch {
	case name == "moark" || strings.Contains(name, "moark"):
		return 10
	case strings.Contains(name, "deepseek"):
		return 20
	case strings.Contains(name, "xiaomi") || strings.Contains(name, "mimo"):
		return 30
	case strings.Contains(name, "doubao") || strings.Contains(name, "volc") || strings.Contains(name, "ark"):
		return 40
	case name == "openai" || strings.Contains(name, "openai"):
		return 50
	case strings.Contains(name, "anthropic") || strings.Contains(name, "claude"):
		return 60
	case strings.Contains(name, "google") || strings.Contains(name, "gemini") || strings.Contains(name, "vertex"):
		return 70
	default:
		return 100
	}
}

func renderAuthPreview(preview string) []string {
	preview = strings.TrimRight(preview, "\n")
	if preview == "" {
		return nil
	}
	lines := strings.Split(preview, "\n")
	if len(lines) <= authMaxPreviewVisibleLines {
		return lines
	}
	visible := append([]string(nil), lines[:authMaxPreviewVisibleLines]...)
	visible = append(visible, statusStyle.Render(fmt.Sprintf("… %d more lines hidden", len(lines)-authMaxPreviewVisibleLines)))
	return visible
}

func (a *App) renderAuthOptions() string {
	opts := a.authOptions()
	if len(opts) == 0 {
		if a.auth.View == authViewExistingProvider && a.auth.Search != "" {
			return statusStyle.Render("No providers match.")
		}
		return ""
	}
	start, end := authVisibleRange(a.auth.Cursor, len(opts), authMaxVisibleOptions)
	visible := opts[start:end]
	var lines []string
	for i, opt := range visible {
		actual := start + i
		cursor := "  "
		style := lipgloss.NewStyle()
		if actual == a.auth.Cursor {
			cursor = "› "
			style = style.Foreground(lipgloss.Color("86")).Bold(true)
		}
		scroll := authScrollMarker(actual, len(opts), start, end)
		lines = append(lines, style.Render(cursor+opt.Title)+scroll)
		if opt.Description != "" {
			lines = append(lines, statusStyle.Render("  "+opt.Description))
		}
		if i != len(visible)-1 {
			lines = append(lines, "")
		}
	}
	if len(opts) > authMaxVisibleOptions {
		lines = append(lines, "", statusStyle.Render(fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(opts))))
	}
	return strings.Join(lines, "\n")
}

func authVisibleRange(cursor, total, limit int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if limit <= 0 || total <= limit {
		return 0, total
	}
	start := cursor - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > total {
		start = total - limit
	}
	return start, start + limit
}

func authScrollMarker(actual, total, start, end int) string {
	if total <= authMaxVisibleOptions {
		return ""
	}
	switch {
	case actual == start && start > 0:
		return statusStyle.Render("  ↑")
	case actual == end-1 && end < total:
		return statusStyle.Render("  ↓")
	case actual == start && start == 0 && end < total:
		return statusStyle.Render("  │")
	case actual == end-1 && start > 0 && end == total:
		return statusStyle.Render("  │")
	default:
		return statusStyle.Render("  │")
	}
}

func authTitle(v authView) string {
	switch v {
	case authViewMain:
		return "Connect a Provider"
	case authViewExistingProvider:
		return "Existing Providers · Provider"
	case authViewCustomID:
		return "Custom Provider · Provider ID"
	case authViewAPI:
		return "Custom Provider · API Type"
	case authViewBaseURLChoice:
		return "Provider Setup · Endpoint"
	case authViewBaseURL:
		return "Provider Setup · Base URL"
	case authViewHTTPProxy:
		return "Provider Setup · HTTP Proxy"
	case authViewForceHTTP11:
		return "Provider Setup · Force HTTP/1.1"
	case authViewAPIKey:
		return "Provider Setup · API Key"
	case authViewModels:
		return "Provider Setup · Model IDs"
	case authViewAdvanced:
		return "Provider Setup · Advanced Parameters"
	case authViewParamInput:
		return "Provider Setup · Parameter"
	case authViewDefault:
		return "Provider Setup · Default"
	case authViewReview:
		return "Provider Setup · Review"
	case authViewEditMenu:
		return "Provider Setup · Edit"
	default:
		return "Provider Setup"
	}
}

func (a *App) authInputPrompt(v authView) string {
	switch v {
	case authViewCustomID:
		return "Enter provider ID:"
	case authViewBaseURL:
		return "Enter API endpoint for this provider:"
	case authViewHTTPProxy:
		return "Enter optional HTTP proxy URL for this provider:"
	case authViewAPIKey:
		return "Enter API key (plain token or ${ENV_VAR} reference):"
	case authViewModels:
		return "Enter model IDs separated by commas:"
	case authViewParamInput:
		return authParamPrompt(a.auth.ParamField)
	default:
		return "Input:"
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
	oldModels := map[string]config.ModelConfig{}
	for _, m := range pc.Models {
		oldModels[m.ID] = m
	}
	pc.API = a.auth.API
	if pc.API == "" {
		pc.API = "openai-chat"
	}
	pc.BaseURL = a.auth.BaseURL
	pc.HTTPProxy = a.auth.HTTPProxy
	pc.ForceHTTP11 = a.auth.ForceHTTP11
	pc.APIKey = a.auth.APIKey
	ids := normalizeAuthModelIDs(a.auth.ModelIDs)
	pc.Models = make([]config.ModelConfig, 0, len(ids))
	for _, id := range ids {
		model := config.ModelConfig{ID: id, Name: id, ContextWindow: 128000, MaxTokens: 8192, Input: []string{"text"}}
		if existing, ok := oldModels[id]; ok {
			model = existing
		}
		applyAuthModelParams(&model, &a.auth)
		pc.Models = append(pc.Models, model)
	}
	modelID := ""
	if len(ids) > 0 {
		modelID = ids[0]
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

// applyAuthModelParams only applies params that were explicitly edited by the user.
// Pre-filled values from settings.go provider defaults are displayed in the UI but
// not applied to models unless the user changed them, preserving per-model config.
func applyAuthModelParams(model *config.ModelConfig, auth *authDialogState) {
	if model == nil || auth == nil {
		return
	}
	if auth.ContextWindowEdited && auth.ContextWindow != "" {
		if v, err := parsePositiveInt(auth.ContextWindow); err == nil {
			model.ContextWindow = v
		}
	}
	if auth.MaxTokensEdited && auth.MaxTokens != "" {
		if v, err := parsePositiveInt(auth.MaxTokens); err == nil {
			model.MaxTokens = v
		}
	}
	if auth.ReasoningEdited {
		model.Reasoning = auth.Reasoning
	}
	if auth.InputTypesEdited && auth.InputTypes != "" {
		model.Input = normalizeAuthModelIDs(auth.InputTypes)
	}
	if auth.TemperatureEdited && auth.Temperature != "" {
		if v, err := parseFloatRange(auth.Temperature, 0, 2); err == nil {
			model.Temperature = &v
		}
	}
	if auth.TopPEdited && auth.TopP != "" {
		if v, err := parseFloatRange(auth.TopP, 0, 1); err == nil {
			model.TopP = &v
		}
	}
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
	preview := struct {
		DefaultProvider string                            `json:"defaultProvider,omitempty"`
		DefaultModel    string                            `json:"defaultModel,omitempty"`
		Providers       map[string]*config.ProviderConfig `json:"providers"`
	}{DefaultProvider: next.DefaultProvider, DefaultModel: next.DefaultModel, Providers: map[string]*config.ProviderConfig{}}
	pc := *next.Providers[a.auth.ProviderID]
	pc.APIKey = maskAuthSecret(pc.APIKey)
	preview.Providers[a.auth.ProviderID] = &pc
	data, _ := json.MarshalIndent(preview, "", "  ")
	a.auth.Preview = string(data)
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
