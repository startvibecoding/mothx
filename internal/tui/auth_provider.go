package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/startvibecoding/vibecoding/internal/tui/components/editor"
)

// --- Provider group constants ---

const (
	providerGroupCredentials = "credentials"
	providerGroupProtocol    = "protocol"
	providerGroupNetwork     = "network"
	providerGroupAdvanced    = "advanced"
)

// providerGroupInfo describes a provider settings group.
type providerGroupInfo struct {
	ID          string
	Title       string
	Description string
}

var providerGroups = []providerGroupInfo{
	{providerGroupCredentials, "A. Credentials", "API Key, Vendor"},
	{providerGroupProtocol, "B. Protocol", "Base URL, Responses"},
	{providerGroupNetwork, "C. Network", "HTTP Proxy, Force HTTP/1.1"},
	{providerGroupAdvanced, "D. Advanced", "Headers, Thinking Format, Cache Control"},
}

// --- authViewProviderGroupList ---

func (a *App) authProviderGroupOptions() []authOption {
	pe := &a.auth.Provider
	opts := make([]authOption, 0, len(providerGroups)+2)
	// API Type at top level (select, not text input)
	apiDesc := pe.API
	if apiDesc == "" {
		apiDesc = "(not set)"
	}
	opts = append(opts, authOption{
		Title:       "API Type",
		Description: apiDesc,
		Value:       "api",
	})
	for _, g := range providerGroups {
		desc := ""
		switch g.ID {
		case providerGroupCredentials:
			desc = a.authProviderCredentialsSummary(pe)
		case providerGroupProtocol:
			desc = a.authProviderProtocolSummary(pe)
		case providerGroupNetwork:
			desc = a.authProviderNetworkSummary(pe)
		case providerGroupAdvanced:
			desc = a.authProviderAdvancedSummary(pe)
		}
		opts = append(opts, authOption{
			Title:       g.Title,
			Description: desc,
			Value:       g.ID,
		})
	}
	// Add Models entry at the end
	modelCount := len(a.auth.ModelOrder)
	opts = append(opts, authOption{
		Title:       "Models ▶",
		Description: fmt.Sprintf("(%d model(s) configured)", modelCount),
		Value:       "models",
	})
	opts = append(opts, authOption{Title: "✓ Done", Description: "Save and exit", Value: "done"})
	return opts
}

func (a *App) authProviderCredentialsSummary(pe *providerEditState) string {
	parts := []string{}
	if pe.APIKey != "" {
		parts = append(parts, "key="+maskAuthSecret(pe.APIKey))
	} else {
		parts = append(parts, "key=(empty)")
	}
	if pe.Vendor != "" {
		parts = append(parts, "vendor="+pe.Vendor)
	}
	return strings.Join(parts, "  ")
}

func (a *App) authProviderProtocolSummary(pe *providerEditState) string {
	parts := []string{}
	parts = append(parts, "url="+shortURL(pe.BaseURL))
	if pe.Responses.ReasoningSummary != "" {
		parts = append(parts, "rs="+pe.Responses.ReasoningSummary)
	}
	return strings.Join(parts, "  ")
}

func (a *App) authProviderNetworkSummary(pe *providerEditState) string {
	parts := []string{}
	if pe.HTTPProxy != "" {
		parts = append(parts, "proxy="+shortURL(pe.HTTPProxy))
	} else {
		parts = append(parts, "proxy=none")
	}
	if pe.ForceHTTP11 {
		parts = append(parts, "force-h1")
	} else {
		parts = append(parts, "http/2")
	}
	return strings.Join(parts, "  ")
}

func (a *App) authProviderAdvancedSummary(pe *providerEditState) string {
	parts := []string{}
	if len(pe.Headers) > 0 {
		parts = append(parts, "hdrs="+strconv.Itoa(len(pe.Headers)))
	} else {
		parts = append(parts, "hdrs=none")
	}
	if pe.ThinkingFormat != "" {
		parts = append(parts, "think="+pe.ThinkingFormat)
	}
	if pe.CacheControl == nil {
		parts = append(parts, "cache=auto")
	} else if *pe.CacheControl {
		parts = append(parts, "cache=on")
	} else {
		parts = append(parts, "cache=off")
	}
	return strings.Join(parts, "  ")
}

