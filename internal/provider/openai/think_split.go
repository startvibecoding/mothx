package openai

import "strings"

const (
	thinkOpenTag  = "<think>"
	thinkCloseTag = "</think>"
)

// thinkSplitter is a streaming state machine that separates reasoning content
// wrapped in <think>...</think> tags from regular text content. Some
// OpenAI-compatible models inline their reasoning in the content field using
// these tags instead of providing a separate reasoning_content field.
//
// It handles tags split across multiple stream chunks by holding back a
// partial-tag suffix until enough characters arrive to disambiguate.
type thinkSplitter struct {
	inThink bool
	// pending holds a trailing fragment that may be the start of a tag and
	// cannot yet be classified as text/think output.
	pending string
}

// push feeds the next content delta and returns any text and thinking output
// that can be emitted so far. Either return value may be empty.
func (s *thinkSplitter) push(delta string) (text string, think string) {
	buf := s.pending + delta
	s.pending = ""

	var textOut, thinkOut strings.Builder

	for len(buf) > 0 {
		if s.inThink {
			idx := strings.Index(buf, thinkCloseTag)
			if idx < 0 {
				// No close tag yet. Emit everything except a possible partial
				// close tag at the end.
				safe, hold := splitPartialSuffix(buf, thinkCloseTag)
				thinkOut.WriteString(safe)
				s.pending = hold
				buf = ""
				continue
			}
			thinkOut.WriteString(buf[:idx])
			buf = buf[idx+len(thinkCloseTag):]
			s.inThink = false
		} else {
			idx := strings.Index(buf, thinkOpenTag)
			if idx < 0 {
				safe, hold := splitPartialSuffix(buf, thinkOpenTag)
				textOut.WriteString(safe)
				s.pending = hold
				buf = ""
				continue
			}
			textOut.WriteString(buf[:idx])
			buf = buf[idx+len(thinkOpenTag):]
			s.inThink = true
		}
	}

	return textOut.String(), thinkOut.String()
}

// flush returns any buffered content remaining after the stream ends. A
// partial tag fragment is treated as literal output in whatever mode is
// active.
func (s *thinkSplitter) flush() (text string, think string) {
	if s.pending == "" {
		return "", ""
	}
	rest := s.pending
	s.pending = ""
	if s.inThink {
		return "", rest
	}
	return rest, ""
}

// splitPartialSuffix returns the portion of s that can be safely emitted now
// and the trailing fragment that might be the beginning of tag. The held
// fragment is the longest suffix of s that is a proper prefix of tag.
func splitPartialSuffix(s, tag string) (safe, hold string) {
	maxHold := len(tag) - 1
	if maxHold > len(s) {
		maxHold = len(s)
	}
	for n := maxHold; n > 0; n-- {
		if strings.HasPrefix(tag, s[len(s)-n:]) {
			return s[:len(s)-n], s[len(s)-n:]
		}
	}
	return s, ""
}
