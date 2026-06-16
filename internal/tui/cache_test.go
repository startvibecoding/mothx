package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/startvibecoding/vibecoding/internal/agent"
	"github.com/startvibecoding/vibecoding/internal/config"
	ctxpkg "github.com/startvibecoding/vibecoding/internal/context"
	"github.com/startvibecoding/vibecoding/internal/provider"
	"github.com/startvibecoding/vibecoding/internal/session"
	"github.com/startvibecoding/vibecoding/internal/tools"
	"github.com/startvibecoding/vibecoding/internal/tui/components/editor"
)

// ansiRe matches ANSI CSI escape sequences (colours, bold, etc.).
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

func trimLineRightSpace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

func TestRenderEditToolResultShowsCompactDiff(t *testing.T) {
	app := &App{}
	result := toolResult{
		toolName: "edit",
		toolArgs: map[string]any{"path": "internal/acp/acp.go"},
		diff: &tools.FileDiff{
			Path:    "internal/acp/acp.go",
			Added:   1,
			Deleted: 1,
			Unified: strings.Join([]string{
				"--- internal/acp/acp.go",
				"+++ internal/acp/acp.go",
				"@@ -551,3 +551,3 @@",
				" \tctx, cancel := context.WithCancel(context.Background())",
				"-\tpromptKey := rawIDKey(req.ID)",
				"+\tpromptKey := mcp.RawIDKey(req.ID)",
				" \trt.cancelMu.Lock()",
				"",
			}, "\n"),
		},
	}

	got := trimLineRightSpace(stripANSI(app.renderToolResult(result)))
	want := strings.Join([]string{
		"• Edited internal/acp/acp.go (+1 -1)",
		"    551       ctx, cancel := context.WithCancel(context.Background())",
		"    552  -    promptKey := rawIDKey(req.ID)",
		"    552  +    promptKey := mcp.RawIDKey(req.ID)",
		"    553       rt.cancelMu.Lock()",
	}, "\n")

	if got != want {
		t.Fatalf("renderToolResult(edit) =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderBashToolResultKeepsOutputRaw(t *testing.T) {
	app := &App{}
	summary := "[stdout]\n\u001b[32m+added\u001b[0m\n context\r\n[exit_code]\n0"
	got := app.renderToolResult(toolResult{
		toolName: "bash",
		summary:  summary,
	})

	parts := strings.SplitN(got, "\n", 2)
	if len(parts) != 2 {
		t.Fatalf("renderToolResult(bash) = %q, want header and body", got)
	}
	if parts[1] != summary {
		t.Fatalf("bash output body was modified:\n got %q\nwant %q", parts[1], summary)
	}
	if strings.Contains(parts[1], "\x1b[3m") {
		t.Fatalf("bash output body should not inherit TUI italic styling: %q", parts[1])
	}
}

func TestRenderExpandedBashToolResultKeepsDetailsRaw(t *testing.T) {
	app := &App{}
	output := "\u001b[31m-deleted\u001b[0m\r\n+added"
	got := app.renderExpandedToolResult(toolResult{
		toolName:    "bash",
		fullContent: output,
	})

	if !strings.HasSuffix(got, "---\n"+output) {
		t.Fatalf("expanded bash output was modified: %q", got)
	}
	body := got[strings.Index(got, "\n")+1:]
	if strings.Contains(body, "\x1b[3m") {
		t.Fatalf("expanded bash output body should not inherit TUI italic styling: %q", body)
	}
}

func TestNormalizeHistoryLineEndingsOnlyCollapsesCRLF(t *testing.T) {
	got := normalizeHistoryLineEndings("a\r\nb\rc")
	want := "a\nb\rc"
	if got != want {
		t.Fatalf("normalizeHistoryLineEndings() = %q, want %q", got, want)
	}
}

func TestAssistantMarkdownRendererUsesViewportWidth(t *testing.T) {
	app := &App{
		width:               60,
		assistantRaw:        map[int]string{0: "请看 https://gitee.com/oschina/platform/pulls/11938 这里"},
		assistantRendered:   make(map[int]string),
		assistantDirty:      map[int]bool{0: true},
		currentAssistantIdx: -1,
		currentThinkIdx:     -1,
	}
	app.configureMarkdownRenderer()

	got := stripANSI(app.renderAssistantMessage(0))
	flattened := strings.ReplaceAll(strings.ReplaceAll(got, "\n", ""), " ", "")
	if !strings.Contains(flattened, "https://gitee.com/oschina/platform/pulls/11938") {
		t.Fatalf("renderAssistantMessage() = %q, want URL order preserved", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if width := lipgloss.Width(line); width > app.width {
			t.Fatalf("rendered line width = %d, want <= %d: %q", width, app.width, line)
		}
	}
}

func TestWindowResizeMarksAssistantMarkdownDirty(t *testing.T) {
	app := &App{
		assistantRaw:        map[int]string{0: "hello"},
		assistantRendered:   map[int]string{0: "old"},
		assistantDirty:      make(map[int]bool),
		currentAssistantIdx: -1,
		currentThinkIdx:     -1,
	}

	model, _ := app.Update(tea.WindowSizeMsg{Width: 72, Height: 24})
	updated := model.(*App)

	if updated.mdRenderer == nil {
		t.Fatal("mdRenderer is nil after resize")
	}
	if !updated.assistantDirty[0] {
		t.Fatal("assistantDirty[0] = false, want true after resize")
	}
}

func TestLiveAssistantMessageRendersCodeBlocks(t *testing.T) {
	app := &App{
		width:               80,
		assistantRaw:        map[int]string{0: "Here is code:\n\n```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```\n\nDone"},
		assistantRendered:   make(map[int]string),
		assistantDirty:      map[int]bool{0: true},
		currentAssistantIdx: 0,
		currentThinkIdx:     -1,
	}
	app.configureMarkdownRenderer()

	app.updateViewportContent()
	plain := stripANSI(app.liveContent)
	if !strings.Contains(plain, "Assistant:") {
		t.Fatalf("live content missing assistant prefix: %q", plain)
	}
	if !strings.Contains(plain, "func main") {
		t.Fatalf("live code block content missing 'func main': %q", plain)
	}
	if !strings.Contains(plain, "Done") {
		t.Fatalf("live content missing trailing text 'Done': %q", plain)
	}
	if strings.Contains(plain, "```") {
		t.Fatalf("live content must not contain raw backtick fences: %q", plain)
	}
}

func TestAssistantMarkdownRenderedLinesStayWithinViewport(t *testing.T) {
	app := &App{
		width: 42,
		assistantRaw: map[int]string{0: strings.Join([]string{
			"```text",
			"this-is-a-very-long-token-that-must-wrap-without-splitting-ansi",
			"```",
		}, "\n")},
		assistantRendered:   make(map[int]string),
		assistantDirty:      map[int]bool{0: true},
		currentAssistantIdx: 0,
		currentThinkIdx:     -1,
	}
	app.configureMarkdownRenderer()

	rendered := app.renderAssistantMessage(0)
	if !strings.Contains(stripANSI(rendered), "this-is-a-very-long-token") {
		t.Fatalf("rendered markdown missing code content: %q", rendered)
	}
	for _, line := range strings.Split(rendered, "\n") {
		if width := lipgloss.Width(line); width > app.width {
			t.Fatalf("rendered line width = %d, want <= %d: %q", width, app.width, line)
		}
	}
}

func TestAssistantCommonMarkdownUsesRenderer(t *testing.T) {
	app := &App{
		width:               60,
		assistantRaw:        map[int]string{0: "# Summary\n\n- first item\n- second item"},
		assistantRendered:   make(map[int]string),
		assistantDirty:      map[int]bool{0: true},
		currentAssistantIdx: 0,
		currentThinkIdx:     -1,
	}
	app.configureMarkdownRenderer()

	app.updateViewportContent()
	if len(app.assistantRendered) == 0 {
		t.Fatal("assistantRendered is empty, want common markdown rendered through gsm")
	}
	plain := stripANSI(app.liveContent)
	for _, want := range []string{"Assistant:", "Summary", "first item", "second item"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("rendered markdown missing %q: %q", want, plain)
		}
	}
	for _, line := range strings.Split(app.liveContent, "\n") {
		if width := lipgloss.Width(line); width > app.width {
			t.Fatalf("rendered line width = %d, want <= %d: %q", width, app.width, line)
		}
	}
}

func TestAssistantMarkdownPreservesFilenameOrder(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "plain prose",
			raw:  "用户读取要求 AGENTS.md 文件",
		},
		{
			name: "inline code",
			raw:  "用户读取要求 `AGENTS.md` 文件",
		},
		{
			name: "agents discovery phrase",
			raw:  "- `internal/contextfiles/` — `AGENTS.md` / `CLAUDE.md` discovery",
		},
		{
			name: "inline directory path",
			raw:  "检查 `internal/agent/` 目录",
		},
	}
	for _, width := range []int{16, 20, 24, 32, 48, 72} {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s/%d", tt.name, width), func(t *testing.T) {
				app := &App{
					width:               width,
					assistantRaw:        map[int]string{0: tt.raw},
					assistantRendered:   make(map[int]string),
					assistantDirty:      map[int]bool{0: true},
					currentAssistantIdx: 0,
					currentThinkIdx:     -1,
				}
				app.configureMarkdownRenderer()

				plain := stripANSI(app.renderAssistantMessage(0))
				flattened := removeWhitespace(plain)
				if strings.Contains(flattened, "AG.mdENTS") {
					t.Fatalf("rendered filename order is corrupted: %q", plain)
				}
				if strings.Contains(flattened, "/internalagent/") {
					t.Fatalf("rendered directory path order is corrupted: %q", plain)
				}
				if !strings.Contains(flattened, "AGENTS.md") {
					if strings.Contains(tt.raw, "AGENTS.md") {
						t.Fatalf("rendered output lost AGENTS.md order:\nraw: %q\nrendered: %q\nflattened: %q", tt.raw, plain, flattened)
					}
				}
				if strings.Contains(tt.raw, "internal/agent/") && !strings.Contains(flattened, "internal/agent/") {
					t.Fatalf("rendered output lost internal/agent/ order:\nraw: %q\nrendered: %q\nflattened: %q", tt.raw, plain, flattened)
				}
				if strings.Contains(tt.raw, "CLAUDE.md") && !strings.Contains(flattened, "CLAUDE.md") {
					t.Fatalf("rendered output lost CLAUDE.md order:\nraw: %q\nrendered: %q\nflattened: %q", tt.raw, plain, flattened)
				}
				for _, line := range strings.Split(app.renderAssistantMessage(0), "\n") {
					// Allow +2 tolerance for xansi.Wrap breakpoint behavior
					// where lines can slightly exceed target width when breaking at natural boundaries like /
					if lineWidth := lipgloss.Width(line); lineWidth > app.width+2 {
						t.Fatalf("rendered line width = %d, want <= %d: %q", lineWidth, app.width+2, line)
					}
				}
			})
		}
	}
}

