package main

import (
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
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

type repoBundle struct {
	settings     *repository.SettingsRepository
	share        *repository.ShareRepository
	accessLog    *repository.AccessLogRepository
	monitor      *repository.MonitorRepository
	notification *repository.NotificationLogRepository
}

func logStartup(cfg *config.Config) {
	log.Printf("WoOpen 正在启动...")
	log.Printf("数据目录: %s", cfg.DataDir)
	log.Printf("服务端口: %s", cfg.Port)
}

func initRepositories(db *repository.Database) repoBundle {
	return repoBundle{
		settings:     repository.NewSettingsRepository(db.DB()),
		share:        repository.NewShareRepository(db.DB()),
		accessLog:    repository.NewAccessLogRepository(db.DB()),
		monitor:      repository.NewMonitorRepository(db.DB()),
		notification: repository.NewNotificationLogRepository(db.DB()),
	}
}

func initHandler(repos repoBundle, wopanClient *wopan.Client, adminPassword string) *handler.Handler {
	listCache := handler.NewListTTLCache(handler.DefaultListCacheTTL)
	return handler.NewHandler(handler.HandlerOptions{
		SettingsRepo:        repos.settings,
		ShareRepo:           repos.share,
		AccessLogRepo:       repos.accessLog,
		MonitorRepo:         repos.monitor,
		NotificationLogRepo: repos.notification,
		WopanClient:         wopanClient,
		AdminPassword:       adminPassword,
		ListCache:           listCache,
	})
}

func initMonitorService(repos repoBundle, wopanClient *wopan.Client, h *handler.Handler) *service.MonitorService {
	notifier := service.NewNotifier(repos.notification)
	monitorService := service.NewMonitorService(repos.settings, repos.monitor, wopanClient, notifier)
	h.SetMonitorService(monitorService)
	return monitorService
}

func maybeStartMonitor(settingsRepo *repository.SettingsRepository, monitorService *service.MonitorService) {
	settings, _ := settingsRepo.Get()
	if settings != nil && settings.MonitorEnabled {
		monitorService.SetInterval(settings.MonitorInterval)
		monitorService.Start()
	}
}

func setupRouter(opts routeOptions) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())
	registerRoutes(r, opts)
	return r
}

// webdavWriteMethods 是会修改云盘内容的方法，只读模式下一律拒绝。
var webdavWriteMethods = map[string]bool{
	"PUT": true, "DELETE": true, "MKCOL": true,
	"MOVE": true, "COPY": true, "PROPPATCH": true,
	"LOCK": true, "UNLOCK": true,
}

