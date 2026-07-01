package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/startvibecoding/vibecoding/internal/config"
)

// previewExpansion tracks which sections of the Review JSON are expanded.
type previewExpansion struct {
	CostExpand   bool
	CompatExpand bool
}

// renderAuthDialog renders the complete auth dialog overlay.
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
	lines = append(lines, a.authTitle(a.auth.View))
	lines = append(lines, "")
	if a.authInputActive() {
		lines = append(lines, a.authInputPrompt(a.auth.View))
		lines = append(lines, a.authInput.View())
		lines = append(lines, "")
		lines = append(lines, statusStyle.Render("Enter to submit, Esc to go back"))
	} else if a.auth.View == authViewReview {
		lines = append(lines, a.renderAuthPreviewLines()...)
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

// renderAuthPreviewLines renders the review preview with foldable cost/compat.
func (a *App) renderAuthPreviewLines() []string {
	if a.auth.Preview == "" {
		return nil
	}
	// If the preview doesn't contain fold markers, just truncate by lines
	if !strings.Contains(a.auth.Preview, previewFoldMarker) {
		return renderAuthPreview(a.auth.Preview)
	}
	rendered := renderFoldedPreview(a.auth.Preview, a.auth.PreviewExpand)
	return []string{rendered}
}

const previewFoldMarker = "◸fold:"

// renderFoldedPreview renders a preview string that contains fold markers.
// Sections marked with ◸fold:<name> are collapsed unless expand[name] is true.
func renderFoldedPreview(preview string, exp previewExpansion) string {
	lines := strings.Split(strings.TrimRight(preview, "\n"), "\n")
	var out []string
	foldedSections := map[string]*bool{
		"cost":   &exp.CostExpand,
		"compat": &exp.CompatExpand,
	}
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "\""+previewFoldMarker) {
			// Extract section name: "◸fold:cost": { ... }
			part := strings.TrimPrefix(trimmed, "\""+previewFoldMarker)
			if idx := strings.Index(part, "\""); idx >= 0 {
				part = part[:idx]
			}
			name := part
			expandPtr, known := foldedSections[name]
			if !known {
				// Unknown fold section — just show the line
				out = append(out, line)
				i++
				continue
			}
			if *expandPtr {
				// Expanded: show the full block
				out = append(out, line)
				i++
				// Copy until closing brace at same indent
				for i < len(lines) {
					out = append(out, lines[i])
					i++
					if strings.TrimSpace(lines[i-1]) == "}" || strings.TrimSpace(lines[i-1]) == "}," {
						break
					}
				}
			} else {
				// Collapsed: show one line with ▶ marker
				out = append(out, fmt.Sprintf("  ▶ %s: { … }", name))
				i++
				// Skip until closing brace
				for i < len(lines) {
					if strings.TrimSpace(lines[i]) == "}" || strings.TrimSpace(lines[i]) == "}," {
						i++
						break
					}
					i++
				}
			}
		} else {
			out = append(out, line)
			i++
		}
	}
	// Truncate if still too long
	if len(out) > authMaxPreviewVisibleLines {
		visible := append([]string(nil), out[:authMaxPreviewVisibleLines]...)
		visible = append(visible, statusStyle.Render(fmt.Sprintf("… %d more lines hidden", len(out)-authMaxPreviewVisibleLines)))
		return strings.Join(visible, "\n")
	}
	return strings.Join(out, "\n")
}

// renderAuthPreview truncates a plain (non-folded) preview to max visible lines.
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
	default:
		return statusStyle.Render("  │")
	}
}

