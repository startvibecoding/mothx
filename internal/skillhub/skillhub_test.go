package skillhub

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillHubSearchAndUserSkills(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/v1/search":
			if got := r.URL.Query().Get("q"); got != "go" {
				t.Errorf("search q = %q", got)
			}
			return jsonResponse(`{"results":[{"slug":"go-expert","name":"Go Expert","owner_name":"mothx","updatedAt":1783990604537}]}`), nil
		case "/api/v1/users/mothx/skills":
			if got := r.URL.Query().Get("page"); got != "2" {
				t.Errorf("user skills page = %q", got)
			}
			return jsonResponse(`{"count":2,"skills":[{"slug":"go-expert","name":"Go Expert","description":"Go testing"},{"slug":"rust","name":"Rust"}]}`), nil
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader("not found")), Header: make(http.Header), Request: r}, nil
		}
	})}
	market := NewSkillHubClient("https://api.test", client)
	page, err := market.Search(context.Background(), SearchQuery{Query: "go", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Market != MarketSkillHub || page.Items[0].Author != "mothx" {
		t.Fatalf("unexpected page: %#v", page)
	}
	page, err = market.UserSkills(context.Background(), "mothx", UserSkillsQuery{Query: "testing", Limit: 20, Page: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "go-expert" {
		t.Fatalf("local user filtering failed: %#v", page.Items)
	}
}

func TestSkillHubDetailAcceptsCurrentVersionTagsAndStatsShape(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(`{"latestVersion":{"version":"1.0.0","createdAt":1774758863165},"owner":{"displayName":"claudiodrusus","handle":"claudiodrusus"},"skill":{"slug":"skill-1","displayName":"Skill 1","summary":"Generate QR codes","tags":{"latest":"1.0.0"},"stats":{"downloads":1238,"installs":43,"stars":2}}}`), nil
	})}
	detail, err := NewSkillHubClient("https://api.test", client).Detail(context.Background(), SkillID{Market: MarketSkillHub, ID: "skill-1"})
	if err != nil {
		t.Fatal(err)
	}
	if detail.Version != "1.0.0" || detail.Name != "Skill 1" || detail.Downloads != 1238 || detail.Installs != 43 || detail.Stars != 2 {
		t.Fatalf("unexpected detail: %#v", detail)
	}
	if len(detail.Tags) != 1 || detail.Tags[0] != "latest=1.0.0" {
		t.Fatalf("tags = %#v", detail.Tags)
	}
}

func TestClawHubSearchUsesCursorAndNormalizes(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/v1/skills" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("cursor"); got != "next" {
			t.Errorf("cursor = %q", got)
		}
		if got := r.URL.Query().Get("author"); got != "openclaw" {
			t.Errorf("author = %q", got)
		}
		return jsonResponse(`{"items":[{"id":"openclaw/git","name":"Git","summary":"git helpers","latestVersion":"2.0.0","owner":"openclaw","updatedAt":"2026-07-14T10:00:00Z"}],"nextCursor":"later"}`), nil
	})}
	page, err := NewClawHubClient("https://api.test", client).Search(context.Background(), SearchQuery{Limit: 10, Cursor: "next", Author: "openclaw"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "openclaw/git" || page.Items[0].Version != "2.0.0" || page.NextCursor != "later" {
		t.Fatalf("unexpected page: %#v", page)
	}
}

func TestClawHubFullTextSearchUsesSearchEndpointAndOwnerRef(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/v1/search" || r.URL.Query().Get("q") != "photo" {
			t.Errorf("search request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		return jsonResponse(`{"results":[{"slug":"photo","displayName":"Photo","summary":"Imaging","downloads":1266,"updatedAt":1778491780967,"ownerHandle":"agistack","owner":{"displayName":"AGIstack"}}]}`), nil
	})}
	page, err := NewClawHubClient("https://api.test", client).Search(context.Background(), SearchQuery{Query: "photo", Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "@agistack/photo" || page.Items[0].Author != "AGIstack" || page.Items[0].Downloads != 1266 {
		t.Fatalf("unexpected search page: %#v", page)
	}
}

func TestClawHubDetailAcceptsRootObject(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/v1/skills/openclaw/git" {
			t.Errorf("path = %s", r.URL.Path)
		}
		return jsonResponse(`{"id":"openclaw/git","name":"Git","version":"1.0.0","author":"openclaw"}`), nil
	})}
	detail, err := NewClawHubClient("https://api.test", client).Detail(context.Background(), SkillID{Market: MarketClawHub, ID: "openclaw/git"})
	if err != nil {
		t.Fatal(err)
	}
	if detail.ID != "openclaw/git" || detail.Version != "1.0.0" {
		t.Fatalf("unexpected detail: %#v", detail)
	}
}

