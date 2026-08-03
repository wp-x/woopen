package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"woopen/internal/middleware"
	"woopen/internal/model"
	"woopen/internal/repository"
	"woopen/internal/wopan"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// MonitorServiceInterface 监控服务接口
type MonitorServiceInterface interface {
	Start()
	Stop()
	IsRunning() bool
	SetInterval(seconds int)
	SetWopanClient(client *wopan.Client)
	CheckNow() (*model.MonitorStatus, error)
	GetStatus() (*model.MonitorStatus, error)
	TestNotify() (map[string]string, error)
	Reload() error
}

// Handler HTTP处理器
type Handler struct {
	settingsRepo        *repository.SettingsRepository
	shareRepo           *repository.ShareRepository
	accessLogRepo       *repository.AccessLogRepository
	monitorRepo         *repository.MonitorRepository
	notificationLogRepo *repository.NotificationLogRepository
	wopanClient         *wopan.Client
	adminPassword       string
	monitorService      MonitorServiceInterface
	listCache           ListCache
	directCache         DirectURLCache
	directLimiter       *directRateLimiter
	loginLimiter        *loginFailureLimiter
	shareLimiter        *sharePasswordLimiter

	// uploadProgress holds best-effort server-side upload progress keyed by upload_id.
	// Used by the admin UI polling endpoint to show real progress during Upload2C.
	uploadProgress sync.Map
}

// HandlerOptions 构建 Handler 的依赖
type HandlerOptions struct {
	SettingsRepo        *repository.SettingsRepository
	ShareRepo           *repository.ShareRepository
	AccessLogRepo       *repository.AccessLogRepository
	MonitorRepo         *repository.MonitorRepository
	NotificationLogRepo *repository.NotificationLogRepository
	WopanClient         *wopan.Client
	AdminPassword       string
	ListCache           ListCache
}

// NewHandler 创建Handler
func NewHandler(opts HandlerOptions) *Handler {
	if opts.ListCache == nil {
		panic("list cache is required")
	}
	return &Handler{
		settingsRepo:        opts.SettingsRepo,
		shareRepo:           opts.ShareRepo,
		accessLogRepo:       opts.AccessLogRepo,
		monitorRepo:         opts.MonitorRepo,
		notificationLogRepo: opts.NotificationLogRepo,
		wopanClient:         opts.WopanClient,
		adminPassword:       opts.AdminPassword,
		listCache:           opts.ListCache,
		directCache:         NewDirectURLCache(DefaultDirectURLCacheTTL),
		directLimiter:       newDirectRateLimiter(),
		loginLimiter:        newLoginFailureLimiter(),
		shareLimiter:        newSharePasswordLimiter(),
	}
}

func sharePasswordKey(c *gin.Context) string {
	return c.RemoteIP() + "|" + c.Param("code")
}

func (h *Handler) checkSharePasswordAttempt(c *gin.Context) bool {
	if h.shareLimiter.allow(sharePasswordKey(c)) {
		return true
	}
	c.Header("Retry-After", "900")
	c.JSON(http.StatusTooManyRequests, model.APIResponse{Code: 429, Message: "密码尝试过多，请稍后再试"})
	return false
}

func (h *Handler) recordSharePasswordFailure(c *gin.Context) {
	h.shareLimiter.recordFailure(sharePasswordKey(c))
}

func (h *Handler) resetSharePasswordFailures(c *gin.Context) {
	h.shareLimiter.reset(sharePasswordKey(c))
}

// SetMonitorService 设置监控服务
func (h *Handler) SetMonitorService(ms MonitorServiceInterface) {
	h.monitorService = ms
}

// SetWopanClient 设置云盘客户端
func (h *Handler) SetWopanClient(client *wopan.Client) {
	h.wopanClient = client
}

// ===================== 公开配置 =====================

// GetSiteConfig 获取公开的站点配置（无需登录）
func (h *Handler) GetSiteConfig(c *gin.Context) {
	settings, err := h.settingsRepo.Get()
	if err != nil {
		baseURL := requestBaseURL(c)
		// 返回默认值
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    0,
			Message: "success",
			Data: gin.H{
				"site_title":        "WoOpen",
				"login_username":    "Administrator",
				"login_title":       "WoOpen_Auth_v10.exe",
				"login_avatar":      "💀",
				"login_role_tag":    "ROLE: ADMIN",
				"login_level_tag":   "LEVEL: 99",
				"login_system_name": "WOOPEN CLOUD SYSTEM",
				"share_footer":      "Powered by WOOPEN_OS // BRUTAL_EDITION",
				// For copy/paste (WebDAV mount, share links, etc.). Prefer reverse-proxy headers.
				"site_url":   baseURL,
				"webdav_url": baseURL + "/dav/",
			},
		})
		return
	}

	baseURL := requestBaseURL(c)
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"site_title":        settings.SiteTitle,
			"login_username":    settings.LoginUsername,
			"login_title":       settings.LoginTitle,
			"login_avatar":      settings.LoginAvatar,
			"login_role_tag":    settings.LoginRoleTag,
			"login_level_tag":   settings.LoginLevelTag,
			"login_system_name": settings.LoginSystemName,
			"share_footer":      settings.ShareFooter,
			"site_url":          baseURL,
			"webdav_url":        baseURL + "/dav/",
			// 账号名属凭据的一部分，不在无鉴权的公开接口暴露；WebDAV 配置从受鉴权的 GetSettings 拉取。
		},
	})
}

// ===================== 认证相关 =====================

