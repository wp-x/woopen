package webdav

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

// wopanFile 实现 webdav.File 接口（文件）
type wopanFile struct {
	fs       *WopanFS
	name     string
	fileID   string
	parentID string
	fileName string
	flag     int
	isNew    bool
	fileInfo os.FileInfo
	reader   io.ReadCloser

	// overwriteFileID is set when this handle represents an overwrite operation
	// (write to an existing file). We upload a new object and then delete the old one.
	overwriteFileID string

	// read position (for Range requests)
	position int64

	// tmp is used to spool writes to disk to avoid holding the entire content in memory.
	tmp      *os.File
	tmpPath  string
	writeLen int64
}

// Close 关闭文件
func (f *wopanFile) Close() error {
	var closeErr error
	if f.reader != nil {
		closeErr = f.reader.Close()
		f.reader = nil
	}

	// If this is a write handle, upload on Close (including 0-byte files).
	if f.isNew && f.tmp != nil {
		if err := f.upload(); err != nil {
			return err
		}
		return closeErr
	}

	// If a client created a 0-byte file without writing any bytes, still honor creation.
	if f.isNew && f.tmp == nil {
		// Create a temporary empty file so UploadFile receives a ReadSeeker.
		if err := f.initTmp(); err != nil {
			return err
		}
		if err := f.upload(); err != nil {
			return err
		}
		return closeErr
	}

	return closeErr
}

// Read 读取文件内容
func (f *wopanFile) Read(p []byte) (n int, err error) {
	if f.reader == nil {
		// 获取下载链接并创建 reader（走缓存 + singleflight）
		downloadURL, err := f.fs.DownloadURL(f.fileID)
		if err != nil {
			return 0, err
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL, nil)
		if err != nil {
			return 0, err
		}
		if f.position > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", f.position))
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, err
		}

		// Accept 200 (no range) and 206 (partial content)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			return 0, fmt.Errorf("下载失败: %s", resp.Status)
		}

		f.reader = resp.Body
	}

	n, err = f.reader.Read(p)
	f.position += int64(n)
	return n, err
}

// Write 写入文件内容
func (f *wopanFile) Write(p []byte) (n int, err error) {
	if err := f.initTmp(); err != nil {
		return 0, err
	}
	n, err = f.tmp.Write(p)
	f.writeLen += int64(n)
	return n, err
}

// Seek 移动文件指针
func (f *wopanFile) Seek(offset int64, whence int) (int64, error) {
	// Read seek: adjust range position and reopen reader on next Read.
	// Write seek is not supported; most WebDAV clients do sequential writes.
	if f.tmp != nil {
		return 0, fmt.Errorf("Seek for write not supported")
	}

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = f.position + offset
	case io.SeekEnd:
		// Unknown without a HEAD call; best effort is to disallow.
		return 0, fmt.Errorf("SeekEnd not supported")
	default:
		return 0, fmt.Errorf("invalid whence")
	}
	if newPos < 0 {
		newPos = 0
	}

	f.position = newPos
	if f.reader != nil {
		_ = f.reader.Close()
		f.reader = nil
	}
	return f.position, nil
}

// Readdir 读取目录内容（文件不支持）
func (f *wopanFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

// Stat 获取文件信息
func (f *wopanFile) Stat() (os.FileInfo, error) {
	if f.fileInfo != nil {
		return f.fileInfo, nil
	}

	// 对于新文件，返回临时信息
	if f.isNew {
		size := f.writeLen
		return &fileInfo{
			name:    f.fileName,
			size:    size,
			mode:    0644,
			isDir:   false,
		}, nil
	}

	return nil, os.ErrNotExist
}

// upload 上传文件到云盘
func (f *wopanFile) upload() error {
	defer func() {
		if f.tmp != nil {
			_ = f.tmp.Close()
		}
		if f.tmpPath != "" {
			_ = os.Remove(f.tmpPath)
		}
	}()

	if f.tmp == nil {
		return fmt.Errorf("no temp file")
	}
	if _, err := f.tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// For overwrites, delete the old file after the new upload succeeds.
	_, err := f.fs.client.UploadFile(
		context.Background(),
		f.parentID,
		f.fileName,
		"application/octet-stream",
		f.writeLen,
		f.tmp,
		nil,
	)

	if err != nil {
		return err
	}
	if f.parentID != "" {
		f.fs.invalidateDirCache(f.parentID)
	}
	f.fs.invalidatePathCache(f.name)
	if f.overwriteFileID != "" {
		_ = f.fs.client.Delete(context.Background(), false, f.overwriteFileID)
	}
	return nil
}

func (f *wopanFile) initTmp() error {
	if f.tmp != nil {
		return nil
	}
	tmp, err := os.CreateTemp("", "woopen-webdav-*"+filepath.Ext(f.fileName))
	if err != nil {
		return err
	}
	f.tmp = tmp
	f.tmpPath = tmp.Name()
	return nil
}

// wopanDir 实现 webdav.File 接口（目录）
type wopanDir struct {
	name     string
	modTime  time.Time
	files    []os.FileInfo
	position int
}

// Close 关闭目录
func (d *wopanDir) Close() error {
	return nil
}

// Read 读取目录内容（目录不支持）
func (d *wopanDir) Read(p []byte) (n int, err error) {
	return 0, os.ErrInvalid
}

// Write 写入目录内容（目录不支持）
func (d *wopanDir) Write(p []byte) (n int, err error) {
	return 0, os.ErrInvalid
}

// Seek 移动目录指针
func (d *wopanDir) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		d.position = int(offset)
	case io.SeekCurrent:
		d.position += int(offset)
	case io.SeekEnd:
		d.position = len(d.files) + int(offset)
	}

	if d.position < 0 {
		d.position = 0
	}
	if d.position > len(d.files) {
		d.position = len(d.files)
	}

	return int64(d.position), nil
}

// Readdir 读取目录内容
func (d *wopanDir) Readdir(count int) ([]os.FileInfo, error) {
	if d.position >= len(d.files) {
		if count <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}

	if count <= 0 {
		// 返回所有剩余文件
		files := d.files[d.position:]
		d.position = len(d.files)
		return files, nil
	}

	// 返回指定数量的文件
	end := d.position + count
	if end > len(d.files) {
		end = len(d.files)
	}

	files := d.files[d.position:end]
	d.position = end

	if d.position >= len(d.files) {
		return files, io.EOF
	}

	return files, nil
}

// Stat 获取目录信息
func (d *wopanDir) Stat() (os.FileInfo, error) {
	modTime := d.modTime
	if modTime.IsZero() {
		modTime = time.Now()
	}
	return &fileInfo{
		name:    path.Base(d.name),
		size:    0,
		mode:    os.ModeDir | 0755,
		modTime: modTime,
		isDir:   true,
	}, nil
}
