package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/startvibecoding/mothx/internal/tui/components/editor"
)

// --- Model group constants ---

const (
	modelGroupBasics       = "basics"
	modelGroupCapabilities = "capabilities"
	modelGroupSampling     = "sampling"
	modelGroupCost         = "cost"
	modelGroupCompat       = "compat"
)

// modelGroupInfo describes a model settings group.
type modelGroupInfo struct {
	ID          string
	Title       string
	Description string
}

var modelGroups = []modelGroupInfo{
	{modelGroupBasics, "A. Basics", "Name, Context window, Max tokens"},
	{modelGroupCapabilities, "B. Capabilities", "Reasoning, Input modalities"},
	{modelGroupSampling, "C. Sampling", "Temperature, Top P"},
	{modelGroupCost, "D. Cost", "Input/Output/CacheRead/CacheWrite pricing"},
	{modelGroupCompat, "E. Compatibility", "Thinking format, API flags"},
}

// currentModelEdit returns the modelEditState currently being edited.
func (a *App) currentModelEdit() *modelEditState {
	if a.auth.CurrentModelID == "" {
		return nil
	}
	return a.auth.Models[a.auth.CurrentModelID]
}

// prepareModelInput sets up the editor for a model field with placeholder
// explanation AND pre-filled default value.
func (a *App) prepareModelInput() {
	prompt := a.authModelInputPrompt()
	value := a.authModelInputValue()
	a.authInput = editor.New(max(20, a.width-8)).SetPlaceholder(prompt).SetMaxLines(3).Focus()
	if value != "" {
		a.authInput = a.authInput.SetValue(value)
	}
}

// authModelInputValue returns the current value for the active model field.
func (a *App) authModelInputValue() string {
	me := a.currentModelEdit()
	if me == nil {
		return ""
	}
	switch a.auth.ParamField {
	case "name":
		return me.Name
	case "contextWindow":
		if me.ContextWindow > 0 {
			return strconv.Itoa(me.ContextWindow)
		}
	case "maxTokens":
		if me.MaxTokens > 0 {
			return strconv.Itoa(me.MaxTokens)
		}
	case "input":
		if len(me.Input) > 0 {
			return strings.Join(me.Input, ",")
		}
	case "temperature":
		if me.Temperature != nil {
			return f64s(*me.Temperature)
		}
	case "topP":
		if me.TopP != nil {
			return f64s(*me.TopP)
		}
	case "costInput":
		if me.CostEnabled {
			return f64s(me.CostInput)
		}
	case "costOutput":
		if me.CostEnabled {
			return f64s(me.CostOutput)
		}
	case "cacheRead":
		if me.CostEnabled {
			return f64s(me.CacheRead)
		}
	case "cacheWrite":
		if me.CostEnabled {
			return f64s(me.CacheWrite)
		}
	case "thinkingFormat":
		return me.Compat.ThinkingFormat
	case "maxTokensField":
		return me.Compat.MaxTokensField
	}
	return ""
}

// --- Model list view ---

func (a *App) authModelListOptions() []authOption {
	opts := make([]authOption, 0, len(a.auth.ModelOrder)+2)
	// Add Model is always first
	opts = append(opts, authOption{Title: "+ Add Model", Description: "Add a new model entry", Value: "add"})
	for _, id := range a.auth.ModelOrder {
		if me, ok := a.auth.Models[id]; ok {
			opts = append(opts, authOption{
				Title:       id,
				Description: me.summaryShort(),
				Value:       "edit:" + id,
			})
		}
	}
	opts = append(opts, authOption{Title: "✓ Done", Description: "All models configured", Value: "done"})
	return opts
}

func (a *App) selectModelList(value string) {
	if value == "done" {
		// All models done → go to default setting
		if a.isReviewEdit() {
			a.returnToReviewAfterEdit()
		} else {
			a.pushAuthView(authViewDefault)
		}
		return
	}
	if value == "add" {
		a.pushAuthView(authViewAddModelID)
		return
	}
	if strings.HasPrefix(value, "edit:") {
		modelID := strings.TrimPrefix(value, "edit:")
		a.auth.CurrentModelID = modelID
		a.pushAuthView(authViewModelGroupList)
		return
	}
}

// --- authViewModelGroupList ---