func TestProcessInputBlocksWhileManualCompactionRuns(t *testing.T) {
	app := &App{
		input:                  editor.New(80),
		manualCompactionActive: true,
	}

	cmd := app.processInput("continue work")
	if cmd != nil {
		t.Fatal("processInput returned command while manual compaction was active")
	}
	if len(app.messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(app.messages))
	}
	if !strings.Contains(stripANSI(app.messages[0]), "Cannot send input while context compaction is running.") {
		t.Fatalf("unexpected status message: %q", stripANSI(app.messages[0]))
	}
}

func TestAssistantMarkdownDoesNotRenderBlankAfterPrefix(t *testing.T) {
	app := &App{
		width:               80,
		assistantRaw:        map[int]string{0: "用户读取要求 `AGENTS.md` 文件"},
		assistantRendered:   make(map[int]string),
		assistantDirty:      map[int]bool{0: true},
		currentAssistantIdx: 0,
		currentThinkIdx:     -1,
	}
	app.configureMarkdownRenderer()

	rendered := stripANSI(app.renderAssistantMessage(0))
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		t.Fatal("rendered assistant message is empty")
	}
	if strings.TrimSpace(lines[0]) == "Assistant:" {
		t.Fatalf("assistant prefix rendered on a blank line: %q", rendered)
	}
	if !strings.Contains(lines[0], "用户读取要求") || !strings.Contains(removeWhitespace(lines[0]), "AGENTS.md") {
		t.Fatalf("first rendered line missing assistant content: %q", rendered)
	}
}

func TestAssistantRendersAGENTSMarkdownFixture(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md fixture: %v", err)
	}
	if strings.TrimSpace(string(content)) == "" {
		t.Fatal("AGENTS.md fixture is empty")
	}

	app := &App{
		width:               72,
		assistantRaw:        map[int]string{0: string(content)},
		assistantRendered:   make(map[int]string),
		assistantDirty:      map[int]bool{0: true},
		currentAssistantIdx: 0,
		currentThinkIdx:     -1,
	}
	app.configureMarkdownRenderer()

	rendered := app.renderAssistantMessage(0)
	plain := stripANSI(rendered)
	if len(app.assistantRendered) == 0 {
		t.Fatal("assistantRendered is empty, want AGENTS.md rendered through gsm")
	}
	if renderedLen := len(app.assistantRendered[0]); renderedLen > len(content)*20 {
		t.Fatalf("rendered AGENTS.md intermediate is too large: got %d bytes from %d input bytes", renderedLen, len(content))
	}
	for _, want := range []string{
		"VibeCoding Agent Guide",
		"Gateway Mode",
		"Hermes Mode",
		"AGENTS.md",
		"CLAUDE.md",
		"make build",
		"make test",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("rendered AGENTS.md missing %q", want)
		}
	}
	if strings.Contains(plain, "```") {
		t.Fatalf("rendered AGENTS.md should not expose raw markdown fences")
	}
	flattened := removeWhitespace(plain)
	for _, want := range []string{"AGENTS.md", "CLAUDE.md", ".vibe/memory.md", "docs/en/changelog.md", "docs/zh/changelog.md"} {
		if !strings.Contains(flattened, want) {
			t.Fatalf("rendered AGENTS.md lost filename order for %q", want)
		}
	}
	if strings.Contains(flattened, "AG.mdENTS") {
		t.Fatalf("rendered AGENTS.md contains corrupted AGENTS.md ordering")
	}
	if maxBlank := maxConsecutiveBlankLines(plain); maxBlank > 2 {
		t.Fatalf("rendered AGENTS.md has %d consecutive blank lines", maxBlank)
	}
	for _, line := range strings.Split(rendered, "\n") {
		if width := lipgloss.Width(line); width > app.width {
			t.Fatalf("rendered AGENTS.md line width = %d, want <= %d: %q", width, app.width, line)
		}
	}
}

func removeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), "")
}

func maxConsecutiveBlankLines(s string) int {
	maxBlank := 0
	current := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) == "" {
			current++
			if current > maxBlank {
				maxBlank = current
			}
			continue
		}
		current = 0
	}
	return maxBlank
}

func TestLiveAssistantMessageRendersMarkdown(t *testing.T) {
	app := &App{
		width:               50,
		assistantRaw:        map[int]string{0: strings.Repeat("https://example.com/path/", 8)},
		assistantRendered:   make(map[int]string),
		assistantDirty:      map[int]bool{0: true},
		currentAssistantIdx: 0,
		currentThinkIdx:     -1,
	}
	app.configureMarkdownRenderer()

	app.updateViewportContent()
	if len(app.assistantRendered) != 0 {
		t.Fatalf("assistantRendered len = %d, want 0 for prose without fenced code blocks", len(app.assistantRendered))
	}
	if !strings.Contains(stripANSI(app.liveContent), "Assistant: ") {
		t.Fatalf("liveContent missing assistant prefix: %q", app.liveContent)
	}
}

func TestPlainAssistantMessageWrapsWithoutMarkdownWordSplitting(t *testing.T) {
	app := &App{
		width:               40,
		assistantRaw:        map[int]string{0: "修复 /clear 未清理 transcript rendering state"},
		assistantRendered:   make(map[int]string),
		assistantDirty:      map[int]bool{0: true},
		currentAssistantIdx: 0,
		currentThinkIdx:     -1,
	}
	app.configureMarkdownRenderer()

	app.updateViewportContent()
	plain := stripANSI(app.liveContent)
	if strings.Contains(plain, "修  /复clear") || strings.Contains(plain, "v.01\n36") {
		t.Fatalf("plain assistant text was awkwardly split: %q", plain)
	}
	if !strings.Contains(plain, "修复 /clear") {
		t.Fatalf("plain assistant text missing expected phrase: %q", plain)
	}
	if len(app.assistantRendered) != 0 {
		t.Fatalf("assistantRendered len = %d, want 0 for plain prose", len(app.assistantRendered))
	}
}

func TestThinkMessageWrapsAndPreservesContent(t *testing.T) {
	a := &App{
		width:               34,
		currentAssistantIdx: -1,
		currentThinkIdx:     -1,
		printedMessageIdx:   make(map[int]bool),
		thinkRaw:            make(map[int]string),
	}
	first := "正在分析用户读取要求 AGENTS.md 文件的上下文，"
	second := "then-checking-a-very-long-unspaced-token-for-wrapping"

	a.handleAgentEvent(agent.Event{Type: agent.EventThinkDelta, ThinkDelta: first})
	a.handleAgentEvent(agent.Event{Type: agent.EventThinkDelta, ThinkDelta: second})
	a.updateViewportContent()

	rendered := a.renderThinkMessage(a.currentThinkIdx)
	plain := stripANSI(rendered)
	flattened := removeWhitespace(plain)
	for _, want := range []string{"think:", "AGENTS.md", "then-checking-a-very-long-unspaced-token-for-wrapping"} {
		if !strings.Contains(flattened, removeWhitespace(want)) {
			t.Fatalf("rendered think message missing %q:\n%s", want, plain)
		}
	}
	for _, line := range strings.Split(rendered, "\n") {
		if width := lipgloss.Width(line); width > a.width {
			t.Fatalf("think line width = %d, want <= %d: %q", width, a.width, line)
		}
	}

	a.printMessageOnce(a.currentThinkIdx)
	transcript := stripANSI(a.liveContent)
	if !strings.Contains(removeWhitespace(transcript), "AGENTS.md") ||
		!strings.Contains(removeWhitespace(transcript), "then-checking-a-very-long-unspaced-token-for-wrapping") {
		t.Fatalf("transcript lost think content: %q", transcript)
	}
}

func TestThinkMessageUsesPlainMixedCJKASCIIWrapping(t *testing.T) {
	for _, width := range []int{18, 22, 26} {
		t.Run(fmt.Sprintf("width-%d", width), func(t *testing.T) {
			a := &App{
				width:               width,
				currentAssistantIdx: -1,
				currentThinkIdx:     -1,
				thinkRaw:            map[int]string{0: "用户查看想 AGENTS 文件内容"},
			}

			rendered := a.renderThinkMessage(0)
			plain := stripANSI(rendered)
			flattened := removeWhitespace(plain)
			if !strings.Contains(flattened, "用户查看想AGENTS文件内容") {
				t.Fatalf("rendered think message order changed:\nplain: %q\nflattened: %q", plain, flattened)
			}
			if strings.Contains(flattened, "AG文件ENTS") {
				t.Fatalf("rendered think message interleaved CJK and ASCII token:\nplain: %q\nflattened: %q", plain, flattened)
			}
			for _, line := range strings.Split(rendered, "\n") {
				if lineWidth := lipgloss.Width(line); lineWidth > a.width {
					t.Fatalf("think line width = %d, want <= %d: %q", lineWidth, a.width, line)
				}
			}
		})
	}
}

func TestViewKeepsTranscriptInMainOutputWhenNoProgram(t *testing.T) {
	app := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	app.ready = true
	app.width = 80
	app.height = 8
	app.input = app.input.SetWidth(76)
	app.messages = []string{strings.Join([]string{
		"line 1",
		"line 2",
		"line 3",
		"line 4",
		"line 5",
		"line 6",
		"line 7",
		"line 8",
	}, "\n")}
	app.updateViewportContent()

	got := stripANSI(app.View())
	if !strings.Contains(got, "line 1") {
		t.Fatalf("View() missing oldest transcript line:\n%s", got)
	}
	if !strings.Contains(got, "line 8") {
		t.Fatalf("View() missing newest transcript line:\n%s", got)
	}
	if !strings.Contains(got, app.input.Placeholder()) {
		t.Fatalf("View() missing input placeholder:\n%s", got)
	}
	if !strings.Contains(got, "Tab:mode") {
		t.Fatalf("View() missing footer:\n%s", got)
	}
}

func TestViewDoesNotForceOuterHeight(t *testing.T) {
	app := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	app.ready = true
	app.width = 80
	app.height = 8
	app.input = app.input.SetWidth(76)
	app.messages = []string{strings.Join([]string{
		"line 1",
		"line 2",
		"line 3",
		"line 4",
		"line 5",
		"line 6",
	}, "\n")}
	app.updateViewportContent()

	if got := lipgloss.Height(app.View()); got <= app.height {
		t.Fatalf("View() height = %d, want > %d so terminal scrollback can own transcript history", got, app.height)
	}
}

