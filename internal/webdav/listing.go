package webdav

import (
	"context"
	"fmt"
	"woopen/internal/model"
	"woopen/internal/wopan"
)

type directoryPageReader func(context.Context, string, int) ([]*model.FileInfo, error)

func (fs *WopanFS) listAllInDir(ctx context.Context, dirID string) ([]*model.FileInfo, error) {
	return readDirectoryPages(ctx, dirID, func(ctx context.Context, id string, page int) ([]*model.FileInfo, error) {
		return fs.client.ListFilesCtx(ctx, wopan.ListFilesOptions{
			DirID: id, Page: page, PageSize: listPageSize,
		})
	})
}

func readDirectoryPages(ctx context.Context, dirID string, read directoryPageReader) ([]*model.FileInfo, error) {
	var files []*model.FileInfo
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		items, err := read(ctx, dirID, page)
		if err != nil {
			return nil, fmt.Errorf("列目录 %q 第 %d 页失败: %w", dirID, page, err)
		}
		files = append(files, items...)
		if len(items) < listPageSize {
			return files, nil
		}
	}
}