func TestClawHubDetailAcceptsSlugEnvelope(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(`{"skill":{"slug":"drivethru-operations","displayName":"Drivethru Operations","summary":"Operations","updatedAt":1784072685483},"latestVersion":{"version":"0.1.0"},"owner":{"displayName":"zmtucker"},"moderation":{"verdict":"clean"}}`), nil
	})}
	detail, err := NewClawHubClient("https://api.test", client).Detail(context.Background(), SkillID{Market: MarketClawHub, ID: "drivethru-operations"})
	if err != nil {
		t.Fatal(err)
	}
	if detail.ID != "drivethru-operations" || detail.Name != "Drivethru Operations" || detail.Version != "0.1.0" || detail.Author != "zmtucker" {
		t.Fatalf("unexpected detail: %#v", detail)
	}
}

func TestClawHubOwnerRefUsesOwnerQueryForDetail(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/v1/skills/photo" || r.URL.Query().Get("owner") != "agistack" {
			t.Errorf("detail request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		return jsonResponse(`{"skill":{"slug":"photo","displayName":"Photo","summary":"Imaging"},"owner":{"displayName":"AGIstack"}}`), nil
	})}
	detail, err := NewClawHubClient("https://api.test", client).Detail(context.Background(), SkillID{Market: MarketClawHub, ID: "@agistack/photo"})
	if err != nil {
		t.Fatal(err)
	}
	if detail.ID != "@agistack/photo" || detail.Slug != "photo" || detail.Author != "AGIstack" {
		t.Fatalf("unexpected owner detail: %#v", detail)
	}
}