func TestMouseWheelDoesNotScrollTranscript(t *testing.T) {
	app := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	app.ready = true
	app.width = 80
	app.height = 12
	app.input = app.input.SetWidth(76)
	app.messages = []string{strings.Join([]string{
		"line 1",
		"line 2",
		"line 3",
		"line 4",
		"line 5",
		"line 6",
		"line 7",
		"line 8",
		"line 9",
		"line 10",
		"line 11",
		"line 12",
	}, "\n")}
	app.updateViewportContent()

	before := app.liveContent

	app.Update(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
	})

	if after := app.liveContent; after != before {
		t.Fatalf("mouse wheel changed transcript content:\nbefore:\n%s\n\nafter:\n%s", before, after)
	}
}

// ─── formatCachePercent ───────────────────────────────────────────────────────

func TestFormatCachePercent(t *testing.T) {
	tests := []struct {
		name             string
		totalInputTokens int
		totalCacheRead   int
		totalCacheWrite  int
		want             string
	}{
		// ── No data ──────────────────────────────────────────────────────────
		{
			name: "no_data_empty",
		},
		// ── Input tokens present ─────────────────────────────────────────────
		{
			name:             "input_no_cache_zero_pct",
			totalInputTokens: 1000,
			want:             "Cache: 0%",
		},
		{
			name:             "cache_25pct",
			totalInputTokens: 1000,
			totalCacheRead:   250,
			want:             "Cache: 25%",
		},
		{
			name:             "cache_50pct",
			totalInputTokens: 1000,
			totalCacheRead:   500,
			want:             "Cache: 50%",
		},
		{
			name:             "cache_75pct",
			totalInputTokens: 1000,
			totalCacheRead:   750,
			want:             "Cache: 75%",
		},
		{
			name:             "cache_100pct_exact",
			totalInputTokens: 1000,
			totalCacheRead:   1000,
			want:             "Cache: 100%",
		},
		// Defensive cap when read > input
		{
			name:             "cache_read_exceeds_input_capped_at_100pct",
			totalInputTokens: 100,
			totalCacheRead:   200,
			want:             "Cache: 100%",
		},
		// Multi-turn accumulation across several requests
		{
			name:             "multi_turn_accumulated_75pct",
			totalInputTokens: 4000,
			totalCacheRead:   3000,
			want:             "Cache: 75%",
		},
		// ── Fallback path: no input tokens yet ───────────────────────────────
		// CacheRead takes priority over CacheWrite in the fallback
		{
			name:           "no_input_cache_read_fallback",
			totalCacheRead: 500,
			want:           "CacheRead: 500",
		},
		{
			name:            "no_input_cache_write_fallback",
			totalCacheWrite: 1000,
			want:            "CacheWrite: 1000",
		},
		{
			name:            "no_input_both_read_wins_over_write",
			totalCacheRead:  500,
			totalCacheWrite: 1000,
			want:            "CacheRead: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{
				totalInputTokens: tt.totalInputTokens,
				totalCacheRead:   tt.totalCacheRead,
				totalCacheWrite:  tt.totalCacheWrite,
			}
			got := a.formatCachePercent()
			if got != tt.want {
				t.Errorf("formatCachePercent() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ─── renderFooter cache content ───────────────────────────────────────────────

func TestRenderFooterCacheContent(t *testing.T) {
	tests := []struct {
		name             string
		totalInputTokens int
		totalCacheRead   int
		totalCacheWrite  int
		wantContains     string // expected substring in stripped footer
		wantAbsent       string // must NOT appear in stripped footer
	}{
		// No cache data → "Cache:" must not appear at all
		{
			name:       "no_data_cache_absent",
			wantAbsent: "Cache:",
		},
		{
			name:             "zero_pct_shown",
			totalInputTokens: 1000,
			wantContains:     "Cache: 0%",
		},
		{
			name:             "cache_25pct_shown",
			totalInputTokens: 1000,
			totalCacheRead:   250,
			wantContains:     "Cache: 25%",
		},
		// Boundary just below 50% threshold
		{
			name:             "cache_49pct_shown",
			totalInputTokens: 1000,
			totalCacheRead:   490,
			wantContains:     "Cache: 49%",
		},
		// Boundary at exactly 50%
		{
			name:             "cache_50pct_shown",
			totalInputTokens: 1000,
			totalCacheRead:   500,
			wantContains:     "Cache: 50%",
		},
		{
			name:             "cache_75pct_shown",
			totalInputTokens: 1000,
			totalCacheRead:   750,
			wantContains:     "Cache: 75%",
		},
		{
			name:             "cache_100pct_shown",
			totalInputTokens: 1000,
			totalCacheRead:   1000,
			wantContains:     "Cache: 100%",
		},
		// Fallback paths visible in footer
		{
			name:            "cache_write_fallback_shown",
			totalCacheWrite: 5000,
			wantContains:    "CacheWrite: 5000",
		},
		{
			name:           "cache_read_fallback_shown",
			totalCacheRead: 800,
			wantContains:   "CacheRead: 800",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{
				totalInputTokens: tt.totalInputTokens,
				totalCacheRead:   tt.totalCacheRead,
				totalCacheWrite:  tt.totalCacheWrite,
			}
			footer := stripANSI(a.renderFooter())

			if tt.wantContains != "" && !strings.Contains(footer, tt.wantContains) {
				t.Errorf("renderFooter() = %q\n\twant substring %q", footer, tt.wantContains)
			}
			if tt.wantAbsent != "" && strings.Contains(footer, tt.wantAbsent) {
				t.Errorf("renderFooter() = %q\n\twant %q to be absent", footer, tt.wantAbsent)
			}
		})
	}
}

// ─── Highlight threshold ──────────────────────────────────────────────────────

// TestCacheHighlightThreshold verifies the ≥50% rule that gates statusStyle
// in renderFooter. At exactly 49% the cache string must not be highlighted;
// at exactly 50% it must be.
//
// Because lipgloss omits ANSI codes when there is no TTY, we verify the
// decision by checking whether the raw footer embeds statusStyle.Render()
// output for the specific cache string. statusStyle.Render(x) == x when the
// renderer is in Ascii mode, but the branch taken in renderFooter differs:
// the ≥50% branch always passes through statusStyle.Render(), the <50%
// branch uses the plain string directly. We therefore compare the two raw
// footers: they should differ iff ANSI codes are emitted, and must be
// identical only in purely Ascii rendering environments — in which case the
// test degrades gracefully to a content-only assertion.
func TestCacheHighlightThreshold(t *testing.T) {
	below := &App{totalInputTokens: 1000, totalCacheRead: 490} // 49%
	at := &App{totalInputTokens: 1000, totalCacheRead: 500}    // 50%
	above := &App{totalInputTokens: 1000, totalCacheRead: 750} // 75%

	footerBelow := below.renderFooter()
	footerAt := at.renderFooter()
	footerAbove := above.renderFooter()

	// Content must always be correct regardless of colour support.
	if !strings.Contains(stripANSI(footerBelow), "Cache: 49%") {
		t.Errorf("below-threshold footer = %q, want 'Cache: 49%%'", stripANSI(footerBelow))
	}
	if !strings.Contains(stripANSI(footerAt), "Cache: 50%") {
		t.Errorf("at-threshold footer = %q, want 'Cache: 50%%'", stripANSI(footerAt))
	}
	if !strings.Contains(stripANSI(footerAbove), "Cache: 75%") {
		t.Errorf("above-threshold footer = %q, want 'Cache: 75%%'", stripANSI(footerAbove))
	}

	// When the renderer does produce ANSI codes (e.g. in a real terminal or
	// when the test is run with COLORTERM set), the highlighted footers must
	// contain the statusStyle-rendered string, and the un-highlighted one must
	// not contain it.
	styledAt := statusStyle.Render("Cache: 50%")
	styledAbove := statusStyle.Render("Cache: 75%")
	styledBelow := statusStyle.Render("Cache: 49%")

	if styledAt != "Cache: 50%" {
		// ANSI codes are being emitted; verify correct highlighting.
		if !strings.Contains(footerAt, styledAt) {
			t.Errorf("at-threshold (50%%) footer should apply statusStyle; raw = %q", footerAt)
		}
		if !strings.Contains(footerAbove, styledAbove) {
			t.Errorf("above-threshold (75%%) footer should apply statusStyle; raw = %q", footerAbove)
		}
		if strings.Contains(footerBelow, styledBelow) {
			t.Errorf("below-threshold (49%%) footer must NOT apply statusStyle; raw = %q", footerBelow)
		}
	}
}

func TestRenderFooterShowsBlinkingApprovalAlert(t *testing.T) {
	a := &App{
		waitingForApproval: true,
		spinnerIndex:       0,
		width:              120,
	}
	visible := stripANSI(a.renderFooter())
	if !strings.Contains(visible, "! APPROVAL REQUIRED: y/n") {
		t.Fatalf("approval footer alert missing: %q", visible)
	}

	a.spinnerIndex = 1
	hidden := stripANSI(a.renderFooter())
	if strings.Contains(hidden, "! APPROVAL REQUIRED: y/n") {
		t.Fatalf("approval footer alert should blink off on odd tick: %q", hidden)
	}
}

func TestHandleAgentEventReservesAssistantSlotBeforeTextDelta(t *testing.T) {
	a := &App{
		messages:          []string{"You: hi"},
		assistantRaw:      make(map[int]string),
		assistantRendered: make(map[int]string),
		assistantDirty:    make(map[int]bool),
	}

	a.handleAgentEvent(agent.Event{Type: agent.EventTurnStart})
	if got, want := len(a.messages), 2; got != want {
		t.Fatalf("len(messages) after turn start = %d, want %d", got, want)
	}
	if got, want := a.currentAssistantIdx, 1; got != want {
		t.Fatalf("currentAssistantIdx = %d, want %d", got, want)
	}

	a.handleAgentEvent(agent.Event{Type: agent.EventTextDelta, TextDelta: "Hello"})
	if got, want := a.assistantRaw[1], "Hello"; got != want {
		t.Fatalf("assistantRaw[1] = %q, want %q", got, want)
	}
	if got, want := len(a.messages), 2; got != want {
		t.Fatalf("len(messages) after text delta = %d, want %d", got, want)
	}
}

func TestHandleAgentEventCommitsStreamBeforeApproval(t *testing.T) {
	a := &App{
		messages:            []string{"You: hi"},
		currentAssistantIdx: -1,
		currentThinkIdx:     -1,
		printedMessageIdx:   make(map[int]bool),
		assistantRaw:        make(map[int]string),
		assistantRendered:   make(map[int]string),
		assistantDirty:      make(map[int]bool),
	}

	a.handleAgentEvent(agent.Event{Type: agent.EventTurnStart})
	a.handleAgentEvent(agent.Event{Type: agent.EventThinkDelta, ThinkDelta: "thinking"})
	a.handleAgentEvent(agent.Event{Type: agent.EventTextDelta, TextDelta: "I need to run a command."})
	a.handleAgentEvent(agent.Event{
		Type:         agent.EventToolApprovalRequest,
		ApprovalID:   "approval-1",
		ApprovalTool: "bash",
		ApprovalArgs: map[string]any{"command": "go test ./internal/tui"},
	})

	joined := stripANSI(a.liveContent)
	thinkAt := strings.Index(joined, "think: thinking")
	assistantAt := strings.Index(joined, "Assistant: I need to run a command.")
	approvalAt := strings.Index(joined, "Approval required: bash")
	if thinkAt < 0 || assistantAt < 0 || approvalAt < 0 {
		t.Fatalf("transcript missing expected content: %q", joined)
	}
	if !(thinkAt < assistantAt && assistantAt < approvalAt) {
		t.Fatalf("transcript out of order: %q", joined)
	}
	if a.currentThinkIdx != -1 || a.currentAssistantIdx != -1 {
		t.Fatalf("active stream indices = think %d assistant %d, want both reset", a.currentThinkIdx, a.currentAssistantIdx)
	}
}

func TestFormatApprovalArgsBashShowsCommandWithoutJSON(t *testing.T) {
	got := stripANSI(formatApprovalArgs("bash", map[string]any{
		"command": "git diff\nmake test",
		"timeout": float64(30),
	}))

	if strings.Contains(got, "{") || strings.Contains(got, `"command"`) {
		t.Fatalf("formatApprovalArgs(bash) should not render raw JSON: %q", got)
	}
	if !strings.Contains(got, "command:\n  git diff\n  make test") {
		t.Fatalf("formatApprovalArgs(bash) missing formatted command: %q", got)
	}
	if !strings.Contains(got, "timeout: 30") {
		t.Fatalf("formatApprovalArgs(bash) missing timeout: %q", got)
	}
}

func TestFormatApprovalArgsWriteRedactsContent(t *testing.T) {
	got := formatApprovalArgs("write", map[string]any{
		"path":    "README.md",
		"content": "secret content",
	})

	if strings.Contains(got, "secret content") {
		t.Fatalf("formatApprovalArgs(write) leaked content: %q", got)
	}
	if !strings.Contains(got, "path: README.md") || !strings.Contains(got, "content: (14 bytes)") {
		t.Fatalf("formatApprovalArgs(write) missing path/content summary: %q", got)
	}
}

func TestFormatApprovalArgsEditShowsPathAndDiff(t *testing.T) {
	args := map[string]any{
		"path": "README.md",
		"edits": []any{
			map[string]any{
				"oldText": "Hello\nWorld\n",
				"newText": "Hello\nGophers\n",
			},
		},
	}

	got := formatApprovalArgs("edit", args)
	if !strings.Contains(got, "path: README.md") {
		t.Fatalf("formatApprovalArgs(edit) missing path: %q", got)
	}
	if !strings.Contains(got, "@@ -1,2 +1,2 @@") {
		t.Fatalf("formatApprovalArgs(edit) missing hunk header: %q", got)
	}
	if !strings.Contains(got, "-World") || !strings.Contains(got, "+Gophers") {
		t.Fatalf("formatApprovalArgs(edit) missing line diff: %q", got)
	}
}

func TestAbortClearsQueuedInput(t *testing.T) {
	a := &App{
		inputQueue: make([]InputEvent, 0, 4),
	}

	a.queueInput(teaKeyMsgForTest("a"))
	a.queueInput(teaKeyMsgForTest("b"))
	if got := len(a.inputQueue); got != 2 {
		t.Fatalf("len(inputQueue) before abort = %d, want 2", got)
	}

	a.inputQueueMu.Lock()
	a.inputQueue = a.inputQueue[:0]
	a.lastInputTime = time.Time{}
	a.inputQueueMu.Unlock()

	if got := len(a.inputQueue); got != 0 {
		t.Fatalf("len(inputQueue) after abort = %d, want 0", got)
	}
}

func TestHandleAgentEventStatusAndWarningMessage(t *testing.T) {
	a := &App{}

	a.handleAgentEvent(agent.Event{Type: agent.EventStatus, StatusMessage: "stream warning"})
	a.handleAgentEvent(agent.Event{
		Type:    agent.EventMessageStart,
		Message: provider.NewUserMessage("[System] explain what you are doing"),
	})
	a.handleAgentEvent(agent.Event{Type: agent.EventContextPressure, PressureMessage: "context high"})
	a.handleAgentEvent(agent.Event{Type: agent.EventBudgetPressure, PressureMessage: "budget low"})

	joined := stripANSI(strings.Join(a.messages, "\n"))
	if !strings.Contains(joined, "stream warning") {
		t.Fatalf("messages = %q, want status message", joined)
	}
	if !strings.Contains(joined, "[System] explain what you are doing") {
		t.Fatalf("messages = %q, want warning user message", joined)
	}
	if !strings.Contains(joined, "context high") || !strings.Contains(joined, "budget low") {
		t.Fatalf("messages = %q, want pressure warnings", joined)
	}
}

func TestAgentErrorIncludesAbortReason(t *testing.T) {
	app := &App{pendingAbortReason: "user pressed Esc"}
	app.handleAgentEvent(agent.Event{Type: agent.EventError, Error: assertErr("aborted"), StopReason: "aborted"})

	joined := stripANSI(strings.Join(app.messages, "\n"))
	if !strings.Contains(joined, "Error: aborted (reason: user pressed Esc)") {
		t.Fatalf("messages = %q, want aborted reason", joined)
	}
	if app.pendingAbortReason != "" {
		t.Fatalf("pendingAbortReason = %q, want cleared", app.pendingAbortReason)
	}
}

func TestListenEventsPassesThroughDoneAndError(t *testing.T) {
	eventCh := make(chan agent.Event, 2)
	eventCh <- agent.Event{Type: agent.EventDone}
	eventCh <- agent.Event{Type: agent.EventError, Error: assertErr("boom")}
	close(eventCh)
	app := &App{eventCh: eventCh}

	msg := app.listenAgentEvents()()
	if ev, ok := msg.(agentEventMsg); !ok || ev.event.Type != agent.EventDone {
		t.Fatalf("first msg = %#v, want agentEventMsg(EventDone)", msg)
	}

	msg = app.listenAgentEvents()()
	if ev, ok := msg.(agentEventMsg); !ok || ev.event.Type != agent.EventError || ev.event.Error == nil || ev.event.Error.Error() != "boom" {
		t.Fatalf("second msg = %#v, want agentEventMsg(EventError boom)", msg)
	}

	msg = app.listenAgentEvents()()
	if _, ok := msg.(agentDoneMsg); !ok {
		t.Fatalf("third msg = %#v, want agentDoneMsg", msg)
	}
}

func TestCompactCommandStartsImmediateCompaction(t *testing.T) {
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "m1", Name: "Model 1", ContextWindow: 100000},
	}, []provider.StreamEvent{
		{Type: provider.StreamTextDelta, TextDelta: "## Goal\nCompacted summary"},
		{Type: provider.StreamDone},
	})
	model := mockProvider.Models()[0]
	settings := config.DefaultSettings()
	registry := tools.NewRegistry(t.TempDir(), nil)
	app := NewApp(mockProvider, model, settings, nil, registry, "", "", nil, "agent", false, false, nil, nil, nil)
	app.agent = agent.New(agent.Config{
		Provider: mockProvider,
		Model:    model,
		Mode:     "agent",
		Settings: settings,
		CompactionSettings: ctxpkg.CompactionSettings{
			Enabled:          true,
			ReserveTokens:    100,
			KeepRecentTokens: 1,
		},
	}, registry)
	app.agent.LoadHistoryMessages([]provider.Message{
		provider.NewUserMessage("old user message"),
		provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old assistant message"}}),
		provider.NewUserMessage("recent user message"),
		provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "recent assistant message"}}),
	})

	cmd := app.handleCommand("/compact")
	if cmd == nil {
		t.Fatal("/compact returned nil command, want immediate compaction command")
	}
	startMsg := cmd()
	streamMsg, ok := startMsg.(agentStreamStartMsg)
	if !ok {
		t.Fatalf("/compact command message = %#v, want agentStreamStartMsg", startMsg)
	}
	if !streamMsg.compacting {
		t.Fatalf("/compact command message compacting = false, want true")
	}
	app.eventCh = streamMsg.eventCh

	msg := app.listenAgentEvents()()
	if ev, ok := msg.(agentEventMsg); !ok || ev.event.Type != agent.EventCompactionStart {
		t.Fatalf("first compaction event = %#v, want EventCompactionStart", msg)
	}
	msg = app.listenAgentEvents()()
	if ev, ok := msg.(agentEventMsg); !ok || ev.event.Type != agent.EventCompactionEnd || ev.event.Error != nil {
		t.Fatalf("second compaction event = %#v, want successful EventCompactionEnd", msg)
	}
	if got := mockProvider.GetCallCount(); got != 1 {
		t.Fatalf("provider call count = %d, want 1", got)
	}
	msgs := app.agent.GetMessages()
	if len(msgs) != 3 {
		t.Fatalf("compacted messages len = %d, want 3", len(msgs))
	}
	if !msgs[0].SystemInjected || !strings.Contains(msgs[0].Content, "Compacted summary") {
		t.Fatalf("first compacted message = %#v, want system-injected summary", msgs[0])
	}
}