// --- Provider sub-form options ---

func (a *App) authProviderCredentialsOptions() []authOption {
	pe := &a.auth.Provider
	opts := []authOption{
		{Title: "API Key", Description: maskAuthSecret(pe.APIKey), Value: "apiKey"},
		{Title: "Vendor", Description: valueOrDefault(pe.Vendor, "(auto-detect)"), Value: "vendor"},
	}
	opts = append(opts, authOption{Title: "Done", Description: "Confirm credentials", Value: "done"})
	return opts
}

func (a *App) authProviderProtocolOptions() []authOption {
	pe := &a.auth.Provider
	opts := []authOption{
		{Title: "Base URL", Description: pe.BaseURL, Value: "baseUrl"},
		{Title: "Responses ▶", Description: a.authProviderResponsesSummary(&pe.Responses), Value: "responses"},
	}
	opts = append(opts, authOption{Title: "Done", Description: "Confirm protocol", Value: "done"})
	return opts
}

func (a *App) authProviderNetworkOptions() []authOption {
	pe := &a.auth.Provider
	opts := []authOption{
		{Title: "HTTP Proxy", Description: valueOrDefault(pe.HTTPProxy, "(none)"), Value: "httpProxy"},
		{Title: "Force HTTP/1.1", Description: boolYesNo(pe.ForceHTTP11), Value: "forceHTTP11"},
	}
	opts = append(opts, authOption{Title: "Done", Description: "Confirm network", Value: "done"})
	return opts
}

func (a *App) authProviderAdvancedOptions() []authOption {
	pe := &a.auth.Provider
	opts := []authOption{
		{Title: "Headers", Description: fmt.Sprintf("%d header(s)", len(pe.Headers)), Value: "headers"},
		{Title: "Thinking Format", Description: valueOrDefault(pe.ThinkingFormat, "(auto)"), Value: "thinkingFormat"},
		{Title: "Cache Control", Description: a.authCacheControlSummary(pe.CacheControl), Value: "cacheControl"},
	}
	opts = append(opts, authOption{Title: "Done", Description: "Confirm advanced", Value: "done"})
	return opts
}

func (a *App) authProviderResponsesSummary(re *responsesEditState) string {
	parts := []string{}
	if re.ReasoningSummary != "" {
		parts = append(parts, "summary="+re.ReasoningSummary)
	}
	if re.PromptCacheEnabled == nil {
		parts = append(parts, "prompt-cache=auto")
	} else if *re.PromptCacheEnabled {
		parts = append(parts, "prompt-cache=on")
	} else {
		parts = append(parts, "prompt-cache=off")
	}
	if re.PromptCacheKey != "" {
		parts = append(parts, "cache-key=set")
	}
	if re.PromptCacheRetention != "" {
		parts = append(parts, "retention="+re.PromptCacheRetention)
	}
	if len(parts) == 0 {
		return "(defaults)"
	}
	return strings.Join(parts, "  ")
}

func (a *App) authCacheControlSummary(v *bool) string {
	if v == nil {
		return "auto"
	}
	if *v {
		return "enabled"
	}
	return "disabled"
}

// --- Responses sub-form ---

func (a *App) authResponsesOptions() []authOption {
	re := &a.auth.Provider.Responses
	opts := []authOption{
		{Title: "Reasoning Summary", Description: valueOrDefault(re.ReasoningSummary, "auto"), Value: "reasoningSummary"},
		{Title: "Prompt Cache Enabled", Description: a.authCacheControlSummary(re.PromptCacheEnabled), Value: "promptCacheEnabled"},
		{Title: "Prompt Cache Key", Description: valueOrDefault(re.PromptCacheKey, "(auto)"), Value: "promptCacheKey"},
		{Title: "Prompt Cache Retention", Description: valueOrDefault(re.PromptCacheRetention, "(default)"), Value: "promptCacheRetention"},
	}
	opts = append(opts, authOption{Title: "Done", Description: "Confirm responses", Value: "done"})
	return opts
}

// --- Headers editor ---

