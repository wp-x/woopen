package webdav

import (
	"io"
	"os"
	"path"
)

type noopFile struct {
	name string
	pos  int64
}

func newNoopFile(name string) *noopFile {
	return &noopFile{name: name}
}

// Close 关闭文件
func (f *noopFile) Close() error {
	return nil
}

// Read 读取文件内容（空文件）
func (f *noopFile) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

// Write 写入文件内容（丢弃）
func (f *noopFile) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	f.pos += int64(len(p))
	return len(p), nil
}

// Seek 移动文件指针
func (f *noopFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.pos = offset
	case io.SeekCurrent:
		f.pos += offset
	case io.SeekEnd:
		f.pos = offset
	default:
		return 0, os.ErrInvalid
	}
	if f.pos < 0 {
		f.pos = 0
	}
	return f.pos, nil
}

// Readdir 读取目录内容（文件不支持）
func (f *noopFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

// Stat 获取文件信息
func (f *noopFile) Stat() (os.FileInfo, error) {
	return &fileInfo{
		name:  path.Base(f.name),
		size:  0,
		mode:  0644,
		isDir: false,
	}, nil
}