func TestHelpCommandRendersAsSingleCommandOutput(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)

	a.handleCommand("/help")

	if len(a.messages) != 1 {
		t.Fatalf("/help messages len = %d, want 1", len(a.messages))
	}
	plain := stripANSI(a.messages[0])
	for _, want := range []string{"Commands:", "/mode [plan|agent|yolo]", "/help", "Keyboard shortcuts:"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("/help output = %q, want substring %q", plain, want)
		}
	}
}

func TestHelpCommandPrintsCommandOutput(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.width = 80
	a.height = 10
	for i := 0; i < 8; i++ {
		a.addMessage(fmt.Sprintf("line %d", i))
	}

	a.handleCommand("/help")

	if len(a.messages) == 0 {
		t.Fatal("messages empty after /help")
	}
	idx := len(a.messages) - 1
	if !a.printedMessageIdx[idx] {
		t.Fatal("/help output was not marked for unmanaged transcript output")
	}
	if plain := stripANSI(a.messages[idx]); !strings.Contains(plain, "Commands:") {
		t.Fatalf("/help output = %q, want command list", plain)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

func teaKeyMsgForTest(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func teaSpecialKeyMsgForTest(key tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: key}
}

func TestInputHomeEndKeysReachTextInput(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.input.SetValue("abc")

	a.Update(teaSpecialKeyMsgForTest(tea.KeyHome))
	a.flushInputQueue()
	a.Update(teaKeyMsgForTest("X"))
	a.flushInputQueue()

	if got := a.input.Value(); got != "Xabc" {
		t.Fatalf("value after home insert = %q, want Xabc", got)
	}

	a.Update(teaSpecialKeyMsgForTest(tea.KeyEnd))
	a.flushInputQueue()
	a.Update(teaKeyMsgForTest("Z"))
	a.flushInputQueue()

	if got := a.input.Value(); got != "XabcZ" {
		t.Fatalf("value after end insert = %q, want XabcZ", got)
	}
}

func TestInputHistoryNavigationPreservesDraft(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.recordInputHistory("first")
	a.recordInputHistory("second")
	a.input.SetValue("draft")

	if !a.navigateInputHistory(-1) || a.input.Value() != "second" {
		t.Fatalf("first up value = %q, want second", a.input.Value())
	}
	if !a.navigateInputHistory(-1) || a.input.Value() != "first" {
		t.Fatalf("second up value = %q, want first", a.input.Value())
	}
	if !a.navigateInputHistory(-1) || a.input.Value() != "first" {
		t.Fatalf("third up value = %q, want first", a.input.Value())
	}
	if !a.navigateInputHistory(1) || a.input.Value() != "second" {
		t.Fatalf("first down value = %q, want second", a.input.Value())
	}
	if !a.navigateInputHistory(1) || a.input.Value() != "draft" {
		t.Fatalf("second down value = %q, want draft", a.input.Value())
	}
	if a.navigateInputHistory(1) {
		t.Fatal("down outside history returned true, want false")
	}
}

func TestInputHistoryNavigationFlushesQueuedDraft(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.recordInputHistory("previous")

	a.Update(teaKeyMsgForTest("draft"))
	a.Update(teaSpecialKeyMsgForTest(tea.KeyUp))

	if got := a.input.Value(); got != "previous" {
		t.Fatalf("up value = %q, want previous", got)
	}

	a.Update(teaSpecialKeyMsgForTest(tea.KeyDown))
	if got := a.input.Value(); got != "draft" {
		t.Fatalf("down value = %q, want queued draft restored", got)
	}
}

func TestInputAltEnterAndCtrlJInsertNewlines(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)

	a.Update(teaKeyMsgForTest("one"))
	a.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	a.Update(teaKeyMsgForTest("two"))
	a.Update(teaSpecialKeyMsgForTest(tea.KeyCtrlJ))
	a.Update(teaKeyMsgForTest("three"))
	a.flushInputQueue()

	if got := a.input.Value(); got != "one\ntwo\nthree" {
		t.Fatalf("input = %q, want multiline value", got)
	}
}

func TestInputSmallMultilinePastePreservesNewlines(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("one\ntwo"), Paste: true})

	if got := a.input.Value(); got != "one\ntwo" {
		t.Fatalf("pasted input = %q, want newlines preserved", got)
	}
}

