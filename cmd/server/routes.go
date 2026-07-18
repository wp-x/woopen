package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"woopen/internal/handler"
	"woopen/internal/middleware"

	"github.com/gin-gonic/gin"
)

// registerRoutes 注册路由
type routeOptions struct {
	handler *handler.Handler
}

type distInfo struct {
	dir    string
	exists bool
}

func registerRoutes(r *gin.Engine, opts routeOptions) {
	dist := resolveDistInfo()
	registerPublicRoutes(r, opts.handler, dist)
	registerAdminRoutes(r, opts.handler)
	registerStaticRoutes(r, dist)
}

func resolveDistInfo() distInfo {
	distDir := filepath.Join(".", "web", "dist")
	_, err := os.Stat(distDir)
	return distInfo{
		dir:    distDir,
		exists: err == nil,
	}
}

func registerPublicRoutes(r *gin.Engine, h *handler.Handler, dist distInfo) {
	// 站点配置（公开）
	r.GET("/api/site-config", h.GetSiteConfig)
	registerShareRoutes(r, h, dist)
}

func registerShareRoutes(r *gin.Engine, h *handler.Handler, dist distInfo) {
	// 分享访问
	r.GET("/s/:code", func(c *gin.Context) {
		// 同一 URL 按 Accept 分流 HTML/JSON：禁止缓存，否则浏览器/CDN 会把
		// 缓存的 HTML 喂给 XHR，前端表现为「链接失效或不存在」
		c.Header("Vary", "Accept")
		c.Header("Cache-Control", "no-store")
		if wantsHTML(c) {
			serveIndex(c, dist.dir, dist.exists)
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

	// 直连（图床/视频床/文件床）入口，稳定 URL 可直接嵌入 <img>/<video>/Markdown
	r.GET("/f/:code", h.DirectLink)
}

func registerAdminRoutes(r *gin.Engine, h *handler.Handler) {
	api := r.Group("/api")
	api.POST("/auth/login", h.Login)

	auth := api.Group("")
	auth.Use(middleware.AuthMiddleware())
	registerAuthRoutes(auth, h)
	registerFileRoutes(auth, h)
	registerAdminShareRoutes(auth, h)
	registerStatsRoutes(auth, h)
	registerSettingsRoutes(auth, h)
	registerMonitorRoutes(auth, h)
}

func registerAuthRoutes(auth *gin.RouterGroup, h *handler.Handler) {
	auth.GET("/auth/me", h.GetMe)
	auth.POST("/auth/logout", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已登出"})
	})
}

func registerFileRoutes(auth *gin.RouterGroup, h *handler.Handler) {
	auth.GET("/files", h.ListFiles)
	auth.GET("/files/link", h.GetFileLink)
	auth.POST("/files/upload", h.UploadFile)
	auth.GET("/files/upload/progress", h.GetUploadProgress)
	auth.POST("/files/mkdir", h.CreateDirectory)
}

func registerAdminShareRoutes(auth *gin.RouterGroup, h *handler.Handler) {
	auth.GET("/shares", h.ListShares)
	auth.POST("/shares", h.CreateShare)
	auth.POST("/shares/batch", h.BatchCreateShare)
	auth.PUT("/shares/:id", h.UpdateShare)
	auth.DELETE("/shares/:id", h.DeleteShare)
	auth.GET("/shares/:id/qrcode", h.GetShareQRCode)
}

func registerStatsRoutes(auth *gin.RouterGroup, h *handler.Handler) {
	auth.GET("/stats/overview", h.GetStatsOverview)
	auth.GET("/stats/trend", h.GetStatsTrend)
}

func registerSettingsRoutes(auth *gin.RouterGroup, h *handler.Handler) {
	auth.GET("/settings", h.GetSettings)
	auth.PUT("/settings", h.UpdateSettings)
	auth.PUT("/settings/password", h.UpdatePassword)
	auth.POST("/settings/token/test", h.TestToken)
}

func registerMonitorRoutes(auth *gin.RouterGroup, h *handler.Handler) {
	auth.GET("/monitor/status", h.GetMonitorStatus)
	auth.POST("/monitor/check", h.CheckMonitorNow)
	auth.GET("/monitor/settings", h.GetMonitorSettings)
	auth.PUT("/monitor/settings", h.UpdateMonitorSettings)
	auth.POST("/monitor/notify/test", h.TestNotify)
	auth.GET("/monitor/notifications", h.GetNotificationLogs)
	auth.DELETE("/monitor/notifications", h.ClearNotificationLogs)
}

func registerStaticRoutes(r *gin.Engine, dist distInfo) {
	if dist.exists {
		r.Static("/assets", filepath.Join(dist.dir, "assets"))
		r.StaticFile("/favicon.ico", filepath.Join(dist.dir, "favicon.ico"))
	}
	r.NoRoute(func(c *gin.Context) {
		serveIndex(c, dist.dir, dist.exists)
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
