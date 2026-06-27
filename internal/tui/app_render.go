package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/startvibecoding/GoStreamingMarkdown/gsm"
	"github.com/startvibecoding/vibecoding/internal/tui/renderutil"
)

func (a *App) updateViewportContent() {
	a.updateViewportContentWithFollow(false)
}

func (a *App) updateViewportContentWithFollow(_ bool) {
	if a.program != nil {
		a.liveContent = ""
		return
	}
	content := a.renderTranscriptContent()
	a.liveContent = content
}

func (a *App) renderTranscriptContent() string {
	count := len(a.messages)
	if a.currentThinkIdx >= count {
		count = a.currentThinkIdx + 1
	}
	if a.currentAssistantIdx >= count {
		count = a.currentAssistantIdx + 1
	}
	blocks := make([]string, 0, count)
	for idx := 0; idx < count; idx++ {
		rendered := strings.TrimRight(a.renderMessageAt(idx), "\n")
		if strings.TrimSpace(rendered) == "" {
			continue
		}
		blocks = append(blocks, rendered)
	}
	return strings.Join(blocks, "\n\n")
}

func (a *App) renderLiveTranscriptContent() string {
	if a.program == nil {
		return a.liveContent
	}

	count := len(a.messages)
	if a.currentThinkIdx >= count {
		count = a.currentThinkIdx + 1
	}
	if a.currentAssistantIdx >= count {
		count = a.currentAssistantIdx + 1
	}
	blocks := make([]string, 0, 2)
	for idx := 0; idx < count; idx++ {
		if a.printedMessageIdx[idx] {
			continue
		}
		isCurrentApproval := a.waitingForApproval && a.currentApprovalIdx >= 0 && idx == a.currentApprovalIdx
		if idx != a.currentThinkIdx && idx != a.currentAssistantIdx && !isCurrentApproval {
			continue
		}
		rendered := strings.TrimRight(a.renderMessageAt(idx), "\n")
		if strings.TrimSpace(rendered) == "" {
			continue
		}
		blocks = append(blocks, rendered)
	}
	return strings.Join(blocks, "\n\n")
}

func (a *App) configureMarkdownRenderer() {
	width := renderutil.MarkdownStyleWrapWidth(a.assistantMarkdownWidth())
	a.mdRenderer = gsm.NewStream(width, nil)
}

func (a *App) assistantMarkdownWidth() int {
	width := a.width
	if width <= 0 {
		width = 80
	}
	width -= lipgloss.Width("Assistant: ")
	if width < 1 {
		return 1
	}
	return width
}

func (a *App) renderFixedHeight(view string) string {
	if a.height <= 0 {
		return view
	}
	view = strings.TrimRight(view, "\n")
	lines := strings.Split(view, "\n")
	if len(lines) > a.height {
		lines = lines[len(lines)-a.height:]
	}
	for len(lines) < a.height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

