package handler

import (
	"sync"
	"time"
	"woopen/internal/model"
)

const (
	DefaultListCacheTTL = 30 * time.Second
	listCacheRootDirID  = "0"
	minListPage         = 1
)

type ListRequest struct {
	DirID    string
	Page     int
	PageSize int
}

type ListCache interface {
	Get(req ListRequest) ([]*model.FileInfo, bool)
	Set(req ListRequest, files []*model.FileInfo)
	InvalidateDir(dirID string)
	InvalidateAll()
}

type listTTLCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[listCacheKey]listCacheEntry
}

type listCacheKey struct {
	dirID    string
	page     int
	pageSize int
}

type listCacheEntry struct {
	files   []*model.FileInfo
	expires time.Time
}

func NewListTTLCache(ttl time.Duration) ListCache {
	if ttl <= 0 {
		panic("list cache ttl must be positive")
	}
	return &listTTLCache{
		ttl:     ttl,
		entries: make(map[listCacheKey]listCacheEntry),
	}
}

func (c *listTTLCache) Get(req ListRequest) ([]*model.FileInfo, bool) {
	key := makeListCacheKey(req)
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if now.After(entry.expires) {
		c.mu.Lock()
		current, ok := c.entries[key]
		if ok && now.After(current.expires) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return nil, false
	}
	return cloneFileInfos(entry.files), true
}

func (c *listTTLCache) Set(req ListRequest, files []*model.FileInfo) {
	key := makeListCacheKey(req)
	entry := listCacheEntry{
		files:   cloneFileInfos(files),
		expires: time.Now().Add(c.ttl),
	}

	c.mu.Lock()
	c.entries[key] = entry
	c.mu.Unlock()
}

func (c *listTTLCache) InvalidateDir(dirID string) {
	normalized := normalizeDirID(dirID)

	c.mu.Lock()
	for key := range c.entries {
		if key.dirID == normalized {
			delete(c.entries, key)
		}
	}
	c.mu.Unlock()
}

func (c *listTTLCache) InvalidateAll() {
	c.mu.Lock()
	c.entries = make(map[listCacheKey]listCacheEntry)
	c.mu.Unlock()
}

func makeListCacheKey(req ListRequest) listCacheKey {
	return listCacheKey{
		dirID:    normalizeDirID(req.DirID),
		page:     normalizePage(req.Page),
		pageSize: req.PageSize,
	}
}

func normalizeDirID(dirID string) string {
	if dirID == listCacheRootDirID {
		return ""
	}
	return dirID
}

func normalizePage(page int) int {
	if page < minListPage {
		return minListPage
	}
	return page
}

func cloneFileInfos(files []*model.FileInfo) []*model.FileInfo {
	if len(files) == 0 {
		return nil
	}
	out := make([]*model.FileInfo, len(files))
	for i, f := range files {
		if f == nil {
			continue
		}
		clone := *f
		out[i] = &clone
	}
	return out
}
