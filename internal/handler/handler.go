package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image/png"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
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

// Handler HTTP处理器
type Handler struct {
	settingsRepo  *repository.SettingsRepository
	shareRepo     *repository.ShareRepository
	accessLogRepo *repository.AccessLogRepository
	wopanClient   *wopan.Client
	adminPassword string
}

// NewHandler 创建Handler
func NewHandler(
	settingsRepo *repository.SettingsRepository,
	shareRepo *repository.ShareRepository,
	accessLogRepo *repository.AccessLogRepository,
	wopanClient *wopan.Client,
	adminPassword string,
) *Handler {
	return &Handler{
		settingsRepo:  settingsRepo,
		shareRepo:     shareRepo,
		accessLogRepo: accessLogRepo,
		wopanClient:   wopanClient,
		adminPassword: adminPassword,
	}
}

// ===================== 公开配置 =====================

// GetSiteConfig 获取公开的站点配置（无需登录）
func (h *Handler) GetSiteConfig(c *gin.Context) {
	settings, err := h.settingsRepo.Get()
	if err != nil {
		// 返回默认值
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    0,
			Message: "success",
			Data: gin.H{
				"site_title":        "WoOpen",
				"login_title":       "WoOpen_Auth_v10.exe",
				"login_avatar":      "💀",
				"login_role_tag":    "ROLE: ADMIN",
				"login_level_tag":   "LEVEL: 99",
				"login_system_name": "WOOPEN CLOUD SYSTEM",
				"share_footer":      "Powered by WOOPEN_OS // BRUTAL_EDITION",
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"site_title":        settings.SiteTitle,
			"login_title":       settings.LoginTitle,
			"login_avatar":      settings.LoginAvatar,
			"login_role_tag":    settings.LoginRoleTag,
			"login_level_tag":   settings.LoginLevelTag,
			"login_system_name": settings.LoginSystemName,
			"share_footer":      settings.ShareFooter,
		},
	})
}

// ===================== 认证相关 =====================

// Login 管理员登录
func (h *Handler) Login(c *gin.Context) {
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
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    401,
			Message: "密码错误",
		})
		return
	}

	// 生成Token
	token, err := middleware.GenerateToken()
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

	if !h.wopanClient.IsInitialized() {
		c.JSON(http.StatusServiceUnavailable, model.APIResponse{
			Code:    503,
			Message: "云盘客户端未初始化，请先配置Token",
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

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"files": files,
			"page":  page,
		},
	})
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

// ===================== 分享管理 =====================

// generateShareCode 生成随机分享码
func generateShareCode() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
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

	share := &model.Share{
		ShareCode:   shareCode,
		TargetType:  req.TargetType,
		TargetID:    req.TargetID,
		TargetPath:  req.TargetPath,
		TargetName:  req.TargetName,
		Password:    req.Password,
		ExpireAt:    expireAt,
		Description: req.Description,
		IsActive:    true,
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
		ShareCode   string `json:"share_code"`
		Password    string `json:"password"`
		ExpireAt    string `json:"expire_at"`
		Description string `json:"description"`
		IsActive    *bool  `json:"is_active"`
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

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"share_code":    share.ShareCode,
			"target_type":   share.TargetType,
			"target_name":   share.TargetName,
			"need_password": needPassword,
		},
	})
}

// VerifySharePassword 验证分享密码
func (h *Handler) VerifySharePassword(c *gin.Context) {
	code := c.Param("code")

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
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    401,
			Message: "密码错误",
		})
		return
	}

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

// DownloadShare 下载分享文件（302重定向）
func (h *Handler) DownloadShare(c *gin.Context) {
	code := c.Param("code")
	fileID := c.Param("fileId")

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
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    401,
				Message: "需要密码验证",
			})
			return
		}
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
			"login_title":       settings.LoginTitle,
			"login_avatar":      settings.LoginAvatar,
			"login_role_tag":    settings.LoginRoleTag,
			"login_level_tag":   settings.LoginLevelTag,
			"login_system_name": settings.LoginSystemName,
			"share_footer":      settings.ShareFooter,
			"initialized":       h.wopanClient.IsInitialized(),
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
	var req struct {
		RefreshToken    string `json:"refresh_token"`
		AccessToken     string `json:"access_token"`
		RootFolderID    string `json:"root_folder_id"`
		SiteTitle       string `json:"site_title"`
		SiteLogo        string `json:"site_logo"`
		LoginTitle      string `json:"login_title"`
		LoginAvatar     string `json:"login_avatar"`
		LoginRoleTag    string `json:"login_role_tag"`
		LoginLevelTag   string `json:"login_level_tag"`
		LoginSystemName string `json:"login_system_name"`
		ShareFooter     string `json:"share_footer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	settings, _ := h.settingsRepo.Get()

	// 只有当新Token不为空且不是掩码时才更新
	tokenChanged := false
	refreshTokenChanged := false
	accessTokenChanged := false
	if req.RefreshToken != "" && !containsMask(req.RefreshToken) {
		settings.RefreshToken = req.RefreshToken
		tokenChanged = true
		refreshTokenChanged = true
	}
	if req.AccessToken != "" && !containsMask(req.AccessToken) {
		settings.AccessToken = req.AccessToken
		tokenChanged = true
		accessTokenChanged = true
	}
	// 如果刷新了 RefreshToken 但未提供新的 AccessToken，清空旧的 AccessToken
	if refreshTokenChanged && !accessTokenChanged {
		settings.AccessToken = ""
	}
	settings.RootFolderID = req.RootFolderID
	settings.SiteTitle = req.SiteTitle
	settings.SiteLogo = req.SiteLogo
	settings.LoginTitle = req.LoginTitle
	settings.LoginAvatar = req.LoginAvatar
	settings.LoginRoleTag = req.LoginRoleTag
	settings.LoginLevelTag = req.LoginLevelTag
	settings.LoginSystemName = req.LoginSystemName
	settings.ShareFooter = req.ShareFooter

	if err := h.settingsRepo.Update(settings); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "保存设置失败",
		})
		return
	}

	// 如果Token更新了，重新初始化客户端
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

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "Token有效",
	})
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
			ShareCode:   shareCode,
			TargetType:  item.TargetType,
			TargetID:    item.TargetID,
			TargetPath:  item.TargetPath,
			TargetName:  item.TargetName,
			Password:    item.Password,
			ExpireAt:    expireAt,
			Description: item.Description,
			IsActive:    true,
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
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    401,
				Message: "需要密码验证",
			})
			return
		}
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
	files, err := h.wopanClient.ListFiles(dirID, 1, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "获取文件列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"files":      files,
			"share_code": share.ShareCode,
			"root_id":    share.TargetID,
			"current_id": dirID,
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
	fileID := c.Param("fileId")

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
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    401,
				Message: "需要密码验证",
			})
			return
		}
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
