package wopan

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"woopen/internal/model"

	"github.com/OpenListTeam/wopan-sdk-go"
	"resty.dev/v3"
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
	c.mu.Lock()
	defer c.mu.Unlock()

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

	// Debug log: opt-in to avoid spamming logs under WebDAV.
	if os.Getenv("WOOPEN_WOPAN_DEBUG") == "1" {
		fmt.Printf("[DEBUG] ListFiles: dirID='%s', page=%d\n", dirID, page)
	}
	pageNum := page - 1
	if pageNum < 0 {
		pageNum = 0
	}

	files, err := c.listFilesLocked(dirID, pageNum, pageSize)
	if err != nil && isLoginFailed(err) {
		// Token 过期，尝试刷新后重试
		if refreshErr := c.client.RefreshToken(); refreshErr != nil {
			return nil, fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		files, err = c.listFilesLocked(dirID, pageNum, pageSize)
	}
	return files, err
}

// listFilesLocked 内部获取文件列表（需要持有锁）
func (c *Client) listFilesLocked(dirID string, pageNum, pageSize int) ([]*model.FileInfo, error) {
	data, err := c.client.QueryAllFilesPersonal(
		dirID,
		pageNum,
		pageSize,
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

// Delete 删除文件/目录（个人云）
func (c *Client) Delete(ctx context.Context, isDir bool, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return fmt.Errorf("客户端未初始化")
	}

	dirList := make([]string, 0, 1)
	fileList := make([]string, 0, 1)
	if isDir {
		dirList = append(dirList, id)
	} else {
		fileList = append(fileList, id)
	}

	err := c.deleteLocked(ctx, dirList, fileList)
	if err != nil && isLoginFailed(err) {
		if refreshErr := c.client.RefreshToken(); refreshErr != nil {
			return fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		err = c.deleteLocked(ctx, dirList, fileList)
	}
	return err
}

func (c *Client) deleteLocked(ctx context.Context, dirList, fileList []string) error {
	return c.client.DeleteFile(wopan.SpaceTypePersonal, dirList, fileList, func(req *resty.Request) {
		req.SetContext(ctx)
	})
}

// Rename 重命名文件/目录（个人云）
func (c *Client) Rename(ctx context.Context, isDir bool, id, newName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return fmt.Errorf("客户端未初始化")
	}

	_type := 1
	if isDir {
		_type = 0
	}

	err := c.renameLocked(ctx, _type, id, newName)
	if err != nil && isLoginFailed(err) {
		if refreshErr := c.client.RefreshToken(); refreshErr != nil {
			return fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		err = c.renameLocked(ctx, _type, id, newName)
	}
	return err
}

func (c *Client) renameLocked(ctx context.Context, _type int, id, newName string) error {
	return c.client.RenameFileOrDirectory(wopan.SpaceTypePersonal, _type, id, newName, "", func(req *resty.Request) {
		req.SetContext(ctx)
	})
}

// Move 移动文件/目录到目标目录（个人云）
func (c *Client) Move(ctx context.Context, isDir bool, id, targetDirID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return fmt.Errorf("客户端未初始化")
	}

	dirList := make([]string, 0, 1)
	fileList := make([]string, 0, 1)
	if isDir {
		dirList = append(dirList, id)
	} else {
		fileList = append(fileList, id)
	}

	err := c.moveLocked(ctx, dirList, fileList, targetDirID)
	if err != nil && isLoginFailed(err) {
		if refreshErr := c.client.RefreshToken(); refreshErr != nil {
			return fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		err = c.moveLocked(ctx, dirList, fileList, targetDirID)
	}
	return err
}

func (c *Client) moveLocked(ctx context.Context, dirList, fileList []string, targetDirID string) error {
	return c.client.MoveFile(dirList, fileList, targetDirID, wopan.SpaceTypePersonal, wopan.SpaceTypePersonal, "", "", func(req *resty.Request) {
		req.SetContext(ctx)
	})
}

// UploadFile 上传文件到指定目录
func (c *Client) UploadFile(ctx context.Context, parentID, fileName, contentType string, size int64, content io.Reader, onProgress func(uploaded, total int64)) (*model.FileInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}

	// 如果没有指定父目录ID，使用根目录
	if parentID == "" {
		parentID = c.rootFolderID
	}
	if parentID == "" {
		parentID = "0"
	}

	// 调用SDK的Upload2C方法
	result, err := c.uploadFileLocked(ctx, parentID, fileName, contentType, size, content, onProgress)
	if err != nil && isLoginFailed(err) {
		// Token过期，刷新后重试
		if refreshErr := c.client.RefreshToken(); refreshErr != nil {
			return nil, fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		result, err = c.uploadFileLocked(ctx, parentID, fileName, contentType, size, content, onProgress)
	}
	return result, err
}

// uploadFileLocked 内部上传文件方法（需要持有锁）
func (c *Client) uploadFileLocked(ctx context.Context, parentID, fileName, contentType string, size int64, content io.Reader, onProgress func(uploaded, total int64)) (*model.FileInfo, error) {
	// 将 io.Reader 转换为 io.ReadSeeker
	var seeker io.ReadSeeker
	if rs, ok := content.(io.ReadSeeker); ok {
		seeker = rs
	} else {
		// 如果不是 ReadSeeker，读取全部内容到内存
		// 说明：Web/API 上传通常是 multipart.File（实现 Seek），不会走到这里。
		data, err := io.ReadAll(content)
		if err != nil {
			return nil, fmt.Errorf("读取文件内容失败: %w", err)
		}
		seeker = bytes.NewReader(data)
		size = int64(len(data))
	}

	// SDK 期望从头开始读取
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("重置上传流失败: %w", err)
	}

	// ContentType 允许为空；为空时让服务器/SDK 兜底
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 构建上传文件参数
	uploadFile := wopan.Upload2CFile{
		Name:        fileName,
		Size:        size,
		Content:     seeker,
		ContentType: contentType,
	}

	// 构建上传选项（RetryTimes 让 SDK 在分片级别自动重试，解决首次连接 EOF 问题）
	uploadOption := wopan.Upload2COption{
		Ctx:        ctx,
		RetryTimes: 2,
		OnProgress: func(uploaded, total int64) {
			if onProgress != nil {
				onProgress(uploaded, total)
			}
		},
	}

	fileID, err := c.client.Upload2C(wopan.SpaceTypePersonal, uploadFile, parentID, "", uploadOption)
	if err != nil {
		return nil, fmt.Errorf("上传文件失败: %w", err)
	}

	// 转换为内部文件信息格式
	fileInfo := &model.FileInfo{
		ID:       fileID,
		FID:      parentID,
		Name:     fileName,
		Size:     size,
		IsDir:    false,
		ModTime:  time.Now(),
		ParentID: parentID,
	}

	return fileInfo, nil
}

// CreateDirectory 创建目录
func (c *Client) CreateDirectory(parentID, dirName string) (*model.FileInfo, error) {
	return c.CreateDirectoryCtx(context.Background(), parentID, dirName)
}

// CreateDirectoryCtx 创建目录（支持 context）
func (c *Client) CreateDirectoryCtx(ctx context.Context, parentID, dirName string) (*model.FileInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}

	// 如果没有指定父目录ID，使用根目录
	if parentID == "" {
		parentID = c.rootFolderID
	}
	if parentID == "" {
		parentID = "0"
	}

	result, err := c.createDirectoryLocked(ctx, parentID, dirName)
	if err != nil && isLoginFailed(err) {
		if refreshErr := c.client.RefreshToken(); refreshErr != nil {
			return nil, fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		result, err = c.createDirectoryLocked(ctx, parentID, dirName)
	}
	return result, err
}

// createDirectoryLocked 内部创建目录方法（需要持有锁）
func (c *Client) createDirectoryLocked(ctx context.Context, parentID, dirName string) (*model.FileInfo, error) {
	// 与 OpenList 一致：个人云使用 SpaceTypePersonal，familyID 为空
	result, err := c.client.CreateDirectory(wopan.SpaceTypePersonal, parentID, dirName, "", func(req *resty.Request) {
		req.SetContext(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	fileInfo := &model.FileInfo{
		ID:       result.Id,
		FID:      parentID,
		Name:     dirName,
		Size:     0,
		IsDir:    true,
		ModTime:  time.Now(),
		ParentID: parentID,
	}

	return fileInfo, nil
}

// TempFileUploader is used by WebDAV to spool data to disk without holding it in memory.
// It is intentionally minimal and internal to this package.
func TempFileUploader(prefix string) (*os.File, error) {
	return os.CreateTemp("", prefix)
}