func newWebDAVHandler(wopanClient *wopan.Client, settingsRepo *repository.SettingsRepository, adminPassword string) http.Handler {
	davFS := webdav.NewWopanFS(wopanClient)
	dav := &xwebdav.Handler{
		Prefix:     "/dav",
		FileSystem: davFS,
		LockSystem: webdav.NewNoLockSystem(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				log.Printf("[WebDAV] %s %s → error: %v", r.Method, r.URL.Path, err)
			}
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[WebDAV] %s %s", r.Method, r.URL.RequestURI())

		settings, _ := settingsRepo.Get()

		// 功能开关：关闭时整体不可用
		if settings != nil && !settings.WebdavEnabled {
			http.Error(w, "WebDAV disabled", http.StatusNotFound)
			return
		}

		// 所有响应都携带 DAV 能力头，让客户端识别为 WebDAV 服务
		w.Header().Set("DAV", "1, 2")
		w.Header().Set("MS-Author-Via", "DAV")

		// OPTIONS 免认证
		if r.Method == http.MethodOptions {
			dav.ServeHTTP(w, r)
			return
		}

		user, pass := "admin", adminPassword
		readonly := false
		if settings != nil {
			user = settings.WebdavUsername
			if settings.WebdavPassword != "" {
				pass = settings.WebdavPassword
			}
			readonly = settings.WebdavReadonly
		}

		if !authenticateWebDAV(w, r, user, pass) {
			return
		}

		// 只读模式拒绝一切写操作
		if readonly && webdavWriteMethods[r.Method] {
			http.Error(w, "WebDAV is read-only", http.StatusForbidden)
			return
		}

		// 挂载卡死的主因：部分客户端对根目录发起 Depth:infinity PROPFIND，
		// 会递归遍历整盘。返回 403 + finite-depth，客户端会退回 Depth:1。
		if r.Method == "PROPFIND" && strings.EqualFold(r.Header.Get("Depth"), "infinity") {
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?><D:error xmlns:D="DAV:"><D:propfind-finite-depth/></D:error>`))
			return
		}

		// GET/HEAD 目录：x/net/webdav 对目录返回 405，但客户端期望 200
		if r.Method == http.MethodHead || r.Method == http.MethodGet {
			reqPath := strings.TrimPrefix(r.URL.Path, "/dav")
			if reqPath == "" {
				reqPath = "/"
			}
			if reqPath == "/" {
				w.Header().Set("Content-Type", "httpd/unix-directory")
				w.Header().Set("Content-Length", "0")
				w.WriteHeader(http.StatusOK)
				return
			}

			resolved, err := davFS.ResolvePath(r.Context(), reqPath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					http.NotFound(w, r)
					return
				}
				http.Error(w, "WebDAV resolve failed: "+err.Error(), http.StatusBadGateway)
				return
			}

			if resolved.Info != nil && resolved.Info.IsDir {
				w.Header().Set("Content-Type", "httpd/unix-directory")
				w.Header().Set("Content-Length", "0")
				w.WriteHeader(http.StatusOK)
				return
			}

			if resolved.Info != nil {
				downloadURL, err := davFS.DownloadURL(resolved.ID)
				if err != nil {
					http.Error(w, "WebDAV redirect failed: "+err.Error(), http.StatusBadGateway)
					return
				}
				http.Redirect(w, r, downloadURL, http.StatusTemporaryRedirect)
				return
			}
		}

		dav.ServeHTTP(w, r)
	})
}

func authenticateWebDAV(w http.ResponseWriter, r *http.Request, wantUser, wantPass string) bool {
	auth := getWebDAVAuthHeader(r)
	if auth == "" {
		log.Printf("[WebDAV] auth missing: %s %s ua=%q", r.Method, r.URL.Path, r.UserAgent())
		davChallenge(w)
		return false
	}
	lower := strings.ToLower(auth)
	if strings.HasPrefix(lower, "bearer ") {
		token := strings.TrimSpace(auth[len("Bearer "):])
		if !middleware.ValidateToken(token) {
			log.Printf("[WebDAV] bearer invalid: %s %s ua=%q", r.Method, r.URL.Path, r.UserAgent())
			davChallenge(w)
			return false
		}
		return true
	}
	user, pass, ok := parseBasicAuth(auth)
	if !ok || user != wantUser || pass != wantPass {
		log.Printf("[WebDAV] auth failed: user=%q ok=%v", user, ok)
		davChallenge(w)
		return false
	}
	return true
}

func getWebDAVAuthHeader(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth != "" {
		return auth
	}
	auth = r.Header.Get("X-Forwarded-Authorization")
	if auth != "" {
		return auth
	}
	return r.Header.Get("X-Original-Authorization")
}

func parseBasicAuth(auth string) (string, string, bool) {
	if !strings.HasPrefix(strings.ToLower(auth), "basic ") {
		return "", "", false
	}
	raw := strings.TrimSpace(auth[len("Basic "):])
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", "", false
	}
	user, pass, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return "", "", false
	}
	return user, pass, true
}

func davChallenge(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="WoOpen WebDAV"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func startHTTPServer(ginEngine *gin.Engine, davHandler http.Handler, port string) {
	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dav" || strings.HasPrefix(r.URL.Path, "/dav/") {
			davHandler.ServeHTTP(w, r)
			return
		}
		ginEngine.ServeHTTP(w, r)
	})
	log.Printf("WoOpen 启动成功! 访问: http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}