// Login 管理员登录
func (h *Handler) Login(c *gin.Context) {
	clientIP := c.RemoteIP()
	if !h.loginLimiter.allow(clientIP) {
		c.Header("Retry-After", "900")
		c.JSON(http.StatusTooManyRequests, model.APIResponse{Code: 429, Message: "登录尝试过多，请稍后再试"})
		return
	}

	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 验证密码
	if req.Password != h.adminPassword {
		h.loginLimiter.recordFailure(clientIP)
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    401,
			Message: "密码错误",
		})
		return
	}
	h.loginLimiter.reset(clientIP)

	// 生成Token
	token, err := middleware.GenerateToken(req.Expire)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "生成Token失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "登录成功",
		Data: model.LoginResponse{
			Token: token,
		},
	})
}

// GetMe 获取当前用户信息
func (h *Handler) GetMe(c *gin.Context) {
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"role": "admin",
		},
	})
}

// ===================== 文件操作 =====================

// ListFiles 获取文件列表
func (h *Handler) ListFiles(c *gin.Context) {
	dirID := c.Query("dir_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	req := ListRequest{
		DirID:    dirID,
		Page:     page,
		PageSize: pageSize,
	}

	if !h.wopanClient.IsInitialized() {
		c.JSON(http.StatusServiceUnavailable, model.APIResponse{
			Code:    503,
			Message: "云盘客户端未初始化，请先配置Token",
		})
		return
	}

	if files, ok := h.listCache.Get(req); ok {
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    0,
			Message: "success",
			Data: gin.H{
				"files": files,
				"page":  page,
			},
		})
		return
	}

	files, err := h.wopanClient.ListFiles(dirID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: fmt.Sprintf("获取文件列表失败: %v", err),
		})
		return
	}
	h.listCache.Set(req, files)

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"files": files,
			"page":  page,
		},
	})
}

type fileOperationItem struct {
	ID    string `json:"id" binding:"required"`
	IsDir bool   `json:"is_dir"`
}

type fileOperationRequest struct {
	Items       []fileOperationItem `json:"items" binding:"required,min=1"`
	TargetDirID string              `json:"target_dir_id"`
}

type renameFileRequest struct {
	ID    string `json:"id" binding:"required"`
	IsDir bool   `json:"is_dir"`
	Name  string `json:"name" binding:"required"`
}

func (h *Handler) DeleteFiles(c *gin.Context) {
	var req fileOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fileOperationError(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	dirIDs, fileIDs := splitFileOperationItems(req.Items)
	if err := h.wopanClient.DeleteBatch(c.Request.Context(), dirIDs, fileIDs); err != nil {
		fileOperationError(c, http.StatusBadGateway, "删除失败: "+err.Error())
		return
	}
	h.listCache.InvalidateAll()
	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "删除成功"})
}

func (h *Handler) RenameFile(c *gin.Context) {
	var req renameFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fileOperationError(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		fileOperationError(c, http.StatusBadRequest, "名称不能为空")
		return
	}
	if err := h.wopanClient.Rename(c.Request.Context(), req.IsDir, req.ID, name); err != nil {
		fileOperationError(c, http.StatusBadGateway, "重命名失败: "+err.Error())
		return
	}
	h.listCache.InvalidateAll()
	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "重命名成功"})
}

func (h *Handler) MoveFiles(c *gin.Context) {
	h.handleCopyMove(c, false)
}

func (h *Handler) CopyFiles(c *gin.Context) {
	h.handleCopyMove(c, true)
}

func (h *Handler) handleCopyMove(c *gin.Context, copyMode bool) {
	var req fileOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TargetDirID) == "" {
		fileOperationError(c, http.StatusBadRequest, "必须提供目标目录")
		return
	}
	dirIDs, fileIDs := splitFileOperationItems(req.Items)
	var err error
	verb := "移动"
	if copyMode {
		err = h.wopanClient.CopyBatch(c.Request.Context(), dirIDs, fileIDs, req.TargetDirID)
		verb = "复制"
	} else {
		err = h.wopanClient.MoveBatch(c.Request.Context(), dirIDs, fileIDs, req.TargetDirID)
	}
	if err != nil {
		fileOperationError(c, http.StatusBadGateway, verb+"失败: "+err.Error())
		return
	}
	h.listCache.InvalidateAll()
	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "操作成功"})
}

func splitFileOperationItems(items []fileOperationItem) ([]string, []string) {
	dirIDs := make([]string, 0, len(items))
	fileIDs := make([]string, 0, len(items))
	for _, item := range items {
		if item.IsDir {
			dirIDs = append(dirIDs, item.ID)
			continue
		}
		fileIDs = append(fileIDs, item.ID)
	}
	return dirIDs, fileIDs
}

func fileOperationError(c *gin.Context, status int, message string) {
	c.JSON(status, model.APIResponse{Code: status, Message: message})
}

// GetFileLink 获取文件下载直链（仅文件）
func (h *Handler) GetFileLink(c *gin.Context) {
	fid := c.Query("fid")
	if fid == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "缺少 fid",
		})
		return
	}

	downloadURL, err := h.wopanClient.GetDownloadURL(fid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取下载链接失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"url": downloadURL,
		},
	})
}

