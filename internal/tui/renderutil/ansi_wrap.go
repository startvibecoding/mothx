package renderutil

import (
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
)

const pathBreakpoints = "/"

// VisibleWidth returns the terminal cell width of s after normalizing tabs.
// ANSI escape sequences are zero-width.
func VisibleWidth(s string) int {
	if s == "" {
		return 0
	}
	return xansi.StringWidth(normalizeTabs(s))
}

// WrapANSI wraps styled text to width cells using Charmbracelet's ANSI-aware
// wrapper. It preserves escape sequences and accounts for grapheme and CJK
// widths without maintaining a local ANSI parser.
func WrapANSI(text string, width int) string {
	return wrapWith(text, width, func(line string) string {
		return xansi.Wrap(line, width, pathBreakpoints)
	})
}

// WrapPlainText wraps raw model text to width cells without applying Markdown
// or path-specific token handling.
func WrapPlainText(text string, width int) string {
	return wrapWith(text, width, func(line string) string {
		return xansi.Hardwrap(line, width, false)
	})
}

func wrapWith(text string, width int, wrapLine func(string) string) string {
	if width <= 0 || text == "" {
		return text
	}
	inputLines := strings.Split(normalizeTabs(text), "\n")
	wrapped := make([]string, 0, len(inputLines))
	for _, line := range inputLines {
		line = trimRightVisibleASCIIWhitespace(line)
		if isANSIBlankLine(line) {
			wrapped = append(wrapped, "")
			continue
		}
		for _, out := range strings.Split(wrapLine(line), "\n") {
			out = trimRightVisibleASCIIWhitespace(out)
			if !isANSIBlankLine(out) {
				wrapped = append(wrapped, out)
			}
		}
	}
	return strings.Join(wrapped, "\n")
}

func normalizeTabs(s string) string {
	return strings.ReplaceAll(s, "\t", "   ")
}

func trimRightVisibleASCIIWhitespace(s string) string {
	plain := xansi.Strip(s)
	trimmed := strings.TrimRight(plain, " \t")
	if len(trimmed) == len(plain) {
		return s
	}
	return xansi.Truncate(s, xansi.StringWidth(trimmed), "")
}

func isANSIBlankLine(s string) bool {
	return strings.TrimSpace(xansi.Strip(s)) == ""
}