func (a *App) authModelGroupOptions() []authOption {
	me := a.currentModelEdit()
	if me == nil {
		return []authOption{}
	}
	opts := make([]authOption, 0, len(modelGroups))
	for _, g := range modelGroups {
		desc := ""
		switch g.ID {
		case modelGroupBasics:
			desc = a.authModelBasicsSummary(me)
		case modelGroupCapabilities:
			desc = a.authModelCapabilitiesSummary(me)
		case modelGroupSampling:
			desc = a.authModelSamplingSummary(me)
		case modelGroupCost:
			desc = a.authModelCostSummary(me)
		case modelGroupCompat:
			desc = a.authModelCompatSummary(&me.Compat)
		}
		opts = append(opts, authOption{
			Title:       g.Title,
			Description: desc,
			Value:       g.ID,
		})
	}
	opts = append(opts, authOption{Title: "Done", Description: "Confirm model parameters", Value: "done"})
	return opts
}

func (a *App) authModelBasicsSummary(me *modelEditState) string {
	parts := []string{}
	if me.Name != me.ID && me.Name != "" {
		parts = append(parts, "name="+me.Name)
	}
	parts = append(parts, "ctx="+authItoa(me.ContextWindow))
	parts = append(parts, "max="+authItoa(me.MaxTokens))
	return strings.Join(parts, "  ")
}

func (a *App) authModelCapabilitiesSummary(me *modelEditState) string {
	parts := []string{}
	if me.Reasoning {
		parts = append(parts, "reasoning")
	}
	parts = append(parts, "in="+strings.Join(me.Input, ","))
	return strings.Join(parts, "  ")
}

func (a *App) authModelSamplingSummary(me *modelEditState) string {
	parts := []string{}
	if me.Temperature != nil {
		parts = append(parts, "t="+f64s(*me.Temperature))
	} else {
		parts = append(parts, "t=auto")
	}
	if me.TopP != nil {
		parts = append(parts, "p="+f64s(*me.TopP))
	} else {
		parts = append(parts, "p=auto")
	}
	return strings.Join(parts, "  ")
}

func (a *App) authModelCostSummary(me *modelEditState) string {
	if !me.CostEnabled {
		return "(disabled)"
	}
	parts := []string{}
	parts = append(parts, "in="+f64s(me.CostInput))
	parts = append(parts, "out="+f64s(me.CostOutput))
	if me.CacheRead > 0 {
		parts = append(parts, "cr="+f64s(me.CacheRead))
	}
	if me.CacheWrite > 0 {
		parts = append(parts, "cw="+f64s(me.CacheWrite))
	}
	return strings.Join(parts, "  ")
}

func (a *App) authModelCompatSummary(ce *compatEditState) string {
	if !ce.Active || ce.activeCount() == 0 {
		return "(none active)"
	}
	return fmt.Sprintf("%d flag(s) active", ce.activeCount())
}

// --- Model sub-form options ---

func (a *App) authModelBasicsOptions() []authOption {
	me := a.currentModelEdit()
	if me == nil {
		return nil
	}
	opts := []authOption{
		{Title: "Display Name", Description: valueOrDefault(me.Name, me.ID), Value: "name"},
		{Title: "Context Window", Description: authItoa(me.ContextWindow), Value: "contextWindow"},
		{Title: "Max Output Tokens", Description: authItoa(me.MaxTokens), Value: "maxTokens"},
	}
	opts = append(opts, authOption{Title: "Done", Description: "Confirm basics", Value: "done"})
	return opts
}

func (a *App) authModelCapabilitiesOptions() []authOption {
	me := a.currentModelEdit()
	if me == nil {
		return nil
	}
	opts := []authOption{
		{Title: "Reasoning", Description: boolYesNo(me.Reasoning), Value: "reasoning"},
		{Title: "Input Modalities", Description: strings.Join(me.Input, ","), Value: "input"},
	}
	opts = append(opts, authOption{Title: "Done", Description: "Confirm capabilities", Value: "done"})
	return opts
}

func (a *App) authModelSamplingOptions() []authOption {
	me := a.currentModelEdit()
	if me == nil {
		return nil
	}
	tempStr := "auto"
	if me.Temperature != nil {
		tempStr = f64s(*me.Temperature)
	}
	toppStr := "auto"
	if me.TopP != nil {
		toppStr = f64s(*me.TopP)
	}
	opts := []authOption{
		{Title: "Temperature", Description: tempStr, Value: "temperature"},
		{Title: "Top P", Description: toppStr, Value: "topP"},
	}
	opts = append(opts, authOption{Title: "Done", Description: "Confirm sampling", Value: "done"})
	return opts
}

