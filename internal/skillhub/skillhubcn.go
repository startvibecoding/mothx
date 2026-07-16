package skillhub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const skillHubDefaultURL = "https://api.skillhub.cn"
const skillHubDownloadBaseURL = "https://skillhub-1388575217.cos.ap-guangzhou.myqcloud.com"

type SkillHubClient struct {
	baseURL     string
	downloadURL string
	httpClient  *http.Client
}

func NewSkillHubClient(baseURL string, client *http.Client) *SkillHubClient {
	if baseURL == "" {
		baseURL = skillHubDefaultURL
	}
	return &SkillHubClient{baseURL: strings.TrimRight(baseURL, "/"), downloadURL: skillHubDownloadBaseURL, httpClient: newHTTPClient(client)}
}

func (c *SkillHubClient) Market() MarketInfo {
	return MarketInfo{ID: MarketSkillHub, Name: "SkillHub.cn", SiteURL: "https://skillhub.cn", Capabilities: MarketCapabilities{Search: true, List: true, PagePagination: true, Categories: true, Showcase: true, UserSkills: true, FileList: true, Evaluation: true}}
}

func (c *SkillHubClient) Search(ctx context.Context, q SearchQuery) (SearchPage, error) {
	limit := boundedLimit(q.Limit)
	if strings.TrimSpace(q.Query) != "" {
		var response struct {
			Results []skillHubItem `json:"results"`
		}
		if err := getJSON(ctx, c.httpClient, endpoint(c.baseURL, "/api/v1/search", url.Values{"q": {q.Query}, "limit": {strconv.Itoa(limit)}}), &response); err != nil {
			return SearchPage{}, err
		}
		return SearchPage{Items: skillHubItems(response.Results), Total: int64(len(response.Results)), Page: 1, PageSize: limit}, nil
	}
	page := q.Page
	if page < 1 {
		page = 1
	}
	values := url.Values{"page": {strconv.Itoa(page)}, "pageSize": {strconv.Itoa(limit)}}
	if q.Category != "" {
		values.Set("category", q.Category)
	}
	if q.Sort != "" {
		values.Set("sortBy", q.Sort)
	}
	if q.Order != "" {
		values.Set("order", q.Order)
	}
	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Total  int64          `json:"total"`
			Skills []skillHubItem `json:"skills"`
		} `json:"data"`
	}
	if err := getJSON(ctx, c.httpClient, endpoint(c.baseURL, "/api/skills", values), &response); err != nil {
		return SearchPage{}, err
	}
	if response.Code != 0 {
		return SearchPage{}, fmt.Errorf("SkillHub list: %s", response.Message)
	}
	return SearchPage{Items: skillHubItems(response.Data.Skills), Total: response.Data.Total, Page: page, PageSize: limit}, nil
}

func (c *SkillHubClient) UserSkills(ctx context.Context, handle string, q UserSkillsQuery) (SearchPage, error) {
	limit := boundedLimit(q.Limit)
	page := q.Page
	if page < 1 {
		page = 1
	}
	var response struct {
		Count  int64          `json:"count"`
		Skills []skillHubItem `json:"skills"`
	}
	path := "/api/v1/users/" + url.PathEscape(handle) + "/skills"
	if err := getJSON(ctx, c.httpClient, endpoint(c.baseURL, path, url.Values{"page": {strconv.Itoa(page)}, "pageSize": {strconv.Itoa(limit)}}), &response); err != nil {
		return SearchPage{}, err
	}
	items := skillHubItems(response.Skills)
	if q.Query != "" {
		items = filterSkills(items, q.Query)
	}
	return SearchPage{Items: items, Total: response.Count, Page: page, PageSize: limit}, nil
}

func (c *SkillHubClient) Detail(ctx context.Context, id SkillID) (SkillDetail, error) {
	var response struct {
		Skill skillHubItem `json:"skill"`
		Owner struct {
			Handle      string `json:"handle"`
			DisplayName string `json:"displayName"`
		} `json:"owner"`
		LatestVersion   skillHubVersion `json:"latestVersion"`
		SecurityReports any             `json:"securityReports"`
	}
	if err := getJSON(ctx, c.httpClient, endpoint(c.baseURL, "/api/v1/skills/"+url.PathEscape(id.ID), nil), &response); err != nil {
		return SkillDetail{}, err
	}
	summary := response.Skill.summary()
	if response.Owner.DisplayName != "" {
		summary.Author = response.Owner.DisplayName
	} else if response.Owner.Handle != "" {
		summary.Author = response.Owner.Handle
	}
	if response.LatestVersion.Version != "" {
		summary.Version = response.LatestVersion.Version
	}
	return SkillDetail{SkillSummary: summary, SecurityReports: response.SecurityReports}, nil
}

func (c *SkillHubClient) Files(ctx context.Context, id SkillID, version string) ([]SkillFile, error) {
	values := url.Values{}
	if version != "" {
		values.Set("version", version)
	}
	var response struct {
		Files []SkillFile `json:"files"`
	}
	err := getJSON(ctx, c.httpClient, endpoint(c.baseURL, "/api/v1/skills/"+url.PathEscape(id.ID)+"/files", values), &response)
	return response.Files, err
}

func (c *SkillHubClient) Evaluation(ctx context.Context, id SkillID) (any, error) {
	var response any
	err := getJSON(ctx, c.httpClient, endpoint(c.baseURL, "/api/v1/skills/"+url.PathEscape(id.ID)+"/evaluation", nil), &response)
	return response, err
}

func (c *SkillHubClient) DownloadSources(id SkillID, version string) []DownloadSource {
	values := url.Values{"slug": {id.ID}}
	if version != "" {
		values.Set("version", version)
	}
	return []DownloadSource{
		{URL: endpoint(c.baseURL, "/api/v1/download", values), Kind: "api"},
		{URL: strings.TrimRight(c.downloadURL, "/") + "/skills/" + url.PathEscape(id.ID) + ".zip", Kind: "cdn", Fallback: true},
	}
}