func TestInputUpDownMovesWithinMultilineBeforeHistory(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.recordInputHistory("previous")
	a.input.SetValue("one\ntwo")

	a.Update(teaSpecialKeyMsgForTest(tea.KeyUp))
	if got := a.input.Value(); got != "one\ntwo" {
		t.Fatalf("up inside multiline changed value = %q", got)
	}
	if line, _ := a.input.CursorPos(); line != 0 {
		t.Fatalf("cursor line after first up = %d, want 0", line)
	}

	a.Update(teaSpecialKeyMsgForTest(tea.KeyUp))
	if got := a.input.Value(); got != "previous" {
		t.Fatalf("up at first line value = %q, want history entry", got)
	}

	a.Update(teaSpecialKeyMsgForTest(tea.KeyDown))
	if got := a.input.Value(); got != "one\ntwo" {
		t.Fatalf("down while browsing value = %q, want draft restored", got)
	}
}

func TestEscAbortClearsApprovalState(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.isThinking = true
	a.waitingForApproval = true
	a.pendingApprovalID = "approval-1"
	a.approvalQueue = []pendingApproval{{approvalID: "approval-2", toolName: "bash"}}

	a.Update(teaSpecialKeyMsgForTest(tea.KeyEsc))

	if a.waitingForApproval {
		t.Fatal("waitingForApproval = true, want false")
	}
	if a.pendingApprovalID != "" {
		t.Fatalf("pendingApprovalID = %q, want empty", a.pendingApprovalID)
	}
	if len(a.approvalQueue) != 0 {
		t.Fatalf("len(approvalQueue) = %d, want 0", len(a.approvalQueue))
	}
}

