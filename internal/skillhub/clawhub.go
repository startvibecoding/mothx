package skillhub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const clawHubDefaultURL = "https://clawhub.ai"

type ClawHubClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewClawHubClient(baseURL string, client *http.Client) *ClawHubClient {
	if baseURL == "" {
		baseURL = clawHubDefaultURL
	}
	return &ClawHubClient{baseURL: strings.TrimRight(baseURL, "/"), httpClient: newHTTPClient(client)}
}

func (c *ClawHubClient) Market() MarketInfo {
	return MarketInfo{ID: MarketClawHub, Name: "ClawHub.ai", SiteURL: "https://clawhub.ai", Capabilities: MarketCapabilities{Search: true, List: true, CursorPagination: true, AuthorFilter: true, FileList: true, FileContent: true}}
}

func (c *ClawHubClient) Search(ctx context.Context, q SearchQuery) (SearchPage, error) {
	if strings.TrimSpace(q.Query) != "" {
		var response struct {
			Results []clawHubSearchItem `json:"results"`
		}
		values := url.Values{"q": {q.Query}, "limit": {strconv.Itoa(boundedLimit(q.Limit))}}
		if err := getJSON(ctx, c.httpClient, endpoint(c.baseURL, "/api/v1/search", values), &response); err != nil {
			return SearchPage{}, err
		}
		items := make([]SkillSummary, 0, len(response.Results))
		for _, item := range response.Results {
			items = append(items, item.summary())
		}
		return SearchPage{Items: items, Total: int64(len(items)), Page: 1, PageSize: boundedLimit(q.Limit)}, nil
	}
	values := url.Values{"limit": {strconv.Itoa(boundedLimit(q.Limit))}}
	if q.Cursor != "" {
		values.Set("cursor", q.Cursor)
	}
	if q.Sort != "" {
		values.Set("order", q.Sort)
	}
	if q.Author != "" {
		values.Set("author", q.Author)
	}
	if q.VerifiedOnly {
		values.Set("verifiedOnly", "true")
	}
	if q.NonSuspiciousOnly {
		values.Set("nonSuspiciousOnly", "true")
	}
	var response struct {
		Items      []clawHubItem `json:"items"`
		NextCursor string        `json:"nextCursor"`
	}
	if err := getJSON(ctx, c.httpClient, endpoint(c.baseURL, "/api/v1/skills", values), &response); err != nil {
		return SearchPage{}, err
	}
	items := make([]SkillSummary, 0, len(response.Items))
	for _, item := range response.Items {
		items = append(items, item.summary())
	}
	return SearchPage{Items: items, NextCursor: response.NextCursor, PageSize: boundedLimit(q.Limit)}, nil
}

// UserSkills is not available from the public ClawHub API. Search supports author filtering.
func (c *ClawHubClient) UserSkills(ctx context.Context, handle string, q UserSkillsQuery) (SearchPage, error) {
	return c.Search(ctx, SearchQuery{Query: q.Query, Limit: q.Limit, Cursor: "", Author: handle})
}

func (c *ClawHubClient) Detail(ctx context.Context, id SkillID) (SkillDetail, error) {
	var response clawHubDetailResponse
	slug, owner := clawSkillRef(id.ID)
	values := url.Values{}
	if owner != "" {
		values.Set("owner", owner)
	}
	if err := getJSON(ctx, c.httpClient, endpoint(c.baseURL, "/api/v1/skills/"+skillPath(slug), values), &response); err != nil {
		return SkillDetail{}, err
	}
	item := response.Skill
	if item.ID == "" && item.Slug == "" {
		item = response.Item
	}
	summary := item.summary()
	summary.ID = id.ID
	detail := SkillDetail{SkillSummary: summary, SecurityReports: response.SecurityReports, Evaluation: response.Evaluation}
	return detail, nil
}

func (c *ClawHubClient) Files(ctx context.Context, id SkillID, version string) ([]SkillFile, error) {
	slug, owner := clawSkillRef(id.ID)
	values := url.Values{}
	if version != "" {
		values.Set("version", version)
	}
	if owner != "" {
		values.Set("owner", owner)
	}
	var response struct {
		Files []SkillFile `json:"files"`
		Items []SkillFile `json:"items"`
	}
	err := getJSON(ctx, c.httpClient, endpoint(c.baseURL, "/api/v1/skills/"+skillPath(slug)+"/files", values), &response)
	if len(response.Files) > 0 {
		return response.Files, err
	}
	return response.Items, err
}

func (c *ClawHubClient) Evaluation(context.Context, SkillID) (any, error) { return nil, nil }