func (a *App) authHeadersOptions() []authOption {
	pe := &a.auth.Provider
	var keys []string
	for k := range pe.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	opts := make([]authOption, 0, len(keys)+2)
	for _, k := range keys {
		opts = append(opts, authOption{
			Title:       k,
			Description: pe.Headers[k],
			Value:       "edit:" + k,
		})
	}
	opts = append(opts, authOption{Title: "+ Add Header", Description: "Add a new HTTP header", Value: "add"})
	opts = append(opts, authOption{Title: "Done", Description: "Confirm headers", Value: "done"})
	return opts
}

func (a *App) authViewAPIChoiceOptions() []authOption {
	return []authOption{
		{Title: "OpenAI-compatible", Description: "api: openai-chat", Value: "openai-chat"},
		{Title: "OpenAI Responses", Description: "api: openai-responses", Value: "openai-responses"},
		{Title: "Anthropic-compatible", Description: "api: anthropic-messages", Value: "anthropic-messages"},
		{Title: "Gemini-compatible", Description: "api: google-gemini", Value: "google-gemini"},
		{Title: "Google Vertex", Description: "api: google-vertex", Value: "google-vertex"},
	}
}

func (a *App) selectAPIChoice(api string) {
	if api == "" {
		return
	}
	oldAPI := a.auth.Provider.API
	oldDefaultURL := defaultBaseURLForAPI(oldAPI)
	a.auth.Provider.API = api
	if strings.TrimSpace(a.auth.Provider.BaseURL) == "" || a.auth.Provider.BaseURL == oldDefaultURL {
		a.auth.Provider.BaseURL = defaultBaseURLForAPI(api)
	}
	a.popAuthView()
}

// --- Input handling for provider sub-forms ---

func (a *App) authProviderInputPrompt() string {
	switch a.auth.ParamField {
	case "apiKey":
		return "API key or ${ENV_VAR} reference (e.g. ${OPENAI_API_KEY})"
	case "vendor":
		return "Vendor adapter name (e.g. bailian, zai, …)"
	case "api":
		return "openai-chat / openai-responses / anthropic-messages / google-gemini / google-vertex"
	case "baseUrl":
		return "API endpoint URL (e.g. https://api.openai.com/v1)"
	case "httpProxy":
		return "HTTP proxy URL (e.g. http://127.0.0.1:7890) or empty"
	case "thinkingFormat":
		return "Thinking format: openai, anthropic, deepseek, xiaomi, zai"
	case "reasoningSummary":
		return "Reasoning summary level: auto, concise, detailed"
	case "promptCacheKey":
		return "Explicit prompt cache key or empty for auto"
	case "promptCacheRetention":
		return "Prompt cache retention value or empty for default"
	case "headerKey":
		return "HTTP header name (e.g. Authorization)"
	case "headerValue":
		return fmt.Sprintf("Value for header '%s'", a.auth.ParamFieldKey)
	case "newModelID":
		return "Enter model ID:"
	case "newModelName":
		return fmt.Sprintf("Enter display name for '%s' (empty = use ID):", a.auth.CurrentModelID)
	default:
		return "Enter value:"
	}
}

func (a *App) authProviderSubmitInput() error {
	value := strings.TrimSpace(a.authInput.Value())
	pe := &a.auth.Provider

	switch a.auth.ParamField {
	case "apiKey":
		pe.APIKey = value
	case "vendor":
		pe.Vendor = value
	case "api":
		if value == "" {
			return fmt.Errorf("API type is required")
		}
		pe.API = value
	case "baseUrl":
		if value == "" {
			return fmt.Errorf("base URL is required")
		}
		pe.BaseURL = value
	case "httpProxy":
		pe.HTTPProxy = value
	case "thinkingFormat":
		pe.ThinkingFormat = value
	case "reasoningSummary":
		pe.Responses.ReasoningSummary = value
	case "promptCacheKey":
		pe.Responses.PromptCacheKey = value
	case "promptCacheRetention":
		pe.Responses.PromptCacheRetention = value
	case "headerKey":
		if value == "" {
			return fmt.Errorf("header name is required")
		}
		a.auth.ParamFieldKey = value
		a.auth.ParamField = "headerValue"
		a.prepareAuthProviderInput()
		return errStayInInput
	case "headerValue":
		if pe.Headers == nil {
			pe.Headers = map[string]string{}
		}
		pe.Headers[a.auth.ParamFieldKey] = value
		a.clearAuthParamField()
		return nil
	}
	a.clearAuthParamField()
	return nil
}

