package browser

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/startvibecoding/vibecoding/internal/tools"
)

func TestEnsureProjectSkillCreatesBrowserSkill(t *testing.T) {
	root := t.TempDir()

	path, created, err := EnsureProjectSkill(root)
	if err != nil {
		t.Fatalf("EnsureProjectSkill() error = %v", err)
	}
	if !created {
		t.Fatal("expected skill to be created")
	}
	wantPath := filepath.Join(root, ".skills", SkillName, "SKILL.md")
	if path != wantPath {
		t.Fatalf("path = %q, want %q", path, wantPath)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"# Vibe Browser",
		"`browser` tool",
		"`snapshot`",
		"`screenshot`",
		"Never claim a UI state changed until you verify it",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("skill content missing %q", want)
		}
	}
}

func TestEnsureProjectSkillDoesNotOverwriteExistingSkill(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".skills", SkillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(path, []byte("custom browser skill"), 0644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	gotPath, created, err := EnsureProjectSkill(root)
	if err != nil {
		t.Fatalf("EnsureProjectSkill() error = %v", err)
	}
	if created {
		t.Fatal("did not expect existing skill to be recreated")
	}
	if gotPath != path {
		t.Fatalf("path = %q, want %q", gotPath, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read custom skill: %v", err)
	}
	if string(data) != "custom browser skill" {
		t.Fatalf("skill was overwritten: %q", string(data))
	}
}

func TestEnsureProjectSkillRespectsLowercaseSkill(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".skills", SkillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(skillDir, "skill.md")
	if err := os.WriteFile(path, []byte("lowercase browser skill"), 0644); err != nil {
		t.Fatalf("write lowercase skill: %v", err)
	}

	gotPath, created, err := EnsureProjectSkill(root)
	if err != nil {
		t.Fatalf("EnsureProjectSkill() error = %v", err)
	}
	if created {
		t.Fatal("did not expect lowercase skill to be recreated")
	}
	if gotPath != path {
		t.Fatalf("path = %q, want %q", gotPath, path)
	}
}

func TestRegisterAndRemoveBrowserTool(t *testing.T) {
	registry := tools.NewRegistry(t.TempDir(), nil)

	RegisterTool(registry)
	if !IsToolRegistered(registry) {
		t.Fatal("expected browser tool to be registered")
	}

	RemoveTool(registry)
	if IsToolRegistered(registry) {
		t.Fatal("expected browser tool to be removed")
	}
}

func TestScreenshotToolResultProcessesImage(t *testing.T) {
	registry := tools.NewRegistry(t.TempDir(), nil)
	tool := NewTool(registry)

	result, err := tool.screenshotToolResult(testPNG(t, 200, 100), map[string]any{
		"maxLongEdge": float64(50),
	})
	if err != nil {
		t.Fatalf("screenshotToolResult() error = %v", err)
	}
	if len(result.Contents) != 2 || result.Contents[1].Image == nil {
		t.Fatalf("contents = %#v, want text + image", result.Contents)
	}
	image := result.Contents[1].Image
	if image.Width != 50 || image.Height != 25 {
		t.Fatalf("image size = %dx%d, want 50x25", image.Width, image.Height)
	}
	if image.OriginalWidth != 200 || image.OriginalHeight != 100 {
		t.Fatalf("original size = %dx%d, want 200x100", image.OriginalWidth, image.OriginalHeight)
	}
	if image.Detail != "detail" {
		t.Fatalf("detail = %q, want detail", image.Detail)
	}
	if !strings.Contains(result.Text, "Browser screenshot") || !strings.Contains(result.Text, "original: 200x100") {
		t.Fatalf("description = %q, want screenshot resize details", result.Text)
	}
}

func TestClientOptionsDefaultLaunchViewport(t *testing.T) {
	opts := clientOptions(map[string]any{})
	if opts.Launch == nil {
		t.Fatal("Launch options are nil")
	}
	if opts.Launch.ViewportWidth != defaultViewportWidth {
		t.Fatalf("ViewportWidth = %d, want %d", opts.Launch.ViewportWidth, defaultViewportWidth)
	}
	if opts.Launch.ViewportHeight != defaultViewportHeight {
		t.Fatalf("ViewportHeight = %d, want %d", opts.Launch.ViewportHeight, defaultViewportHeight)
	}
	if !opts.Launch.Headless {
		t.Fatal("default launch should remain headless")
	}
}

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buf.Bytes()
}

func TestClientOptionsAllowsViewportAndHeadlessOverride(t *testing.T) {
	opts := clientOptions(map[string]any{
		"viewportWidth":  float64(1366),
		"viewportHeight": float64(768),
		"headless":       false,
	})
	if opts.Launch == nil {
		t.Fatal("Launch options are nil")
	}
	if opts.Launch.ViewportWidth != 1366 {
		t.Fatalf("ViewportWidth = %d, want 1366", opts.Launch.ViewportWidth)
	}
	if opts.Launch.ViewportHeight != 768 {
		t.Fatalf("ViewportHeight = %d, want 768", opts.Launch.ViewportHeight)
	}
	if opts.Launch.Headless {
		t.Fatal("headless override was not honored")
	}
}
