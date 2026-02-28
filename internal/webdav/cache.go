package webdav

import (
	"path"
	"strings"
	"time"
	"woopen/internal/model"
)

const (
	listPageSize = 200
	cacheTTL     = 45 * time.Second
)

type dirCacheEntry struct {
	files   []*model.FileInfo
	expires time.Time
}

type pathCacheEntry struct {
	id       string
	parentID string
	info     *model.FileInfo
	expires  time.Time
}

func normalizePath(p string) string {
	clean := path.Clean(p)
	if clean == "" || clean == "." {
		return "/"
	}
	if !strings.HasPrefix(clean, "/") {
		return "/" + clean
	}
	return clean
}

func cloneFileInfos(files []*model.FileInfo) []*model.FileInfo {
	if len(files) == 0 {
		return nil
	}
	out := make([]*model.FileInfo, len(files))
	copy(out, files)
	return out
}

func (fs *WopanFS) getPathCacheEntry(p string) (*pathCacheEntry, bool) {
	key := normalizePath(p)
	now := time.Now()

	fs.pathCacheMu.RLock()
	entry, ok := fs.pathCache[key]
	fs.pathCacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	if now.After(entry.expires) {
		fs.pathCacheMu.Lock()
		delete(fs.pathCache, key)
		fs.pathCacheMu.Unlock()
		return nil, false
	}
	return entry, true
}

func (fs *WopanFS) setPathCacheEntry(p string, entry pathCacheEntry) {
	key := normalizePath(p)
	clone := entry
	clone.expires = time.Now().Add(cacheTTL)
	fs.pathCacheMu.Lock()
	fs.pathCache[key] = &clone
	fs.pathCacheMu.Unlock()
}

func (fs *WopanFS) invalidatePathCache(p string) {
	key := normalizePath(p)
	fs.pathCacheMu.Lock()
	delete(fs.pathCache, key)
	fs.pathCacheMu.Unlock()
}

func (fs *WopanFS) invalidatePathPrefix(prefix string) {
	key := normalizePath(prefix)
	if key == "/" {
		fs.pathCacheMu.Lock()
		fs.pathCache = make(map[string]*pathCacheEntry)
		fs.pathCacheMu.Unlock()
		return
	}
	matchPrefix := key
	if !strings.HasSuffix(matchPrefix, "/") {
		matchPrefix += "/"
	}
	fs.pathCacheMu.Lock()
	for k := range fs.pathCache {
		if k == key || strings.HasPrefix(k, matchPrefix) {
			delete(fs.pathCache, k)
		}
	}
	fs.pathCacheMu.Unlock()
}

func (fs *WopanFS) getDirCacheEntry(dirID string) ([]*model.FileInfo, bool) {
	now := time.Now()
	fs.dirCacheMu.RLock()
	entry, ok := fs.dirCache[dirID]
	fs.dirCacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	if now.After(entry.expires) {
		fs.dirCacheMu.Lock()
		delete(fs.dirCache, dirID)
		fs.dirCacheMu.Unlock()
		return nil, false
	}
	return cloneFileInfos(entry.files), true
}

func (fs *WopanFS) setDirCacheEntry(dirID string, files []*model.FileInfo) {
	fs.dirCacheMu.Lock()
	fs.dirCache[dirID] = &dirCacheEntry{
		files:   cloneFileInfos(files),
		expires: time.Now().Add(cacheTTL),
	}
	fs.dirCacheMu.Unlock()
}

func (fs *WopanFS) invalidateDirCache(dirID string) {
	fs.dirCacheMu.Lock()
	delete(fs.dirCache, dirID)
	fs.dirCacheMu.Unlock()
}
