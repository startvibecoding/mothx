package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/startvibecoding/mothx/internal/platform"
)

const (
	pastedImageMaxBytes = 20 << 20
	pastedImageMaxAge   = 7 * 24 * time.Hour
)

type ClipboardImageSaver interface {
	SaveImage(ctx context.Context, projectDir string) (path string, ok bool, err error)
}

type FileOpener interface {
	Open(path string) error
}

type systemClipboardImageSaver struct {
	now func() time.Time
}

type systemFileOpener struct{}

func newSystemClipboardImageSaver() ClipboardImageSaver {
	return systemClipboardImageSaver{now: time.Now}
}

func (systemFileOpener) Open(path string) error {
	return platform.OpenFile(path)
}

func (a *App) handlePasteImageCommand() {
	if a.clipboardImageSaver == nil {
		a.clipboardImageSaver = newSystemClipboardImageSaver()
	}
	path, ok, err := a.clipboardImageSaver.SaveImage(context.Background(), a.currentCwd())
	if err != nil {
		a.addCommandError(fmt.Sprintf("Paste image failed: %v", err))
		return
	}
	if !ok {
		a.addCommandStatus("Clipboard does not contain a PNG image. Copy an image, then run /paste-image again.")
		return
	}
	displayPath := pastedImageDisplayPath(a.currentCwd(), path)
	a.input = a.input.InsertString("Image Path : " + displayPath)
	a.lastPastedImagePath = path
	a.updateCommandSuggestions()
	a.scheduleRender()
	a.addCommandStatus(fmt.Sprintf("Image pasted: %s", displayPath), "Press Ctrl+R to preview.")
}

func (a *App) previewLastPastedImage() tea.Cmd {
	if a.lastPastedImagePath == "" {
		a.addCommandStatus("No pasted image to preview. Run /paste-image first.")
		return nil
	}
	if a.fileOpener == nil {
		a.fileOpener = systemFileOpener{}
	}
	if err := a.fileOpener.Open(a.lastPastedImagePath); err != nil {
		a.addCommandError(fmt.Sprintf("Could not open pasted image: %v. Path: %s", err, a.lastPastedImagePath))
		return nil
	}
	a.addCommandStatus(fmt.Sprintf("Opened preview: %s", a.lastPastedImagePath))
	return nil
}

func (s systemClipboardImageSaver) SaveImage(ctx context.Context, projectDir string) (string, bool, error) {
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	dir := pastedImageDir(projectDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", false, fmt.Errorf("create paste cache: %w", err)
	}
	cleanupOldPastedImages(dir, now)

	path := filepath.Join(dir, fmt.Sprintf("paste-%s.png", now.Format("20060102-150405.000000000")))
	ok, err := saveClipboardPNG(ctx, path)
	if err != nil || !ok {
		_ = os.Remove(path)
		return "", ok, err
	}
	info, err := os.Stat(path)
	if err != nil {
		_ = os.Remove(path)
		return "", false, fmt.Errorf("stat pasted image: %w", err)
	}
	if info.Size() == 0 {
		_ = os.Remove(path)
		return "", false, nil
	}
	if info.Size() > pastedImageMaxBytes {
		_ = os.Remove(path)
		return "", false, fmt.Errorf("pasted image too large: %d bytes (max %d)", info.Size(), pastedImageMaxBytes)
	}
	return path, true, nil
}

func pastedImageDir(projectDir string) string {
	if projectDir == "" {
		projectDir = "."
	}
	return filepath.Join(projectDir, ".vibe", "tmp")
}

func pastedImageDisplayPath(projectDir string, path string) string {
	if rel, err := filepath.Rel(projectDir, path); err == nil && rel != "." && !filepath.IsAbs(rel) {
		return filepath.ToSlash(rel)
	}
	return path
}

func cleanupOldPastedImages(dir string, now time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := now.Add(-pastedImageMaxAge)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".png" || len(name) < len("paste-.png") || name[:6] != "paste-" {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
	}
}

func saveClipboardPNG(ctx context.Context, target string) (bool, error) {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("pngpaste"); err != nil {
			return false, fmt.Errorf("pngpaste not found; install pngpaste or enter an image path manually")
		}
		err := exec.CommandContext(ctx, "pngpaste", target).Run()
		if err != nil {
			return false, nil
		}
		return true, nil
	case "windows":
		return saveWindowsClipboardPNG(ctx, target)
	default:
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			if _, err := exec.LookPath("wl-paste"); err == nil {
				ok, err := saveClipboardCommandOutput(ctx, target, "wl-paste", "--type", "image/png")
				if ok || err != nil {
					return ok, err
				}
			}
		}
		if _, err := exec.LookPath("xclip"); err == nil {
			return saveClipboardCommandOutput(ctx, target, "xclip", "-selection", "clipboard", "-t", "image/png", "-o")
		}
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			return false, fmt.Errorf("wl-paste or xclip not found; install wl-clipboard or xclip, or enter an image path manually")
		}
		return false, fmt.Errorf("xclip not found; install xclip or enter an image path manually")
	}
}

func saveClipboardCommandOutput(ctx context.Context, target string, name string, args ...string) (bool, error) {
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return false, fmt.Errorf("create pasted image: %w", err)
	}
	defer f.Close()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = f
	if err := cmd.Run(); err != nil {
		_ = os.Remove(target)
		return false, nil
	}
	return true, nil
}

func saveWindowsClipboardPNG(ctx context.Context, target string) (bool, error) {
	powershell, err := exec.LookPath("powershell.exe")
	if err != nil {
		powershell, err = exec.LookPath("powershell")
	}
	if err != nil {
		return false, fmt.Errorf("PowerShell not found; enter an image path manually")
	}
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$image = [System.Windows.Forms.Clipboard]::GetImage()
if ($null -eq $image) { exit 2 }
$image.Save(%q, [System.Drawing.Imaging.ImageFormat]::Png)
`, target)
	cmd := exec.CommandContext(ctx, powershell, "-NoProfile", "-NonInteractive", "-Command", script)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 2 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