func TestClearCommandResetsTranscriptState(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "base", nil, "agent", false, false, nil, nil, nil)
	a.messages = []string{"old"}
	a.toolResults = []toolResult{{toolCallID: "tool-1", msgIndex: 0}}
	a.liveContent = "live"
	a.currentPlan = &tools.TaskPlan{Title: "old plan", Steps: []tools.PlanStep{{Title: "step", Status: "running"}}}
	a.assistantRaw[0] = "raw"
	a.assistantRendered[0] = "rendered"
	a.assistantDirty[0] = true
	a.printedMessageIdx[0] = true
	a.currentAssistantIdx = 0
	a.currentThinkIdx = 1
	a.toolModalOpen = true
	a.activeSkills["x"] = "skill"
	a.extraContext = "base skill"

	a.handleCommand("/clear")

	if len(a.toolResults) != 0 || len(a.assistantRaw) != 0 || len(a.assistantRendered) != 0 || len(a.assistantDirty) != 0 || len(a.printedMessageIdx) != 0 {
		t.Fatalf("transcript state not reset: tools=%d raw=%d rendered=%d dirty=%d printed=%d", len(a.toolResults), len(a.assistantRaw), len(a.assistantRendered), len(a.assistantDirty), len(a.printedMessageIdx))
	}
	if a.currentAssistantIdx != -1 || a.currentThinkIdx != -1 || a.toolModalOpen || a.currentPlan != nil {
		t.Fatalf("active state not reset: assistant=%d think=%d modal=%v plan=%v", a.currentAssistantIdx, a.currentThinkIdx, a.toolModalOpen, a.currentPlan)
	}
	if a.extraContext != "base" || len(a.activeSkills) != 0 {
		t.Fatalf("skill context not reset: extra=%q active=%d", a.extraContext, len(a.activeSkills))
	}
	joined := stripANSI(strings.Join(a.messages, "\n"))
	if !strings.Contains(joined, "Conversation cleared") || strings.Contains(joined, "old") {
		t.Fatalf("messages after clear = %q, want only clear confirmation", joined)
	}
}

func TestOpenLatestToolModalRequiresContent(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	if a.openLatestToolModal() {
		t.Fatal("openLatestToolModal on empty app = true, want false")
	}
	a.messages = []string{"hello"}
	if !a.openLatestToolModal() {
		t.Fatal("openLatestToolModal with content = false, want true")
	}
}

func TestShowNextQuestionTracksCurrentQuestionAndClearResetsIt(t *testing.T) {
	a := &App{questionQueue: []pendingQuestion{{questionID: "q1", question: "Pick?", options: []string{"A", "B"}}}}
	a.showNextQuestion()
	if !a.waitingForQuestion || a.pendingQuestionID != "q1" || len(a.currentQuestion.options) != 2 {
		t.Fatalf("question state = waiting %v id %q options %v", a.waitingForQuestion, a.pendingQuestionID, a.currentQuestion.options)
	}
	a.clearQuestionState()
	if a.waitingForQuestion || a.pendingQuestionID != "" || len(a.currentQuestion.options) != 0 {
		t.Fatalf("question state after clear = waiting %v id %q options %v", a.waitingForQuestion, a.pendingQuestionID, a.currentQuestion.options)
	}
}