// errStayInInput is a sentinel error indicating the input handler should not
// advance to the next view (used when a flow requires two sequential inputs).
var errStayInInput = fmt.Errorf("stay in input")

// prepareAuthProviderInput sets up the editor for the current ParamField.
func (a *App) prepareAuthProviderInput() {
	prompt := a.authProviderInputPrompt()
	value := a.authProviderInputValue()
	a.authInput = editor.New(max(20, a.width-8)).SetPlaceholder(prompt).SetMaxLines(3).Focus()
	if value != "" {
		a.authInput = a.authInput.SetValue(value)
	}
}

func (a *App) authProviderInputValue() string {
	pe := &a.auth.Provider
	switch a.auth.ParamField {
	case "apiKey":
		return pe.APIKey
	case "vendor":
		return pe.Vendor
	case "api":
		return pe.API
	case "baseUrl":
		return pe.BaseURL
	case "httpProxy":
		return pe.HTTPProxy
	case "thinkingFormat":
		return pe.ThinkingFormat
	case "reasoningSummary":
		return pe.Responses.ReasoningSummary
	case "promptCacheKey":
		return pe.Responses.PromptCacheKey
	case "promptCacheRetention":
		return pe.Responses.PromptCacheRetention
	case "headerValue":
		if v, ok := pe.Headers[a.auth.ParamFieldKey]; ok {
			return v
		}
	}
	return ""
}

// --- Provider field selection dispatcher ---

func (a *App) selectProviderFieldValue(value string) {
	switch value {
	case "done":
		a.popAuthView()
		return
	}

	// Special handling for toggle fields
	switch value {
	case "forceHTTP11":
		a.auth.Provider.ForceHTTP11 = !a.auth.Provider.ForceHTTP11
		a.scheduleRender()
		return
	case "cacheControl":
		a.auth.Provider.CacheControl = cycleTriState(a.auth.Provider.CacheControl)
		a.scheduleRender()
		return
	case "promptCacheEnabled":
		a.auth.Provider.Responses.PromptCacheEnabled = cycleTriState(a.auth.Provider.Responses.PromptCacheEnabled)
		a.scheduleRender()
		return
	}

	// For "responses" jump to responses sub-form
	if value == "responses" {
		a.pushAuthView(authViewResponsesEdit)
		return
	}
	// For "headers" jump to headers editor
	if value == "headers" {
		a.pushAuthView(authViewHeadersEdit)
		return
	}
	// For "api" jump to API type choice list
	if value == "api" {
		a.pushAuthView(authViewAPIChoice)
		return
	}

	a.auth.ParamField = value
	a.auth.ParamFieldKey = ""
	a.prepareAuthProviderInput()
	a.scheduleRender()
}

// --- Headers editor selection ---

func (a *App) selectHeaderValue(value string) {
	if value == "done" {
		a.popAuthView()
		return
	}
	if value == "add" {
		a.auth.ParamField = "headerKey"
		a.auth.ParamFieldKey = ""
		a.prepareAuthProviderInput()
		a.scheduleRender()
		return
	}
	if strings.HasPrefix(value, "edit:") {
		key := strings.TrimPrefix(value, "edit:")
		a.auth.ParamField = "headerValue"
		a.auth.ParamFieldKey = key
		a.prepareAuthProviderInput()
		a.scheduleRender()
		return
	}
}

// --- Tri-state helper ---

func cycleTriState(v *bool) *bool {
	if v == nil {
		b := true
		return &b
	}
	if *v {
		b := false
		return &b
	}
	return nil // was false → back to auto (nil)
}

// --- Utility ---

func boolYesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func valueOrDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func authViewProviderFromID(id string) authView {
	switch id {
	case providerGroupCredentials:
		return authViewProviderCredentials
	case providerGroupProtocol:
		return authViewProviderProtocol
	case providerGroupNetwork:
		return authViewProviderNetwork
	case providerGroupAdvanced:
		return authViewProviderAdvanced
	}
	return authViewProviderGroupList
}
