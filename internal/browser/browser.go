package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	vbclient "github.com/startvibecoding/vibe-browser/pkg/client"
	vbprotocol "github.com/startvibecoding/vibe-browser/pkg/protocol"

	"github.com/startvibecoding/mothx/internal/imageproc"
	"github.com/startvibecoding/mothx/internal/provider"
	"github.com/startvibecoding/mothx/internal/tools"
)

const (
	ToolName              = "browser"
	SkillName             = "vibe-browser"
	defaultViewportWidth  = 1920
	defaultViewportHeight = 1080
)

const defaultSkillContent = `# Vibe Browser

Use this skill when the user asks to inspect, test, automate, or capture a web page with the browser tool.

The ` + "`browser`" + ` tool exposes browser automation through an action field. Prefer this loop:

1. ` + "`open`" + ` or ` + "`navigate`" + ` to the page.
2. ` + "`snapshot`" + ` with ` + "`interactive=true`" + ` to inspect controls and stable refs/selectors.
3. Interact with ` + "`click`" + `, ` + "`fill`" + `, ` + "`type`" + `, ` + "`press`" + `, ` + "`select`" + `, ` + "`check`" + `, ` + "`uncheck`" + `, ` + "`scroll`" + `.
4. After page-changing actions, wait with ` + "`wait_for_selector`" + `, ` + "`wait_for_text`" + `, ` + "`wait_for_url`" + `, or a short ` + "`wait_ms`" + `.
5. Re-run ` + "`snapshot`" + ` or read with ` + "`get_text`" + `, ` + "`get_html`" + `, ` + "`get_attr`" + `, ` + "`get_url`" + `, ` + "`get_title`" + `.
6. Use ` + "`screenshot`" + ` for visual verification; pass ` + "`outputPath`" + ` to save under the project.

Common actions:

- Navigation: ` + "`open`" + `, ` + "`navigate`" + `, ` + "`back`" + `, ` + "`forward`" + `, ` + "`reload`" + `, ` + "`close`" + `.
- Inspection: ` + "`snapshot`" + `, ` + "`get_text`" + `, ` + "`get_html`" + `, ` + "`get_value`" + `, ` + "`get_attr`" + `, ` + "`get_url`" + `, ` + "`get_title`" + `, ` + "`eval`" + `.
- State checks: ` + "`is_visible`" + `, ` + "`is_enabled`" + `, ` + "`is_checked`" + `.
- Waiting: ` + "`wait_ms`" + `, ` + "`wait_for_selector`" + `, ` + "`wait_for_text`" + `, ` + "`wait_for_url`" + `.
- Browser state: ` + "`set_viewport`" + `, ` + "`set_geolocation`" + `, ` + "`set_offline`" + `, ` + "`set_headers`" + `, ` + "`cookies_get`" + `, ` + "`cookies_clear`" + `, ` + "`tab_new`" + `, ` + "`tab_close`" + `.

Keep selectors specific and prefer refs/selectors observed in a fresh snapshot. Never claim a UI state changed until you verify it with a snapshot, read, URL/title check, or screenshot.
`

// EnsureProjectSkill creates the project-local browser skill if it does not
// already exist. Existing SKILL.md or skill.md files are never overwritten so
// user customizations keep priority.
func EnsureProjectSkill(projectRoot string) (path string, created bool, err error) {
	if projectRoot == "" {
		return "", false, fmt.Errorf("project root is required")
	}
	skillDir := filepath.Join(projectRoot, ".skills", SkillName)
	upperPath := filepath.Join(skillDir, "SKILL.md")
	lowerPath := filepath.Join(skillDir, "skill.md")

	if _, err := os.Stat(upperPath); err == nil {
		return upperPath, false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", false, err
	}
	if _, err := os.Stat(lowerPath); err == nil {
		return lowerPath, false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", false, err
	}
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return "", false, err
	}
	f, err := os.OpenFile(upperPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return upperPath, false, nil
		}
		return "", false, err
	}
	if _, err := f.WriteString(defaultSkillContent); err != nil {
		_ = f.Close()
		return "", false, err
	}
	if err := f.Close(); err != nil {
		return "", false, err
	}
	return upperPath, true, nil
}

func RegisterTool(registry *tools.Registry) {
	if registry == nil {
		return
	}
	registry.Register(NewTool(registry))
}