func (a *App) authModelCostOptions() []authOption {
	me := a.currentModelEdit()
	if me == nil {
		return nil
	}
	opts := []authOption{
		{Title: "Enable Cost Tracking", Description: boolYesNo(me.CostEnabled), Value: "costEnabled"},
	}
	if me.CostEnabled {
		opts = append(opts,
			authOption{Title: "Input Cost (per 1M tokens)", Description: f64s(me.CostInput), Value: "costInput"},
			authOption{Title: "Output Cost (per 1M tokens)", Description: f64s(me.CostOutput), Value: "costOutput"},
			authOption{Title: "Cache Read Cost", Description: f64s(me.CacheRead), Value: "cacheRead"},
			authOption{Title: "Cache Write Cost", Description: f64s(me.CacheWrite), Value: "cacheWrite"},
		)
	}
	opts = append(opts, authOption{Title: "Done", Description: "Confirm cost", Value: "done"})
	return opts
}

func (a *App) authModelCompatOptions() []authOption {
	me := a.currentModelEdit()
	if me == nil {
		return nil
	}
	ce := &me.Compat
	opts := []authOption{
		{Title: "Thinking Format", Description: valueOrDefault(ce.ThinkingFormat, "(auto)"), Value: "thinkingFormat"},
		{Title: "Req.ReasoningContent→Asst", Description: boolYesNo(ce.RequiresReasoningContentOnAssistant), Value: "reqReasoningAsst"},
		{Title: "Req.ReasoningContent→AsstMsgs", Description: boolYesNo(ce.RequiresReasoningContentOnAssistantMessages), Value: "reqReasoningAsstMsgs"},
		{Title: "Force Adaptive Thinking", Description: boolYesNo(ce.ForceAdaptiveThinking), Value: "forceAdaptiveThinking"},
		{Title: "Parse ReasoningInContent", Description: boolYesNo(ce.ParseReasoningInContent), Value: "parseReasoningInContent"},
	}
	// API Params
	opts = append(opts,
		authOption{Title: "Supports Developer Role", Description: triStateStr(ce.SupportsDeveloperRole), Value: "supportsDeveloperRole"},
		authOption{Title: "Supports Store", Description: triStateStr(ce.SupportsStore), Value: "supportsStore"},
		authOption{Title: "Supports ReasoningEffort", Description: triStateStr(ce.SupportsReasoningEffort), Value: "supportsReasoningEffort"},
		authOption{Title: "Supports Strict Mode", Description: triStateStr(ce.SupportsStrictMode), Value: "supportsStrictMode"},
		authOption{Title: "Max Tokens Field", Description: valueOrDefault(ce.MaxTokensField, "(default)"), Value: "maxTokensField"},
	)
	// Cache
	opts = append(opts,
		authOption{Title: "CacheControlOnTools", Description: triStateStr(ce.SupportsCacheControlOnTools), Value: "cacheControlOnTools"},
		authOption{Title: "LongCacheRetention", Description: triStateStr(ce.SupportsLongCacheRetention), Value: "longCacheRetention"},
		authOption{Title: "PromptCacheKey", Description: triStateStr(ce.SupportsPromptCacheKey), Value: "promptCacheKey"},
		authOption{Title: "ReasoningSummary", Description: triStateStr(ce.SupportsReasoningSummary), Value: "reasoningSummary"},
	)
	// Streaming
	opts = append(opts,
		authOption{Title: "SessionAffinityHeaders", Description: boolYesNo(ce.SendSessionAffinityHeaders), Value: "sessionAffinityHeaders"},
		authOption{Title: "EagerToolStreaming", Description: triStateStr(ce.SupportsEagerToolInputStreaming), Value: "eagerToolStreaming"},
	)
	opts = append(opts,
		authOption{Title: "Reset All to Auto", Description: "Clear all compat flags", Value: "resetAll"},
		authOption{Title: "Done", Description: "Confirm compatibility", Value: "done"},
	)
	return opts
}

func authModelGroupFromID(id string) authView {
	switch id {
	case modelGroupBasics:
		return authViewModelBasics
	case modelGroupCapabilities:
		return authViewModelCapabilities
	case modelGroupSampling:
		return authViewModelSampling
	case modelGroupCost:
		return authViewModelCost
	case modelGroupCompat:
		return authViewModelCompat
	}
	return authViewModelGroupList
}

