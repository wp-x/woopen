package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"woopen/internal/config"
	"woopen/internal/handler"
	"woopen/internal/middleware"
	"woopen/internal/repository"
	"woopen/internal/service"
	"woopen/internal/webdav"
	"woopen/internal/wopan"

	"github.com/gin-gonic/gin"
	xwebdav "golang.org/x/net/webdav"
)

func main() {
	// 加载配置
	cfg := config.DefaultConfig()
	log.Printf("WoOpen 正在启动...")
	log.Printf("数据目录: %s", cfg.DataDir)
	log.Printf("服务端口: %s", cfg.Port)

	// 设置JWT密钥
	middleware.JWTSecret = cfg.JWTSecret

	// 初始化数据库
	db, err := repository.NewDatabase(cfg.DatabasePath())
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer db.Close()

	// 初始化仓库
	settingsRepo := repository.NewSettingsRepository(db.DB())
	shareRepo := repository.NewShareRepository(db.DB())
	accessLogRepo := repository.NewAccessLogRepository(db.DB())
	monitorRepo := repository.NewMonitorRepository(db.DB())
	notificationLogRepo := repository.NewNotificationLogRepository(db.DB())

	// 初始化云盘客户端
	wopanClient := initWopanClient(settingsRepo)

	// 创建Handler
	h := handler.NewHandler(settingsRepo, shareRepo, accessLogRepo, monitorRepo, notificationLogRepo, wopanClient, cfg.AdminPassword)

	// 初始化监控服务
	notifier := service.NewNotifier(notificationLogRepo)
	monitorService := service.NewMonitorService(settingsRepo, monitorRepo, wopanClient, notifier)
	h.SetMonitorService(monitorService)

	// 根据配置决定是否启动监控
	settings, _ := settingsRepo.Get()
	if settings != nil && settings.MonitorEnabled {
		monitorService.SetInterval(settings.MonitorInterval)
		monitorService.Start()
	}

	// 设置Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS中间件
	r.Use(middleware.CORSMiddleware())

	// 注册路由
	registerRoutes(r, h, wopanClient, cfg.AdminPassword)

	// 启动服务
	log.Printf("WoOpen 启动成功! 访问: http://localhost:%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}

// initWopanClient 初始化云盘客户端
func initWopanClient(settingsRepo *repository.SettingsRepository) *wopan.Client {
	settings, err := settingsRepo.Get()
	if err != nil {
		log.Printf("获取设置失败: %v", err)
		return wopan.NewClient("")
	}

	if settings.RefreshToken == "" && settings.AccessToken == "" {
		log.Println("未配置 refresh_token 或 access_token，请在管理后台配置")
		return wopan.NewClient("")
	}

	client := wopan.NewClient(settings.RefreshToken)
	client.SetRootFolderID(settings.RootFolderID)

	// AccessToken 可选；提供后可减少首次刷新
	if settings.AccessToken != "" {
		client.SetAccessToken(settings.AccessToken)
	}

	// 设置Token更新回调
	client.SetTokenUpdateCallback(func(refreshToken, accessToken string) {
		if err := settingsRepo.UpdateToken(refreshToken, accessToken); err != nil {
			log.Printf("保存Token失败: %v", err)
		} else {
			log.Println("Token已自动更新并保存")
		}
	})

	// 初始化客户端（个人云模式）
	if err := client.Init(); err != nil {
		log.Printf("初始化云盘客户端失败: %v (请检查Token是否有效)", err)
		return client
	}

	log.Println("云盘客户端初始化成功")
	return client
}

// registerRoutes 注册路由
func registerRoutes(r *gin.Engine, h *handler.Handler, wopanClient *wopan.Client, adminPassword string) {
	distDir := filepath.Join(".", "web", "dist")
	distExists := false
	if _, err := os.Stat(distDir); err == nil {
		distExists = true
	}

	// ==================== 公开API ====================
	// 站点配置（公开）
	r.GET("/api/site-config", h.GetSiteConfig)

	// 分享访问
	r.GET("/s/:code", func(c *gin.Context) {
		if wantsHTML(c) {
			serveIndex(c, distDir, distExists)
			return
		}
		h.AccessShare(c)
	})
	r.POST("/s/:code/verify", h.VerifySharePassword)
	r.GET("/s/:code/download", h.DownloadShare)
	r.GET("/s/:code/download/:fileId", h.DownloadShare)
	r.GET("/s/:code/preview", h.PreviewShare)
	r.GET("/s/:code/preview/:fileId", h.PreviewShare)
	r.GET("/s/:code/files", h.GetShareFiles)
	r.GET("/s/:code/info", h.GetFilePreviewInfo)

	// ==================== 管理API ====================
	api := r.Group("/api")
	{
		// 认证
		api.POST("/auth/login", h.Login)

		// 需要认证的路由
		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			// 用户信息
			auth.GET("/auth/me", h.GetMe)
			auth.POST("/auth/logout", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已登出"})
			})

			// 文件操作
			auth.GET("/files", h.ListFiles)
			auth.GET("/files/link", h.GetFileLink)
			auth.POST("/files/upload", h.UploadFile)
			auth.GET("/files/upload/progress", h.GetUploadProgress)
			auth.POST("/files/mkdir", h.CreateDirectory)

			// 分享管理
			auth.GET("/shares", h.ListShares)
			auth.POST("/shares", h.CreateShare)
			auth.POST("/shares/batch", h.BatchCreateShare)
			auth.PUT("/shares/:id", h.UpdateShare)
			auth.DELETE("/shares/:id", h.DeleteShare)
			auth.GET("/shares/:id/qrcode", h.GetShareQRCode)

			// 统计
			auth.GET("/stats/overview", h.GetStatsOverview)
			auth.GET("/stats/trend", h.GetStatsTrend)

			// 设置
			auth.GET("/settings", h.GetSettings)
			auth.PUT("/settings", h.UpdateSettings)
			auth.PUT("/settings/password", h.UpdatePassword)
			auth.POST("/settings/token/test", h.TestToken)

			// 监控
			auth.GET("/monitor/status", h.GetMonitorStatus)
			auth.POST("/monitor/check", h.CheckMonitorNow)
			auth.GET("/monitor/settings", h.GetMonitorSettings)
			auth.PUT("/monitor/settings", h.UpdateMonitorSettings)
			auth.POST("/monitor/notify/test", h.TestNotify)
			auth.GET("/monitor/notifications", h.GetNotificationLogs)
			auth.DELETE("/monitor/notifications", h.ClearNotificationLogs)
		}
	}

	// ==================== WebDAV ====================
	davFS := webdav.NewWopanFS(wopanClient)
	davHandler := &xwebdav.Handler{
		FileSystem: davFS,
		LockSystem: xwebdav.NewMemLS(),
	}

	// WebDAV 路由（需要认证）
	r.Any("/dav/*filepath", middleware.WebDAVAuthMiddleware(adminPassword), gin.WrapH(davHandler))

	// ==================== 静态文件 ====================
	if distExists {
		r.Static("/assets", filepath.Join(distDir, "assets"))
		r.StaticFile("/favicon.ico", filepath.Join(distDir, "favicon.ico"))
	}
	r.NoRoute(func(c *gin.Context) {
		serveIndex(c, distDir, distExists)
	})
}

