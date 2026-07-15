package handler

import (
	"sync"
	"time"
)

// DefaultDirectURLCacheTTL 直链缓存有效期。
// 联通 CDN 签名直链本身会过期，此处需设为「实测有效期的 70%」。
// 阶段0 实测方法见 docs/plan-direct-link.md；未实测前取保守值 3 分钟。
// ponytail: 常量即可，实测后改这里；真需要热更新再挪进 settings。
const DefaultDirectURLCacheTTL = 3 * time.Minute

// DirectURLCache 按云盘文件ID缓存联通签名直链，降低高频外链对上游API的调用。
type DirectURLCache interface {
	Get(fileID string) (string, bool)
	Set(fileID, url string)
}

type directTTLCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]directCacheEntry
}

type directCacheEntry struct {
	url     string
	expires time.Time
}

// NewDirectURLCache 创建直链缓存；ttl<=0 时恒 miss（不缓存）。
func NewDirectURLCache(ttl time.Duration) DirectURLCache {
	return &directTTLCache{
		ttl:     ttl,
		entries: make(map[string]directCacheEntry),
	}
}

func (c *directTTLCache) Get(fileID string) (string, bool) {
	if c.ttl <= 0 {
		return "", false
	}
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.entries[fileID]
	c.mu.RUnlock()
	if !ok {
		return "", false
	}
	if now.After(entry.expires) {
		c.mu.Lock()
		if current, ok := c.entries[fileID]; ok && now.After(current.expires) {
			delete(c.entries, fileID)
		}
		c.mu.Unlock()
		return "", false
	}
	return entry.url, true
}

func (c *directTTLCache) Set(fileID, url string) {
	if c.ttl <= 0 {
		return
	}
	c.mu.Lock()
	c.entries[fileID] = directCacheEntry{url: url, expires: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}
