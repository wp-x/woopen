package wopan

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
	"woopen/internal/model"

	"github.com/OpenListTeam/wopan-sdk-go"
)

// Client 联通云盘客户端封装
type Client struct {
	client       *wopan.WoClient
	refreshToken string
	accessToken  string
	rootFolderID string
	mu           sync.RWMutex

	// Token更新回调
	onTokenUpdate func(refreshToken, accessToken string)
}

// NewClient 创建联通云盘客户端
func NewClient(refreshToken string) *Client {
	return &Client{
		refreshToken: refreshToken,
	}
}

// SetTokenUpdateCallback 设置Token更新回调
func (c *Client) SetTokenUpdateCallback(callback func(refreshToken, accessToken string)) {
	c.onTokenUpdate = callback
}

// SetRootFolderID 设置根文件夹ID
func (c *Client) SetRootFolderID(rootFolderID string) {
	c.rootFolderID = rootFolderID
}

// SetAccessToken 设置AccessToken
func (c *Client) SetAccessToken(accessToken string) {
	c.accessToken = accessToken
}

// Init 初始化客户端（个人云模式，跳过家庭云初始化）
func (c *Client) Init() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.refreshToken == "" && c.accessToken == "" {
		return fmt.Errorf("refresh_token 和 access_token 不能同时为空")
	}

	// 创建客户端
	if c.refreshToken != "" {
		c.client = wopan.DefaultWithRefreshToken(c.refreshToken)
	} else {
		c.client = wopan.DefaultWithAccessToken(c.accessToken)
	}

	// **关键：AccessToken 用于生成 AES 加密密钥**
	// 若未提供 AccessToken，则交由 InitData/RefreshToken 自动刷新获取
	if c.accessToken != "" {
		c.client.SetAccessToken(c.accessToken)
	}

	// 设置Token刷新回调
	c.client.OnRefreshToken(func(accessToken, refreshToken string) {
		c.accessToken = accessToken
		c.refreshToken = refreshToken
		// 通知外部保存Token
		if c.onTokenUpdate != nil {
			c.onTokenUpdate(refreshToken, accessToken)
		}
	})

	// 直接初始化数据，**跳过 FamilyUserCurrentEncode**
	// 因为个人云不需要家庭信息，且调用它会导致加密错误
	if err := c.client.InitData(); err != nil {
		if isLoginFailed(err) && c.refreshToken != "" && c.accessToken != "" {
			if refreshErr := c.client.RefreshToken(); refreshErr != nil {
				return fmt.Errorf("初始化客户端数据失败: %w", refreshErr)
			}
			if retryErr := c.client.InitData(); retryErr != nil {
				return fmt.Errorf("初始化客户端数据失败: %w", retryErr)
			}
			return nil
		}
		return fmt.Errorf("初始化客户端数据失败: %w", err)
	}

	return nil
}

// ListFiles 获取文件列表（个人云）
func (c *Client) ListFiles(dirID string, page, pageSize int) ([]*model.FileInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.client == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}

	// 如果没有指定目录ID，使用根目录
	if dirID == "" {
		dirID = c.rootFolderID
	}

	// 联通云盘 API：根目录使用 "0"（空字符串会导致全量查询）
	if dirID == "" {
		dirID = "0"
	}

	// 提示：当前空字符串可能导致递归列出所有文件，需要找到准确的非递归根目录 ID（目前未知的API特性）
	fmt.Printf("[INFO] ListFiles: dirID='%s', page=%d\n", dirID, page)
	pageNum := page - 1
	if pageNum < 0 {
		pageNum = 0
	}
	data, err := c.client.QueryAllFilesPersonal(
		dirID,
		pageNum,
		pageSize,
		0, // 默认排序
	)
	if err != nil {
		return nil, fmt.Errorf("获取文件列表失败: %w", err)
	}

	// DEBUG: 分析文件结构
	fmt.Printf("[DEBUG] QueryAllFiles returned %d items\n", len(data.Files))
	for i, f := range data.Files {
		if i < 5 {
			fmt.Printf("[DEBUG] Item %d: Name=%s, ID=%s, Type=%d, ParentID=%s\n", i, f.Name, f.Id, f.Type, f.Fid)
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return "", fmt.Errorf("客户端未初始化")
	}

	downloadURL, err := c.getDownloadURLLocked(fileID)
	if err != nil && isLoginFailed(err) {
		if refreshErr := c.client.RefreshToken(); refreshErr != nil {
			return "", fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		downloadURL, err = c.getDownloadURLLocked(fileID)
	}
	return downloadURL, err
}

// IsInitialized 检查客户端是否已初始化
func (c *Client) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client != nil
}

// GetRefreshToken 获取当前的RefreshToken
func (c *Client) GetRefreshToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.refreshToken
}

// GetAccessToken 获取当前的AccessToken
func (c *Client) GetAccessToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken
}

func (c *Client) getDownloadURLLocked(fileID string) (string, error) {
	res, err := c.client.GetDownloadUrlV2([]string{fileID})
	if err != nil {
		return "", fmt.Errorf("获取下载链接失败: %w", err)
	}
	if len(res.List) == 0 {
		return "", fmt.Errorf("未找到下载链接")
	}
	return sanitizeDownloadURL(res.List[0].DownloadUrl), nil
}

func isLoginFailed(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "rsp_code: 8005") ||
		strings.Contains(msg, "rsp_code: 1001") ||
		strings.Contains(msg, "登录失败") ||
		strings.Contains(msg, "无效的令牌")
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