func TestInstallValidatesArchiveAndWritesMetadata(t *testing.T) {
	archive := makeArchive(t, map[string]string{"wrapped/SKILL.md": "# Test\n", "wrapped/references/a.md": "reference"})
	client := &fakeClient{detail: SkillDetail{SkillSummary: SkillSummary{Market: MarketSkillHub, ID: "test-skill", Slug: "test-skill", Version: "1.0.0"}}, archive: archive}
	target := filepath.Join(t.TempDir(), ".mothx", "skills")
	result, err := Install(context.Background(), client, InstallRequest{Market: MarketSkillHub, ID: "test-skill", Scope: "project", TargetDir: target})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Installed || result.Dir != filepath.Join(target, "test-skill") {
		t.Fatalf("unexpected install result: %#v", result)
	}
	if data, err := os.ReadFile(filepath.Join(result.Dir, "SKILL.md")); err != nil || string(data) != "# Test\n" {
		t.Fatalf("skill file: %q, %v", data, err)
	}
	metadata, err := readMetadata(result.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Market != MarketSkillHub || metadata.ID != "test-skill" || metadata.Version != "1.0.0" {
		t.Fatalf("metadata = %#v", metadata)
	}
	second, err := Install(context.Background(), client, InstallRequest{Market: MarketSkillHub, ID: "test-skill", Scope: "project", TargetDir: target})
	if err != nil || !second.AlreadyInstalled {
		t.Fatalf("same-version installation = %#v, %v", second, err)
	}
	index, err := NewLocalIndex("", []string{target})
	if err != nil {
		t.Fatal(err)
	}
	if state := index.State(MarketSkillHub, "test-skill"); state == nil || state.Scope != "project" {
		t.Fatalf("installed state = %#v", state)
	}
}

func TestInstallRejectsTraversalAndLocalSkill(t *testing.T) {
	target := t.TempDir()
	traversal := makeArchive(t, map[string]string{"../outside/SKILL.md": "# bad"})
	client := &fakeClient{detail: SkillDetail{SkillSummary: SkillSummary{Market: MarketSkillHub, ID: "bad", Slug: "bad", Version: "1"}}, archive: traversal}
	_, err := Install(context.Background(), client, InstallRequest{Market: MarketSkillHub, ID: "bad", TargetDir: target})
	if !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("traversal error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(target), "outside")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("archive escaped target: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(target, "local"), 0755); err != nil {
		t.Fatal(err)
	}
	local := &fakeClient{detail: SkillDetail{SkillSummary: SkillSummary{Market: MarketSkillHub, ID: "local", Slug: "local", Version: "1"}}, archive: makeArchive(t, map[string]string{"SKILL.md": "# test"})}
	_, err = Install(context.Background(), local, InstallRequest{Market: MarketSkillHub, ID: "local", TargetDir: target})
	if !errors.Is(err, ErrLocalSkillExists) {
		t.Fatalf("local skill error = %v", err)
	}
	_, err = Install(context.Background(), local, InstallRequest{Market: MarketSkillHub, ID: "local", TargetDir: target, Overwrite: true})
	if !errors.Is(err, ErrLocalSkillExists) {
		t.Fatalf("local skill overwrite error = %v", err)
	}
}

func TestInstallUpdatesManagedSkillAndRejectsDifferentOwner(t *testing.T) {
	target := t.TempDir()
	client := &fakeClient{
		detail:  SkillDetail{SkillSummary: SkillSummary{Market: MarketSkillHub, ID: "managed", Slug: "managed", Version: "1.0.0"}},
		archive: makeArchive(t, map[string]string{"SKILL.md": "version one"}),
	}
	first, err := Install(context.Background(), client, InstallRequest{Market: MarketSkillHub, ID: "managed", Scope: "project", TargetDir: target})
	if err != nil {
		t.Fatal(err)
	}
	client.detail.Version = "2.0.0"
	client.archive = makeArchive(t, map[string]string{"SKILL.md": "version two"})
	updated, err := Install(context.Background(), client, InstallRequest{Market: MarketSkillHub, ID: "managed", Version: "2.0.0", Scope: "project", TargetDir: target, Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if data, readErr := os.ReadFile(filepath.Join(updated.Dir, "SKILL.md")); readErr != nil || string(data) != "version two" {
		t.Fatalf("updated skill file = %q, %v", data, readErr)
	}
	metadata, err := readMetadata(updated.Dir)
	if err != nil || metadata.Version != "2.0.0" {
		t.Fatalf("updated metadata = %#v, %v", metadata, err)
	}
	backups, err := filepath.Glob(filepath.Join(target, ".backup", "managed-*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("backup paths = %#v, %v", backups, err)
	}

	if err := writeMetadata(first.Dir, InstallMetadata{Market: MarketClawHub, ID: "another", Version: "1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(context.Background(), client, InstallRequest{Market: MarketSkillHub, ID: "managed", Version: "3.0.0", TargetDir: target, Overwrite: true}); err == nil || !strings.Contains(err.Error(), "managed by") {
		t.Fatalf("different owner overwrite error = %v", err)
	}
}

func TestLocalIndexMarksAvailableUpdate(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "go")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeMetadata(dir, InstallMetadata{Market: MarketSkillHub, ID: "go", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	index, err := NewLocalIndex("", []string{root})
	if err != nil {
		t.Fatal(err)
	}
	items := []SkillSummary{{Market: MarketSkillHub, ID: "go", Version: "2.0.0"}}
	index.Apply(items)
	if items[0].Installed == nil || !items[0].Installed.UpdateAvailable {
		t.Fatalf("installed state = %#v", items[0].Installed)
	}
}

func TestServiceOfficialAggregatesAndAppliesInstalled(t *testing.T) {
	global := t.TempDir()
	dir := filepath.Join(global, "go-expert")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeMetadata(dir, InstallMetadata{Market: MarketSkillHub, ID: "go-expert", Version: "1"}); err != nil {
		t.Fatal(err)
	}
	client := &fakeClient{users: map[string][]SkillSummary{"one": {{Market: MarketSkillHub, ID: "go-expert", Name: "Go", Downloads: 10}}, "two": {{Market: MarketSkillHub, ID: "go-expert", Name: "Go", Downloads: 10}, {Market: MarketSkillHub, ID: "other", Name: "Other", Downloads: 20}}}}
	service := NewService(global, nil, []string{"one", "two"}, client)
	page, err := service.Official(context.Background(), UserSkillsQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].ID != "other" || page.Items[1].Installed == nil || !page.Items[1].Installed.Installed {
		t.Fatalf("unexpected official page: %#v", page)
	}
	if page.Total != 2 {
		t.Fatalf("official total = %d, want 2", page.Total)
	}
}

func TestSkillHubBrowseSendsDownloadSortAndParsesCertification(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("sortBy") != "downloads" || r.URL.Query().Get("order") != "desc" {
			t.Errorf("sort query = %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("category") != "dev-programming" {
			t.Errorf("category query = %s", r.URL.RawQuery)
		}
		return jsonResponse(`{"code":0,"data":{"total":1,"skills":[{"slug":"mail","name":"Mail","source":"enterprise","downloads":26129,"publisher":{"name":"QQ Mail","verified":true,"certifiedName":"Tencent"}}]}}`), nil
	})}
	page, err := NewSkillHubClient("https://api.test", client).Search(context.Background(), SearchQuery{Limit: 10, Sort: "downloads", Order: "desc", Category: "dev-programming"})
	if err != nil {
		t.Fatal(err)
	}
	item := page.Items[0]
	if item.Downloads != 26129 || item.Source != "enterprise" || !item.PublisherVerified || item.PublisherName != "QQ Mail" || item.CertifiedName != "Tencent" {
		t.Fatalf("unexpected certified item: %#v", item)
	}
}

func TestSkillHubCategories(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/v1/categories" {
			t.Errorf("path = %s", r.URL.Path)
		}
		return jsonResponse(`{"items":[{"key":"dev-programming","name":"开发编程","nameEn":"Development"}]}`), nil
	})}
	categories, err := NewSkillHubClient("https://api.test", client).Categories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(categories) != 1 || categories[0].Key != "dev-programming" {
		t.Fatalf("categories = %#v", categories)
	}
}

