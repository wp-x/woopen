package webdav

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"time"
	"woopen/internal/wopan"

	"golang.org/x/net/webdav"
)

// WopanFS 实现 webdav.FileSystem 接口
type WopanFS struct {
	client *wopan.Client

	dirCacheMu  sync.RWMutex
	dirCache    map[string]*dirCacheEntry
	pathCacheMu sync.RWMutex
	pathCache   map[string]*pathCacheEntry

	// singleflight 合并并发的相同上游请求，避免缓存击穿把联通 API 打爆。
	listSF singleGroup
	urlSF  singleGroup

	// 下载链接缓存：联通链接有一定有效期，短 TTL 复用可大幅减少 GetDownloadURL 调用。
	urlCacheMu sync.RWMutex
	urlCache   map[string]urlCacheEntry
}

// NewWopanFS 创建新的 WebDAV 文件系统
func NewWopanFS(client *wopan.Client) *WopanFS {
	return &WopanFS{
		client:    client,
		dirCache:  make(map[string]*dirCacheEntry),
		pathCache: make(map[string]*pathCacheEntry),
		urlCache:  make(map[string]urlCacheEntry),
	}
}

type urlCacheEntry struct {
	url     string
	expires time.Time
}

// urlCacheTTL 保守取值，远小于联通下载链接实际有效期，避免返回过期链接。
const urlCacheTTL = 8 * time.Minute

// DownloadURL 返回文件下载直链，带缓存 + singleflight 去重。
// WebDAV 客户端会对同一文件密集发起 HEAD/GET，直连每次都锁 metaMu 打上游，
// 这里合并并短时缓存，是挂载不卡的关键。
func (fs *WopanFS) DownloadURL(fileID string) (string, error) {
	fs.urlCacheMu.RLock()
	e, ok := fs.urlCache[fileID]
	fs.urlCacheMu.RUnlock()
	if ok && time.Now().Before(e.expires) {
		return e.url, nil
	}

	v, err, _ := fs.urlSF.Do(fileID, func() (interface{}, error) {
		url, err := fs.client.GetDownloadURL(fileID)
		if err != nil {
			return "", err
		}
		fs.urlCacheMu.Lock()
		fs.urlCache[fileID] = urlCacheEntry{url: url, expires: time.Now().Add(urlCacheTTL)}
		fs.urlCacheMu.Unlock()
		return url, nil
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

// Mkdir 创建目录
func (fs *WopanFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	cleanName := path.Clean(name)
	if cleanName == "/" || cleanName == "." {
		return os.ErrExist
	}

	parentPath := path.Dir(cleanName)
	dirName := path.Base(cleanName)

	// 获取父目录ID
	parentID, err := fs.getFileIDByPath(ctx, parentPath)
	if err != nil {
		return err
	}

	_, err = fs.client.CreateDirectory(parentID, dirName)
	if err != nil {
		return err
	}
	fs.invalidateDirCache(parentID)
	fs.invalidatePathCache(cleanName)
	return nil
}

// OpenFile 打开文件
func (fs *WopanFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	cleanName := path.Clean(name)

	if cleanName == "/" || cleanName == "." {
		return fs.openDirectory(ctx, cleanName)
	}

	id, parentID, info, err := fs.lookupPath(ctx, cleanName)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if os.IsNotExist(err) || info == nil {
		if (flag & os.O_CREATE) == 0 {
			return nil, os.ErrNotExist
		}
		parentPath := path.Dir(cleanName)
		newParentID, parentErr := fs.getFileIDByPath(ctx, parentPath)
		if parentErr != nil {
			return nil, parentErr
		}
		return fs.createFile(ctx, createFileParams{
			name:     cleanName,
			flag:     flag,
			perm:     perm,
			parentID: newParentID,
		})
	}

	if info.IsDir {
		return fs.openDirectory(ctx, cleanName)
	}

	return fs.openExistingFile(openExistingFileParams{
		name:     cleanName,
		fileID:   id,
		parentID: parentID,
		info:     fs.fileInfoFromModel(info),
		flag:     flag,
	})
}

// RemoveAll 删除文件或目录
func (fs *WopanFS) RemoveAll(ctx context.Context, name string) error {
	cleanName := path.Clean(name)
	if cleanName == "/" || cleanName == "." {
		return fmt.Errorf("refuse to delete root")
	}

	_, parentID, info, err := fs.lookupPath(ctx, cleanName)
	if err != nil {
		return err
	}
	if info == nil {
		return os.ErrNotExist
	}
	if err := fs.client.Delete(ctx, info.IsDir, info.ID); err != nil {
		return err
	}
	if parentID != "" {
		fs.invalidateDirCache(parentID)
	}
	if info.IsDir {
		fs.invalidatePathPrefix(cleanName)
		return nil
	}
	fs.invalidatePathCache(cleanName)
	return nil
}

// Rename 重命名文件或目录
func (fs *WopanFS) Rename(ctx context.Context, oldName, newName string) error {
	cleanOld := path.Clean(oldName)
	cleanNew := path.Clean(newName)
	if cleanOld == "/" || cleanOld == "." || cleanNew == "/" || cleanNew == "." {
		return fmt.Errorf("invalid rename")
	}
	if err := validateTargetParent(cleanNew); err != nil {
		return err
	}

	// Do not overwrite existing targets implicitly.
	if _, _, _, err := fs.lookupPath(ctx, cleanNew); err == nil {
		return os.ErrExist
	}

	id, _, info, err := fs.lookupPath(ctx, cleanOld)
	if err != nil {
		return err
	}
	if info == nil {
		return os.ErrNotExist
	}

	oldBase := path.Base(cleanOld)
	newBase := path.Base(cleanNew)
	newParentPath := path.Dir(cleanNew)
	newParentID, err := fs.getFileIDByPath(ctx, newParentPath)
	if err != nil {
		return err
	}

	// If parent differs, move first.
	oldParentPath := path.Dir(cleanOld)
	oldParentID, err := fs.getFileIDByPath(ctx, oldParentPath)
	if err != nil {
		return err
	}

	if newParentID != oldParentID {
		if err := fs.client.Move(ctx, info.IsDir, id, newParentID); err != nil {
			return err
		}
	}

	// Then rename if needed.
	if oldBase != newBase {
		if err := fs.client.Rename(ctx, info.IsDir, id, newBase); err != nil {
			return err
		}
	}

	if oldParentID != "" {
		fs.invalidateDirCache(oldParentID)
	}
	if newParentID != "" {
		fs.invalidateDirCache(newParentID)
	}
	if info.IsDir {
		fs.invalidatePathPrefix(cleanOld)
		fs.invalidatePathPrefix(cleanNew)
		return nil
	}
	fs.invalidatePathCache(cleanOld)
	fs.invalidatePathCache(cleanNew)
	return nil
}

// Stat 获取文件信息
func (fs *WopanFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	cleanName := path.Clean(name)

	// 根目录
	if cleanName == "/" || cleanName == "." {
		return &fileInfo{
			name:    "/",
			size:    0,
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
			isDir:   true,
		}, nil
	}

	// 先确认该路径存在（同时可区分 NotExist）
	_, _, info, err := fs.lookupPath(ctx, cleanName)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, os.ErrNotExist
	}

	return fs.fileInfoFromModel(info), nil
}