// --- Model field selection ---

func (a *App) selectModelFieldValue(value string) {
	if value == "done" {
		a.popAuthView()
		return
	}
	me := a.currentModelEdit()
	if me == nil {
		return
	}

	// Toggle fields
	switch value {
	case "reasoning":
		me.Reasoning = !me.Reasoning
		a.scheduleRender()
		return
	case "costEnabled":
		me.CostEnabled = !me.CostEnabled
		a.scheduleRender()
		return
	case "reqReasoningAsst":
		me.Compat.RequiresReasoningContentOnAssistant = !me.Compat.RequiresReasoningContentOnAssistant
		me.Compat.Active = true
		a.scheduleRender()
		return
	case "reqReasoningAsstMsgs":
		me.Compat.RequiresReasoningContentOnAssistantMessages = !me.Compat.RequiresReasoningContentOnAssistantMessages
		me.Compat.Active = true
		a.scheduleRender()
		return
	case "forceAdaptiveThinking":
		me.Compat.ForceAdaptiveThinking = !me.Compat.ForceAdaptiveThinking
		me.Compat.Active = true
		a.scheduleRender()
		return
	case "parseReasoningInContent":
		me.Compat.ParseReasoningInContent = !me.Compat.ParseReasoningInContent
		me.Compat.Active = true
		a.scheduleRender()
		return
	case "sessionAffinityHeaders":
		me.Compat.SendSessionAffinityHeaders = !me.Compat.SendSessionAffinityHeaders
		me.Compat.Active = true
		a.scheduleRender()
		return
	case "resetAll":
		me.Compat = compatEditState{}
		a.scheduleRender()
		return
	}

	// Tri-state pointer fields
	switch value {
	case "supportsDeveloperRole", "supportsStore", "supportsReasoningEffort", "supportsStrictMode",
		"cacheControlOnTools", "longCacheRetention", "promptCacheKey", "reasoningSummary", "eagerToolStreaming":
		a.toggleModelTriState(value)
		a.scheduleRender()
		return
	}

	// Text input fields
	a.auth.ParamField = value
	a.auth.ParamFieldKey = ""
	a.prepareModelInput()
	a.scheduleRender()
}

// --- Model input submit ---

func (a *App) authModelInputPrompt() string {
	switch a.auth.ParamField {
	case "name":
		return "Display name shown in model picker (e.g. Claude Sonnet 4.6)"
	case "contextWindow":
		return "Max context size in tokens (e.g. 200000)"
	case "maxTokens":
		return "Max output length in tokens (e.g. 64000)"
	case "input":
		return "Input types: text,image,audio,video,pdf"
	case "temperature":
		return "Sampling temperature 0.0–2.0 (0=deterministic)"
	case "topP":
		return "Nucleus sampling 0.0–1.0 (1.0=disabled)"
	case "costInput":
		return "Cost per 1M input tokens (USD)"
	case "costOutput":
		return "Cost per 1M output tokens (USD)"
	case "cacheRead":
		return "Cost per 1M cache-read tokens (USD)"
	case "cacheWrite":
		return "Cost per 1M cache-write tokens (USD)"
	case "thinkingFormat":
		return "Format: openai, anthropic, deepseek, xiaomi, zai"
	case "maxTokensField":
		return "API field name for max tokens (e.g. max_completion_tokens)"
	}
	if strings.HasPrefix(a.auth.ParamField, "req") || a.auth.ParamField == "forceAdaptiveThinking" ||
		a.auth.ParamField == "parseReasoningInContent" || a.auth.ParamField == "sessionAffinityHeaders" {
		return "Press Space to toggle, Enter to confirm"
	}
	if strings.HasPrefix(a.auth.ParamField, "supports") || a.auth.ParamField == "eagerToolStreaming" {
		return "Press Space to cycle: auto → enabled → disabled"
	}
	return "Enter value"
}