func TestClawHubCurrentListShapeParsesDownloadsAndVersion(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(`{"items":[{"slug":"tool","displayName":"Tool","tags":{"latest":"1.2.0"},"stats":{"downloads":1562,"installs":6,"stars":3},"latestVersion":{"version":"1.2.0"},"updatedAt":1784072685483}]}`), nil
	})}
	page, err := NewClawHubClient("https://api.test", client).Search(context.Background(), SearchQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Downloads != 1562 || page.Items[0].Version != "1.2.0" || page.Items[0].UpdatedAt.IsZero() {
		t.Fatalf("unexpected ClawHub item: %#v", page.Items)
	}
}

func TestServiceDetailIncludesFiles(t *testing.T) {
	client := &fakeClient{
		detail:   SkillDetail{SkillSummary: SkillSummary{Market: MarketSkillHub, ID: "go", Version: "1"}},
		fileList: true,
		files:    []SkillFile{{Path: "SKILL.md"}, {Path: "references/go.md"}},
	}
	detail, err := NewService("", nil, nil, client).Detail(context.Background(), MarketSkillHub, "go")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Files) != 2 || detail.Files[0].Path != "SKILL.md" {
		t.Fatalf("detail files = %#v", detail.Files)
	}
	if len(detail.DownloadSources) != 1 || detail.DownloadSources[0].Kind != "test" {
		t.Fatalf("download sources = %#v", detail.DownloadSources)
	}
}

func TestServiceDetailIncludesEvaluation(t *testing.T) {
	want := map[string]any{"dimensions": map[string]any{"quality": map[string]any{}}}
	client := &fakeClient{
		detail:        SkillDetail{SkillSummary: SkillSummary{Market: MarketSkillHub, ID: "go", Version: "1"}},
		evaluationCap: true,
		evaluation:    want,
	}
	detail, err := NewService("", nil, nil, client).Detail(context.Background(), MarketSkillHub, "go")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Evaluation == nil {
		t.Fatal("evaluation was not attached to detail")
	}
}

type fakeClient struct {
	detail        SkillDetail
	archive       []byte
	users         map[string][]SkillSummary
	fileList      bool
	files         []SkillFile
	evaluationCap bool
	evaluation    any
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }
func jsonResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func (f *fakeClient) Market() MarketInfo {
	return MarketInfo{ID: MarketSkillHub, Capabilities: MarketCapabilities{FileList: f.fileList, Evaluation: f.evaluationCap}}
}
func (f *fakeClient) Search(context.Context, SearchQuery) (SearchPage, error) {
	return SearchPage{}, nil
}
func (f *fakeClient) UserSkills(_ context.Context, handle string, _ UserSkillsQuery) (SearchPage, error) {
	return SearchPage{Items: f.users[handle], Total: int64(len(f.users[handle]))}, nil
}
func (f *fakeClient) Detail(context.Context, SkillID) (SkillDetail, error) { return f.detail, nil }
func (f *fakeClient) Files(context.Context, SkillID, string) ([]SkillFile, error) {
	return f.files, nil
}
func (f *fakeClient) Evaluation(context.Context, SkillID) (any, error) { return f.evaluation, nil }
func (f *fakeClient) DownloadSources(SkillID, string) []DownloadSource {
	return []DownloadSource{{URL: "https://example.test/skill.zip", Kind: "test"}}
}
func (f *fakeClient) Download(context.Context, SkillID, string) (io.ReadCloser, DownloadMeta, error) {
	return io.NopCloser(bytes.NewReader(f.archive)), DownloadMeta{SourceURL: "https://example.test/skill.zip"}, nil
}
func (f *fakeClient) Categories(context.Context) ([]Category, error) { return nil, nil }

func makeArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for path, content := range files {
		entry, err := writer.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}