// UploadFile 上传文件
func (h *Handler) UploadFile(c *gin.Context) {
	if !h.wopanClient.IsInitialized() {
		c.JSON(http.StatusServiceUnavailable, model.APIResponse{
			Code:    503,
			Message: "云盘客户端未初始化，请先配置Token",
		})
		return
	}

	// 获取父目录ID
	parentID := c.PostForm("parent_id")

	// upload_id 由前端生成并随表单上传，用于轮询进度；为空时服务端兜底生成。
	uploadID := c.PostForm("upload_id")
	if uploadID == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err == nil {
			uploadID = hex.EncodeToString(b)
		} else {
			// 极端情况下兜底，避免空 key 覆盖。
			uploadID = fmt.Sprintf("u_%d", time.Now().UnixNano())
		}
	}

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "获取上传文件失败: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// 初始化进度条目（best-effort：仅内存态）
	entry := &uploadProgressEntry{}
	entry.setUploading(header.Filename, parentID, header.Size)
	h.uploadProgress.Store(uploadID, entry)

	// 上传文件
	fileInfo, err := h.wopanClient.UploadFile(
		c.Request.Context(),
		parentID,
		header.Filename,
		header.Header.Get("Content-Type"),
		header.Size,
		file,
		func(uploaded, total int64) {
			entry.setProgress(uploaded, total)
		},
	)
	if err != nil {
		entry.setDone("failed", err)
		// 避免 map 长期增长：失败后短暂保留，便于前端拿到错误信息。
		time.AfterFunc(2*time.Minute, func() {
			h.uploadProgress.Delete(uploadID)
		})

		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "上传文件失败: " + err.Error(),
		})
		return
	}

	entry.setDone("success", nil)
	time.AfterFunc(2*time.Minute, func() {
		h.uploadProgress.Delete(uploadID)
	})
	h.listCache.InvalidateDir(parentID)

	c.Header("X-Upload-Id", uploadID)
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "上传成功",
		Data:    fileInfo,
	})
}

// CreateDirectory 创建目录
func (h *Handler) CreateDirectory(c *gin.Context) {
	if !h.wopanClient.IsInitialized() {
		c.JSON(http.StatusServiceUnavailable, model.APIResponse{
			Code:    503,
			Message: "云盘客户端未初始化，请先配置Token",
		})
		return
	}

	var req struct {
		ParentID string `json:"parent_id"`
		Name     string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	fileInfo, err := h.wopanClient.CreateDirectory(req.ParentID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "创建目录失败: " + err.Error(),
		})
		return
	}
	h.listCache.InvalidateDir(req.ParentID)

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "创建成功",
		Data:    fileInfo,
	})
}

// ===================== 分享管理 =====================

// generateShareCode 生成随机分享码
func generateShareCode() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func directShareEnabled(targetType string, requested bool) bool {
	return targetType == "file" && requested
}

// CreateShare 创建分享
func (h *Handler) CreateShare(c *gin.Context) {
	var req model.CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 生成或使用自定义分享码
	shareCode := req.ShareCode
	if shareCode == "" {
		shareCode = generateShareCode()
	}

	// 检查分享码是否已存在
	exists, _ := h.shareRepo.CodeExists(shareCode)
	if exists {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "分享码已存在",
		})
		return
	}

	// 解析过期时间
	var expireAt *time.Time
	if req.ExpireAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpireAt)
		if err == nil {
			expireAt = &t
		}
	}

	isDirect := directShareEnabled(req.TargetType, req.IsDirect)
	// 直连仅支持单文件；文件夹使用 /s/ 目录树分享。
	password := req.Password
	if isDirect {
		password = ""
	}

	share := &model.Share{
		ShareCode:    shareCode,
		TargetType:   req.TargetType,
		TargetID:     req.TargetID,
		TargetPath:   req.TargetPath,
		TargetName:   req.TargetName,
		Password:     password,
		ExpireAt:     expireAt,
		Description:  req.Description,
		IsActive:     true,
		MaxDownloads: req.MaxDownloads,
		IsDirect:     isDirect,
	}

	if err := h.shareRepo.Create(share); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "创建分享失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "创建成功",
		Data:    share,
	})
}

