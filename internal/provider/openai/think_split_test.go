package openai

import "testing"

func TestThinkSplitterSingleChunk(t *testing.T) {
	s := &thinkSplitter{}
	text, think := s.push("<think>reasoning here</think>visible answer")
	if think != "reasoning here" {
		t.Fatalf("think = %q, want %q", think, "reasoning here")
	}
	if text != "visible answer" {
		t.Fatalf("text = %q, want %q", text, "visible answer")
	}
	ft, fk := s.flush()
	if ft != "" || fk != "" {
		t.Fatalf("flush = (%q, %q), want empty", ft, fk)
	}
}

func TestThinkSplitterPlainText(t *testing.T) {
	s := &thinkSplitter{}
	text, think := s.push("just plain text")
	if think != "" {
		t.Fatalf("think = %q, want empty", think)
	}
	if text != "just plain text" {
		t.Fatalf("text = %q, want %q", text, "just plain text")
	}
}

func TestThinkSplitterTagSplitAcrossChunks(t *testing.T) {
	s := &thinkSplitter{}
	var text, think string
	chunks := []string{"<thi", "nk>think", "ing</th", "ink>ans", "wer"}
	for _, c := range chunks {
		tx, tk := s.push(c)
		text += tx
		think += tk
	}
	ft, fk := s.flush()
	text += ft
	think += fk
	if think != "thinking" {
		t.Fatalf("think = %q, want %q", think, "thinking")
	}
	if text != "answer" {
		t.Fatalf("text = %q, want %q", text, "answer")
	}
}

func TestThinkSplitterTextBeforeThink(t *testing.T) {
	s := &thinkSplitter{}
	text, think := s.push("hello <think>secret</think> world")
	if think != "secret" {
		t.Fatalf("think = %q, want %q", think, "secret")
	}
	if text != "hello  world" {
		t.Fatalf("text = %q, want %q", text, "hello  world")
	}
}

func TestThinkSplitterUnclosedThink(t *testing.T) {
	s := &thinkSplitter{}
	text, think := s.push("<think>still thinking")
	if text != "" {
		t.Fatalf("text = %q, want empty", text)
	}
	if think != "still thinking" {
		t.Fatalf("think = %q, want %q", think, "still thinking")
	}
	ft, fk := s.flush()
	if ft != "" || fk != "" {
		t.Fatalf("flush = (%q, %q), want empty", ft, fk)
	}
}

func TestThinkSplitterPartialFalseAlarm(t *testing.T) {
	// A "<" that turns out not to be a tag must be emitted as text.
	s := &thinkSplitter{}
	var text string
	for _, c := range []string{"a <", "b c"} {
		tx, _ := s.push(c)
		text += tx
	}
	ft, _ := s.flush()
	text += ft
	if text != "a <b c" {
		t.Fatalf("text = %q, want %q", text, "a <b c")
	}
}

func TestThinkSplitterFlushPartialTag(t *testing.T) {
	// A dangling partial tag at end of stream is emitted literally.
	s := &thinkSplitter{}
	text, _ := s.push("done <thi")
	if text != "done " {
		t.Fatalf("text = %q, want %q", text, "done ")
	}
	ft, _ := s.flush()
	if ft != "<thi" {
		t.Fatalf("flush text = %q, want %q", ft, "<thi")
	}
}