func (a *App) authTitle(v authView) string {
	switch v {
	case authViewMain:
		return "Connect a Provider"
	case authViewExistingProvider:
		return "Existing Providers · Provider"
	case authViewCustomID:
		return "Custom Provider · Provider ID"
	case authViewSettingsDetail:
		return fmt.Sprintf("Settings · %s", a.auth.ProviderID)
	case authViewProviderGroupList:
		return fmt.Sprintf("Provider · %s · Settings", a.auth.ProviderID)
	case authViewProviderCredentials:
		return "Provider · Credentials"
	case authViewProviderProtocol:
		return "Provider · Protocol"
	case authViewProviderNetwork:
		return "Provider · Network"
	case authViewProviderAdvanced:
		return "Provider · Advanced"
	case authViewHeadersEdit:
		return "Provider · Headers"
	case authViewResponsesEdit:
		return "Provider · Responses"
	case authViewModelList:
		return "Provider · Models"
	case authViewModelGroupList:
		return fmt.Sprintf("Model · %s · Parameters", a.auth.CurrentModelID)
	case authViewModelBasics:
		return "Model · Basics"
	case authViewModelCapabilities:
		return "Model · Capabilities"
	case authViewModelSampling:
		return "Model · Sampling"
	case authViewModelCost:
		return "Model · Cost"
	case authViewModelCompat:
		return "Model · Compatibility"
	case authViewAddModelID:
		return "Add Model · ID"
	case authViewAddModelName:
		return "Add Model · Name"
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
	case authViewAddModelID:
		return "Enter model ID:"
	case authViewAddModelName:
		return fmt.Sprintf("Enter display name for '%s' (empty = use ID):", a.auth.CurrentModelID)
	case authViewProviderCredentials, authViewProviderProtocol, authViewProviderNetwork,
		authViewProviderAdvanced, authViewResponsesEdit:
		return a.authProviderInputPrompt()
	case authViewHeadersEdit:
		if a.auth.ParamField == "headerKey" {
			return "Enter header name:"
		}
		return fmt.Sprintf("Enter value for header '%s':", a.auth.ParamFieldKey)
	case authViewModelBasics, authViewModelCapabilities, authViewModelSampling,
		authViewModelCost, authViewModelCompat:
		return a.authModelInputPrompt()
	default:
		return "Input:"
	}
}

// --- Provider selection helpers (moved from auth_dialog.go) ---

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

// previewBuildFoldedJSON builds a preview JSON string with cost/compat sections
// marked with fold markers so renderFoldedPreview can collapse them.
func previewBuildFoldedJSON(next *config.Settings, providerID string, maskKey bool) string {
	preview := struct {
		DefaultProvider string                            `json:"defaultProvider,omitempty"`
		DefaultModel    string                            `json:"defaultModel,omitempty"`
		Providers       map[string]*config.ProviderConfig `json:"providers"`
	}{DefaultProvider: next.DefaultProvider, DefaultModel: next.DefaultModel, Providers: map[string]*config.ProviderConfig{}}
	pc := *next.Providers[providerID]
	if maskKey {
		pc.APIKey = maskAuthSecret(pc.APIKey)
	}
	preview.Providers[providerID] = &pc
	data, _ := json.MarshalIndent(preview, "", "  ")
	return insertFoldMarkers(string(data))
}

// insertFoldMarkers finds "cost" and "compat" objects in the JSON and marks them
// with a fold key so the renderer can collapse them.
func insertFoldMarkers(jsonStr string) string {
	// Simple approach: replace `"cost": {` with a marked version
	// This works because json.MarshalIndent always puts the key on its own line
	result := jsonStr
	// We add a sibling key right before "cost" that marks the fold
	result = replaceKeyWithFold(result, "cost")
	result = replaceKeyWithFold(result, "compat")
	return result
}

// replaceKeyWithFold replaces `"key": {` with `"◸fold:key": {` + original key
// so the renderer knows which section can be collapsed.
func replaceKeyWithFold(s, keyName string) string {
	// Pattern: newline + spaces + "key": {
	oldLine := fmt.Sprintf("%q:", keyName)
	newLine := fmt.Sprintf("%q: ", previewFoldMarker+keyName)
	return strings.ReplaceAll(s, oldLine, newLine)
}