// ListShares 获取分享列表
func (h *Handler) ListShares(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	shares, total, err := h.shareRepo.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取分享列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"shares":    shares,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// UpdateShare 更新分享
func (h *Handler) UpdateShare(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	share, err := h.shareRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    404,
			Message: "分享不存在",
		})
		return
	}

	var req struct {
		ShareCode    string `json:"share_code"`
		Password     string `json:"password"`
		ExpireAt     string `json:"expire_at"`
		Description  string `json:"description"`
		IsActive     *bool  `json:"is_active"`
		MaxDownloads *int64 `json:"max_downloads"`
		IsDirect     *bool  `json:"is_direct"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	if req.ShareCode != "" && req.ShareCode != share.ShareCode {
		exists, _ := h.shareRepo.CodeExists(req.ShareCode)
		if exists {
			c.JSON(http.StatusBadRequest, model.APIResponse{
				Code:    400,
				Message: "分享码已存在",
			})
			return
		}
		share.ShareCode = req.ShareCode
	}

	share.Password = req.Password
	share.Description = req.Description
	if req.IsActive != nil {
		share.IsActive = *req.IsActive
	}
	if req.MaxDownloads != nil {
		share.MaxDownloads = *req.MaxDownloads
	}
	if req.IsDirect != nil {
		share.IsDirect = *req.IsDirect
	}
	share.IsDirect = directShareEnabled(share.TargetType, share.IsDirect)
	// 直连模式无法交互输入密码，强制清空
	if share.IsDirect {
		share.Password = ""
	}

	if req.ExpireAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpireAt)
		if err == nil {
			share.ExpireAt = &t
		}
	} else {
		share.ExpireAt = nil
	}

	if err := h.shareRepo.Update(share); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "更新失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "更新成功",
		Data:    share,
	})
}

// DeleteShare 删除分享
func (h *Handler) DeleteShare(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.shareRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "删除失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "删除成功",
	})
}

// ===================== 公开访问 =====================

// AccessShare 访问分享页面
func (h *Handler) AccessShare(c *gin.Context) {
	code := c.Param("code")

	share, err := h.shareRepo.GetByCode(code)
	if err != nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    404,
			Message: "分享不存在或已失效",
		})
		return
	}

	// 检查是否启用
	if !share.IsActive {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    404,
			Message: "分享已禁用",
		})
		return
	}

	// 检查是否过期
	if share.ExpireAt != nil && share.ExpireAt.Before(time.Now()) {
		c.JSON(http.StatusGone, model.APIResponse{
			Code:    410,
			Message: "分享已过期",
		})
		return
	}

	// 检查是否需要密码
	needPassword := share.Password != ""

	// 记录访问日志
	h.shareRepo.IncrementViewCount(share.ID)
	h.accessLogRepo.Create(&model.AccessLog{
		ShareID:   share.ID,
		Action:    "view",
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Referer:   c.GetHeader("Referer"),
	})

	// 计算剩余下载次数
	var remainingDownloads int64 = -1 // -1 表示不限制
	downloadExhausted := false
	if share.MaxDownloads > 0 {
		remainingDownloads = share.MaxDownloads - share.DownloadCount
		if remainingDownloads < 0 {
			remainingDownloads = 0
		}
		downloadExhausted = share.DownloadCount >= share.MaxDownloads
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"share_code":          share.ShareCode,
			"target_type":         share.TargetType,
			"target_name":         share.TargetName,
			"need_password":       needPassword,
			"max_downloads":       share.MaxDownloads,
			"download_count":      share.DownloadCount,
			"remaining_downloads": remainingDownloads,
			"download_exhausted":  downloadExhausted,
		},
	})
}

// VerifySharePassword 验证分享密码
func (h *Handler) VerifySharePassword(c *gin.Context) {
	code := c.Param("code")
	if !h.checkSharePasswordAttempt(c) {
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	share, err := h.shareRepo.GetByCode(code)
	if err != nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    404,
			Message: "分享不存在",
		})
		return
	}

	if share.Password != req.Password {
		h.recordSharePasswordFailure(c)
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    401,
			Message: "密码错误",
		})
		return
	}
	h.resetSharePasswordFailures(c)

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "验证成功",
		Data: gin.H{
			"target_type": share.TargetType,
			"target_id":   share.TargetID,
			"target_name": share.TargetName,
		},
	})
}

// shareFileID 取分享内文件ID：优先 query（fid 可能含 '/'，路径段放不下），兼容旧路径参数。
func shareFileID(c *gin.Context) string {
	if fid := c.Query("fid"); fid != "" {
		return fid
	}
	return c.Param("fileId")
}

// DownloadShare 下载分享文件（302重定向）
func (h *Handler) DownloadShare(c *gin.Context) {
	code := c.Param("code")
	fileID := shareFileID(c)

	share, err := h.shareRepo.GetByCode(code)
	if err != nil || !share.IsActive {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    404,
			Message: "分享不存在或已失效",
		})
		return
	}

	// 检查过期
	if share.ExpireAt != nil && share.ExpireAt.Before(time.Now()) {
		c.JSON(http.StatusGone, model.APIResponse{
			Code:    410,
			Message: "分享已过期",
		})
		return
	}

	// 验证密码（通过Cookie或Header）
	if share.Password != "" {
		pwd := c.Query("pwd")
		if pwd != share.Password {
			if !h.checkSharePasswordAttempt(c) {
				return
			}
			h.recordSharePasswordFailure(c)
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    401,
				Message: "需要密码验证",
			})
			return
		}
		h.resetSharePasswordFailures(c)
	}

	// 检查下载次数限制
	if share.MaxDownloads > 0 && share.DownloadCount >= share.MaxDownloads {
		c.JSON(http.StatusForbidden, model.APIResponse{
			Code:    403,
			Message: "下载次数已达上限",
		})
		return
	}

	// 对于单文件分享，fileID可以为空，使用分享的target_id
	if fileID == "" || fileID == "0" {
		fileID = share.TargetID
	}

	// 检查是否是文件夹分享中的文件
	// TODO: 对于文件夹分享，需要验证fileID是否在分享的文件夹内

	// 获取下载链接
	downloadURL, err := h.wopanClient.GetDownloadURL(fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取下载链接失败: " + err.Error(),
		})
		return
	}

	// 记录下载
	h.shareRepo.IncrementDownloadCount(share.ID)
	h.accessLogRepo.Create(&model.AccessLog{
		ShareID:   share.ID,
		Action:    "download",
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Referer:   c.GetHeader("Referer"),
	})

	// 302重定向到直链
	c.Redirect(http.StatusFound, downloadURL)
}

// ===================== 统计相关 =====================

// GetStatsOverview 获取统计概览
func (h *Handler) GetStatsOverview(c *gin.Context) {
	totalViews, totalDownloads, todayViews, err := h.accessLogRepo.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取统计失败",
		})
		return
	}

	// 获取总分享数
	shares, totalShares, _ := h.shareRepo.List(1, 1)
	_ = shares

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"total_shares":    totalShares,
			"total_views":     totalViews,
			"total_downloads": totalDownloads,
			"today_views":     todayViews,
		},
	})
}

// GetStatsTrend 获取趋势数据
func (h *Handler) GetStatsTrend(c *gin.Context) {
	trend, err := h.accessLogRepo.GetTrend()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取趋势数据失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data:    trend,
	})
}

// ===================== 系统设置 =====================

// GetSettings 获取设置
func (h *Handler) GetSettings(c *gin.Context) {
	settings, err := h.settingsRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取设置失败",
		})
		return
	}

	// 隐藏敏感信息
	maskedToken := ""
	if settings.RefreshToken != "" {
		if len(settings.RefreshToken) > 10 {
			maskedToken = settings.RefreshToken[:5] + "****" + settings.RefreshToken[len(settings.RefreshToken)-5:]
		} else {
			maskedToken = "****"
		}
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"refresh_token":     maskedToken,
			"access_token":      maskedAccessToken(settings.AccessToken),
			"root_folder_id":    settings.RootFolderID,
			"site_title":        settings.SiteTitle,
			"site_logo":         settings.SiteLogo,
			"login_username":    settings.LoginUsername,
			"login_title":       settings.LoginTitle,
			"login_avatar":      settings.LoginAvatar,
			"login_role_tag":    settings.LoginRoleTag,
			"login_level_tag":   settings.LoginLevelTag,
			"login_system_name": settings.LoginSystemName,
			"share_footer":      settings.ShareFooter,
			"initialized":       h.wopanClient.IsInitialized(),

			"direct_referer_whitelist":   settings.DirectRefererWhitelist,
			"direct_allow_empty_referer": settings.DirectAllowEmptyReferer,
			"direct_rate_limit":          settings.DirectRateLimit,
			"direct_reject_placeholder":  settings.DirectRejectPlaceholder,

			"webdav_enabled":      settings.WebdavEnabled,
			"webdav_username":     settings.WebdavUsername,
			"webdav_readonly":     settings.WebdavReadonly,
			"webdav_password_set": settings.WebdavPassword != "",
		},
	})
}

// maskedAccessToken 掩码处理 AccessToken
func maskedAccessToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 12 {
		return "****"
	}
	return token[:6] + "****" + token[len(token)-6:]
}

// UpdateSettings 更新设置
func (h *Handler) UpdateSettings(c *gin.Context) {
	// 使用 map 解析请求，区分"未传字段"和"传了空字符串"
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	settings, err := h.settingsRepo.Get()
	if err != nil || settings == nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取设置失败",
		})
		return
	}

	// Token 更新逻辑
	tokenChanged := false
	refreshTokenChanged := false
	accessTokenChanged := false
	if v, ok := raw["refresh_token"].(string); ok && v != "" && !containsMask(v) {
		settings.RefreshToken = v
		tokenChanged = true
		refreshTokenChanged = true
	}
	if v, ok := raw["access_token"].(string); ok && v != "" && !containsMask(v) {
		settings.AccessToken = v
		tokenChanged = true
		accessTokenChanged = true
	}
	if refreshTokenChanged && !accessTokenChanged {
		settings.AccessToken = ""
	}

	// 只更新请求中明确包含的字段，避免未传字段覆盖已有数据
	if v, ok := raw["root_folder_id"].(string); ok {
		settings.RootFolderID = v
	}
	if v, ok := raw["site_title"].(string); ok {
		settings.SiteTitle = v
	}
	if v, ok := raw["site_logo"].(string); ok {
		settings.SiteLogo = v
	}
	if v, ok := raw["login_username"].(string); ok {
		settings.LoginUsername = v
	}
	if v, ok := raw["login_title"].(string); ok {
		settings.LoginTitle = v
	}
	if v, ok := raw["login_avatar"].(string); ok {
		settings.LoginAvatar = v
	}
	if v, ok := raw["login_role_tag"].(string); ok {
		settings.LoginRoleTag = v
	}
	if v, ok := raw["login_level_tag"].(string); ok {
		settings.LoginLevelTag = v
	}
	if v, ok := raw["login_system_name"].(string); ok {
		settings.LoginSystemName = v
	}
	if v, ok := raw["share_footer"].(string); ok {
		settings.ShareFooter = v
	}
	if v, ok := raw["direct_referer_whitelist"].(string); ok {
		settings.DirectRefererWhitelist = v
	}
	if v, ok := raw["direct_allow_empty_referer"].(bool); ok {
		settings.DirectAllowEmptyReferer = v
	}
	if v, ok := raw["direct_rate_limit"].(float64); ok {
		settings.DirectRateLimit = int(v)
	}
	if v, ok := raw["direct_reject_placeholder"].(bool); ok {
		settings.DirectRejectPlaceholder = v
	}
	if v, ok := raw["webdav_enabled"].(bool); ok {
		settings.WebdavEnabled = v
	}
	if v, ok := raw["webdav_username"].(string); ok && v != "" {
		settings.WebdavUsername = v
	}
	// 空字符串表示保持原密码，仅在传入非空且非掩码时更新
	if v, ok := raw["webdav_password"].(string); ok && v != "" && !containsMask(v) {
		settings.WebdavPassword = v
	}
	if v, ok := raw["webdav_readonly"].(bool); ok {
		settings.WebdavReadonly = v
	}

	if err := h.settingsRepo.Update(settings); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "保存设置失败",
		})
		return
	}

	if tokenChanged {
		go h.reinitWopanClient(settings)
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "设置已保存",
	})
}

// reinitWopanClient 重新初始化云盘客户端
func (h *Handler) reinitWopanClient(settings *model.Settings) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("重新初始化云盘客户端 panic: %v\n", r)
		}
	}()

	client := wopan.NewClient(settings.RefreshToken)
	client.SetRootFolderID(settings.RootFolderID)
	if settings.AccessToken != "" {
		client.SetAccessToken(settings.AccessToken)
	}
	client.SetTokenUpdateCallback(func(refreshToken, accessToken string) {
		h.settingsRepo.UpdateToken(refreshToken, accessToken)
	})
	if err := client.Init(); err != nil {
		fmt.Printf("重新初始化云盘客户端失败: %v\n", err)
		return
	}
	h.wopanClient = client
	if h.monitorService != nil {
		h.monitorService.SetWopanClient(client)
	}
}

// UpdatePassword 修改管理员密码
func (h *Handler) UpdatePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 验证旧密码
	if req.OldPassword != h.adminPassword {
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    401,
			Message: "原密码错误",
		})
		return
	}

	// 更新密码（这里需要重启生效，或者实现持久化）
	h.adminPassword = req.NewPassword

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "密码已修改",
	})
}

// TestToken 测试Token有效性
func (h *Handler) TestToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
		AccessToken  string `json:"access_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请提供refresh_token或access_token",
		})
		return
	}
	if req.RefreshToken == "" && req.AccessToken == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请提供refresh_token或access_token",
		})
		return
	}

	// 尝试创建客户端
	client := wopan.NewClient(req.RefreshToken)
	if req.AccessToken != "" {
		client.SetAccessToken(req.AccessToken)
	}
	if err := client.Init(); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "Token无效: " + err.Error(),
		})
		return
	}

	// 使用实际的 API 调用来验证 Token（带重试机制）
	if err := h.validateTokenWithRetry(client); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "Token验证失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "Token有效，连接成功",
	})
}

