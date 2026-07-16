package skillhub

import (
	"sync"
	"time"
)

type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

type memoryCache struct {
	mu         sync.Mutex
	ttl        time.Duration
	pages      map[string]cacheEntry[SearchPage]
	details    map[string]cacheEntry[SkillDetail]
	categories map[string]cacheEntry[[]Category]
}

func newMemoryCache(ttl time.Duration) *memoryCache {
	return &memoryCache{ttl: ttl, pages: map[string]cacheEntry[SearchPage]{}, details: map[string]cacheEntry[SkillDetail]{}, categories: map[string]cacheEntry[[]Category]{}}
}
func (c *memoryCache) getPage(key string) (SearchPage, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.pages[key]
	if !ok || !time.Now().Before(e.expiresAt) {
		return SearchPage{}, false
	}
	return cloneSearchPage(e.value), true
}
func (c *memoryCache) setPage(key string, value SearchPage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pages[key] = cacheEntry[SearchPage]{cloneSearchPage(value), time.Now().Add(c.ttl)}
}
func (c *memoryCache) getDetail(key string) (SkillDetail, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.details[key]
	if !ok || !time.Now().Before(e.expiresAt) {
		return SkillDetail{}, false
	}
	return cloneSkillDetail(e.value), true
}
func (c *memoryCache) setDetail(key string, value SkillDetail) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.details[key] = cacheEntry[SkillDetail]{cloneSkillDetail(value), time.Now().Add(c.ttl)}
}
func (c *memoryCache) getCategories(key string) ([]Category, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.categories[key]
	if !ok || !time.Now().Before(e.expiresAt) {
		return nil, false
	}
	return cloneCategories(e.value), true
}
func (c *memoryCache) setCategories(key string, value []Category) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.categories[key] = cacheEntry[[]Category]{cloneCategories(value), time.Now().Add(c.ttl)}
}

func cloneSearchPage(page SearchPage) SearchPage {
	page.Items = append([]SkillSummary(nil), page.Items...)
	for i := range page.Items {
		page.Items[i].Tags = append([]string(nil), page.Items[i].Tags...)
	}
	return page
}
func cloneCategories(values []Category) []Category {
	out := append([]Category(nil), values...)
	for i := range out {
		out[i].Children = cloneCategories(out[i].Children)
	}
	return out
}
func cloneSkillDetail(detail SkillDetail) SkillDetail {
	detail.SkillSummary = cloneSearchPage(SearchPage{Items: []SkillSummary{detail.SkillSummary}}).Items[0]
	detail.Files = append([]SkillFile(nil), detail.Files...)
	detail.DownloadSources = append([]DownloadSource(nil), detail.DownloadSources...)
	return detail
}
func (c *memoryCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pages = map[string]cacheEntry[SearchPage]{}
	c.details = map[string]cacheEntry[SkillDetail]{}
	c.categories = map[string]cacheEntry[[]Category]{}
}