func RemoveTool(registry *tools.Registry) {
	if registry == nil {
		return
	}
	registry.Remove(ToolName)
}

func IsToolRegistered(registry *tools.Registry) bool {
	if registry == nil {
		return false
	}
	_, ok := registry.Get(ToolName)
	return ok
}

type Tool struct {
	registry *tools.Registry
	mu       sync.Mutex
	client   *vbclient.Client
}

func NewTool(registry *tools.Registry) *Tool {
	return &Tool{registry: registry}
}

func (t *Tool) Name() string { return ToolName }

func (t *Tool) Description() string {
	return "Control a Chromium-family browser through the vibe-browser SDK. Use action=open/navigate/snapshot/click/fill/type/press/screenshot/etc."
}

func (t *Tool) PromptSnippet() string {
	return "Control a browser through vibe-browser when browser support is enabled"
}

func (t *Tool) PromptGuidelines() []string {
	return []string{
		"Use browser snapshot before interacting so selectors/refs are grounded in the current page",
		"After click/fill/press/navigation, wait for a selector/text/url or take another snapshot before reporting success",
		"Use screenshot with outputPath for visual verification artifacts",
	}
}

func (t *Tool) Parameters() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "action": {"type": "string", "description": "Browser action: open, navigate, back, forward, reload, snapshot, click, dblclick, hover, focus, fill, type, press, scroll, check, uncheck, select, get_text, get_html, get_value, get_attr, get_url, get_title, is_visible, is_enabled, is_checked, eval, wait_ms, wait_for_selector, wait_for_text, wait_for_url, screenshot, set_viewport, set_geolocation, set_offline, set_headers, cookies_get, cookies_clear, tab_new, tab_close, close"},
    "url": {"type": "string"},
    "selector": {"type": "string", "description": "CSS selector or ref from snapshot"},
    "value": {"type": "string"},
    "text": {"type": "string"},
    "key": {"type": "string"},
    "attr": {"type": "string"},
    "expression": {"type": "string"},
    "outputPath": {"type": "string", "description": "Project-relative path for screenshot output"},
    "format": {"type": "string", "enum": ["png", "jpeg", "webp"]},
    "quality": {"type": "integer"},
    "imageMode": {"type": "string", "enum": ["auto", "fast", "detail", "raw"], "description": "Image processing mode for returned screenshots. Defaults to detail."},
    "maxLongEdge": {"type": "integer", "description": "Optional maximum long edge in pixels for returned screenshot resizing"},
    "fullPage": {"type": "boolean"},
    "interactive": {"type": "boolean"},
    "compact": {"type": "boolean"},
    "depth": {"type": "integer"},
    "urls": {"type": "boolean"},
    "width": {"type": "integer"},
    "height": {"type": "integer"},
    "viewportWidth": {"type": "integer", "description": "Initial browser viewport width. Defaults to 1920."},
    "viewportHeight": {"type": "integer", "description": "Initial browser viewport height. Defaults to 1080."},
    "ms": {"type": "integer"},
    "deltaX": {"type": "number"},
    "deltaY": {"type": "number"},
    "latitude": {"type": "number"},
    "longitude": {"type": "number"},
    "accuracy": {"type": "number"},
    "offline": {"type": "boolean"},
    "headers": {"type": "object", "additionalProperties": {"type": "string"}},
    "targetId": {"type": "string"},
    "headless": {"type": "boolean"},
    "browser": {"type": "string", "enum": ["chrome", "chromium", "brave", "edge", "chrome-canary"]},
    "cdpUrl": {"type": "string"},
    "executablePath": {"type": "string"},
    "daemon": {"type": "boolean"},
    "session": {"type": "string"}
  },
  "required": ["action"]
}`)
}

func (t *Tool) ExecutionTimeout(params map[string]any) (time.Duration, bool) {
	return 2 * time.Minute, true
}

func (t *Tool) Execute(ctx context.Context, params map[string]any) (result tools.ToolResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = tools.ToolResult{}
			err = fmt.Errorf("%v", recovered)
		}
	}()

	t.mu.Lock()
	defer t.mu.Unlock()

	action := strings.ToLower(strings.TrimSpace(stringParam(params, "action")))
	if action == "" {
		return tools.ToolResult{}, fmt.Errorf("action is required")
	}
	if action == "close" {
		if t.client != nil {
			_ = t.client.Close()
			t.client = nil
		}
		return tools.NewTextToolResult("browser closed"), nil
	}

	c, err := t.ensureClient(ctx, params)
	if err != nil {
		return tools.ToolResult{}, err
	}

	switch action {
	case "open":
		if url := stringParam(params, "url"); url != "" {
			if err := c.Navigate(ctx, url); err != nil {
				return tools.ToolResult{}, err
			}
		}
		return t.pageSummary(ctx, c, "browser opened")
	case "navigate":
		url := requireString(params, "url")
		if err := c.Navigate(ctx, url); err != nil {
			return tools.ToolResult{}, err
		}
		return t.pageSummary(ctx, c, "navigated")
	case "back":
		return textErr("went back", c.Back(ctx))
	case "forward":
		return textErr("went forward", c.Forward(ctx))
	case "reload":
		return textErr("reloaded", c.Reload(ctx))
	case "snapshot":
		s, err := c.SnapshotWithOptions(ctx, &vbprotocol.SnapshotOptions{
			Selector:    stringParam(params, "selector"),
			Interactive: boolParam(params, "interactive"),
			Compact:     boolParam(params, "compact"),
			Depth:       intParam(params, "depth"),
			URLs:        boolParam(params, "urls"),
		})
		return tools.NewTextToolResult(s), err
	case "click":
		return textErr("clicked", c.Click(ctx, requireString(params, "selector")))
	case "dblclick":
		return textErr("double-clicked", c.DoubleClick(ctx, requireString(params, "selector")))
	case "hover":
		return textErr("hovered", c.Hover(ctx, requireString(params, "selector")))
	case "focus":
		return textErr("focused", c.Focus(ctx, requireString(params, "selector")))
	case "fill":
		return textErr("filled", c.Fill(ctx, requireString(params, "selector"), requireString(params, "value")))
	case "type":
		return textErr("typed", c.Type(ctx, requireString(params, "selector"), requireString(params, "text")))
	case "press":
		return textErr("pressed", c.Press(ctx, requireString(params, "key")))
	case "scroll":
		return textErr("scrolled", c.Scroll(ctx, floatParam(params, "deltaX"), floatParam(params, "deltaY")))
	case "check":
		return textErr("checked", c.Check(ctx, requireString(params, "selector")))
	case "uncheck":
		return textErr("unchecked", c.Uncheck(ctx, requireString(params, "selector")))
	case "select":
		return textErr("selected", c.Select(ctx, requireString(params, "selector"), requireString(params, "value")))
	case "get_text":
		return stringValue(ctx, c.GetText, requireString(params, "selector"))
	case "get_html":
		return stringValue(ctx, c.GetHTML, requireString(params, "selector"))
	case "get_value":
		return stringValue(ctx, c.GetValue, requireString(params, "selector"))
	case "get_attr":
		return stringPairValue(ctx, c.GetAttr, requireString(params, "selector"), requireString(params, "attr"))
	case "get_url":
		return valueResult(c.URL(ctx))
	case "get_title":
		return valueResult(c.Title(ctx))
	case "is_visible":
		return valueResult(c.IsVisible(ctx, requireString(params, "selector")))
	case "is_enabled":
		return valueResult(c.IsEnabled(ctx, requireString(params, "selector")))
	case "is_checked":
		return valueResult(c.IsChecked(ctx, requireString(params, "selector")))
	case "eval":
		return valueResult(c.Eval(ctx, requireString(params, "expression")))
	case "wait_ms":
		return textErr("waited", c.WaitMS(ctx, intParam(params, "ms")))
	case "wait_for_selector":
		return textErr("selector appeared", c.WaitForSelector(ctx, requireString(params, "selector")))
	case "wait_for_text":
		return textErr("text appeared", c.WaitForText(ctx, requireString(params, "text")))
	case "wait_for_url":
		return textErr("url matched", c.WaitForURL(ctx, requireString(params, "url")))
	case "screenshot":
		return t.screenshot(ctx, c, params)
	case "set_viewport":
		return textErr("viewport set", c.SetViewport(ctx, intParam(params, "width"), intParam(params, "height")))
	case "set_geolocation":
		return textErr("geolocation set", c.SetGeolocation(ctx, floatParam(params, "latitude"), floatParam(params, "longitude"), floatParam(params, "accuracy")))
	case "set_offline":
		return textErr("offline state set", c.SetOffline(ctx, boolParam(params, "offline")))
	case "set_headers":
		return textErr("headers set", c.SetHeaders(ctx, stringMapParam(params, "headers")))
	case "cookies_get":
		return valueResult(c.GetCookies(ctx))
	case "cookies_clear":
		return textErr("cookies cleared", c.ClearCookies(ctx))
	case "tab_new":
		return valueResult(c.NewTab(ctx, stringParam(params, "url")))
	case "tab_close":
		return textErr("tab closed", c.CloseTab(ctx, requireString(params, "targetId")))
	default:
		return tools.ToolResult{}, fmt.Errorf("unknown browser action: %s", action)
	}
}

func (t *Tool) ensureClient(ctx context.Context, params map[string]any) (*vbclient.Client, error) {
	if t.client != nil && t.client.IsConnected() {
		return t.client, nil
	}
	opts := clientOptions(params)
	var c *vbclient.Client
	var err error
	if boolParam(params, "daemon") {
		c, err = vbclient.Connect(ctx, opts)
	} else {
		c, err = vbclient.Open(ctx, opts)
	}
	if err != nil {
		return nil, err
	}
	t.client = c
	return c, nil
}

func clientOptions(params map[string]any) *vbclient.Options {
	opts := &vbclient.Options{
		CDPURL:          firstNonEmpty(stringParam(params, "cdpUrl"), os.Getenv("VIBE_BROWSER_CDP_URL")),
		Session:         firstNonEmpty(stringParam(params, "session"), os.Getenv("VIBE_BROWSER_SESSION")),
		ExecutablePath:  firstNonEmpty(stringParam(params, "executablePath"), os.Getenv("CHROME_PATH")),
		DaemonSocketDir: os.Getenv("VIBE_BROWSER_SOCKET_DIR"),
		Launch: &vbprotocol.LaunchOptions{
			Headless:       true,
			ViewportWidth:  intParamDefault(params, "viewportWidth", defaultViewportWidth),
			ViewportHeight: intParamDefault(params, "viewportHeight", defaultViewportHeight),
		},
	}
	browserName := firstNonEmpty(stringParam(params, "browser"), os.Getenv("VIBE_BROWSER_BROWSER"))
	if browserName != "" {
		opts.Browser = vbprotocol.BrowserType(browserName)
		opts.Launch.Browser = opts.Browser
	}
	opts.Launch.ExecutablePath = opts.ExecutablePath
	if headless, ok := boolParamOK(params, "headless"); ok {
		opts.Launch.Headless = headless
	}
	return opts
}

func (t *Tool) screenshot(ctx context.Context, c *vbclient.Client, params map[string]any) (tools.ToolResult, error) {
	format := stringParam(params, "format")
	if format == "" {
		format = "png"
	}
	data, err := c.ScreenshotWithOptions(ctx, &vbprotocol.ScreenshotOptions{
		Format:   format,
		Quality:  intParam(params, "quality"),
		FullPage: boolParam(params, "fullPage"),
		Selector: stringParam(params, "selector"),
	})
	if err != nil {
		return tools.ToolResult{}, err
	}
	if outputPath := stringParam(params, "outputPath"); outputPath != "" {
		resolved, err := t.registry.ResolvePath(outputPath)
		if err != nil {
			return tools.ToolResult{}, err
		}
		if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
			return tools.ToolResult{}, err
		}
		if err := os.WriteFile(resolved, data, 0644); err != nil {
			return tools.ToolResult{}, err
		}
		return tools.NewTextToolResult(fmt.Sprintf("screenshot saved: %s", resolved)), nil
	}
	return t.screenshotToolResult(data, params)
}

func (t *Tool) screenshotToolResult(data []byte, params map[string]any) (tools.ToolResult, error) {
	policy := t.screenshotImagePolicy(params)
	result, err := imageproc.PrepareBytes(data, policy)
	if err != nil {
		return tools.ToolResult{}, fmt.Errorf("process screenshot: %w", err)
	}
	image := provider.ImageContent{
		Data:           base64.StdEncoding.EncodeToString(result.Data),
		MimeType:       result.MimeType,
		Width:          result.Meta.Width,
		Height:         result.Meta.Height,
		Bytes:          result.Meta.Bytes,
		OriginalWidth:  result.Meta.OriginalWidth,
		OriginalHeight: result.Meta.OriginalHeight,
		OriginalBytes:  result.Meta.OriginalBytes,
		Detail:         result.Meta.Detail,
		Scale:          result.Meta.Scale,
	}
	return tools.NewImageToolResultWithContent(browserScreenshotDescription(result), image), nil
}

func (t *Tool) screenshotImagePolicy(params map[string]any) imageproc.Policy {
	mode := imageproc.ModeDetail
	if v := stringParam(params, "imageMode"); v != "" {
		mode = imageproc.NormalizeMode(v)
	}
	policy := imageproc.DefaultPolicy(mode)
	if t.registry != nil {
		policy = t.registry.ImagePolicy(mode)
	}
	if v := intParam(params, "maxLongEdge"); v > 0 {
		policy.MaxLongEdge = v
	}
	return policy
}

func browserScreenshotDescription(result imageproc.Result) string {
	original := fmt.Sprintf("%dx%d %s", result.Meta.OriginalWidth, result.Meta.OriginalHeight, formatBytes(result.Meta.OriginalBytes))
	sent := fmt.Sprintf("%dx%d %s %s", result.Meta.Width, result.Meta.Height, formatBytes(result.Meta.Bytes), result.MimeType)
	if result.Meta.Resized || result.Meta.Transcoded || result.Meta.OriginalBytes != result.Meta.Bytes {
		return fmt.Sprintf("[Browser screenshot, original: %s, sent: %s, mode: %s]", original, sent, result.Meta.Detail)
	}
	return fmt.Sprintf("[Browser screenshot, %s, mode: %s]", sent, result.Meta.Detail)
}

func formatBytes(n int) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	kb := float64(n) / unit
	if kb < unit {
		return fmt.Sprintf("%.1fKB", kb)
	}
	return fmt.Sprintf("%.1fMB", kb/unit)
}

func (t *Tool) pageSummary(ctx context.Context, c *vbclient.Client, prefix string) (tools.ToolResult, error) {
	title, _ := c.Title(ctx)
	url, _ := c.URL(ctx)
	return tools.NewTextToolResult(strings.TrimSpace(fmt.Sprintf("%s\nTitle: %s\nURL: %s", prefix, title, url))), nil
}

func textErr(text string, err error) (tools.ToolResult, error) {
	if err != nil {
		return tools.ToolResult{}, err
	}
	return tools.NewTextToolResult(text), nil
}

func stringValue(ctx context.Context, fn func(context.Context, string) (string, error), arg string) (tools.ToolResult, error) {
	return valueResult(fn(ctx, arg))
}

func stringPairValue(ctx context.Context, fn func(context.Context, string, string) (string, error), a string, b string) (tools.ToolResult, error) {
	return valueResult(fn(ctx, a, b))
}

func valueResult(v any, err error) (tools.ToolResult, error) {
	if err != nil {
		return tools.ToolResult{}, err
	}
	switch val := v.(type) {
	case string:
		return tools.NewTextToolResult(val), nil
	case bool:
		return tools.NewTextToolResult(fmt.Sprintf("%v", val)), nil
	default:
		data, marshalErr := json.MarshalIndent(val, "", "  ")
		if marshalErr != nil {
			return tools.NewTextToolResult(fmt.Sprintf("%v", val)), nil
		}
		return tools.NewTextToolResult(string(data)), nil
	}
}

func requireString(params map[string]any, key string) string {
	v := stringParam(params, key)
	if v == "" {
		panic(fmt.Sprintf("%s is required", key))
	}
	return v
}

func stringParam(params map[string]any, key string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func boolParam(params map[string]any, key string) bool {
	v, _ := boolParamOK(params, key)
	return v
}

func boolParamOK(params map[string]any, key string) (bool, bool) {
	if v, ok := params[key].(bool); ok {
		return v, true
	}
	return false, false
}

func intParam(params map[string]any, key string) int {
	switch v := params[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}

func intParamDefault(params map[string]any, key string, defaultValue int) int {
	if value := intParam(params, key); value > 0 {
		return value
	}
	return defaultValue
}

func floatParam(params map[string]any, key string) float64 {
	switch v := params[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

func stringMapParam(params map[string]any, key string) map[string]string {
	out := map[string]string{}
	switch raw := params[key].(type) {
	case map[string]string:
		for k, v := range raw {
			out[k] = v
		}
	case map[string]any:
		for k, v := range raw {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
