package wopan

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
	"woopen/internal/model"

	"github.com/OpenListTeam/wopan-sdk-go"
)

// ListFiles 获取文件列表（个人云）
func (c *Client) ListFiles(dirID string, page, pageSize int) ([]*model.FileInfo, error) {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()

	client, err := c.getMetaClient()
	if err != nil {
		return nil, err
	}
	c.syncClientTokens(client)

	// 如果没有指定目录ID，使用根目录
	if dirID == "" {
		dirID = c.getRootFolderID()
	}

	// 联通云盘 API：根目录使用 "0"（空字符串会导致全量查询）
	if dirID == "" {
		dirID = "0"
	}

	// Debug log: opt-in to avoid spamming logs under WebDAV.
	if os.Getenv("WOOPEN_WOPAN_DEBUG") == "1" {
		fmt.Printf("[DEBUG] ListFiles: dirID='%s', page=%d\n", dirID, page)
	}
	pageNum := page - 1
	if pageNum < 0 {
		pageNum = 0
	}

	files, err := c.listFilesLocked(client, listFilesParams{
		dirID:    dirID,
		pageNum:  pageNum,
		pageSize: pageSize,
	})
	if err != nil && isLoginFailed(err) {
		// Token 过期，尝试刷新后重试
		if refreshErr := client.RefreshToken(); refreshErr != nil {
			return nil, fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		files, err = c.listFilesLocked(client, listFilesParams{
			dirID:    dirID,
			pageNum:  pageNum,
			pageSize: pageSize,
		})
	}
	return files, err
}

type listFilesParams struct {
	dirID    string
	pageNum  int
	pageSize int
}

// listFilesLocked 内部获取文件列表（需要持有锁）
func (c *Client) listFilesLocked(client *wopan.WoClient, params listFilesParams) ([]*model.FileInfo, error) {
	data, err := client.QueryAllFilesPersonal(
		params.dirID,
		params.pageNum,
		params.pageSize,
		0, // 默认排序
	)
	if err != nil {
		return nil, fmt.Errorf("获取文件列表失败: %w", err)
	}

	if os.Getenv("WOOPEN_WOPAN_DEBUG") == "1" {
		fmt.Printf("[DEBUG] QueryAllFiles returned %d items\n", len(data.Files))
		for i, f := range data.Files {
			if i < 5 {
				fmt.Printf("[DEBUG] Item %d: Name=%s, ID=%s, Type=%d, ParentID=%s\n", i, f.Name, f.Id, f.Type, f.Fid)
			}
		}
	}

	files := make([]*model.FileInfo, 0, len(data.Files))
	for _, f := range data.Files {
		// API 返回的时间格式为 "yyyyMMddHHmmss"，例如 "20241009162834"
		modTime, _ := time.Parse("20060102150405", f.CreateTime)
		files = append(files, &model.FileInfo{
			ID:       f.Id,
			FID:      f.Fid,
			Name:     f.Name,
			Size:     f.Size,
			IsDir:    f.Type == 0, // Type 0 表示目录
			ModTime:  modTime,
			ParentID: f.Fid,
		})
	}
	return files, nil
}

// GetDownloadURL 获取下载链接
func (c *Client) GetDownloadURL(fileID string) (string, error) {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()

	client, err := c.getMetaClient()
	if err != nil {
		return "", err
	}
	c.syncClientTokens(client)

	downloadURL, err := c.getDownloadURLLocked(client, fileID)
	if err != nil && isLoginFailed(err) {
		if refreshErr := client.RefreshToken(); refreshErr != nil {
			return "", fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		downloadURL, err = c.getDownloadURLLocked(client, fileID)
	}
	return downloadURL, err
}

func (c *Client) getDownloadURLLocked(client *wopan.WoClient, fileID string) (string, error) {
	res, err := client.GetDownloadUrlV2([]string{fileID})
	if err != nil {
		return "", fmt.Errorf("获取下载链接失败: %w", err)
	}
	if len(res.List) == 0 {
		return "", fmt.Errorf("未找到下载链接")
	}
	return sanitizeDownloadURL(res.List[0].DownloadUrl), nil
}

func sanitizeDownloadURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.RawQuery == "" {
		return raw
	}
	parts := strings.Split(u.RawQuery, "&")
	changed := false
	for i, part := range parts {
		if !strings.HasPrefix(part, "fid=") {
			continue
		}
		val := strings.TrimPrefix(part, "fid=")
		// 保留原始的 '+'（QueryUnescape 会把 '+' 视为空格）
		normalized := strings.ReplaceAll(val, "+", "%2B")
		decoded, err := url.QueryUnescape(normalized)
		if err != nil {
			decoded = val
		}
		encoded := url.QueryEscape(decoded)
		if encoded != val {
			parts[i] = "fid=" + encoded
			changed = true
		}
	}
	if !changed {
		return raw
	}
	u.RawQuery = strings.Join(parts, "&")
	return u.String()
}
