package webdav

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"
	"woopen/internal/model"

	"golang.org/x/net/webdav"
)

const listPageSize = 200

// getFileIDByPath 根据路径获取文件ID（可用于目录或文件）
func (fs *WopanFS) getFileIDByPath(ctx context.Context, filePath string) (string, error) {
	id, _, _, err := fs.lookupPath(ctx, filePath)
	return id, err
}

// lookupPath resolves a unix-style path into cloud IDs.
//
// Returns:
// - id: the object's ID (directory id or file id)
// - parentID: the parent directory ID ("0" for root)
// - info: best-effort FileInfo (nil for root)
func (fs *WopanFS) lookupPath(ctx context.Context, filePath string) (id string, parentID string, info *model.FileInfo, err error) {
	filePath = path.Clean(filePath)

	// 根目录
	if filePath == "/" || filePath == "." {
		return "0", "0", nil, nil
	}

	// 分割路径
	parts := strings.Split(strings.Trim(filePath, "/"), "/")
	currentID := "0" // 从根目录开始
	currentParentID := "0"

	// 逐级查找
	for _, part := range parts {
		if part == "" {
			continue
		}

		child, err := fs.findChildByName(ctx, currentID, part)
		if err != nil {
			return "", "", nil, err
		}
		currentParentID = currentID
		currentID = child.ID
		info = child
	}

	return currentID, currentParentID, info, nil
}

func (fs *WopanFS) findChildByName(ctx context.Context, dirID, name string) (*model.FileInfo, error) {
	// Iterate pages to avoid "not found" when dir has > page size entries.
	page := 1
	for {
		files, err := fs.client.ListFiles(dirID, page, listPageSize)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if f.Name == name {
				return f, nil
			}
		}
		if len(files) < listPageSize {
			break
		}
		page++
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

// createFile 创建新文件
func (fs *WopanFS) createFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	parentPath := path.Dir(name)
	fileName := path.Base(name)

	// 获取父目录ID
	parentID, err := fs.getFileIDByPath(ctx, parentPath)
	if err != nil {
		return nil, err
	}

	return &wopanFile{
		fs:       fs,
		name:     name,
		parentID: parentID,
		fileName: fileName,
		flag:     flag,
		isNew:    true,
		// Use temp file spooling for WebDAV writes; avoid holding everything in memory.
	}, nil
}

// openExistingFile 打开已存在的文件
func (fs *WopanFS) openExistingFile(ctx context.Context, name string, info os.FileInfo, flag int) (webdav.File, error) {
	if info.IsDir() {
		return fs.openDirectory(ctx, name)
	}

	fileID, parentID, _, err := fs.lookupPath(ctx, name)
	if err != nil {
		return nil, err
	}

	// If opened for writing, implement overwrite by uploading a new file and deleting the old one.
	if (flag&os.O_WRONLY) != 0 || (flag&os.O_RDWR) != 0 || (flag&os.O_TRUNC) != 0 {
		return &wopanFile{
			fs:              fs,
			name:            name,
			fileID:          fileID,
			overwriteFileID: fileID,
			parentID:        parentID,
			fileName:        path.Base(name),
			flag:            flag,
			isNew:           true, // treat as "new upload on Close"
			fileInfo:        info,
		}, nil
	}

	return &wopanFile{
		fs:       fs,
		name:     name,
		fileID:   fileID,
		flag:     flag,
		isNew:    false,
		fileInfo: info,
	}, nil
}

// openDirectory 打开目录
func (fs *WopanFS) openDirectory(ctx context.Context, name string) (webdav.File, error) {
	dirID, err := fs.getFileIDByPath(ctx, name)
	if err != nil {
		return nil, err
	}

	files, err := fs.listAllInDir(ctx, dirID)
	if err != nil {
		return nil, err
	}

	// 转换为 os.FileInfo
	fileInfos := make([]os.FileInfo, 0, len(files))
	for _, f := range files {
		fileInfos = append(fileInfos, &fileInfo{
			name:    f.Name,
			size:    f.Size,
			mode:    fs.getFileMode(f),
			modTime: f.ModTime,
			isDir:   f.IsDir,
		})
	}

	return &wopanDir{
		name:     name,
		files:    fileInfos,
		position: 0,
	}, nil
}

// getFileMode 获取文件模式
func (fs *WopanFS) getFileMode(f *model.FileInfo) os.FileMode {
	if f.IsDir {
		return os.ModeDir | 0755
	}
	return 0644
}

// fileInfo 实现 os.FileInfo 接口
type fileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (fi *fileInfo) Name() string       { return fi.name }
func (fi *fileInfo) Size() int64        { return fi.size }
func (fi *fileInfo) Mode() os.FileMode  { return fi.mode }
func (fi *fileInfo) ModTime() time.Time { return fi.modTime }
func (fi *fileInfo) IsDir() bool        { return fi.isDir }
func (fi *fileInfo) Sys() interface{}   { return nil }

// memBuffer 内存缓冲区
type memBuffer struct {
	data []byte
}

func (b *memBuffer) Write(p []byte) (n int, err error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *memBuffer) Bytes() []byte {
	return b.data
}

// validateTargetParent ensures WebDAV rename/move doesn't try to create nested invalid targets.
func validateTargetParent(p string) error {
	p = path.Clean(p)
	if p == "" || p == "." {
		return fmt.Errorf("invalid path")
	}
	return nil
}