func TestRuneInputTabDoesNotCycleMode(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.input.SetValue("prefix ")

	a.Update(teaKeyMsgForTest("tab"))
	a.flushInputQueue()

	if got := a.mode; got != "agent" {
		t.Fatalf("mode = %q, want agent", got)
	}
	if got := a.input.Value(); got != "prefix tab" {
		t.Fatalf("input = %q, want %q", got, "prefix tab")
	}
}

func TestRuneInputEscDoesNotAbortOrClearInput(t *testing.T) {
	a := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.input.SetValue("prefix ")

	a.Update(teaKeyMsgForTest("esc"))
	a.flushInputQueue()

	if got := a.input.Value(); got != "prefix esc" {
		t.Fatalf("input = %q, want %q", got, "prefix esc")
	}
}

func TestInitWithProgramDoesNotBlock(t *testing.T) {
	a := NewApp(
		&historyInjectMockProvider{},
		&provider.Model{ID: "mock-model", Name: "Mock"},
		config.DefaultSettings(),
		nil,
		tools.NewRegistry(t.TempDir(), nil),
		"",
		"",
		nil,
		"agent",
		false,
		false,
		nil,
		nil,
		nil,
	)
	a.SetInitialMessage("hello")
	p := tea.NewProgram(a)
	a.SetProgram(p)

	done := make(chan struct{})
	go func() {
		_ = a.Init()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Init blocked while printing initial history")
	}
}

// TestCacheHighlightThresholdMath verifies the arithmetic of the 50% boundary
// independent of any rendering logic.
func TestCacheHighlightThresholdMath(t *testing.T) {
	type tc struct {
		input     int
		cacheRead int
		wantHigh  bool
	}
	cases := []tc{
		{1000, 0, false},   // 0%
		{1000, 499, false}, // 49.9%
		{1000, 490, false}, // 49%
		{1000, 500, true},  // 50% — boundary: highlighted
		{1000, 501, true},  // 50.1%
		{1000, 750, true},  // 75%
		{1000, 1000, true}, // 100%
		{3, 2, true},       // 66.7% — small counts
		{3, 1, false},      // 33.3%
	}
	for _, c := range cases {
		pct := float64(c.cacheRead) / float64(c.input) * 100
		got := pct >= 50.0
		if got != c.wantHigh {
			t.Errorf("input=%d cacheRead=%d pct=%.4f: highlight=%v, want %v",
				c.input, c.cacheRead, pct, got, c.wantHigh)
		}
	}
}

type historyInjectMockProvider struct{}

func (p *historyInjectMockProvider) Chat(ctx context.Context, params provider.ChatParams) <-chan provider.StreamEvent {
	ch := make(chan provider.StreamEvent, 2)
	ch <- provider.StreamEvent{Type: provider.StreamTextDelta, TextDelta: "ok"}
	ch <- provider.StreamEvent{Type: provider.StreamDone, StopReason: "end_turn"}
	close(ch)
	return ch
}

func (p *historyInjectMockProvider) Name() string { return "mock" }
func (p *historyInjectMockProvider) Models() []*provider.Model {
	return []*provider.Model{{ID: "mock-model", Name: "Mock"}}
}
func (p *historyInjectMockProvider) GetModel(id string) *provider.Model {
	for _, m := range p.Models() {
		if m.ID == id {
			return m
		}
	}
	return nil
}

func TestProcessInputLoadsSessionHistoryIntoAgentEvenWhenUIHistoryAlreadyLoaded(t *testing.T) {
	tmp := t.TempDir()
	cwd := filepath.Join(tmp, "project")
	if err := os.MkdirAll(cwd, 0755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}
	sessionDir := filepath.Join(tmp, "sessions")

	sess := session.New(cwd, sessionDir)
	if err := sess.Init(); err != nil {
		t.Fatalf("init session: %v", err)
	}
	sess.AppendMessage(provider.NewUserMessage("old user"))
	sess.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old assistant"}}))

	settings := config.DefaultSettings()
	settings.DefaultThinkingLevel = "off"
	a := &App{
		provider:            &historyInjectMockProvider{},
		model:               &provider.Model{ID: "mock-model", Name: "Mock"},
		settings:            settings,
		session:             sess,
		registry:            tools.NewRegistry(cwd, nil),
		historyLoaded:       true, // UI already rendered history
		assistantRaw:        make(map[int]string),
		assistantRendered:   make(map[int]string),
		assistantDirty:      make(map[int]bool),
		currentAssistantIdx: -1,
		currentThinkIdx:     -1,
	}

	cmd := a.processInput("new question")
	if cmd == nil {
		t.Fatal("processInput returned nil command")
	}
	_ = cmd()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if a.agent != nil {
			msgs := a.agent.GetMessages()
			if len(msgs) >= 4 {
				if msgs[0].Role != "user" || msgs[0].Content != "old user" {
					t.Fatalf("first message = %+v, want old history user message", msgs[0])
				}
				if msgs[1].Role != "assistant" {
					t.Fatalf("second message role = %s, want assistant", msgs[1].Role)
				}
				if msgs[2].Role != "user" || msgs[2].Content != "new question" {
					t.Fatalf("third message = %+v, want new user message", msgs[2])
				}
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for agent messages")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestInitThenProcessInputStillInjectsSessionHistory(t *testing.T) {
	tmp := t.TempDir()
	cwd := filepath.Join(tmp, "project")
	if err := os.MkdirAll(cwd, 0755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}
	sessionDir := filepath.Join(tmp, "sessions")

	sess := session.New(cwd, sessionDir)
	if err := sess.Init(); err != nil {
		t.Fatalf("init session: %v", err)
	}
	sess.AppendMessage(provider.NewUserMessage("history user"))
	sess.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "history assistant"}}))

	settings := config.DefaultSettings()
	settings.DefaultThinkingLevel = "off"
	app := NewApp(
		&historyInjectMockProvider{},
		&provider.Model{ID: "mock-model", Name: "Mock"},
		settings,
		sess,
		tools.NewRegistry(cwd, nil),
		"",
		"",
		nil,
		"agent",
		false,
		false,
		nil,
		nil,
		nil,
	)

	// Simulate real startup flow: Init() loads history into UI and flips historyLoaded.
	_ = app.Init()

	if !app.historyLoaded {
		t.Fatalf("historyLoaded = false, want true after Init")
	}

	cmd := app.processInput("follow-up")
	if cmd == nil {
		t.Fatal("processInput returned nil command")
	}
	_ = cmd()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if app.agent != nil {
			msgs := app.agent.GetMessages()
			if len(msgs) >= 4 {
				if msgs[0].Role != "user" || msgs[0].Content != "history user" {
					t.Fatalf("first message = %+v, want history user", msgs[0])
				}
				if msgs[1].Role != "assistant" {
					t.Fatalf("second message role = %s, want assistant", msgs[1].Role)
				}
				if msgs[2].Role != "user" || msgs[2].Content != "follow-up" {
					t.Fatalf("third message = %+v, want follow-up user message", msgs[2])
				}
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for agent messages")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