// validateTokenWithRetry 验证 Token 有效性（带重试机制）
func (h *Handler) validateTokenWithRetry(client *wopan.Client) error {
	maxRetries := 2
	retryDelay := 2 * time.Second

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			log.Printf("[TestToken] 重试 %d/%d\n", i, maxRetries)
			time.Sleep(retryDelay)
		}

		// 使用轻量级的文件列表查询来验证 Token
		_, err := client.ListFiles("0", 1, 1)

		if err == nil {
			// 验证成功
			return nil
		}

		// 判断是否为真正的 Token 失效错误
		if isTokenInvalidError(err) {
			// 确认是 Token 失效，不再重试
			return err
		}

		// 其他错误（如网络问题、临时限流等），继续重试
		log.Printf("[TestToken] 遇到临时错误: %v\n", err)
	}

	// 重试后仍然失败，返回最后一次错误
	_, lastErr := client.ListFiles("0", 1, 1)
	return lastErr
}

// isTokenInvalidError 判断是否为 Token 失效错误
func isTokenInvalidError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// 只有明确的 Token 失效错误码才判定为失效
	return strings.Contains(msg, "rsp_code: 8005") ||
		strings.Contains(msg, "rsp_code: 1001") ||
		strings.Contains(msg, "登录失败") ||
		strings.Contains(msg, "无效的令牌") ||
		strings.Contains(msg, "token已过期") ||
		strings.Contains(msg, "token invalid")
}

