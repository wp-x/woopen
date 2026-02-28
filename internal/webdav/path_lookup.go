package webdav

import (
	"context"
	"os"
	"path"
	"strings"
	"woopen/internal/model"
)

type ResolvedPath struct {
	ID       string
	ParentID string
	Info     *model.FileInfo
}

// getFileIDByPath 根据路径获取文件ID（可用于目录或文件）
func (fs *WopanFS) getFileIDByPath(ctx context.Context, filePath string) (string, error) {
	id, _, _, err := fs.lookupPath(ctx, filePath)
	return id, err
}

func (fs *WopanFS) ResolvePath(ctx context.Context, filePath string) (ResolvedPath, error) {
	cleanPath := path.Clean(filePath)
	if cleanPath == "/" || cleanPath == "." {
		return ResolvedPath{
			ID:       "0",
			ParentID: "0",
			Info:     nil,
		}, nil
	}
	id, parentID, info, err := fs.lookupPath(ctx, cleanPath)
	if err != nil {
		return ResolvedPath{}, err
	}
	if info == nil {
		return ResolvedPath{}, os.ErrNotExist
	}
	return ResolvedPath{
		ID:       id,
		ParentID: parentID,
		Info:     info,
	}, nil
}

// lookupPath resolves a unix-style path into cloud IDs.
//
// Returns:
// - id: the object's ID (directory id or file id)
// - parentID: the parent directory ID ("0" for root)
// - info: best-effort FileInfo (nil for root)
func (fs *WopanFS) lookupPath(ctx context.Context, filePath string) (id string, parentID string, info *model.FileInfo, err error) {
	cleanPath := normalizePath(filePath)

	// 根目录
	if cleanPath == "/" {
		return "0", "0", nil, nil
	}
	if isIgnoredName(cleanPath) {
		return "", "", nil, os.ErrNotExist
	}

	if cached, ok := fs.getPathCacheEntry(cleanPath); ok {
		return cached.id, cached.parentID, cached.info, nil
	}

	// 分割路径
	parts := strings.Split(strings.Trim(cleanPath, "/"), "/")
	currentID := "0" // 从根目录开始
	currentParentID := "0"
	currentPath := ""

	// 逐级查找
	for _, part := range parts {
		if part == "" {
			continue
		}

		if currentPath == "" {
			currentPath = "/" + part
		} else {
			currentPath = currentPath + "/" + part
		}

		if cached, ok := fs.getPathCacheEntry(currentPath); ok {
			currentID = cached.id
			currentParentID = cached.parentID
			info = cached.info
			continue
		}

		child, lookupErr := fs.findChildByName(ctx, currentID, part)
		if lookupErr != nil {
			return "", "", nil, lookupErr
		}
		currentParentID = currentID
		currentID = child.ID
		info = child
		fs.setPathCacheEntry(currentPath, pathCacheEntry{
			id:       currentID,
			parentID: currentParentID,
			info:     child,
		})
	}

	fs.setPathCacheEntry(cleanPath, pathCacheEntry{
		id:       currentID,
		parentID: currentParentID,
		info:     info,
	})

	return currentID, currentParentID, info, nil
}

func (fs *WopanFS) findChildByName(ctx context.Context, dirID, name string) (*model.FileInfo, error) {
	files, err := fs.listAllInDirCached(ctx, dirID)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.Name == name {
			return f, nil
		}
	}
	return nil, os.ErrNotExist
}

func (fs *WopanFS) listAllInDir(ctx context.Context, dirID string) ([]*model.FileInfo, error) {
	var out []*model.FileInfo
	page := 1
	for {
		files, err := fs.client.ListFiles(dirID, page, listPageSize)
		if err != nil {
			return nil, err
		}
		out = append(out, files...)
		if len(files) < listPageSize {
			break
		}
		page++
	}
	return out, nil
}

func (fs *WopanFS) listAllInDirCached(ctx context.Context, dirID string) ([]*model.FileInfo, error) {
	if cached, ok := fs.getDirCacheEntry(dirID); ok {
		return cached, nil
	}
	files, err := fs.listAllInDir(ctx, dirID)
	if err != nil {
		return nil, err
	}
	fs.setDirCacheEntry(dirID, files)
	return cloneFileInfos(files), nil
}

func (fs *WopanFS) primeChildPathCache(parentPath, parentID string, files []*model.FileInfo) {
	if parentID == "" || len(files) == 0 {
		return
	}
	basePath := normalizePath(parentPath)
	for _, f := range files {
		if f == nil || f.Name == "" || isIgnoredName(f.Name) {
			continue
		}
		childPath := joinChildPath(basePath, f.Name)
		fs.setPathCacheEntry(childPath, pathCacheEntry{
			id:       f.ID,
			parentID: parentID,
			info:     f,
		})
	}
}

func joinChildPath(parent, name string) string {
	if parent == "/" {
		return "/" + name
	}
	return parent + "/" + name
}