func (c *ClawHubClient) FileContent(ctx context.Context, id SkillID, version, path string) (string, error) {
	slug, owner := clawSkillRef(id.ID)
	values := url.Values{}
	if version != "" {
		values.Set("version", version)
	}
	if owner != "" {
		values.Set("owner", owner)
	}
	var response struct {
		Content string `json:"content"`
	}
	url := endpoint(c.baseURL, "/api/v1/skills/"+skillPath(slug)+"/files/"+skillPath(path), values)
	if err := getJSON(ctx, c.httpClient, url, &response); err == nil && response.Content != "" {
		return response.Content, nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	result, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer result.Body.Close()
	if result.StatusCode < 200 || result.StatusCode >= 300 {
		return "", fmt.Errorf("GET %s: %s", url, result.Status)
	}
	body, err := io.ReadAll(io.LimitReader(result.Body, 1<<20))
	return string(body), err
}

func (c *ClawHubClient) DownloadSources(id SkillID, version string) []DownloadSource {
	slug, owner := clawSkillRef(id.ID)
	values := url.Values{}
	if version != "" {
		values.Set("version", version)
	}
	if owner != "" {
		values.Set("owner", owner)
	}
	return []DownloadSource{{URL: endpoint(c.baseURL, "/api/v1/skills/"+skillPath(slug)+"/download", values), Kind: "api"}}
}

func (c *ClawHubClient) Download(ctx context.Context, id SkillID, version string) (io.ReadCloser, DownloadMeta, error) {
	slug, owner := clawSkillRef(id.ID)
	values := url.Values{}
	if version != "" {
		values.Set("version", version)
	}
	if owner != "" {
		values.Set("owner", owner)
	}
	path := endpoint(c.baseURL, "/api/v1/skills/"+skillPath(slug)+"/download", values)
	body, err := download(ctx, c.httpClient, path)
	if err != nil {
		return nil, DownloadMeta{}, err
	}
	return body, DownloadMeta{SourceURL: path}, nil
}

func (c *ClawHubClient) Categories(context.Context) ([]Category, error) { return nil, nil }

func skillPath(id string) string {
	return strings.Join(func() []string {
		parts := strings.Split(id, "/")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			out = append(out, url.PathEscape(part))
		}
		return out
	}(), "/")
}

func clawSkillRef(id string) (slug, owner string) {
	if !strings.HasPrefix(id, "@") {
		return id, ""
	}
	parts := strings.SplitN(strings.TrimPrefix(id, "@"), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return id, ""
	}
	return parts[1], parts[0]
}

type clawHubDetailResponse struct {
	Skill clawHubItem `json:"skill"`
	Item  clawHubItem `json:"item"`
	Owner struct {
		Handle      string `json:"handle"`
		DisplayName string `json:"displayName"`
	} `json:"owner"`
	SecurityReports any `json:"securityReports"`
	Evaluation      any `json:"evaluation"`
}

