package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDebugJSONWritesRequestAndCompleteResponse(t *testing.T) {
	t.Setenv("VIBECODING_DEBUG", "1")
	workDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	DebugJSON("OpenAI request JSON", []byte(`{"model":"test","stream":true}`))
	DebugCompleteResponse(DebugResponse{
		Provider: "openai",
		API:      "chat-completions",
		Content:  "complete response",
	})

	data, err := os.ReadFile(filepath.Join(workDir, "debug.log"))
	if err != nil {
		t.Fatal(err)
	}
	log := string(data)
	if !strings.Contains(log, `OpenAI request JSON: {"model":"test","stream":true}`) {
		t.Fatalf("debug log missing request JSON: %s", log)
	}
	if !strings.Contains(log, `Response JSON: {"provider":"openai","api":"chat-completions","content":"complete response"}`) {
		t.Fatalf("debug log missing complete response JSON: %s", log)
	}
	if strings.Contains(log, "data:") {
		t.Fatalf("debug log contains an SSE fragment: %s", log)
	}
}