func wantsHTML(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	return strings.Contains(accept, "text/html") || strings.Contains(accept, "application/xhtml+xml")
}

func serveIndex(c *gin.Context, distDir string, distExists bool) {
	if distExists {
		c.File(filepath.Join(distDir, "index.html"))
		return
	}
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, devModeHTML())
}

// devModeHTML 开发模式HTML
func devModeHTML() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>WoOpen - 开发模式</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
			   max-width: 800px; margin: 100px auto; padding: 20px; }
		h1 { color: #409eff; }
		.api { background: #f5f7fa; padding: 20px; border-radius: 8px; margin: 20px 0; }
		code { background: #e4e7ed; padding: 2px 6px; border-radius: 4px; }
	</style>
</head>
<body>
	<h1>🚀 WoOpen 后端已启动</h1>
	<p>前端正在开发中，请运行前端开发服务器。</p>
	<div class="api">
		<h3>可用的API端点：</h3>
		<ul>
			<li><code>POST /api/auth/login</code> - 登录</li>
			<li><code>GET /api/files</code> - 获取文件列表</li>
			<li><code>GET /api/shares</code> - 获取分享列表</li>
			<li><code>POST /api/shares</code> - 创建分享</li>
			<li><code>GET /api/settings</code> - 获取设置</li>
			<li><code>PUT /api/settings</code> - 更新设置</li>
		</ul>
	</div>
	<p>默认管理员密码: <code>admin123</code></p>
</body>
</html>`)
}