// containsMask 检查字符串是否包含掩码
func containsMask(s string) bool {
	return len(s) > 0 && (s[0] == '*' || (len(s) > 4 && s[4:8] == "****"))
}

// hashPassword bcrypt加密密码
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// ===================== 批量分享 =====================

// BatchCreateShareRequest 批量创建分享请求
type BatchCreateShareRequest struct {
	Items []model.CreateShareRequest `json:"items" binding:"required"`
}

// BatchCreateShare 批量创建分享
func (h *Handler) BatchCreateShare(c *gin.Context) {
	var req BatchCreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请至少选择一个文件",
		})
		return
	}

	if len(req.Items) > 50 {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "一次最多创建50个分享",
		})
		return
	}

	results := make([]*model.Share, 0, len(req.Items))
	for _, item := range req.Items {
		shareCode := item.ShareCode
		if shareCode == "" {
			shareCode = generateShareCode()
		}

		// 检查分享码
		exists, _ := h.shareRepo.CodeExists(shareCode)
		if exists {
			shareCode = generateShareCode() // 重新生成
		}

		var expireAt *time.Time
		if item.ExpireAt != "" {
			t, err := time.Parse(time.RFC3339, item.ExpireAt)
			if err == nil {
				expireAt = &t
			}
		}

		share := &model.Share{
			ShareCode:    shareCode,
			TargetType:   item.TargetType,
			TargetID:     item.TargetID,
			TargetPath:   item.TargetPath,
			TargetName:   item.TargetName,
			Password:     item.Password,
			ExpireAt:     expireAt,
			Description:  item.Description,
			IsActive:     true,
			MaxDownloads: item.MaxDownloads,
			IsDirect:     directShareEnabled(item.TargetType, item.IsDirect),
		}
		if share.IsDirect {
			share.Password = ""
		}

		if err := h.shareRepo.Create(share); err != nil {
			continue // 跳过失败的
		}
		results = append(results, share)
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: fmt.Sprintf("成功创建 %d 个分享", len(results)),
		Data:    results,
	})
}

// ===================== 文件夹分享文件列表 =====================

// GetShareFiles 获取分享的文件夹内容
func (h *Handler) GetShareFiles(c *gin.Context) {
	code := c.Param("code")
	dirID := c.Query("dir_id") // 子目录ID

	share, err := h.shareRepo.GetByCode(code)
	if err != nil || !share.IsActive {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    404,
			Message: "分享不存在或已失效",
		})
		return
	}

	// 检查过期
	if share.ExpireAt != nil && share.ExpireAt.Before(time.Now()) {
		c.JSON(http.StatusGone, model.APIResponse{
			Code:    410,
			Message: "分享已过期",
		})
		return
	}

	// 验证密码
	if share.Password != "" {
		pwd := c.Query("pwd")
		if pwd != share.Password {
			if !h.checkSharePasswordAttempt(c) {
				return
			}
			h.recordSharePasswordFailure(c)
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    401,
				Message: "需要密码验证",
			})
			return
		}
		h.resetSharePasswordFailures(c)
	}

	// 只有文件夹分享才能获取子文件列表
	if share.TargetType != "folder" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "该分享不是文件夹",
		})
		return
	}

	// 如果没有指定子目录，使用分享的根目录
	if dirID == "" {
		dirID = share.TargetID
	}

	// 获取文件列表
	files, err := h.listAllShareFiles(dirID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取文件列表失败",
		})
		return
	}

	// 计算剩余下载次数
	var remainingDownloads int64 = -1 // -1 表示不限制
	downloadExhausted := false
	if share.MaxDownloads > 0 {
		remainingDownloads = share.MaxDownloads - share.DownloadCount
		if remainingDownloads < 0 {
			remainingDownloads = 0
		}
		downloadExhausted = share.DownloadCount >= share.MaxDownloads
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"files":               files,
			"share_code":          share.ShareCode,
			"root_id":             share.TargetID,
			"current_id":          dirID,
			"max_downloads":       share.MaxDownloads,
			"download_count":      share.DownloadCount,
			"remaining_downloads": remainingDownloads,
			"download_exhausted":  downloadExhausted,
		},
	})
}

