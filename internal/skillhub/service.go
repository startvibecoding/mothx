package skillhub

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/startvibecoding/mothx/internal/skills"
)

type Service struct {
	clients         map[Market]MarketClient
	globalDir       string
	projectDirs     []string
	officialHandles []string
}

func NewService(globalDir string, projectDirs []string, officialHandles []string, clients ...MarketClient) *Service {
	if len(clients) == 0 {
		clients = []MarketClient{NewSkillHubClient("", nil), NewClawHubClient("", nil)}
	}
	indexed := make(map[Market]MarketClient, len(clients))
	for _, client := range clients {
		if client != nil {
			indexed[client.Market().ID] = client
		}
	}
	return &Service{clients: indexed, globalDir: globalDir, projectDirs: projectDirs, officialHandles: officialHandles}
}

func NewServiceForWorkDir(globalDir, workDir string, officialHandles []string, clients ...MarketClient) *Service {
	return NewService(globalDir, skills.ProjectSkillDirs(workDir), officialHandles, clients...)
}
func (s *Service) Markets() []MarketInfo {
	result := make([]MarketInfo, 0, len(s.clients))
	for _, client := range s.clients {
		result = append(result, client.Market())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
func (s *Service) Search(ctx context.Context, market Market, query SearchQuery) (SearchPage, error) {
	client, err := s.client(market)
	if err != nil {
		return SearchPage{}, err
	}
	page, err := client.Search(ctx, query)
	if err != nil {
		return SearchPage{}, err
	}
	s.applyInstalled(page.Items)
	return page, nil
}
func (s *Service) Categories(ctx context.Context, market Market) ([]Category, error) {
	client, err := s.client(market)
	if err != nil {
		return nil, err
	}
	if !client.Market().Capabilities.Categories {
		return nil, nil
	}
	return client.Categories(ctx)
}
func (s *Service) Detail(ctx context.Context, market Market, id string) (SkillDetail, error) {
	client, err := s.client(market)
	if err != nil {
		return SkillDetail{}, err
	}
	detail, err := client.Detail(ctx, SkillID{Market: market, ID: id})
	if err != nil {
		return SkillDetail{}, err
	}
	if client.Market().Capabilities.FileList {
		if files, filesErr := client.Files(ctx, SkillID{Market: market, ID: id}, detail.Version); filesErr == nil {
			detail.Files = files
		}
	}
	if client.Market().Capabilities.Evaluation {
		if evaluation, evaluationErr := client.Evaluation(ctx, SkillID{Market: market, ID: id}); evaluationErr == nil {
			detail.Evaluation = evaluation
		}
	}
	detail.DownloadSources = client.DownloadSources(SkillID{Market: market, ID: id}, detail.Version)
	detail.Installed = s.state(market, id)
	if detail.Installed != nil {
		detail.Installed.UpdateAvailable = versionsDiffer(detail.Installed.Version, detail.Version)
	}
	return detail, nil
}

func versionsDiffer(installed, remote string) bool {
	return installed != "" && remote != "" && installed != remote
}
func (s *Service) Official(ctx context.Context, query UserSkillsQuery) (SearchPage, error) {
	client, err := s.client(MarketSkillHub)
	if err != nil {
		return SearchPage{}, err
	}
	pageSize := boundedLimit(query.Limit)
	pageNo := query.Page
	if pageNo < 1 {
		pageNo = 1
	}
	allItems := make([]SkillSummary, 0)
	seen := make(map[string]bool)
	for _, handle := range s.officialHandles {
		for remotePage := 1; ; remotePage++ {
			page, err := client.UserSkills(ctx, handle, UserSkillsQuery{Query: query.Query, Limit: 100, Page: remotePage})
			if err != nil {
				return SearchPage{}, fmt.Errorf("official handle %q: %w", handle, err)
			}
			for _, item := range page.Items {
				key := item.ID
				if !seen[key] {
					seen[key] = true
					allItems = append(allItems, item)
				}
			}
			if int64(remotePage*100) >= page.Total {
				break
			}
		}
	}
	sort.SliceStable(allItems, func(i, j int) bool { return allItems[i].Downloads > allItems[j].Downloads })
	start := (pageNo - 1) * pageSize
	if start > len(allItems) {
		start = len(allItems)
	}
	end := start + pageSize
	if end > len(allItems) {
		end = len(allItems)
	}
	result := SearchPage{Items: allItems[start:end], Total: int64(len(allItems)), Page: pageNo, PageSize: pageSize}
	s.applyInstalled(result.Items)
	return result, nil
}
func (s *Service) Install(ctx context.Context, request InstallRequest) (InstallResult, error) {
	if request.Market == "" {
		request.Market = MarketSkillHub
	}
	client, err := s.client(request.Market)
	if err != nil {
		return InstallResult{}, err
	}
	if request.TargetDir == "" {
		if request.Scope == "global" {
			request.TargetDir = s.globalDir
		} else if len(s.projectDirs) > 0 {
			request.TargetDir = s.projectDirs[0]
		} else {
			return InstallResult{}, errors.New("project skills directory is unavailable")
		}
	}
	return Install(ctx, client, request)
}
func (s *Service) client(market Market) (MarketClient, error) {
	client := s.clients[market]
	if client == nil {
		return nil, fmt.Errorf("unsupported skill market %q", market)
	}
	return client, nil
}
func (s *Service) state(market Market, id string) *InstalledState {
	index, err := NewLocalIndex(s.globalDir, s.projectDirs)
	if err != nil {
		return nil
	}
	return index.State(market, id)
}
func (s *Service) applyInstalled(items []SkillSummary) {
	index, err := NewLocalIndex(s.globalDir, s.projectDirs)
	if err == nil {
		index.Apply(items)
	}
}