func (a *App) authModelSubmitInput() error {
	value := strings.TrimSpace(a.authInput.Value())
	me := a.currentModelEdit()
	if me == nil {
		return fmt.Errorf("no model selected")
	}

	switch a.auth.ParamField {
	case "name":
		me.Name = value
		if me.Name == "" {
			me.Name = me.ID
		}
	case "contextWindow":
		if value != "" {
			v, err := parsePositiveInt(value)
			if err != nil {
				return fmt.Errorf("context window must be a positive integer")
			}
			me.ContextWindow = v
		}
	case "maxTokens":
		if value != "" {
			v, err := parsePositiveInt(value)
			if err != nil {
				return fmt.Errorf("max tokens must be a positive integer")
			}
			me.MaxTokens = v
		}
	case "input":
		ids := normalizeAuthModelIDs(value)
		if len(ids) == 0 {
			ids = []string{"text"}
		}
		me.Input = ids
	case "temperature":
		if value != "" {
			v, err := parseFloatRange(value, 0, 2)
			if err != nil {
				return fmt.Errorf("temperature must be between 0 and 2")
			}
			me.Temperature = &v
		} else {
			me.Temperature = nil
		}
	case "topP":
		if value != "" {
			v, err := parseFloatRange(value, 0, 1)
			if err != nil {
				return fmt.Errorf("top_p must be between 0 and 1")
			}
			me.TopP = &v
		} else {
			me.TopP = nil
		}
	case "costInput":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid number")
		}
		me.CostInput = v
	case "costOutput":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid number")
		}
		me.CostOutput = v
	case "cacheRead":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid number")
		}
		me.CacheRead = v
	case "cacheWrite":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid number")
		}
		me.CacheWrite = v
	case "thinkingFormat":
		me.Compat.ThinkingFormat = value
		me.Compat.Active = true
	case "maxTokensField":
		me.Compat.MaxTokensField = value
		me.Compat.Active = true
	}

	a.clearAuthParamField()
	return nil
}

// --- Tri-state pointer toggle from key handler ---

func (a *App) toggleModelTriState(field string) {
	me := a.currentModelEdit()
	if me == nil {
		return
	}
	ce := &me.Compat
	var p **bool
	switch field {
	case "supportsDeveloperRole":
		p = &ce.SupportsDeveloperRole
	case "supportsStore":
		p = &ce.SupportsStore
	case "supportsReasoningEffort":
		p = &ce.SupportsReasoningEffort
	case "supportsStrictMode":
		p = &ce.SupportsStrictMode
	case "cacheControlOnTools":
		p = &ce.SupportsCacheControlOnTools
	case "longCacheRetention":
		p = &ce.SupportsLongCacheRetention
	case "promptCacheKey":
		p = &ce.SupportsPromptCacheKey
	case "reasoningSummary":
		p = &ce.SupportsReasoningSummary
	case "eagerToolStreaming":
		p = &ce.SupportsEagerToolInputStreaming
	default:
		return
	}
	*p = cycleTriState(*p)
	ce.Active = true
}

// --- Settings detail ---

func (a *App) authSettingsDetailOptions() []authOption {
	opts := []authOption{
		{Title: "Provider Settings", Description: a.auth.Provider.summaryShort(), Value: "providerGroups"},
	}
	for _, id := range a.auth.ModelOrder {
		if me, ok := a.auth.Models[id]; ok {
			opts = append(opts, authOption{
				Title:       "Model: " + id,
				Description: me.summaryShort(),
				Value:       "model:" + id,
			})
		}
	}
	opts = append(opts,
		authOption{Title: "+ Add Model", Description: "Add a new model entry", Value: "addModel"},
		authOption{Title: "Set as Default", Description: fmt.Sprintf("current: %v", a.auth.SetDefault), Value: "setDefault"},
		authOption{Title: "Review & Save", Description: "Preview and save all changes", Value: "review"},
	)
	return opts
}

func (a *App) selectSettingsDetail(value string) {
	switch value {
	case "providerGroups":
		a.pushAuthView(authViewProviderGroupList)
	case "addModel":
		a.pushAuthView(authViewAddModelID)
	case "setDefault":
		a.auth.SetDefault = !a.auth.SetDefault
		a.scheduleRender()
	case "review":
		a.prepareAuthPreview()
		a.pushAuthView(authViewReview)
	default:
		if strings.HasPrefix(value, "model:") {
			modelID := strings.TrimPrefix(value, "model:")
			a.auth.CurrentModelID = modelID
			a.pushAuthView(authViewModelGroupList)
		}
	}
}

// --- Utility ---

func triStateStr(v *bool) string {
	if v == nil {
		return "(auto)"
	}
	if *v {
		return "enabled"
	}
	return "disabled"
}