// ===================== 二维码生成 =====================

// GetShareQRCode 获取分享二维码
func (h *Handler) GetShareQRCode(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	share, err := h.shareRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    404,
			Message: "分享不存在",
		})
		return
	}

	// 获取站点URL
	settings, _ := h.settingsRepo.Get()
	siteURL := settings.SiteLogo // 先用 SiteLogo 占位，实际应该用 SiteURL
	if siteURL == "" {
		siteURL = c.Request.Host
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		siteURL = fmt.Sprintf("%s://%s", scheme, siteURL)
	}

	shareURL := fmt.Sprintf("%s/s/%s", siteURL, share.ShareCode)

	// 生成二维码
	qrCode, err := qr.Encode(shareURL, qr.M, qr.Auto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "生成二维码失败",
		})
		return
	}

	// 缩放到 256x256
	qrCode, err = barcode.Scale(qrCode, 256, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "缩放二维码失败",
		})
		return
	}

	// 编码为 PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, qrCode); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "编码二维码失败",
		})
		return
	}

	// 返回 base64 数据
	base64Data := base64.StdEncoding.EncodeToString(buf.Bytes())

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"qrcode":    "data:image/png;base64," + base64Data,
			"share_url": shareURL,
		},
	})
}

// ===================== 预览支持 =====================

// PreviewShare 预览分享文件（302重定向）
func (h *Handler) PreviewShare(c *gin.Context) {
	code := c.Param("code")
	fileID := shareFileID(c)

	share, err := h.shareRepo.GetByCode(code)
	if err != nil || !share.IsActive {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    404,
			Message: "分享不存在或已失效",
		})
		return
	}

	// 检查过期
	if share.ExpireAt != nil && share.ExpireAt.Before(time.Now()) {
		c.JSON(http.StatusGone, model.APIResponse{
			Code:    410,
			Message: "分享已过期",
		})
		return
	}

	// 验证密码
	if share.Password != "" {
		pwd := c.Query("pwd")
		if pwd != share.Password {
			if !h.checkSharePasswordAttempt(c) {
				return
			}
			h.recordSharePasswordFailure(c)
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    401,
				Message: "需要密码验证",
			})
			return
		}
		h.resetSharePasswordFailures(c)
	}

	if fileID == "" || fileID == "0" {
		fileID = share.TargetID
	}

	// 获取下载链接（预览也是相同链接）
	downloadURL, err := h.wopanClient.GetDownloadURL(fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取预览链接失败: " + err.Error(),
		})
		return
	}

	// 302重定向
	c.Redirect(http.StatusFound, downloadURL)
}

// GetFilePreviewInfo 获取文件预览信息
func (h *Handler) GetFilePreviewInfo(c *gin.Context) {
	code := c.Param("code")
	fileID := c.Query("file_id")

	share, err := h.shareRepo.GetByCode(code)
	if err != nil || !share.IsActive {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    404,
			Message: "分享不存在",
		})
		return
	}

	if fileID == "" {
		fileID = share.TargetID
	}

	// 根据文件扩展名判断预览类型
	fileName := share.TargetName
	ext := strings.ToLower(filepath.Ext(fileName))

	previewType := getPreviewType(ext)

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"file_name":    fileName,
			"file_id":      fileID,
			"preview_type": previewType,
			"can_preview":  previewType != "none",
		},
	})
}

// getPreviewType 根据扩展名获取预览类型
func getPreviewType(ext string) string {
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".svg": true, ".ico": true, ".bmp": true,
	}
	videoExts := map[string]bool{
		".mp4": true, ".webm": true, ".ogg": true, ".m4v": true,
	}
	audioExts := map[string]bool{
		".mp3": true, ".wav": true, ".ogg": true, ".flac": true,
		".m4a": true, ".aac": true,
	}
	textExts := map[string]bool{
		".txt": true, ".md": true, ".json": true, ".xml": true,
		".yaml": true, ".yml": true, ".log": true, ".css": true,
		".js": true, ".html": true, ".htm": true, ".go": true,
		".py": true, ".java": true, ".c": true, ".cpp": true,
		".h": true, ".sh": true, ".bash": true, ".ini": true,
		".conf": true, ".toml": true,
	}
	pdfExts := map[string]bool{
		".pdf": true,
	}

	if imageExts[ext] {
		return "image"
	}
	if videoExts[ext] {
		return "video"
	}
	if audioExts[ext] {
		return "audio"
	}
	if textExts[ext] {
		return "text"
	}
	if pdfExts[ext] {
		return "pdf"
	}
	return "none"
}

// ===================== 监控相关 =====================

// GetMonitorStatus 获取监控状态
func (h *Handler) GetMonitorStatus(c *gin.Context) {
	if h.monitorService == nil {
		c.JSON(http.StatusServiceUnavailable, model.APIResponse{
			Code:    503,
			Message: "监控服务未初始化",
		})
		return
	}

	status, err := h.monitorService.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取监控状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"status":  status,
			"running": h.monitorService.IsRunning(),
		},
	})
}

// CheckMonitorNow 立即执行一次监控检查
func (h *Handler) CheckMonitorNow(c *gin.Context) {
	if h.monitorService == nil {
		c.JSON(http.StatusServiceUnavailable, model.APIResponse{
			Code:    503,
			Message: "监控服务未初始化",
		})
		return
	}

	status, err := h.monitorService.CheckNow()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "执行检查失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "检查完成",
		Data:    status,
	})
}

