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

type createFileParams struct {
	name     string
	flag     int
	perm     os.FileMode
	parentID string
}

// createFile 创建新文件
func (fs *WopanFS) createFile(ctx context.Context, params createFileParams) (webdav.File, error) {
	cleanName := path.Clean(params.name)
	if isIgnoredName(cleanName) {
		return newNoopFile(cleanName), nil
	}
	fileName := path.Base(cleanName)
	parentID := params.parentID
	if parentID == "" {
		parentPath := path.Dir(cleanName)
		id, err := fs.getFileIDByPath(ctx, parentPath)
		if err != nil {
			return nil, err
		}
		parentID = id
	}

	return &wopanFile{
		fs:       fs,
		name:     cleanName,
		parentID: parentID,
		fileName: fileName,
		flag:     params.flag,
		isNew:    true,
		// Use temp file spooling for WebDAV writes; avoid holding everything in memory.
	}, nil
}

type openExistingFileParams struct {
	name     string
	fileID   string
	parentID string
	info     os.FileInfo
	flag     int
}

// openExistingFile 打开已存在的文件
func (fs *WopanFS) openExistingFile(params openExistingFileParams) (webdav.File, error) {
	if params.info == nil {
		return nil, os.ErrNotExist
	}
	if isIgnoredName(params.name) {
		return newNoopFile(params.name), nil
	}
	if params.info.IsDir() {
		return nil, os.ErrInvalid
	}

	// If opened for writing, implement overwrite by uploading a new file and deleting the old one.
	if (params.flag&os.O_WRONLY) != 0 || (params.flag&os.O_RDWR) != 0 || (params.flag&os.O_TRUNC) != 0 {
		return &wopanFile{
			fs:              fs,
			name:            params.name,
			fileID:          params.fileID,
			overwriteFileID: params.fileID,
			parentID:        params.parentID,
			fileName:        path.Base(params.name),
			flag:            params.flag,
			isNew:           true, // treat as "new upload on Close"
			fileInfo:        params.info,
		}, nil
	}

	return &wopanFile{
		fs:       fs,
		name:     params.name,
		fileID:   params.fileID,
		flag:     params.flag,
		isNew:    false,
		fileInfo: params.info,
	}, nil
}

// openDirectory 打开目录
func (fs *WopanFS) openDirectory(ctx context.Context, name string) (webdav.File, error) {
	dirID, err := fs.getFileIDByPath(ctx, name)
	if err != nil {
		return nil, err
	}

	files, err := fs.listAllInDirCached(ctx, dirID)
	if err != nil {
		return nil, err
	}
	fs.primeChildPathCache(name, dirID, files)

	now := time.Now()
	fileInfos := make([]os.FileInfo, 0, len(files))
	for _, f := range files {
		if f == nil || isIgnoredName(f.Name) {
			continue
		}
		modTime := f.ModTime
		if modTime.IsZero() {
			modTime = now
		}
		fileInfos = append(fileInfos, &fileInfo{
			name:    f.Name,
			size:    f.Size,
			mode:    fs.getFileMode(f),
			modTime: modTime,
			isDir:   f.IsDir,
		})
	}

	return &wopanDir{
		name:    name,
		modTime: now,
		files:   fileInfos,
	}, nil
}

// getFileMode 获取文件模式
func (fs *WopanFS) getFileMode(f *model.FileInfo) os.FileMode {
	if f.IsDir {
		return os.ModeDir | 0755
	}
	return 0644
}

func (fs *WopanFS) fileInfoFromModel(info *model.FileInfo) os.FileInfo {
	if info == nil {
		return nil
	}
	modTime := info.ModTime
	if modTime.IsZero() {
		modTime = time.Now()
	}
	return &fileInfo{
		name:    info.Name,
		size:    info.Size,
		mode:    fs.getFileMode(info),
		modTime: modTime,
		isDir:   info.IsDir,
	}
}

func isIgnoredName(name string) bool {
	base := path.Base(name)
	if base == ".DS_Store" || base == ".qspacesettings" {
		return true
	}
	return strings.HasPrefix(base, "._")
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