func (r *clawHubDetailResponse) UnmarshalJSON(data []byte) error {
	var envelope struct {
		Skill         json.RawMessage `json:"skill"`
		Item          json.RawMessage `json:"item"`
		LatestVersion skillHubVersion `json:"latestVersion"`
		Owner         struct {
			Handle      string `json:"handle"`
			DisplayName string `json:"displayName"`
		} `json:"owner"`
		Moderation      any `json:"moderation"`
		SecurityReports any `json:"securityReports"`
		Evaluation      any `json:"evaluation"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	r.SecurityReports = envelope.SecurityReports
	if r.SecurityReports == nil {
		r.SecurityReports = envelope.Moderation
	}
	r.Evaluation = envelope.Evaluation
	r.Owner = envelope.Owner
	if len(envelope.Skill) > 0 && string(envelope.Skill) != "null" {
		if err := json.Unmarshal(envelope.Skill, &r.Skill); err != nil {
			return err
		}
		r.Skill.LatestVersion = envelope.LatestVersion
		if r.Skill.Author == "" {
			if envelope.Owner.DisplayName != "" {
				r.Skill.Author = envelope.Owner.DisplayName
			} else {
				r.Skill.Author = envelope.Owner.Handle
			}
		}
		return nil
	}
	if len(envelope.Item) > 0 && string(envelope.Item) != "null" {
		return json.Unmarshal(envelope.Item, &r.Item)
	}
	return json.Unmarshal(data, &r.Item)
}

// The API has evolved its field names. This shape accepts both documented and
// current variants while keeping the public model stable for the UI.
type clawHubItem struct {
	ID            string          `json:"id"`
	Slug          string          `json:"slug"`
	Name          string          `json:"name"`
	DisplayName   string          `json:"displayName"`
	Description   string          `json:"description"`
	Summary       string          `json:"summary"`
	Version       string          `json:"version"`
	LatestVersion skillHubVersion `json:"latestVersion"`
	Author        string          `json:"author"`
	Owner         string          `json:"owner"`
	Category      string          `json:"category"`
	Tags          skillHubTags    `json:"tags"`
	Topics        []string        `json:"topics"`
	Stats         struct {
		Downloads int64 `json:"downloads"`
		Installs  int64 `json:"installs"`
		Stars     int64 `json:"stars"`
	} `json:"stats"`
	IconURL         string            `json:"iconUrl"`
	Homepage        string            `json:"homepage"`
	SourceURL       string            `json:"sourceUrl"`
	Downloads       int64             `json:"downloads"`
	Installs        int64             `json:"installs"`
	Stars           int64             `json:"stars"`
	Score           float64           `json:"score"`
	Verified        bool              `json:"verified"`
	Suspicious      bool              `json:"suspicious"`
	UpdatedAt       flexibleTimestamp `json:"updatedAt"`
	UpdatedAtMillis int64             `json:"updatedAtMs"`
}

func (s clawHubItem) summary() SkillSummary {
	id := s.ID
	if id == "" {
		id = s.Slug
	}
	slug := s.Slug
	if slug == "" {
		slug = id
	}
	name := s.Name
	if name == "" {
		name = s.DisplayName
	}
	description := s.Summary
	if description == "" {
		description = s.Description
	}
	author := s.Author
	if author == "" {
		author = s.Owner
	}
	version := s.Version
	if version == "" {
		version = s.LatestVersion.Version
	}
	var updated time.Time
	if s.UpdatedAtMillis > 0 {
		updated = time.UnixMilli(s.UpdatedAtMillis)
	} else {
		updated = s.UpdatedAt.Time
	}
	downloads, installs, stars := s.Downloads, s.Installs, s.Stars
	if downloads == 0 {
		downloads = s.Stats.Downloads
	}
	if installs == 0 {
		installs = s.Stats.Installs
	}
	if stars == 0 {
		stars = s.Stats.Stars
	}
	tags := append([]string{}, s.Topics...)
	tags = append(tags, []string(s.Tags)...)
	return SkillSummary{Market: MarketClawHub, ID: id, Slug: slug, Name: name, DisplayName: s.DisplayName, Description: description, Version: version, Author: author, Category: s.Category, Tags: tags, IconURL: s.IconURL, Homepage: s.Homepage, SourceURL: s.SourceURL, Downloads: downloads, Installs: installs, Stars: stars, Score: s.Score, Verified: s.Verified, Suspicious: s.Suspicious, UpdatedAt: updated}
}

type flexibleTimestamp struct{ Time time.Time }

func (t *flexibleTimestamp) UnmarshalJSON(data []byte) error {
	value := strings.TrimSpace(string(data))
	if value == "" || value == "null" {
		return nil
	}
	if strings.HasPrefix(value, `"`) {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		parsed, err := time.Parse(time.RFC3339, text)
		if err != nil {
			return fmt.Errorf("parse timestamp %q: %w", text, err)
		}
		t.Time = parsed
		return nil
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("parse timestamp %q: %w", value, err)
	}
	if number > 1e12 {
		t.Time = time.UnixMilli(int64(number))
	} else {
		t.Time = time.Unix(int64(number), 0)
	}
	return nil
}

type clawHubSearchItem struct {
	Slug        string            `json:"slug"`
	DisplayName string            `json:"displayName"`
	Summary     string            `json:"summary"`
	Version     string            `json:"version"`
	Downloads   int64             `json:"downloads"`
	Score       float64           `json:"score"`
	UpdatedAt   flexibleTimestamp `json:"updatedAt"`
	OwnerHandle string            `json:"ownerHandle"`
	Owner       struct {
		Handle      string `json:"handle"`
		DisplayName string `json:"displayName"`
	} `json:"owner"`
}

func (s clawHubSearchItem) summary() SkillSummary {
	owner := s.OwnerHandle
	if owner == "" {
		owner = s.Owner.Handle
	}
	id := s.Slug
	if owner != "" {
		id = "@" + strings.TrimPrefix(owner, "@") + "/" + s.Slug
	}
	author := s.Owner.DisplayName
	if author == "" {
		author = owner
	}
	return SkillSummary{
		Market: MarketClawHub, ID: id, Slug: s.Slug, Name: s.DisplayName, DisplayName: s.DisplayName,
		Description: s.Summary, Version: s.Version, Author: author, Downloads: s.Downloads, Score: s.Score, UpdatedAt: s.UpdatedAt.Time,
	}
}
