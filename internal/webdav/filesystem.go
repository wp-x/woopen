package webdav

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"
	"woopen/internal/wopan"

	"golang.org/x/net/webdav"
)

// WopanFS 实现 webdav.FileSystem 接口
type WopanFS struct {
	client *wopan.Client
}

// NewWopanFS 创建新的 WebDAV 文件系统
func NewWopanFS(client *wopan.Client) *WopanFS {
	return &WopanFS{
		client: client,
	}
}

// Mkdir 创建目录
func (fs *WopanFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	name = path.Clean(name)
	if name == "/" || name == "." {
		return os.ErrExist
	}

	parentPath := path.Dir(name)
	dirName := path.Base(name)

	// 获取父目录ID
	parentID, err := fs.getFileIDByPath(ctx, parentPath)
	if err != nil {
		return err
	}

	_, err = fs.client.CreateDirectory(parentID, dirName)
	return err
}

// OpenFile 打开文件
func (fs *WopanFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	name = path.Clean(name)

	// 获取文件信息
	fileInfo, err := fs.Stat(ctx, name)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// 如果文件不存在且需要创建
	if os.IsNotExist(err) && (flag&os.O_CREATE) != 0 {
		return fs.createFile(ctx, name, flag, perm)
	}

	if os.IsNotExist(err) {
		return nil, err
	}

	// 打开已存在的文件
	return fs.openExistingFile(ctx, name, fileInfo, flag)
}

// RemoveAll 删除文件或目录
func (fs *WopanFS) RemoveAll(ctx context.Context, name string) error {
	name = path.Clean(name)
	if name == "/" || name == "." {
		return fmt.Errorf("refuse to delete root")
	}

	_, _, info, err := fs.lookupPath(ctx, name)
	if err != nil {
		return err
	}
	if info == nil {
		return os.ErrNotExist
	}
	return fs.client.Delete(ctx, info.IsDir, info.ID)
}

// Rename 重命名文件或目录
func (fs *WopanFS) Rename(ctx context.Context, oldName, newName string) error {
	oldName = path.Clean(oldName)
	newName = path.Clean(newName)
	if oldName == "/" || oldName == "." || newName == "/" || newName == "." {
		return fmt.Errorf("invalid rename")
	}
	if err := validateTargetParent(newName); err != nil {
		return err
	}

	// Do not overwrite existing targets implicitly.
	if _, _, _, err := fs.lookupPath(ctx, newName); err == nil {
		return os.ErrExist
	}

	id, _, info, err := fs.lookupPath(ctx, oldName)
	if err != nil {
		return err
	}
	if info == nil {
		return os.ErrNotExist
	}

	oldBase := path.Base(oldName)
	newBase := path.Base(newName)
	newParentPath := path.Dir(newName)
	newParentID, err := fs.getFileIDByPath(ctx, newParentPath)
	if err != nil {
		return err
	}

	// If parent differs, move first.
	oldParentPath := path.Dir(oldName)
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
		return fs.client.Rename(ctx, info.IsDir, id, newBase)
	}
	return nil
}

// Stat 获取文件信息
func (fs *WopanFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	name = path.Clean(name)

	// 根目录
	if name == "/" || name == "." {
		return &fileInfo{
			name:    "/",
			size:    0,
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
			isDir:   true,
		}, nil
	}

	// 先确认该路径存在（同时可区分 NotExist）
	_, _, info, err := fs.lookupPath(ctx, name)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, os.ErrNotExist
	}

	// 获取父目录的内容再匹配目标（client.ListFiles 期望的是目录 ID，而不是路径）
	parentID, err := fs.getFileIDByPath(ctx, path.Dir(name))
	if err != nil {
		return nil, err
	}

	fileName := path.Base(name)
	f, err := fs.findChildByName(ctx, parentID, fileName)
	if err != nil {
		return nil, err
	}
	return &fileInfo{
		name:    f.Name,
		size:    f.Size,
		mode:    fs.getFileMode(f),
		modTime: f.ModTime,
		isDir:   f.IsDir,
	}, nil
}
