package renderutil

import "strings"

const (
	minMarkdownStyleWrap = 80
	maxMarkdownStyleWrap = 160
)

// MarkdownStyleWrapWidth keeps Glamour from wrapping at tiny viewport widths
// while avoiding huge padded intermediate strings. Final viewport wrapping is
// still handled by WrapANSI.
func MarkdownStyleWrapWidth(contentWidth int) int {
	if contentWidth < minMarkdownStyleWrap {
		return minMarkdownStyleWrap
	}
	if contentWidth > maxMarkdownStyleWrap {
		return maxMarkdownStyleWrap
	}
	return contentWidth
}

// TrimANSIBlankLines removes leading and trailing lines that are visually
// blank after ANSI escape sequences are ignored.
func TrimANSIBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	start := 0
	end := len(lines)
	for start < end && isANSIBlankLine(lines[start]) {
		start++
	}
	for end > start && isANSIBlankLine(lines[end-1]) {
		end--
	}
	return strings.Join(lines[start:end], "\n")
}

// LooksLikeMarkdown reports whether s contains Markdown that benefits from a
// terminal renderer. Plain prose stays on the text wrapping path so long URLs
// and ordinary sentences are not reformatted unnecessarily.
func LooksLikeMarkdown(s string) bool {
	if strings.Contains(s, "```") || strings.Contains(s, "~~~") || strings.Contains(s, "`") {
		return true
	}
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case isMarkdownHeading(trimmed):
			return true
		case strings.HasPrefix(trimmed, "> "):
			return true
		case strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ "):
			return true
		case isOrderedMarkdownList(trimmed):
			return true
		case strings.HasPrefix(trimmed, "|") && strings.Count(trimmed, "|") >= 2:
			return true
		case strings.Contains(trimmed, "**") || strings.Contains(trimmed, "__"):
			return true
		}
	}
	return false
}

func isMarkdownHeading(s string) bool {
	i := 0
	for i < len(s) && s[i] == '#' {
		i++
	}
	return i > 0 && i <= 6 && i < len(s) && isMarkdownSpace(s[i])
}

func isOrderedMarkdownList(s string) bool {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i > 0 && i+1 < len(s) && s[i] == '.' && isMarkdownSpace(s[i+1])
}

func isMarkdownSpace(b byte) bool {
	return b == ' ' || b == '\t'
}
