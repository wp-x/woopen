package webdav

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"woopen/internal/model"
)

func TestReadDirectoryPagesIncludesSyncDirectoryAfterFirstPage(t *testing.T) {
	const openListPageSize = 100
	if listPageSize != openListPageSize {
		t.Fatalf("WebDAV page size = %d, want %d", listPageSize, openListPageSize)
	}
	for _, name := range []string{"我的同步盘", "同步盘", "自定义同步目录"} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			read := func(_ context.Context, id string, page int) ([]*model.FileInfo, error) {
				calls++
				if id != "account-root" {
					t.Fatalf("directory = %q", id)
				}
				if page == 1 {
					return directoryPage(openListPageSize), nil
				}
				if page != 2 {
					t.Fatalf("unexpected page %d", page)
				}
				return []*model.FileInfo{{ID: "sync-dir", Name: name, IsDir: true}}, nil
			}
			files, err := readDirectoryPages(context.Background(), "account-root", read)
			if err != nil || calls != 2 || len(files) != openListPageSize+1 {
				t.Fatalf("calls=%d files=%d error=%v", calls, len(files), err)
			}
			if files[openListPageSize].Name != name {
				t.Fatalf("sync directory was omitted or renamed")
			}
		})
	}
}

func directoryPage(size int) []*model.FileInfo {
	files := make([]*model.FileInfo, size)
	for i := range files {
		files[i] = &model.FileInfo{ID: fmt.Sprint(i), Name: fmt.Sprintf("item-%d", i)}
	}
	return files
}

func TestReadDirectoryPagesFetchesEmptyTerminalPage(t *testing.T) {
	calls := 0
	read := func(_ context.Context, _ string, page int) ([]*model.FileInfo, error) {
		calls++
		if page == 1 {
			return directoryPage(listPageSize), nil
		}
		return nil, nil
	}
	files, err := readDirectoryPages(context.Background(), "0", read)
	if err != nil || calls != 2 || len(files) != listPageSize {
		t.Fatalf("calls=%d files=%d error=%v", calls, len(files), err)
	}
}

func TestReadDirectoryPagesSurfacesFailure(t *testing.T) {
	upstreamErr := errors.New("upstream refused request")
	read := func(_ context.Context, _ string, page int) ([]*model.FileInfo, error) {
		if page == 1 {
			return directoryPage(listPageSize), nil
		}
		return nil, upstreamErr
	}
	files, err := readDirectoryPages(context.Background(), "0", read)
	if files != nil || !errors.Is(err, upstreamErr) {
		t.Fatalf("partial files=%v error=%v", files, err)
	}
}

func TestReadDirectoryPagesStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	read := func(_ context.Context, _ string, _ int) ([]*model.FileInfo, error) {
		cancel()
		return directoryPage(listPageSize), nil
	}
	files, err := readDirectoryPages(ctx, "0", read)
	if files != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("partial files=%v error=%v", files, err)
	}
}