func (c *SkillHubClient) Download(ctx context.Context, id SkillID, version string) (io.ReadCloser, DownloadMeta, error) {
	values := url.Values{"slug": {id.ID}}
	if version != "" {
		values.Set("version", version)
	}
	primary := endpoint(c.baseURL, "/api/v1/download", values)
	body, err := download(ctx, c.httpClient, primary)
	if err == nil {
		return body, DownloadMeta{SourceURL: primary}, nil
	}
	fallback := strings.TrimRight(c.downloadURL, "/") + "/skills/" + url.PathEscape(id.ID) + ".zip"
	body, fallbackErr := download(ctx, c.httpClient, fallback)
	if fallbackErr != nil {
		return nil, DownloadMeta{}, fmt.Errorf("download %s: primary: %v; fallback: %w", id.ID, err, fallbackErr)
	}
	return body, DownloadMeta{SourceURL: fallback}, nil
}

func (c *SkillHubClient) Showcase(ctx context.Context, kind string, q SearchQuery) (SearchPage, error) {
	var response struct {
		Items  []skillHubItem `json:"items"`
		Skills []skillHubItem `json:"skills"`
	}
	if err := getJSON(ctx, c.httpClient, endpoint(c.baseURL, "/api/v1/showcase/"+url.PathEscape(kind), url.Values{"limit": {strconv.Itoa(boundedLimit(q.Limit))}}), &response); err != nil {
		return SearchPage{}, err
	}
	items := response.Items
	if len(items) == 0 {
		items = response.Skills
	}
	return SearchPage{Items: skillHubItems(items), Total: int64(len(items)), Page: 1, PageSize: boundedLimit(q.Limit)}, nil
}

func (c *SkillHubClient) FileContent(context.Context, SkillID, string, string) (string, error) {
	return "", fmt.Errorf("SkillHub.cn does not expose skill file content")
}
func (c *SkillHubClient) Categories(ctx context.Context) ([]Category, error) {
	var response struct {
		Items []Category `json:"items"`
	}
	err := getJSON(ctx, c.httpClient, endpoint(c.baseURL, "/api/v1/categories", nil), &response)
	return response.Items, err
}

type skillHubItem struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Summary     string `json:"summary"`
	Version     string `json:"version"`
	Category    string `json:"category"`
	OwnerName   string `json:"ownerName"`
	OwnerNameV1 string `json:"owner_name"`
	Homepage    string `json:"homepage"`
	IconURL     string `json:"iconUrl"`
	IconURLV1   string `json:"icon_url"`
	SourceURL   string `json:"sourceUrl"`
	Source      string `json:"source"`
	Publisher   *struct {
		Name          string `json:"name"`
		Verified      bool   `json:"verified"`
		CertifiedName string `json:"certifiedName"`
	} `json:"publisher"`
	Downloads int64        `json:"downloads"`
	Installs  int64        `json:"installs"`
	Stars     int64        `json:"stars"`
	Score     float64      `json:"score"`
	Verified  bool         `json:"verified"`
	Tags      skillHubTags `json:"tags"`
	Stats     struct {
		Downloads int64 `json:"downloads"`
		Installs  int64 `json:"installs"`
		Stars     int64 `json:"stars"`
	} `json:"stats"`
	UpdatedAtMillis int64 `json:"updatedAt"`
	UpdatedAtLegacy int64 `json:"updated_at"`
}

func (s skillHubItem) summary() SkillSummary {
	description := s.Description
	if description == "" {
		description = s.Summary
	}
	name := s.Name
	if name == "" {
		name = s.DisplayName
	}
	author := s.OwnerName
	if author == "" {
		author = s.OwnerNameV1
	}
	icon := s.IconURL
	if icon == "" {
		icon = s.IconURLV1
	}
	updated := s.UpdatedAtMillis
	if updated == 0 {
		updated = s.UpdatedAtLegacy
	}
	var updatedAt time.Time
	if updated > 0 {
		updatedAt = time.UnixMilli(updated)
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
	publisherName, certifiedName, publisherVerified := "", "", false
	if s.Publisher != nil {
		publisherName, certifiedName, publisherVerified = s.Publisher.Name, s.Publisher.CertifiedName, s.Publisher.Verified
	}
	return SkillSummary{Market: MarketSkillHub, ID: s.Slug, Slug: s.Slug, Name: name, DisplayName: s.DisplayName, Description: description, Version: s.Version, Author: author, Category: s.Category, Tags: []string(s.Tags), IconURL: icon, Homepage: s.Homepage, SourceURL: s.SourceURL, Source: s.Source, PublisherName: publisherName, CertifiedName: certifiedName, PublisherVerified: publisherVerified, Downloads: downloads, Installs: installs, Stars: stars, Score: s.Score, Verified: s.Verified, UpdatedAt: updatedAt}
}
func skillHubItems(items []skillHubItem) []SkillSummary {
	out := make([]SkillSummary, 0, len(items))
	for _, item := range items {
		out = append(out, item.summary())
	}
	return out
}

type skillHubVersion struct{ Version string }

func (v *skillHubVersion) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var text string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		v.Version = text
		return nil
	}
	var object struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	v.Version = object.Version
	return nil
}

type skillHubTags []string

func (t *skillHubTags) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if len(data) > 0 && data[0] == '[' {
		var values []string
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		*t = values
		return nil
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		if value, ok := values[key].(string); ok && value != "" {
			result = append(result, key+"="+value)
		} else {
			result = append(result, key)
		}
	}
	*t = result
	return nil
}
