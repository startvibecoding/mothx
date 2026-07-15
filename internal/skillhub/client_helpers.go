package skillhub

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func boundedLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func filterSkills(items []SkillSummary, query string) []SkillSummary {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return items
	}
	out := make([]SkillSummary, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), query) || strings.Contains(strings.ToLower(item.DisplayName), query) || strings.Contains(strings.ToLower(item.Description), query) {
			out = append(out, item)
		}
	}
	return out
}

func download(ctx context.Context, client *http.Client, endpoint string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", endpoint, resp.Status)
	}
	return resp.Body, nil
}