// TestNotify 测试通知（所有已配置渠道）
func (h *Handler) TestNotify(c *gin.Context) {
	if h.monitorService == nil {
		c.JSON(http.StatusServiceUnavailable, model.APIResponse{
			Code:    503,
			Message: "监控服务未初始化",
		})
		return
	}

	results, err := h.monitorService.TestNotify()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "测试失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "测试完成",
		Data:    results,
	})
}

// GetNotificationLogs 获取通知记录
func (h *Handler) GetNotificationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if h.notificationLogRepo == nil {
		c.JSON(http.StatusServiceUnavailable, model.APIResponse{
			Code:    503,
			Message: "通知服务未初始化",
		})
		return
	}

	logs, total, err := h.notificationLogRepo.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取通知记录失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"logs":      logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// UpdateMonitorSettings 更新监控设置
func (h *Handler) UpdateMonitorSettings(c *gin.Context) {
	var req struct {
		MonitorEnabled   *bool  `json:"monitor_enabled"`
		MonitorInterval  *int   `json:"monitor_interval"`
		NotifyEnabled    *bool  `json:"notify_enabled"`
		DefaultChannel   string `json:"default_notify_channel"`
		BarkURL          string `json:"bark_url"`
		ServerchanKey    string `json:"serverchan_key"`
		TelegramBotToken string `json:"telegram_bot_token"`
		TelegramChatID   string `json:"telegram_chat_id"`
		PushplusToken    string `json:"pushplus_token"`
		DingtalkWebhook  string `json:"dingtalk_webhook"`
		WecomWebhook     string `json:"wecom_webhook"`
		WecomAppConfig   string `json:"wecom_app_config"`
		FeishuWebhook    string `json:"feishu_webhook"`
		WebhookURL       string `json:"webhook_url"`
		PushdeerKey      string `json:"pushdeer_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	settings, err := h.settingsRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取设置失败",
		})
		return
	}

	if req.MonitorEnabled != nil {
		settings.MonitorEnabled = *req.MonitorEnabled
	}
	if req.MonitorInterval != nil {
		if *req.MonitorInterval < 60 {
			*req.MonitorInterval = 60 // 最小1分钟
		}
		settings.MonitorInterval = *req.MonitorInterval
	}
	if req.NotifyEnabled != nil {
		settings.NotifyEnabled = *req.NotifyEnabled
	}
	if req.DefaultChannel != "" {
		settings.DefaultNotifyChannel = req.DefaultChannel
	}
	// 通知渠道配置（只有传入非空值时才更新，空字符串表示保持原值）
	if req.BarkURL != "" {
		settings.BarkURL = req.BarkURL
	}
	if req.ServerchanKey != "" {
		settings.ServerchanKey = req.ServerchanKey
	}
	if req.TelegramBotToken != "" {
		settings.TelegramBotToken = req.TelegramBotToken
	}
	if req.TelegramChatID != "" {
		settings.TelegramChatID = req.TelegramChatID
	}
	if req.PushplusToken != "" {
		settings.PushplusToken = req.PushplusToken
	}
	if req.DingtalkWebhook != "" {
		settings.DingtalkWebhook = req.DingtalkWebhook
	}
	if req.WecomWebhook != "" {
		settings.WecomWebhook = req.WecomWebhook
	}
	if req.WecomAppConfig != "" {
		settings.WecomAppConfig = req.WecomAppConfig
	}
	if req.FeishuWebhook != "" {
		settings.FeishuWebhook = req.FeishuWebhook
	}
	if req.WebhookURL != "" {
		settings.WebhookURL = req.WebhookURL
	}
	if req.PushdeerKey != "" {
		settings.PushdeerKey = req.PushdeerKey
	}

	if err := h.settingsRepo.Update(settings); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "保存设置失败",
		})
		return
	}

	// 重新加载监控服务配置
	if h.monitorService != nil {
		h.monitorService.Reload()
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "监控设置已保存",
	})
}

// ClearNotificationLogs 清空通知记录
func (h *Handler) ClearNotificationLogs(c *gin.Context) {
	if h.notificationLogRepo == nil {
		c.JSON(http.StatusServiceUnavailable, model.APIResponse{
			Code:    503,
			Message: "通知服务未初始化",
		})
		return
	}

	// 只保留0条，即清空所有
	if err := h.notificationLogRepo.CleanOldLogs(0); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "清空失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "通知记录已清空",
	})
}

// GetMonitorSettings 获取监控设置
func (h *Handler) GetMonitorSettings(c *gin.Context) {
	settings, err := h.settingsRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取设置失败",
		})
		return
	}

	// 敏感信息脱敏
	maskString := func(s string) string {
		if s == "" {
			return ""
		}
		if len(s) <= 8 {
			return "****"
		}
		return s[:4] + "****"
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"monitor_enabled":        settings.MonitorEnabled,
			"monitor_interval":       settings.MonitorInterval,
			"notify_enabled":         settings.NotifyEnabled,
			"default_notify_channel": settings.DefaultNotifyChannel,
			"bark_url":               maskString(settings.BarkURL),
			"serverchan_key":         maskString(settings.ServerchanKey),
			"telegram_bot_token":     maskString(settings.TelegramBotToken),
			"telegram_chat_id":       settings.TelegramChatID, // Chat ID 不脱敏
			"pushplus_token":         maskString(settings.PushplusToken),
			"dingtalk_webhook":       maskString(settings.DingtalkWebhook),
			"wecom_webhook":          maskString(settings.WecomWebhook),
			"wecom_app_config":       maskString(settings.WecomAppConfig),
			"feishu_webhook":         maskString(settings.FeishuWebhook),
			"webhook_url":            maskString(settings.WebhookURL),
			"pushdeer_key":           maskString(settings.PushdeerKey),
		},
	})
}
