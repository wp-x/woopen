package handler

import (
	"errors"
	"woopen/internal/model"
)

const (
	shareListPageStart = 1
	shareListPageSize  = 100
)

func (h *Handler) listAllShareFiles(dirID string) ([]*model.FileInfo, error) {
	if dirID == "" {
		return nil, errors.New("缺少目录ID")
	}
	page := shareListPageStart
	files := make([]*model.FileInfo, 0, shareListPageSize)
	for {
		pageFiles, err := h.wopanClient.ListFiles(dirID, page, shareListPageSize)
		if err != nil {
			return nil, err
		}
		if len(pageFiles) == 0 {
			return files, nil
		}
		files = append(files, pageFiles...)
		if len(pageFiles) < shareListPageSize {
			break
		}
		page++
	}
	return files, nil
}
