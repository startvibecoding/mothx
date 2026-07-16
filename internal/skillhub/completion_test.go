package skillhub

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalIndexIncludesAndMergesLocalSkill(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "handmade"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "handmade", "SKILL.md"), []byte("# Handmade"), 0644); err != nil {
		t.Fatal(err)
	}
	index, err := NewLocalIndex("", []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if entries := index.List(); len(entries) != 1 || !entries[0].Local || entries[0].Name != "handmade" {
		t.Fatalf("local entries = %#v", entries)
	}
	items := []SkillSummary{{Market: MarketSkillHub, ID: "handmade", Slug: "handmade"}}
	index.Apply(items)
	if items[0].Installed == nil || !items[0].Installed.Local {
		t.Fatalf("local state was not merged: %#v", items[0])
	}
}

func TestServiceCachesSearch(t *testing.T) {
	client := &countingClient{}
	service := NewService(t.TempDir(), nil, nil, client)
	if _, err := service.Search(context.Background(), MarketSkillHub, SearchQuery{Limit: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Search(context.Background(), MarketSkillHub, SearchQuery{Limit: 1}); err != nil {
		t.Fatal(err)
	}
	if client.searches != 1 {
		t.Fatalf("searches = %d, want 1", client.searches)
	}
}

type countingClient struct{ searches int }

func (c *countingClient) Market() MarketInfo {
	return MarketInfo{ID: MarketSkillHub, Capabilities: MarketCapabilities{Search: true}}
}
func (c *countingClient) Search(context.Context, SearchQuery) (SearchPage, error) {
	c.searches++
	return SearchPage{}, nil
}
func (c *countingClient) UserSkills(context.Context, string, UserSkillsQuery) (SearchPage, error) {
	return SearchPage{}, nil
}
func (c *countingClient) Detail(context.Context, SkillID) (SkillDetail, error) {
	return SkillDetail{}, nil
}
func (c *countingClient) Files(context.Context, SkillID, string) ([]SkillFile, error) {
	return nil, nil
}
func (c *countingClient) Evaluation(context.Context, SkillID) (any, error) { return nil, nil }
func (c *countingClient) DownloadSources(SkillID, string) []DownloadSource { return nil }
func (c *countingClient) Download(context.Context, SkillID, string) (io.ReadCloser, DownloadMeta, error) {
	return nil, DownloadMeta{}, errors.New("not implemented")
}
func (c *countingClient) Categories(context.Context) ([]Category, error) { return nil, nil }
